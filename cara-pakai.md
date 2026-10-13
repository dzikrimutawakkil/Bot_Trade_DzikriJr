# 📖 Cara Pakai Bot DzikriJr — Weekly Momentum Strategy

## 🎯 Filosofi: Slow & Steady Wins the Race

DzikriJr menggunakan strategi **Weekly Momentum Rotation** yang telah divalidasi melalui backtest 5 tahun. Ini bukan bot scalping atau day trading. Ini adalah sistem swing trading mingguan yang:
- **Hold 1-4 minggu** per posisi
- **Rotasi setiap Senin** berdasarkan momentum
- **DCA 1 juta/bulan** untuk compound growth
- **Backtest proven:** Expectancy +1.34%, Win Rate 63.9%

Kuncinya adalah **disiplin mengikuti sistem** dan **tidak trading berdasarkan emosi**.

---

## 📅 Weekly Trading Cycle

### 🗓️ Jumat Sore (18:05) — Rekomendasi Bot
Bot otomatis mengirim rekomendasi 3-4 saham dengan momentum terbaik via Telegram.

**Yang Anda Lakukan:**
- Baca rekomendasi bot
- Catat daftar saham yang direkomendasikan
- **JANGAN LANGSUNG BELI**. Tunggu sampai Senin pagi.

### 📊 Weekend (Sabtu-Minggu) — Analisis Santai
Waktu untuk riset dan validasi tanpa tekanan market.

**Checklist Weekend:**
1. **Buka chart saham rekomendasi** di TradingView atau aplikasi sekuritas
2. **Cek trend visual:**
   - Apakah MA5 masih di atas MA20? ✅
   - Apakah chart masih bullish? ✅
3. **Baca news terkait** (Google News, IDX Channel)
4. **Cek fundamental dasar** (P/E ratio, debt ratio) jika mau lebih hati-hati
5. **Mental preparation:** Siapkan mental untuk entry Senin pagi

**Tips:** Jangan overthink. Bot sudah melakukan screening teknikal. Fokus Anda adalah validasi Foreign Flow dan trend integrity.

### 🚀 Senin Pagi (09:00-09:30) — Execution Time

**THE GOLDEN WINDOW:** Ini satu-satunya waktu entry dalam seminggu.

**Step-by-step Execution:**

1. **09:00 - Login ke aplikasi sekuritas** (Bibit/Stockbit/Ajaib)
2. **09:05 - Validasi Foreign Flow:**
   - Masuk ke menu **Broker Summary** atau **Net Foreign**
   - Cek saham rekomendasi bot
   - **Rule:** Jika **F Sell > F Buy** (asing dumping), **SKIP saham itu**
3. **09:10 - Tentukan alokasi:**
   - Bagi modal aktif ke 3-4 posisi (equal weight)
   - Contoh: Modal Rp 4 juta → Rp 1 juta per saham
4. **09:15 - Entry market price:**
   - Buy di harga market (jangan nunggu dip, ini momentum strategy)
   - Catat harga entry
5. **09:20 - Pasang Auto-Order:**
   - **Take Profit:** +5% dari harga entry
   - **Cut Loss:** -3% dari harga entry
6. **09:25 - Input ke bot:**
   - Ketik `/buy TICKER HARGA LOT` di Telegram
   - Contoh: `/buy BBRI 4500 22`
7. **09:30 - Tutup aplikasi.** Selesai. Jangan buka lagi sampai sore.

**🔴 PANTANGAN:**
- Jangan beli di tengah minggu (kecuali rebalance darurat)
- Jangan edit Auto-Order (TP/CL) tanpa alasan kuat
- Jangan panic sell saat market volatile

### 🕐 Hari Kerja (Selasa-Jumat) — Hands-Off Mode

**Morning Routine (08:45):**
- Baca evaluasi bot di Telegram
- Cek status portofolio: `/status`
- **Jika bot bilang HOLD → ya HOLD.** Jangan edit.
- **Jika bot bilang CL mendekati → siap-siap mental.** Tapi Auto-Order sudah aktif, jadi sistem akan handle.

**Lunch (12:20):**
- Baca lunch summary bot (optional)
- **JANGAN BUKA APLIKASI SEKURITAS** saat makan siang. Ini waktu istirahat.

**Closing (16:20):**
- Baca laporan penutupan bot
- Cek apakah ada posisi yang kena TP/CL hari ini
- Jika ada yang sold → catat untuk reflection

**🟢 ATURAN EMAS:**
- Buka aplikasi sekuritas **HANYA** di pagi (09:00) atau saat kena alert TP/CL
- Tidak ada checking portofolio setiap 15 menit. Ini swing trading, bukan scalping.
- Trust the system. Bot punya edge +1.34% dari backtest 5 tahun.

### 🔄 Next Monday — Rebalance

**Jumat 18:05 berikutnya:**
- Bot kirim rekomendasi rotasi baru
- **Posisi lama yang trend-nya patah** akan dikasih sinyal exit
- **Posisi baru** akan masuk rekomendasi

**Senin 09:00 berikutnya:**
- Exit posisi lama (jika ada yang perlu rotasi)
- Entry posisi baru
- Repeat cycle

---

## 💰 DCA Strategy (Rp 1 Juta/Bulan)

**Setup DCA:**
1. Setiap tanggal 1 (atau gajian), transfer Rp 1 juta ke RDN sekuritas
2. Tunggu sampai Senin berikutnya
3. Modal baru otomatis akan dialokasikan ke posisi aktif (equal weight)

**Contoh:**
- **Bulan 1:** Modal Rp 1 juta → 4 posisi @ Rp 250k
- **Bulan 2:** Tambah Rp 1 juta → 4 posisi @ Rp 500k
- **Bulan 3:** Tambah Rp 1 juta → 4 posisi @ Rp 750k
- Dst...

**Goal:** Setelah 5 tahun dengan DCA konsisten, backtest menunjukkan modal tumbuh ~100x (dari Rp 60 juta injected → Rp 6+ miliar). **Tapi ini backtest, bukan jaminan.**

---

## 🛠️ Skenario Khusus

### 🟢 Skenario 1: Portofolio Kosong (Fresh Start)

**Langkah:**
1. Tunggu Jumat sore untuk rekomendasi bot
2. Weekend: analisis santai
3. Senin pagi: eksekusi 3-4 saham sekaligus
4. Pasang Auto-Order semua posisi
5. Input semua transaksi ke bot via `/buy`

**Jangan:**
- Entry di hari random (Selasa, Rabu, Kamis)
- FOMO buy saat market sedang green
- Beli tanpa pasang Auto-Order

### 🟡 Skenario 2: Memegang Posisi (Hold Mode)

**Morning Routine:**
1. **08:45** — Baca evaluasi bot
2. **Cek status:** `/status`
3. **Jika HOLD** → tutup Telegram, fokus kerja
4. **Jika mendekati TP/CL** → siap mental, tapi Auto-Order will handle

**Jangan:**
- Edit TP/CL gara-gara panik
- Sell manual sebelum kena TP/CL (kecuali alasan fundamental berubah)
- Overthink. Trust the system.

### 🔴 Skenario 3: Kena Cut Loss

**Psikologi:**
- CL adalah bagian dari strategi. Tidak ada strategi dengan 100% win rate.
- Bot punya win rate 63.9%, artinya **36.1% trade akan loss**. Itu normal.
- CL -3% melindungi modal dari kerugian lebih besar (-10%, -20%, dll).

**Langkah Setelah CL:**
1. **Catat transaksi:** `/sell TICKER HARGA LOT`
2. **Evaluasi:** Apakah execution sudah benar? (entry sesuai rekomendasi, pasang Auto-Order, dll)
3. **Move on.** Tunggu rekomendasi Jumat berikutnya.
4. **JANGAN revenge trading** (beli saham random untuk "balas dendam")

### 🟢 Skenario 4: Kena Take Profit

**Celebration Time!** 🎉

**Langkah:**
1. **Catat transaksi:** `/sell TICKER HARGA LOT`
2. **Cek stats:** `/stats` untuk lihat win rate aktual Anda
3. **Tunggu Jumat** untuk rekomendasi rotasi baru
4. **Reinvest profit** ke posisi baru Senin depan

**Jangan:**
- Langsung beli saham lain di hari yang sama (emosi lagi high)
- Withdraw profit (kecuali butuh darurat). Biar compound.

### ⚠️ Skenario 5: Emergency Exit (Fundamental Change)

**Kapan boleh exit manual sebelum TP/CL?**
- **Force majeure:** Krisis ekonomi global, perang, pandemi
- **Company-specific:** Saham di-suspend, fraud, bankruptcy
- **Personal emergency:** Butuh cash mendadak

**Langkah:**
1. Sell manual di aplikasi sekuritas
2. Catat: `/sell TICKER HARGA LOT`
3. Tunggu Jumat untuk rekomendasi replacement

**Jangan:**
- Exit gara-gara FUD Twitter/Telegram group
- Exit gara-gara 1-2 hari merah (ini swing trading, wajar ada koreksi)

---

## 📊 Monitoring Tools

### `/status` — Portfolio Dashboard
Gunakan ini sebagai dashboard utama:
- Floating PNL semua posisi
- Batas TP/CL per saham
- Modal tersisa (cash)
- Pending orders (antrean)

**Kapan cek:**
- Pagi sebelum market buka (08:45)
- Setelah entry baru (Senin 09:30)
- Sebelum weekend (Jumat 16:00)

### `/stats` — Performance Tracker
Gunakan ini untuk review bulanan:
- Win rate aktual Anda
- Expectancy aktual
- Profit factor
- Total realized PnL

**Kapan cek:**
- Akhir bulan (untuk evaluasi performa)
- Setelah 10+ trades (untuk validasi edge)

### `/evaluate` — AI Second Opinion
Gunakan ini untuk konsultasi:
- Saat ragu hold atau cut
- Saat market volatile
- Untuk validasi keputusan

**Jangan:**
- Gunakan setiap hari (overthink)
- Override Auto-Order berdasarkan evaluate saja

---

## 🎓 Tips & Best Practices

### ✅ DO:
- **Follow the system religiously.** Entry Senin, rotasi Jumat.
- **DCA konsisten.** Rp 1 juta/bulan, no skip.
- **Pasang Auto-Order immediately** setelah entry.
- **Trust the backtest.** Expectancy +1.34% valid kalau dijalankan konsisten.
- **Review monthly stats** untuk validasi edge masih jalan.

### ❌ DON'T:
- **Trade berdasarkan emosi.** Takut, serakah, FOMO.
- **Edit TP/CL sembarangan.** Itu parameter backtest-proven.
- **Revenge trading** setelah kena CL.
- **Buka aplikasi sekuritas setiap 15 menit.** Ini bukan scalping.
- **Skip DCA bulan ini** gara-gara market lagi turun. Justru itu peluang.

### 🧠 Mindset:
- **Probabilistic thinking:** Tidak ada strategi 100% win. Fokus pada expectancy jangka panjang.
- **Process over outcome:** 1 loss ≠ sistem gagal. 100 trades = validasi sistem.
- **Patience:** Compound butuh waktu. 5 tahun, bukan 5 bulan.

---

## 🆘 Troubleshooting

### "Bot gak kirim rekomendasi Jumat sore"
- Cek apakah bot running: kirim `/status` di Telegram
- Cek log server: `go run cmd/bot/main.go`
- Manual trigger: `/recommend`

### "Semua rekomendasi bot F Sell > F Buy"
- Skip semua. Tunggu Jumat depan.
- Market lagi distribusi phase. Cash is a position.

### "Auto-Order gak tereksekusi padahal harga sudah sentuh"
- Cek setting Auto-Order di aplikasi sekuritas (pastikan status Active)
- Kadang ada gap (harga loncat), Auto-Order gak tersentuh
- Manual monitor jika mendekati TP/CL

### "Loss terus 5 trades berturut-turut"
- **Normal variance.** Coin flip 5x tails pun mungkin.
- Validasi: apakah execution sudah benar? (entry sesuai sistem, pasang Auto-Order, dll)
- Jika execution benar, continue. Edge perlu sample size besar (50+ trades).
- Jika execution salah (sering skip Auto-Order, entry random), fix execution dulu.

### "Win rate saya cuma 40%, backtest 63.9%"
- Cek sample size. <30 trades = belum representatif.
- Cek execution quality:
  - Apakah selalu entry Senin? ✅
  - Apakah selalu pasang Auto-Order TP +5%, CL -3%? ✅
  - Apakah skip saham F Sell > F Buy? ✅
- Jika semua benar, continue sampai 100+ trades untuk validasi.

---

## 📞 Support

Untuk pertanyaan teknis atau bug report:
- **GitHub Issues:** [Link repo]
- **Telegram:** @dzikrimutawakkil (jangan spam)

**Reminder:** Bot ini adalah tool, bukan holy grail. Trade at your own risk. 🚀
