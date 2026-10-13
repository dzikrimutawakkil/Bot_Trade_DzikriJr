# Spec: Logger, Migrasi CSV & Statistik yang Benar (v1.0)
Dibuat: 2026-10-12
Status: 🔲 TODO (ACTIVE — Dipilih 2026-10-12)
Bergantung pada: Spec 01 ✅ DONE, Spec 02 ✅ DONE

## Latar belakang (bug yang diperbaiki)
1. **Format CSV tidak sinkron.** `LogTrade` menulis 7 kolom (`Waktu,Aksi,Saham,Harga,Lot,PNL (%),Catatan`), tetapi `statistics.go` membaca PNL dari index 6 (yang berisi *Catatan*) dan `trade_history.csv` milik user punya 8 kolom (`...,Lot,Strategi,PNL (%),Catatan`). Dengan logger sekarang `/stats` melewati semua baris.
2. **Fee dihitung dua kali.** `PNL (%)` yang dicatat `/sell` sudah net (`CalculateNetPNL`), lalu `statistics.go` mengurangi fee lagi dan memakai harga jual sebagai basis. Contoh data nyata: BBNI (+Rp 1.484 net) tampil sebagai rugi Rp 142; total Rp ±11.4 ribu padahal real ±Rp 15.9 ribu.
3. **Alasan exit tidak tercatat.** `/sell` melabeli "Take Profit"/"Cut Loss" memakai ambang 7%/3% yang tidak dipakai alert, sehingga cut loss -2.26% tercatat "Manual Sell".
4. Contoh output di `SPEC_DONE_WinRateTracker.md` (Avg Profit +1.40%, Net +Rp 461) tidak cocok dengan data CSV. Gunakan angka di bagian "Test" di bawah sebagai acuan.

## Format CSV baru (v3, 10 kolom)
```
Waktu,Aksi,Saham,Harga,Lot,Strategi,PNL (%),PNL (Rp),AlasanExit,Catatan
```
- `Strategi`: default `rules.Default.Strategy` ("C-BoW").
- `PNL (%)` dan `PNL (Rp)`: **net setelah fee**. BUY: `0.00` dan `0`.
- `AlasanExit` (hanya SELL): `TP1`, `TP`, `TSL`, `CL`, `MANUAL`. BUY: kosong.
- `Catatan`: teks bebas, tetap seperti sekarang ("Entry awal", "Manual Sell (Partial)", ...).

## Perubahan

### 1. `internal/storage/logger.go`
```go
type TradeRecord struct {
    Time       time.Time
    Action     string  // "BUY" | "SELL"
    Symbol     string
    Price      float64
    Lots       int
    Strategy   string  // kosong => rules.Default.Strategy
    PNLPercent float64
    PNLRp      float64
    ExitReason string
    Note       string
}
func LogTrade(rec TradeRecord)                       // Time kosong => time.Now(); tulis header v3 jika file baru
func ReadTradeHistory(path string) ([]TradeRecord, error)
func MigrateTradeHistory(path string) error
```
Ganti semua pemanggil `LogTrade(...)` (3 di `handler_portfolio.go`, 2 di `monitoring.go`) ke struct.

### 2. Layout dideteksi **per baris** (bukan hanya dari header)
File lama bisa campuran karena logger berubah di tengah jalan.
| Jumlah kolom | Layout |
|--------------|--------|
| 7 | v1: `Waktu,Aksi,Saham,Harga,Lot,PNL (%),Catatan` |
| 8 | v2: `Waktu,Aksi,Saham,Harga,Lot,Strategi,PNL (%),Catatan` |
| 10 | v3 (baru) |
Baris dengan jumlah kolom lain, baris kosong, dan baris separator (`---`/`===`) dilewati dan di-log sekali. Header dikenali jika kolom pertama `Waktu`.

### 3. `MigrateTradeHistory(path)` dipanggil di `main.go` sebelum `LoadData`
- Jika file tidak ada atau seluruh baris sudah v3 → tidak melakukan apa pun.
- Selain itu: salin ke `trade_history.csv.bak` (jika sudah ada, tambah suffix timestamp), lalu tulis ulang sebagai v3.
- `Strategi` kosong → `C-BoW`.
- `AlasanExit` untuk SELL: dari `Catatan` — mengandung "Take Profit" → `TP`, "Cut Loss" → `CL`, "Trailing" → `TSL`, selain itu `MANUAL`.
- `PNL (Rp)` untuk SELL: **replay posisi per saham secara berurutan**. BUY menambah lot dan memperbarui rata-rata tertimbang harga; SELL memakai rata-rata saat itu: `Rp = Harga*(1-SellFee)*Lot*100 - Rata2*(1+BuyFee)*Lot*100`. Jika tidak ada BUY sebelumnya untuk saham itu, fallback `Harga*(1-SellFee)*Lot*100*(1 - 1/(1+pnl%/100))`.
- Migrasi idempoten (dijalankan dua kali hasilnya sama).

### 4. `/sell` — `handler_portfolio.go`
- Hitung `rupiahPNL` bertanda (positif/negatif) **sebelum** dibalik untuk tampilan.
- `ExitReason` dari level plan (sebelum posisi diubah):
  1. `price <= plan.CutLoss` → `CL`
  2. `rules.TrailingActive(plan) && price <= TrailingStopPrice(plan.HighestPrice)` → `TSL`
  3. `price >= plan.TakeProfit` → `TP1` jika parsial, `TP` jika clear
  4. selain itu `MANUAL`
- `Catatan` tetap berformat "<Label> (Partial|Clear)"; label: "Cut Loss", "Trailing Stop", "Take Profit", "Manual Sell". Hapus perbandingan terhadap `TPPercent/CLPercent`.

### 5. `internal/storage/statistics.go`
- `CalculateStatistics()` = wrapper ke `CalculateStatisticsFromFile(path string)` (agar bisa dites).
- Baca via `ReadTradeHistory`; hanya baris `SELL`; menang jika `PNLPercent > 0`, kalah jika `<= 0`.
- Pakai `PNLRp` apa adanya. **Hapus** perhitungan ulang fee dan faktor `1.11`.
- Field `TradeStatistics` (ganti field Gross): `TotalTrades, WinningTrades, LosingTrades, WinRate, AvgProfit (%), AvgLoss (%), TotalProfitRp, TotalLossRp, NetProfitRp, RiskRewardRatio, ExpectancyRp, LastUpdated`.
- `RiskRewardRatio = rata-rata PNLRp menang ÷ |rata-rata PNLRp kalah|` (net; 0 jika tidak ada kalah).
- `ExpectancyRp = NetProfitRp / TotalTrades`.
- `FormatStatsMessage`: tambah baris `📐 Expectancy: Rp X / trade`; ganti catatan fee menjadi `_PNL net setelah fee Buy 0.15% | Sell 0.25%_` (angka dari `rules.Default`). Tambah keterangan `_Trade = jumlah transaksi jual (parsial dihitung terpisah)_`.
- Perbaiki `formatRp` agar pembulatan konsisten (tanpa desimal; `Rp 1.234`).

## Test

Fixture `internal/storage/testdata/trade_history_v2.csv` (data asli user):
```
Waktu,Aksi,Saham,Harga,Lot,Strategi,PNL (%),Catatan
2026-04-17 15:46:35,BUY,MTEL,535,10,C-BoW,0.00,Entry awal
2026-04-23 08:18:31,SELL,MTEL,555,5,C-BoW,3.32,Manual Sell (Partial)
2026-04-24 09:30:16,SELL,MTEL,525,5,C-BoW,-2.26,Manual Sell (Clear)
2026-04-24 15:49:31,BUY,MDKA,3340,1,C-BoW,0.00,Entry awal
2026-04-28 09:31:52,SELL,MDKA,3470,1,C-BoW,3.48,Manual Sell (Clear)
2026-04-28 15:47:03,BUY,BBNI,3770,1,C-BoW,0.00,Entry awal
2026-05-04 16:13:26,SELL,BBNI,3800,1,C-BoW,0.39,Manual Sell (Clear)
```
Buat juga `trade_history_v1.csv` (kolom Strategi dibuang, 7 kolom) dan `trade_history_mixed.csv` (header 8 kolom, baris SELL terakhir dalam 7 kolom).

Hasil yang diharapkan dari ketiganya setelah migrasi/`ReadTradeHistory` (toleransi ±1 Rp / ±0.01):

| SELL | PNL (Rp) | AlasanExit |
|------|----------|------------|
| MTEL 555 ×5 | 8905 | MANUAL |
| MTEL 525 ×5 | -6058 (-6057.5) | MANUAL |
| MDKA 3470 ×1 | 11632 | MANUAL |
| BBNI 3800 ×1 | 1484 | MANUAL |

`CalculateStatisticsFromFile` pada fixture hasil migrasi:
- TotalTrades 4, Winning 3, Losing 1, WinRate 75.0
- AvgProfit ≈ +2.40%, AvgLoss = -2.26%
- TotalProfitRp ≈ 22.021, TotalLossRp ≈ -6.058, NetProfitRp ≈ 15.964
- RiskRewardRatio ≈ 1.21, ExpectancyRp ≈ 3.991

Test lain: migrasi dua kali menghasilkan file identik; file `.bak` terbentuk; baris rusak dilewati tanpa panic; `LogTrade` pada file baru menulis header v3; `ExitReason` `/sell` (fungsi inferensi diekstrak agar bisa dites) untuk plan entry 535, CL 515, TP 555, highest 556: harga 514 → CL, 540 → TSL, 555 parsial → TP1, 548 → MANUAL.

## Files
| File | Perubahan |
|------|-----------|
| `internal/storage/logger.go` | `TradeRecord`, `LogTrade`, `ReadTradeHistory`, `MigrateTradeHistory` |
| `internal/storage/statistics.go` | Hitung ulang sesuai di atas |
| `internal/storage/testdata/*.csv` (baru), `*_test.go` (baru) | Fixture + test |
| `internal/portfolio/handler_portfolio.go`, `monitoring.go` | Pemanggil `LogTrade`, `ExitReason` |
| `internal/telegram/handler.go` | Tidak berubah, kecuali kompilasi |
| `cmd/bot/main.go` | Panggil `MigrateTradeHistory` |

## Checklist
- [ ] `go build ./... && go vet ./... && go test ./...` bersih tanpa `.env`
- [ ] Migrasi tiga fixture menghasilkan tabel di atas
- [ ] `/stats` pada data user menampilkan Net ≈ +Rp 15.964 (bukan ±11.4 ribu)
- [ ] Tidak ada lagi `1.11` / fee kedua di `statistics.go`
- [ ] Trade baru tercatat 10 kolom dengan `AlasanExit` yang benar

## Di luar scope
Statistik per posisi (BUY↔SELL dipasangkan), grafik, ekspor.

## Catatan implementasi
(diisi Claude Code)
