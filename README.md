# 🤖 Bot Trade DzikriJr (AI Quant & Weekly Momentum)

**DzikriJr** adalah bot Telegram asisten trading saham cerdas yang dibangun menggunakan **Golang** dan ditenagai oleh **Google Gemini AI**. 

Bot ini mengeksekusi strategi **Weekly Momentum Rotation** dengan DCA bulanan Rp 1 juta — strategi kuantitatif yang telah terbukti melalui backtest 5 tahun dengan **expectancy +1.34%** dan **win rate 63.9%**. DzikriJr dirancang sebagai "Co-Pilot" untuk swing trading mingguan (1-4 minggu holding) yang fokus pada perlindungan modal dan rotasi sistematis, sementara keputusan eksekusi final tetap berada di tangan pengguna ("The Human CEO").

---

## 🌟 Fitur Unggulan Terkini (Advanced)

### 🧠 Momentum Weekly Algorithm
* **Weekly Rotation Signal:** Setiap Senin, bot memindai 29 saham watchlist untuk menemukan saham dalam uptrend kuat (MA5 > MA20 > MA50) dengan momentum terbaik.
* **Top 3-4 Selection:** Bot memilih 3-4 saham dengan momentum tertinggi untuk dirotasi ke dalam portofolio dengan equal weight.
* **Smart Rebalancing:** Posisi yang trendnya patah (MA5 < MA20) otomatis keluar pada rebalance mingguan berikutnya.

### 🛡️ Pertahanan Modal (Capital Protection)
* **Fixed Cut Loss:** Proteksi ketat di -3% untuk membatasi kerugian maksimal per posisi.
* **Take Profit:** Target +5% untuk mengunci profit secara sistematis.
* **Time Stop:** Maksimal holding 20 hari bursa untuk menghindari posisi yang macet terlalu lama.
* **Dynamic Position Sizing:** Menghitung otomatis alokasi modal per saham dengan membatasi risiko maksimal per transaksi.

### 💼 Advanced Portfolio Management
* **DCA Monthly Integration:** Sistem menerima injeksi modal Rp 1 juta setiap bulan dan otomatis mengalokasikannya ke posisi aktif.
* **Max 4 Positions:** Diversifikasi terkontrol untuk menghindari over-exposure.
* **Partial Sell (Take Profit Parsial):** Mendukung penjualan sebagian lot dengan pencatatan akurat pada `trade_history.csv` dan kalkulasi Realized PNL yang dinamis.
* **Smart Monitoring:** Memantau portofolio secara background dan memberikan alert saat posisi mencapai TP/CL.

### 📊 Laporan Rutin Otomatis (Cron Jobs)
* 🕘 **08:45 WIB** → Evaluasi portofolio pagi oleh AI sebelum market buka (Evaluasi TSL & Cut Loss).
* 🕛 **12:20 WIB** → Laporan Makan Siang (Lunch Summary PNL).
* 🕛 **12:30 WIB** → Evaluasi portofolio siang (Evaluasi TSL & Cut Loss lanjutan).
* 📅 **Senin 18:05 WIB** → **THE GOLDEN DAY:** Eksekusi otomatis algoritma momentum untuk rebalance mingguan. Bot mengirim rekomendasi 3-4 saham untuk dibeli Senin pagi berikutnya.
* 🕓 **16:20 WIB** → Laporan penutupan pasar harian.

---

## 📈 Strategi Utama: Weekly Momentum Rotation

Bot beroperasi dengan parameter yang telah divalidasi melalui backtest 5 tahun (2021-2026):

**Entry Logic:**
1. **Timing:** Setiap Senin pagi (09:00-09:30) setelah menerima rekomendasi Jumat sore.
2. **Filter:** Hanya saham dalam uptrend (MA5 > MA20 > MA50).
3. **Selection:** Top 3-4 saham dengan momentum tertinggi (sorted by distance from MA5).
4. **Allocation:** Equal weight distribution dengan max 4 posisi aktif.

**Exit Rules:**
1. **Cut Loss Struktural:** -3% fixed untuk proteksi modal.
2. **Take Profit:** +5% untuk lock profit secara sistematis.
3. **Trend Break:** Exit saat MA5 < MA20 pada rebalance mingguan.
4. **Time Stop:** Maksimal 20 hari bursa untuk menghindari dead money.

**Backtest Results (OOS 2024-2026):**
- Total trades: 2,266
- Win rate: 63.9%
- Expectancy: +1.34%
- Max Drawdown: 23.43%
- Average holding: 1.6 hari bursa
- Profit Factor: 2.45

---

## 💬 Perintah Telegram (Commands)

| Perintah | Format / Contoh | Deskripsi |
|---|---|---|
| `/recommend` | `/recommend` | Menjalankan algoritma momentum untuk men-scan watchlist (biasanya otomatis Jumat sore). |
| `/research` | `/research MTEL` | Analisis mendalam AI + kalkulator position sizing untuk saham spesifik. |
| `/buy` | `/buy BBCA 9000 10` | Memasukkan saham ke portofolio aktif (atau averaging down jika sudah ada). |
| `/sell` | `/sell BBCA 9500 5` | Mencatat penjualan saham. Bisa jual parsial (isi Lot) atau jual semua (kosongkan Lot). |
| `/antre` | `/antre TOWR 480 10` | Memasang jaring/antrean. Akan masuk portofolio otomatis jika harga menyentuh. |
| `/cancel_antre`| `/cancel_antre TOWR`| Mencabut pantauan antrean dari sistem. |
| `/status` | `/status` | Menampilkan detail portofolio, Floating PNL, batas TSL/CL, dan status antrean aktif. |
| `/evaluate` | `/evaluate` | AI memberikan saran Hold/Cut/TP atas seluruh portofolio berdasarkan kondisi market terbaru. |
| `/stats` | `/stats` | Menampilkan statistik trading: win rate, expectancy, profit factor dari history aktual. |
| `/reset` | `/reset` | Mengosongkan paksa database portofolio. |

---

## ⚠️ SOP Emas "The Human CEO" (Wajib Ditaati)

DzikriJr menggunakan data EOD (End of Day) Yahoo Finance yang buta terhadap **Foreign Flow** dan **Broker Summary**. Anda sebagai CEO **wajib** menambal titik buta ini dengan SOP eksekusi berikut:

### 📅 Weekly Rotation Flow:

1. **Jumat 18:05:** Bot mengirimkan 3-4 saham momentum terbaik via Telegram.
2. **Weekend (Sabtu-Minggu):** Analisis santai. Cek chart saham rekomendasi, baca news, validasi trend masih sehat.
3. **Senin Pagi 09:00-09:30 (The Human Veto):** 
   * Buka aplikasi sekuritas Anda (Bibit/Stockbit/Ajaib).
   * Validasi dua hal:
     - **Live Chart:** Pastikan trend masih sehat (MA5 masih di atas MA20).
     - **Foreign Flow:** Cek data **F Buy vs F Sell**. Jika asing mendistribusikan barang besar-besaran (F Sell >> F Buy), **SKIP SAHAM ITU**.
   * Eksekusi: Jika lolos validasi → **Beli di market price**.
4. **Pasang Sabuk:** Setelah beli, segera pasang **Auto-Order** (TP +5% dan CL -3%) di aplikasi sekuritas. Tutup aplikasi dan fokus bekerja.
5. **Holding Period:** Biarkan posisi berjalan 1-4 minggu. Bot akan evaluasi setiap pagi (08:45) dan memberikan alert jika mendekati TP/CL.
6. **Next Monday:** Ulangi proses. Bot akan memberikan rekomendasi rotasi baru setiap Jumat sore.

### 🎯 Aturan Emas:
* **Senin pagi (09:00-09:30) adalah waktu entry.** Tidak ada entry di hari lain kecuali emergency.
* **Jumat sore adalah waktu analisis.** Tidak ada FOMO buy di tengah minggu.
* **Jam istirahat siang (12:00-13:30) khusus untuk makan dan ngopi,** bukan untuk buka aplikasi sekuritas.
* **DCA 1 juta setiap bulan.** Konsisten adalah kunci.

---

## 🚀 Instalasi & Konfigurasi

1. Pastikan Anda menggunakan **Go 1.26+**.
2. Clone repositori ini dan install dependencies:
   ```bash
   go mod tidy
   ```
3. Copy `.env.example` ke `.env` dan isi konfigurasi:
   ```bash
   cp .env.example .env
   ```
   Isi dengan:
   * `BOT_TOKEN` — Token dari BotFather Telegram.
   * `MY_CHAT_ID` — ID Telegram pribadi Anda (agar bot bersifat private).
   * `GEMINI_API_KEY` — API Key gratis dari Google AI Studio.

4. Jalankan bot:
   ```bash
   go run cmd/bot/main.go
   ```

5. (Opsional) Jalankan backtest untuk validasi:
   ```bash
   go run cmd/backtest/main.go --strategy momentum-weekly --years 5 --capital 1000000 --dca-monthly 1000000 --out backtest_out_momentum
   ```

---

## 🧪 Testing & Validation

Bot ini dilengkapi dengan sistem backtest yang komprehensif:

```bash
# Run full backtest suite
go test ./...

# Run backtest with momentum strategy
go run ./cmd/backtest --strategy momentum-weekly --years 5 --capital 1000000 --dca-monthly 1000000 --out backtest_out_momentum
```

Output backtest:
- `trades.csv` — Log semua transaksi
- `equity.csv` — Kurva ekuitas harian
- `report.md` — Laporan lengkap dengan metrik (expectancy, win rate, profit factor, max DD)
- `compare.md` — Perbandingan dengan baseline (random entry, DCA naif)

---

## 📚 Dokumentasi Lanjutan

- **[Cara Pakai](cara-pakai.md)** — Panduan lengkap penggunaan bot sehari-hari
- **[Architecture](architecture.md)** — Struktur kode dan design decisions
- **[CLAUDE.md](CLAUDE.md)** — Spesifikasi teknis dan backlog development

---

## ⚖️ Disclaimer

**RISK WARNING:** Trading saham memiliki risiko tinggi. Backtest bukan jaminan profit di masa depan. Bot ini adalah alat bantu, bukan holy grail. Selalu lakukan due diligence, manage risk, dan jangan investasi lebih dari yang Anda mampu untuk kehilangan.

Data backtest menggunakan Yahoo Finance EOD yang **tidak include dividen** dan **tidak adjusted untuk survivorship bias**. Performa aktual bisa berbeda.

**Gunakan dengan bijak. Trade at your own risk.**

---

## 📄 License

MIT License — Feel free to modify and use for personal trading.

---

## 🙏 Credits

Built with ❤️ by Dzikri using:
- [Go Telegram Bot API](https://github.com/go-telegram-bot-api/telegram-bot-api)
- [Google Gemini AI](https://github.com/google/generative-ai-go)
- [Robfig Cron](https://github.com/robfig/cron)

Special thanks to the Quant community for research and validation.
