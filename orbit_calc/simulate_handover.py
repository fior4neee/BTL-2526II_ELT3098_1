import argparse
import math
import random
import sys
import time
from datetime import datetime, timezone

import requests
from ephemeris_publisher import (
    DEFAULT_TLE_PATH,
    DEFAULT_SELECTED_PATH,
    build_satellite_constellation,
    calculate_satellite_position,
    load_tle_file,
    load_selected_satellites,
    save_selected_satellites,
)

CORE_API = "http://localhost:8081/api/v1"
MIN_ELEVATION_DEG = 15.0
# Minimum interval (seconds) between API calls from this simulator to avoid UI flashing
API_DELAY = 0.1

SATELLITES = [
    {"id": "VNU-LEO-001", "period_min": 90, "inclination": 53.0, "phase_offset": 0},
    {"id": "VNU-LEO-002", "period_min": 90, "inclination": 53.0, "phase_offset": 60},
    {"id": "VNU-LEO-003", "period_min": 90, "inclination": 53.0, "phase_offset": 120},
    {"id": "VNU-LEO-004", "period_min": 90, "inclination": 53.0, "phase_offset": 180},
    {"id": "VNU-LEO-005", "period_min": 90, "inclination": 53.0, "phase_offset": 240},
    {"id": "VNU-LEO-006", "period_min": 90, "inclination": 53.0, "phase_offset": 300},
]


def request_json(path, method="GET", json_body=None):
    url = f"{CORE_API}{path}"
    resp = requests.request(method, url, json=json_body, timeout=10)
    resp.raise_for_status()
    return resp.json()


def get_gateways():
    data = request_json("/gateways")
    raw = data.get("data") if isinstance(data, dict) and "data" in data else data.get("gateways", [])
    gateways = []
    for g in raw:
        loc = g.get("location", {})
        gateways.append({
            "id": g.get("id"),
            "name": g.get("name"),
            "lat": loc.get("lat", 0.0),
            "lon": loc.get("lon", 0.0),
            "alt": loc.get("alt", 0.0),
            "status": g.get("status", "alive"),
        })
    return gateways


def normalize_session(raw):
    if not isinstance(raw, dict):
        return None
    return {
        "id": raw.get("id") or raw.get("session_id"),
        "router_mac": raw.get("router_mac"),
        "satellite_id": raw.get("satellite_id"),
        "current_gateway_id": raw.get("current_gateway_id"),
        "state": raw.get("state"),
    }


def connect_router(mac):
    data = request_json("/router/connect", method="POST", json_body={"router_mac": mac})
    session_raw = data.get("session")
    if not session_raw:
        raise RuntimeError(f"Unexpected router connect response: {data}")
    session = normalize_session(session_raw)
    if not session or not session["id"]:
        raise RuntimeError(f"Missing session id in router connect response: {session_raw}")
    return session


def normalize_longitude(lon):
    lon = ((lon + 180) % 360) - 180
    return lon


def calculate_elevation(sat_pos, gw):
    earth_radius = 6371.0
    sat_radius = earth_radius + sat_pos["altitude"]
    phi_gw = math.radians(gw["lat"])
    phi_sat = math.radians(sat_pos["latitude"])
    delta_lon = math.radians(sat_pos["longitude"] - gw["lon"])
    cos_c = math.sin(phi_gw) * math.sin(phi_sat) + math.cos(phi_gw) * math.cos(phi_sat) * math.cos(delta_lon)
    cos_c = max(-1.0, min(1.0, cos_c))
    central_angle = math.acos(cos_c)
    sin_c = math.sin(central_angle)
    if sin_c == 0:
        return 90.0
    elevation = math.degrees(math.atan2(sat_radius - earth_radius * math.cos(central_angle), earth_radius * sin_c))
    return max(-90.0, min(90.0, elevation))


def choose_best_gateway(sat_pos, gateways):
    best = None
    best_elev = MIN_ELEVATION_DEG
    for gw in gateways:
        if gw["status"] != "alive":
            continue
        elev = calculate_elevation(sat_pos, gw)
        if elev >= best_elev:
            best = gw
            best_elev = elev
    return best, best_elev


def session_for_gateway(sessions, gateway_id):
    for s in sessions:
        if s["current_gateway_id"] == gateway_id:
            return s
    return None


def get_sessions():
    data = request_json("/sessions")
    sessions_raw = data.get("data") if isinstance(data, dict) and "data" in data else data.get("sessions", [])
    return [s for s in (normalize_session(raw) for raw in sessions_raw) if s and s["id"]]


def refresh_sessions(session_ids):
    all_sessions = get_sessions()
    return [s for s in all_sessions if s["id"] in session_ids]


def trigger_handover(session_id, target_gateway_id):
    data = request_json("/handover/trigger", method="POST", json_body={"session_id": session_id, "target_gateway_id": target_gateway_id})
    return data.get("handover")


def disconnect_session(session_id):
    try:
        request_json("/router/disconnect", method="POST", json_body={"session_id": session_id})
        print(f"Disconnected session {session_id}")
    except Exception as exc:
        print(f"Failed to disconnect {session_id}: {exc}")


def make_mac(index: int) -> str:
    return f"02:00:{(index >> 16) & 0xFF:02X}:{(index >> 8) & 0xFF:02X}:{index & 0xFF:02X}:00"


def load_satellites(tle_file, num_satellites):
    # Prefer shared selected satellites if present so publisher and simulator use same set
    selected = load_selected_satellites(DEFAULT_SELECTED_PATH)
    if selected:
        print(f"Loaded {len(selected)} selected satellites from {DEFAULT_SELECTED_PATH}")
        return selected

    satellites = []
    tla_list = []
    if tle_file:
        try:
            tla_list = load_tle_file(tle_file)
            print(f"Loaded {len(tla_list)} satellites from TLE file {tle_file}")
        except Exception as exc:
            print(f"Warning: failed to load TLE file {tle_file}: {exc}")

    if tla_list:
        if num_satellites and num_satellites > 0 and num_satellites < len(tla_list):
            # select a random subset and save for reuse by publisher
            satellites = random.sample(tla_list, num_satellites)
            try:
                save_selected_satellites(DEFAULT_SELECTED_PATH, satellites)
                print(f"Selected and saved {len(satellites)} satellites to {DEFAULT_SELECTED_PATH}")
            except Exception as exc:
                print(f"Warning: failed to save selected satellites: {exc}")
        else:
            satellites = tla_list
    else:
        count = num_satellites if num_satellites and num_satellites > 0 else len(SATELLITES)
        satellites = build_satellite_constellation(count)
        print(f"Using generated fallback constellation with {len(satellites)} satellites")

    return satellites


def create_sessions(num_sessions, gateways):
    sessions = []
    for i in range(num_sessions):
        mac = make_mac(i)
        try:
            session = connect_router(mac)
        except Exception as exc:
            print(f"Failed to create session for {mac}: {exc}")
            continue
        sessions.append(session)
        print(f"Created session {session['id']} at gateway {session['current_gateway_id']} for {mac}")
        # Throttle API calls to avoid UI flashing when creating many sessions
        try:
            time.sleep(API_DELAY)
        except Exception:
            pass
    return sessions


def assign_initial_gateways(sessions, session_satellites, gateways, satellites, current_time):
    for session in sessions:
        sat_id = session_satellites[session["id"]]
        sat = next((s for s in satellites if s["id"] == sat_id), None)
        if not sat:
            continue
        sat_pos = calculate_satellite_position(sat, current_time)
        best_gw, best_elev = choose_best_gateway(sat_pos, gateways)
        if best_gw and best_gw["id"] != session["current_gateway_id"]:
            print(f"Initial handover for {session['id']}: {session['current_gateway_id']} -> {best_gw['id']} (elev {best_elev:.1f})")
            try:
                trigger_handover(session["id"], best_gw["id"])
                try:
                    time.sleep(API_DELAY)
                except Exception:
                    pass
            except Exception as exc:
                print(f"  failed to assign initial gateway for {session['id']}: {exc}")


def simulate_handover_step(steps, sessions, session_satellites, gateways, satellites, current_time):
    for session in sessions:
        sat_id = session_satellites.get(session["id"])
        sat = next((s for s in satellites if s["id"] == sat_id), None)
        if not sat:
            continue

        sat_pos = calculate_satellite_position(sat, current_time)
        best_gw, best_elev = choose_best_gateway(sat_pos, gateways)
        current_gw = next((g for g in gateways if g["id"] == session["current_gateway_id"]), None)
        current_elev = calculate_elevation(sat_pos, current_gw) if current_gw else -999.0

        print(f"Session {session['id']} satellite {sat_id}: current {session['current_gateway_id']} elev {current_elev:.1f}, best {best_gw['id'] if best_gw else 'none'} elev {best_elev:.1f}")

        if best_gw is None:
            print(f"  no gateway visible for session {session['id']} (satellite over ocean or out of region)")
            continue

        if best_gw["id"] != session["current_gateway_id"] and (current_elev < MIN_ELEVATION_DEG or best_elev > current_elev + 2.0):
            try:
                ho = trigger_handover(session["id"], best_gw["id"])
                print(f"  handover {ho['id']} for {session['id']}: {session['current_gateway_id']} -> {best_gw['id']}")
                try:
                    time.sleep(API_DELAY)
                except Exception:
                    pass
            except Exception as exc:
                print(f"  failed to handover {session['id']}: {exc}")
        else:
            print(f"  staying on {session['current_gateway_id']} (elev {current_elev:.1f})")


def main():
    parser = argparse.ArgumentParser(description="Simulate realistic satellite-driven handovers using orbit geometry.")
    parser.add_argument("--sessions", type=int, default=2, help="Number of simulated active router sessions")
    parser.add_argument("--satellites", type=int, default=None, help="Number of satellites to load or generate")
    parser.add_argument("--tle-file", type=str, default=str(DEFAULT_TLE_PATH), help="TLE file to load satellite definitions from")
    parser.add_argument("--rounds", type=int, default=2, help="Number of handover rounds to simulate")
    parser.add_argument("--delay", type=float, default=5.0, help="Seconds to wait between simulation steps")
    parser.add_argument("--continuous", action="store_true", help="Run continuously until interrupted")
    args = parser.parse_args()

    if args.sessions < 1:
        print("Warning: session count must be at least 1. Using 1.")
        args.sessions = 1
    if args.rounds < 1:
        args.rounds = 1
    if args.continuous:
        print("Running in continuous mode. Press Ctrl+C to stop.")

    # Map simulation delay to API update interval (throttle) to avoid UI flashing
    global API_DELAY
    API_DELAY = max(0.05, args.delay / 5.0)
    print(f"API call interval set to {API_DELAY:.2f}s based on --delay {args.delay}s")

    if args.satellites is not None and args.satellites < 1:
        print("Warning: satellite count must be at least 1. Ignoring --satellites.")
        args.satellites = None

    print("Fetching gateway list...")
    gateways = get_gateways()
    if not gateways:
        print("No gateways found. Is core_network running?")
        sys.exit(1)

    satellites = load_satellites(args.tle_file, args.satellites)
    print(f"Loaded {len(satellites)} satellite definitions.")
    print(f"Loaded {len(gateways)} gateways.")

    session_satellites = {}
    sessions = create_sessions(args.sessions, gateways)
    if not sessions:
        print("No sessions were created. Exiting.")
        sys.exit(1)

    for i, session in enumerate(sessions):
        session_satellites[session["id"]] = satellites[i % len(satellites)]["id"]

    current_time = datetime.now(timezone.utc)
    assign_initial_gateways(sessions, session_satellites, gateways, satellites, current_time)
    sessions = refresh_sessions([s["id"] for s in sessions])

    round_idx = 1
    try:
        while True:
            current_time = datetime.now(timezone.utc)
            print(f"\n=== Simulation step {round_idx} @ {current_time.isoformat()} ===")
            simulate_handover_step(round_idx, sessions, session_satellites, gateways, satellites, current_time)
            sessions = refresh_sessions([s["id"] for s in sessions])

            if not args.continuous and round_idx >= args.rounds:
                break

            round_idx += 1
            print(f"Waiting {args.delay} seconds before next step...")
            time.sleep(args.delay)
    except KeyboardInterrupt:
        print("\nSimulation stopped by user.")

    print("\nFinal session state:")
    for session in sessions:
        print(f" - {session['id']} @ {session['current_gateway_id']} ({session['state']})")

    # Disconnect all created sessions so gateway counters are decremented
    print("\nDisconnecting created sessions...")
    for session in sessions:
        disconnect_session(session["id"])
        try:
            time.sleep(API_DELAY)
        except Exception:
            pass

    print("Done.")


if __name__ == "__main__":
    main()
