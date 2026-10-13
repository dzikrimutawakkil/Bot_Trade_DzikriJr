package portfolio

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"learn-go/internal/config"
	"learn-go/internal/market"
	"learn-go/internal/models"
	"learn-go/internal/pricing"
	"learn-go/internal/rules"
	"learn-go/internal/storage"
	"learn-go/internal/utils"
)

func ProcessBuyCommand(bot *tgbotapi.BotAPI, args []string) {
	if len(args) < 4 {
		utils.SendSimpleMessage(bot, "❌ Format salah! Gunakan: `/buy [KODE] [HARGA] [LOT]`\nContoh: `/buy BBCA 9000 10`\nOpsional: `/buy BBCA 9000 10 8700` (dengan Cut Loss)")
		return
	}

	symbol := strings.ToUpper(args[1])
	entry, errPrice := strconv.ParseFloat(args[2], 64)
	lots, errLot := strconv.Atoi(args[3])

	if errPrice != nil || errLot != nil || entry <= 0 || lots <= 0 {
		utils.SendSimpleMessage(bot, "❌ ERROR: Harga dan Lot harus berupa angka dan lebih besar dari 0!")
		return
	}

	var cutLossArg float64 = -1
	if len(args) >= 5 {
		cl, err := strconv.ParseFloat(args[4], 64)
		if err != nil || cl <= 0 || cl >= entry {
			utils.SendSimpleMessage(bot, "❌ CL harus angka positif dan lebih kecil dari harga entry!")
			return
		}
		cutLossArg = cl
	}

	config.DataMutex.RLock()
	existingPlan, exists := config.MyStocks[symbol]
	config.DataMutex.RUnlock()

	if exists {
		totalOldCost := existingPlan.EntryPrice * float64(existingPlan.Lots)
		totalNewCost := entry * float64(lots)

		newTotalLots := existingPlan.Lots + lots
		newAvgPrice := (totalOldCost + totalNewCost) / float64(newTotalLots)

		isAveragingDown := entry < existingPlan.EntryPrice

		cutLoss := existingPlan.CutLoss
		if cutLossArg > 0 {
			cutLoss = pricing.RoundToFraction(cutLossArg)
		}

		plan := models.TradingPlan{
			Symbol:       symbol,
			EntryPrice:   newAvgPrice,
			Lots:         newTotalLots,
			HighestPrice: existingPlan.HighestPrice,
			BuyDate:      existingPlan.BuyDate,
			TakeProfit:   rules.Default.TP1Price(newAvgPrice),
			CutLoss:      cutLoss,
			TP1Done:      false,
		}

		config.DataMutex.Lock()
		config.MyStocks[symbol] = plan
		config.DataMutex.Unlock()

		storage.SaveData()
		storage.LogTrade(storage.TradeRecord{
			Action:     "BUY",
			Symbol:     symbol,
			Price:      entry,
			Lots:       lots,
			PNLPercent: 0,
			PNLRp:      0,
			ExitReason: "",
			Note:       "Averaging Down / Nyicil",
		})

		totalModalBaru := newAvgPrice * float64(newTotalLots) * 100 * (1 + rules.Default.BuyFee)

		tslDisplay := rules.Default.EffectiveStop(plan)
		tslActiveLabel := ""
		if !rules.Default.TrailingActive(plan) {
			tslActiveLabel = " (aktif setelah +2% net)"
		}

		label := "TAMBAH POSISI"
		if isAveragingDown {
			label = "AVERAGE DOWN"
		}

		response := fmt.Sprintf("⚖️ **%s %s BERHASIL!**\n\n"+
			"📅 **Tanggal Entry Pertama:** %s\n"+
			"🛒 **Total Lot Sekarang:** %d\n"+
			"🎯 **Harga Rata-rata Baru:** %s\n"+
			"💸 **Total Modal Terpakai:** %s _(Termasuk Fee 0.15%%)_\n\n"+
			"🚀 **Target Profit (TP1):** %s\n"+
			"🩸 **Batas Cut Loss:** %s\n"+
			"🛡️ **Trailing Stop:** %s%s\n\n"+
			"_Bot telah menyesuaikan batas pengawalan! 🚀_",
			label, symbol, plan.BuyDate, newTotalLots, utils.FormatRupiah(newAvgPrice),
			utils.FormatRupiah(totalModalBaru),
			utils.FormatRupiah(plan.TakeProfit),
			utils.FormatRupiah(plan.CutLoss),
			utils.FormatRupiah(tslDisplay), tslActiveLabel)

		utils.SendMarkdownMessage(bot, response)
	} else {
		cutLoss := rules.Default.DefaultCutLoss(entry)
		if cutLossArg > 0 {
			cutLoss = pricing.RoundToFraction(cutLossArg)
		}

		plan := models.TradingPlan{
			Symbol:       symbol,
			EntryPrice:   entry,
			TakeProfit:   rules.Default.TP1Price(entry),
			CutLoss:      cutLoss,
			HighestPrice: entry,
			Lots:         lots,
			BuyDate:      time.Now().Format("2006-01-02"),
			TP1Done:      false,
		}

		config.DataMutex.Lock()
		config.MyStocks[symbol] = plan
		config.DataMutex.Unlock()

		storage.SaveData()
		storage.LogTrade(storage.TradeRecord{
			Action:     "BUY",
			Symbol:     symbol,
			Price:      entry,
			Lots:       lots,
			PNLPercent: 0,
			PNLRp:      0,
			ExitReason: "",
			Note:       "Entry awal",
		})

		totalModal := entry * float64(lots) * 100 * (1 + rules.Default.BuyFee)

		tslDisplay := rules.Default.EffectiveStop(plan)
		tslActiveLabel := ""
		if !rules.Default.TrailingActive(plan) {
			tslActiveLabel = " (aktif setelah +2% net)"
		}

		response := fmt.Sprintf("✅ **%s BERHASIL DIBELI!**\n\n"+
			"📅 **Tanggal Beli:** %s\n"+
			"🛒 **Lot:** %d\n"+
			"💸 **Modal Terpakai:** %s _(Termasuk Fee 0.15%%)_\n\n"+
			"🚀 **Target Profit (TP1):** %s\n"+
			"🩸 **Batas Cut Loss:** %s\n"+
			"🛡️ **Trailing Stop:** %s%s\n\n"+
			"_Bot akan otomatis mengawal saham ini! 🚀_",
			symbol, plan.BuyDate, lots, utils.FormatRupiah(totalModal),
			utils.FormatRupiah(plan.TakeProfit),
			utils.FormatRupiah(plan.CutLoss),
			utils.FormatRupiah(tslDisplay), tslActiveLabel)

		utils.SendMarkdownMessage(bot, response)
	}
}

func ProcessSellCommand(bot *tgbotapi.BotAPI, args []string) {
	if len(args) < 3 {
		utils.SendSimpleMessage(bot, "❌ Format salah! Gunakan: `/sell [KODE] [HARGA] [LOT]`\nContoh jual sebagian: `/sell MTEL 555 5`\nContoh jual semua: `/sell MTEL 555`")
		return
	}

	symbol := strings.ToUpper(args[1])
	sellPrice, err := strconv.ParseFloat(args[2], 64)

	if err != nil || sellPrice <= 0 {
		utils.SendSimpleMessage(bot, "❌ ERROR: Harga jual harus berupa angka valid dan lebih dari 0!")
		return
	}

	config.DataMutex.RLock()
	plan, ada := config.MyStocks[symbol]
	config.DataMutex.RUnlock()

	if !ada {
		utils.SendSimpleMessage(bot, "❌ Saham tidak ditemukan di portofolio.")
		return
	}

	sellLots := plan.Lots
	isPartial := false

	if len(args) >= 4 {
		parsedLots, err := strconv.Atoi(args[3])
		if err != nil || parsedLots <= 0 {
			utils.SendSimpleMessage(bot, "❌ Jumlah lot tidak valid!")
			return
		}
		if parsedLots > plan.Lots {
			utils.SendSimpleMessage(bot, fmt.Sprintf("❌ Lot tidak cukup! Kamu hanya punya %d lot %s.", plan.Lots, symbol))
			return
		}
		sellLots = parsedLots
		if sellLots < plan.Lots {
			isPartial = true
		}
	}

	netPNL := rules.Default.NetPNL(plan.EntryPrice, sellPrice)
	totalPenjualan := sellPrice * float64(sellLots) * 100 * (1 - rules.Default.SellFee)
	totalModal := plan.EntryPrice * float64(sellLots) * 100 * (1 + rules.Default.BuyFee)
	rupiahPNL := totalPenjualan - totalModal

	exitReason := storage.ResolveExitReason(
		sellPrice,
		plan.EntryPrice,
		plan.CutLoss,
		plan.TakeProfit,
		plan.HighestPrice,
		plan.TP1Done,
		rules.Default.TrailingActive(plan),
	)

	catatanLabel := map[string]string{
		"CL":     "Cut Loss",
		"TSL":    "Trailing Stop",
		"TP1":    "Take Profit",
		"TP":     "Take Profit",
		"MANUAL": "Manual Sell",
	}[exitReason]
	if catatanLabel == "" {
		catatanLabel = "Manual Sell"
	}
	if isPartial {
		catatanLabel += " (Partial)"
	} else {
		catatanLabel += " (Clear)"
	}

	displayPNLRp := rupiahPNL
	rupiahLabel := "Cuan Bersih"
	statusEmoji := "🟢"
	if displayPNLRp < 0 {
		rupiahLabel = "Rugi Bersih"
		statusEmoji = "🔴"
		displayPNLRp = -displayPNLRp
	}

	storage.LogTrade(storage.TradeRecord{
		Action:     "SELL",
		Symbol:     symbol,
		Price:      sellPrice,
		Lots:       sellLots,
		PNLPercent: netPNL,
		PNLRp:      rupiahPNL,
		ExitReason: exitReason,
		Note:       catatanLabel,
	})

	sisaLot := plan.Lots - sellLots

	config.DataMutex.Lock()
	if sisaLot == 0 {
		delete(config.MyStocks, symbol)
	} else {
		plan.Lots = sisaLot
		if isPartial {
			plan.TP1Done = true
		}
		config.MyStocks[symbol] = plan
	}
	config.DataMutex.Unlock()

	storage.SaveData()

	statusPorto := "🗑️ *Posisi Ditutup (Clear)*"
	if sisaLot > 0 {
		statusPorto = fmt.Sprintf("💼 *Sisa di Porto:* %d Lot", sisaLot)
	}

	pesan := fmt.Sprintf("%s **%s BERHASIL DIJUAL!**\n\n"+
		"🤝 **Harga Jual:** %s\n"+
		"🛒 **Terjual:** %d Lot\n"+
		"📊 **PNL:** **%.2f%%**\n"+
		"💰 **%s:** %s\n\n"+
		"%s\n"+
		"📝 _Catatan: %s_",
		statusEmoji, symbol, utils.FormatRupiah(sellPrice), sellLots,
		netPNL, rupiahLabel, utils.FormatRupiah(displayPNLRp),
		statusPorto, catatanLabel)

	utils.SendMarkdownMessage(bot, pesan)
}

func ProcessStatusCommand(bot *tgbotapi.BotAPI) {
	config.DataMutex.RLock()
	myStocksCopy := make(map[string]models.TradingPlan)
	for k, v := range config.MyStocks {
		myStocksCopy[k] = v
	}

	pendingOrdersCopy := make(map[string]models.ActiveOrder)
	for k, v := range config.PendingOrders {
		pendingOrdersCopy[k] = v
	}
	config.DataMutex.RUnlock()

	if len(myStocksCopy) == 0 && len(pendingOrdersCopy) == 0 {
		utils.SendSimpleMessage(bot, "Belum ada saham yang dipantau maupun antrean yang aktif.")
		return
	}

	var sb strings.Builder

	if len(myStocksCopy) > 0 {
		sb.WriteString("📋 **Status Portofolio (Net Bersih):**\n\n")

		for _, plan := range myStocksCopy {
			yahooPrice := market.GetLivePrice(plan.Symbol)
			googlePrice := market.GetGooglePrice(plan.Symbol)

			sourceMarker := "[G]"
			if googlePrice == 0 {
				googlePrice = yahooPrice
				sourceMarker = "[Y-fallback]"
			}

			perfYahoo := rules.Default.NetPNL(plan.EntryPrice, yahooPrice)
			perfGoogle := rules.Default.NetPNL(plan.EntryPrice, googlePrice)

			totalBuyValue := plan.EntryPrice * (1 + rules.Default.BuyFee) * float64(plan.Lots) * 100
			totalSellValue := googlePrice * (1 - rules.Default.SellFee) * float64(plan.Lots) * 100
			totalPNL := totalSellValue - totalBuyValue

			tslDisplay := rules.Default.EffectiveStop(plan)
			tslLabel := utils.FormatRupiah(tslDisplay)
			if !rules.Default.TrailingActive(plan) {
				tslLabel = "belum aktif"
			}

			trendEmoji := "📈"
			if totalPNL < 0 {
				trendEmoji = "📉"
			}

			sb.WriteString(fmt.Sprintf("🔹 **%s** (%d Lot)\n", plan.Symbol, plan.Lots))
			sb.WriteString(fmt.Sprintf("   Avg Beli: %s\n", utils.FormatRupiah(plan.EntryPrice)))
			sb.WriteString(fmt.Sprintf("   [Y] Now : %s (%.2f%%)\n", utils.FormatRupiah(yahooPrice), perfYahoo))
			sb.WriteString(fmt.Sprintf("   %s Now : %s (%.2f%%)\n", sourceMarker, utils.FormatRupiah(googlePrice), perfGoogle))
			sb.WriteString(fmt.Sprintf("   🏔️ Pucuk : %s\n", utils.FormatRupiah(plan.HighestPrice)))
			sb.WriteString(fmt.Sprintf("   🎯 TP1   : %s\n", utils.FormatRupiah(plan.TakeProfit)))
			sb.WriteString(fmt.Sprintf("   🩸 CL    : %s\n", utils.FormatRupiah(plan.CutLoss)))
			sb.WriteString(fmt.Sprintf("   🛡️ TSL   : %s\n", tslLabel))

			pnlLabel := "Cuan Bersih"
			if totalPNL < 0 {
				pnlLabel = "Rugi Bersih"
				totalPNL = -totalPNL
			}

			sb.WriteString(fmt.Sprintf("   👉 **%s: %s %s**\n\n", pnlLabel, utils.FormatRupiah(totalPNL), trendEmoji))
		}
		sb.WriteString("_ket: PNL sudah dipotong pajak Bibit (Beli 0.15%%, Jual 0.25%%)._\n\n")
	}

	if len(pendingOrdersCopy) > 0 {
		if len(myStocksCopy) > 0 {
			sb.WriteString("➖➖➖➖➖➖➖➖➖➖\n\n")
		}

		sb.WriteString("🎣 **Status Antrean Aktif:**\n\n")

		for _, order := range pendingOrdersCopy {
			currentPrice := market.GetLivePrice(order.Symbol)
			diffPercent := 0.0
			if order.OrderPrice > 0 && currentPrice > 0 {
				diffPercent = ((currentPrice - order.OrderPrice) / order.OrderPrice) * 100
			}

			sb.WriteString(fmt.Sprintf("⏳ **%s** (%d Lot)\n", order.Symbol, order.Lot))
			sb.WriteString(fmt.Sprintf("   Antrean Beli: %s\n", utils.FormatRupiah(order.OrderPrice)))

			if currentPrice > 0 {
				sb.WriteString(fmt.Sprintf("   Harga Now   : %s (Selisih +%.2f%%)\n\n", utils.FormatRupiah(currentPrice), diffPercent))
			} else {
				sb.WriteString("   Harga Now   : Market Tutup/Error\n\n")
			}
		}
	}

	utils.SendMarkdownMessage(bot, sb.String())
}

func ProcessResetCommand(bot *tgbotapi.BotAPI, confirmed bool) {
	if !confirmed {
		utils.SendSimpleMessage(bot, "Ketik `/reset confirm` untuk menghapus SEMUA pantauan (posisi + antrean).")
		return
	}

	config.DataMutex.Lock()
	config.MyStocks = make(map[string]models.TradingPlan)
	config.PendingOrders = make(map[string]models.ActiveOrder)
	config.DataMutex.Unlock()

	storage.SaveData()
	utils.SendSimpleMessage(bot, "🧹 Semua pantauan dihapus!")
}
