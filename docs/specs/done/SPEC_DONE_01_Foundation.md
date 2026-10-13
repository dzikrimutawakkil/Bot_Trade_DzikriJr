# Active Spec: Foundation — testability & package murni (v1.0)

> Dibuat: 2026-10-12
> Source: `docs/specs/SPEC_TODO_01_Foundation.md`

---

## Latar belakang
- `internal/config/config.go` punya `init()` yang `panic` jika `BOT_TOKEN`, `MY_CHAT_ID`, atau `GEMINI_API_KEY` kosong. Akibatnya `go test` di package mana pun yang meng-import `config` langsung crash tanpa `.env`.
- `RoundToFraction` dan `CalculateNetPNL` ada di `internal/utils`, yang meng-import `config` dan telegram. Backtester dan test ringan tidak boleh bergantung pada itu.
- `cmd/bot/main.go` meng-hardcode `250000` padahal `config.MinPurchaseAmount` sudah ada, dan membaca `MyStocks` tanpa lock.
- Belum ada test, `.gitignore`, atau `.env.example`.

## Perubahan

### 1. `config.LoadEnv() error` menggantikan `init()`
- Hapus `func init()` di `config.go`.
- Tambah `func LoadEnv() error`: coba `godotenv.Load` dari folder executable lalu dari cwd (error diabaikan), baca `BOT_TOKEN`, `MY_CHAT_ID` (`strconv.ParseInt`), `GEMINI_API_KEY`, isi variabel global `BotToken`, `MyChatID`, `GeminiAPIKey`.
- Kembalikan error yang menyebut **semua** variabel yang hilang/tidak valid sekaligus. Jangan `panic`.
- `main.go`: panggil `config.LoadEnv()` paling awal, `log.Fatalf` jika error. Hapus `godotenv.Load()` dan import `godotenv` dari `main.go`.

### 2. Package baru `internal/pricing` (tanpa import internal apa pun)
Pindahkan dari `internal/utils/utils.go`: `RoundToFraction` dan `CalculateNetPNL`. Hapus dari utils, perbarui semua pemanggil (`monitoring.go`, `handler_portfolio.go`, `evaluation.go`, `handler_research.go`, config-related). Perilaku tidak berubah.

### 3. Helper modal di `config`
```go
func CapitalInUse() float64      // RLock; sum(EntryPrice*Lots*100*(1+BuyFee)) semua MyStocks
func RemainingCapital() float64  // TotalModalTrading - CapitalInUse()
```
`main.go` (cron 15:40 Hunter) memakai `config.RemainingCapital()` dan `config.MinPurchaseAmount`. Hapus angka `250000` dan loop manual yang membaca `MyStocks` tanpa lock.

### 4. File repo
- `.gitignore`: `.env`, `stocks.json`, `trade_history.csv`, `*.log`, `data/`, `backtest_out/`, binary hasil build.
- `.env.example`: `BOT_TOKEN=`, `MY_CHAT_ID=`, `GEMINI_API_KEY=` (tanpa nilai).

## Test (library standar `testing`, table-driven)

`internal/pricing/pricing_test.go`

| Fungsi | Input | Hasil |
|--------|-------|-------|
| RoundToFraction | 556.4 | 555 |
| RoundToFraction | 3481.6 | 3480 |
| RoundToFraction | 199.6 | 200 |
| RoundToFraction | 5012 | 5000 |
| RoundToFraction | 514.8 | 515 |
| RoundToFraction | 500 | 500 |
| CalculateNetPNL(entry, jual, 0.0015, 0.0025) | 535, 555 | 3.32 (±0.005) |
| CalculateNetPNL | 535, 525 | -2.26 (±0.005) |
| CalculateNetPNL | 3340, 3470 | 3.48 (±0.005) |
| CalculateNetPNL | 3770, 3800 | 0.39 (±0.005) |

(Empat kasus terakhir diambil dari `trade_history.csv` asli.)

`internal/config/config_test.go`
- Import `config` tanpa env tidak panic (test ini sendiri membuktikannya).
- `LoadEnv()` dengan env kosong mengembalikan error yang memuat ketiga nama variabel.
- `CapitalInUse()` dengan `MyStocks = {A: entry 1000, 2 lot; B: entry 500, 10 lot}` ≈ 701.050 (toleransi 1 Rp).

## Files
| File | Perubahan |
|------|-----------|
| `internal/config/config.go` | Hapus `init()`, tambah `LoadEnv`, `CapitalInUse`, `RemainingCapital` |
| `internal/pricing/pricing.go` (baru) | `RoundToFraction`, `CalculateNetPNL` |
| `internal/utils/utils.go` | Hapus dua fungsi di atas |
| `cmd/bot/main.go` | `LoadEnv`, pakai `RemainingCapital` + `MinPurchaseAmount` |
| semua pemanggil | Ganti ke `pricing.*` |
| `.gitignore`, `.env.example` (baru) | - |

## Checklist
- [ ] `go build ./... && go vet ./... && go test ./...` bersih tanpa `.env`
- [ ] Tidak ada `panic` di jalur startup config
- [ ] `internal/pricing` tidak meng-import package internal lain
- [ ] `main.go` tidak lagi punya angka 250000 atau akses `MyStocks` tanpa lock
- [ ] `.gitignore` dan `.env.example` ada

## Di luar scope
Perubahan nilai TP/CL/TSL (Spec 02), format CSV (Spec 03).

## Catatan implementasi
(diisi setelah implementasi selesai)
