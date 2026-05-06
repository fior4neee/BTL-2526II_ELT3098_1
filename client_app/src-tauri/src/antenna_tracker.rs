use serde::{Deserialize, Serialize};
use std::f64::consts::PI;

const EARTH_RADIUS_KM: f64 = 6371.0;
const BOLTZMANN_DBW_PER_HZ_K: f64 = -228.6;

#[derive(Debug, Clone, Copy)]
pub struct SatelliteEphemeris {
    pub latitude_deg: f64,
    pub longitude_deg: f64,
    pub altitude_km: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TrackingSettings {
    pub selected_satellite: String,
    pub steering_mode: String,
    pub signal_threshold_db: f64,
    pub log_level: String,
}

impl Default for TrackingSettings {
    fn default() -> Self {
        Self {
            selected_satellite: "auto".to_string(),
            steering_mode: "auto".to_string(),
            signal_threshold_db: 12.0,
            log_level: "info".to_string(),
        }
    }
}

#[derive(Debug, Clone, Copy)]
pub struct LinkBudgetInput {
    pub satellite_eirp_dbw: f64,
    pub atmospheric_loss_db: f64,
    pub receiver_gt_db: f64,
    pub bandwidth_hz: f64,
    pub bit_rate_bps: f64,
}

impl Default for LinkBudgetInput {
    fn default() -> Self {
        Self {
            satellite_eirp_dbw: 48.0,
            atmospheric_loss_db: 1.9,
            receiver_gt_db: 11.5,
            bandwidth_hz: 25_000_000.0,
            bit_rate_bps: 50_000_000.0,
        }
    }
}

#[derive(Debug, Clone, Serialize)]
pub struct TelemetryFrame {
    pub timestamp_ms: u128,
    pub satellite_name: String,
    pub gateway_name: String,
    pub azimuth_deg: f64,
    pub elevation_deg: f64,
    pub beam_quality: f64,
    pub carrier_power_dbm: f64,
    pub c_n_ratio_db: f64,
    pub eb_n0_db: f64,
    pub ber: f64,
    pub path_loss_db: f64,
    pub eirp_dbw: f64,
    pub modulation_scheme: String,
    pub link_status: String,
    pub time_to_horizon_s: u32,
    pub handover_active: bool,
    pub packet_loss_pct: f64,
    pub latency_ms: f64,
    pub jitter_ms: f64,
    pub data_down_mbps: f64,
    pub data_up_mbps: f64,
}

pub struct AntennaTracker {
    user_lat_deg: f64,
    user_lon_deg: f64,
    element_count: u32,
    tick: u64,
    settings: TrackingSettings,
    link_budget: LinkBudgetInput,
}

impl AntennaTracker {
    pub fn new(user_lat_deg: f64, user_lon_deg: f64, link_budget: LinkBudgetInput) -> Self {
        Self {
            user_lat_deg,
            user_lon_deg,
            element_count: 256,
            tick: 0,
            settings: TrackingSettings::default(),
            link_budget,
        }
    }

    pub fn update_settings(&mut self, settings: TrackingSettings) {
        self.settings = settings;
    }

    pub fn next_frame(&mut self) -> TelemetryFrame {
        self.tick += 1;
        let ephemeris = self.synthetic_ephemeris();
        let pointing = self.pointing_to(ephemeris);
        let range_km = slant_range_km(self.user_lat_deg, self.user_lon_deg, ephemeris);
        let path_loss_db =
            free_space_path_loss_db(range_km, 12.0) + self.link_budget.atmospheric_loss_db;
        let array_gain_db = phased_array_gain_db(self.element_count, pointing.elevation_deg);
        let carrier_power_dbm =
            self.link_budget.satellite_eirp_dbw + array_gain_db - path_loss_db + 30.0;
        let c_n_ratio_db = self.link_budget.satellite_eirp_dbw - path_loss_db
            + self.link_budget.receiver_gt_db
            - BOLTZMANN_DBW_PER_HZ_K
            - hz_to_db(self.link_budget.bandwidth_hz);
        let eb_n0_db = c_n_ratio_db + hz_to_db(self.link_budget.bandwidth_hz)
            - hz_to_db(self.link_budget.bit_rate_bps);
        let beam_quality =
            beam_quality(pointing.elevation_deg, self.settings.steering_mode.as_str());
        let ber = qpsk_ber(eb_n0_db);
        let link_status = if c_n_ratio_db < 9.0 {
            "outage"
        } else if pointing.elevation_deg < 15.0 || c_n_ratio_db < self.settings.signal_threshold_db
        {
            "searching"
        } else {
            "connected"
        };
        let handover_active = self.tick % 220 == 0;
        let gateway_name = match (self.tick / 220) % 3 {
            0 => "Hanoi",
            1 => "Danang",
            _ => "HCMC",
        };

        TelemetryFrame {
            timestamp_ms: now_ms(),
            satellite_name: self.satellite_name(),
            gateway_name: gateway_name.to_string(),
            azimuth_deg: pointing.azimuth_deg,
            elevation_deg: pointing.elevation_deg,
            beam_quality,
            carrier_power_dbm,
            c_n_ratio_db,
            eb_n0_db,
            ber,
            path_loss_db,
            eirp_dbw: self.link_budget.satellite_eirp_dbw,
            modulation_scheme: modulation_for(c_n_ratio_db).to_string(),
            link_status: link_status.to_string(),
            time_to_horizon_s: ((pointing.elevation_deg - 4.0).max(0.0) * 22.0).round() as u32,
            handover_active,
            packet_loss_pct: if handover_active { 0.06 } else { 0.01 },
            latency_ms: 38.0
                + (18.0 - c_n_ratio_db).max(0.0) * 2.2
                + if handover_active { 18.0 } else { 0.0 },
            jitter_ms: 3.0
                + (16.0 - c_n_ratio_db).max(0.0) * 0.65
                + if handover_active { 5.0 } else { 0.0 },
            data_down_mbps: (42.0 + c_n_ratio_db * 1.9 + wave(self.tick, 31.0) * 18.0).max(2.0),
            data_up_mbps: (8.0 + c_n_ratio_db * 0.35 + wave(self.tick, 43.0) * 4.0).max(1.0),
        }
    }

    pub fn pointing_to(&self, satellite: SatelliteEphemeris) -> Pointing {
        calculate_pointing(self.user_lat_deg, self.user_lon_deg, satellite)
    }

    fn synthetic_ephemeris(&self) -> SatelliteEphemeris {
        let phase = self.tick as f64 / 180.0;
        SatelliteEphemeris {
            latitude_deg: 16.0 + phase.sin() * 18.0,
            longitude_deg: 106.0 + (phase * 0.72).cos() * 18.0,
            altitude_km: 550.0 + (phase * 1.7).sin() * 18.0,
        }
    }

    fn satellite_name(&self) -> String {
        if self.settings.selected_satellite != "auto" {
            return self.settings.selected_satellite.clone();
        }
        let names = ["VNU-LEO-014", "VNU-LEO-021", "VNU-LEO-033", "VNU-LEO-048"];
        names[((self.tick / 220) as usize) % names.len()].to_string()
    }
}

#[derive(Debug, Clone, Copy)]
pub struct Pointing {
    pub azimuth_deg: f64,
    pub elevation_deg: f64,
}

pub fn calculate_pointing(
    user_lat_deg: f64,
    user_lon_deg: f64,
    sat: SatelliteEphemeris,
) -> Pointing {
    let lat1 = user_lat_deg.to_radians();
    let lon1 = user_lon_deg.to_radians();
    let lat2 = sat.latitude_deg.to_radians();
    let lon2 = sat.longitude_deg.to_radians();
    let d_lon = lon2 - lon1;

    let y = d_lon.sin() * lat2.cos();
    let x = lat1.cos() * lat2.sin() - lat1.sin() * lat2.cos() * d_lon.cos();
    let azimuth_deg = (y.atan2(x).to_degrees() + 360.0) % 360.0;

    let central_angle = (lat1.sin() * lat2.sin() + lat1.cos() * lat2.cos() * d_lon.cos()).acos();
    let radius_ratio = EARTH_RADIUS_KM / (EARTH_RADIUS_KM + sat.altitude_km);
    let elevation_rad =
        ((central_angle.cos() - radius_ratio) / central_angle.sin().max(1e-9)).atan();

    Pointing {
        azimuth_deg,
        elevation_deg: elevation_rad.to_degrees().clamp(-90.0, 90.0),
    }
}

pub fn free_space_path_loss_db(range_km: f64, frequency_ghz: f64) -> f64 {
    92.45 + 20.0 * range_km.log10() + 20.0 * frequency_ghz.log10()
}

fn slant_range_km(user_lat_deg: f64, user_lon_deg: f64, sat: SatelliteEphemeris) -> f64 {
    let lat1 = user_lat_deg.to_radians();
    let lon1 = user_lon_deg.to_radians();
    let lat2 = sat.latitude_deg.to_radians();
    let lon2 = sat.longitude_deg.to_radians();
    let central_angle =
        (lat1.sin() * lat2.sin() + lat1.cos() * lat2.cos() * (lon2 - lon1).cos()).acos();
    let orbital_radius = EARTH_RADIUS_KM + sat.altitude_km;
    (EARTH_RADIUS_KM.powi(2) + orbital_radius.powi(2)
        - 2.0 * EARTH_RADIUS_KM * orbital_radius * central_angle.cos())
    .sqrt()
}

fn phased_array_gain_db(element_count: u32, elevation_deg: f64) -> f64 {
    let aperture_gain = 10.0 * (element_count as f64).log10();
    let scan_loss = -3.0 * (1.0 - elevation_deg.to_radians().sin().max(0.15));
    aperture_gain + scan_loss
}

fn beam_quality(elevation_deg: f64, mode: &str) -> f64 {
    let base = 0.48 + (elevation_deg / 90.0) * 0.47;
    let mode_bonus = if mode == "null-steering" { -0.06 } else { 0.02 };
    (base + mode_bonus).clamp(0.2, 0.99)
}

fn modulation_for(c_n_ratio_db: f64) -> &'static str {
    if c_n_ratio_db > 18.0 {
        "16QAM 3/4"
    } else if c_n_ratio_db > 13.0 {
        "QPSK 5/6"
    } else {
        "QPSK 1/2"
    }
}

fn qpsk_ber(eb_n0_db: f64) -> f64 {
    let linear = 10_f64.powf(eb_n0_db / 10.0);
    (0.5 * (-linear).exp()).clamp(1e-9, 0.5)
}

fn hz_to_db(value: f64) -> f64 {
    10.0 * value.log10()
}

fn wave(tick: u64, period: f64) -> f64 {
    ((tick as f64 / period) * PI).sin()
}

fn now_ms() -> u128 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .expect("system time before unix epoch")
        .as_millis()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn pointing_places_due_east_satellite_near_azimuth_90() {
        let pointing = calculate_pointing(
            0.0,
            0.0,
            SatelliteEphemeris {
                latitude_deg: 0.0,
                longitude_deg: 3.0,
                altitude_km: 550.0,
            },
        );

        assert!((pointing.azimuth_deg - 90.0).abs() < 0.01);
        assert!(pointing.elevation_deg > 0.0);
    }

    #[test]
    fn free_space_path_loss_matches_reference_formula() {
        let loss = free_space_path_loss_db(1000.0, 12.0);
        assert!((loss - 174.0336249).abs() < 0.001);
    }

    #[test]
    fn tracker_emits_viable_signal_frame() {
        let mut tracker = AntennaTracker::new(21.0278, 105.8342, LinkBudgetInput::default());
        let frame = tracker.next_frame();

        assert!((0.0..=360.0).contains(&frame.azimuth_deg));
        assert!(frame.path_loss_db > 120.0);
        assert!(frame.beam_quality > 0.2);
    }
}
