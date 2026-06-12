# Dokumentasi API Integrasi Website A

Dokumen ini untuk tim Website A agar bisa integrasi login, SSO ke Website Utama, cek session, dan logout menggunakan backend BeGMT2.

Base URL API:

```text
https://api-domain-anda.com
```

Untuk development lokal:

```text
http://localhost:8080
```

Semua endpoint yang butuh login wajib mengirim header:

```text
Authorization: Bearer <token>
```

## Konsep Flow

Website A login ke backend BeGMT2. Jika login berhasil, Website A menyimpan token di sisi Website A. Saat user perlu masuk ke Website Utama tanpa login ulang, Website A meminta one-time SSO code ke backend, lalu redirect user ke URL Website Utama yang diberikan backend.

Token JWT jangan dikirim ke Website Utama lewat URL. Yang dikirim lewat URL hanya `code` sekali pakai.

## 1. Login Dari Website A

Endpoint:

```text
POST /api/auth/login
```

Body:

```json
{
  "email": "user@example.com",
  "password": "password123",
  "client": "website_a"
}
```

Response sukses:

```json
{
  "message": "login successful",
  "token": "jwt-token",
  "session": {
    "id": 1,
    "session_id": "session-id",
    "user_id": 10,
    "client": "website_a",
    "expires_at": "2026-06-13T03:00:00+07:00",
    "revoked_at": null,
    "created_at": "2026-06-12T03:00:00+07:00",
    "updated_at": "2026-06-12T03:00:00+07:00"
  },
  "user": {
    "id": 10,
    "name": "User Name",
    "email": "user@example.com",
    "role": "user"
  }
}
```

Yang perlu disimpan Website A:

```text
token
```

Rekomendasi penyimpanan:

```text
HttpOnly cookie milik domain Website A
```

Jika belum memungkinkan, boleh disimpan sesuai mekanisme auth yang sudah ada di Website A, tetapi jangan kirim token ini ke Website Utama lewat URL.

## 2. Cek Session Login

Endpoint:

```text
GET /api/auth/session
```

Header:

```text
Authorization: Bearer <token>
```

Response jika session masih aktif:

```json
{
  "authenticated": true,
  "session_id": "session-id",
  "user": {
    "id": 10,
    "name": "User Name",
    "email": "user@example.com",
    "role": "user"
  }
}
```

Response jika token invalid, expired, atau user sudah logout dari website lain:

```json
{
  "message": "session expired or revoked"
}
```

HTTP status:

```text
401 Unauthorized
```

Website A sebaiknya memanggil endpoint ini saat aplikasi dibuka atau saat restore session.

## 3. Redirect User Ke Website Utama Tanpa Login Ulang

Endpoint:

```text
POST /api/auth/sso/code
```

Header:

```text
Authorization: Bearer <token>
```

Body:

```json
{
  "target_client": "website_utama",
  "state": "random-state-dari-website-a"
}
```

Keterangan:

```text
target_client wajib bernilai website_utama
state opsional, tetapi direkomendasikan untuk validasi request redirect
```

Response sukses:

```json
{
  "code": "one-time-code",
  "expires_at": "2026-06-12T03:01:00+07:00",
  "redirect_url": "https://backstage-gmt-group.vercel.app/sso/callback?code=one-time-code&state=random-state-dari-website-a"
}
```

Yang perlu dilakukan Website A:

```js
window.location.href = response.redirect_url
```

Catatan penting:

```text
code hanya berlaku singkat, default 60 detik
code hanya bisa dipakai sekali
JWT Website A tidak boleh dikirim ke Website Utama
```

## 4. Logout

Endpoint:

```text
POST /api/auth/logout
```

Header:

```text
Authorization: Bearer <token>
```

Response sukses:

```json
{
  "message": "logout successful"
}
```

Efek logout:

```text
Semua session aktif user di backend akan dicabut.
Jika user sedang login di Website Utama, Website Utama akan ikut dianggap logout saat cek session atau memanggil endpoint protected.
Website A tetap harus menghapus token/session lokalnya sendiri.
```

Langkah logout di Website A:

```text
1. Panggil POST /api/auth/logout
2. Hapus token/session lokal Website A
3. Redirect user ke halaman login Website A
```

## 5. Jika User Datang Dari Website Utama Ke Website A

Jika nanti Website Utama juga ingin mengirim user ke Website A tanpa login ulang, Website A perlu menyediakan halaman callback, misalnya:

```text
https://gmtgroup2.vercel.app/sso/callback
```

Di halaman callback, Website A membaca query:

```text
code
state
```

Lalu Website A menukar code ke backend:

```text
POST /api/auth/sso/exchange
```

Body:

```json
{
  "code": "one-time-code-dari-url",
  "target_client": "website_a"
}
```

Response sukses:

```json
{
  "message": "sso exchange successful",
  "token": "jwt-token-untuk-website-a",
  "session": {
    "session_id": "session-id",
    "client": "website_a"
  },
  "user": {
    "id": 10,
    "name": "User Name",
    "email": "user@example.com",
    "role": "user"
  }
}
```

Setelah sukses:

```text
1. Simpan token sebagai session Website A
2. Redirect user ke halaman utama Website A
```

## Contoh Integrasi JavaScript

### Login

```js
async function loginWebsiteA(email, password) {
  const response = await fetch("https://api-domain-anda.com/api/auth/login", {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify({
      email,
      password,
      client: "website_a"
    })
  });

  if (!response.ok) {
    throw new Error("Login gagal");
  }

  return response.json();
}
```

### Buat SSO Redirect Ke Website Utama

```js
async function redirectToWebsiteUtama(token) {
  const state = crypto.randomUUID();

  const response = await fetch("https://api-domain-anda.com/api/auth/sso/code", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "Authorization": `Bearer ${token}`
    },
    body: JSON.stringify({
      target_client: "website_utama",
      state
    })
  });

  if (!response.ok) {
    throw new Error("Gagal membuat SSO code");
  }

  const data = await response.json();
  window.location.href = data.redirect_url;
}
```

### Cek Session

```js
async function checkSession(token) {
  const response = await fetch("https://api-domain-anda.com/api/auth/session", {
    headers: {
      "Authorization": `Bearer ${token}`
    }
  });

  if (response.status === 401) {
    return null;
  }

  if (!response.ok) {
    throw new Error("Gagal cek session");
  }

  return response.json();
}
```

### Logout

```js
async function logout(token) {
  await fetch("https://api-domain-anda.com/api/auth/logout", {
    method: "POST",
    headers: {
      "Authorization": `Bearer ${token}`
    }
  });

  // Hapus token/session lokal Website A setelah request ini.
}
```

## Error Yang Perlu Ditangani

### Login gagal

Status:

```text
401 Unauthorized
```

Response:

```json
{
  "message": "email or password is incorrect"
}
```

### Token expired atau sudah logout global

Status:

```text
401 Unauthorized
```

Response:

```json
{
  "message": "session expired or revoked"
}
```

### Target client SSO salah

Status:

```text
400 Bad Request
```

Response:

```json
{
  "message": "target client or redirect uri is not allowed"
}
```

### SSO code expired atau sudah dipakai

Status:

```text
400 Bad Request
```

Response:

```json
{
  "message": "invalid or expired sso code"
}
```

## Checklist Untuk Tim Website A

- Simpan token hasil login sebagai session Website A.
- Panggil `GET /api/auth/session` saat aplikasi dibuka.
- Untuk masuk ke Website Utama, panggil `POST /api/auth/sso/code`, lalu redirect ke `redirect_url`.
- Jangan kirim JWT lewat URL.
- Saat logout, panggil `POST /api/auth/logout`, lalu hapus session lokal Website A.
- Jika menerima user dari Website Utama, buat halaman `/sso/callback` dan panggil `POST /api/auth/sso/exchange`.
