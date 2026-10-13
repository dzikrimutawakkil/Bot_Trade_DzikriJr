# ACTIVE SPEC: 08-1 Momentum Weekly Rotation

**Nama:** Weekly Momentum Rotation Backtester
**Spec source:** `docs/specs/SPEC_TODO_08_1_MomentumWeekly.md`
**Status:** 🔄 In Progress
**Target:** Sistem akumulasi DCA 1jt/bulan outperform passive holding

---

## Goal

Buat backtester untuk strategi **Weekly Momentum Rotation** dengan hold 1-4 minggu. Compatible dengan monthly DCA Rp 1.000.000.

---

## Core Logic

### Signal: Momentum Weekly Rotation
```
Senin setiap minggu:
1. Hitung MA5, MA20, MA50 untuk 30 saham watchlist
2. FILTER: MA5 > MA20 AND MA20 > MA50 (uptrend)
3. SORT: (Close - MA5) / MA5 descending
4. SELECT: top 3-4 saham
5. WEIGHT: equal weight
6. ENTRY: open price × 1.001 slippage
```

### Exit (SAMA untuk semua — fokus di entry):
- Cut Loss: -3% fixed
- Take Profit: +5%
- Time Stop: 20 hari bursa
- Rebalance mingguan: exit posisi broken trend (MA5 < MA20)

### DCA Mode:
- Setiap bulan: inject Rp 1.000.000 fresh capital
- Max 4 posisi aktif
- Idle cash di-reinvest saat replace posisi

---

## Parameter Backtest

| Flag | Value |
|------|-------|
| --years | 5 |
| --capital | 1000000 |
| --dca-monthly | 1000000 |
| --max-positions | 4 |
| --slippage-pct | 0.1 |
| --seed | 42 |
| --strategy | momentum-weekly |
| --out | backtest_out_momentum |

**OOS: 2024-2026 (2 tahun terakhir)**

---

## Criteria Lolos

| # | Kriteria | Threshold |
|---|----------|-----------|
| K1 | Total trades | ≥ 100 |
| K2 | **Expectancy OOS net** | **> 0%** ← PRIMARY |
| K3 | Win rate OOS | > 45% |
| K4 | R1 (slippage 0.3%) | expectancy OOS > 0 |
| K5 | Max DD | < 50% |
| K6 | Konsisten | ≥ 2/3 tahun OOS expectancy > 0 |
| K7 | Beat Baseline | > random3 AND > DCA naif |

---

## Files to Create/Modify

1. **`internal/signals/momentum_weekly.go`** — signal logic (MA5>MA20>MA50 filter, sort by momentum, select top-3/4)
2. **`internal/signals/indicators.go`** — helper: SMA(data, period, idx), UptrendFilter(bars, lookback)
3. **`internal/backtest/engine.go`** — handle DCA monthly mode, weekly rebalance, trend-break exit
4. **`internal/backtest/equity.go`** — track equity per day including DCA injections
5. **`internal/backtest/baseline.go`** — random3, DCA-naif, buy&hold baselines
6. **`internal/backtest/report.go`** — generate report.md + compare.md with all criteria
7. **`cmd/backtest/main.go`** — flags: `--strategy momentum-weekly --dca-monthly 1000000`

---

## Acceptance Criteria

- [ ] `go build ./... && go vet ./... && go test ./...` bersih tanpa `.env`
- [ ] `go run ./cmd/backtest --strategy momentum-weekly --years 5 --capital 1000000 --dca-monthly 1000000 --out backtest_out_momentum` menghasilkan:
  - `backtest_out_momentum/trades.csv`
  - `backtest_out_momentum/equity.csv`
  - `backtest_out_momentum/report.md`
  - `backtest_out_momentum/compare.md`
- [ ] Criteria K2 (expectancy OOS > 0%) terpenuhi
- [ ] K7: outperform random3 baseline
- [ ] Deterministic (seed 42 produces same results)
- [ ] DCA monthly injection tracked correctly in equity curve

---

## Design Principles

1. **Murni fungsi** — tidak ada side effect, testable
2. **Exit rules SAMA** untuk semua strategi yang akan diuji (only entry yang beda)
3. **No lookahead** — indikator dihitung dengan data sampai hari t saja
4. **DCA tracked** — equity curve include monthly injections
5. **Batch output** — tulis CSV + MD files di akhir (bukan per trade)
