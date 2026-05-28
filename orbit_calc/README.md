# orbit_calc — Module Thiết Kế Quỹ Đạo VNU-LEO

**Ngôn ngữ**: Python 3.11+  
**Vai trò trong hệ thống**: Thiết kế chùm vệ tinh LEO, validate phủ sóng Việt Nam,
tính link budget; xuất TLE + metadata JSON cho `core_network`.

---

## 1. Tổng Quan

Module `orbit_calc` thực hiện ba nhiệm vụ:

1. **Thiết kế chùm vệ tinh** — Walker Delta constellation math (N, P, F, i, h),
   tối ưu tham số để đạt ≥99.9% phủ sóng liên tục trên dải 8°N–24°N (toàn Việt Nam).
2. **Validate phủ sóng** — Mô phỏng 24 giờ SGP4, kiểm tra coverage và max outage
   tại ba gateway thực tế (Hà Nội, Đà Nẵng, TP.HCM) với ngưỡng elevation riêng.
3. **Link budget** — FSPL + mô hình mưa ITU-R P.618, Shannon capacity, dwell time
   cho hai band (Ku/Internet-VoIP và Ka/Data-Weather).

**Output chính cho `core_network`**:

| File | Mô tả |
|------|-------|
| `outputs/unified/constellation.tle` | 3LE SGP4-compatible, 48 vệ tinh |
| `outputs/unified/constellation.json` | Orbital elements + coverage stats + service info |
| `outputs/unified/summary.txt` | Human-readable design summary |

---

## 2. Cấu Trúc File

```
orbit_calc/
├── coverage_calculator.py   # Walker Delta math, VietnamCoverageAnalyzer, TLE export
├── traffic_model.py         # Link budget (FSPL, ITU-R P.618 rain), Shannon capacity
├── generate_unified.py      # Driver: sinh TLE + validate 24h coverage
├── simulation.ipynb         # Jupyter: ground track, footprint, handover, revisit
├── test_sgp4.py             # Verify TLE parseable by sgp4 library
└── outputs/
    ├── unified/             # Production TLE dùng cho core_network
    │   ├── constellation.tle
    │   ├── constellation.json
    │   └── summary.txt
    └── simulation/          # Outputs từ notebook (PNG, JSON reports)
```

### Mô tả từng file

**`coverage_calculator.py`**  
Chứa: `WalkerDelta` (dataclass tham số), `walker_delta_positions()` (vị trí tức thời
từ Walker math), `elevation_angle_deg()`, `VietnamCoverageAnalyzer` (chạy sim 24h,
export TLE/JSON), `estimate_minimum_satellites()`.

**`traffic_model.py`**  
Chứa: `TrafficModel`, `INTERNET_VOIP_PROFILE`, `DATA_WEATHER_PROFILE`.
Tính FSPL, rain attenuation (ITU-R P.618), C/N ratio, link margin, Shannon capacity,
dwell time, slant range.

**`generate_unified.py`**  
Driver chính. Gọi `recommended_constellation("unified")`, xuất TLE 48 vệ tinh,
chạy validate 24h tại 3 gateway, ghi `outputs/unified/`.

**`simulation.ipynb`**  
8 sections: imports → constellation params → ground track → coverage footprint →
handover zones → revisit timeline → link budget vs elevation → full export.
Sử dụng constellation tối ưu 48/6/2.

**`test_sgp4.py`**  
Kiểm tra nhanh: load TLE bằng `sgp4` library, propagate sat[0] tại epoch,
verify `err=0` và `|r| ≈ 7569 km`.

---

## 3. Kết Quả Tối Ưu Hiện Tại

### Cấu hình cuối: Walker Delta 48/6/2 · i = 22° · h = 1200 km

| Tham số | Giá trị |
|---------|---------|
| Tổng vệ tinh (T) | 48 |
| Số mặt phẳng (P) | 6 |
| Phasing factor (F) | 2 |
| Inclination | 22° |
| Altitude | 1200 km |
| Sats per plane | 8 |
| Orbital period | ~108 min |

### So sánh trước / sau tối ưu

| Chỉ tiêu | Baseline cũ | Tối ưu mới | Thay đổi |
|----------|------------|------------|----------|
| Cấu hình Internet | 24/6/1 · i=53° · h=500km | 48/6/2 · i=22° · h=1200km | — |
| Cấu hình Weather | 18/6/1 · i=53° · h=1000km | (chung) | — |
| Tổng vệ tinh | 256+90 = 346 | **48** | −298 sat (−86%) |
| Coverage Hà Nội (ε=25°) | 71.25% | **100.00%** | +28.75 pp |
| Coverage Đà Nẵng (ε=20°) | 79.51% | **100.00%** | +20.49 pp |
| Coverage TP.HCM (ε=25°) | 71.25% | **100.00%** | +28.75 pp |
| Max outage | 2520 s (42 min) | **0 s** | −100% |
| Latency zenith | 1.67 ms | 4.00 ms | +2.3 ms (vẫn tốt) |
| Latency ε=25° | 3.8 ms | 8.10 ms | < 50ms VoIP ✓ |

### Validate 24h SGP4

```
Hanoi    (ε≥25.0°): coverage=100.00%  max_outage=0s
Danang   (ε≥20.0°): coverage=100.00%  max_outage=0s
HCMC     (ε≥25.0°): coverage=100.00%  max_outage=0s
```

---

## 4. Lý Do Chọn Cấu Hình

### Inclination 22° thay vì 53°

Việt Nam nằm ở dải vĩ độ 8°N–24°N. Vệ tinh có inclination bằng vĩ độ trung bình
của vùng mục tiêu sẽ có **dwell time tối đa** (thời gian bay qua).

| Inclination | Dwell time trên VN |
|-------------|-------------------|
| i = 53° | ~11% (dải VN nằm xa equatorial bulge) |
| i = 22° | ~38% (dải VN trùng với vùng quẹo quỹ đạo) |

### Altitude 1200 km thay vì 500/1000 km

Footprint radius tại ε=25°:

| Altitude | Footprint radius | Số sat cần thiết (VN, ε=25°) |
|----------|-----------------|------------------------------|
| 500 km   | 530 km  | ~256 |
| 1000 km  | 1150 km | ~90  |
| 1200 km  | 1714 km | **~48** |

Altitude 1200 km là điểm tối ưu: footprint đủ lớn để 48 sat phủ toàn VN, latency
vẫn dưới ngưỡng 50ms cho VoIP.

### Single constellation phục vụ 2 dịch vụ

Walker Delta 48/6/2 ở h=1200km đủ phủ sóng cho cả:
- **Internet/VoIP** (band Ku, min elevation 25°)
- **Data/Weather** (band Ka, min elevation 15°, yêu cầu thấp hơn)

Thay vì duy trì 2 constellation riêng (346 vệ tinh), dùng 1 constellation (48 vệ tinh).

---

## 5. Quick Start

### Setup môi trường

```bash
cd orbit_calc
python -m venv .venv
# Windows:
.venv\Scripts\activate
# Linux/macOS:
source .venv/bin/activate

pip install numpy scipy sgp4 matplotlib jupyter pandas nbconvert ipykernel
```

### Chạy các lệnh chính

```bash
# 1. Sinh TLE production + validate 24h coverage
python generate_unified.py

# 2. Chạy full coverage report (tất cả 6 gateway Vietnam)
python coverage_calculator.py

# 3. Link budget report (FSPL + rain attenuation + Shannon capacity)
python traffic_model.py

# 4. Chạy notebook và lưu kết quả (tạo simulation_executed.ipynb, gitignored)
jupyter nbconvert --to notebook --execute simulation.ipynb --output simulation_executed.ipynb --ExecutePreprocessor.timeout=300

# 5. Mở notebook để chỉnh sửa
jupyter lab simulation.ipynb
```

### Kết quả mong đợi sau `generate_unified.py`

```
Config: Walker Delta 48/6/2 i=22.0° h=1200.0km
  T = 48  (vs old 256+90 = 346 -> saves 298 sats, -86%)
  ...
[VALIDATION] Running 24h coverage simulation...
  Hanoi    (e>=25.0°): coverage=100.00%  max_outage=0s
  Danang   (e>=20.0°): coverage=100.00%  max_outage=0s
  HCMC     (e>=25.0°): coverage=100.00%  max_outage=0s
Done.
```

---

## 6. Interface với `core_network`

`core_network` đọc TLE khi khởi động qua `data/deploy_setting.json`:

```json
{
    "app_port": "8081",
    "gateway_data_path": "data/gateways.json",
    "settings_path": "data/system_settings.json",
    "tle_path": "../orbit_calc/outputs/unified/constellation.tle"
}
```

**Workflow tích hợp**:

```
orbit_calc/                           core_network/
  generate_unified.py                   main.go
       |                                    |
       v                                    v
  outputs/unified/constellation.tle  ---> LoadTLE()
  outputs/unified/constellation.json      NewSatelliteTracker()
                                          satTracker.Update(time.Now())
```

**Format TLE** (3-line element set, SGP4-compatible):

```
VNU-LEO-0101
1 99001U 26001A   26147.00000000  .00000000  00000-0  00000-0 0  9990
2 99001  22.0000   0.0000 0000001   0.0000   0.0000 13.37XXXXX    0
```

**Format `constellation.json`**:

```json
{
  "schema_version": "2.0-unified",
  "design": {
    "type": "Walker Delta (unified)",
    "notation": "i=22.0° : 48/6/2",
    "N": 48, "P": 6, "F": 2,
    "inclination_deg": 22.0,
    "altitude_km": 1200.0
  },
  "services": {
    "internet_voip": {"band": "Ku (12/14 GHz)", "min_elev_service_deg": 25.0},
    "data_weather":  {"band": "Ka (20/30 GHz)", "min_elev_service_deg": 15.0}
  }
}
```

**Khi nào cần regenerate TLE**: sau khi thay đổi tham số trong
`recommended_constellation()`, hoặc muốn cập nhật TLE epoch. `core_network` load
TLE lúc startup; để reload, restart `go run .`.

---

## 7. Validation

### Grid phủ sóng toàn Việt Nam

- **9×5 = 45 điểm** trải đều trên dải 8°N–24°N / 102°E–110°E
- Ngưỡng elevation: ε=15° (any-VN coverage)
- Timestep: 60s, duration: 24h

### 3 Gateway thực tế

| Gateway | Vĩ độ | Kinh độ | Min elevation (ε) |
|---------|-------|---------|------------------|
| Hà Nội  | 21.0285°N | 105.8542°E | 25° |
| Đà Nẵng | 16.0471°N | 108.2062°E | 20° |
| TP.HCM  | 10.7626°N | 106.6601°E | 25° |

### Verify TLE bằng SGP4

```python
# test_sgp4.py
from sgp4.api import Satrec
import numpy as np

with open('outputs/unified/constellation.tle') as f:
    lines = [l.strip() for l in f if l.strip()]

sats = [Satrec.twoline2rv(lines[i+1], lines[i+2])
        for i in range(0, len(lines), 3)]

err, r, v = sats[0].sgp4(sats[0].jdsatepoch, sats[0].jdsatepochF)
assert err == 0
assert abs(np.linalg.norm(r) - 7569) < 20  # 6378 + 1200 km, ±20km SGP4
```

Chạy: `python test_sgp4.py` → `Loaded 48 sats`, `err=0`, `|r|=7568.9 km`.

---

## 8. Troubleshooting

### `UnicodeEncodeError` trên Windows

Các file Python trong module đã thêm `sys.stdout.reconfigure(encoding="utf-8")`
ở đầu. Nếu gặp lỗi này ở file khác:

```python
import sys
sys.stdout.reconfigure(encoding="utf-8")
```

### TLE không parse được

```bash
pip install "sgp4>=2.20"
# Kiểm tra số dòng:
# Windows PowerShell:
(Get-Content outputs/unified/constellation.tle | Measure-Object -Line).Lines
# Expect: 144 (48 sats * 3 lines)
```

### Coverage < 100% sau khi regenerate TLE

TLE epoch mặc định là `datetime.now(UTC)` tại thời điểm chạy. Nếu validate lại sau
nhiều ngày, epoch trong TLE sẽ khác — đây là bình thường. Chạy lại
`python generate_unified.py` để cập nhật epoch.

### `ImportError: cannot import name 'X' from coverage_calculator`

Kiểm tra phiên bản file — một số hàm (`max_elevation_from_any_satellite`,
`footprint_radius_km`) cần `coverage_calculator.py` phiên bản mới nhất:

```bash
git log --oneline orbit_calc/coverage_calculator.py | head -5
```

---

## References

- **Walker, J.G. (1984)**. "Satellite Constellations." *Journal of the British
  Interplanetary Society*, 37, 559–572. — Công thức Walker Delta constellation,
  phân bổ RAAN và phasing factor.

- **ITU-R P.618-13 (2017)**. "Propagation data and prediction methods required for
  the design of Earth-space telecommunication systems." — Mô hình rain attenuation
  dùng trong `traffic_model.py`.

- **Vallado, D.A. (2013)**. *Fundamentals of Astrodynamics and Applications*, 4th ed.
  Microcosm Press. — SGP4 propagation model, orbital mechanics reference.

- **sgp4 Python library** (Brandon Rhodes, v2.22+):
  SGP4/SDP4 propagator dùng trong `test_sgp4.py` và `coverage_calculator.py`.
