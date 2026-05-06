use serde::Serialize;

#[derive(Debug, Clone, Serialize)]
pub struct LocationFrame {
    pub timestamp_ms: u128,
    pub latitude: f64,
    pub longitude: f64,
    pub altitude_m: f64,
    pub plan_name: String,
    pub plan_type: String,
    pub geofence_status: String,
    pub monthly_used_gb: f64,
    pub monthly_cap_gb: f64,
    pub session_duration_s: u64,
    pub session_down_gb: f64,
    pub session_up_gb: f64,
    pub estimated_cost_usd: f64,
}

pub struct GeoReporter {
    home_latitude: f64,
    home_longitude: f64,
    fence_radius_km: f64,
    tick: u64,
}

impl GeoReporter {
    pub fn fixed_hanoi() -> Self {
        Self {
            home_latitude: 21.0278,
            home_longitude: 105.8342,
            fence_radius_km: 2.0,
            tick: 0,
        }
    }

    pub fn next_frame(&mut self) -> LocationFrame {
        self.tick += 1;
        let latitude = self.home_latitude + (self.tick as f64 / 90.0).sin() * 0.012;
        let longitude = self.home_longitude + (self.tick as f64 / 110.0).cos() * 0.014;
        let distance_km =
            haversine_km(self.home_latitude, self.home_longitude, latitude, longitude);
        let geofence_status = if distance_km <= self.fence_radius_km {
            "inside"
        } else if distance_km <= self.fence_radius_km * 1.25 {
            "warning"
        } else {
            "breach"
        };

        LocationFrame {
            timestamp_ms: now_ms(),
            latitude,
            longitude,
            altitude_m: 14.0,
            plan_name: "Fixed Business 300".to_string(),
            plan_type: "Fixed".to_string(),
            geofence_status: geofence_status.to_string(),
            monthly_used_gb: 812.4 + self.tick as f64 * 0.008,
            monthly_cap_gb: 1500.0,
            session_duration_s: self.tick,
            session_down_gb: self.tick as f64 * 0.00082,
            session_up_gb: self.tick as f64 * 0.00011,
            estimated_cost_usd: 118.5 + self.tick as f64 * 0.002,
        }
    }
}

pub fn haversine_km(lat1: f64, lon1: f64, lat2: f64, lon2: f64) -> f64 {
    let radius_km = 6371.0;
    let d_lat = (lat2 - lat1).to_radians();
    let d_lon = (lon2 - lon1).to_radians();
    let lat1 = lat1.to_radians();
    let lat2 = lat2.to_radians();
    let a = (d_lat / 2.0).sin().powi(2) + lat1.cos() * lat2.cos() * (d_lon / 2.0).sin().powi(2);
    2.0 * radius_km * a.sqrt().asin()
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
    fn haversine_reports_short_hanoi_drift() {
        let distance = haversine_km(21.0278, 105.8342, 21.0288, 105.8352);
        assert!(distance > 0.1);
        assert!(distance < 0.2);
    }

    #[test]
    fn geo_reporter_emits_fixed_plan_status() {
        let mut reporter = GeoReporter::fixed_hanoi();
        let frame = reporter.next_frame();
        assert_eq!(frame.plan_type, "Fixed");
        assert!(frame.monthly_cap_gb > frame.monthly_used_gb);
    }
}
