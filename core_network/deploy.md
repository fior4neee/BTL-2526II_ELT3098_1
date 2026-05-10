---

# 🛰️ VNU-LEO Core Network - Deployment Guide

Tài liệu này hướng dẫn chi tiết các bước để cài đặt và vận hành module **Core Network** của dự án VNU-LEO trên máy tính cục bộ.

## 📋 Yêu cầu hệ thống (Prerequisites)

Dưới đây là các bước chuẩn bị nếu máy tính của bạn hoàn toàn mới:

### 1. Cài đặt ngôn ngữ lập trình Go

Bạn cần cài đặt Go (phiên bản 1.18 trở lên) để có thể biên dịch và chạy mã nguồn.

* **Link tải:** [https://go.dev/dl/](https://go.dev/dl/)
* Chọn phiên bản phù hợp với hệ điều hành của bạn (Windows, macOS, hoặc Linux).
* Sau khi cài đặt xong, hãy mở Terminal (hoặc CMD) và gõ lệnh sau để kiểm tra:
```bash
go version

```



---

## 🛠️ Các bước thiết lập dự án (Setup Steps)

Giả sử bạn đã tải mã nguồn về và đang ở trong thư mục dự án.

### Bước 1: Khởi tạo Go Module

Go Module là trình quản lý thư viện của Go. Bạn cần khởi tạo nó trước khi cài đặt bất kỳ thư viện nào.
Mở Terminal tại thư mục chứa các file (`main.go`, `models.go`, ...):

```bash
go mod init vnu_leo_core

```

### Bước 2: Cài đặt thư viện Gin Web Framework

Hệ thống sử dụng Framework **Gin** để xây dựng REST API. Hãy tải nó về bằng lệnh:

```bash
go get -u github.com/gin-gonic/gin

```

*Lưu ý: Lệnh này sẽ tự động tạo file `go.sum` và cập nhật file `go.mod` của bạn.*

### Bước 3: Chuẩn bị cấu trúc thư mục và dữ liệu

Đảm bảo thư mục dự án của bạn có cấu trúc như sau:

```text
/your-project-folder/
├── main.go
├── models.go
├── gateway_manager.go
├── handover_manager.go
├── go.mod
├── go.sum
└── data/
    └── gateways.json  <-- Đảm bảo có file này

```

Nếu bạn chưa có file `data/gateways.json`, hãy tạo thư mục `data` và tạo file với nội dung mẫu:

```json
[
  {
    "id": "GW-HAN-01",
    "name": "Hanoi Gateway",
    "location": { "lat": 21.0285, "lon": 105.8542, "alt": 0.015 },
    "max_sessions": 50000,
    "min_elevation_deg": 25.0,
    "antenna": { "gain_dbi": 42.5, "beam_width_deg": 1.5 },
    "status": "alive"
  }
]

```

---

## 🚀 Khởi chạy hệ thống (Running)

Tại thư mục gốc của dự án, chạy lệnh sau:

```bash
go run .

```

Nếu thành công, hệ thống sẽ in ra các dòng log tiếng Anh như sau:

```text
--- VNU-LEO Core Network Booting ---
Successfully loaded 1 gateways from local storage.
Handover Manager is online.
VNU-LEO Core API listening on :8080

```

---

## 🧪 Kiểm tra trạng thái (Testing)

Sau khi Server đã chạy (đang lắng nghe ở port 8080), bạn có thể dùng trình duyệt web hoặc Postman để kiểm tra:

1. **Kiểm tra sức khỏe (Health Check):**
* **URL:** `http://localhost:8080/api/v1/health`
* **Kết quả mong muốn:** `{"status":"ok"}`


2. **Lấy danh sách trạm (Gateways):**
* **URL:** `http://localhost:8080/api/v1/gateways`
* **Mục đích:** Xác nhận dữ liệu từ file JSON đã được nạp thành công.


3. **Kết nối thử một Router (POST Request):**
* Sử dụng Postman gửi một request POST đến `http://localhost:8080/api/v1/router/connect`
* **Body (JSON):**
```json
{
  "router_mac": "AA:BB:CC:DD:EE:FF",
  "lat": 21.0,
  "lon": 105.8
}

```





## 🐞 Gỡ lỗi (Troubleshooting)

* **Lỗi `StateHold is not a type`:** Đảm bảo bạn đã sửa file `models.go` theo hướng dẫn (đổi sang `SessionState`).
* **Lỗi `cannot find module...`:** Chạy lại lệnh `go mod tidy` để Go tự động cập nhật và tải các thư viện còn thiếu.
* **Lỗi `gateways.json: no such file or directory`:** Kiểm tra lại xem bạn đã tạo thư mục `data` và file `gateways.json` đúng vị trí chưa.

---

*VNU-LEO Project - 2026*