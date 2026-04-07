# 🤖 Bot Trade DzikriJr (AI Fast Swing Assistant)

**DzikriJr** adalah bot Telegram asisten *trading saham cerdas* yang dibangun menggunakan **Golang** dan ditenagai oleh **Google Gemini AI**.

Bot ini dirancang dengan arsitektur **Clean Code (Domain-Driven Design)** untuk mengeksekusi strategi **Fast Swing Trading (1–5 hari)** di Bursa Efek Indonesia (IHSG).

Selain memberikan rekomendasi teknikal, bot ini juga berperan sebagai **manajer portofolio pribadi** yang membantu menjaga psikologis trader melalui:
- ⚠️ Early Warning System  
- 📊 Auto Journaling  
- 📈 Monitoring portofolio real-time  

---

## 🌟 Fitur Unggulan

### 🧠 AI-Powered Deep Research
Menggabungkan:
- Analisis fundamental (sentimen berita dari Google News RSS)
- Indikator teknikal (MA5, MA20, RSI, Volume)

Semua disintesis oleh **Gemini AI** untuk menghasilkan sinyal **BUY / SELL** yang lebih rasional.

---

### 🛡️ Early Warning System (EWS) & Trailing Stop
- Monitoring harga otomatis di background
- Notifikasi Telegram jika:
  - ✅ Target Take Profit tercapai  
  - ❌ Menyentuh Trailing Stop / Cut Loss  

---

### 📊 Auto-Journaling (CSV Logger)
Setiap transaksi:
- `/buy` dan `/sell`
- Dicatat otomatis ke `trade_history.csv`

Isi data:
- Harga beli & jual
- PNL (%)
- Alasan jual

➡️ Cocok untuk evaluasi bulanan (*trading journal*)

---

### ⏰ Smart Briefing (Cron Jobs)
Bot akan otomatis kirim:
- 🕘 08:45 WIB → sebelum market buka  
- 🕛 12:00 WIB → saat jam istirahat  

Tujuan:
- Evaluasi posisi
- Menghindari **Noon Trap**

---

### 📡 Dual Data Source
- 📊 Yahoo Finance → data historis
- ⚡ Google Finance → harga real-time (scraping)

---

## 🏗️ Arsitektur Proyek (Feature-Based)

Struktur modular untuk scalability & maintainability:

```bash
Bot_Trade_DzikriJr/
├── cmd/
│   └── bot/
│       └── main.go          # Entry point + inisialisasi bot & cron
├── internal/
│   ├── config/              # Config global (API key, watchlist, TP/CL)
│   ├── market/              # Integrasi Yahoo Finance & scraping Google Finance
│   ├── models/              # Struct (TradingPlan, History, dll)
│   ├── portfolio/           # Logic PNL, EWS, trailing stop
│   ├── research/            # AI prompt, indikator teknikal, news scraping
│   ├── storage/             # JSON storage & CSV logger
│   └── telegram/            # Handler & routing command Telegram

```
## 📈 Strategi Trading: Fast Swing (Sniper Mode)

Strategi agresif namun tetap terkontrol:

| Parameter | Value |
|----------|------|
| ⏱️ Durasi | 1 – 5 hari |
| 🎯 Take Profit | +3% hingga +5% |
| 🛑 Cut Loss | -2% |
| 📊 Indikator | MA5, MA20, RSI, Volume |

**Setup utama:**
- Harga di atas MA5
- Volume meningkat (indikasi akumulasi)

---

## 💬 Command Telegram

| Command | Contoh | Deskripsi |
|--------|--------|----------|
| `/research` | `/research ADRO` | Analisis saham (AI + teknikal + fundamental) |
| `/buy` | `/buy MIKA 2120 3` | Tambah saham ke portofolio |
| `/sell` | `/sell MIKA` | Jual saham + hitung PNL + logging |
| `/status` | `/status` | Lihat ringkasan portofolio |

---


## 🚀 Cara Menjalankan

### 1. Clone Repository
```bash
git clone https://github.com/username/Bot_Trade_DzikriJr.git
cd Bot_Trade_DzikriJr
```

---

### 2. Konfigurasi API Key

Edit file:

`internal/config/utils.go`

atau gunakan `.env`

Isi:

```env
BOT_TOKEN=your_telegram_bot_token
GEMINI_API_KEY=your_gemini_api_key
MY_CHAT_ID=your_telegram_chat_id
```

---

### 3. Install Dependency
```bash
go mod tidy
```

---

### 4. Jalankan Bot
```bash
go run cmd/bot/main.go
```

---

## ⚠️ Disclaimer

Bot ini hanyalah **tools bantu analisis**, bukan financial advisor.

Semua keputusan trading:
> sepenuhnya tanggung jawab pengguna.

Trading saham memiliki risiko tinggi, terutama untuk strategi jangka pendek.

Gunakan **risk management** yang baik.

---

## 💡 Quote

> *"Cut your losses short and let your profits run."*