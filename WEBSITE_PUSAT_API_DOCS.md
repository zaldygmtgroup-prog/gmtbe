# Dokumentasi API Website Pusat

Dokumen ini untuk tim Website Pusat. Isinya mencakup semua fitur backend BeGMT2 yang relevan untuk frontend pusat: auth, SSO beda domain, user session, product, preorder, sales approval, notification, agent wallet, withdraw, super admin, onboarding agent, dan PDF PO.

Base URL API production:

```text
https://api-domain-anda.com
```

Base URL lokal:

```text
http://localhost:8080
```

Header untuk endpoint protected:

```text
Authorization: Bearer <token>
```

Format umum error:

```json
{
  "message": "error message",
  "error": "detail optional"
}
```

## Role Dan Akses

Role yang tersedia:

```text
user
agent
super_admin
sales
marketing
```

Ringkasan akses:

| Role          | Fitur Utama                                                  |
| ------------- | ------------------------------------------------------------ |
| `user`        | login, profile, apply menjadi agent                          |
| `agent`       | onboarding, product list, preorder, PDF PO, wallet, withdraw |
| `sales`       | menerima notifikasi PO, approve/invalid preorder             |
| `super_admin` | dashboard admin, list/approve withdraw                       |
| `marketing`   | role sudah tersedia, endpoint khusus belum ada               |

Catatan:

- Product CRUD saat ini belum dibatasi role di route backend.
- Endpoint preorder umum `/api/preorders` wajib login, tetapi ownership hanya dicek pada update/delete/submit/PDF.
- Endpoint khusus agent memakai prefix `/api/agent`.

## Auth Dan Session

### Login Website Pusat

```text
POST /api/auth/login
```

Body:

```json
{
  "email": "sales@example.com",
  "password": "password123",
  "client": "website_utama"
}
```

Response:

```json
{
  "message": "login successful",
  "token": "jwt-token",
  "session": {
    "session_id": "session-id",
    "user_id": 2,
    "client": "website_utama",
    "expires_at": "2026-06-13T03:00:00+07:00",
    "revoked_at": null
  },
  "user": {
    "id": 2,
    "name": "Sales",
    "email": "sales@example.com",
    "role": "sales"
  }
}
```

Frontend Website Pusat perlu menyimpan `token` sebagai session lokal Website Pusat. Disarankan memakai HttpOnly cookie milik domain Website Pusat jika arsitektur frontend memungkinkan.

### Register User

```text
POST /api/auth/register
```

Body:

```json
{
  "name": "User Baru",
  "ttl": "Jakarta, 10 Januari 2000",
  "phone_number": "081234567890",
  "gender": "laki-laki",
  "email": "user@example.com",
  "domicile": "Jakarta",
  "company_name": "PT Contoh",
  "job": "Manager",
  "instagram": "user.ig",
  "facebook": "User FB",
  "tiktok": "user.tt",
  "photo": "uploads/users/photo.jpg",
  "ktp_photo": "uploads/users/ktp.jpg",
  "full_address": "Jl. Contoh No. 10",
  "bank_name": "BCA",
  "account_number": "1234567890",
  "status": "active",
  "password": "password123",
  "role": "user"
}
```

Jika `role` kosong, backend otomatis memakai `user`.

### Cek Session

```text
GET /api/auth/session
```

Auth: wajib login.

Response:

```json
{
  "authenticated": true,
  "session_id": "session-id",
  "user": {
    "id": 2,
    "name": "Sales",
    "email": "sales@example.com",
    "role": "sales"
  }
}
```

Jika user logout dari Website A atau session dicabut, endpoint ini akan mengembalikan `401`.

### Profile User Login

```text
GET /api/auth/me
```

Auth: wajib login.

Response:

```json
{
  "user": {
    "id": 1,
    "name": "User Name",
    "email": "user@example.com",
    "role": "user",
    "detail_user": {
      "company_name": "PT Contoh",
      "status": "verif"
    }
  }
}
```

### Logout Global

```text
POST /api/auth/logout
```

Auth: wajib login.

Efek:

- Semua session aktif user dicabut di backend.
- Website Pusat tetap harus menghapus token lokalnya sendiri.
- Website A akan ikut dianggap logout saat memanggil `/api/auth/session` atau endpoint protected.

Response:

```json
{
  "message": "logout successful"
}
```

### Forgot Password

```text
POST /api/auth/forgot-password
```

Body:

```json
{
  "email": "user@example.com"
}
```

Jika email terdaftar, token reset dikirim ke nomor WhatsApp user melalui Pancake. Untuk pengiriman di luar window 24 jam WhatsApp, backend memakai template reset password approved dari `PANCAKE_RESET_PASSWORD_TEMPLATE_ID`.

### Verify Reset Token

```text
POST /api/auth/verify-reset-token
```

Body:

```json
{
  "email": "user@example.com",
  "token": "123456"
}
```

### Reset Password

```text
POST /api/auth/reset-password
```

Body:

```json
{
  "email": "user@example.com",
  "token": "123456",
  "new_password": "passwordbaru123"
}
```

## SSO Dengan Website A

SSO dipakai karena Website A dan Website Pusat beda domain. Token JWT tidak dikirim lewat URL. URL hanya membawa `code` sekali pakai.

Konfigurasi backend:

```env
SSO_CODE_EXPIRES_SECONDS=60
SSO_CLIENTS=website_a=https://gmtgroup2.vercel.app/sso/callback,website_utama=https://backstage-gmt-group.vercel.app/sso/callback
```

### Website Pusat Menerima User Dari Website A

Website A akan redirect user ke:

```text
https://backstage-gmt-group.vercel.app/sso/callback?code=<one-time-code>&state=<state>
```

Di halaman callback Website Pusat, ambil `code`, lalu panggil:

```text
POST /api/auth/sso/exchange
```

Body:

```json
{
  "code": "one-time-code-dari-url",
  "target_client": "website_utama"
}
```

Response:

```json
{
  "message": "sso exchange successful",
  "token": "jwt-token-website-pusat",
  "session": {
    "session_id": "session-id",
    "client": "website_utama"
  },
  "user": {
    "id": 10,
    "name": "User Name",
    "email": "user@example.com",
    "role": "agent"
  }
}
```

Setelah sukses:

```text
1. Simpan token sebagai session Website Pusat.
2. Redirect user ke halaman dashboard sesuai role.
```

### Website Pusat Mengirim User Ke Website A

Jika user sudah login di Website Pusat dan perlu dibawa ke Website A:

```text
POST /api/auth/sso/code
```

Auth: wajib login.

Body:

```json
{
  "target_client": "website_a",
  "state": "random-state"
}
```

Response:

```json
{
  "code": "one-time-code",
  "expires_at": "2026-06-12T03:01:00+07:00",
  "redirect_url": "https://gmtgroup2.vercel.app/sso/callback?code=one-time-code&state=random-state"
}
```

Frontend redirect ke `redirect_url`.

## Apply Menjadi Agent

```text
POST /api/auth/apply-agent
```

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
  "full_address": "Jl. Contoh No. 10",
  "bank_name": "BCA",
  "account_number": "1234567890"
}
```

Efek:

- Mengisi data pengajuan agent di `detail_users`.
- Set `detail_users.status = "verif"`.
- Role tetap `user` sampai admin mengubah role melalui proses terpisah.

## Products

### List Product

```text
GET /api/products
GET /api/products?search=keyword
```

Response:

```json
{
  "products": [
    {
      "id": 1,
      "namaproduct": "Produk A",
      "foto": "uploads/products/produk-a.jpg",
      "deskripsi": "Deskripsi produk",
      "unit": "unit",
      "price": 20000000,
      "status": "tersedia",
      "komisi": 0,
      "created_at": "2026-06-12T03:00:00+07:00",
      "updated_at": "2026-06-12T03:00:00+07:00"
    }
  ]
}
```

### Detail Product

```text
GET /api/products/:id
```

### Create Product

```text
POST /api/products
```

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

```text
PUT /api/products/:id
```

Body sama seperti create.

### Delete Product

```text
DELETE /api/products/:id
```

## Preorders

Status PO:

```text
draft
in_review
approve
invalid
```

Status pembayaran:

```text
unpaid
pending
paid
expired
failed
refund
```

Rule komisi:

- `draft`: komisi belum masuk wallet.
- `in_review`: komisi belum masuk wallet.
- `invalid`: komisi tidak masuk wallet.
- `approve`: `total_komisi` masuk wallet agent.

### List Semua PO

```text
GET /api/preorders
GET /api/preorders?search=customer
GET /api/preorders?status=in_review
GET /api/preorders?search=customer&status=in_review
```

Auth: wajib login.

Search mencari nama customer, email, nomor HP, dan nama product snapshot.

### Detail PO

```text
GET /api/preorders/:id
```

Auth: wajib login.

### Create PO Multi-Product

```text
POST /api/preorders
```

Auth: wajib login. `id_agent` diambil dari user login.

Body:

```json
{
  "nama_customer": "PT Cahaya Eventindo",
  "nama_perusahaan": "PT Cahaya Eventindo",
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
      "qty": 2,
      "discount_percent": 10
    }
  ]
}
```

Response:

```json
{
  "message": "Preorder created",
  "preorder": {
    "id": 12,
    "po_number": "INV/GMT/2026/06/0001",
    "status": "draft",
    "payment_status": "unpaid",
    "subtotal": 55000000,
    "total_discount": 3450000,
    "total": 51550000,
    "total_komisi": 2577500
  }
}
```

### Update PO

```text
PUT /api/preorders/:id
```

Auth: wajib login.

Rule:

- Hanya pemilik PO.
- Hanya status `draft`.

Body sama seperti create.

### Delete PO

```text
DELETE /api/preorders/:id
```

Rule:

- Hanya pemilik PO.
- Hanya status `draft`.

### Submit PO Ke Sales

```text
POST /api/preorders/:id/submit
```

Rule:

- Hanya pemilik PO.
- Hanya status `draft`.
- Wajib sudah upload bukti transfer lewat `POST /api/preorders/:id/payment-proof`.

Efek:

- Status berubah ke `in_review`.
- Notifikasi dibuat untuk role `sales`.
- Event realtime dikirim ke SSE sales.

### Upload Bukti Transfer

```text
POST /api/preorders/:id/payment-proof
```

Auth: wajib login sebagai `agent` official.

Content-Type: `multipart/form-data`.

Field:

```text
payment_proof: file jpg, jpeg, png, atau pdf
```

Rule:

- Agent hanya bisa upload bukti transfer untuk PO miliknya sendiri.
- Hanya PO dengan status `draft`.
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

### Cetak PDF PO

```text
GET /api/preorders/:id/pdf
```

Auth: wajib login.

Rule:

- Agent hanya bisa mencetak PDF miliknya sendiri.
- PDF `QUOTATION` menampilkan kop surat GMT dan detail PO.

Response:

```text
Content-Type: application/pdf
```

## Agent Area

Semua endpoint di bagian ini wajib login sebagai `agent`.

### List PO Milik Agent

```text
GET /api/agent/preorders
GET /api/agent/preorders?status=draft
```

### Wallet Agent

```text
GET /api/agent/wallet
```

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

### Hitung Komisi Manual

```text
POST /api/agent/commissions
```

Body:

```json
{
  "product_name": "Produk A",
  "product_price": 20000000,
  "discount_amount": 1000000
}
```

Efek:

- Komisi langsung masuk wallet agent.
- Tercatat di `agent_commissions`.

### Create Withdraw

```text
POST /api/agent/withdraws
```

Body:

```json
{
  "amount": 500000
}
```

Efek:

- Status awal `on_progress`.
- `available_balance` berkurang.
- `pending_withdraw` bertambah.

### List Withdraw Agent

```text
GET /api/agent/withdraws
```

Response:

```json
{
  "withdraws": [
    {
      "id": 12,
      "withdraw_number": "WD-1003",
      "amount": 1500000,
      "status": "on_progress",
      "created_at": "2026-06-10T09:20:00+07:00",
      "approved_at": null
    }
  ]
}
```

## Sales Area

Semua endpoint di bagian ini wajib login sebagai `sales`.

### Realtime Notifikasi PO Baru

```text
GET /api/sales/notifications/stream
```

Endpoint memakai Server-Sent Events.

Contoh frontend:

```js
const stream = new EventSource("/api/sales/notifications/stream");

stream.addEventListener("notification", (event) => {
  const data = JSON.parse(event.data);
  console.log(data);
});
```

Jika frontend perlu mengirim Authorization header ke SSE, gunakan polyfill EventSource yang mendukung custom header, atau buat proxy dari server frontend.

### Update Status PO

```text
PUT /api/sales/preorders/:id/status
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

Rule:

- Hanya PO status `in_review`.
- Jika approve, komisi masuk wallet agent.

## Notifications

Semua endpoint notification wajib login.

### List Notification

```text
GET /api/notifications
GET /api/notifications?status=belum_terbaca
GET /api/notifications?status=terbaca
```

Response:

```json
{
  "notifications": [
    {
      "id": 1,
      "role": "sales",
      "title": "Preorder Baru",
      "message": "Preorder #12 masuk untuk review sales",
      "data": "{\"id_preorder\":12,\"id_agent\":3,\"status\":\"in_review\"}",
      "read_at": null,
      "status": "belum_terbaca"
    }
  ]
}
```

### Detail Notification

```text
GET /api/notifications/:id
```

### Mark One As Read

```text
PUT /api/notifications/:id/read
```

### Mark All As Read

```text
PUT /api/notifications/read-all
```

## Super Admin Area

Semua endpoint di bagian ini wajib login sebagai `super_admin`.

### Dashboard

```text
GET /api/super-admin/dashboard
```

Response sementara:

```json
{
  "message": "super admin dashboard"
}
```

### List Semua Withdraw

```text
GET /api/super-admin/withdraws
GET /api/super-admin/withdraws?status=on_progress
```

### Approve Withdraw

```text
PUT /api/super-admin/withdraws/:id/approve
```

Efek:

- Status withdraw menjadi `approval`.
- `pending_withdraw` berkurang.
- `withdrawn_balance` bertambah.

## Onboarding Agent

Semua endpoint onboarding wajib login sebagai `agent`.

### List Video

```text
GET /api/agent/onboarding/videos
```

Response:

```json
{
  "videos": [
    {
      "id": 1,
      "slug": "agent-introduction",
      "title": "Pengenalan Role Agent",
      "description": "Dasar tugas agent",
      "video_url": "https://...",
      "duration_seconds": 380,
      "sort_order": 1,
      "is_required": true
    }
  ]
}
```

### Get Progress

```text
GET /api/agent/onboarding/progress
```

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
      "completed_at": "2026-06-12T02:30:00+07:00"
    }
  ]
}
```

### Save Progress

```text
POST /api/agent/onboarding/progress
```

Body:

```json
{
  "video_id": 1,
  "watched_seconds": 380,
  "duration_seconds": 380,
  "status": "completed"
}
```

Rule:

- Jika `watched_seconds >= duration_seconds * 0.9`, status otomatis menjadi `completed`.
- Tidak bisa complete video lanjutan sebelum video sebelumnya complete.
- Jika sudah completed, status tetap completed.

### Reset Progress

```text
DELETE /api/agent/onboarding/progress
```

## Articles (CMS)

Endpoint artikel untuk CMS dan scraping import. GET bersifat public, endpoint lain wajib login sebagai `super_admin` atau `marketing`.

Status artikel:

```text
draft
published
archived
```

### List Artikel

```text
GET /api/articles
GET /api/articles?search=lampu&status=draft&page=1&limit=20
GET /api/articles?category=Media
```

Query opsional:

| Parameter  | Deskripsi                                      | Default |
| ---------- | ---------------------------------------------- | ------- |
| `search`   | Cari berdasarkan title, excerpt, atau category | -       |
| `category` | Filter berdasarkan category artikel            | -       |
| `status`   | Filter berdasarkan status                      | -       |
| `page`     | Halaman                                        | 1       |
| `limit`    | Jumlah per halaman                             | 20      |

Response:

```json
{
  "articles": [
    {
      "id": 1,
      "title": "Judul artikel",
      "slug": "judul-artikel",
      "category": "Media",
      "excerpt": "Ringkasan artikel",
      "content": "<h1>Konten HTML</h1>",
      "featured_image": "https://url-gambar.com/image.jpg",
      "author": "Global Multipro Technology",
      "source_url": "https://website-lama.com/artikel/contoh",
      "status": "draft",
      "seo": {
        "title": "SEO title",
        "description": "SEO description",
        "canonical_url": "https://website-lama.com/artikel/contoh"
      },
      "published_at": "2026-06-08T03:36:40Z",
      "created_at": "2026-06-10T00:17:09Z",
      "updated_at": "2026-06-10T00:17:09Z"
    }
  ],
  "meta": {
    "total": 50,
    "page": 1,
    "limit": 20,
    "total_pages": 3
  }
}
```

### Detail Artikel

```text
GET /api/articles/:id
```

Parameter `:id` bisa berupa ID numerik atau slug artikel.

Response:

```json
{
  "article": {
    "id": 1,
    "title": "Judul artikel",
    "slug": "judul-artikel",
    "category": "Media",
    "excerpt": "Ringkasan artikel",
    "content": "<h1>Konten HTML lengkap</h1>",
    "featured_image": "https://url-gambar.com/image.jpg",
    "author": "Global Multipro Technology",
    "source_url": "https://website-lama.com/artikel/contoh",
    "status": "draft",
    "seo": {
      "title": "SEO title",
      "description": "SEO description",
      "canonical_url": "https://website-lama.com/artikel/contoh"
    },
    "published_at": "2026-06-08T03:36:40Z",
    "created_at": "2026-06-10T00:17:09Z",
    "updated_at": "2026-06-10T00:17:09Z"
  }
}
```

### Create Artikel

```text
POST /api/articles
```

Auth: wajib login sebagai `super_admin` atau `marketing`.

Body:

```json
{
    "title": "Judul artikel",
    "slug": "judul-artikel",
    "category": "Media",
    "excerpt": "Ringkasan artikel",
    "content": "<h1>Konten artikel HTML...</h1>",
    "featured_image": "https://url-gambar.com/image.jpg",
    "author": "Global Multipro Technology",
    "source_url": "https://website-lama.com/artikel/contoh",
    "status": "draft",
    "published_at": "2026-06-08T03:36:40.000Z",
    "updated_at": "2026-06-10T00:17:09.000Z",
    "seo": {
      "title": "SEO title",
      "description": "SEO description",
      "canonical_url": "https://website-lama.com/artikel/contoh"
    },
    "metadata": {
      "gallery": ["https://url.com/img1.jpg", "https://url.com/img2.jpg"],
      "related_products": [1, 2, 5],
      "related_articles": [10, 12]
    }
  }
```

Response sukses (201):

```json
{
  "message": "Article created",
  "article": {
    "id": 1,
    "title": "Judul artikel",
    "slug": "judul-artikel",
    "category": "Media",
    "status": "draft"
  }
}
```

Jika slug sudah ada, response 409:

```json
{
  "message": "slug already exists"
}
```

### Update Artikel

```text
PUT /api/articles/:id
```

Auth: wajib login sebagai `super_admin` atau `marketing`.

Body sama seperti create.

### Delete Artikel

```text
DELETE /api/articles/:id
```

Auth: wajib login sebagai `super_admin` atau `marketing`.

Response:

```json
{
  "message": "Article deleted"
}
```

### Bulk Import Artikel

```text
POST /api/articles/import
```

Auth: wajib login sebagai `super_admin` atau `marketing`.

Cocok untuk import banyak artikel sekaligus dari scraper.

Body:

```json
{
  "articles": [
    {
      "title": "Judul artikel 1",
      "slug": "judul-artikel-1",
      "category": "Media",
      "excerpt": "Ringkasan",
      "content": "<h1>Konten HTML</h1>",
      "featured_image": "https://...",
      "author": "Global Multipro Technology",
      "source_url": "https://...",
      "status": "draft",
      "published_at": "2026-06-08T03:36:40.000Z",
      "updated_at": "2026-06-10T00:17:09.000Z",
      "seo": {
        "title": "SEO title",
        "description": "SEO description",
        "canonical_url": "https://..."
      }
    }
  ]
}
```

Response (201):

```json
{
  "message": "Import completed",
  "created_count": 5,
  "skipped_count": 2,
  "created": [
    {
      "id": 1,
      "title": "Judul artikel 1",
      "slug": "judul-artikel-1",
      "category": "Media",
      "status": "draft"
    }
  ],
  "skipped_slugs": ["judul-sudah-ada", "judul-duplikat"]
}
```

Catatan: Artikel dengan slug yang sudah ada di database akan di-skip, bukan error.

## Health Check

```text
GET /health
```

Response:

```json
{
  "status": "ok"
}
```

## Error Penting Untuk Frontend

### Unauthorized

Status:

```text
401 Unauthorized
```

Contoh response:

```json
{
  "message": "session expired or revoked"
}
```

Tindakan frontend:

```text
Hapus token lokal dan redirect ke login.
```

### Forbidden Role

Status:

```text
403 Forbidden
```

Contoh response:

```json
{
  "message": "you do not have access to this resource"
}
```

### Conflict State

Status:

```text
409 Conflict
```

Biasanya terjadi saat:

- Update/delete/submit PO yang statusnya bukan `draft`.
- Approve/invalid PO yang statusnya bukan `in_review`.
- Withdraw sudah diproses.

## Checklist Implementasi Website Pusat

- Simpan token login Website Pusat sebagai session lokal.
- Panggil `GET /api/auth/session` saat aplikasi dibuka.
- Siapkan halaman `/sso/callback` untuk menerima user dari Website A.
- Di callback SSO, panggil `POST /api/auth/sso/exchange` dengan `target_client = website_utama`.
- Redirect user ke dashboard sesuai role setelah login/session valid.
- Untuk logout, panggil `POST /api/auth/logout`, lalu hapus token lokal.
- Gunakan role dari response user untuk membatasi menu frontend.
- Sales page perlu handle SSE `/api/sales/notifications/stream`.
- Agent page perlu handle status PO dan wallet setelah PO approve.
- Super admin page perlu handle approval withdraw.
