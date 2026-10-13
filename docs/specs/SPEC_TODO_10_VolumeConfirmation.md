# SPEC-10: Volume Confirmation Filter

**Status:** TODO  
**Priority:** Quick Win 🚀 (High Impact, Low Effort)  
**Estimated Effort:** 3-5 jam  
**Target:** Mengurangi false breakout, meningkatkan win rate dari 63.9% → ~68%, mengurangi CL dari 455 → ~350 trades

---

## Problem Statement

Backtest S08-1 (Weekly Momentum Rotation) menunjukkan:
- **455 dari 2266 trades (20%)** exit dengan reason `CL` (cut loss)
- Win rate 63.9% masih bisa lebih tinggi — artinya ada banyak **false momentum signal**
- Sample trades menunjukkan entry di saham dengan momentum tapi **tanpa volume confirmation** → price naik karena thin trading, bukan genuine demand

**Root Cause:**
- Signal momentum hanya lihat **% gain** (relative return) tanpa validasi volume
- Saham bisa naik 10% karena 1-2 lot traded (manipulasi atau low liquidity)
- Entry di "fake momentum" → langsung CL karena harga balik turun

**Impact:**
- 455 CL × -3% avg loss = **total -1365% loss** yang seharusnya bisa dihindari
- Win rate tertekan karena banyak entry buruk
- Expectancy tergerus oleh false signal

---

## Success Criteria

**Kriteria Lolos (K1-K6):**

| ID | Kriteria | Threshold | Measurement |
|----|----------|-----------|-------------|
| K1 | CL count turun | < 380 trades (~17% dari total, turun dari 20%) | Count exit_reason='CL' |
| K2 | Win rate naik | > 67% (dari 63.9%) | Win trades / Total trades |
| K3 | OOS expectancy naik | > 1.7% (dari 1.34%) | OOS period mean PnL% |
| K4 | Total trades tidak turun drastis | > 1800 trades (toleransi -20%) | Total row di trades.csv |
| K5 | Max DD tidak naik | <= 24% | Max drawdown pct |
| K6 | TP rate tetap/naik | >= 34% (dari baseline 34%) | Count exit_reason='TP' / total |

**Jika K1-K6 semua passed → LOLOS → merge ke main.**  
**Jika K4 failed (trades < 1800) → signal terlalu ketat → relax threshold.**  
**Jika K2 failed (win rate tidak naik) → filter tidak efektif → skip strategi ini.**

---

## Technical Design

### 1. New Filter di Signal Generation

File: `internal/signals/momentum.go`

**Current (simplified):**
```go
func GenerateWeeklyMomentumSignals(date time.Time, watchlist []string) []Signal {
    signals := []Signal{}
    
    for _, symbol := range watchlist {
        // Hitung weekly return
        weeklyReturn := CalculateWeeklyReturn(symbol, date)
        
        if weeklyReturn > 0 {
            signals = append(signals, Signal{
                Symbol: symbol,
                Score: weeklyReturn,
                Date: date,
            })
        }
    }
    
    // Sort by score descending
    sort.Slice(signals, func(i, j int) bool {
        return signals[i].Score > signals[j].Score
    })
    
    return signals
}
```

**New (with volume confirmation):**
```go
const (
    MinVolumeMultiplier = 1.2  // Volume harus >= 1.2x average
)

func GenerateWeeklyMomentumSignals(date time.Time, watchlist []string) []Signal {
    signals := []Signal{}
    
    for _, symbol := range watchlist {
        // Fetch data
        data, err := market.GetHistoricalPrices(symbol, startDate, date, false)
        if err != nil || len(data) < 20 {
            continue
        }
        
        // Hitung weekly return
        weeklyReturn := CalculateWeeklyReturn(data)
        if weeklyReturn <= 0 {
            continue  // Skip negative momentum
        }
        
        // NEW: Volume confirmation
        todayVolume := data[len(data)-1].Volume
        avgVolume20 := CalculateAvgVolume(data, 20)
        
        if todayVolume < MinVolumeMultiplier * avgVolume20 {
            log.Printf("SKIP %s: volume too thin (%.2f < %.2f)", 
                symbol, todayVolume, MinVolumeMultiplier*avgVolume20)
            continue  // Skip thin volume momentum
        }
        
        signals = append(signals, Signal{
            Symbol: symbol,
            Score: weeklyReturn,
            Date: date,
            Volume: todayVolume,        // Store for logging
            AvgVolume: avgVolume20,
        })
    }
    
    sort.Slice(signals, func(i, j int) bool {
        return signals[i].Score > signals[j].Score
    })
    
    return signals
}

// Helper: Calculate average volume over N days
func CalculateAvgVolume(data []models.HistoricalData, days int) float64 {
    if len(data) < days {
        days = len(data)
    }
    
    var sum float64
    for i := len(data) - days; i < len(data); i++ {
        sum += data[i].Volume
    }
    return sum / float64(days)
}
```

### 2. Signal Struct Update

File: `internal/signals/types.go` (atau di `momentum.go`)

```go
type Signal struct {
    Symbol    string
    Score     float64  // Weekly return pct
    Date      time.Time
    Volume    float64  // Today's volume (for logging/audit)
    AvgVolume float64  // 20-day avg volume
}
```

### 3. Backtest Integration

Backtest engine (`internal/backtest/engine.go`) sudah memanggil `GenerateWeeklyMomentumSignals()` — **tidak perlu ubah logic**, hanya signal yang ter-filter otomatis berkurang.

**Impact:**
- Jumlah signal per minggu turun dari ~10-15 → ~8-12 (hanya yang genuine high volume)
- Entry quality naik → CL turun, win rate naik

### 4. Configuration Tuning

**MinVolumeMultiplier = 1.2** adalah starting point konservatif.

| Threshold | Expectation |
|-----------|-------------|
| 1.2x (konservatif) | Filter ~15-20% signal, win rate +3-4% |
| 1.5x (agresif) | Filter ~30-40% signal, win rate +5-6% tapi total trades turun banyak |
| 1.0x (no filter) | Baseline S08-1 |

**Jika K4 failed (total trades < 1800):**
- Turunkan ke `MinVolumeMultiplier = 1.1` atau bahkan `1.05`
- Re-run backtest

---

## Implementation Checklist

**Phase 1: Code Changes**
- [ ] Add constant `MinVolumeMultiplier = 1.2` di `internal/signals/momentum.go`
- [ ] Implement helper `CalculateAvgVolume(data, days)` jika belum ada
- [ ] Add volume confirmation check di `GenerateWeeklyMomentumSignals()`
- [ ] Update `Signal` struct: tambah field `Volume` dan `AvgVolume` (optional, untuk logging)
- [ ] Add debug logging untuk skipped symbols (audit purpose)

**Phase 2: Testing**
- [ ] Run backtest: `go run cmd/backtest/main.go -strategy momentum -out backtest_out_momentum_vol`
- [ ] Hitung metrics K1-K6 (script sama seperti SPEC-09)
- [ ] Compare dengan baseline S08-1:
  ```bash
  # Quick comparison
  echo "Baseline CL: 455"
  awk -F',' '$6=="CL"' backtest_out_momentum_vol/trades.csv | wc -l
  
  echo "Baseline Win Rate: 63.9%"
  awk -F',' 'NR>1 {if($9>0) win++; total++} END {printf "%.2f%%\n", win/total*100}' backtest_out_momentum_vol/trades.csv
  ```

**Phase 3: Iterasi (jika perlu)**
- [ ] Jika K4 failed (trades < 1800): turunkan threshold ke 1.1x, re-run
- [ ] Jika K2 failed (win rate tidak naik): cek apakah ada bug di volume calculation
- [ ] Jika K5 failed (DD naik): kemungkinan filter terlalu ketat, miss defensive rotation

**Phase 4: Documentation**
- [ ] Pindahkan file ini ke `docs/specs/done/SPEC_DONE_10_VolumeConfirmation.md`
- [ ] Update `docs/specs/00_INDEX.md`
- [ ] Commit: `feat(signals): add volume confirmation filter for momentum strategy`

---

## Risk & Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Volume data tidak akurat dari Yahoo | False rejection of good signals | Validasi manual 5-10 sample trades dengan volume rendah |
| Threshold 1.2x terlalu ketat | Total trades turun drastis (K4 fail) | Turunkan ke 1.1x atau 1.05x |
| Threshold 1.2x terlalu longgar | CL masih banyak (K1 fail) | Naikkan ke 1.3x atau 1.5x |
| Large cap vs small cap bias | Small cap ter-filter semua | Analisis distribution by market cap post-backtest |

---

## Theory Behind This Filter

**Volume adalah "vote" dari market participants:**
- High volume + price up = **consensus bullish** (many buyers)
- Low volume + price up = **manipulasi atau thin trading** (few buyers, easy to reverse)

**Academic support:**
- "On Balance Volume" indicator (Joseph Granville, 1963)
- "Volume precedes price" — momentum dikonfirmasi oleh participation

**IDX context:**
- Banyak saham illiquid di watchlist (ESSA, TOWR, ERAA) yang prone to manipulation
- Volume filter wajib untuk avoid "trap" entry

---

## Notes

- Filter ini **orthogonal** dengan SPEC-09 (min holding) — bisa di-stack untuk efek kombinasi
- Filter ini juga akan berguna untuk **live bot** nanti (tidak hanya backtest)
- Volume data dari Yahoo Finance API field `Volume` sudah dalam satuan lot (1 lot = 100 shares untuk IDX)
- Future enhancement: tambah "volume acceleration" (hari ini > kemarin > 2 hari lalu) untuk detect breakout awal
