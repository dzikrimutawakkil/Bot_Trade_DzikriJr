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

	plan := models.TradingPlan{
		Symbol:     symbol,
		EntryPrice: entry,
		TakeProfit: entry * (1 + config.TPPercent),
		CutLoss:    entry * (1 - config.CLPercent),
		Lots:       lots,
	}
	config.MyStocks[symbol] = plan
	storage.SaveData()

	totalModal := entry * float64(lots) * 100
	response := fmt.Sprintf("✅ **%s Terpasang!**\nLot: %d\nModal: %s\nTP: %s | CL: %s",
		symbol, lots, utils.FormatRupiah(totalModal), utils.FormatRupiah(plan.TakeProfit), utils.FormatRupiah(plan.CutLoss))
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
	sb.WriteString("📋 **Status Portofolio (Dual-Check):**\n\n")

	for _, plan := range config.MyStocks {
		yahooPrice := market.GetLivePrice(plan.Symbol)
		googlePrice := market.GetGooglePrice(plan.Symbol)

		sourceMarker := "[G]"
		if googlePrice == 0 {
			googlePrice = yahooPrice
			sourceMarker = "[Y-fallback]"
		}

		totalPNL := (googlePrice - plan.EntryPrice) * float64(plan.Lots) * 100
		perfYahoo := ((yahooPrice - plan.EntryPrice) / plan.EntryPrice) * 100
		perfGoogle := ((googlePrice - plan.EntryPrice) / plan.EntryPrice) * 100

		trendEmoji := "📈"
		if googlePrice < plan.EntryPrice {
			trendEmoji = "📉"
		}

		sb.WriteString(fmt.Sprintf("🔹 **%s** (%d Lot)\n", plan.Symbol, plan.Lots))
		sb.WriteString(fmt.Sprintf("   Entry : %s\n", utils.FormatRupiah(plan.EntryPrice)))
		sb.WriteString(fmt.Sprintf("   [Y] Now : %s (%.2f%%)\n", utils.FormatRupiah(yahooPrice), perfYahoo))
		sb.WriteString(fmt.Sprintf("   %s Now : %s (%.2f%%)\n", sourceMarker, utils.FormatRupiah(googlePrice), perfGoogle))

		pnlLabel := "Cuan"
		if totalPNL < 0 {
			pnlLabel = "Rugi"
		}
		sb.WriteString(fmt.Sprintf("   👉 **%s: %s %s**\n\n", pnlLabel, utils.FormatRupiah(totalPNL), trendEmoji))
	}

	sb.WriteString("_ket: [Y]=Yahoo (Delay 15m), [G]=Google (Real-time)_")
	utils.SendMarkdownMessage(bot, sb.String())
}

// Logika /reset punya fungsi sendiri
func ProcessResetCommand(bot *tgbotapi.BotAPI) {
	config.MyStocks = make(map[string]models.TradingPlan)
	storage.SaveData()
	utils.SendSimpleMessage(bot, "🧹 Semua pantauan dihapus!")
}