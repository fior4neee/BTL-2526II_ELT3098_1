"""
Ephemeris Data Publisher
Sends simulated satellite positions to core_network API
Run: python ephemeris_publisher.py
"""

import json
import time
import requests
from datetime import datetime, timedelta
import math

CORE_NETWORK_URL = "http://localhost:8081/api/v1"

# Simulated satellite constellation (3 satellites in orbit)
SATELLITES = [
    {"id": "VNU-LEO-001", "period_min": 90, "inclination": 97.5},
    {"id": "VNU-LEO-002", "period_min": 90, "inclination": 97.5},
    {"id": "VNU-LEO-003", "period_min": 90, "inclination": 97.5},
]

# Vietnam gateway locations
GATEWAYS = [
    {"name": "Hanoi GW", "lat": 21.0285, "lon": 105.8542},
    {"name": "Danang GW", "lat": 16.0544, "lon": 108.2022},
    {"name": "HCMC GW", "lat": 10.7769, "lon": 106.7009},
]


def calculate_satellite_position(sat_id, time_offset_sec):
    """Simulate satellite position using circular orbit approximation."""
    sat = next((s for s in SATELLITES if s["id"] == sat_id), None)
    if not sat:
        return None

    # Orbital parameters (simplified)
    period = sat["period_min"] * 60  # Convert to seconds
    mean_altitude = 600  # km
    earth_radius = 6371  # km
    orbital_radius = earth_radius + mean_altitude

    # Mean motion (rad/sec)
    n = 2 * math.pi / period

    # Mean anomaly (progress around orbit)
    mean_anomaly = (n * time_offset_sec) % (2 * math.pi)

    # Simplified position (longitude sweeps, latitude is constant)
    lon_offset = math.degrees(mean_anomaly)
    lat = sat["inclination"]

    return {
        "latitude": lat,
        "longitude": lon_offset % 360,
        "altitude": mean_altitude,
    }


def calculate_elevation_angle(sat_lat, sat_lon, gw_lat, gw_lon):
    """Calculate elevation angle from gateway to satellite (simplified)."""
    # Rough distance calculation
    dlat = sat_lat - gw_lat
    dlon = sat_lon - gw_lon

    distance = math.sqrt(dlat**2 + dlon**2) * 111.0  # km (rough conversion)

    # Simplified elevation angle (higher when closer)
    if distance < 100:
        elevation = 45 + (100 - distance) * 0.3
    elif distance < 2000:
        elevation = 10 + (2000 - distance) * 0.02
    else:
        elevation = -5

    elevation = max(-90, min(90, elevation))
    return elevation


def publish_ephemeris():
    """Send current satellite ephemeris to core_network."""
    current_time = datetime.utcnow()
    time_offset = (current_time - datetime.utcnow().replace(hour=0, minute=0, second=0, microsecond=0)).total_seconds()

    ephemeris_data = []

    for sat in SATELLITES:
        pos = calculate_satellite_position(sat["id"], time_offset)
        if not pos:
            continue

        # Find best gateway (closest one)
        best_elevation = -90
        for gw in GATEWAYS:
            elev = calculate_elevation_angle(pos["latitude"], pos["longitude"], gw["lat"], gw["lon"])
            if elev > best_elevation:
                best_elevation = elev

        if best_elevation > 0:  # Only if visible
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


def main():
    """Publish ephemeris data every 10 seconds."""
    print("=== VNU-LEO Ephemeris Publisher ===")
    print(f"Target: {CORE_NETWORK_URL}/ephemeris/update")
    print("Sending updates every 10 seconds...\n")

    try:
        while True:
            publish_ephemeris()
            time.sleep(10)
    except KeyboardInterrupt:
        print("\nShutdown requested.")


if __name__ == "__main__":
    main()
