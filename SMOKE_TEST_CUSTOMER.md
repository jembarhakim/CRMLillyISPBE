# Smoke Test End-to-End — Portal Pelanggan

Status terakhir dari validasi: backend build sukses & runtime aktif, tapi
**login customer tidak bisa di-smoke-test** karena (a) seeder tidak memberi
password ke customer, dan (b) MySQL + BE tidak hidup pada saat pengecekan.
Artikel ini memperbaiki (a) dan memberikan skrip uji yang siap jalan untuk (b).

## 0. Prasyarat

1. MySQL hidup:
   - `MYSQL_HOST=localhost`, `MYSQL_PORT=3306`, `MYSQL_USER=root`,
     `MYSQL_PASSWORD=` (kosong), `MYSQL_DATABASE=iqgncnzy_skripsi`
   - Pastikan database ada: `CREATE DATABASE IF NOT EXISTS iqgncnzy_skripsi;`
2. Jalankan seeder (sekarang customer diberi password `password`):
   ```bash
   cd C:\Dev\Magang\CRMLillyISPBE
   go run database-seeder.go
   ```
3. Jalankan backend:
   ```bash
   go run cmd/myapp/main.go
   ```
   (atau `start-server.bat`)

### Akun seeder yang tersedia

| Jenis | Login | Sandi |
|---|---|---|
| Staff (admin) | `admin@lillyisp.com` | `password` |
| Customer retail (smoke test) | `081234567899` | `password` |
| Customer PT Maju Bersama | `021-5550123` | `password` |

> Login-customer memakai kolom `phone` (bukan email). OTP WhatsApp memakai
> nomor yang sama; tanpa `WA_OTP_PROVIDER` kode hanya ditulis ke log BE.

## 1. Smoke test login customer (curl)

```powershell
curl.exe -s -X POST http://localhost:3001/api/auth/login-customer ^
  -H "Content-Type: application/json" ^
  -d '{"phone":"081234567899","password":"password"}'
```

Harapan (success=true):
```json
{ "success": true, "data": { "token": "eyJ...", "user": { "id": "...", "name": "Budi Santoso" } } }
```

### Bila gagal
| Response | Arti | Aksi |
|---|---|---|
| `record not found` | DB tidak punya customer itu | jalankan seeder |
| Network error | BE mati | `go run cmd/myapp/main.go` |
| `password salah` | sandi bukan `password` | update `customer.password` |

## 2. Smoke test create ticket customer

```powershell
# Ambil token dari langkah 1 (tempel di $token)
$token = "eyJ..."

curl.exe -s -X POST http://localhost:3001/api/customer/tickets ^
  -H "Content-Type: application/json" ^
  -H "Authorization: Bearer $token" ^
  -d '{"title":"Internet mati sejak pagi","description":"Tidak bisa buka situs sama sekali","trouble_type_id":"internet"}'
```

Harapan: `success: true`, `data.id` terisi, `stage: "received"`,
`stage_label: "Tiket Diterima"`.

## 3. Skenario lengkap

| # | Langkah | Endpoint | Harapan |
|---|---|---|---|
| 1 | Login customer | `POST /api/auth/login-customer` | token CUSTOMER |
| 2 | Dashboard customer | `GET /api/customer/dashboard` | status 200 |
| 3 | Buat tiket | `POST /api/customer/tickets` | tiket dibuat |
| 4 | Cek token customer diadmin | `GET /api/admin/company` | **403** (guard bekerja) |
| 5 | Login staff | `POST /api/auth/login` | token staff |
| 6 | Cek token staff di admin | `GET /api/admin/company` | status 200 |

## 4. WhatsApp OTP & Midtrans (butuh env asli)

- OTP: set `WA_OTP_PROVIDER` + `FONNTE_TOKEN` (atau `WA_SERVICE_URL`).
  Tanpa ini, kode hanya muncul di log BE.
- Midtrans: set `NUXT_PUBLIC_MIDTRANS_CLIENT_KEY` (FE) — server key tetap
  di BE (`internal/api/common/midtrands/midtrans.go`). Sandbox default sudah
  dipakai (`app.sandbox.midtrans.com`).

## 5. Perintah cepat

```powershell
# jalankan dari C:\Dev\Magang\CRMLillyISPBE
go run database-seeder.go
go run cmd/myapp/main.go
```
