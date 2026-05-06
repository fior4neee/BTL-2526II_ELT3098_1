# TÊN DỰ ÁN: Thiết kế và Mô phỏng Hệ thống Vệ tinh VNU-LEO  

## Giả định tình huống:  

ĐHQGHN (VNU) đang trong giai đoạn nghiên cứu thiết kế hệ thống LEO thử nghiệm. Hệ thống gồm: Trạm mặt đất (Gateway), Router người dùng cuối (như Starlink), và chùm vệ tinh quỹ đạo thấp (LEO).  

## Yêu cầu  

1. **Bài toán Quỹ đạo & Phủ sóng**: Lập trình tính toán và đề xuất số lượng vệ tinh LEO cần thiết, cách sắp xếp quỹ đạo để đảm bảo phủ sóng liên tục toàn bộ Việt Nam. Phục vụ 2 ngành: Internet/VoIP (độ trễ thấp) và truyền tải dữ liệu Ảnh/Radar thời tiết (băng thông cao).    

2. **Bài toán Gateway (Core Network)**: Lập trình mô phỏng quá trình duy trì phiên kết nối và chuyển giao (Handover) khi vệ tinh di chuyển qua các Gateway đặt tại Hà Nội, Đà Nẵng, TP. Hồ Chí Minh.   

3. **Bài toán Client (End-user Router)**: Xây dựng phần mềm trực quan hóa mô phỏng hoạt động của Router người dùng. Phân mềm phải trực quan hóa được quá trình ăng-ten mảng pha bám bắt vệ tinh, cập nhật các thông số C/N, suy hao, và chất lượng tín hiệu theo thời gian thực.  

## Công cụ & công nghệ khuyến nghị  

Thầy đặc biệt khuyến khích các em tiếp cận và ứng dụng các framework/công cụ mới nhất trên thị trường để quản lý và phát triển dự án. Cụ thể:  

- **Quản lý tiến độ & Phân chia công việc**: Sử dụng [Clickup.com](https://clickup.com/). Các nhóm cần mô phỏng quy trình làm việc Agile/Scrum, chia task rõ ràng cho từng thành viên (AI làm Front-end, ai làm Backend/Network, ai làm RF/Toán).  

- **Báo cáo & Thuyết trình**: Khuyến khích dùng [Gamma.app](https://gamma.app/) để tạo slide thuyết trình nhanh, chuyên nghiệp và có tính thẩm mỹ cao, thay vì các công cụ truyền thống.  

- **Giao diện Web Admin (Quản lý hạ tầng/Gateway)**: Các em có thể tận dụng Svelte / SvelteKit. Thầy gợi tham khảo các thư viện UI/mã nguồn mở tại [madewithsvelte.com](https://madewithsvelte.com/) để xây dựng dashboard quản lý mạng nhanh chóng.  

- **Ứng dụng Desktop (App người dùng cuối)**: Ưu tiên sử dụng [Tauri v2](https://v2.tauri.app/) để phát triển ứng dụng quản lý lưu lượng, tính toán chi phí và theo dõi trạng thái bám bắt vệ tinh. Giao diện đẹp, hiệu năng cực cao và tiêu tốn ít RAM.  

## Nhiệm vụ nâng cao  

Dành cho các nhóm muốn thử sức với bài toán thực tế chuẩn công nghiệp (Sẽ được cộng điểm thưởng cuối kỳ và ưu tiên xét tuyển vào Lab Antaja / dự án VNU).  

Thay vì chỉ dừng lại ở tính toán vật lý, các em hãy mở rộng hệ thống phần mềm mô phỏng của mình bằng cách đóng vai là Nhà cung cấp dịch vụ mạng vệ tinh (ISP). Các tính năng mở rộng (chọn 1 hoặc nhiều):  

1. **Quản lý Định danh & Chống giả mạo thiết bị (Device Verification)**: Xây dựng cơ chế cấp phép (Provisioning) cho Router người dùng cuối. Làm sao để Gateway nhận diện đúng thiết bị chính hãng đã đăng ký và từ chối các phần cứng giả mạo (MAC/Hardware ID Spoofing) cố tình truy cập vào mạng LEO?  

2. **Xác thực Không - Thời gian & Quản lý gói cước (Spatio-temporal Billing)**: Mô phỏng hệ thống phân quyền gói cước: Gói Cố định (Fixed - chỉ được kết nối tại một tọa độ nhà đăng ký) và Gói Di động (Mobility - dùng trên tàu thuyền, xe RV).  
    - Viết logic giám sát: Dựa vào tọa độ thực thời của Router và quỹ đạo vệ tinh LEO, hệ thống (Svelte Web Admin) phải tự động phát hiện và ngắt kết nối bóp băng thông nếu người dùng "Gói Cố định" di chuyển thiết bị ra khỏi vùng địa lý (Geo-fencing) cho phép.  

3. **Dashboard Giám sát Hạ tầng Tổng thể (Network Monitoring System)**: Tích hợp dữ liệu từ Gateway và các Router để hiển thị trên một Web Admin tập trung (SvelteKit/Vue...). Giao diện cho phép quản trị viên xem trạng thái sống/chết (Alive/Dead) của các trạm, lưu lượng đang tiêu thụ, và lịch sử chuyển giao (handover) giữa các trạm.  

*Gợi ý: Các em làm mạnh về Web/App có thể tập trung vào phần Optional này để gánh điểm cho phần tính toán công suất vô tuyến thuần túy.*

**Lưu ý**: Thầy không hạn chế sự sáng tạo của các em. Các em hoàn toàn có thể tự do đề xuất và sử dụng bất kỳ công cụ, ngôn ngữ, hoặc framework mới nào (Python, Go, Rust, React, v.v.) miễn là giải quyết được bài toán một cách tối ưu nhất.

## Đầu ra dự kiến  

- Slide thuyết trình giải pháp hệ thống, sơ đồ thiết kế kiến trúc.  

- Demo mã nguồn mô phỏng chạy thực tế.