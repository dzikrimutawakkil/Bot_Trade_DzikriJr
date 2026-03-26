package portfolio

import (
	"fmt"
	"strconv"
	"strings"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"learn-go/internal/market"
    "learn-go/internal/storage"
	"learn-go/internal/models"
	"learn-go/internal/config"
	"learn-go/internal/utils"
)

func ProcessBuyCommand(bot *tgbotapi.BotAPI, args []string) {
	if len(args) < 4 {
		utils.SendSimpleMessage(bot, "❌ Format salah! Gunakan: `/buy [KODE] [HARGA] [LOT]`")
		return
	}

	symbol := strings.ToUpper(args[1])
	entry, _ := strconv.ParseFloat(args[2], 64)
	lots, _ := strconv.Atoi(args[3])

	// Hitung batas TSL awal (berdasarkan harga beli saat ini)
	initialTSL := entry * (1 - config.TrailingStopPercent)

	plan := models.TradingPlan{
		Symbol:       symbol,
		EntryPrice:   entry,
		TakeProfit:   entry * (1 + config.TPPercent),
		CutLoss:      entry * (1 - config.CLPercent),
		HighestPrice: entry, // Inisialisasi awal pucuk = harga beli
		Lots:         lots,
	}
	config.MyStocks[symbol] = plan
	storage.SaveData()

	// Hitung modal asli yang terpotong di RDN (termasuk Fee Beli Bibit 0.15%)
	totalModal := entry * float64(lots) * 100 * (1 + config.BuyFee)

	response := fmt.Sprintf("✅ **%s BERHASIL DIBELI!**\n\n"+
		"🛒 **Lot:** %d\n"+
		"💸 **Modal Terpakai:** %s _(Termasuk Fee 0.15%%)_\n\n"+
		"🎯 **Target Profit:** %s\n"+
		"🛡️ **Trailing Stop Awal:** %s\n"+
		"🩸 **Batas Cut Loss:** %s\n\n"+
		"_Bot akan otomatis mengawal saham ini! 🚀_",
		symbol, lots, utils.FormatRupiah(totalModal), 
		utils.FormatRupiah(plan.TakeProfit), 
		utils.FormatRupiah(initialTSL), 
		utils.FormatRupiah(plan.CutLoss))
		
	utils.SendMarkdownMessage(bot, response)
}

// Logika /sell yang tadinya numpuk di dalam loop, sekarang punya fungsi sendiri
func ProcessSellCommand(bot *tgbotapi.BotAPI, args []string) {
	if len(args) != 2 {
		utils.SendSimpleMessage(bot, "❌ Format salah! Gunakan: `/sell [KODE]`")
		return
	}
	symbol := strings.ToUpper(args[1])
	if _, ada := config.MyStocks[symbol]; ada {
		delete(config.MyStocks, symbol)
		storage.SaveData()
		utils.SendSimpleMessage(bot, fmt.Sprintf("✅ Pantauan %s dihentikan.", symbol))
	} else {
		utils.SendSimpleMessage(bot, "❌ Saham tidak ditemukan.")
	}
}

func ProcessStatusCommand(bot *tgbotapi.BotAPI) {
	if len(config.MyStocks) == 0 {
		utils.SendSimpleMessage(bot, "Belum ada saham yang dipantau.")
		return
	}

	var sb strings.Builder
	sb.WriteString("📋 **Status Portofolio (Net Bersih):**\n\n")

	for _, plan := range config.MyStocks {
		yahooPrice := market.GetLivePrice(plan.Symbol)
		googlePrice := market.GetGooglePrice(plan.Symbol)

		sourceMarker := "[G]"
		if googlePrice == 0 {
			googlePrice = yahooPrice
			sourceMarker = "[Y-fallback]"
		}

		// 1. Hitung Persentase Net PNL (Dipotong Fee)
		perfYahoo := utils.CalculateNetPNL(plan.EntryPrice, yahooPrice, config.BuyFee, config.SellFee)
		perfGoogle := utils.CalculateNetPNL(plan.EntryPrice, googlePrice, config.BuyFee, config.SellFee)

		// 2. Hitung Total Nominal (Rupiah) Bersih
		totalBuyValue := plan.EntryPrice * (1 + config.BuyFee) * float64(plan.Lots) * 100
		totalSellValue := googlePrice * (1 - config.SellFee) * float64(plan.Lots) * 100
		totalPNL := totalSellValue - totalBuyValue

		// 3. Hitung Batas Trailing Stop Saat Ini
		tslLimit := plan.HighestPrice * (1 - config.TrailingStopPercent)

		trendEmoji := "📈"
		if totalPNL < 0 {
			trendEmoji = "📉"
		}

		sb.WriteString(fmt.Sprintf("🔹 **%s** (%d Lot)\n", plan.Symbol, plan.Lots))
		sb.WriteString(fmt.Sprintf("   Avg Beli: %s\n", utils.FormatRupiah(plan.EntryPrice)))
		sb.WriteString(fmt.Sprintf("   [Y] Now : %s (%.2f%%)\n", utils.FormatRupiah(yahooPrice), perfYahoo))
		sb.WriteString(fmt.Sprintf("   %s Now : %s (%.2f%%)\n", sourceMarker, utils.FormatRupiah(googlePrice), perfGoogle))
		
		// Info Pucuk & TSL (Sangat penting buat Swing Trader!)
		sb.WriteString(fmt.Sprintf("   🏔️ Pucuk : %s\n", utils.FormatRupiah(plan.HighestPrice)))
		sb.WriteString(fmt.Sprintf("   🛡️ TSL   : %s\n", utils.FormatRupiah(tslLimit)))

		pnlLabel := "Cuan Bersih"
		if totalPNL < 0 {
			pnlLabel = "Rugi Bersih"
		}
		// Gunakan math.Abs supaya tanda minusnya tidak dobel saat format Rupiah
		sb.WriteString(fmt.Sprintf("   👉 **%s: %s %s**\n\n", pnlLabel, utils.FormatRupiah(totalPNL), trendEmoji))
	}

	sb.WriteString("_ket: PNL sudah dipotong pajak Bibit (Beli 0.15%, Jual 0.25%)._")
	utils.SendMarkdownMessage(bot, sb.String())
}

// Logika /reset punya fungsi sendiri
func ProcessResetCommand(bot *tgbotapi.BotAPI) {
	config.MyStocks = make(map[string]models.TradingPlan)
	storage.SaveData()
	utils.SendSimpleMessage(bot, "🧹 Semua pantauan dihapus!")
}