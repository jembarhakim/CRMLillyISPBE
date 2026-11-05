# The Dude status webhook

Backend ini menerima perubahan status dari The Dude di `POST /api/webhook/dude-status` dan menyiarkan payload yang sama ke koneksi WebSocket `GET /api/webhook/dude-status/ws`.

Server pada repository ini secara default berjalan di port `3001` (`cmd/myapp/main.go`). Ganti port pada URL contoh di bawah jika deployment Anda menggunakan port atau domain lain.

## Payload

```json
{
  "device_name": "Router Kantor",
  "ip_address": "192.168.1.1",
  "status": "up"
}
```

Status perangkat disimpan pada tabel `dude_device_statuses`; data dengan IP yang sama akan memperbarui nama, status, dan waktu terakhir terlihat. Tabel dibuat oleh proses `AutoMigrate` aplikasi. WebSocket mengirim JSON payload di atas kepada seluruh client yang terhubung.

Webhook ini berbasis perubahan status, bukan polling. Jika perangkat tetap online atau offline, tidak perlu mengirim status berulang kali: status terakhir tetap tersimpan di database sampai ada notifikasi perubahan berikutnya. Perangkat yang belum pernah mengirim notifikasi akan belum memiliki status di CRM sampai notifikasi pertamanya diterima.

Setelah perangkat mengirim notifikasi status untuk pertama kalinya, perangkat muncul pada bagian **The Dude Routers** di peta dashboard. Pilih perangkat dari daftar untuk mengisi nama pengguna/pemilik, nomor telepon, alamat, catatan, dan titik lokasi pada peta. Data ini disimpan terpisah dari data pelanggan CRM; nama perangkat dan IP tetap mengikuti update The Dude.

API dashboard yang digunakan:

- `GET /api/admin/dude-devices` — daftar router yang sudah menerima webhook.
- `PUT /api/admin/dude-devices/:id` — simpan informasi pengguna dan koordinat lokasi; perlu token admin/customer service.
- `POST /api/admin/dude-devices/ping-all` — mulai pemeriksaan ICMP paralel untuk IP yang sudah terdaftar; perlu token SUPERADMIN.
- `GET /api/admin/dude-devices/ping-all` — baca progres pemeriksaan ping yang sedang berjalan.

Ping-all berjalan di background dengan jumlah worker terbatas. Hasil yang berhasil diperiksa akan memperbarui status, waktu terakhir terlihat, dan dikirim ke WebSocket yang sama dengan webhook. IP yang tidak valid atau gagal diperiksa dilaporkan sebagai gagal, bukan dianggap `down`.

Ping-all hanya bisa memeriksa perangkat yang sudah memiliki IP di tabel `dude_device_statuses`. Tabel ini saat ini terisi dari webhook status; bila masih kosong, endpoint mengembalikan konflik dan tidak ada target yang bisa diping. Ping ICMP juga dapat diblokir oleh firewall atau membutuhkan konfigurasi jaringan yang sesuai pada server backend.

Router tanpa koordinat tetap terlihat di daftar sidebar, tetapi baru menjadi pin pada peta setelah titik lokasi dipilih dan disimpan.

## RouterOS Execute on Server

Tambahkan script berikut pada notification The Dude dengan tipe **Execute on Server**. Sesuaikan URL, lalu sesuaikan placeholder `%name`, `%address`, dan `%status` bila versi The Dude Anda menggunakan variabel notifikasi yang berbeda.

```routeros
:local deviceName "%name"
:local ipAddress "%address"
:local deviceStatus "%status"
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
