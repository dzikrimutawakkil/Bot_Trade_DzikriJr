# 🤖 Bot Trade DzikriJr (Asisten AI Fast Swing)

**DzikriJr** adalah bot Telegram asisten *trading* saham pintar yang dibangun menggunakan **Golang** dan ditenagai oleh **Google Gemini AI**.

Bot ini dirancang dengan arsitektur **Clean Code (Domain-Driven Design)** untuk mengeksekusi strategi **Fast Swing Trading (1–5 hari)** di Bursa Efek Indonesia (IHSG). Ia mencari peluang "Momentum dalam Tren Sehat" (harga sedang naik, momentum baru mulai, dan sedang istirahat di dekat titik pantul).

Selain memberikan rekomendasi teknikal, bot ini juga bertindak sebagai **manajer portofolio pribadi** yang membantu menjaga psikologi *trader* melalui analisis otomatis dan perlindungan modal.

---

## 🌟 Fitur Unggulan Terkini (Advanced)

### 🧠 Deep AI Research & "The Hunter Algorithm"
* **Scanning Pintar:** Menggunakan algoritma pemburu yang membatasi panggilan API AI maksimal 10 kali dan berhenti lebih awal jika sudah menemukan 3 setup "BELI" yang optimal.
* **Analisis Sinergi:** Menggabungkan analisis fundamental (berita RSS Google News) dan indikator teknikal (MA5, MA20, Volume Kering) yang disintesis oleh Gemini AI.

### 🛡️ Market Filter & Manajemen Risiko
* **IHSG Tracker:** Terus memantau arah pasar utama (^JKSE). Jika market sedang hancur (di bawah MA20), bot akan menyarankan "CASH IS KING" dan menolak memberikan rekomendasi beli agresif.
* **Position Sizing Calculator:** Menghitung otomatis maksimal LOT yang boleh dibeli berdasarkan batas risiko 1% dari total modal, memastikan kerugian tidak pernah menguras saldo.

### ⏳ Sistem Antrean Otomatis (Auto-Match)
* Memungkinkan kamu memasang jaring (limit order) via perintah `/antre`.
* **Smart Monitoring:** Pemantau harga akan otomatis me-*match* pesanan jika harga market turun menyentuh harga antreanmu, lalu memindahkannya ke portofolio aktif lengkap dengan Trailing Stop yang langsung menyala.
* **Runaway Price Alert:** Kalau harga saham malah kabur naik 3% meninggalkan antreanmu, bot akan menyuruhmu menarik antrean tersebut.

### 📊 Laporan Rutin Otomatis (Cron Jobs)
Bot mengirimkan laporan terjadwal:
* 🕘 **08:45 WIB** → Evaluasi portofolio pagi oleh AI sebelum market buka.
* 🕛 **12:20 WIB** → Rangkuman pergerakan market sesi 1 (Lunch Summary).
* 🕓 **16:20 WIB** → Laporan penutupan pasar.

---

## 🏗️ Struktur Proyek & Penyimpanan

Proyek ini menggunakan arsitektur berbasis fitur:
* **`cmd/bot/main.go`**: Titik masuk (entry point) dan inisialisasi rutin Cron.
* **Dual Storage (`storage`)**: Menyimpan histori riwayat trading dalam format `trade_history.csv` dan menjaga daftar portofolio aktif serta antrean dalam satu wadah di `stocks.json`.

---

## 📈 Strategi Trading: Buy on Weakness (BoW)

Sistem ini sangat disiplin dan anti-FOMO:
* **Target Utama:** Mencari *entry* di dekat area *support* MA20 saat tekanan jual ritel mulai habis (volume kering).
* **Risiko Terkunci:** Batas *Cut Loss* ditentukan secara *rigid*.

---

## 💬 Perintah Telegram (Commands)

| Perintah | Contoh Penggunaan | Deskripsi |
|---|---|---|
| `/recommend` | `/recommend` | Menjalankan *Hunter Algorithm* untuk men-scan market. |
| `/research` | `/research EXCL` | Analisis mendalam AI khusus untuk satu saham & posisi lot ideal. |
| `/antre` | `/antre TOWR 482 10` | Memasang jaring otomatis untuk saham yang sedang diincar. |
| `/cancel_antre`| `/cancel_antre TOWR`| Mencabut pantauan antrean dari memori bot. |
| `/buy` | `/buy BRMS 150 5` | Memasukkan kepemilikan saham langsung ke portofolio aktif. |
| `/sell` | `/sell BRMS 160` | Mencatat penjualan saham dan menyimpan log ke CSV. |
| `/status` | `/status` | Menampilkan PNL bersih portofolio saat ini & selisih harga antrean aktif. |
| `/evaluate` | `/evaluate` | Memaksa AI untuk mengevaluasi posisi (Hold/Sell) saham di portofolio. |
| `/reset` | `/reset` | Mengosongkan paksa seluruh database portofoliomu. |

---

## 🚀 Cara Menjalankan

1. Pastikan package terinstal:
  ```bash
   go mod tidy
  ```
2. Setup token Telegram dan Gemini di file .env (atau langsung di config.go).

3. Jalankan:
  ```
  Bash
  go run cmd/bot/main.go
  ```

⚠️ Disclaimer
Bot ini murni alat bantu analisis komputasi, bukan penasihat keuangan. Segala kerugian dan risiko trading adalah tanggung jawab pengguna. Harap selalu jaga money management!