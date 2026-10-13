# SPEC-09: Minimum Holding Period Before Rebalance

**Status:** ❌ REJECTED  
**Priority:** Quick Win 🚀 (High Impact, Low Effort)  
**Actual Effort:** 2 jam  
**Outcome:** Strategi ditolak setelah backtest — 4 dari 5 kriteria gagal

---

## Problem Statement

Backtest S08-1 (Weekly Momentum Rotation) menunjukkan:
- **944 dari 2266 trades (41.7%)** exit dengan reason `REBAL` (portfolio rotation)
- Mayoritas REBAL terjadi dalam **0-2 hari hold period** — terlalu cepat
- Trading cost 0.4% per roundtrip × 944 = **cost drag sangat besar**
- Momentum belum sempat develop karena posisi langsung di-rotate

**Hypothesis:**
Tambahkan minimum holding period 2 hari sebelum rotasi → kurangi churn → tingkatkan net expectancy.

---

## Implementation

### Code Changes

**File:** `internal/backtest/engine_weekly.go`

```go
const MinHoldDaysBeforeRebal = 2

// Di rebalancing logic:
for sym := range positions {
    pos := positions[sym]
    trendBroken := signals.TrendBroken(allBars[sym], day)
    noLongerPicked := !keep[sym]
    
    if trendBroken {
        toClose = append(toClose, sym)
    } else if noLongerPicked {
        if pos.HoldDays >= MinHoldDaysBeforeRebal {
            toClose = append(toClose, sym)
        } else {
            log.Printf("Skipping REBAL for %s: hold_days=%d < min=%d", 
                sym, pos.HoldDays, MinHoldDaysBeforeRebal)
        }
    }
}
```

**Key logic:**
- `TREND_BROKEN` exits bypass min-hold (safety mechanism)
- Pure rotation (`noLongerPicked`) must wait 2 trading days
- HoldDays counter increments daily, checked on weekly rebalance

---

## Backtest Results

### Baseline (S08-1) vs V2 (MinHold=2)

| Metrik | Baseline | V2 (MinHold=2) | Delta | Target |
|--------|----------|----------------|-------|--------|
| **Total trades** | 2266 | 1566 | -700 (-30.9%) | - |
| **REBAL count** | 944 (41.7%) | 441 (28.2%) | -503 (-53.3%) | < 650 ✅ |
| **Avg hold days** | 1.6 | 1.7 | +0.1 | > 3.5 ❌ |
| **OOS expectancy** | 1.34% | 1.28% | -0.06% | > 1.6% ❌ |
| **Win rate** | 62.8% | 58.6% | -4.2% | >= 63.9% ❌ |
| **Max DD** | 23.43% | 26.18% | +2.75% | <= 24% ❌ |

### Success Criteria (K1-K5)

| ID | Kriteria | Threshold | Hasil | Status |
|----|----------|-----------|-------|--------|
| K1 | REBAL count turun | < 650 trades (~28%) | 441 (28.2%) | ✅ PASS |
| K2 | Avg hold days naik | > 3.5 hari | 1.7 hari | ❌ FAIL |
| K3 | OOS expectancy naik | > 1.6% | 1.28% | ❌ FAIL |
| K4 | Win rate tetap/naik | >= 63.9% | 58.6% | ❌ FAIL |
| K5 | Max DD tidak naik | <= 24% | 26.18% | ❌ FAIL |

**Verdict:** 1 PASS, 4 FAIL → **STRATEGI DITOLAK** per spec rules.

---

## Root Cause Analysis

### Why Did It Fail?

1. **Trade-off terekspos:**
   - ✅ Fewer trades (1566 vs 2266) = lower transaction volume
   - ❌ Holding dying momentum = lower expectancy, win rate, higher DD

2. **Baseline REBAL exits were profitable:**
   - Avg PNL per REBAL: **+0.71%** (baseline)
   - Avg PNL per REBAL: **+0.62%** (V2)
   - REBAL wasn't the problem — it was a profitable exit mechanism!

3. **Weekly momentum works BECAUSE of rapid rotation:**
   - When top-3 ranking changes, it signals momentum shift
   - Forcing 2-day hold = fighting the trend = holding losers
   - Strategy edge is in the ranking, not in holding duration

4. **Other exit rules still fire early:**
   - CL: 444 trades (avg -3.44% PNL)
   - TP: 623 trades (avg +5.04% PNL)
   - TIME: 1 trade
   - Even with min-hold constraint, avg hold only 1.7 days because CL/TP dominate

5. **Avg hold barely moved (1.6 → 1.7 days):**
   - Min-hold only affects REBAL candidates
   - Positions still exit via CL/TP/TREND_BROKEN before reaching 2 days
   - Need to address ALL exit paths, not just REBAL, to meaningfully increase hold period

---

## Lessons Learned

### ❌ False Hypothesis
"High REBAL count = waste" is **WRONG**. REBAL at +0.71% avg PNL was a feature, not a bug.

### ✅ True Insights
1. **Weekly momentum is a high-turnover strategy by design** — edge comes from chasing momentum shifts, not from holding
2. **41.7% REBAL rate is acceptable** if profitable per trade
3. **Cost drag is real BUT** baseline already beat all baselines (random, DCA, buy-hold) despite 944 REBAL exits
4. **To increase hold period meaningfully:** must address CL/TP rules, not just REBAL logic

### 🎯 Better Approaches (for future SPECs)
Instead of constraining rotation, improve signal quality:
- **SPEC-10 (Volume Confirmation):** reduce false momentum signals → fewer bad entries → fewer CL exits → naturally longer holds
- **SPEC-11 (Volatility Filter):** avoid choppy/noisy entries → positions have room to develop → fewer whipsaws

---

## Conclusion

Min holding period constraint successfully reduced REBAL churn (-53%) but destroyed strategy performance across 4 key metrics. **The hypothesis that reducing REBAL would improve returns was invalidated.**

Weekly momentum rotation requires rapid rebalancing to capture momentum shifts. Any constraint that slows rotation degrades edge.

**Decision:** Revert changes, keep baseline S08-1 as-is, focus optimization efforts on signal quality (SPEC-10, SPEC-11) instead of execution constraints.

---

## Reverting Changes

```bash
git checkout internal/backtest/engine_weekly.go
```

File reverted to baseline (no min-hold constraint).
