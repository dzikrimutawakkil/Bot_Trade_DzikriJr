# 🤖 Bot Trade DzikriJr (AI Quant & EOD Screener)

**DzikriJr** adalah bot Telegram asisten *trading* saham cerdas yang dibangun menggunakan **Golang** dan ditenagai oleh **Google Gemini AI**. 

Berevolusi dari bot pencatat biasa, DzikriJr kini menjadi **End-of-Day (EOD) Quant Screener** yang mengeksekusi strategi **Confirmed Buy on Weakness (C-BoW)**. Bot ini dirancang sebagai "Co-Pilot" untuk *Fast Swing Trading* (1-5 hari) yang fokus pada perlindungan modal, menyaring ratusan saham, dan memberikan *Trading Plan* matematis, sementara keputusan eksekusi final tetap berada di tangan pengguna ("The Human CEO").

---

## 🌟 Fitur Unggulan Terkini (Advanced)

### 🧠 The Hunter Algorithm & Gemini AI
* **C-BoW Technical Filter:** Pemindaian ketat menggunakan data *End-of-Day* (Yahoo Finance) untuk mencari pola *High-Quality Hammer* (ekor bawah 2x lipat body) dan *Strong Green Bounce* di area *support* MA20.
* **Deep Fundamental Synthesis:** Menggabungkan berita RSS (Google News) dengan data teknikal, lalu dikirim ke AI Gemini untuk menghasilkan skor sentimen, analisis probabilitas, dan *trading plan* yang logis.
* **Smart Rate-Limiting:** Membatasi pemanggilan API AI (maksimal 10x) dan berhenti otomatis setelah menemukan 3 setup terbaik untuk menghemat *resource*.

### 🛡️ Pertahanan Modal (Capital Protection)
* **Anti-Falling Knife (Anti-ARB):** Otomatis memblokir (skor 0) saham yang hari sebelumnya turun >10%, mencegah bot menangkap "pisau jatuh" di saham buangan bandar.
* **Anti-Distribution Trap:** Mencegah pembelian pada saham yang sudah terpompa >3.5% dari titik terendahnya di hari yang sama untuk menghindari jebakan *Exit Liquidity*.
* **Dynamic Position Sizing:** Menghitung otomatis maksimal lot yang boleh dibeli dengan membatasi risiko maksimal 1% dari total modal per transaksi.

### 💼 Advance Portfolio Management
* **Averaging Down (Scaling In):** Otomatis menghitung ulang harga rata-rata (*Average Price*) dan mereset batas Cut Loss/Trailing Stop saat dilakukan pembelian tambahan pada saham yang sama.
* **Partial Sell (Take Profit Parsial):** Mendukung penjualan sebagian lot (misal: TP1 jual 50% lot) dengan pencatatan akurat pada `trade_history.csv` dan kalkulasi *Realized PNL* yang dinamis.
* **Smart Monitoring & Auto-Match:** Memantau antrean pembelian secara *background*. Jika harga pasar menyentuh harga antrean, bot otomatis memindahkan saham ke portofolio aktif dan menyalakan radar *Trailing Stop*.

### 📊 Laporan Rutin Otomatis (Cron Jobs)
* 🕘 **08:45 WIB** → Evaluasi portofolio pagi oleh AI sebelum market buka (Evaluasi TSL & Cut Loss).
* 🕛 **12:20 WIB** → Laporan Makan Siang (Lunch Summary PNL).
* 🕛 **12:30 WIB** → Evaluasi portofolio pagi oleh AI sebelum market dilanjutkan (Evaluasi TSL & Cut Loss).
* 🕓 **15:40 WIB** → **THE GOLDEN HOUR:** Eksekusi otomatis algoritma pencari saham untuk strategi BOC (*Buy on Close*).
* 🕓 **16:20 WIB** → Laporan penutupan pasar.

---

## 📈 Strategi Utama: Confirmed BoW (C-BoW) & Hit and Run

Bot beroperasi dengan parameter perlindungan matematis:
1. **Entry:** *Buy on Close* (BOC) pada pukul 15:45 setelah tervalidasi AI.
2. **Cut Loss Struktural:** Diatur ketat pada 1% di bawah ujung ekor bawah (*Low*) dari *candlestick* konfirmasi.
3. **Take Profit (Hit & Run):**
   - **TP1 (Jual 50%):** Pada target +4% untuk mengamankan modal.
   - **TP2 (Let it Ride):** Sisa 50% dijaga menggunakan *Trailing Stop Loss* (TSL) sebesar 3% dari rekor harga tertinggi (Pucuk).

---

## 💬 Perintah Telegram (Commands)

| Perintah | Format / Contoh | Deskripsi |
|---|---|---|
| `/recommend` | `/recommend` | Menjalankan *Hunter Algorithm* untuk men-scan IHSG & Watchlist. |
| `/research` | `/research MTEL` | Analisis mendalam AI + Kalkulator *Position Sizing* untuk saham spesifik. |
| `/buy` | `/buy BBCA 9000 10` | Memasukkan saham ke portofolio aktif (atau *Averaging Down* jika sudah ada). |
| `/sell` | `/sell BBCA 9500 5` | Mencatat penjualan saham. Bisa jual parsial (isi Lot) atau jual semua (kosongkan Lot). |
| `/antre` | `/antre TOWR 480 10` | Memasang jaring/antrean. Akan masuk portofolio otomatis jika harga menyentuh. |
| `/cancel_antre`| `/cancel_antre TOWR`| Mencabut pantauan antrean dari sistem. |
| `/status` | `/status` | Menampilkan detail portofolio, Floating PNL, batas TSL/CL, dan status antrean aktif. |
| `/evaluate` | `/evaluate` | AI memberikan saran *Hold/Cut/TP* atas seluruh portofolio berdasarkan kondisi *market* terbaru. |
| `/reset` | `/reset` | Mengosongkan paksa database portofolio. |

---

## ⚠️ SOP Emas "The Human CEO" (Wajib Ditaati)

DzikriJr menggunakan data EOD (End of Day) Yahoo Finance yang buta terhadap **Foreign Flow** dan **Broker Summary**. Anda sebagai CEO **wajib** menambal titik buta ini dengan SOP eksekusi berikut:

1. **Jam 15:40:** Bot mengirimkan 1-3 saham *Confirmed BoW* via Telegram.
2. **Jam 15:45 (The Human Veto):** Buka aplikasi sekuritas Anda (Bibit/Stockbit/Ajaib). Validasi dua hal:
   * **Live Chart:** Pastikan bentuk *candle* MASIH *Hammer* atau bertahan di *support* MA20.
   * **Foreign Flow:** Cek data **F Buy vs F Sell**. Jika asing mendistribusikan barang besar-besaran (F Sell >> F Buy), **ABAIKAN REKOMENDASI BOT**.
3. **Eksekusi:** Jika chart masih bagus dan asing netral/akumulasi ➡️ **Hajar Kanan (BOC)**.
4. **Pasang Sabuk:** Keesokan paginya (08:50), masukkan angka *Cut Loss* dan *Take Profit* dari laporan bot ke fitur **Auto-Order** di aplikasi sekuritas. Tutup aplikasi dan fokus bekerja.

---

## 🚀 Instalasi & Konfigurasi

1. Pastikan Anda menggunakan **Go 1.26+**.
2. Clone repositori ini dan install dependencies:
  ```bash
   go mod tidy
  ```
3. Konfigurasikan token pada internal/config/utils.go atau .env:
* BotToken: Token dari BotFather Telegram.
* MyChatID: ID Telegram pribadi Anda (agar bot bersifat private).
* GeminiAPIKey: API Key gratis dari Google AI Studio.

4. Jalankan bot:
  ``` Bash
    go run cmd/bot/main.go
  ```