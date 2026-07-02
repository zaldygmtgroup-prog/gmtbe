# API Endpoints

Dokumen ini menjelaskan endpoint API yang tersedia dan fitur yang memakai endpoint tersebut.

Base URL lokal:

```text
http://localhost:8080
```

Untuk endpoint yang membutuhkan login, kirim header:

```text
Authorization: Bearer <token>
```

Akun default testing dibuat otomatis setelah migration jika email belum ada:

```text
Super Admin: superadmin@example.com / password123
Sales: sales@example.com / password123
```

## Health

### `GET /health`

Dipakai untuk mengecek apakah server API sedang hidup.

Response:

```json
{
  "status": "ok"
}
```

## Auth

### `POST /api/auth/register`

Dipakai untuk fitur registrasi akun baru.

Role yang tersedia:

- `user`
- `agent`
- `super_admin`
- `sales`
- `marketing`

Body:

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
  "role": "user"
}
```

Catatan:

- Jika `role` kosong, otomatis menjadi `user`.
- Data utama masuk ke tabel `users`.
- Data tambahan masuk ke tabel `detail_users`.

### `POST /api/auth/login`

Dipakai untuk fitur login dan mendapatkan JWT token.

Body:

```json
{
  "email": "admin@example.com",
  "password": "password123"
}
```

Response berisi `token` yang dipakai untuk endpoint protected.

### `POST /api/auth/register/google`

Dipakai untuk registrasi akun baru menggunakan Google Sign-In.

Body:

```json
{
  "id_token": "google-id-token",
  "client": "website_a"
}
```

Jika email Google sudah terdaftar, response `409 Conflict`.
Jika berhasil, user baru dibuat dengan role `user` dan response berisi `token`, `session`, dan `user`.

### `POST /api/auth/google`

Dipakai untuk login menggunakan Google Sign-In dan mendapatkan JWT token backend.

Body:

```json
{
  "id_token": "google-id-token",
  "client": "website_a"
}
```

Catatan:

- Backend wajib memiliki env `GOOGLE_CLIENT_ID`.
- `id_token` diverifikasi ke Google dan `aud` harus sama dengan `GOOGLE_CLIENT_ID`.
- Jika email Google belum terdaftar, sistem membuat user baru dengan role `user`.
- Response berisi `token`, `session`, dan `user` seperti endpoint login biasa.

### `POST /api/auth/forgot-password`

Dipakai untuk fitur lupa password tahap pertama: cek email dan kirim token reset ke WhatsApp melalui Pancake.

Body:

```json
{
  "email": "admin@example.com"
}
```

Jika email terdaftar, sistem membuat token 6 digit dan mengirimkannya ke nomor WhatsApp user. Untuk pengiriman di luar window 24 jam WhatsApp, backend harus memakai template reset password Pancake/WhatsApp yang sudah approved melalui env `PANCAKE_RESET_PASSWORD_TEMPLATE_ID`.

### `POST /api/auth/verify-reset-token`

Dipakai untuk fitur verifikasi token reset password.

Body:

```json
{
  "email": "admin@example.com",
  "token": "123456"
}
```

### `POST /api/auth/reset-password`

Dipakai untuk fitur mengganti password setelah token valid.

Body:

```json
{
  "email": "admin@example.com",
  "token": "123456",
  "new_password": "passwordbaru123"
}
```

### `GET /api/auth/me`

Dipakai untuk mengambil data user yang sedang login.

Auth: wajib login.

### `POST /api/auth/apply-agent`

Dipakai untuk fitur user mengajukan diri menjadi agent.

Auth: wajib login sebagai `user`.

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

Efek:

- Mengisi/update data `detail_users`.
- Set `detail_users.status = "verif"`.
- Role tetap `user` sampai admin mengubah role.

## Products

### `GET /api/products`

Dipakai untuk fitur list product.

Query opsional:

```text
?search=keyword
```

Contoh:

```http
GET /api/products?search=rumah
```

Response:

```json
{
  "products": [
    {
      "id": 1,
      "namaproduct": "GMT Lighting Package",
      "foto": "uploads/products/lighting.jpg",
      "deskripsi": "Paket lighting event indoor",
      "unit": "paket",
      "price": 20000000
      "status": "tersedia",
      "komisi": 0,
      "created_at": "2026-06-12T03:00:00Z",
      "updated_at": "2026-06-12T03:00:00Z"
    }
  ]
}
```

### `GET /api/products/:id`

Dipakai untuk fitur detail product.

Contoh:

```http
GET /api/products/1
```

### `POST /api/products`

Dipakai untuk fitur tambah product.

Untuk sementara endpoint ini belum dibatasi role.

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

### `PUT /api/products/:id`

Dipakai untuk fitur edit product.

Untuk sementara endpoint ini belum dibatasi role.

Body:

```json
{
  "namaproduct": "Produk A Update",
  "foto": "uploads/products/produk-a.jpg",
  "deskripsi": "Deskripsi produk A update",
  "unit": "unit",
  "price": 21000000
}
```

### `DELETE /api/products/:id`

Dipakai untuk fitur hapus product.

Untuk sementara endpoint ini belum dibatasi role.

## Preorders

### `GET /api/preorders`

Dipakai untuk fitur list PO.

Query opsional:

```text
?search=keyword
?status=draft
?search=keyword&status=in_review
```

Search mencari:

- nama product
- nama customer
- email customer
- nomor HP customer

### `GET /api/preorders/:id`

Dipakai untuk fitur detail PO.

### `POST /api/preorders`

Dipakai untuk fitur membuat PO baru (multi-product).

Body:

```json
{
  "nama_customer": "PT Cahaya Eventindo",
  "email": "procurement@cahayaevent.id",
  "alamat": "Jl. Gatot Subroto No. 12",
  "no_hp": "081234567890",
  "catatan": "Butuh instalasi akhir bulan",
  "items": [
    {
      "id_product": 1,
      "qty": 1,
      "discount_percent": 5
    },
    {
      "id_product": 2,
      "qty": 1,
      "discount_percent": 7
    }
  ]
}
```

Efek:

- Status awal `draft`.
- Sistem menghitung `subtotal`, `total_discount`, `total`, dan `total_komisi`.
- Komisi belum masuk wallet saat status masih `draft`.

Response:

```json
{
  "message": "Preorder created",
  "preorder": {
    "id": 12,
    "po_number": "INV/GMT/2026/06/0001",
    "status": "draft",
    "subtotal": 55000000,
    "total_discount": 3450000,
    "total": 51550000,
    "total_komisi": 5155000
  }
}
```

### `PUT /api/preorders/:id`

Dipakai untuk fitur edit PO.

Hanya PO dengan status `draft` yang bisa diubah oleh Agent pemilik.

Body sama seperti create PO.

### `DELETE /api/preorders/:id`

Dipakai untuk fitur hapus PO.

### `POST /api/preorders/:id/submit`

Dipakai untuk fitur submit PO ke sales.

Efek:

- Status berubah dari `draft` menjadi `in_review`.
- Membuat notifikasi untuk role `sales`.
- Mengirim realtime event ke endpoint SSE sales.
- Komisi belum masuk wallet saat status `in_review`.

Rule:

- Wajib upload bukti transfer terlebih dulu lewat `POST /api/preorders/:id/payment-proof`.

### `POST /api/preorders/:id/payment-proof`

Dipakai untuk upload bukti transfer sebelum PO disubmit ke sales.

Auth: wajib login sebagai `agent` official.

Content-Type: `multipart/form-data`.

Field:

```text
payment_proof: file jpg, jpeg, png, atau pdf
```

Rule:

- Hanya pemilik PO.
- Hanya status `draft`.
- File selain `jpg`, `jpeg`, `png`, dan `pdf` ditolak.

Response:

```json
{
  "message": "payment proof uploaded",
  "payment": {
    "payment_status": "pending",
    "payment_proof": "/uploads/payment_proofs/1781234567890.png"
  }
}
```

### Status PO

Status yang tersedia:

- `draft`
- `in_review`
- `approve`
- `invalid`

Rule komisi:

- `draft`: belum masuk wallet agent.
- `in_review`: belum masuk wallet agent.
- `invalid`: tidak masuk wallet agent.
- `approve`: `total_komisi` masuk ke wallet agent.

## Sales

### `GET /api/sales/notifications/stream`

Dipakai untuk fitur realtime notification sales.

Auth: wajib login sebagai `sales`.

Endpoint ini memakai Server-Sent Events.

### `PUT /api/sales/preorders/:id/status`

Dipakai sales untuk approve atau invalid PO.

Auth: wajib login sebagai `sales`.

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

Efek:

- Jika `approve`, komisi PO masuk ke wallet agent.
- Jika `approve`, backend mengirim pesan WhatsApp ke customer melalui Pancake:
  1. Instruksi pembayaran seperti sebelumnya.
  2. File invoice/quotation PDF dari PO tersebut.
- Jika `invalid`, komisi tidak masuk wallet agent.

Catatan pengiriman invoice WhatsApp:

- Backend membuat PDF yang sama dengan `GET /api/preorders/:id/pdf`.
- Backend upload PDF ke Pancake `upload_contents`, lalu mengirimnya sebagai document message memakai `content_ids`.
- Pastikan env Pancake sudah terisi:

```env
PANCAKE_PAGE_ID=waba_xxx
PANCAKE_PAGE_ACCESS_TOKEN=xxx
```

## Notifications

### `GET /api/notifications`

Dipakai untuk fitur list notifikasi user berdasarkan role login.

Auth: wajib login.

Filter status:

```http
GET /api/notifications?status=belum_terbaca
GET /api/notifications?status=terbaca
```

Status dihitung dari field `read_at`:

- `read_at = NULL`: `belum_terbaca`
- `read_at != NULL`: `terbaca`

### `GET /api/notifications/:id`

Dipakai untuk fitur detail notifikasi.

Auth: wajib login.

### `PUT /api/notifications/:id/read`

Dipakai untuk fitur tandai satu notifikasi sebagai terbaca.

Auth: wajib login.

### `PUT /api/notifications/read-all`

Dipakai untuk fitur tandai semua notifikasi role user login sebagai terbaca.

Auth: wajib login.

## Agent

### `GET /api/agent/wallet`

Dipakai untuk fitur melihat wallet agent.

Auth: wajib login sebagai `agent`.

Response:

```json
{
  "wallet": {
    "total_commission": 12500000,
    "available_balance": 8500000,
    "pending_withdraw": 1500000,
    "withdrawn_balance": 2500000
  }
}
```

### `POST /api/agent/commissions`

Dipakai untuk fitur simulasi/hitung komisi product secara langsung.

Auth: wajib login sebagai `agent`.

Body:

```json
{
  "product_name": "Produk A",
  "product_price": 20000000,
  "discount_amount": 1000000
}
```

Rumus:

```text
final_price = product_price - discount_amount
commission_amount = final_price * AGENT_COMMISSION_PERCENT / 100
```

Efek:

- Komisi langsung masuk ke wallet agent.
- Tercatat di tabel `agent_commissions`.

### `POST /api/agent/withdraws`

Dipakai untuk fitur pengajuan withdraw agent.

Auth: wajib login sebagai `agent`.

Body:

```json
{
  "amount": 500000
}
```

Efek:

- Status withdraw awal `on_progress`.
- `available_balance` berkurang.
- `pending_withdraw` bertambah.
- `withdraw_number` terbuat secara otomatis (misal: `WD-1004`).

Response:

```json
{
  "message": "Withdraw request created",
  "withdraw": {
    "id": 13,
    "withdraw_number": "WD-1004",
    "amount": 500000,
    "status": "on_progress",
    "created_at": "2026-06-12T03:10:00.000Z"
  },
  "wallet": {
    "total_commission": 12500000,
    "available_balance": 8000000,
    "pending_withdraw": 2000000,
    "withdrawn_balance": 2500000
  }
}
```

### `GET /api/agent/withdraws`

Dipakai untuk fitur list pengajuan withdraw milik agent yang login.

Auth: wajib login sebagai `agent`.

Response:

```json
{
  "withdraws": [
    {
      "id": 12,
      "withdraw_number": "WD-1003",
      "amount": 1500000,
      "status": "on_progress",
      "created_at": "2026-06-10T09:20:00.000Z",
      "approved_at": null
    }
  ]
}
```

### `GET /api/agent/preorders`

Dipakai untuk mengambil daftar preorder milik agent yang sedang login.

Auth: wajib login sebagai `agent`.

Query opsional:

- `?status=draft`
- `?status=in_review`

Response:

```json
{
  "preorders": [
    {
      "id": 12,
      "po_number": "INV/GMT/2026/06/0001",
      "status": "in_review",
      "nama_customer": "PT Cahaya Eventindo",
      "email": "procurement@cahayaevent.id",
      "no_hp": "081234567890",
      "alamat": "Jl. Gatot Subroto No. 12",
      "catatan": "Butuh instalasi akhir bulan",
      "subtotal": 55000000,
      "total_discount": 3450000,
      "total": 51550000,
      "total_komisi": 5155000,
      "created_at": "2026-06-11T09:15:00.000Z",
      "items": [
        {
          "id": 1,
          "id_product": 1,
          "namaproduct": "GMT Lighting Package",
          "foto": "uploads/products/lighting.jpg",
          "deskripsi": "Paket lighting event indoor",
          "unit": "paket",
          "unit_price": 20000000,
          "qty": 1,
          "discount_percent": 5,
          "discount_amount": 1000000,
          "subtotal": 20000000,
          "total": 19000000,
          "komisi": 1900000
        }
      ]
    }
  ]
}
```

### `GET /api/preorders/:id/pdf`

Dipakai untuk mencetak PDF PO.

Auth: wajib login sebagai `agent`.
Rule: Agent hanya bisa mencetak PDF preorder miliknya sendiri.

Response:

- `Content-Type: application/pdf` (binary PDF content)

### `GET /api/agent/preorders/stream`

Dipakai untuk fitur realtime monitoring perubahan status PO agent. Endpoint ini memakai Server-Sent Events (SSE).

Auth: wajib login sebagai `agent` (official).

Response berupa stream dengan format:

```text
event: preorder_updated
data: {"id": 102, "po_number": "PO-102", "status": "approve", "payment_status": "unpaid", "total": 150000000, "total_komisi": 1500000}
```

## Super Admin

### `GET /api/super-admin/dashboard`

Dipakai untuk fitur dashboard super admin sementara.

Auth: wajib login sebagai `super_admin`.

### `GET /api/super-admin/withdraws`

Dipakai untuk fitur list semua pengajuan withdraw.

Auth: wajib login sebagai `super_admin`.

Filter status:

```http
GET /api/super-admin/withdraws?status=on_progress
```

### `PUT /api/super-admin/withdraws/:id/approve`

Dipakai untuk fitur approve withdraw agent.

Auth: wajib login sebagai `super_admin`.

Efek:

- Status withdraw berubah menjadi `approval`.
- `pending_withdraw` berkurang.
- `withdrawn_balance` bertambah.

## Onboarding Agent

### `GET /api/agent/onboarding/videos`

Dipakai untuk melihat daftar video onboarding yang harus ditonton oleh agent.

Auth: wajib login sebagai `agent`.

Response:

```json
{
  "videos": [
    {
      "id": 1,
      "slug": "agent-introduction",
      "title": "Pengenalan Role Agent",
      "description": "Dasar tugas agent...",
      "video_url": "https://...",
      "duration_seconds": 380,
      "sort_order": 1,
      "is_required": true
    }
  ]
}
```

### `GET /api/agent/onboarding/progress`

Dipakai untuk mengambil detail progress onboarding dari agent yang sedang login.

Auth: wajib login sebagai `agent`.

Response:

```json
{
  "completed_count": 1,
  "total_required": 3,
  "completion_percent": 33,
  "is_completed": false,
  "progress": [
    {
      "video_id": 1,
      "slug": "agent-introduction",
      "status": "completed",
      "watched_seconds": 380,
      "completed_at": "2026-06-12T02:30:00.000Z"
    },
    {
      "video_id": 2,
      "slug": "product-and-po-flow",
      "status": "not_started",
      "watched_seconds": 0,
      "completed_at": null
    }
  ]
}
```

### `POST /api/agent/onboarding/progress`

Dipakai untuk menyimpan progress saat video ditonton.

Auth: wajib login sebagai `agent`.

Body:

```json
{
  "video_id": 1,
  "watched_seconds": 380,
  "duration_seconds": 380,
  "status": "completed"
}
```

Aturan penting:

- Jika `watched_seconds >= duration_seconds * 0.9`, status akan dipromosikan otomatis menjadi `completed`.
- Jika status dikirim sebagai `completed` tetapi `watched_seconds < duration_seconds * 0.9`, request ditolak.
- Sequence validation: video dengan urutan lebih lanjut tidak bisa di-complete jika video sebelumnya belum berstatus `completed`.
- Field `completed_at` hanya diisi sekali saat pertama kali video berstatus `completed`.

Response:

```json
{
  "message": "Progress saved",
  "progress": {
    "video_id": 1,
    "status": "completed",
    "watched_seconds": 380,
    "completed_at": "2026-06-12T02:30:00.000Z"
  }
}
```

### `DELETE /api/agent/onboarding/progress`

Dipakai untuk mereset seluruh progress onboarding agent (berguna untuk testing).

Auth: wajib login sebagai `agent`.

Response:

```json
{
  "message": "Progress reset successfully"
}
```

## SSO Beda Domain

SSO memakai one-time code. JWT tidak dikirim lewat URL; URL callback hanya membawa `code` sementara.

Konfigurasi `.env`:

```env
SSO_CODE_EXPIRES_SECONDS=60
SSO_CLIENTS=website_a=https://gmtgroup2.vercel.app/sso/callback,website_utama=https://backstage-gmt-group.vercel.app/sso/callback
```

### `GET /api/auth/session`

Auth: wajib login.

Response:

```json
{
  "authenticated": true,
  "session_id": "session-token",
  "user": {}
}
```

### `POST /api/auth/sso/code`

Dipanggil oleh website asal yang sudah login sebelum redirect ke website tujuan.

Auth: wajib login.

Body:

```json
{
  "target_client": "website_utama",
  "state": "optional-csrf-state"
}
```

Response:

```json
{
  "code": "one-time-code",
  "expires_at": "2026-06-12T03:30:00+07:00",
  "redirect_url": "https://backstage-gmt-group.vercel.app/sso/callback?code=one-time-code&state=optional-csrf-state"
}
```

### `POST /api/auth/sso/exchange`

Dipanggil oleh website tujuan dari halaman callback untuk menukar `code` menjadi JWT milik domain tersebut.

Body:

```json
{
  "code": "one-time-code",
  "target_client": "website_utama"
}
```

Response:

```json
{
  "message": "sso exchange successful",
  "token": "jwt",
  "session": {},
  "user": {}
}
```

### `POST /api/auth/logout`

Auth: wajib login.

Logout mencabut semua session aktif user di backend. Website lain akan ikut logout saat memanggil endpoint protected atau `GET /api/auth/session`.

Response:

```json
{
  "message": "logout successful"
}
```

## Pancake Chat Analytics

Set environment berikut sebelum mengaktifkan webhook:

```env
PANCAKE_WEBHOOK_SECRET=random-secret-panjang
ANALYTICS_TIMEZONE=Asia/Jakarta
```

### `POST /api/integrations/pancake/webhook?secret=<PANCAKE_WEBHOOK_SECRET>`

URL publik yang didaftarkan pada Pancake **Settings > Tools > Webhook**. Event `messaging`
disimpan secara idempotent; event `post` dan `subscription` diakui tetapi diabaikan oleh analytics.
Secret juga dapat dikirim melalui header `X-Pancake-Webhook-Secret`.

### `POST /api/pancake/conversions`

Auth: role `super_admin`, `sales`, atau `marketing`.

Catat penjualan dari Pancake POS atau order system agar conversion rate dan atribusi campaign
dapat dihitung. `external_order_id` bersifat idempotent.

```json
{
  "external_order_id": "ORDER-1001",
  "page_id": "waba_1234567890",
  "conversation_id": "waba_1234567890_628123456789",
  "customer_id": "628123456789",
  "campaign_id": "CMP-01",
  "campaign_name": "Promo Juni",
  "product_name": "Produk A",
  "amount": 2500000,
  "converted_at": "2026-06-22T10:00:00+07:00"
}
```

### `GET /api/pancake/analytics`

Auth: role `super_admin`, `sales`, atau `marketing`.

Secara default menghitung hari ini menurut `ANALYTICS_TIMEZONE`. Filter opsional:

```text
GET /api/pancake/analytics?page_id=waba_123&from=2026-06-01T00:00:00%2B07:00&to=2026-07-01T00:00:00%2B07:00
```

Response memuat `new_leads`, `most_asked_products`, `chat_to_purchase`,
`closing_potential_customers`, `retarget_customers`, `wa_campaign_sales`,
`customer_activity`, dan `top_keywords`.

Catatan: analytics mulai lengkap sejak webhook diaktifkan. Data penjualan harus dikirim ke endpoint
conversion karena webhook messaging Pancake tidak memuat transaksi.

## Education

### `GET /api/educations`

Dipakai untuk mengambil daftar acara, pelatihan, atau seminar yang akan datang. Endpoint ini bersifat publik.
Jika frontend mengirim `Authorization: Bearer <token>` yang valid, setiap item akan menyertakan
`is_registered` berdasarkan status pendaftaran user tersebut. Jika token tidak dikirim atau tidak
valid, user dianggap guest dan `is_registered` bernilai `false`.

Query opsional:
- `?month=2026-06` (Filter berdasarkan bulan)
- `?type=Offline` (Filter berdasarkan kategori acara)
- `?status=Available` (Filter berdasarkan status acara)
- `?page=1&limit=10` (Pagination)

Response:
```json
{
  "success": true,
  "message": "List of education events retrieved successfully",
  "data": [
    {
      "id": "edu_12345",
      "title": "d&b Electro Acoustic Training",
      "date": "2026-06-29",
      "time": "12:00",
      "type": "Offline",
      "status": "Available",
      "is_registered": false,
      ...
    }
  ],
  "meta": {
    "total": 1,
    "page": 1,
    "limit": 10,
    "total_pages": 1
  }
}
```

### `GET /api/educations/:id`

Dipakai untuk mengambil informasi lengkap tentang satu acara berdasarkan ID-nya. Endpoint ini bersifat publik.
Frontend boleh mengirim `Authorization: Bearer <token>`; jika token valid, response akan menunjukkan
apakah user sudah terdaftar pada event tersebut.

Response:
```json
{
  "success": true,
  "data": {
    "id": "edu_12345",
    "title": "d&b Electro Acoustic Training",
    "description": "...",
    "full_description": "...",
    "max_attendees": 50,
    "current_attendees": 12,
    "is_registered": true,
    ...
  }
}
```

### `POST /api/educations`

Dipakai untuk fitur membuat acara education baru.
Auth: wajib login sebagai `super_admin`.

### `PUT /api/educations/:id`

Dipakai untuk fitur mengubah acara education yang sudah ada.
Auth: wajib login sebagai `super_admin`.

### `DELETE /api/educations/:id`

Dipakai untuk fitur menghapus acara education.
Auth: wajib login sebagai `super_admin`.

### `POST /api/educations/:id/register`

Dipakai untuk mendaftarkan pengguna yang sedang login ke acara tertentu.
Auth: wajib login (mengirim `Authorization: Bearer <token>`).

Body:
```json
{
  "salutation": "Ms",
  "first_name": "Fety",
  "surname": "Group",
  "email": "fety@gmtgroup.co.id",
  "confirm_email": "fety@gmtgroup.co.id",
  "phone_landline": "+6221...",
  "phone_mobile": "+62812...",
  "company": "GMT Group",
  "position": "Staff",
  "address": {
    "street": "Jl. Contoh No 123",
    "postcode": "12345",
    "town": "Jakarta",
    "country": "Indonesia"
  },
  "meal_preference": "None",
  "additional_information": "Alergi kacang",
  "consents": {
    "conditions_of_participation": true,
    "privacy_policy": true,
    "marketing_updates": false
  },
  "recaptcha_token": "token_from_google_recaptcha"
}
```

Response Berhasil (201 Created):
```json
{
  "success": true,
  "message": "Registration successful. Check your email for the ticket.",
  "data": {
    "registration_id": "reg_98765",
    "event_id": "edu_12345",
    "user_id": "usr_111",
    "status": "Confirmed"
  }
}
```

Error Umum:
- `400 Bad Request`: Validasi gagal (misalnya email tidak cocok atau belum setuju privacy policy).
- `409 Conflict`: Slot acara sudah penuh atau pengguna sudah terdaftar sebelumnya.
