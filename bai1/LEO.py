import math

#  Lập trình tính toán và đề xuất số lượng vệ tinh LEO cần thiết, 
# cách sắp xếp quỹ đạo để đảm bảo phủ sóng liên tục toàn bộ Việt Nam
#
def calculate_leo_constellation(h, min_elevation=30):
    """
    Tính toán số lượng vệ tinh LEO cần thiết cho 1 điểm hoặc khu vực
    h: Độ cao vệ tinh (km)
    min_elevation: Góc ngẩng tối thiểu (độ) - thường > 30 để có tín hiệu tốt
    """
    R_earth = 6371  # Bán kính trái đất (km)
    
    # 1. Tính góc nhìn tối đa (half-angle)
    # Cos(eta) = R_earth * cos(min_elevation) / (R_earth + h)
    cos_eta = (R_earth * math.cos(math.radians(min_elevation))) / (R_earth + h)
    eta = math.acos(cos_eta)
    
    # 2. Tính góc trung tâm (central angle - gamma)
    gamma = 90 - min_elevation - math.degrees(eta)
    
    # 3. Diện tích phủ sóng của 1 vệ tinh (km^2)
    # Area = 2 * pi * R_earth^2 * (1 - cos(gamma))
    area_coverage = 2 * math.pi * (R_earth**2) * (1 - math.cos(math.radians(gamma)))
    
    # 4. Số lượng vệ tinh tối thiểu để phủ toàn cầu (công thức xấp xỉ)
    # N = Area_earth / Area_coverage
    total_earth_area = 4 * math.pi * (R_earth**2)
    n_min = total_earth_area / area_coverage
    
    print(f"Độ cao: {h} km")
    print(f"Bán kính phủ sóng 1 vệ tinh: {gamma:.2f} độ")
    print(f"Diện tích phủ sóng 1 vệ tinh: {area_coverage:.2f} km^2")
    print(f"Số lượng vệ tinh tối thiểu (toàn cầu, 1 lớp): {math.ceil(n_min)}")
    
    # Đối với khu vực cụ thể như VN (bán cầu bắc), cần mật độ cao hơn.
    # Ước tính cho khu vực < 30 độ vĩ bắc:
    return math.ceil(n_min)

# Chạy thử
calculate_leo_constellation(h=550, min_elevation=30)
