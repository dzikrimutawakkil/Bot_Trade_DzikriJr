# Cara Menjalankan Sistem DzikriJr

Panduan lengkap untuk menjalankan **Backtest**, **Forward Test**, dan **Live Trading Bot**.

---

## Prasyarat

### Go Toolchain
- **Minimum:** Go 1.23+
- **Direkomendasikan:** Go 1.26.1
- Pastikan `go` ada di PATH

```powershell
# Cek versi
go version

# Jika versi lama, install via winget
winget install --id GoLang.Go -e --version 1.26.1 --accept-package-agreements --accept-source-agreements
```

### File `.env` (Wajib)
Buat file `.env` di root folder (HARUS ada di `.gitignore`):

```env
# Telegram Bot
BOT_TOKEN=123456789:AAxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
MY_CHAT_ID=123456789

# Google Gemini AI
GEMINI_API_KEY=AIzaxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Forward Test Mode (true = aktif, false = nonaktif)
FORWARD_TEST_MODE=false
FORWARD_TEST_MODAL=1000000
```

---

## 1. Backtest

### Command Dasar
```bash
go run cmd/backtest/main.go [flags]
```

### Flags Paling Sering Dipakai

| Flag | Default | Contoh | Deskripsi |
|------|---------|--------|-----------|
| `-strategy` | `cbow` | `momentum-weekly` | Strategi yang digunakan |
| `-years` | `5` | `5` | Tahun data historis |
| `-symbols` | watchlist | `BBRI,BMRI,ADRO,ITMG` | Simbol Saham (koma-separated) |
| `-max-hold` | `5` (lain) / `20` (weekly) | `20` | Max hold days |
| `-refresh` | `false` | `true` | Refetch data dari Yahoo Finance |
| `-compare` | `false` | `true` | Run semua strategi + compare report |
| `-no-baseline` | `false` | `true` | Skip random3/dca-naif/buy-hold |
| `-slippage-pct` | `0.2` | `0.2` | Slippage percentage |
| `-cl-fixed-pct` | `3.0` | `3.0` | Cut Loss percentage |
| `-tp-pct` | `5.0` | `5.0` | Take Profit percentage |

### Strategi yang Tersedia

| Strategy | Deskripsi |
|----------|-----------|
| `cbow` | Filter teknikal C-BoW |
| `momentum-weekly` | Weekly Momentum Rotation (S08) |
| `breakout20` | Breakout 20% dari MA20 |
| `pullback20` | Pullback ke MA20 |

### Contoh Penggunaan

```bash
# Backtest momentum-weekly 5 tahun 4 simbol
go run cmd/backtest/main.go -strategy momentum-weekly -years 5 -symbols BBRI,BMRI,ADRO,ITMG

# Backtest momentum-weekly full universe 5 tahun (dengan baselines)
go run cmd/backtest/main.go -strategy momentum-weekly -years 5

# Backtest momentum-weekly dengan slippage lebih konservatif
go run cmd/backtest/main.go -strategy momentum-weekly -years 5 -slippage-pct 0.3

# Backtest tanpa baselines (lebih cepat)
go run cmd/backtest/main.go -strategy momentum-weekly -years 5 -no-baseline

# Backtest C-BoW strategi dengan compare mode
go run cmd/backtest/main.go -strategy cbow -compare -years 5

# Backtest dengan force refresh data
go run cmd/backtest/main.go -strategy momentum-weekly -years 5 -refresh

# Backtest custom simbols dan short history
go run cmd/backtest/main.go -strategy momentum-weekly -years 2 -symbols BBCA,BBRI,BMRI
```

### Output Backtest
```
backtest_out/
├── report.md       # Laporan lengkap (metrik, K1-K7 criteria)
├── equity.csv      # Data equity curve per hari
├── trades.csv      # Detail setiap trade
└── compare.md     # (jika -compare) Comparison vs baseline
```

---

## 2. Forward Test (Simulasi)

Forward test adalah **simulasi** trading dengan modal virtual Rp 1.000.000. Tidak ada uang sungguhan.

### Persiapan

**Step 1: Aktifkan Forward Test Mode**
```powershell
# Windows PowerShell
$env:FORWARD_TEST_MODE="true"
$env:FORWARD_TEST_MODAL="1000000"

# Linux/Mac
export FORWARD_TEST_MODE="true"
export FORWARD_TEST_MODAL="1000000"
```

**Step 2: Jalankan Bot**
```bash
go run cmd/bot/main.go
```

### Apa yang Terjadi

- **Setiap 16:20 (Senin-Jumat):**
  - `EvaluateDaily()` — cek semua posisi untuk exit conditions
  - Exit reasons: CL (-3%), TSL (trailing stop -3%), TP1 (+5%), TIME (20 hari)

- **Setiap Jumat 18:05:**
  - `ProcessWeeklyEntry()` — scan MA5 > MA20 > MA50, pilih top 4 momentum
  - Equal weight allocation
  - Log entry ke `forward_test_history.csv`

- **Setiap Jumat 16:20:**
  - `CheckTrendBreak()` — cek MA5 < MA20 untuk semua posisi aktif
  - Exit otomatis jika trend broken

### Format Summary (Telegram)

Bot kirim laporan seperti ini ke Telegram:

```
📊 FORWARD TEST — Week 3
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Modal Awal : Rp 1.000.000
Equity     : Rp 1.023.500 (+2.35%)
Realized   : Rp +35.000
Unrealized : Rp -11.500
Win Rate   : 4/6 (66.7%)
Closed     : 6 trades
Max DD     : 3.21%

📁 Posisi Aktif (2):
  • BBRI: 4600 (+1.20%, 3 hari)
  • BMRI: 2150 (+0.50%, 1 hari)

📈 Recent Closed (6):
🟢 BBRI: +2.50% / Rp +11.500 (TP1)
🔴 ADRO: -3.10% / Rp -14.200 (CL)
```

### Stop Forward Test
```powershell
# Nonaktifkan (bot tetap jalan tapi tidak proses forward test)
$env:FORWARD_TEST_MODE=""

# Atau matikan bot
# Ctrl+C
```

---

## 3. Live Trading Bot

⚠️ **PERINGATAN:** Live trading menggunakan uang sungguhan. Pastikan memahami risikonya.

### Persiapan

**Step 1: Set Environment**
```powershell
# NONAKTIFKAN forward test mode
$env:FORWARD_TEST_MODE="false"
```

**Step 2: Pastikan `.env` Benar**
```env
BOT_TOKEN=123456789:AAxxxxx
GEMINI_API_KEY=AIzaxxxxx
MY_CHAT_ID=123456789
FORWARD_TEST_MODE=false
```

**Step 3: Jalankan Bot**
```bash
go run cmd/bot/main.go
```

### Cron Schedule (Asia/Jakarta)

| Waktu | Fungsi | Keterangan |
|-------|--------|-----------|
| 08:45 | `ProcessPortfolioEvaluation()` | Evaluasi TSL & CL pagi + increment HoldDays |
| 12:30 | `ProcessPortfolioEvaluation()` | Evaluasi siang |
| 15:40 | `ProcessS08Recommendation()` | Scan S08 momentum candidates (tanpa Gemini) |
| 16:20 | `EvaluateForwardTest()` + `runDailySummary()` | Cek exit semua posisi + laporan |
| 16:20 (Jumat) | `CheckTrendBreak()` | Cek MA5 < MA20 untuk semua posisi |
| 18:05 (Jumat) | `ProcessWeeklyRebalance()` + `ProcessWeeklyEntry()` | Weekly entry live + forward test |

### Perintah Telegram

| Command | Deskripsi |
|---------|-----------|
| `/buy [KODE] [HARGA] [LOT]` | Beli saham |
| `/buy BBCA 9000 10` | Beli BBCA 9000, 10 lot |
| `/buy BBCA 9000 10 8700` | Beli dengan Cut Loss custom |
| `/sell [KODE] [HARGA]` | Jual semua |
| `/sell BBCA 9100 5` | Jual 5 lot BBCA |
| `/status` | Status portofolio |
| `/recommend` | Scan S08 momentum |
| `/research [KODE]` | Deep research dengan Gemini |
| `/antre [KODE] [HARGA] [LOT]` | Antrean beli (auto-match jika harga turun ke target) |
| `/cancel_antre [KODE]` | Batalkan antrean |
| `/reset confirm` | Hapus semua pantauan |
| `/forwardtest` | Laporan forward test |
| `/shadow` | Status shadow log sinyal |

### Exit Rules Live

| Exit | Trigger | Action |
|------|---------|--------|
| **CL** | Harga ≤ -3% dari entry | Notifikasi + suggested sell |
| **TSL** | Harga ≤ trailing stop (aktif setelah +2%) | Notifikasi + suggested sell |
| **TP1** | Harga ≥ +5% | Sell 50% lot |
| **TIME** | HoldDays ≥ 20 hari bursa | Notifikasi untuk manual sell |
| **TREND BREAK** | MA5 < MA20 (Jumat) | Exit + rebalance |

---

## 4. Struktur File Sistem

### File State

| File | Isi | Format |
|------|-----|--------|
| `stocks.json` | Portofolio live aktif | JSON |
| `forward_test.json` | State forward test | JSON |
| `forward_test_history.csv` | Log trades forward test | CSV (append only) |
| `trade_history.csv` | Log semua transaksi live | CSV (append only) |

### Folder Packages

```
internal/
├── forwardtest/       # Forward test simulation
│   ├── forwardtest.go  # State, entry, evaluate, MaxDD tracking
│   └── logger.go       # CSV logging
├── portfolio/        # Live trading
│   ├── monitoring.go    # Price monitor + TSL/CL/TIME
│   ├── evaluation.go   # Daily evaluation
│   └── process_weekly.go # Weekly rebalance S08
├── research/         # Research & signals
│   ├── handler_research.go # S08 momentum scanner
│   └── summary.go     # Daily summary + Trend Break
└── rules/           # Trading rules (TP/CL/TSL)
```

---

## 5. Troubleshooting

### Error: "package is not in GOROOT"
```bash
# Go toolchain perlu di-upgrade
winget install --id GoLang.Go -e --version 1.26.1 --accept-package-agreements --accept-source-agreements

# Refresh PATH atau buka terminal baru
```

### Error: "strconv.FormatFloat" atau "undefined: xyz"
```bash
# Go version terlalu lama, upgrade
go version
# Harus >= 1.23
```

### Forward Test Tidak Jalan
```powershell
# Cek environment variable
$env:FORWARD_TEST_MODE

# Harus "true" (case-sensitive)
$env:FORWARD_TEST_MODE="true"
```

### Tidak Ada Data Telegram
```bash
# Cek .env
# BOT_TOKEN dan MY_CHAT_ID harus valid
# MY_CHAT_ID adalah angka (bukan string "123456789", tapi 123456789)
```

### Build Gagal
```bash
# Clear module cache
go clean -modcache

# Re-download dependencies
go mod tidy

# Build ulang
go build ./...
```

### Backtest Timeout / Data Kosong
```bash
# Gunakan -refresh untuk force refetch
go run cmd/backtest/main.go -refresh -strategy momentum-weekly -years 5

# Cek koneksi internet
# Yahoo Finance mungkin rate-limit
```

---

## 6. Build Binary (Production)

Jika ingin build binary langsung (tanpa `go run`):

```bash
# Build bot
go build -o dzikrijr-bot.exe ./cmd/bot/

# Build backtester
go build -o backtest.exe ./cmd/backtest/

# Jalankan binary langsung
./dzikrijr-bot.exe
./backtest.exe -strategy momentum-weekly
```

Binary lebih cepat karena tidak perlu compile setiap kali dijalankan.
