import numpy as np
import matplotlib.pyplot as plt

# bài 2 handover

# 1. Khai báo Hằng số Trái Đất và Vệ tinh
R_e = 6371.0 # Bán kính Trái Đất (km)
h = 550.0    # Độ cao vệ tinh LEO (km)
r = R_e + h  # Khoảng cách từ tâm Trái Đất đến vệ tinh

# Tọa độ 3 Gateway chính (Vĩ độ, Kinh độ)
gateways = {
    'Ha_Noi': (21.02, 105.83),
    'Da_Nang': (16.05, 108.22),
    'TP_HCM': (10.76, 106.69)
}

# 2. Hàm tính Góc Ngẩng (Elevation Angle) theo chuẩn tài liệu Viễn thông
def calculate_elevation(lat_gs, lon_gs, lat_sat, lon_sat):
    # Đổi độ sang Radian để tính lượng giác
    lat_gs, lon_gs = np.radians(lat_gs), np.radians(lon_gs)
    lat_sat, lon_sat = np.radians(lat_sat), np.radians(lon_sat)
    
    # Tính hiệu kinh tuyến
    delta_lon = np.abs(lon_gs - lon_sat)
    
    # Tính góc ở tâm Trái Đất (gamma)
    cos_gamma = np.cos(delta_lon) * np.cos(lat_gs) * np.cos(lat_sat) + np.sin(lat_gs) * np.sin(lat_sat)
    gamma = np.arccos(cos_gamma)
    
    # Tính khoảng cách 3D (Slant Range - d)
    d = np.sqrt(R_e**2 + r**2 - 2 * R_e * r * cos_gamma)
    
    # Tính góc ngẩng (Elevation - El)
    cos_el = (r * np.sin(gamma)) / d
    elevation_rad = np.arccos(cos_el)
    
    return np.degrees(elevation_rad) # Trả về góc bằng Độ (Degrees)

# 3. Tạo đường bay giả lập cho vệ tinh LEO (Bay dọc Việt Nam)
num_steps = 100
sat_lat = np.linspace(25.0, 5.0, num_steps) # Bay từ Vĩ độ 25 (qua TQ) xuống Vĩ độ 5 (qua biển)
sat_lon = np.ones(num_steps) * 106.0        # Bay dọc theo kinh tuyến 106

# 4. THUẬT TOÁN HANDOVER
current_gs = None
MIN_ELEVATION = 15.0 # Ngưỡng cắt mạng: Góc ngẩng dưới 15 độ là sóng quá yếu, rớt mạng

print("--- BẮT ĐẦU MÔ PHỎNG HANDOVER ---")
for i in range(num_steps):
    elevations = {}
    
    # Quét góc ngẩng từ vệ tinh đến cả 3 trạm
    for name, pos in gateways.items():
        el = calculate_elevation(pos[0], pos[1], sat_lat[i], sat_lon[i])
        elevations[name] = el
        
    # Tìm trạm có góc ngẩng CAO NHẤT (Sóng khỏe nhất)
    best_gs = max(elevations, key=elevations.get)
    max_el = elevations[best_gs]
    
    # Logic Chuyển giao
    if max_el >= MIN_ELEVATION:
        if current_gs != best_gs:
            print(f"Bước {i:02d} | Tọa độ vệ tinh: {sat_lat[i]:.2f}°N | Chuyển giao (Handover) tới Trạm: {best_gs} (Góc ngẩng: {max_el:.1f}°)")
            current_gs = best_gs
    else:
        if current_gs is not None:
            print(f"Bước {i:02d} | Tọa độ vệ tinh: {sat_lat[i]:.2f}°N | Rớt mạng! (Không trạm nào có góc ngẩng > 15°)")
            current_gs = None

print("--- KẾT THÚC MÔ PHỎNG ---")