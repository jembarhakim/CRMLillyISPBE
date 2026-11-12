# The Dude status monitoring

CRM dapat mengisi status perangkat melalui dua cara: sinkronisasi otomatis RouterOS API (utama) atau webhook status The Dude (opsional). Sinkronisasi API tidak memerlukan server The Dude mengakses laptop/backend.

Server pada repository ini secara default berjalan di port `3001` (`cmd/myapp/main.go`). Ganti port pada URL contoh di bawah jika deployment Anda menggunakan port atau domain lain.

## Payload

```json
{
  "device_name": "Router Kantor",
  "ip_address": "192.168.1.1",
  "status": "up"
}
```

Status perangkat disimpan pada tabel `dude_device_statuses`; data dengan IP yang sama akan memperbarui nama, status, dan waktu terakhir terlihat. Waktu down dicatat saat status down/offline teramati oleh ping, webhook, atau flag DOWN saat impor CSV. Untuk data lama yang belum memiliki waktu down, aplikasi mengisi `last_down_at` dari `last_seen`; waktu kejadian sebelum tercatat tidak dapat dipulihkan. Tabel dibuat oleh proses `AutoMigrate` aplikasi. WebSocket mengirim JSON payload di atas kepada seluruh client yang terhubung.

## Import network map and automatic sync

Export the desired network map from The Dude as CSV and import it from the CRM **Monitoring Jaringan** page. The CSV must contain `Name`, `Addresses`, and `Maps` columns; `Flag` is used as the initial status when present. Select a map during preview. Only valid IP addresses in that map are imported. Re-importing updates the existing device matched by IP and preserves CRM contact/location fields.

The backend starts a RouterOS API status synchronization loop automatically. It checks imported devices through the connected router's `/ping` command every 60 seconds; the first check runs shortly after startup. `/api/admin/dude-devices/sync` reports its status, and an admin can POST to the same route to start a check immediately. A probe/API/database failure is counted as failed and does not change the stored status to `down`. The RouterOS API account needs the `api`, `read`, `dude`, and `test` policies. The router must be able to route to the imported device networks. Devices that block ICMP may require The Dude webhook method instead.

CSV endpoints (admin token required):

- `POST /api/admin/dude-devices/import/preview` — multipart field `file`; returns map names and valid-IP counts.
- `POST /api/admin/dude-devices/import` — multipart fields `file` and `map_name`; upserts the selected map's devices by IP.

Files are limited to 5 MiB. The CSV importer accepts MikroTik/The Dude export headers named `Name`, `Addresses`, `Maps`, and optional `Flag`.

## Optional status webhook

Webhook status remains available at `POST /api/webhook/dude-status` and broadcasts updates to WebSocket `GET /api/webhook/dude-status/ws`. Use it for devices whose status cannot be determined by the RouterOS ping check.

Setelah perangkat mengirim notifikasi status untuk pertama kalinya, perangkat muncul pada bagian **The Dude Routers** di peta dashboard. Pilih perangkat dari daftar untuk mengisi nama pengguna/pemilik, nomor telepon, alamat, catatan, dan titik lokasi pada peta. Data ini disimpan terpisah dari data pelanggan CRM; nama perangkat dan IP tetap mengikuti update The Dude.

API dashboard yang digunakan:

- `GET /api/admin/dude-devices` — daftar router yang sudah menerima webhook.
- `PUT /api/admin/dude-devices/:id` — simpan informasi pengguna dan koordinat lokasi; perlu token admin/customer service.
- `POST /api/admin/dude-devices/ping-all` — mulai pemeriksaan ICMP paralel untuk IP yang sudah terdaftar; perlu token SUPERADMIN.
- `GET /api/admin/dude-devices/ping-all` — baca progres pemeriksaan ping yang sedang berjalan.

Ping-all berjalan di background dengan jumlah worker terbatas. Hasil yang berhasil diperiksa akan memperbarui status, waktu terakhir terlihat, dan dikirim ke WebSocket yang sama dengan webhook. IP yang tidak valid atau gagal diperiksa dilaporkan sebagai gagal, bukan dianggap `down`.

Ping-all hanya bisa memeriksa perangkat yang sudah memiliki IP di tabel `dude_device_statuses`. Tabel ini saat ini terisi dari webhook status; bila masih kosong, endpoint mengembalikan konflik dan tidak ada target yang bisa diping. Ping ICMP juga dapat diblokir oleh firewall atau membutuhkan konfigurasi jaringan yang sesuai pada server backend.

Router tanpa koordinat tetap terlihat di daftar sidebar, tetapi baru menjadi pin pada peta setelah titik lokasi dipilih dan disimpan.

### RouterOS Execute on Server

Tambahkan script berikut pada notification The Dude dengan tipe **Execute on Server**. Sesuaikan URL dan placeholder `%name`, `%address`, serta `%flag` dengan variabel yang tersedia pada versi The Dude Anda. Nilai `%flag` dikirim sebagai properti JSON `status` yang dibaca backend.

```routeros
:local deviceName "%name"
:local ipAddress "%address"
:local deviceStatus "%flag"
:local payload ("{\"device_name\":\"" . $deviceName . "\",\"ip_address\":\"" . $ipAddress . "\",\"status\":\"" . $deviceStatus . "\"}")

/tool fetch url="http://IP-ATAU-DOMAIN-BACKEND:3001/api/webhook/dude-status" http-method=post http-header-field="Content-Type: application/json" http-data=$payload keep-result=no
```

Pastikan server The Dude dapat menjangkau backend lewat jaringan/firewall. Endpoint webhook saat ini tidak memerlukan autentikasi; batasi aksesnya melalui firewall atau reverse proxy jika endpoint dapat diakses dari internet.

Untuk verifikasi awal, uji notifikasi pada satu perangkat yang aman dan pastikan request webhook diterima backend. Notifikasi perubahan status berikutnya akan memperbarui status tersimpan secara otomatis; jangan memutus perangkat produksi hanya untuk memicu pengujian. API RouterOS yang digunakan untuk daftar The Dude dapat saja hanya mengembalikan ID dan nama, sehingga status awal tidak bisa diisi dari daftar tersebut jika The Dude belum mengirim notifikasi.

## Contoh frontend

```js
const socket = new WebSocket("ws://localhost:3001/api/webhook/dude-status/ws");
socket.onmessage = ({ data }) => {
  const deviceStatus = JSON.parse(data);
  console.log(deviceStatus.device_name, deviceStatus.ip_address, deviceStatus.status);
};
```

Untuk koneksi lintas origin, CORS HTTP dan origin WebSocket mengizinkan frontend project ini pada `http://localhost:3000` dan `http://127.0.0.1:3000`, serta Vue dev server pada `http://localhost:5173` dan `http://127.0.0.1:5173`. Jika frontend memakai origin lain, tambahkan origin tersebut pada konfigurasi CORS di `cmd/myapp/main.go` dan allowlist WebSocket di `internal/api/webhook/dude_status/route.go`.
