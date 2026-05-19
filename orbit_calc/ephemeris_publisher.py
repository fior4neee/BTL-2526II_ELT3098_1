"""
Ephemeris Data Publisher
Sends simulated satellite positions to core_network API
Run: python ephemeris_publisher.py
"""

import argparse
import json
import time
import requests
from datetime import datetime, timezone, timedelta
import math
import random
from pathlib import Path

from sgp4.api import Satrec, jday

CORE_NETWORK_URL = "http://localhost:8081/api/v1"
DEFAULT_TLE_PATH = Path(__file__).resolve().parent / "outputs" / "simulation" / "internet" / "vnu_leo.tle"
DEFAULT_SELECTED_PATH = Path(__file__).resolve().parent / "selected_sats.json"

# Fallback simulated satellite constellation (if no TLE file is available)
SATELLITES = [
    {"id": "VNU-LEO-001", "period_min": 90, "inclination": 97.5, "phase_offset": 0},
    {"id": "VNU-LEO-002", "period_min": 90, "inclination": 97.5, "phase_offset": 60},
    {"id": "VNU-LEO-003", "period_min": 90, "inclination": 97.5, "phase_offset": 120},
    {"id": "VNU-LEO-004", "period_min": 90, "inclination": 97.5, "phase_offset": 180},
    {"id": "VNU-LEO-005", "period_min": 90, "inclination": 97.5, "phase_offset": 240},
    {"id": "VNU-LEO-006", "period_min": 90, "inclination": 97.5, "phase_offset": 300},
]

# Vietnam gateway locations
GATEWAYS = [
    {"name": "Hanoi GW", "lat": 21.0285, "lon": 105.8542},
    {"name": "Danang GW", "lat": 16.0544, "lon": 108.2022},
    {"name": "HCMC GW", "lat": 10.7769, "lon": 106.7009},
]


def normalize_longitude(lon):
    lon = ((lon + 180) % 360) - 180
    return lon


def utc_datetime_to_julian(dt):
    return jday(dt.year, dt.month, dt.day, dt.hour, dt.minute, dt.second + dt.microsecond / 1e6)


def gmst_from_datetime(dt):
    jd, fr = utc_datetime_to_julian(dt)
    T = ((jd + fr) - 2451545.0) / 36525.0
    gmst = 280.46061837 + 360.98564736629 * ((jd + fr) - 2451545.0)
    gmst += 0.000387933 * T * T - (T * T * T) / 38710000.0
    return math.radians(gmst % 360.0)


def teme_to_ecef(position_teme, dt):
    theta = gmst_from_datetime(dt)
    x, y, z = position_teme
    x_ecef = x * math.cos(theta) + y * math.sin(theta)
    y_ecef = -x * math.sin(theta) + y * math.cos(theta)
    return x_ecef, y_ecef, z


def ecef_to_geodetic(x, y, z):
    a = 6378.137
    f = 1.0 / 298.257223563
    e2 = f * (2 - f)
    r = math.hypot(x, y)
    lon = math.atan2(y, x)
    lat = math.atan2(z, r * (1 - e2))

    for _ in range(5):
        N = a / math.sqrt(1 - e2 * math.sin(lat) ** 2)
        alt = r / math.cos(lat) - N
        lat = math.atan2(z, r * (1 - e2 * N / (N + alt)))

    N = a / math.sqrt(1 - e2 * math.sin(lat) ** 2)
    alt = r / math.cos(lat) - N
    return math.degrees(lat), normalize_longitude(math.degrees(lon)), alt


def propagate_tle_satellite(sat, current_time):
    satrec = Satrec.twoline2rv(sat["line1"], sat["line2"])
    jd, fr = utc_datetime_to_julian(current_time)
    err_code, position, _velocity = satrec.sgp4(jd, fr)
    if err_code != 0:
        return None

    lat, lon, alt = ecef_to_geodetic(*teme_to_ecef(position, current_time))
    return {
        "latitude": lat,
        "longitude": lon,
        "altitude": alt,
    }


def calculate_satellite_position(sat, current_time):
    if "line1" in sat and "line2" in sat:
        return propagate_tle_satellite(sat, current_time)

    time_offset_sec = (current_time - current_time.replace(hour=0, minute=0, second=0, microsecond=0)).total_seconds()
    period = sat["period_min"] * 60  # Convert to seconds
    mean_altitude = 600  # km
    earth_radius = 6371  # km

    n = 2 * math.pi / period
    mean_anomaly = (n * time_offset_sec) % (2 * math.pi)

    inclination = sat["inclination"]
    if inclination > 90:
        inclination = 180 - inclination
    latitude = math.sin(mean_anomaly) * inclination
    longitude = normalize_longitude(math.degrees(mean_anomaly) + sat.get("phase_offset", 0))

    return {
        "latitude": latitude,
        "longitude": longitude,
        "altitude": mean_altitude,
    }


def load_tle_file(file_path, max_satellites=None):
    path = Path(file_path)
    if not path.exists():
        raise FileNotFoundError(f"TLE file not found: {path}")

    lines = [line.strip() for line in path.read_text().splitlines() if line.strip()]
    if len(lines) % 3 != 0:
        raise ValueError("TLE file does not contain a valid triplet of name/line1/line2 entries")

    satellites = []
    for idx in range(0, len(lines), 3):
        name = lines[idx]
        line1 = lines[idx + 1]
        line2 = lines[idx + 2]
        satellites.append({"id": name, "line1": line1, "line2": line2})
        if max_satellites and len(satellites) >= max_satellites:
            break
    return satellites


def save_selected_satellites(path, satellites):
    p = Path(path)
    out = []
    for s in satellites:
        entry = {"id": s.get("id")}
        if "line1" in s and "line2" in s:
            entry["line1"] = s["line1"]
            entry["line2"] = s["line2"]
        else:
            entry["period_min"] = s.get("period_min")
            entry["inclination"] = s.get("inclination")
            entry["phase_offset"] = s.get("phase_offset")
        out.append(entry)
    p.write_text(json.dumps(out, indent=2))


def load_selected_satellites(path):
    p = Path(path)
    if not p.exists():
        return None
    raw = json.loads(p.read_text())
    return raw


def calculate_elevation_angle(sat_lat, sat_lon, sat_alt, gw_lat, gw_lon):
    """Calculate elevation angle from gateway to satellite using spherical geometry."""
    earth_radius = 6371.0  # km
    sat_radius = earth_radius + sat_alt

    # Convert to radians
    phi_gw = math.radians(gw_lat)
    phi_sat = math.radians(sat_lat)
    delta_lon = math.radians(sat_lon - gw_lon)

    # Central angle between gateway and satellite subpoint on Earth
    cos_c = math.sin(phi_gw) * math.sin(phi_sat) + math.cos(phi_gw) * math.cos(phi_sat) * math.cos(delta_lon)
    cos_c = max(-1.0, min(1.0, cos_c))
    central_angle = math.acos(cos_c)

    # Range from gateway to satellite
    sin_c = math.sin(central_angle)
    if sin_c == 0:
        return 90.0

    # Elevation formula for a spherical Earth
    elevation = math.degrees(math.atan2(sat_radius - earth_radius * math.cos(central_angle), earth_radius * sin_c))
    return max(-90.0, min(90.0, elevation))


def publish_ephemeris(satellites):
    """Send current satellite ephemeris to core_network."""
    current_time = datetime.now(timezone.utc)
    midnight = current_time.replace(hour=0, minute=0, second=0, microsecond=0)
    time_offset = (current_time - midnight).total_seconds()

    ephemeris_data = []

    for sat in satellites:
        pos = calculate_satellite_position(sat, current_time)
        if not pos:
            continue

        best_elevation = -90
        for gw in GATEWAYS:
            elev = calculate_elevation_angle(pos["latitude"], pos["longitude"], pos["altitude"], gw["lat"], gw["lon"])
            if elev > best_elevation:
                best_elevation = elev

        if best_elevation > 0:
            ephemeris_data.append({
                "satellite_id": sat["id"],
                "latitude": pos["latitude"],
                "longitude": pos["longitude"],
                "altitude": pos["altitude"],
                "elevation": best_elevation,
                "azimuth": (math.degrees(time_offset / 3600) * 10) % 360,
                "timestamp": current_time.isoformat(),
            })

    if ephemeris_data:
        try:
            response = requests.post(
                f"{CORE_NETWORK_URL}/ephemeris/update",
                json={"data": ephemeris_data},
                timeout=5,
            )
            print(f"[{datetime.now().strftime('%H:%M:%S')}] Sent {len(ephemeris_data)} ephemeris records → {response.status_code}")
        except Exception as e:
            print(f"[{datetime.now().strftime('%H:%M:%S')}] Error: {e}")
    else:
        print(f"[{datetime.now().strftime('%H:%M:%S')}] No satellites visible")


def build_satellite_constellation(count):
    """Generate a satellite constellation with the requested number of satellites."""
    if count < 1:
        raise ValueError("Satellite count must be at least 1")

    base_sat = SATELLITES[0]
    satellites = []
    for idx in range(count):
        phase_offset = (360.0 / count) * idx
        satellites.append({
            "id": f"VNU-LEO-{idx + 1:03d}",
            "period_min": base_sat["period_min"],
            "inclination": base_sat["inclination"],
            "phase_offset": phase_offset,
        })
    return satellites


def main():
    parser = argparse.ArgumentParser(description="Ephemeris publisher for VNU-LEO simulated constellation.")
    parser.add_argument("--satellites", type=int, default=None, help="Number of satellites in the simulated constellation")
    parser.add_argument("--tle-file", type=str, default=str(DEFAULT_TLE_PATH), help="TLE file to load satellite definitions from")
    parser.add_argument("--interval", type=float, default=10.0, help="Seconds between ephemeris updates")
    args = parser.parse_args()

    satellites = []
    tla_list = []
    if args.tle_file:
        try:
            tla_list = load_tle_file(args.tle_file)
        except Exception as e:
            print(f"Warning: failed to load TLE file {args.tle_file}: {e}")
            print("Falling back to generated constellation.")

    # If a selected_sats file exists, use it to ensure consistent selection across tools
    selected = load_selected_satellites(DEFAULT_SELECTED_PATH)
    if selected:
        satellites = selected
        print(f"Using previously selected {len(satellites)} satellites from {DEFAULT_SELECTED_PATH}")
    else:
        if args.satellites and tla_list:
            # pick random subset from TLE list
            k = min(args.satellites, len(tla_list))
            satellites = random.sample(tla_list, k)
            save_selected_satellites(DEFAULT_SELECTED_PATH, satellites)
            print(f"Selected and saved {len(satellites)} satellites to {DEFAULT_SELECTED_PATH}")
        elif tla_list:
            satellites = tla_list
        else:
            count = args.satellites if args.satellites and args.satellites > 0 else len(SATELLITES)
            satellites = build_satellite_constellation(count)

    print("=== VNU-LEO Ephemeris Publisher ===")
    print(f"Target: {CORE_NETWORK_URL}/ephemeris/update")
    print(f"Simulating {len(satellites)} satellites")
    print(f"Sending updates every {args.interval} seconds...\n")

    try:
        while True:
            publish_ephemeris(satellites)
            time.sleep(args.interval)
    except KeyboardInterrupt:
        print("\nShutdown requested.")


if __name__ == "__main__":
    main()
