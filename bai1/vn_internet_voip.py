import math

# cái này cũng giải bài 1, nhưng với chỉ số riêng của VN trả lời câu hỏi Internet/VOIP, truyền tải dữ liệu ảnh radar

def calculate_leo_constellation_vietnam(h, min_elevation, use_case_name):
    """
    Tính toán số lượng vệ tinh LEO và đề xuất quỹ đạo cho không phận Việt Nam.
    h: Độ cao vệ tinh (km)
    min_elevation: Góc ngẩng tối thiểu (độ)
    """
    R_earth = 6371  # Bán kính trái đất (km)
    
    # 1. Tính toán vùng phủ sóng (Footprint)
    cos_eta = (R_earth * math.cos(math.radians(min_elevation))) / (R_earth + h)
    eta = math.acos(cos_eta)
    gamma = 90 - min_elevation - math.degrees(eta)
    
    area_coverage = 2 * math.pi * (R_earth**2) * (1 - math.cos(math.radians(gamma)))
    total_earth_area = 4 * math.pi * (R_earth**2)
    n_global_min = total_earth_area / area_coverage
    
    # 2. Hệ số bù đắp cho dải vĩ độ Việt Nam (Từ 8°N đến 24°N)
    # Việt Nam nằm gần xích đạo. Không cần phủ sóng 2 cực, ta dùng quỹ đạo nghiêng (Inclined Orbit)
    # Hệ số tối ưu hóa (tương đối) cho quỹ đạo Walker Delta góc nghiêng 30-40 độ: ~0.4
    n_vietnam_optimized = math.ceil(n_global_min * 0.4)
    
    # 3. Tính độ trễ truyền dẫn một chiều (One-way Propagation Delay) - FSPL
    c = 299792.458 # Tốc độ ánh sáng (km/s)
    max_distance = math.sqrt(R_earth**2 + (R_earth+h)**2 - 2*R_earth*(R_earth+h)*math.cos(math.radians(gamma)))
    max_delay_ms = (max_distance / c) * 1000

    print(f"=== KỊCH BẢN: {use_case_name} ===")
    print(f"- Độ cao thiết kế: {h} km | Góc ngẩng: {min_elevation}°")
    print(f"- Bán kính vùng phủ sóng: {gamma:.2f}°")
    print(f"- Độ trễ sóng vô tuyến tối đa: {max_delay_ms:.2f} ms")
    print(f"- Đề xuất số lượng vệ tinh (Chùm Walker Delta): {n_vietnam_optimized} vệ tinh")
    print(f"- Cách sắp xếp quỹ đạo: Góc nghiêng (Inclination) i = 35°. Lý do: Bao trọn vĩ độ Việt Nam (24°N) mà không lãng phí vệ tinh ở hai cực.\n")
    
    return n_vietnam_optimized

# Kịch bản 1: Internet / VoIP (Yêu cầu độ trễ cực thấp)
# Chọn quỹ đạo thấp (500km) để ping nhỏ, nhưng bù lại vùng phủ nhỏ -> tốn nhiều vệ tinh
calculate_leo_constellation_vietnam(h=500, min_elevation=30, use_case_name="Internet/VoIP (Độ trễ thấp)")

# Kịch bản 2: Truyền tải Ảnh / Radar thời tiết (Yêu cầu băng thông cao)
# Chọn quỹ đạo cao hơn (1000km). Độ trễ cao hơn (không sao vì tải file không cần realtime), 
# vùng phủ rộng -> thời gian vệ tinh bay ngang trạm lâu hơn -> truyền được cục dữ liệu to hơn -> cần ít vệ tinh.
calculate_leo_constellation_vietnam(h=1000, min_elevation=30, use_case_name="Dữ liệu Ảnh/Radar (Băng thông cao)")