# Spec: Backtester Filter Teknikal C-BoW (v1.0)
Dibuat: 2026-10-12
Status: 🔲 TODO
Bergantung pada: Spec 01, 02, 04 (F3 ANTI-ARB, F5 `Bar`/`GetBars`)

## Tujuan
Mengukur apakah **filter teknikal deterministik** (`GetStockScore` + filter IHSG) + aturan exit di `internal/rules` menghasilkan expectancy positif **setelah fee dan slippage**, tanpa memakai uang atau Telegram. Bagian Gemini tidak bisa di-backtest (berita historis); itu ditangani Spec 06.

## Prinsip
- **Tanpa lookahead:** sinyal di hari *t* hanya boleh memakai bar sampai hari *t*.
- Mesin backtest **tidak boleh** meng-import `telegram`, `genai`, atau membaca env. `go run ./cmd/backtest` harus jalan tanpa `.env`.
- Aturan exit dan fee diambil dari `rules.Default` (sama dengan live).

## Perubahan

### 1. Package baru `internal/signals` (murni; import hanya `models`, `pricing`)
Pindahkan logika dari `research.GetStockScore` dan `research.GetMarketFilterStatus`, tanpa mengubah aturan:
```go
type Params struct {
    MinScore       float64 // 8
    PumpFilterPct  float64 // 3.5 ; 0 = filter dimatikan
    ARBDropPct     float64 // 10  ; 0 = filter dimatikan
}
func DefaultParams() Params

type Signal struct {
    Score    float64 // -1 jika diblokir
    Code     string  // HAMMER, GREEN_BOUNCE, SIAGA, PUCUK, PISAU, TANGGUNG, BLOCKED_ARB, BLOCKED_PUMP, NO_DATA
    Verdict  string  // teks Telegram yang sudah ada
    DistToMA float64
    MA20     float64
    Low      float64 // low candle sinyal (untuk CL struktural)
}
func Evaluate(bars []models.Bar, p Params) Signal            // bars = data bersih, urut waktu, bar terakhir = hari sinyal
func MarketSafe(ihsgCloses []float64) (safe bool, state string) // state: "DOWNTREND" | "KOREKSI" | "UPTREND"
```
`research.GetStockScore` dan `GetMarketFilterStatus` menjadi pembungkus tipis (ambil data → `signals.*` → bentuk pesan). Perilaku live **identik**: tambahkan test regresi yang membandingkan `GetStockScore` lama (nilai kembalian dari bar sintetis) dengan hasil baru.

Verifikasi urutan filter: Filter ANTI-ARB → Filter pump dari low → Hammer → Green Bounce → Siaga → Pucuk → Pisau → Tanggung.

### 2. Data historis `internal/backtest/data.go`
- `LoadBars(symbol string, years int, refresh bool) ([]models.Bar, error)` memakai `market.GetBars(symbol, "5y")`, di-cache ke `data/cache/<SYMBOL>.json` (cache dipakai jika ada dan `refresh == false`).
- IHSG: `^JKSE`. Keselarasan tanggal: filter IHSG di hari *t* memakai close IHSG sampai tanggal *t*.
- Catatan di laporan: harga Yahoo tidak disesuaikan dividen; watchlist dipilih belakangan (survivorship bias).

### 3. Mesin `internal/backtest/engine.go`
Per tanggal *t* (urut), per saham (urut abjad), jika tidak sedang memegang saham itu dan jumlah posisi < `MaxPositions`:
- Sinyal: `signals.Evaluate(bars[:i+1])` dengan `Score >= MinScore` **dan** (`MarketSafe` atau `--ignore-market`). Butuh ≥ 60 bar riwayat sebelum *t*.
- **Entry:** harga close hari *t* × (1 + slippage). Dibulatkan `pricing.RoundToFraction`. Alokasi = `equity / MaxPositions`, lot = `floor(alokasi / (harga×100×(1+BuyFee)))`; lewati jika < 1 lot.
- Level: `CL = rules.StructuralCutLoss(Signal.Low)`, `TP1 = rules.TP1Price(entry)`. Awal `highest = entry`.

Per hari *d > t* untuk posisi terbuka (urutan **konservatif**, hentikan setelah exit penuh):
1. `open <= CL` → exit semua di `open` (alasan `CL_GAP`).
2. `low <= CL` → exit semua di `CL` (`CL`).
3. Trailing (aktif jika `NetPNL(entry, highest_sebelum_hari_d) >= TrailingActivationNetPct`); `TSL = TrailingStopPrice(highest_sebelum_hari_d)`: `open <= TSL` → exit di `open` (`TSL_GAP`); `low <= TSL` → exit di `TSL` (`TSL`).
4. TP1 (jika belum): `high >= TP1` → jual `TP1SellFraction` lot di `max(open, TP1)` (`TP1`); tandai selesai (minimal 1 lot dijual; jika lot total = 1, jual semua).
5. `highest = max(highest, high)` (diperbarui di akhir hari).
6. Time stop: pada hari ke-`MaxHoldDays` setelah entry (default 5), exit sisa di close (`TIME`). `0` = nonaktif.
- Fee: `BuyFee` saat beli, `SellFee` setiap jual (net). Mark-to-market equity harian pakai close.

### 4. CLI `cmd/backtest/main.go`
```
--symbols      daftar dipisah koma (default: config.Watchlist)
--years        5
--capital      1000000
--max-positions 3
--slippage-pct 0.2
--max-hold     5
--min-score    8
--no-pump-filter / --no-arb-filter   (matikan filter untuk ablation)
--ignore-market
--oos-from     YYYY-MM-DD (default: tanggal pada 70% rentang data)
--seed         42
--out          backtest_out/
--refresh      (abaikan cache)
```

### 5. Output (`--out`)
- `trades.csv`: `entry_date,symbol,signal_code,entry,exit_date,exit_reason,lots_sold_parts,pnl_pct_net,pnl_rp_net,hold_days`
- `equity.csv`: tanggal, equity, drawdown.
- `report.md` berisi:
  1. Parameter + peringatan survivorship bias + "Gemini tidak termasuk".
  2. **Funnel sinyal:** jumlah hari-saham per `Code` (termasuk `BLOCKED_ARB`/`BLOCKED_PUMP`) dan berapa yang lolos filter IHSG.
  3. Metrik **total / in-sample / out-of-sample**: jumlah trade, win rate, rata-rata menang/kalah (%), expectancy (% dan Rp), profit factor, max drawdown, rata-rata hold, **alasan exit** (jumlah & rata-rata PNL).
  4. Rincian per tahun dan per saham.
  5. **Baseline acak:** 200 kali simulasi dengan jumlah trade sama, tanggal+saham dipilih acak (seed tetap, hanya hari dengan syarat IHSG sama) memakai exit yang sama. Laporkan rata-rata expectancy acak dan **persentil** expectancy strategi terhadap distribusi acak.
  6. **Ablation:** jalankan ulang tanpa filter pump dan tanpa filter ARB; tabel perbandingan jumlah trade & expectancy. (Hipotesis yang diuji: filter pump 3.5% dari low ikut memblokir hammer berekor panjang.)
  7. Kesimpulan otomatis satu kalimat: "Expectancy OOS positif/negatif setelah fee; persentil X terhadap entry acak".

## Test (data sintetis, tanpa jaringan)
`signals`:
- Bar sintetis hammer valid (low ≤ MA20, close > MA20, wick ≥ 2×body, ditutup di 30% atas) dan pump-from-low ≤ 3.5% → `HAMMER`, Score 10.
- Candle kemarin turun >10% close-ke-close → `BLOCKED_ARB`, Score -1.
- Close ≥ 3.5% di atas low hari ini → `BLOCKED_PUMP` (juga saat `PumpFilterPct = 0` → tidak diblokir).
- Close di bawah MA20 → `PISAU`, Score 1.

`engine` (entry 1000, signal low 980 → CL 970, TP1 1040, slippage 0, fee 0 untuk kejelasan):
1. Hari+1: open 1010, high 1045, low 1005 → exit parsial `TP1` di 1040; hari+2: open 1030, low 960 → sisa exit di 970 (`CL`). Alasan exit berurutan `[TP1, CL]`.
2. Hari+1: open 960 → seluruh posisi exit di 960 (`CL_GAP`).
3. Tidak ada level tersentuh sampai hari ke-5 → `TIME` di close hari ke-5.
4. Satu hari memuat low ≤ CL **dan** high ≥ TP1 → `CL` (konservatif), TP1 tidak terjadi.
5. Dengan fee 0.15%/0.25%: PNL net pada kasus 3 lebih kecil dari PNL gross sebesar ±0.40% dari nilai posisi.
6. Hasil identik untuk dua kali run dengan seed yang sama.

## Files
`internal/signals/{signals.go,signals_test.go}`, `internal/backtest/{data.go,engine.go,report.go,*_test.go}`, `cmd/backtest/main.go`; ubah `internal/research/analyst.go` menjadi pembungkus.

## Checklist
- [ ] `go build ./... && go vet ./... && go test ./...` bersih tanpa `.env`
- [ ] `go run ./cmd/backtest --years 5` menghasilkan `backtest_out/{trades.csv,equity.csv,report.md}`
- [ ] Bot live memberi hasil sinyal yang sama seperti sebelum refactor (test regresi lulus)
- [ ] Laporan memuat funnel, IS/OOS, baseline acak, ablation
- [ ] Tidak ada import `telegram`/`genai` di `internal/signals` dan `internal/backtest`

## Di luar scope
Optimasi parameter otomatis (grid search), data intraday, dividen/aksi korporasi, backtest bagian AI.

## Catatan implementasi
(diisi Claude Code)
