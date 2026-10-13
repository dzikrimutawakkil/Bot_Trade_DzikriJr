// internal/storage/statistics.go
package storage

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const TradeHistoryFile = "trade_history.csv"

// TradeStatistics menyimpan hasil kalkulasi statistik trading.
type TradeStatistics struct {
	TotalTrades     int     // Total transaksi sell
	WinningTrades   int     // Jumlah transaksi untung
	LosingTrades    int     // Jumlah transaksi rugi
	WinRate         float64 // Persentase win rate (0-100)
	AvgProfit       float64 // Rata-rata profit % per trade (net after fees)
	AvgLoss         float64 // Rata-rata loss % per trade (net after fees)
	TotalProfitRp   float64 // Total profit dalam Rupiah
	TotalLossRp     float64 // Total loss dalam Rupiah
	NetProfitRp     float64 // Net profit dalam Rupiah
	RiskRewardRatio float64 // avg PNLRp menang ÷ |avg PNLRp kalah| (net)
	ExpectancyRp    float64 // NetProfitRp / TotalTrades
	LastUpdated     string  // Timestamp update terakhir
}

// CalculateStatistics mem-parse trade_history.csv dan hitung statistik.
func CalculateStatistics() (*TradeStatistics, error) {
	return CalculateStatisticsFromFile(TradeHistoryFile)
}

// CalculateStatisticsFromFile membaca file CSV dan mengembalikan statistik.
// Hanya baris SELL; menang jika PNLPercent > 0, kalah jika <= 0.
// Pakai PNLRp apa adanya (sudah net setelah fee).
func CalculateStatisticsFromFile(path string) (*TradeStatistics, error) {
	stats := &TradeStatistics{
		LastUpdated: time.Now().Format("2006-01-02 15:04"),
	}

	records, err := ReadTradeHistory(path)
	if err != nil {
		return stats, nil
	}

	var winRP, lossRP []float64

	for _, rec := range records {
		if rec.Action != "SELL" {
			continue
		}

		stats.TotalTrades++

		if rec.PNLPercent > 0 {
			stats.WinningTrades++
			stats.TotalProfitRp += rec.PNLRp
			winRP = append(winRP, rec.PNLRp)
		} else {
			stats.LosingTrades++
			stats.TotalLossRp += rec.PNLRp
			lossRP = append(lossRP, rec.PNLRp)
		}
	}

	if stats.WinningTrades > 0 {
		stats.AvgProfit = stats.TotalProfitRp / float64(stats.WinningTrades)
	}
	if stats.LosingTrades > 0 {
		stats.AvgLoss = stats.TotalLossRp / float64(stats.LosingTrades)
	}

	if stats.TotalTrades > 0 {
		stats.WinRate = float64(stats.WinningTrades) / float64(stats.TotalTrades) * 100
	}

	stats.NetProfitRp = stats.TotalProfitRp + stats.TotalLossRp

	// Risk/Reward: avg PNLRp menang ÷ |avg PNLRp kalah|
	if len(lossRP) > 0 {
		avgWin := sum(winRP) / float64(len(winRP))
		avgLoss := sum(lossRP) / float64(len(lossRP)) // negative
		if avgLoss < 0 {
			stats.RiskRewardRatio = avgWin / (-avgLoss)
		}
	}

	// Expectancy Rp per trade
	if stats.TotalTrades > 0 {
		stats.ExpectancyRp = stats.NetProfitRp / float64(stats.TotalTrades)
	}

	return stats, nil
}

func sum(vals []float64) float64 {
	var s float64
	for _, v := range vals {
		s += v
	}
	return s
}

// formatRp memformat number ke format Rupiah tanpa desimal.
// Format: "Rp 1.234" atau "-Rp 1.234" (untuk negatif).
func formatRp(amount float64) string {
	negative := amount < 0
	if negative {
		amount = -amount
	}
	intPart := int64(amount)
	intStr := strconv.FormatInt(intPart, 10)
	var formatted string
	for len(intStr) > 3 {
		formatted = "." + intStr[len(intStr)-3:] + formatted
		intStr = intStr[:len(intStr)-3]
	}
	formatted = intStr + formatted
	if negative {
		return "-Rp " + formatted
	}
	return "Rp " + formatted
}

// FormatStatsMessage membuat pesan statistik untuk Telegram.
func FormatStatsMessage(stats *TradeStatistics) string {
	if stats.TotalTrades == 0 {
		return "📊Belum ada data trading untuk dianalisa.\nMulai trading dulu, Bos!"
	}

	avgProfitStr := fmt.Sprintf("+%.2f%%", stats.AvgProfit)
	avgLossStr := fmt.Sprintf("%.2f%%", stats.AvgLoss)

	totalProfitRpStr := formatRp(stats.TotalProfitRp)
	totalLossRpStr := formatRp(stats.TotalLossRp)
	netProfitRpStr := formatRp(stats.NetProfitRp)
	expectancyRpStr := formatRp(stats.ExpectancyRp)

	if stats.TotalProfitRp > 0 {
		totalProfitRpStr = "+" + totalProfitRpStr
	}
	if stats.NetProfitRp > 0 {
		netProfitRpStr = "+" + netProfitRpStr
	}
	if stats.ExpectancyRp > 0 {
		expectancyRpStr = "+" + expectancyRpStr
	}

	riskRewardStr := "N/A"
	if stats.RiskRewardRatio > 0 {
		riskRewardStr = fmt.Sprintf("%.2f", stats.RiskRewardRatio)
	}

	winRateBarLen := int(stats.WinRate / 10)
	winRateBar := strings.Repeat("🟢", winRateBarLen) + strings.Repeat("⚫", 10-winRateBarLen)

	return fmt.Sprintf(`📊 *STATISTIK TRADING DZIKRIJR*
━━━━━━━━━━━━━━━
📈 *Performance Summary*
├ Total Trades : %d
├ ✅ Winning   : %d trades
└ ❌ Losing    : %d trades

🎯 *Win Rate:* %.1f%%
%s

📍 *Avg Profit:* %s
📍 *Avg Loss:*   %s

⚖️ *Risk/Reward Ratio:* %s
   _(Avg PNLRp menang ÷ |Avg PNLRp kalah|)_

💰 *Total Profit:* %s
💸 *Total Loss:*  %s
━━━━━━━━━━━━━━━
💵 *Net Profit:*  %s
━━━━━━━━━━━━━━━
📐 *Expectancy:* %s / trade
━━━━━━━━━━━━━━━
📝 _PNL net setelah fee Buy 0.15%% | Sell 0.25%%_
_Trade = jumlah transaksi jual (parsial dihitung terpisah)_
_Updated: %s_`,
		stats.TotalTrades,
		stats.WinningTrades,
		stats.LosingTrades,
		stats.WinRate,
		winRateBar,
		avgProfitStr,
		avgLossStr,
		riskRewardStr,
		totalProfitRpStr,
		totalLossRpStr,
		netProfitRpStr,
		expectancyRpStr,
		stats.LastUpdated,
	)
}
