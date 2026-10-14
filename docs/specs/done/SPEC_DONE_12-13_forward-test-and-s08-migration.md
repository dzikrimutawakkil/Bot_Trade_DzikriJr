# ACTIVE_SPEC: Forward Test Environment + Live S08 Migration

**Tanggal:** 2026-10-14
**Status:** APPROVED by CEO
**Priority:** 🚀 Quick Wins (High Impact, Medium Effort)

---

## Keputusan CEO

| Keputusan | Jawaban |
|-----------|---------|
| Forward Test Modal | **Rp 1.000.000** |
| Forward Test Duration | **1 Bulan** |
| Live System Strategy | **Weekly Momentum Rotation S08** |

---

## TASK 1: Complete Forward Test Environment

### Scope
- Lengkapi package `internal/forwardtest/`
- Tambahkan Max DD tracking
- Tambahkan CSV logging
- Improve summary report

### Komponen yang Diperbaiki

```
internal/forwardtest/
├── forwardtest.go       # + MaxDD tracking, improve FormatSummary
├── logger.go            # NEW: Log trades to forward_test_history.csv
├── persistence.go       # Existing: SaveData/LoadData (thread-safe)
└── handler.go           # Existing: /forwardtest command
```

### FT-01: Max DD Tracking
Tambahkan field di `models.FTState`:
```go
MaxDD          float64  // Max Drawdown dalam %
PeakEquity     float64  // Highest equity pernah tercapai
```

Update logic setiap `EvaluateDaily()`:
1. Hitung unrealized PnL semua posisi
2. Hitung current equity = StartEquity + realized + unrealized
3. Update PeakEquity jika current > PeakEquity
4. Hitung DD = (PeakEquity - current) / PeakEquity * 100
5. Update MaxDD jika DD > MaxDD

### FT-02: CSV Logging
Buat `forwardtest/logger.go`:
```go
func LogTrade(trade models.FTClosedTrade)
func GetWeeklyStats() WeeklyStats
```

Format CSV `forward_test_history.csv`:
```
timestamp,symbol,action,entry_price,exit_price,lots,entry_date,exit_date,hold_days,pnl_percent,pnl_rp,exit_reason
```

Kolom tambahan:
- `timestamp` — waktu trade ditutup
- `action` — BUY/CL/TSL/TP1/TIME/REBALANCE

### FT-03: FormatSummary Improvement
Tambahkan stats di `FormatSummary()`:
```
📊 **FORWARD TEST — Week N/4**
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Modal Awal : Rp 1.000.000
Equity     : Rp X.XXX.XXX (+X.X%)
Realized   : Rp +XXX.XXX
Unrealized : Rp +XXX.XXX
Win Rate   : X/X (XX%)
Closed     : X trades
Max DD     : X.X%

📁 **Posisi Aktif (N):**
  • BBRI: 4500 (+1.2%, 3 hari)

📈 **Recent Closed:**
🟢 BBRI: +2.5% / Rp +XX.XXX (CL)
🔴 BMRI: -1.2% / Rp -XX.XXX (TSL)
```

---

## TASK 2: Wire S08 to Live System

### Gap Analysis

| # | Issue | Status | Fix | File |
|---|-------|--------|-----|------|
| 2.1 | `ProcessRecommendation` (15:40) masih C-BoW | ❌ | Replace dengan S08 momentum entry | `research/handler_research.go` |
| 2.2 | TIME stop check missing di live monitoring | ❌ | Add HoldDays >= 20 check | `portfolio/monitoring.go` |
| 2.3 | Trend Break exit (MA5 < MA20) missing | ❌ | Add weekly trend check di cron | `research/summary.go` |
| 2.4 | `ProcessWeeklyEntry` hanya untuk forward test | ❌ | Panggil juga untuk live portfolio | `cmd/bot/main.go` |

### 2.1: Replace ProcessRecommendation with S08 Momentum
Cron 15:40 `ProcessRecommendation()` → Ganti logic-nya:

**SEBELUM (C-BoW):**
- Sort watchlist by score (technical)
- Call Gemini untuk deep analysis
- Select top 3 dengan BUY recommendation

**SESUDAH (S08 Momentum):**
```go
// Panggil SelectWeeklyCandidates
allBars := fetchAllBars()
candidates := signals.SelectWeeklyCandidates(allBars, config.Watchlist, today, 4)

// Filter volatility
candidates = filterByVolatility(candidates, 0.06)

// Equal weight allocation
alloc := config.RemainingCapital() / len(candidates)
for _, c := range candidates {
    openPosition(symbol, c.Close, alloc)
}
```

### 2.2: Add TIME Stop to Live Monitoring
Di `portfolio/monitoring.go` — tambahkan check di loop monitoring:

```go
// TIME STOP: Exit if hold days >= 20
if pos.HoldDays >= 20 {
    // Trigger sell notification
    notifyAndLog("⏰ TIME STOP — Saham sudah 20 hari, jual semua")
}
```

**Note:** Ini akan membutuhkan update `TradingPlan` struct untuk tambahkan `HoldDays` field.

### 2.3: Add Trend Break Exit
Di `research/summary.go` atau cron weekly yang baru — cek setiap Jumat:

```go
for sym := range config.MyStocks {
    bars := market.GetBars(sym, "1mo")
    if signals.TrendBroken(bars, time.Now()) {
        notifyAndLog(fmt.Sprintf("📉 TREND BREAK — %s MA5 < MA20, exit all", sym))
    }
}
```

### 2.4: Wire ProcessWeeklyEntry for Live
Di `cmd/bot/main.go` — tambahkan pemanggilan untuk live portfolio:

```go
// Existing: Forward test weekly entry
go forwardtest.ProcessWeeklyEntry()

// NEW: Live portfolio weekly entry (same logic, different state)
go portfolio.ProcessWeeklyEntryLive()
```

---

## Cron Schedule Update (Asia/Jakarta)

| Time | Function | Action |
|------|----------|--------|
| 08:45 | `ProcessPortfolioEvaluation()` | Evaluasi TSL & CL pagi |
| 12:30 | `ProcessPortfolioEvaluation()` | Evaluasi siang |
| **16:20** | `EvaluateForwardTest()` + `EvaluateLivePositions()` | Cek exit semua posisi |
| **Jumat 18:05** | `ProcessWeeklyEntry()` | Entry sinyal baru (Forward Test + Live) |
| **Jumat 16:20** | `CheckTrendBreak()` | Cek MA5 < MA20 untuk semua posisi |

---

## Deliverables

### Forward Test Package
- [ ] FT-01: Max DD tracking in `forwardtest.go`
- [ ] FT-02: `forwardtest/logger.go` — CSV logging
- [ ] FT-03: `FormatSummary()` with Max DD stats

### Live S08 Migration
- [ ] 2.1: Replace `ProcessRecommendation` with S08 momentum
- [ ] 2.2: Add TIME stop (HoldDays >= 20) to monitoring
- [ ] 2.3: Add Trend Break check in weekly cron
- [ ] 2.4: Wire `ProcessWeeklyEntry` for live portfolio

### Testing
- [ ] `go build ./...` — verify no compile errors
- [ ] Test forward test summary output
- [ ] Verify live monitoring picks up S08 rules

---

## Constraints
- MUST pakai `config.DataMutex` setiap akses `MyStocks` atau `PendingOrders`
- MUST NOT ubah struct di `models/types.go` tanpa migrasi data
- MUST NOT hardcode TP/CL/TSL — gunakan `rules.Default`
- Forward test state TIDAK mempengaruhi live portfolio (`stocks.json`)
