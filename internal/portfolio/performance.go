package portfolio

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"learn-go/internal/config"
	"learn-go/internal/utils"
)

func ProcessPerformanceCommand(bot *tgbotapi.BotAPI) {
	file, err := os.Open("trade_history.csv")
	if err != nil {
		utils.SendSimpleMessage(bot, "⚠️ Belum ada data history trading.")
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil || len(records) <= 1 {
		utils.SendSimpleMessage(bot, "⚠️ Data history trading kosong.")
		return
	}

	var totalTrades, wins, losses int
	var totalWinPercent, totalLossPercent float64
	var totalNetRupiah float64

	// Map untuk mencatat modal per transaksi agar bisa dicocokkan saat SELL
	// Key: Symbol-Lot (Sederhana untuk matching)
	buyData := make(map[string]float64)

	type StratStat struct {
		Wins   int
		Losses int
		Profit float64
	}
	strategyStats := make(map[string]*StratStat)

	for _, row := range records[1:] {
		if len(row) < 8 { continue }

		action := row[1]
		symbol := row[2]
		price, _ := strconv.ParseFloat(row[3], 64)
		lots, _ := strconv.Atoi(row[4])
		strategy := row[5]
		pnlStr := row[6]

		key := fmt.Sprintf("%s-%d", symbol, lots)

		if action == "BUY" {
			// Hitung modal asli (Harga * Lot * 100 * (1 + Fee Beli))
			modal := price * float64(lots) * 100 * (1 + config.BuyFee)
			buyData[key] = modal
		} else if action == "SELL" {
			pnl, _ := strconv.ParseFloat(pnlStr, 64)
			
			// Hitung hasil penjualan bersih (Harga * Lot * 100 * (1 - Fee Jual))
			netSales := price * float64(lots) * 100 * (1 - config.SellFee)
			
			// Ambil modal belinya
			modal, exists := buyData[key]
			rupiahPNL := 0.0
			if exists {
				rupiahPNL = netSales - modal
				totalNetRupiah += rupiahPNL
				delete(buyData, key) // Hapus agar tidak dobel jika ada trade baru
			}

			if _, exists := strategyStats[strategy]; !exists {
				strategyStats[strategy] = &StratStat{}
			}

			totalTrades++
			strategyStats[strategy].Profit += rupiahPNL
			if pnl > 0 {
				wins++
				totalWinPercent += pnl
				strategyStats[strategy].Wins++
			} else {
				losses++
				totalLossPercent += pnl
				strategyStats[strategy].Losses++
			}
		}
	}

	if totalTrades == 0 {
		utils.SendSimpleMessage(bot, "ℹ️ Belum ada posisi trading yang ditutup.")
		return
	}

	winRate := (float64(wins) / float64(totalTrades)) * 100
	avgWin := 0.0
	if wins > 0 { avgWin = totalWinPercent / float64(wins) }
	avgLoss := 0.0
	if losses > 0 { avgLoss = totalLossPercent / float64(losses) }
	rrRatio := 0.0
	if avgLoss != 0 { rrRatio = avgWin / math.Abs(avgLoss) }

	var sb strings.Builder
	sb.WriteString("📊 **LAPORAN PERFORMA KEUANGAN** 📊\n\n")
	sb.WriteString(fmt.Sprintf("Total Trade: **%d kali**\n", totalTrades))
	sb.WriteString(fmt.Sprintf("✅ Win: %d | ❌ Loss: %d (WR: %.1f%%)\n\n", wins, losses, winRate))

	// Tampilkan Nominal Rupiah
	pnlEmoji := "💰"
	pnlLabel := "Total Profit Bersih"
	if totalNetRupiah < 0 {
		pnlEmoji = "💸"
		pnlLabel = "Total Rugi Bersih"
	}
	sb.WriteString(fmt.Sprintf("%s **%s:**\n`%s`\n\n", pnlEmoji, pnlLabel, utils.FormatRupiah(totalNetRupiah)))

	sb.WriteString(fmt.Sprintf("📈 Avg Win: `+%.2f%%`\n", avgWin))
	sb.WriteString(fmt.Sprintf("📉 Avg Loss: `%.2f%%`\n", avgLoss))
	sb.WriteString(fmt.Sprintf("⚖️ RR Ratio: `1 : %.2f`\n\n", rrRatio))

	sb.WriteString("🏆 **Breakdown Strategi (Net Rupiah):**\n")
	for strat, st := range strategyStats {
		sb.WriteString(fmt.Sprintf("🔸 **%s**: %s (WR: %.0f%%)\n", strat, utils.FormatRupiah(st.Profit), (float64(st.Wins)/float64(st.Wins+st.Losses))*100))
	}

	sb.WriteString("\n_Catatan: Perhitungan sudah termasuk estimasi pajak Bibit (0.15% Beli, 0.25% Jual)._")

	utils.SendMarkdownMessage(bot, sb.String())
}