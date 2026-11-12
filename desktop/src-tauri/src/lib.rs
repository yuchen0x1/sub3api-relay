use std::sync::Mutex;

use serde::{Deserialize, Serialize};

#[derive(Clone, Deserialize, Serialize)]
#[serde(rename_all = "kebab-case")]
pub enum AuthScheme {
    ApiKey,
    Bearer,
}

#[derive(Clone, Deserialize, Serialize)]
pub struct AuthSession {
    pub token: String,
    pub scheme: AuthScheme,
}

#[derive(Default)]
pub struct AuthState(Mutex<Option<AuthSession>>);

#[tauri::command]
fn get_auth_session(state: tauri::State<AuthState>) -> Result<Option<AuthSession>, String> {
    state.0.lock().map(|session| session.clone()).map_err(|_| "auth state is unavailable".to_string())
}

#[tauri::command]
fn set_auth_session(session: AuthSession, state: tauri::State<AuthState>) -> Result<(), String> {
    if session.token.trim().is_empty() {
        return Err("auth token must not be empty".to_string());
    }
    state.0.lock().map(|mut current| *current = Some(session)).map_err(|_| "auth state is unavailable".to_string())
}

#[tauri::command]
fn clear_auth_session(state: tauri::State<AuthState>) -> Result<(), String> {
    state.0.lock().map(|mut current| *current = None).map_err(|_| "auth state is unavailable".to_string())
}

pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_process::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .manage(AuthState::default())
        .invoke_handler(tauri::generate_handler![get_auth_session, set_auth_session, clear_auth_session])
        .run(tauri::generate_context!("tauri.conf.json"))
        .expect("error while running Sub3API Cockpit");
}
