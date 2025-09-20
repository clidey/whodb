#![cfg_attr(
    all(not(debug_assertions), target_os = "windows"),
    windows_subsystem = "windows"
)]

use std::process::{Command, Stdio};
use std::sync::Mutex;
use tauri::{AppHandle, Manager, State, Window};
use tokio::time::{sleep, Duration};

struct BackendProcess {
    child: Option<std::process::Child>,
}

impl BackendProcess {
    fn new() -> Self {
        Self { child: None }
    }

    fn start(&mut self, app_handle: &AppHandle) -> Result<(), String> {
        if self.child.is_some() {
            return Ok(()); // Already running
        }

        let resource_path = app_handle
            .path_resolver()
            .resolve_resource("binaries/whodb")
            .ok_or("Failed to resolve backend binary path")?;

        let backend_path = if cfg!(target_os = "windows") {
            resource_path.with_extension("exe")
        } else {
            resource_path
        };

        let child = Command::new(&backend_path)
            .stdout(Stdio::inherit())
            .stderr(Stdio::inherit())
            .spawn()
            .map_err(|e| format!("Failed to start backend: {}", e))?;

        self.child = Some(child);
        Ok(())
    }

    fn stop(&mut self) {
        if let Some(mut child) = self.child.take() {
            let _ = child.kill();
            let _ = child.wait();
        }
    }
}

type BackendState = Mutex<BackendProcess>;

#[tauri::command]
async fn wait_for_backend() -> Result<String, String> {
    let client = reqwest::Client::builder()
        .timeout(Duration::from_secs(2))
        .build()
        .map_err(|e| format!("Failed to create HTTP client: {}", e))?;

    // Try for up to 30 seconds
    for _ in 0..30 {
        match client.get("http://localhost:8080").send().await {
            Ok(response) if response.status().is_success() => {
                return Ok("Backend is ready".to_string());
            }
            _ => {
                sleep(Duration::from_millis(1000)).await;
            }
        }
    }

    Err("Backend failed to start within 30 seconds".to_string())
}

#[tauri::command]
fn get_backend_url() -> String {
    "http://localhost:8080".to_string()
}

fn main() {
    let backend = BackendState::new(BackendProcess::new());

    tauri::Builder::default()
        .manage(backend)
        .setup(|app| {
            let backend_state: State<BackendState> = app.state();
            let mut backend = backend_state.lock().unwrap();
            
            // Start the backend process
            backend.start(app.app_handle())?;

            // Set up single instance plugin
            #[cfg(not(any(target_os = "android", target_os = "ios")))]
            {
                use tauri_plugin_single_instance::init;
                app.handle().plugin(init(|app, _argv, _cwd| {
                    let windows = app.windows();
                    windows
                        .values()
                        .next()
                        .expect("Sorry, no window found")
                        .set_focus()
                        .expect("Can't focus window");
                }))?;
            }

            Ok(())
        })
        .on_window_event(|event| match event.event() {
            tauri::WindowEvent::Destroyed => {
                // Stop backend when main window is closed
                let backend_state: State<BackendState> = event.window().state();
                let mut backend = backend_state.lock().unwrap();
                backend.stop();
            }
            _ => {}
        })
        .invoke_handler(tauri::generate_handler![wait_for_backend, get_backend_url])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}