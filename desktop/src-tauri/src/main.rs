// Prevents additional console window on Windows in release, DO NOT REMOVE!!
// Comment out the line below to see console output in release builds
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use serde::{Deserialize, Serialize};
use std::net::TcpListener;
use std::process::{Child, Command, Stdio};
use std::sync::Mutex;
use std::thread;
use std::time::Duration;
use tauri::Manager;

#[derive(Debug, Serialize, Deserialize)]
struct BackendInfo {
    port: u16,
    pid: Option<u32>,
}

// Global state to track the backend process
static BACKEND_INFO: Mutex<Option<BackendInfo>> = Mutex::new(None);
static BACKEND_CHILD: Mutex<Option<Child>> = Mutex::new(None);

// Learn more about Tauri commands at https://tauri.app/v1/guides/features/command
#[tauri::command]
fn greet(name: &str) -> String {
    format!("Hello, {}! You've been greeted from Rust!", name)
}

#[tauri::command]
fn get_backend_port() -> Option<u16> {
    println!("[DEBUG] get_backend_port called");
    if let Ok(info) = BACKEND_INFO.lock() {
        let port = info.as_ref().map(|i| i.port);
        println!("[DEBUG] Returning port: {:?}", port);
        port
    } else {
        println!("[DEBUG] Failed to lock BACKEND_INFO");
        None
    }
}

fn find_available_port() -> Result<u16, Box<dyn std::error::Error>> {
    // Try to bind to port 0 to get an available port
    let listener = TcpListener::bind("127.0.0.1:0")?;
    let addr = listener.local_addr()?;
    Ok(addr.port())
}

fn start_backend() -> Result<BackendInfo, Box<dyn std::error::Error>> {
    println!("[DEBUG] Starting backend process...");

    // Find an available port
    let port = find_available_port()?;
    println!("[DEBUG] Found available port: {}", port);

    // Get the path to the core binary
    let exe_path = std::env::current_exe()?;
    let exe_dir = exe_path
        .parent()
        .ok_or("Could not get executable directory")?;

    // Try different possible locations for the core binary across platforms
    // On Windows during development, prefer the bin directory over target directory
    let exe_candidates = ["whodb-core", "whodb-core.exe", "bin/whodb-core", "bin/whodb-core.exe"];
    let mut possible_paths = Vec::new();

    // First check the bin directory relative to src-tauri
    if cfg!(debug_assertions) {
        // In debug mode, look in src-tauri/bin first
        if let Some(manifest_dir) = option_env!("CARGO_MANIFEST_DIR") {
            let manifest_path = std::path::Path::new(manifest_dir);
            possible_paths.push(manifest_path.join("bin").join("whodb-core.exe"));
            possible_paths.push(manifest_path.join("bin").join("whodb-core"));
        }
    }

    for name in &exe_candidates {
        possible_paths.push(exe_dir.join(name));
        possible_paths.push(exe_dir.join("resources").join(name));
        possible_paths.push(exe_dir.join("..").join("resources").join(name));
    }

    let mut core_binary = None;
    for path in &possible_paths {
        if path.exists() {
            core_binary = Some(path.clone());
            break;
        }
    }

    let core_binary = core_binary.ok_or_else(|| {
        eprintln!("[ERROR] Core binary not found. Searched paths:");
        for path in &possible_paths {
            eprintln!("  - {}", path.display());
        }
        "Core binary not found in any expected location"
    })?;

    println!("[DEBUG] Found core binary at: {}", core_binary.display());

    // Start the backend process with the random port
    println!("[DEBUG] Starting command: {:?}", &core_binary);
    println!("[DEBUG] With PORT={}", port);
    println!("[DEBUG] With WHODB_ALLOWED_ORIGINS=tauri://*,taur://*,app://*,http://localhost:1420,http://localhost:*,https://*");

    let child = Command::new(&core_binary)
        .env("PORT", port.to_string())
        .env(
            "WHODB_ALLOWED_ORIGINS",
            // Allow Tauri custom protocols and local dev origins
            // Include variants to cover potential scheme differences across platforms
            "tauri://*,taur://*,app://*,http://localhost:1420,http://localhost:*,https://*",
        )
        .stdout(Stdio::inherit())  // Changed to inherit to see output
        .stderr(Stdio::inherit())  // Changed to inherit to see output
        .spawn()?;

    let pid = child.id();
    println!("[DEBUG] Backend process started with PID: {}", pid);

    // Store the child process globally
    if let Ok(mut child_guard) = BACKEND_CHILD.lock() {
        *child_guard = Some(child);
    }

    // Give the process a moment to start
    thread::sleep(Duration::from_millis(1000));

    // Check if the process is still running
    if let Ok(mut child_guard) = BACKEND_CHILD.lock() {
        if let Some(ref mut child) = *child_guard {
            match child.try_wait() {
                Ok(Some(status)) => {
                    eprintln!("[ERROR] Backend process exited immediately!");
                    eprintln!("[ERROR] Exit status: {:?}", status);

                    // Note: Can't read stderr since we're using inherit mode

                    return Err(format!(
                        "Backend process exited immediately with status: {:?}",
                        status
                    )
                    .into());
                }
                Ok(None) => {
                    println!("[DEBUG] Backend process is running successfully");
                }
                Err(e) => {
                    return Err(format!("Error checking backend process: {}", e).into());
                }
            }
        }
    }

    Ok(BackendInfo {
        port,
        pid: Some(pid),
    })
}

fn cleanup_backend() {
    println!("🧹 Cleaning up backend process...");
    if let Ok(mut child_guard) = BACKEND_CHILD.lock() {
        if let Some(mut child) = child_guard.take() {
            match child.kill() {
                Ok(_) => println!("✅ Backend process terminated"),
                Err(e) => eprintln!("❌ Failed to terminate backend process: {}", e),
            }
        }
    }
}

fn main() {
    // Set up cleanup on exit
    let _cleanup_handler = || {
        cleanup_backend();
    };

    // Start the backend process
    match start_backend() {
        Ok(backend_info) => {
            println!("🚀 Started WhoDB backend on port {}", backend_info.port);

            // Store the backend info globally
            if let Ok(mut info) = BACKEND_INFO.lock() {
                *info = Some(backend_info);
            }
        }
        Err(e) => {
            eprintln!("❌ Failed to start backend: {}", e);
            // Continue anyway - the frontend might be able to connect to an external backend
        }
    }

    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .invoke_handler(tauri::generate_handler![greet, get_backend_port])
        .on_window_event(|_, event| {
            if matches!(event, tauri::WindowEvent::CloseRequested { .. }) {
                cleanup_backend();
            }
        })
        .setup(|app| {
            // Set up Stronghold with built-in Argon2
            let salt_path = app
                .path()
                .app_local_data_dir()
                .expect("could not resolve app local data path")
                .join("salt.txt");

            app.handle().plugin(
                tauri_plugin_stronghold::Builder::with_argon2(&salt_path).build()
            )?;

            #[cfg(debug_assertions)]
            {
                // Open developer tools in debug builds
                if let Some(window) = app.get_webview_window("main") {
                    window.open_devtools();
                }
            }
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
