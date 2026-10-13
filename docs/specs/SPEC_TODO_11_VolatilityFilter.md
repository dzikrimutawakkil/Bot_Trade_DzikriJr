# SPEC-11: Intraday Volatility Filter

**Status:** TODO  
**Priority:** Quick Win 🚀 (High Impact, Low Effort)  
**Estimated Effort:** 2-3 jam  
**Target:** Mengurangi choppy entries, meningkatkan avg hold period dari ~2 hari → 4-5 hari, menurunkan trading cost impact

---

## Problem Statement

Backtest S08-1 (Weekly Momentum Rotation) menunjukkan:
- **Avg hold period hanya ~2 hari** (mayoritas 0-3 hari)
- Banyak entry-exit same day atau next day → **trading cost 0.4% per roundtrip sangat signifikan**
- Sample trades menunjukkan entry di saham dengan **high intraday volatility** (choppy price action) → hit CL cepat atau REBAL karena momentum tidak sustainable

**Root Cause:**
- Signal momentum tidak filter berdasarkan **intraday range** (high - low)
- Saham dengan daily range >5-7% cenderung:
  - Gap up pagi → profit-taking siang → close flat/negative
  - Momentum artificial (pump-dump pattern)
  - High noise, low signal quality

**Impact:**
- Trading cost 0.4% × high frequency trades = **net expectancy tergerus**
- Hold period pendek → tidak cukup waktu untuk TP materialize
- False momentum karena volatility spike, bukan genuine trend

---

## Success Criteria

**Kriteria Lolos (K1-K5):**

| ID | Kriteria | Threshold | Measurement |
|----|----------|-----------|-------------|
| K1 | Avg hold days naik | > 4.0 hari (dari ~2 hari) | Mean(hold_trading_days) |
| K2 | OOS expectancy naik | > 1.6% (dari 1.34%) | OOS period mean PnL% |
| K3 | CL rate turun | < 18% (dari 20%) | Count(CL) / Total trades |
| K4 | Total trades tidak turun drastis | > 1900 trades (toleransi -15%) | Total row di trades.csv |
| K5 | Max DD tidak naik | <= 24% | Max drawdown pct |

**Jika K1-K5 semua passed → LOLOS → merge ke main.**  
**Jika K1 failed (avg hold tidak naik) → filter tidak efektif.**  
**Jika K4 failed (trades < 1900) → threshold terlalu ketat.**

---

## Technical Design

### 1. New Filter di Signal Generation

File: `internal/signals/momentum.go`

**New constant:**
```go
const (
    MaxIntradayVolatilityPct = 5.0  // Max intraday range = 5%
)
```

**Modify `GenerateWeeklyMomentumSignals()`:**
```go
func GenerateWeeklyMomentumSignals(date time.Time, watchlist []string) []Signal {
    signals := []Signal{}
    
    for _, symbol := range watchlist {
        data, err := market.GetHistoricalPrices(symbol, startDate, date, false)
        if err != nil || len(data) < 20 {
            continue
        }
        
        // Weekly return check
        weeklyReturn := CalculateWeeklyReturn(data)
        if weeklyReturn <= 0 {
            continue
        }
        
        // Volume confirmation (from SPEC-10)
        todayVolume := data[len(data)-1].Volume
        avgVolume20 := CalculateAvgVolume(data, 20)
        if todayVolume < MinVolumeMultiplier * avgVolume20 {
            continue
        }
        
        // NEW: Intraday volatility filter
        today := data[len(data)-1]
        intradayRange := (today.High - today.Low) / today.Low * 100.0
        
        if intradayRange > MaxIntradayVolatilityPct {
            log.Printf("SKIP %s: intraday vol too high (%.2f%% > %.2f%%)", 
                symbol, intradayRange, MaxIntradayVolatilityPct)
            continue
        }
        
        signals = append(signals, Signal{
            Symbol:        symbol,
            Score:         weeklyReturn,
            Date:          date,
            Volume:        todayVolume,
            AvgVolume:     avgVolume20,
            IntradayRange: intradayRange,  // For logging
        })
    }
    
    sort.Slice(signals, func(i, j int) bool {
        return signals[i].Score > signals[j].Score
    })
    
    return signals
}
```

### 2. Signal Struct Update

```go
type Signal struct {
    Symbol        string
    Score         float64    // Weekly return pct
    Date          time.Time
    Volume        float64
    AvgVolume     float64
    IntradayRange float64    // (High - Low) / Low * 100
}
```

### 3. Rationale: Why 5%?

| Intraday Range | Interpretation |
|----------------|----------------|
| < 3% | Low volatility, stable trend (ideal for momentum) |
| 3-5% | Normal IDX volatility (acceptable) |
| 5-7% | High volatility, choppy (filter this) |
| > 7% | Extreme volatility, pump-dump pattern (definitely filter) |

**Starting point: 5%** adalah sweet spot antara filter efektif vs tidak terlalu ketat.

**Jika K4 failed (trades < 1900):**
- Naikkan threshold ke 6% atau 7%
- Re-run backtest

**Jika K1 failed (avg hold tidak naik):**
- Turunkan threshold ke 4% (lebih selektif)

### 4. Data Availability Check

Yahoo Finance API sudah return field `High` dan `Low` per candle daily.

**Test di existing code:**
```bash
# Quick check apakah High/Low tersedia
go run cmd/backtest/main.go -strategy momentum -out test_vol -limit 10
# Inspect trades.csv, pastikan tidak ada error fetch OHLCV
```

---

## Implementation Checklist

**Phase 1: Code Changes**
- [ ] Add constant `MaxIntradayVolatilityPct = 5.0` di `internal/signals/momentum.go`
- [ ] Add intraday range calculation di `GenerateWeeklyMomentumSignals()`
- [ ] Add field `IntradayRange` ke `Signal` struct (optional, for logging)
- [ ] Add debug logging untuk skipped symbols

**Phase 2: Testing**
- [ ] Run backtest: `go run cmd/backtest/main.go -strategy momentum -out backtest_out_momentum_vol_filter`
- [ ] Hitung metrics K1-K5:
  ```bash
  # K1: Avg hold days
  awk -F',' 'NR>1 {sum+=$12; count++} END {printf "Avg hold: %.2f days\n", sum/count}' trades.csv
  
  # K2: OOS expectancy
  awk -F',' 'NR>1 && $1>="2025-01-01" {sum+=$9; count++} END {printf "OOS exp: %.4f%%\n", sum/count}' trades.csv
  
  # K3: CL rate
  awk -F',' 'NR>1 {total++; if($6=="CL") cl++} END {printf "CL rate: %.2f%%\n", cl/total*100}' trades.csv
  
  # K4: Total trades
  wc -l trades.csv
  
  # K5: Max DD
  awk -F',' 'NR>1 {if($3>max) max=$3} END {printf "Max DD: %.2f%%\n", max}' equity.csv
  ```

**Phase 3: Kombinasi dengan SPEC-09 & SPEC-10**
- [ ] Jika SPEC-09 & SPEC-10 sudah implemented, test kombinasi ketiga filter:
  - Min holding period (SPEC-09)
  - Volume confirmation (SPEC-10)
  - Volatility filter (SPEC-11)
- [ ] Expected combined impact:
  - **Expectancy: 1.34% → 2.0%+**
  - **Win rate: 63.9% → 70%+**
  - **Avg hold: 2 days → 5 days**
  - **Max DD: 23.43% → ~20%**

**Phase 4: Documentation**
- [ ] Pindahkan file ini ke `docs/specs/done/SPEC_DONE_11_VolatilityFilter.md`
- [ ] Update `docs/specs/00_INDEX.md`
- [ ] Commit: `feat(signals): add intraday volatility filter for stable momentum`

---

## Risk & Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| 5% threshold terlalu ketat | Total trades turun drastis (K4 fail) | Naikkan ke 6% atau 7% |
| 5% threshold terlalu longgar | Avg hold tidak naik (K1 fail) | Turunkan ke 4% atau 3.5% |
| High/Low data tidak akurat | False filtering | Spot-check 10 sample trades dengan volatility tinggi |
| Filter bias ke large cap | Small cap saham ter-reject semua | Post-backtest analysis: distribution by market cap |

---

## Theory Behind This Filter

**Volatility ≠ Momentum:**
- **Momentum** = directional price movement with follow-through (sustainable trend)
- **Volatility** = large price swings without clear direction (noise)

**High intraday volatility indicates:**
- Pump-and-dump pattern (retail speculation)
- News-driven spike (temporary, not sustainable)
- Thin liquidity (easy to manipulate)

**Low intraday volatility indicates:**
- Institutional accumulation (steady buying)
- Genuine demand without panic (smooth trend)
- Lower probability of intraday reversal

**Academic support:**
- "Volatility is the enemy of trend followers" — systematic trading literature
- Mean reversion tendency in high volatility stocks

---

## Integration with Other Filters

Filter ini **stacks** dengan SPEC-09 & SPEC-10:

| Filter | Purpose | Impact |
|--------|---------|--------|
| SPEC-09: Min holding | Reduce churn | REBAL down 36% |
| SPEC-10: Volume confirm | Reduce false momentum | CL down 23% |
| SPEC-11: Volatility filter | Increase hold period | Avg hold +100% |

**Combined effect (estimated):**
- Baseline: 2266 trades, 1.34% exp, 63.9% WR, 2d hold
- **Post-filters: ~1800 trades, 2.0% exp, 70% WR, 5d hold** 🚀

---

## Notes

- Filter ini cocok untuk **trend-following strategy** seperti momentum rotation
- Tidak cocok untuk **mean-reversion strategy** (volatility spike justru entry signal)
- Future enhancement: adaptive threshold based on IHSG volatility (VIX-like indicator)
- Bisa di-extend ke "multi-day volatility" (avg intraday range 5 hari) untuk lebih robust
