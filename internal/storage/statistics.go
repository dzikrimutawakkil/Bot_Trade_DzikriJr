// internal/storage/statistics.go
package storage

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"learn-go/internal/config"
)

const TradeHistoryFile = "trade_history.csv"

// TradeStatistics menyimpan hasil kalkulasi statistik trading
type TradeStatistics struct {
	TotalTrades         int     // Total transaksi sell yang tercatat
	WinningTrades      int     // Jumlah transaksi untung
	LosingTrades        int     // Jumlah transaksi rugi
	WinRate             float64 // Persentase win rate (0-100)
	AvgProfit           float64 // Rata-rata profit % per trade (net after fees)
	AvgLoss             float64 // Rata-rata loss % per trade (net after fees)
	TotalProfit         float64 // Total profit % (sum, net)
	TotalLoss           float64 // Total loss % (sum, net)
	NetProfit           float64 // Total net profit/loss %
	TotalProfitRp       float64 // Total profit dalam Rupiah (net after fees)
	TotalLossRp         float64 // Total loss dalam Rupiah (net after fees)
	NetProfitRp         float64 // Net profit dalam Rupiah (net)
	RiskRewardRatio     float64 // Risk/Reward Ratio (avg profit gross / avg loss gross)
	TotalGrossProfitRp  float64 // Total gross profit (sebelum fees) untuk R/R calculation
	TotalGrossLossRp    float64 // Total gross loss (sebelum fees) untuk R/R calculation
	LastUpdated         string  // Timestamp update terakhir
}

// CalculateStatistics mem-parse trade_history.csv dan hitung statistik trading
// dengan biaya transaksi sekuritas (Buy 0.15%, Sell 0.25% + PPN).
func CalculateStatistics() (*TradeStatistics, error) {
	stats := &TradeStatistics{
		LastUpdated: time.Now().Format("2006-01-02 15:04"),
	}

	file, err := os.Open(TradeHistoryFile)
	if err != nil {
		return stats, nil
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	records, err := reader.ReadAll()
	if err != nil || len(records) <= 1 {
		return stats, nil
	}

	// Baca konfigurasi fee dari config
	buyFee := config.BuyFee        // 0.15%
	sellFee := config.SellFee     // 0.25%
	sellFeeWithVAT := sellFee * 1.11 // 0.2775%

	for i, record := range records {
		if i == 0 {
			continue
		}

		// Skip separator bars
		if len(record) == 1 && (strings.Contains(record[0], "---") || strings.Contains(record[0], "===")) {
			continue
		}

		// Skip blank rows
		if len(record) == 0 || (len(record) == 1 && strings.TrimSpace(record[0]) == "") {
			continue
		}

		// Format CSV: Waktu,Aksi,Saham,Harga,Lot,Strategi,PNL (%),Catatan
		if len(record) < 7 {
			continue
		}

		aksi := strings.TrimSpace(record[1])
		hargaStr := strings.TrimSpace(record[3])
		lotStr := strings.TrimSpace(record[4])
		pnlStr := strings.TrimSpace(record[6]) // PNL % (index 6)

		harga, _ := strconv.ParseFloat(hargaStr, 64)
		lot, _ := strconv.ParseInt(lotStr, 10, 64)
		shares := float64(lot) * 100 // 1 Lot = 100 saham

		// Skip BUY transactions
		if strings.ToUpper(aksi) == "BUY" {
			continue
		}

		// Parse PnL percentage
		pnlStr = strings.TrimSuffix(pnlStr, "%")
		pnlStr = strings.ReplaceAll(pnlStr, ",", ".")
		pnl, err := strconv.ParseFloat(pnlStr, 64)
		if err != nil {
			continue
		}

		// Hitung Gross PnL dalam Rupiah
		// PnL Rp (Gross) = Harga Jual × Lot × 100 × (PNL% / 100)
		grossPnL := harga * float64(lot) * 100 * (pnl / 100)

		stats.TotalTrades++

		if pnl > 0 {
			stats.WinningTrades++
			stats.TotalProfit += pnl

			// Net Profit Rp = Gross - Buy Fee - Sell Fee (dengan PPN)
			buyCost := harga * shares * buyFee
			sellCost := harga * shares * sellFeeWithVAT
			netPnLRp := grossPnL - buyCost - sellCost
			stats.TotalProfitRp += netPnLRp

			// Gross untuk Risk/Reward
			stats.TotalGrossProfitRp += grossPnL
		} else {
			stats.LosingTrades++
			stats.TotalLoss += pnl

			// Net Loss Rp = Gross + Buy Fee + Sell Fee (dengan PPN)
			buyCost := harga * shares * buyFee
			sellCost := harga * shares * sellFeeWithVAT
			netPnLRp := grossPnL - buyCost - sellCost // grossPnL already negative
			stats.TotalLossRp += netPnLRp

			// Gross untuk Risk/Reward
			stats.TotalGrossLossRp += grossPnL
		}
	}

	// Hitung rata-rata
	if stats.WinningTrades > 0 {
		stats.AvgProfit = stats.TotalProfit / float64(stats.WinningTrades)
	}
	if stats.LosingTrades > 0 {
		stats.AvgLoss = stats.TotalLoss / float64(stats.LosingTrades)
	}

	// Hitung Win Rate
	if stats.TotalTrades > 0 {
		stats.WinRate = float64(stats.WinningTrades) / float64(stats.TotalTrades) * 100
	}

	// Hitung Net Profit
	stats.NetProfit = stats.TotalProfit + stats.TotalLoss
	stats.NetProfitRp = stats.TotalProfitRp + stats.TotalLossRp

	// Hitung Risk/Reward Ratio (gross, sebelum fees)
	if stats.TotalGrossLossRp != 0 {
		// AvgGrossProfit / AvgGrossLoss (loss is negative, so negate it)
		avgGrossLoss := stats.TotalGrossLossRp / float64(stats.LosingTrades)
		if avgGrossLoss < 0 {
			stats.RiskRewardRatio = (stats.TotalGrossProfitRp / float64(stats.WinningTrades)) / (-avgGrossLoss)
		}
	}

	return stats, nil
}

// formatRp memformat number ke format Rupiah dengan separator ribuan
// Format: "Rp 1.234.567" atau "-Rp 1.234.567" (untuk negatif)
func formatRp(amount float64) string {
	negative := amount < 0
	if negative {
		amount = -amount
	}

	// Format integer part dengan separator ribuan
	intPart := int64(amount)
	decPart := int64((amount - float64(intPart)) * 100)

	// Build string dengan reverse approach (sama seperti FormatRupiah di utils)
	intStr := strconv.FormatInt(intPart, 10)
	var formattedInt string
	for len(intStr) > 3 {
		formattedInt = "." + intStr[len(intStr)-3:] + formattedInt
		intStr = intStr[:len(intStr)-3]
	}
	formattedInt = intStr + formattedInt

	// Format: Rp X.XXX.XX (tanpa desimal jika 0) atau Rp X.XXX,XX
	var result string
	if decPart > 0 {
		result = fmt.Sprintf("Rp %s,%02d", formattedInt, decPart)
	} else {
		result = "Rp " + formattedInt
	}

	if negative {
		return "-Rp " + formattedInt
	}
	return result
}

// FormatStatsMessage membuat pesan statistik untuk Telegram
func FormatStatsMessage(stats *TradeStatistics) string {
	// Handle empty stats
	if stats.TotalTrades == 0 {
		return "📊Belum ada data trading untuk dianalisa.\nMulai trading dulu, Bos!"
	}

	// Format percentages
	avgProfitStr := fmt.Sprintf("+%.2f%%", stats.AvgProfit)
	avgLossStr := fmt.Sprintf("%.2f%%", stats.AvgLoss)

	// Format Rp dengan prefix yang jelas
	totalProfitRpStr := formatRp(stats.TotalProfitRp)
	totalLossRpStr := formatRp(stats.TotalLossRp)
	netProfitRpStr := formatRp(stats.NetProfitRp)

	// Tambah + untuk profit Rp
	if stats.TotalProfitRp > 0 {
		totalProfitRpStr = "+" + totalProfitRpStr
	}
	if stats.NetProfitRp > 0 {
		netProfitRpStr = "+" + netProfitRpStr
	}

	// Format Risk/Reward Ratio
	riskRewardStr := "N/A"
	if stats.RiskRewardRatio > 0 {
		riskRewardStr = fmt.Sprintf("%.2f", stats.RiskRewardRatio)
	}

	// Win Rate Bar (ASCII style: 🟢🟢🟢🟢⚫⚫⚫⚫⚫⚫ 75%)
	winRateBarLen := int(stats.WinRate / 10) // 10 blocks = 100%
	winRateBar := strings.Repeat("🟢", winRateBarLen) + strings.Repeat("⚫", 10-winRateBarLen)
	winRateStr := fmt.Sprintf("%.1f%%", stats.WinRate)

	return fmt.Sprintf(`📊 *STATISTIK TRADING DZIKRIJR*
━━━━━━━━━━━━━━━
📈 *Performance Summary*
├ Total Trades : %d
├ ✅ Winning   : %d trades
└ ❌ Losing    : %d trades

🎯 *Win Rate:* %s
%s

📍 *Avg Profit:* %s
📍 *Avg Loss:*   %s

⚖️ *Risk/Reward Ratio:* %s
   _(Avg Profit ÷ Avg Loss)_

💰 *Total Profit:* %s
💸 *Total Loss:*  %s
━━━━━━━━━━━━━━━
💵 *Net Profit:*  %s
━━━━━━━━━━━━━━━
📝 _Fee: Buy 0.15%% | Sell 0.25%%+PPN_
_Updated: %s_`,
		stats.TotalTrades,
		stats.WinningTrades,
		stats.LosingTrades,
		winRateStr,
		winRateBar,
		avgProfitStr,
		avgLossStr,
		riskRewardStr,
		totalProfitRpStr,
		totalLossRpStr,
		netProfitRpStr,
		stats.LastUpdated,
	)
}