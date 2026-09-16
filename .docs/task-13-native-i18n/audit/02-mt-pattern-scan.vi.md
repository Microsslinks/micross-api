# 02 · 机翻特征扫描 · VI（task-13 phase 2）

> 创建：2026-09-17
> 数据来源：`web/src/i18n/locales/vi.json` vs `en.json`
> 工具：`web/scripts/scan-mt-patterns.mjs`

## 摘要

| 指标 | 值 |
|---|---|
| 总键数 | 5927 |
| 完全未翻译（与 en 相同）| 633 |
| 至少命中一条机翻特征 | 295 |
| 句式分叉数 | 0 |

## 规则命中统计

| 规则 | 命中数 | 说明 |
|---|---|---|
| `borrowed-english` | 251 | value 保留了英文技术借词（upstream / cache / token 等） |
| `length-too-long` | 44 | value 长度 > en 原文 1.5×（典型：过度展开） |
| `length-too-short` | 1 | value 长度 < en 原文 0.3×（典型：翻译过度精简） |

## 详细命中清单（每规则前 30）

### borrowed-english（251 处，展示前 30）

| key | en | vi |
|---|---|---|
| `upstream token factories` | upstream token factories | nhà máy token thượng nguồn |
| `2. Copy the application token` | 2. Copy the application token | 2. Sao chép token ứng dụng |
| `Access Policy (JSON)` | Access Policy (JSON) | Chính sách truy cập (JSON) |
| `Access Token` | Access Token | Token truy cập |
| `Add API` | Add API | Thêm API |
| `Add API Shortcut` | Add API Shortcut | Thêm lối tắt API |
| `Add OAuth Provider` | Add OAuth Provider | Thêm nhà cung cấp OAuth |
| `Admin access required` | Admin access required | Yêu cầu quyền truy cập Admin |
| `All API tokens` | All API tokens | Tất cả khóa API |
| `Allow upstream callbacks` | Allow upstream callbacks | Cho phép callback upstream |
| `API Access` | API Access | Truy cập API |
| `API Addresses` | API Addresses | Địa chỉ API |
| `API Base URL *` | API Base URL * | URL cơ sở API * |
| `API Endpoints` | API Endpoints | Điểm cuối API |
| `API Info` | API Info | Thông tin API |
| `API key` | API key | Khóa API |
| `API Key` | API Key | Khóa API |
| `API Key (Production)` | API Key (Production) | API Key (Sản xuất) |
| `API Key (Sandbox)` | API Key (Sandbox) | Khóa API (Sandbox) |
| `API Key *` | API Key * | Khóa API * |
| `API key is required` | API key is required | Khóa API là bắt buộc |
| `API Keys` | API Keys | Khóa API |
| `API Private Key` | API Private Key | Khóa riêng API |
| `API Requests` | API Requests | Yêu cầu API |
| `API secret` | API secret | Bí mật API |
| `API token management` | API token management | Quản lý token API |
| `API usage records` | API usage records | Lịch sử sử dụng API |
| `Apply All Upstream Updates` | Apply All Upstream Updates | Áp dụng Tất cả Cập nhật Upstream |
| `Audio Tokens` | Audio Tokens | Token âm thanh |
| `Billable input tokens` | Billable input tokens | Token đầu vào tính phí |
| ... | （剩余 221 处略）| |

### length-too-long（44 处，展示前 30）

| key | en | vi |
|---|---|---|
| `Contact Us` | Contact Us | Liên hệ với chúng tôi |
| `Welcome to our site...` | Welcome to our site... | Chào mừng đến với trang web của chúng tôi... |
| `#1 by usage` | #1 by usage | Hạng 1 theo mức sử dụng |
| `Active apps` | Active apps | Ứng dụng đang hoạt động |
| `Admin Only` | Admin Only | Chỉ dành cho quản trị viên |
| `Amount Due` | Amount Due | Số tiền cần thanh toán |
| `apps tracked` | apps tracked | ứng dụng được theo dõi |
| `Bad Request` | Bad Request | Yêu cầu không hợp lệ |
| `Batch Edit` | Batch Edit | Chỉnh sửa hàng loạt |
| `Chat Client Name` | Chat Client Name | Tên ứng dụng khách trò chuyện |
| `Chat preset not found` | Chat preset not found | Thiết lập sẵn trò chuyện không tìm thấy |
| `Chat Presets` | Chat Presets | Cài đặt sẵn trò chuyện |
| `Common User` | Common User | Người dùng thông thường |
| `Console area` | Console area | Khu vực bảng điều khiển |
| `Console Area` | Console Area | Khu vực bảng điều khiển |
| `Copy token` | Copy token | Sao chép mã thông báo |
| `Discount ratio for cache hits.` | Discount ratio for cache hits. | Tỷ lệ chiết khấu cho lượt truy cập bộ nhớ đệm thành công. |
| `Edit chat preset` | Edit chat preset | Chỉnh sửa cài đặt trước trò chuyện |
| `Edit Vendor` | Edit Vendor | Chỉnh sửa Nhà cung cấp |
| `Finish Time` | Finish Time | Thời gian hoàn thành |
| `How a call is priced` | How a call is priced | Một cuộc gọi được tính giá như thế nào |
| `JSON Editor` | JSON Editor | Trình chỉnh sửa JSON |
| `Last Tested` | Last Tested | Được kiểm tra lần cuối |
| `Login Info` | Login Info | Thông tin đăng nhập |
| `Max Retries` | Max Retries | Số lần thử lại tối đa |
| `Memory Hits` | Memory Hits | Lượt truy cập bộ nhớ |
| `Next reset` | Next reset | Lần đặt lại tiếp theo |
| `No changes made` | No changes made | Không có thay đổi nào được thực hiện |
| `No chat presets match your search` | No chat presets match your search | Không có cài đặt trước trò chuyện nào khớp với tìm kiếm của bạn |
| `No models to copy` | No models to copy | Không có mô hình nào để sao chép |
| ... | （剩余 14 处略）| |

### length-too-short（1 处，展示前 30）

| key | en | vi |
|---|---|---|
| `When enabled, prompts are scanned before reaching upstream models.` | When enabled, prompts are scanned before reaching upstream models. | Khi được bật, |

