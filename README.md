# BeGMT2 Auth API

Backend auth dengan Go, Gin, GORM, MySQL, JWT, bcrypt, dan reset password via token WhatsApp lewat Pancake.

Dokumentasi endpoint lengkap ada di [API_ENDPOINTS.md](API_ENDPOINTS.md).

## Role

- `user`
- `agent`
- `super_admin`
- `sales`
- `marketing`

## Setup

1. Buat database MySQL:

```sql
CREATE DATABASE begmt2 CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

2. Copy `.env.example` menjadi `.env`, lalu isi konfigurasi database dan Pancake.

Komisi agent bisa diubah lewat `.env`:

```env
AGENT_COMMISSION_PERCENT=5
```

Akun default testing dibuat otomatis setelah migration jika email belum ada:

```text
Super Admin: superadmin@example.com / password123
Sales: sales@example.com / password123
```

3. Jalankan:

```bash
go mod tidy
go run .
```

## Deploy ke Railway

Project ini sudah siap untuk Railway dengan `railway.json`.
Railway akan build binary Go dan menjalankan `./begmt2`.
Port akan otomatis mengikuti env `PORT` dari Railway, atau `APP_PORT` jika kamu set manual.

Tambahkan service MySQL di Railway, lalu set environment variables berikut di service backend:

```env
APP_ENV=production
JWT_SECRET=ganti-dengan-secret-panjang
JWT_EXPIRES_HOURS=24
GOOGLE_CLIENT_ID=your-google-oauth-client-id.apps.googleusercontent.com
PANCAKE_WEBHOOK_SECRET=ganti-dengan-random-secret-panjang
ANALYTICS_TIMEZONE=Asia/Jakarta

RESET_TOKEN_EXPIRES_MINUTES=15
AGENT_COMMISSION_PERCENT=5

DEFAULT_ADMIN_EMAIL=superadmin@example.com
DEFAULT_ADMIN_PASSWORD=ganti-password-kuat
DEFAULT_SALES_EMAIL=sales@example.com
DEFAULT_SALES_PASSWORD=ganti-password-kuat

SSO_CODE_EXPIRES_SECONDS=60
SSO_CLIENTS=website_a=https://gmtgroup2.vercel.app/sso/callback,website_utama=https://backstage-gmt-group.vercel.app/sso/callback
CORS_ALLOWED_ORIGINS=https://gmtgroup2.vercel.app,https://backstage-gmt-group.vercel.app

PANCAKE_PAGE_ID=waba_xxxxx
PANCAKE_PAGE_ACCESS_TOKEN=page-access-token
PANCAKE_WA_TEMPLATE_ID=optional-template-untuk-payment-instruction
PANCAKE_RESET_PASSWORD_TEMPLATE_ID=approved-template-reset-password
```

Jika backend dan MySQL berada dalam project Railway yang sama, aplikasi juga bisa membaca variable MySQL bawaan Railway:
`MYSQLHOST`, `MYSQLPORT`, `MYSQLUSER`, `MYSQLPASSWORD`, dan `MYSQLDATABASE`.
Format underscore seperti `MYSQL_HOST`, `MYSQL_PORT`, `MYSQL_USER`, `MYSQL_PASSWORD`, dan `MYSQL_DATABASE` juga didukung.
Kalau kamu memakai database eksternal, isi variable `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, dan `DB_NAME`.
Aplikasi juga bisa membaca `DATABASE_URL`, `MYSQL_URL`, atau `MYSQL_PUBLIC_URL` dengan format `mysql://user:password@host:port/database`.

## Endpoint

### Register

`POST /api/auth/register`

```json
{
  "name": "Admin",
  "ttl": "Jakarta, 10 Januari 2000",
  "phone_number": "081234567890",
  "gender": "laki-laki",
  "email": "admin@example.com",
  "domicile": "Jakarta",
  "company_name": "PT Contoh Maju",
  "job": "Manager",
  "instagram": "admin.ig",
  "facebook": "Admin FB",
  "tiktok": "admin.tt",
  "photo": "uploads/users/photo.jpg",
  "ktp_photo": "uploads/users/ktp.jpg",
  "full_address": "Jl. Contoh No. 10, Jakarta",
  "bank_name": "BCA",
  "account_number": "1234567890",
  "status": "active",
  "password": "password123",
  "role": "super_admin"
}
```

Jika `role` kosong, otomatis menjadi `user`.
Data utama masuk ke tabel `users`. Field detail masuk ke tabel `detail_users`.
Field `job`, `instagram`, `facebook`, `tiktok`, `photo`, `ktp_photo`, `full_address`, `bank_name`, `account_number`, dan `status` opsional dan bisa bernilai `NULL`.

Jika database sudah pernah dibuat sebelum pemisahan tabel, jalankan migrasi manual:

```bash
mysql -u root -p begmt2 < database/migrations/001_split_user_detail.sql
```

Jika tabel `detail_users` sudah ada dan hanya perlu menambah field nullable terbaru:

```bash
mysql -u root -p begmt2 < database/migrations/002_add_nullable_detail_user_fields.sql
```

### Login

`POST /api/auth/login`

```json
{
  "email": "admin@example.com",
  "password": "password123"
}
```

Response berisi JWT token.

### Register Google

`POST /api/auth/register/google`

```json
{
  "id_token": "google-id-token",
  "client": "website_a"
}
```

Jika email Google sudah terdaftar, response `409 Conflict`.
Jika berhasil, user baru dibuat dengan role `user` dan response berisi JWT token, session, dan data user.

### Login Google

`POST /api/auth/google`

```json
{
  "id_token": "google-id-token",
  "client": "website_a"
}
```

Backend memverifikasi `id_token` ke Google dan mencocokkan audience dengan `GOOGLE_CLIENT_ID`.
Jika email belum terdaftar, user baru otomatis dibuat dengan role `user`.
Response berisi JWT token, session, dan data user seperti login biasa.

### Lupa Password

`POST /api/auth/forgot-password`

```json
{
  "email": "admin@example.com"
}
```

Jika email ada, sistem membuat token 6 digit, menyimpan hash token ke database, lalu mengirim token ke nomor WhatsApp user melalui Pancake. Untuk pengiriman di luar window 24 jam WhatsApp, isi `PANCAKE_RESET_PASSWORD_TEMPLATE_ID` dengan template reset password yang sudah approved.

### Verifikasi Token

`POST /api/auth/verify-reset-token`

```json
{
  "email": "admin@example.com",
  "token": "123456"
}
```

### Ganti Password

`POST /api/auth/reset-password`

```json
{
  "email": "admin@example.com",
  "token": "123456",
  "new_password": "passwordbaru123"
}
```

### User Login Saat Ini

`GET /api/auth/me`

Header:

```text
Authorization: Bearer <token>
```

## Agent Service

### Wallet Agent

`GET /api/agent/wallet`

Header:

```text
Authorization: Bearer <token agent>
```

### Hitung dan Simpan Komisi Produk

`POST /api/agent/commissions`

Header:

```text
Authorization: Bearer <token agent>
```

Body:

```json
{
  "product_name": "Produk A",
  "product_price": 20000000,
  "discount_amount": 1000000
}
```

Rumus sementara:

```text
final_price = product_price - discount_amount
commission_amount = final_price * AGENT_COMMISSION_PERCENT / 100
```

Komisi otomatis menambah `agent_wallets.total_commission` dan `agent_wallets.available_balance`.

### Pengajuan Withdraw

`POST /api/agent/withdraws`

Header:

```text
Authorization: Bearer <token agent>
```

Body:

```json
{
  "amount": 500000
}
```

Status awal withdraw adalah `on_progress`. Amount akan dipindah dari `available_balance` ke `pending_withdraw`.

### List Withdraw Agent

`GET /api/agent/withdraws`

Header:

```text
Authorization: Bearer <token agent>
```

### List Semua Withdraw Untuk Super Admin

`GET /api/super-admin/withdraws`

Opsional filter:

```text
/api/super-admin/withdraws?status=on_progress
```

### Approve Withdraw Oleh Super Admin

`PUT /api/super-admin/withdraws/:id/approve`

Jika disetujui, status menjadi `approval`, `pending_withdraw` berkurang, dan `withdrawn_balance` bertambah.

Migrasi manual service agent:

```bash
mysql -u root -p begmt2 < database/migrations/003_agent_service.sql
```

## Product Service

Tabel product sementara berisi `id_product`, `namaproduct`, `foto`, `deskripsi`, `unit`, dan `price`.

### List dan Search Product

`GET /api/products`

Search:

```text
GET /api/products?search=produk
```

### Detail Product

`GET /api/products/:id`

### Create Product

`POST /api/products`

Body:

```json
{
  "namaproduct": "Produk A",
  "foto": "uploads/products/produk-a.jpg",
  "deskripsi": "Deskripsi produk A",
  "unit": "unit",
  "price": 20000000,
  "status": "tersedia",
  "komisi": 0
}
```

### Update Product

`PUT /api/products/:id`

### Delete Product

`DELETE /api/products/:id`

Migrasi manual product:

```bash
mysql -u root -p begmt2 < database/migrations/004_products.sql
```

## Preorder Service

Status PO:

- `draft`
- `in_review`
- `approve`
- `invalid`

### Create PO

`POST /api/preorders`

Body:

```json
{
  "id_product": 1,
  "id_agent": 2,
  "qty": 3,
  "nama_customer": "Customer A",
  "email": "customer@example.com",
  "alamat": "Jl. Customer No. 1",
  "no_hp": "081234567890",
  "catatan": "Catatan tambahan",
  "payment_mode": "split"
}
```

Sistem menghitung otomatis `subtotal`, `total_komisi`, dan `total`. Status awal adalah `draft`. `payment_mode` opsional: `full`, `100%`, atau `100` untuk pembayaran 100% sekali kirim; dan `split`, `50%`, atau `50` untuk 50% DP dan 50% pelunasan.

### List dan Search PO

`GET /api/preorders`

Filter:

```text
GET /api/preorders?search=customer&status=draft
```

Search mencari nama product, nama customer, email customer, dan nomor HP.

### Detail PO

`GET /api/preorders/:id`

### Update PO

`PUT /api/preorders/:id`

Hanya PO status `draft` yang bisa diupdate.

### Delete PO

`DELETE /api/preorders/:id`

### Submit PO

`POST /api/preorders/:id/submit`

Mengubah status dari `draft` menjadi `in_review`, membuat notifikasi untuk sales, dan mengirim event realtime via SSE.

### Realtime Notifikasi Sales

`GET /api/sales/notifications/stream`

Header:

```text
Authorization: Bearer <token sales>
```

Endpoint ini memakai Server-Sent Events.

### List Notifikasi

`GET /api/notifications`

Header:

```text
Authorization: Bearer <token>
```

Filter status:

```text
GET /api/notifications?status=belum_terbaca
GET /api/notifications?status=terbaca
```

Status notifikasi dihitung dari `read_at`: jika `NULL` maka `belum_terbaca`, jika terisi maka `terbaca`.

### Detail Notifikasi

`GET /api/notifications/:id`

### Tandai Notifikasi Terbaca

`PUT /api/notifications/:id/read`

### Tandai Semua Notifikasi Terbaca

`PUT /api/notifications/read-all`

### Update Status PO Oleh Sales

`PUT /api/sales/preorders/:id/status`

Header:

```text
Authorization: Bearer <token sales>
```

Approve:

```json
{
  "status": "approve"
}
```

Invalid:

```json
{
  "status": "invalid",
  "invalid_reason": "Data customer tidak valid"
}
```

Jika status menjadi `approve`, `total_komisi` PO masuk ke wallet agent dan tercatat di `agent_commissions`. Backend juga mengirim tagihan pertama via Pancake: 100% untuk `payment_mode=full`, atau DP 50% untuk `payment_mode=split`.

### Kirim Quotation Pembayaran Oleh Sales

`POST /api/sales/preorders/:id/payment-quotation`

Body:

```json
{
  "stage": "remaining"
}
```

`stage` tersedia: `full`, `dp`, atau `remaining`. Untuk mode `split`, sales dapat mengirim `dp` terlebih dahulu lalu `remaining` setelah bukti DP diupload.

### Upload Bukti Pembayaran Oleh Sales

`POST /api/sales/preorders/:id/payment-proof`

Content-Type: `multipart/form-data`.

```text
payment_proof: file jpg, jpeg, png, atau pdf
stage: full, dp, atau remaining
```

Migrasi manual preorder:

```bash
mysql -u root -p begmt2 < database/migrations/005_preorders.sql
```

### Apply Menjadi Agent

`POST /api/auth/apply-agent`

Header:

```text
Authorization: Bearer <token>
```

Body:

```json
{
  "job": "Sales Executive",
  "instagram": "user.ig",
  "facebook": "User FB",
  "tiktok": "user.tt",
  "photo": "uploads/users/photo.jpg",
  "ktp_photo": "uploads/users/ktp.jpg",
  "full_address": "Jl. Contoh No. 10, Jakarta",
  "bank_name": "BCA",
  "account_number": "1234567890"
}
```

Endpoint ini hanya untuk role `user`. Sistem akan mengisi data pengajuan dan mengubah `detail_users.status` menjadi `verif`.
Role tetap `user` sampai admin mengubah role lewat dashboard.

### Contoh Route Role Super Admin

`GET /api/super-admin/dashboard`

Header:

```text
Authorization: Bearer <token>
```
