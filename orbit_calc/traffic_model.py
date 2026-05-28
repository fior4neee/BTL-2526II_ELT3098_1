"""
traffic_model.py
================
VNU-LEO Link Budget & Traffic Model.

Computes:
  - Free-space path loss (FSPL)
  - Atmospheric / rain attenuation (ITU-R P.618 simplified)
  - Link margin and C/N ratio
  - Capacity estimates (Shannon bound) for two service profiles
  - Dwell time (satellite visibility window) per pass

References
----------
- ITU-R P.618-13: Propagation data for satellite systems
- Roddy, D. (2006). Satellite Communications, 4th ed. McGraw-Hill.
- Pratt, T., Bostian, C., Allnutt, J. (2003). Satellite Communications, 2nd ed.
"""

from __future__ import annotations

import json
import math
from dataclasses import dataclass, field, asdict
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional

import numpy as np

# ---------------------------------------------------------------------------
# Physical / link constants
# ---------------------------------------------------------------------------
C_M_S = 299_792_458.0          # Speed of light [m/s]
C_KM_S = C_M_S / 1_000.0
KB_JK = 1.380649e-23            # Boltzmann constant [J/K]
KB_DBW_HZ_K = 10 * math.log10(KB_JK)   # ≈ -228.6 dBW/Hz/K
RE_KM = 6371.0
MU_KM3S2 = 398600.4418


# ---------------------------------------------------------------------------
# Antenna & terminal parameters
# ---------------------------------------------------------------------------

@dataclass
class Antenna:
    """Simplified antenna model."""
    diameter_m: float           # Aperture diameter [m]
    efficiency: float = 0.55    # Aperture efficiency (0–1)
    noise_temp_K: float = 150.0 # System noise temperature [K]
    frequency_ghz: float = 14.0 # Centre frequency [GHz]

    @property
    def wavelength_m(self) -> float:
        return C_M_S / (self.frequency_ghz * 1e9)

    @property
    def gain_dbi(self) -> float:
        """Peak gain [dBi] — parabolic aperture formula."""
        if self.diameter_m <= 0:
            return 0.0
        g_linear = self.efficiency * (math.pi * self.diameter_m / self.wavelength_m) ** 2
        return 10 * math.log10(max(g_linear, 1.0))

    @property
    def g_over_t_db(self) -> float:
        """G/T [dB/K] — figure of merit."""
        return self.gain_dbi - 10 * math.log10(self.noise_temp_K)


@dataclass
class SatelliteTerminal:
    """Satellite transponder / payload parameters."""
    tx_power_w: float           # Transmit power [W]
    tx_antenna_gain_dbi: float  # Transmit antenna gain [dBi]
    rx_antenna_gain_dbi: float  # Receive antenna gain [dBi]
    noise_figure_db: float = 3.0
    noise_temp_K: float = 500.0
    frequency_ghz: float = 14.0

    @property
    def eirp_dbw(self) -> float:
        return 10 * math.log10(max(self.tx_power_w, 1e-9)) + self.tx_antenna_gain_dbi

    @property
    def g_over_t_db(self) -> float:
        return self.rx_antenna_gain_dbi - 10 * math.log10(self.noise_temp_K)


# ---------------------------------------------------------------------------
# Link budget dataclass
# ---------------------------------------------------------------------------

@dataclass
class LinkBudget:
    """Full link budget result for one link direction."""
    service: str
    direction: str              # "downlink" or "uplink"
    altitude_km: float
    elevation_deg: float
    frequency_ghz: float

    # TX side
    tx_power_dbw: float
    tx_gain_dbi: float
    eirp_dbw: float

    # Path
    fspl_db: float
    atmospheric_loss_db: float
    total_path_loss_db: float
    slant_range_km: float

    # RX side
    rx_gain_dbi: float
    rx_noise_temp_k: float
    g_over_t_db: float

    # Figures of merit
    cn0_dbhz: float             # C/N₀ [dBHz]
    cn_db: float                # C/N [dB] for given noise bandwidth
    noise_bandwidth_hz: float
    eb_n0_db: float             # Eₓ/N₀ [dB]
    bitrate_bps: float
    required_eb_n0_db: float    # for target BER
    link_margin_db: float       # eb_n0 − required_eb_n0

    # Capacity
    shannon_capacity_mbps: float


# ---------------------------------------------------------------------------
# Path loss & atmospheric models
# ---------------------------------------------------------------------------

def free_space_path_loss_db(slant_range_km: float, frequency_ghz: float) -> float:
    """FSPL = 20·log₁₀(4π·d·f / c) [dB]."""
    d_m = slant_range_km * 1e3
    f_hz = frequency_ghz * 1e9
    fspl = 20 * math.log10(4 * math.pi * d_m * f_hz / C_M_S)
    return fspl


def rain_attenuation_db(frequency_ghz: float, elevation_deg: float,
                         rain_rate_mm_hr: float = 20.0,
                         latitude_deg: float = 16.0) -> float:
    """Simplified rain attenuation using ITU-R P.618 power-law model.

    Uses ITU-R P.838 coefficients for linear polarisation.
    This is a simplified version; full model requires rain height and
    effective path length calculation.
    """
    # ITU-R P.838 coefficients (vertical polarisation, approximated)
    # For Ku-band (≈14 GHz): k≈0.0851, α≈1.310
    # For Ka-band (≈30 GHz): k≈0.187,  α≈1.021
    freq_coeffs = {
        # freq_ghz: (k, alpha)
        10: (0.0101, 1.276),
        14: (0.0315, 1.154),
        20: (0.0751, 1.099),
        30: (0.187,  1.021),
    }
    # Interpolate
    freqs = sorted(freq_coeffs.keys())
    if frequency_ghz <= freqs[0]:
        k, alpha = freq_coeffs[freqs[0]]
    elif frequency_ghz >= freqs[-1]:
        k, alpha = freq_coeffs[freqs[-1]]
    else:
        for i in range(len(freqs) - 1):
            if freqs[i] <= frequency_ghz <= freqs[i + 1]:
                t = (frequency_ghz - freqs[i]) / (freqs[i + 1] - freqs[i])
                k = freq_coeffs[freqs[i]][0] * (1 - t) + freq_coeffs[freqs[i + 1]][0] * t
                alpha = freq_coeffs[freqs[i]][1] * (1 - t) + freq_coeffs[freqs[i + 1]][1] * t
                break

    # Specific attenuation [dB/km]
    gamma_r = k * (rain_rate_mm_hr ** alpha)

    # Effective path length (simplified: L_eff = h_rain / sin(elevation))
    # Vietnam rain height ≈ 4.5 km (tropical)
    h_rain_km = 4.5
    el_rad = math.radians(max(elevation_deg, 5.0))
    L_eff_km = h_rain_km / math.sin(el_rad)
    L_eff_km = min(L_eff_km, 10.0)    # cap at 10 km for very low elevation

    # Reduction factor (ITU-R P.618 simplified)
    r = 1.0 / (1.0 + 0.78 * math.sqrt(L_eff_km * gamma_r / frequency_ghz)
                - 0.38 * (1 - math.exp(-2 * L_eff_km)))
    r = max(0.1, min(r, 1.0))

    A_rain = gamma_r * L_eff_km * r
    return max(0.0, A_rain)


def atmospheric_loss_db(frequency_ghz: float, elevation_deg: float) -> float:
    """Total atmospheric loss = gaseous (O2 + H2O) + rain attenuation [dB].

    Gaseous values are typical clear-sky averages for tropical Vietnam.
    """
    el_rad = math.radians(max(elevation_deg, 5.0))
    # Zenith gaseous attenuation (simplified, tropical humid atmosphere)
    if frequency_ghz < 10:
        gaseous_zenith = 0.1
    elif frequency_ghz < 20:
        gaseous_zenith = 0.15 + (frequency_ghz - 10) * 0.02
    else:
        gaseous_zenith = 0.35

    # Path through atmosphere: zenith / sin(elevation)
    gaseous = gaseous_zenith / math.sin(el_rad)
    rain = rain_attenuation_db(frequency_ghz, elevation_deg)
    return gaseous + rain


def slant_range_from_elevation(altitude_km: float, elevation_deg: float) -> float:
    """Slant range [km] using exact spherical-Earth formula."""
    el_rad = math.radians(elevation_deg)
    a = RE_KM + altitude_km
    sr = math.sqrt(a ** 2 - (RE_KM * math.cos(el_rad)) ** 2) - RE_KM * math.sin(el_rad)
    return max(sr, altitude_km)


# ---------------------------------------------------------------------------
# Service profiles
# ---------------------------------------------------------------------------

@dataclass
class ServiceProfile:
    """Defines requirements for a given service tier."""
    name: str
    altitude_km: float
    frequency_ghz: float          # operating frequency
    bitrate_bps: float            # target user bitrate
    required_eb_n0_db: float      # BER = 1e-6 for chosen modcod
    modcod: str                   # e.g. "QPSK 1/2", "16APSK 3/4"
    noise_bandwidth_hz: float     # = bitrate / spectral_efficiency
    latency_budget_ms: float      # end-to-end one-way budget
    availability_target_pct: float = 99.0
    min_elevation_deg: float = 15.0

    # User terminal
    ut_diameter_m: float = 0.45
    ut_efficiency: float = 0.55
    ut_noise_temp_k: float = 150.0
    ut_tx_power_w: float = 4.0

    # Satellite terminal
    sat_tx_power_w: float = 40.0
    sat_tx_gain_dbi: float = 36.0
    sat_rx_gain_dbi: float = 30.0
    sat_noise_temp_k: float = 500.0


INTERNET_VOIP_PROFILE = ServiceProfile(
    name="Internet/VoIP",
    altitude_km=1200.0,
    frequency_ghz=14.0,       # Ku-band uplink (user → sat)
    bitrate_bps=10e6,         # 10 Mbps per user
    required_eb_n0_db=6.5,    # QPSK 1/2, DVB-S2 target BER 1e-6
    modcod="QPSK 1/2",
    noise_bandwidth_hz=10e6,  # ≈ bitrate (spectral eff ≈ 1 bit/s/Hz for QPSK 1/2)
    latency_budget_ms=50.0,
    availability_target_pct=99.0,
    min_elevation_deg=15.0,
    ut_diameter_m=0.45,
    ut_tx_power_w=4.0,
    sat_tx_power_w=40.0,
    sat_tx_gain_dbi=36.0,
    sat_rx_gain_dbi=30.0,
)

DATA_WEATHER_PROFILE = ServiceProfile(
    name="Data/Weather Imaging",
    altitude_km=1200.0,
    frequency_ghz=26.0,       # Ka-band for higher throughput
    bitrate_bps=100e6,        # 100 Mbps burst
    required_eb_n0_db=9.4,    # 16APSK 3/4
    modcod="16APSK 3/4",
    noise_bandwidth_hz=50e6,
    latency_budget_ms=200.0,
    availability_target_pct=95.0,  # weather imaging tolerates lower availability
    min_elevation_deg=15.0,
    ut_diameter_m=1.2,        # larger dish for data imaging stations
    ut_tx_power_w=20.0,
    sat_tx_power_w=80.0,
    sat_tx_gain_dbi=40.0,
    sat_rx_gain_dbi=34.0,
)


# ---------------------------------------------------------------------------
# Traffic / dwell time model
# ---------------------------------------------------------------------------

def dwell_time_s(altitude_km: float, el_min_deg: float = 15.0,
                 latitude_deg: float = 16.0,
                 inclination_deg: float = 53.0) -> float:
    """Maximum dwell time for a satellite pass over a ground station.

    Uses the spherical-Earth geometry: the satellite is visible while
    the Earth-central angle η < ρ (coverage half-angle).

    Approximate formula (overhead pass worst case):
        t_dwell = T_orbit / π · arcsin(sin(ρ) / cos(lat))
    for inclined orbit.

    Returns dwell time [s].
    """
    a = RE_KM + altitude_km
    T_s = 2.0 * math.pi * math.sqrt(a ** 3 / MU_KM3S2)

    el_rad = math.radians(el_min_deg)
    rho = math.acos(RE_KM * math.cos(el_rad) / (RE_KM + altitude_km)) - el_rad  # Earth central angle

    lat_rad = math.radians(latitude_deg)
    inc_rad = math.radians(inclination_deg)

    # Maximum ground-track length visible = 2*rho (at best, direct overhead)
    # Angular speed of satellite over ground:
    #   omega_sat = 2*pi / T_orbit (approx, ignoring Earth rotation for dwell)
    # Dwell ≈ 2*rho / (omega_sat) = T_orbit * rho / pi
    t_max = T_s * rho / math.pi

    # Correction for latitude vs. inclination (not directly overhead)
    if abs(latitude_deg) < inclination_deg:
        correction = math.sqrt(1.0 - (math.sin(lat_rad) / math.sin(inc_rad)) ** 2)
        t_max *= min(1.0, 1.0 / (correction + 1e-6))
        t_max = min(t_max, T_s * rho / math.pi * 2)  # cap at geometric max

    return t_max


def data_volume_per_pass_gb(altitude_km: float, bitrate_bps: float,
                              el_min_deg: float = 15.0,
                              latitude_deg: float = 16.0) -> float:
    """Estimated data volume [GB] transferable in a single satellite pass."""
    t = dwell_time_s(altitude_km, el_min_deg, latitude_deg)
    bits = bitrate_bps * t
    return bits / 8e9


# ---------------------------------------------------------------------------
# Link budget calculator
# ---------------------------------------------------------------------------

class TrafficModel:
    """Compute link budgets and traffic capacity for VNU-LEO services.

    Parameters
    ----------
    profile : ServiceProfile
        One of INTERNET_VOIP_PROFILE or DATA_WEATHER_PROFILE.
    elevation_deg : float
        Link elevation angle for the budget calculation.
    """

    def __init__(self, profile: ServiceProfile, elevation_deg: float = 30.0):
        self.profile = profile
        self.elevation_deg = elevation_deg

    # ------------------------------------------------------------------
    # Core link budget
    # ------------------------------------------------------------------

    def compute_downlink(self) -> LinkBudget:
        """Compute downlink (satellite → user terminal) budget."""
        p = self.profile
        el = self.elevation_deg
        f = p.frequency_ghz
        h = p.altitude_km

        # User terminal receive antenna
        ut = Antenna(
            diameter_m=p.ut_diameter_m,
            efficiency=p.ut_efficiency,
            noise_temp_K=p.ut_noise_temp_k,
            frequency_ghz=f,
        )

        # Satellite EIRP
        tx_power_dbw = 10 * math.log10(p.sat_tx_power_w)
        eirp = tx_power_dbw + p.sat_tx_gain_dbi

        # Path
        sr_km = slant_range_from_elevation(h, el)
        fspl = free_space_path_loss_db(sr_km, f)
        atm = atmospheric_loss_db(f, el)
        total_loss = fspl + atm

        # Received C/N₀
        rx_g = ut.gain_dbi
        g_t = ut.g_over_t_db
        cn0 = eirp - total_loss + g_t - KB_DBW_HZ_K

        # C/N for noise bandwidth
        bw_hz = p.noise_bandwidth_hz
        cn = cn0 - 10 * math.log10(bw_hz)

        # Eb/N0
        br = p.bitrate_bps
        eb_n0 = cn0 - 10 * math.log10(br)

        margin = eb_n0 - p.required_eb_n0_db

        # Shannon capacity (linear SNR → capacity)
        snr_linear = 10 ** (cn / 10.0)
        shannon_mbps = bw_hz * math.log2(1 + snr_linear) / 1e6

        return LinkBudget(
            service=p.name,
            direction="downlink",
            altitude_km=h,
            elevation_deg=el,
            frequency_ghz=f,
            tx_power_dbw=round(tx_power_dbw, 2),
            tx_gain_dbi=round(p.sat_tx_gain_dbi, 2),
            eirp_dbw=round(eirp, 2),
            fspl_db=round(fspl, 2),
            atmospheric_loss_db=round(atm, 2),
            total_path_loss_db=round(total_loss, 2),
            slant_range_km=round(sr_km, 2),
            rx_gain_dbi=round(rx_g, 2),
            rx_noise_temp_k=p.ut_noise_temp_k,
            g_over_t_db=round(g_t, 2),
            cn0_dbhz=round(cn0, 2),
            cn_db=round(cn, 2),
            noise_bandwidth_hz=bw_hz,
            eb_n0_db=round(eb_n0, 2),
            bitrate_bps=br,
            required_eb_n0_db=p.required_eb_n0_db,
            link_margin_db=round(margin, 2),
            shannon_capacity_mbps=round(shannon_mbps, 2),
        )

    def compute_uplink(self) -> LinkBudget:
        """Compute uplink (user terminal → satellite) budget."""
        p = self.profile
        el = self.elevation_deg
        f = p.frequency_ghz * 0.75  # Uplink typically lower freq (Rx/Tx ratio ~0.75 Ku)
        h = p.altitude_km

        ut = Antenna(
            diameter_m=p.ut_diameter_m,
            efficiency=p.ut_efficiency,
            noise_temp_K=p.ut_noise_temp_k,
            frequency_ghz=f,
        )

        tx_power_dbw = 10 * math.log10(max(p.ut_tx_power_w, 1e-9))
        eirp = tx_power_dbw + ut.gain_dbi

        sr_km = slant_range_from_elevation(h, el)
        fspl = free_space_path_loss_db(sr_km, f)
        atm = atmospheric_loss_db(f, el)
        total_loss = fspl + atm

        rx_g = p.sat_rx_gain_dbi
        t_sys = p.sat_noise_temp_k
        g_t = rx_g - 10 * math.log10(t_sys)
        cn0 = eirp - total_loss + g_t - KB_DBW_HZ_K

        bw_hz = p.noise_bandwidth_hz
        cn = cn0 - 10 * math.log10(bw_hz)

        br = p.bitrate_bps
        eb_n0 = cn0 - 10 * math.log10(br)
        margin = eb_n0 - p.required_eb_n0_db

        snr_linear = 10 ** (cn / 10.0)
        shannon_mbps = bw_hz * math.log2(1 + snr_linear) / 1e6

        return LinkBudget(
            service=p.name,
            direction="uplink",
            altitude_km=h,
            elevation_deg=el,
            frequency_ghz=f,
            tx_power_dbw=round(tx_power_dbw, 2),
            tx_gain_dbi=round(ut.gain_dbi, 2),
            eirp_dbw=round(eirp, 2),
            fspl_db=round(fspl, 2),
            atmospheric_loss_db=round(atm, 2),
            total_path_loss_db=round(total_loss, 2),
            slant_range_km=round(sr_km, 2),
            rx_gain_dbi=round(rx_g, 2),
            rx_noise_temp_k=p.sat_noise_temp_k,
            g_over_t_db=round(g_t, 2),
            cn0_dbhz=round(cn0, 2),
            cn_db=round(cn, 2),
            noise_bandwidth_hz=bw_hz,
            eb_n0_db=round(eb_n0, 2),
            bitrate_bps=br,
            required_eb_n0_db=p.required_eb_n0_db,
            link_margin_db=round(margin, 2),
            shannon_capacity_mbps=round(shannon_mbps, 2),
        )

    # ------------------------------------------------------------------
    # Traffic model
    # ------------------------------------------------------------------

    def traffic_summary(self) -> dict:
        """Return full traffic analysis dictionary."""
        p = self.profile
        dl = self.compute_downlink()
        ul = self.compute_uplink()

        # Elevation sweep for margin plot data
        elevations = list(range(15, 91, 5))
        margin_curve = []
        for el in elevations:
            tm = TrafficModel(p, el)
            bud = tm.compute_downlink()
            margin_curve.append({
                "elevation_deg": el,
                "cn_db": bud.cn_db,
                "eb_n0_db": bud.eb_n0_db,
                "link_margin_db": bud.link_margin_db,
                "slant_range_km": bud.slant_range_km,
            })

        # Dwell time per station
        stations = {
            "Hanoi": 21.0, "Danang": 16.0, "HCMC": 10.8,
        }
        dwell = {}
        for st, lat in stations.items():
            t = dwell_time_s(p.altitude_km, p.min_elevation_deg, lat)
            vol = data_volume_per_pass_gb(p.altitude_km, p.bitrate_bps,
                                           p.min_elevation_deg, lat)
            dwell[st] = {
                "dwell_time_s": round(t, 1),
                "dwell_time_min": round(t / 60, 2),
                "data_per_pass_gb": round(vol, 3),
            }

        return {
            "service": p.name,
            "altitude_km": p.altitude_km,
            "target_bitrate_mbps": p.bitrate_bps / 1e6,
            "modcod": p.modcod,
            "required_eb_n0_db": p.required_eb_n0_db,
            "availability_target_pct": p.availability_target_pct,
            "latency_budget_ms": p.latency_budget_ms,
            "downlink": asdict(dl),
            "uplink": asdict(ul),
            "margin_vs_elevation": margin_curve,
            "dwell_time_by_station": dwell,
            "one_way_delay_ms": round(
                slant_range_from_elevation(p.altitude_km, self.elevation_deg) / C_KM_S * 1000, 2),
        }

    def print_budget(self, budget: LinkBudget) -> None:
        """Pretty-print a link budget table."""
        SEP = "─" * 60
        print(f"\n  {budget.service}  —  {budget.direction.upper()} LINK BUDGET")
        print(f"  Frequency: {budget.frequency_ghz:.1f} GHz  |  "
              f"Elevation: {budget.elevation_deg}°  |  "
              f"Altitude: {budget.altitude_km} km")
        print(SEP)
        rows = [
            ("TX Power", f"{budget.tx_power_dbw:.2f} dBW"),
            ("TX Antenna Gain", f"{budget.tx_gain_dbi:.2f} dBi"),
            ("EIRP", f"{budget.eirp_dbw:.2f} dBW"),
            ("─── Path ───", ""),
            ("Slant Range", f"{budget.slant_range_km:.1f} km"),
            ("FSPL", f"{budget.fspl_db:.2f} dB"),
            ("Atmospheric Loss", f"{budget.atmospheric_loss_db:.2f} dB"),
            ("Total Path Loss", f"{budget.total_path_loss_db:.2f} dB"),
            ("─── RX ───", ""),
            ("RX Antenna Gain", f"{budget.rx_gain_dbi:.2f} dBi"),
            ("System Noise Temp", f"{budget.rx_noise_temp_k:.0f} K"),
            ("G/T", f"{budget.g_over_t_db:.2f} dB/K"),
            ("─── Performance ───", ""),
            ("C/N₀", f"{budget.cn0_dbhz:.2f} dBHz"),
            ("Noise BW", f"{budget.noise_bandwidth_hz/1e6:.2f} MHz"),
            ("C/N", f"{budget.cn_db:.2f} dB"),
            ("Eb/N₀", f"{budget.eb_n0_db:.2f} dB"),
            ("Required Eb/N₀", f"{budget.required_eb_n0_db:.2f} dB"),
            ("Link Margin", f"{budget.link_margin_db:.2f} dB"),
            ("Shannon Capacity", f"{budget.shannon_capacity_mbps:.2f} Mbps"),
        ]
        for label, value in rows:
            if value == "":
                print(f"  {label}")
            else:
                marker = "✅" if label == "Link Margin" and budget.link_margin_db >= 3.0 else (
                    "❌" if label == "Link Margin" and budget.link_margin_db < 0 else " ")
                print(f"  {label:<26} {value:>12} {marker}")
        print(SEP)

    def export_json(self, path: str | Path = "traffic_report.json") -> Path:
        """Export traffic summary to JSON for core_network consumption."""
        payload = {
            "generated_at": datetime.now(timezone.utc).isoformat(),
            **self.traffic_summary(),
        }
        out = Path(path)
        out.parent.mkdir(parents=True, exist_ok=True)
        out.write_text(json.dumps(payload, indent=2))
        print(f"[export_json] Written → {out.resolve()}")
        return out


# ---------------------------------------------------------------------------
# CLI entry point
# ---------------------------------------------------------------------------

def main() -> None:
    """Run link budget analysis for both VNU-LEO service profiles."""
    import sys
    sys.stdout.reconfigure(encoding="utf-8")
    print("=" * 72)
    print("  VNU-LEO Traffic Model & Link Budget  —  Phase 1")
    print("=" * 72)

    for profile, label in [
        (INTERNET_VOIP_PROFILE, "internet"),
        (DATA_WEATHER_PROFILE, "weather"),
    ]:
        print(f"\n{'='*72}")
        print(f"  SERVICE: {profile.name}  (h={profile.altitude_km} km)")
        print("=" * 72)

        # Nominal elevation = 30° (typical operational angle)
        tm = TrafficModel(profile, elevation_deg=30.0)
        dl = tm.compute_downlink()
        ul = tm.compute_uplink()

        tm.print_budget(dl)
        tm.print_budget(ul)

        # Dwell time summary
        print(f"\n  Dwell time & data per pass (h={profile.altitude_km}km):")
        for st, lat in [("Hanoi", 21.0), ("Danang", 16.0), ("HCMC", 10.8)]:
            t = dwell_time_s(profile.altitude_km, profile.min_elevation_deg, lat)
            vol = data_volume_per_pass_gb(profile.altitude_km, profile.bitrate_bps,
                                           profile.min_elevation_deg, lat)
            print(f"    {st:<10}: {t/60:.1f} min  |  {vol:.2f} GB/pass")

        # Export
        out_dir = Path("outputs") / label
        tm.export_json(out_dir / "traffic_report.json")

    print("\n✅ Traffic model complete.  See outputs/ for JSON reports.\n")


if __name__ == "__main__":
    main()
