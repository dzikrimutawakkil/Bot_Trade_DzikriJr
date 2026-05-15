# Spec: Win Rate Tracker (v2.0)
Dibuat: 2026-05-14
Update: 2026-05-15
Status: ✅ DONE

## Perubahan dari v1.0
1. **Fix `formatRp`** — perbaikan algoritma separator ribuan
2. **Biaya transaksi** — Buy 0.15%, Sell 0.25% + PPN 11%
3. **Risk/Reward Ratio** — metrik baru
4. **Win Rate Bar** — visual progress bar dengan emoji 🟢⚫
5. **Markdown formatting** — menggunakan `SendMarkdownMessage`

---

## Fitur `/stats`
Parse `trade_history.csv` dan hitung statistik trading:

### Struct `TradeStatistics`
```go
type TradeStatistics struct {
    TotalTrades         int     // Total transaksi sell
    WinningTrades       int     // Jumlah transaksi untung
    LosingTrades        int     // Jumlah transaksi rugi
    WinRate             float64 // Persentase win rate (0-100)
    AvgProfit           float64 // Rata-rata profit % (net after fees)
    AvgLoss             float64 // Rata-rata loss % (net after fees)
    TotalProfitRp       float64 // Total profit Rp (net after fees)
    TotalLossRp         float64 // Total loss Rp (net after fees)
    NetProfitRp         float64 // Net profit Rp
    RiskRewardRatio     float64 // Avg profit gross / Avg loss gross
    TotalGrossProfitRp  float64 // Gross profit untuk R/R
    TotalGrossLossRp    float64 // Gross loss untuk R/R
    LastUpdated         string  // Timestamp
}
```

### Fee Calculation
- **Buy Fee:** 0.15% (dari `config.BuyFee`)
- **Sell Fee:** 0.25% + PPN 11% = 0.2775% (dari `config.SellFee * 1.11`)
- **Net PnL = Gross PnL - Buy Cost - Sell Cost**

### Output `/stats`
```
📊 *STATISTIK TRADING DZIKRIJR*
━━━━━━━━━━━━━━━
📈 *Performance Summary*
├ Total Trades : 4
├ ✅ Winning   : 3 trades
└ ❌ Losing    : 1 trades

🎯 *Win Rate:* 75.0%
🟢🟢🟢🟢🟢🟢🟢⚫⚫⚫

📍 *Avg Profit:* +1.40%
📍 *Avg Loss:*   -2.26%

⚖️ *Risk/Reward Ratio:* 0.62
   _(Avg Profit ÷ Avg Loss)_

💰 *Total Profit:* +Rp 1.055
💸 *Total Loss:*  -Rp 593
━━━━━━━━━━━━━━━
💵 *Net Profit:*  +Rp 461
━━━━━━━━━━━━━━━
📝 _Fee: Buy 0.15% | Sell 0.25%+PPN_
_Updated: 2026-05-15 16:25_
```

---

## Files Modified
| File | Perubahan |
|------|-----------|
| `internal/storage/statistics.go` | Fix formatRp, add fees, add R/R ratio |
| `internal/telegram/handler.go` | Use SendMarkdownMessage |

## Checklist
- [x] `formatRp` menghasilkan format Rupiah yang benar
- [x] Fee Buy 0.15% dan Sell 0.25% + PPN dihitung
- [x] Risk/Reward Ratio ditampilkan
- [x] Win Rate Bar visual progress
- [x] `go build ./...` bersih