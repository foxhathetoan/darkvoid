# Gmail SMTP Setup Guide

## 1. Tạo App Password

Gmail không cho phép dùng mật khẩu tài khoản trực tiếp cho SMTP. Bạn cần tạo **App Password**.

### Yêu cầu

- Tài khoản Google đã bật **2-Step Verification** (xác minh 2 bước).

### Các bước

1. Truy cập [https://myaccount.google.com/security](https://myaccount.google.com/security)
2. Đảm bảo **2-Step Verification** đã bật (nếu chưa, bật lên trước)
3. Truy cập [https://myaccount.google.com/apppasswords](https://myaccount.google.com/apppasswords)
4. Đặt tên app, ví dụ: `DarkVoid`
5. Nhấn **Create** — Google sẽ hiển thị mật khẩu 16 ký tự (dạng `xxxx xxxx xxxx xxxx`)
6. **Copy mật khẩu này** — bạn chỉ thấy nó 1 lần duy nhất

## 2. Cấu hình `.env`

Thêm các biến sau vào file `.env`:

```env
MAILER_PROVIDER=smtp
MAILER_HOST=smtp.gmail.com
MAILER_PORT=587
MAILER_USERNAME=your-email@gmail.com
MAILER_PASSWORD=xxxx xxxx xxxx xxxx
MAILER_FROM=DarkVoid <your-email@gmail.com>
MAILER_BASE_URL=http://localhost:3000
```

| Biến | Giá trị | Ghi chú |
|------|---------|---------|
| `MAILER_PROVIDER` | `smtp` | Dùng `nop` để tắt gửi mail thật (chỉ log) |
| `MAILER_HOST` | `smtp.gmail.com` | SMTP server của Gmail |
| `MAILER_PORT` | `587` | TLS (STARTTLS). Không dùng port 465 |
| `MAILER_USERNAME` | Email Gmail của bạn | Dùng đúng email đã tạo App Password |
| `MAILER_PASSWORD` | App Password 16 ký tự | **Không phải** mật khẩu đăng nhập Gmail |
| `MAILER_FROM` | `DarkVoid <email>` | Tên hiển thị + email gửi |
| `MAILER_BASE_URL` | URL frontend | Dùng để build link verify/reset trong email |

## 3. Lưu ý quan trọng

### Giới hạn gửi mail của Gmail

- **500 email/ngày** với tài khoản cá nhân
- **2000 email/ngày** với Google Workspace
- Nếu vượt giới hạn, Gmail sẽ tạm khoá gửi mail trong 24h

### Bảo mật

- **Không commit** file `.env` lên git (đã có trong `.gitignore`)
- App Password có thể bị thu hồi bất cứ lúc nào tại [https://myaccount.google.com/apppasswords](https://myaccount.google.com/apppasswords)
- Trong production, nên dùng dịch vụ email chuyên dụng (SES, SendGrid, Resend) thay vì Gmail

### Tắt gửi mail (dev mode)

Đặt `MAILER_PROVIDER=nop` — email sẽ chỉ được log ra console, không gửi thật:

```env
MAILER_PROVIDER=nop
```

## 4. Test nhanh

Sau khi cấu hình xong, gọi API register để kiểm tra:

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "your-email@gmail.com",
    "display_name": "Test User",
    "password": "SecurePass123"
  }'
```

Nếu cấu hình đúng, bạn sẽ nhận được 2 email:
1. **Welcome** — chào mừng tài khoản mới
2. **Verify Email** — link xác thực email

Kiểm tra log server nếu không nhận được email — lỗi SMTP sẽ hiển thị ở đó.
