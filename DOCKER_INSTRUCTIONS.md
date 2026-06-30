# Panduan Build, Push, Pull, dan Run Docker

Dokumen ini menjelaskan langkah-langkah untuk membangun image Docker secara lokal, mengunggahnya ke Docker Hub, lalu mengunduh serta menjalankannya di server Anda.

---

## 1. Persiapan Awal
Sebelum memulai, pastikan Anda telah menginstal:
- **Docker** & **Docker Compose** di komputer lokal dan server.
- Akun **Docker Hub**.

### Hubungkan Docker Lokal ke Docker Hub Anda
Buka terminal (CMD, PowerShell, atau Terminal Bash) di komputer lokal Anda dan jalankan perintah login:
```bash
docker login
```
Masukkan *username* dan *password* (atau *Personal Access Token*) Docker Hub Anda.

---

## 2. Build Secara Lokal
Anda dapat melakukan build menggunakan perintah `docker compose build` atau perintah manual `docker build`.

### Opsi A: Menggunakan Docker Compose (Direkomendasikan)
1. Buka file `docker-compose.yml` di komputer lokal Anda.
2. Ubah baris `image: zaldygmtgroup/begmt2:latest` dan ganti `zaldygmtgroup` dengan **username Docker Hub Anda**.
3. Jalankan perintah berikut di direktori root project untuk membangun image:
   ```bash
   docker compose build
   ```

### Opsi B: Menggunakan Perintah Docker CLI Manual
Jika Anda ingin melakukan build tanpa docker compose secara lokal:
```bash
docker build -t <username-dockerhub>/begmt2:latest .
```
*(Ganti `<username-dockerhub>` dengan username Docker Hub Anda)*

---

## 3. Push Image ke Docker Hub
Setelah proses build selesai, unggah image tersebut ke Docker Hub agar dapat diakses dari server.

### Opsi A: Menggunakan Docker Compose (Direkomendasikan)
```bash
docker compose push
```

### Opsi B: Menggunakan Perintah Docker CLI Manual
```bash
docker push <username-dockerhub>/begmt2:latest
```

---

## 4. Deploy & Run di Server
Untuk menjalankan aplikasi di server, Anda hanya membutuhkan **dua file**:
1. `docker-compose.yml`
2. `.env` (berisi konfigurasi environment server Anda)

### Langkah-langkah Deployment:
1. **Salin File ke Server**:
   Salin file `docker-compose.yml` dan file `.env` dari lokal ke sebuah direktori di server Anda (misalnya `/opt/begmt2`). Anda dapat menggunakan `scp`, `sftp`, `rsync`, atau copy-paste langsung di server.

2. **Sesuaikan File `.env` di Server**:
   Pastikan konfigurasi database dan variable lainnya di server telah disesuaikan (seperti port, email credentials, JWT secret, dll).

3. **Login Docker Hub di Server** (Jika repository Anda bersifat *Private*):
   ```bash
   docker login
   ```

4. **Pull Image Terbaru**:
   Jalankan perintah berikut di direktori tempat file `docker-compose.yml` berada di server:
   ```bash
   docker compose pull
   ```

5. **Jalankan Containers (Run)**:
   Mulai jalankan container secara *background* (detached mode):
   ```bash
   docker compose up -d
   ```

6. **Periksa Status**:
   Untuk melihat apakah container berjalan dengan baik:
   ```bash
   docker compose ps
   ```
   Untuk melihat log aplikasi secara langsung:
   ```bash
   docker compose logs -f app
   ```

---

## 5. Tips Tambahan
* **Persistensi Data**: File yang diunggah oleh pengguna disimpan di folder `uploads`. Docker Compose secara otomatis memetakan folder ini ke volume bernama `begmt2_uploads_data` sehingga file tidak akan hilang ketika container di-restart atau di-update.
* **Update Aplikasi**: Di masa mendatang, jika Anda mengubah kode Go, Anda cukup menjalankan:
  1. Di Lokal: `docker compose build` lalu `docker compose push`
  2. Di Server: `docker compose pull` lalu `docker compose up -d`
