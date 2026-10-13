# Spec: Satu Sumber Aturan TP / CL / TSL (v1.0)
Dibuat: 2026-10-12
Status: ✅ DONE
Bergantung pada: Spec 01

## Latar belakang (bug yang diperbaiki)
1. **Tiga angka cut loss berbeda:** rekomendasi menghitung CL struktural (`low × 0.99`), `/buy` menyimpan CL flat 3%, alert monitor memakai 2% net (`GoogleCLTarget`). `plan.CutLoss` dan `plan.TakeProfit` **tidak pernah dibaca** oleh `monitoring.go`.
2. **TP berbeda:** rekomendasi TP1 +4%, `/buy` TP 7%, alert TP 5% net. Trailing: teks "3%", config 2.5%, komentar 1.5%.
3. **Trailing stop praktis tidak pernah berfungsi.** Aktivasi memakai *harga sekarang* (`yahooPNL >= TSLActivationTrigger`). Level TSL baru berada di atas +2% net jika harga tertinggi sudah ≥ ±+5% gross, yaitu hampir di level alert TP. Di bawah itu alert trailing tidak pernah muncul. Aktivasi harus berdasarkan **harga tertinggi**.
4. **Posisi hidup lagi:** monitor menulis balik `config.MyStocks[symbol] = plan` dari snapshot. Jika `/sell` terjadi saat loop berjalan, saham yang sudah dijual muncul lagi.
5. **Position sizing:** `maxLotsByCash` memakai total modal, bukan sisa modal. Jika CL ≥ harga, `lossPerLot` dipaksa 1 sehingga `maxLotsByRisk` membengkak. Logika sizing diduplikasi di `ProcessResearchCommand` dan `ProcessRecommendation`.
6. `/buy` pada saham yang sudah ada mereset `BuyDate` dan `HighestPrice`, dan selalu menimpa CL dengan 3% dari rata-rata.

## Perubahan

### 1. Package baru `internal/rules` (import hanya `pricing` dan `models`)
```go
type Rules struct {
    Strategy                 string  // "C-BoW"
    BuyFee, SellFee          float64 // 0.0015, 0.0025
    DefaultCutLossPct        float64 // 3.0
    StructuralCLBufferPct    float64 // 1.0
    BuyZoneBelowPct          float64 // 1.0
    TP1Pct                   float64 // 4.0 (gross di atas entry)
    TP1SellFraction          float64 // 0.5
    TrailingStopPct          float64 // 3.0
    TrailingActivationNetPct float64 // 2.0 (net PNL% di HighestPrice)
    RunawayPct               float64 // 3.0
    MaxRiskPerTradePct       float64 // 5.0
    DefensiveRiskDivisor     float64 // 2.0
    MinPurchaseAmount        float64 // 250000
}
var Default = Rules{ /* nilai di komentar di atas */ }
```
Hapus dari `config`: `TPPercent, CLPercent, YahooTPTrigger, YahooCLTrigger, GoogleTPTarget, GoogleCLTarget, TrailingStopPercent, TSLActivationTrigger, BuyFee, SellFee, MaxRiskPerTrade, MinPurchaseAmount, RunawayPercent`. Ganti semua referensi ke `rules.Default.*`. Tetap di `config`: `TotalModalTrading`, `CheckPeriod`, `EmergencyDelay`, `Watchlist`, `MyStocks`, `PendingOrders`, `DataMutex`, kredensial.

Fungsi (semua murni):
```go
func (r Rules) DefaultCutLoss(entry float64) float64
func (r Rules) StructuralCutLoss(signalLow float64) float64
func (r Rules) TP1Price(entry float64) float64
func (r Rules) TrailingStopPrice(highest float64) float64
func (r Rules) NetPNL(entry, price float64) float64
func (r Rules) TrailingActive(p models.TradingPlan) bool
func (r Rules) EffectiveStop(p models.TradingPlan) float64

type Decision int
const ( None Decision = iota; CutLoss; TrailingStop; TakeProfit1 )
func (r Rules) Evaluate(p models.TradingPlan, price float64) Decision
```

### 2. Model
`models.TradingPlan` tambah `TP1Done bool`. `TakeProfit` kini berarti **harga TP1**.

### 3. Sizing murni `internal/rules/plan.go`
`SuggestPlan(PlanInput) PlanOutput` — fix MaxLots=0 saat CL ≥ harga.

### 4. Monitor rewrite
Pakai `Evaluate()`, `UpdatePlan()`, `UpdateOrder()`.

### 5. `/buy` CL opsional
Format: `/buy KODE HARGA LOT [CL]`. Averaging tidak reset `HighestPrice`/`BuyDate`.

### 6. Research plan text
`buildPlanText()` di `internal/research/plan_text.go`.

## Checklist
- [x] `go build ./... && go vet ./... && go test ./...` bersih tanpa `.env`
- [x] Konstanta lama (YahooTPTrigger, GoogleCLTarget, dll.) tidak ditemukan
- [x] Monitor memakai `plan.CutLoss`/`plan.TakeProfit`; race condition tulis-balik snapshot sudah diperbaiki
- [x] `/buy KODE HARGA LOT CL` bekerja; tanpa CL tetap bekerja
- [x] Rekomendasi dengan `MaxLots == 0` tidak menampilkan `/buy`
- [x] Tidak ada literal "3%"/"4%" di string pesan; semua dari `rules.Default`

## Catatan implementasi
1. `rules.go`: semua fungsi murni, tidak ada mutex, hanya import `pricing` dan `models`.
2. `TrailingActivationNetPct` = 2.0 (% net PNL di harga tertinggi).
3. Monitor rewrite: iterate keys (bukan snapshot copy plan) untuk avoid race; `UpdatePlan` hanya write jika plan masih ada.
4. `/buy`: CL opsional args[4]; averaging tidak reset `HighestPrice`/`BuyDate`; `TP1Done=false` selalu.
5. `/sell`: set `TP1Done=true` pada partial sell via `UpdatePlan`.
6. `buildPlanText`: tidak tampilkan `/buy` jika `MaxLots==0`.
7. Test `UpdatePlan`/`UpdateOrder` ada di `config_test.go`.
8. `buyFee`/`sellFee` di `statistics.go` pakai `rules.Default.*`.
9. Bug pre-existing `analyst.go:175` (fmt.Sprintf arg mismatch) sudah diperbaiki di Spec 01.
