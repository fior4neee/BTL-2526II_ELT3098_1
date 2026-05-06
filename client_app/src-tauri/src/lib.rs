mod antenna_tracker;
mod geo_reporter;

use antenna_tracker::{AntennaTracker, LinkBudgetInput, TrackingSettings};
use geo_reporter::GeoReporter;
use std::sync::{Arc, Mutex};
use std::thread;
use std::time::Duration;
use tauri::{AppHandle, Emitter, Manager, State};

struct ClientState {
    tracker: Mutex<AntennaTracker>,
    geo: Mutex<GeoReporter>,
}

#[tauri::command]
fn get_signal_snapshot(state: State<'_, Arc<ClientState>>) -> antenna_tracker::TelemetryFrame {
    let mut tracker = state.tracker.lock().expect("tracker mutex poisoned");
    tracker.next_frame()
}

#[tauri::command]
fn get_location_snapshot(state: State<'_, Arc<ClientState>>) -> geo_reporter::LocationFrame {
    let mut geo = state.geo.lock().expect("geo mutex poisoned");
    geo.next_frame()
}

#[tauri::command]
fn update_tracking_settings(
    settings: TrackingSettings,
    state: State<'_, Arc<ClientState>>,
) -> Result<(), String> {
    let mut tracker = state.tracker.lock().map_err(|err| err.to_string())?;
    tracker.update_settings(settings);
    Ok(())
}

fn emit_telemetry(app: AppHandle, state: Arc<ClientState>) {
    thread::spawn(move || loop {
        if let Ok(mut tracker) = state.tracker.lock() {
            let frame = tracker.next_frame();
            let _ = app.emit("signal-telemetry", frame);
        }
        thread::sleep(Duration::from_millis(100));
    });
}

fn emit_location(app: AppHandle, state: Arc<ClientState>) {
    thread::spawn(move || loop {
        if let Ok(mut geo) = state.geo.lock() {
            let frame = geo.next_frame();
            let _ = app.emit("location-telemetry", frame);
        }
        thread::sleep(Duration::from_secs(1));
    });
}

pub fn run() {
    let state = Arc::new(ClientState {
        tracker: Mutex::new(AntennaTracker::new(
            21.0278,
            105.8342,
            LinkBudgetInput::default(),
        )),
        geo: Mutex::new(GeoReporter::fixed_hanoi()),
    });

    tauri::Builder::default()
        .manage(state.clone())
        .invoke_handler(tauri::generate_handler![
            get_signal_snapshot,
            get_location_snapshot,
            update_tracking_settings
        ])
        .setup(move |app| {
            emit_telemetry(app.handle().clone(), state.clone());
            emit_location(app.handle().clone(), state.clone());
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
