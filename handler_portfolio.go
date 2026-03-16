package main

import (
	"fmt"
	"strconv"
	"strings"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func processBuyCommand(bot *tgbotapi.BotAPI, args []string) {
	if len(args) < 4 {
		sendSimpleMessage(bot, "❌ Format salah! Gunakan: `/buy [KODE] [HARGA] [LOT]`")
		return
	}

	symbol := strings.ToUpper(args[1])
	entry, _ := strconv.ParseFloat(args[2], 64)
	lots, _ := strconv.Atoi(args[3])

	plan := TradingPlan{
		Symbol:     symbol,
		EntryPrice: entry,
		TakeProfit: entry * (1 + TPPercent),
		CutLoss:    entry * (1 - CLPercent),
		Lots:       lots,
	}
	myStocks[symbol] = plan
	saveData()

	totalModal := entry * float64(lots) * 100
	response := fmt.Sprintf("✅ **%s Terpasang!**\nLot: %d\nModal: %s\nTP: %s | CL: %s",
		symbol, lots, formatRupiah(totalModal), formatRupiah(plan.TakeProfit), formatRupiah(plan.CutLoss))
	sendMarkdownMessage(bot, response)
}

// Logika /sell yang tadinya numpuk di dalam loop, sekarang punya fungsi sendiri
func processSellCommand(bot *tgbotapi.BotAPI, args []string) {
	if len(args) != 2 {
		sendSimpleMessage(bot, "❌ Format salah! Gunakan: `/sell [KODE]`")
		return
	}
	symbol := strings.ToUpper(args[1])
	if _, ada := myStocks[symbol]; ada {
		delete(myStocks, symbol)
		saveData()
		sendSimpleMessage(bot, fmt.Sprintf("✅ Pantauan %s dihentikan.", symbol))
	} else {
		sendSimpleMessage(bot, "❌ Saham tidak ditemukan.")
	}
}

func processStatusCommand(bot *tgbotapi.BotAPI) {
	if len(myStocks) == 0 {
		sendSimpleMessage(bot, "Belum ada saham yang dipantau.")
		return
	}

	var sb strings.Builder
	sb.WriteString("📋 **Status Portofolio (Dual-Check):**\n\n")

	for _, plan := range myStocks {
		yahooPrice := getLivePrice(plan.Symbol)
		googlePrice := getGooglePrice(plan.Symbol)

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
		sb.WriteString(fmt.Sprintf("   Entry : %s\n", formatRupiah(plan.EntryPrice)))
		sb.WriteString(fmt.Sprintf("   [Y] Now : %s (%.2f%%)\n", formatRupiah(yahooPrice), perfYahoo))
		sb.WriteString(fmt.Sprintf("   %s Now : %s (%.2f%%)\n", sourceMarker, formatRupiah(googlePrice), perfGoogle))

		pnlLabel := "Cuan"
		if totalPNL < 0 {
			pnlLabel = "Rugi"
		}
		sb.WriteString(fmt.Sprintf("   👉 **%s: %s %s**\n\n", pnlLabel, formatRupiah(totalPNL), trendEmoji))
	}

	sb.WriteString("_ket: [Y]=Yahoo (Delay 15m), [G]=Google (Real-time)_")
	sendMarkdownMessage(bot, sb.String())
}

// Logika /reset punya fungsi sendiri
func processResetCommand(bot *tgbotapi.BotAPI) {
	myStocks = make(map[string]TradingPlan)
	saveData()
	sendSimpleMessage(bot, "🧹 Semua pantauan dihapus!")
}