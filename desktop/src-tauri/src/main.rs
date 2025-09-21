// Prevents additional console window on Windows in release, DO NOT REMOVE!!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use std::process::{Command, Stdio, Child};
use std::sync::Mutex;
use std::thread;
use std::time::Duration;
use std::net::TcpListener;
use serde::{Deserialize, Serialize};

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
    if let Ok(info) = BACKEND_INFO.lock() {
        info.as_ref().map(|i| i.port)
    } else {
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
    // Find an available port
    let port = find_available_port()?;
    
    // Get the path to the core binary
    let exe_path = std::env::current_exe()?;
    let exe_dir = exe_path.parent().ok_or("Could not get executable directory")?;
    
    // Try different possible locations for the core binary
    let possible_paths = vec![
        exe_dir.join("whodb-core"), // Development mode
        exe_dir.join("resources").join("whodb-core"), // Bundled mode
        exe_dir.join("..").join("resources").join("whodb-core"), // Alternative bundled location
    ];
    
    let mut core_binary = None;
    for path in possible_paths {
        if path.exists() {
            core_binary = Some(path);
            break;
        }
    }
    
    let core_binary = core_binary.ok_or("Core binary not found in any expected location")?;
    
    // Start the backend process with the random port
    let child = Command::new(&core_binary)
        .env("PORT", port.to_string())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()?;
    
    let pid = child.id();
    
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
                    return Err(format!("Backend process exited immediately with status: {:?}", status).into());
                }
                Ok(None) => {
                    // Process is still running, which is good
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
        .invoke_handler(tauri::generate_handler![greet, get_backend_port])
        .on_window_event(|event| {
            if let tauri::WindowEvent::CloseRequested { .. } = event.event() {
                cleanup_backend();
            }
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
