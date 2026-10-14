# SPEC-10: Volume Confirmation Filter — ❌ REJECTED

## Problem
Current strategy (OOS): 1104 trades, WR=63.2%, 399 CL exits (-3.41% avg). Many entries are false breakouts — momentum exists but lacks volume confirmation, resulting in quick reversals.

## Solution
Add volume filter to `SelectWeeklyCandidates`: require `volume_today > 1.2x avg_volume_20d`. This filters noise entries where price moves without real buying pressure.

## Target
- **WR:** 63.2% → 68%+
- **CL exits:** 399 → <350
- **Hold period:** 1.4d → 4-5d (fewer noise trades)

## Implementation
Volume filter added to `internal/signals/momentum_weekly.go`:
```go
avgVol20 := SMA(volumes, 20, idx)
if avgVol20 == 0 || volumes[idx] < 1.2*avgVol20 {
    continue
}
```

## Backtest Results (OOS 2024-2026)

| Metric | Baseline | Volume 1.2x | Delta | Target | Pass? |
|--------|----------|-------------|-------|--------|-------|
| Trades | 1104 | 912 | -192 (-17%) | — | ✅ |
| **Win Rate** | **63.2%** | **60.7%** | **-2.5%** | >63.2% | ❌ |
| **Expectancy** | **1.32%** | **1.11%** | **-0.21%** | ≥1.32% | ❌ |
| CL Exits | 399 | 355 | -44 (-11%) | <350 | ✅ |
| Avg Hold | 1.4d | 1.1d | -0.3d | >1.4d | ❌ |
| Max DD | 25.28% | 14.98% | -10.3% | — | ✅ |

## Rejection Reason

**FAILED 3/4 criteria:**
1. ✅ CL reduced >10%
2. ❌ Win rate declined (target: improve)
3. ❌ Expectancy declined 16% (target: maintain)
4. ❌ Hold period decreased (target: increase)

**Root Cause:** Volume filter too aggressive. In IDX market, volume spikes are **lagging indicators** — they follow price breakouts rather than precede them. Weekly momentum rotation captures early movers with low volume, which are actually the profitable trades. Filtering them degraded the edge.

**Lesson Learned:** Don't fix what isn't broken. Baseline strategy (exp=1.32%, WR=63.2%) already robust. Volume confirmation doesn't add value to weekly rotation strategy in IDX context.

## Date
- Tested: 2026-10-13
- Decision: REJECTED
- Status: Code rolled back, baseline strategy preserved
