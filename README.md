# 🚀 DzikriJrBot: Wealth Management Assistant

Bot Telegram berbasis **Go** untuk otomasi *screening* saham LQ45 dan monitoring portofolio secara *real-time*. Dirancang untuk efisiensi eksekusi bagi *swing trader*.


## 🛠 Core Features

* **Automatic Hunter:** Scan 42 saham LQ45 setiap jam 08:45 WIB.
* **Smart Scoring:** Analisis teknikal menggunakan kombinasi **MA20** dan **RSI**.
* **Aggressive Sorting:** Mengurutkan rekomendasi berdasarkan kedekatan harga dengan garis MA20 (*buy on rebound*).
* **Portfolio Guard:** Monitoring *Take Profit* (**7.5%**) dan *Stop Loss* (**2.5%**) otomatis.
* **Direct Intel:** Link berita emiten (Stockbit & Google News) langsung di dalam bot.


## 📈 Strategy Logic

Bot bekerja dengan prinsip **Trend Following + Momentum**:

* **Trend:** Harga > MA20.
* **Momentum:** RSI di area 40-60.
* **Risk-to-Reward:** 1:3 (Menjaga akun tetap tumbuh meski *win rate* 50%).


## 💻 Setup & Run

1. Isi `MyChatID` dan `BotToken` di `config.go`.
2. Jalankan aplikasi:
```bash
go run .

```

## 📈 Supported Stocks (LQ45 Pool)
Bot ini memantau 42 emiten paling likuid di IHSG:
> ACES, ADRO, AKRA, AMRT, ANKM, ASII, BBCA, BBNI, BBRI, BBTN, BMRI, BRIS, BRPT, BUKA, CPIN, EMTK, ESSA, EXCL, GOTO, HRUM, ICBP, INCO, INDY, INKP, INTP, ITMG, KLBF, MAPI, MBMA, MDKA, MEDC, MIKA, PGAS, PGEO, PTBA, SIDO, SMGR, SRTG, TLKM, TPIA, UNTR, UNVR.


## ⌨️ Command List

* `/recommend` - Cari 3 saham terbaik saat ini.
* `/buy [KODE] [HARGA] [LOT]` - Daftarkan saham ke pantauan "Satpam".
* `/status` - Cek *real-time* profit/loss portofolio.
* `/sell [KODE]` - Berhenti memantau saham tertentu.
* `/reset` - Bersihkan semua data pantauan.

---

*Built for personal wealth automation.*

---