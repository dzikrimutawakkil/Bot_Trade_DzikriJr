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
	"math"
	"time"
)

// Pastikan kamu import "time" di bagian atas file ini!
// import "time"

func ProcessBuyCommand(bot *tgbotapi.BotAPI, args []string) {
    if len(args) < 4 {
        utils.SendSimpleMessage(bot, "❌ Format salah! Gunakan: `/buy [KODE] [HARGA] [LOT]`\nContoh: `/buy BBCA 9000 10`")
        return
    }

    symbol := strings.ToUpper(args[1])
    entry, errPrice := strconv.ParseFloat(args[2], 64)
    lots, errLot := strconv.Atoi(args[3])

    if errPrice != nil || errLot != nil || entry <= 0 || lots <= 0 {
        utils.SendSimpleMessage(bot, "❌ ERROR: Harga dan Lot harus berupa angka dan lebih besar dari 0!")
        return
    }

    // --- LOGIKA AVERAGING DOWN (SCALING IN) ---
    config.DataMutex.RLock()
    existingPlan, exists := config.MyStocks[symbol]
    config.DataMutex.RUnlock()

    if exists {
        totalOldCost := existingPlan.EntryPrice * float64(existingPlan.Lots)
        totalNewCost := entry * float64(lots)
        
        newTotalLots := existingPlan.Lots + lots
        newAveragePrice := (totalOldCost + totalNewCost) / float64(newTotalLots)

        existingPlan.EntryPrice = newAveragePrice
        existingPlan.Lots = newTotalLots
        existingPlan.TakeProfit = utils.RoundToFraction(newAveragePrice * (1 + config.TPPercent))
        existingPlan.CutLoss = utils.RoundToFraction(newAveragePrice * (1 - config.CLPercent))
        existingPlan.HighestPrice = newAveragePrice
        
        // 🔄 UPDATE TANGGAL: Jika average down, anggap ini sebagai titik awal hold yang baru
        existingPlan.BuyDate = time.Now().Format("2006-01-02") 

        config.DataMutex.Lock()
        config.MyStocks[symbol] = existingPlan
        config.DataMutex.Unlock()

        storage.SaveData()
        storage.LogTrade("BUY", symbol, entry, lots, 0.0, "Averaging Down / Nyicil")

        initialTSLRaw := newAveragePrice * (1 - config.TrailingStopPercent)
        initialTSL := utils.RoundToFraction(initialTSLRaw)
        totalModalBaru := newAveragePrice * float64(newTotalLots) * 100 * (1 + config.BuyFee)

        response := fmt.Sprintf("⚖️ **AVERAGE DOWN %s BERHASIL!**\n\n"+
            "📅 **Tanggal Diperbarui:** %s\n"+ // <-- Tambahan visual di Telegram (opsional)
            "🛒 **Total Lot Sekarang:** %d\n"+
            "🎯 **Harga Rata-rata Baru:** %s\n"+
            "💸 **Total Modal Terpakai:** %s _(Termasuk Fee 0.15%%)_\n\n"+
            "🚀 **Target Profit Baru:** %s\n"+
            "🛡️ **Trailing Stop Awal:** %s\n"+
            "🩸 **Batas Cut Loss Baru:** %s\n\n"+
            "_Bot telah menyesuaikan batas pengawalan! 🚀_",
            symbol, existingPlan.BuyDate, newTotalLots, utils.FormatRupiah(newAveragePrice), utils.FormatRupiah(totalModalBaru), 
            utils.FormatRupiah(existingPlan.TakeProfit), 
            utils.FormatRupiah(math.Max(initialTSL, existingPlan.CutLoss)), utils.FormatRupiah(existingPlan.CutLoss))
            
        utils.SendMarkdownMessage(bot, response)

    } else {
        // --- ENTRY AWAL (SAHAM BARU) ---
        initialTSLRaw := entry * (1 - config.TrailingStopPercent)
        initialTSL := utils.RoundToFraction(initialTSLRaw)

        plan := models.TradingPlan{
            Symbol:       symbol,
            EntryPrice:   entry,
            TakeProfit:   utils.RoundToFraction(entry * (1 + config.TPPercent)),
            CutLoss:      utils.RoundToFraction(entry * (1 - config.CLPercent)),
            HighestPrice: entry, 
            Lots:         lots,
            BuyDate:      time.Now().Format("2006-01-02"),
        }

        config.DataMutex.Lock()
        config.MyStocks[symbol] = plan
        config.DataMutex.Unlock()

        storage.SaveData()
        storage.LogTrade("BUY", symbol, entry, lots, 0.0, "Entry awal")

        totalModal := entry * float64(lots) * 100 * (1 + config.BuyFee)

        response := fmt.Sprintf("✅ **%s BERHASIL DIBELI!**\n\n"+
            "📅 **Tanggal Beli:** %s\n"+ // <-- Tambahan visual di Telegram (opsional)
            "🛒 **Lot:** %d\n"+
            "💸 **Modal Terpakai:** %s _(Termasuk Fee 0.15%%)_\n\n"+
            "🎯 **Target Profit:** %s\n"+
            "🛡️ **Trailing Stop Awal:** %s\n"+
            "🩸 **Batas Cut Loss:** %s\n\n"+
            "_Bot akan otomatis mengawal saham ini! 🚀_",
            symbol, plan.BuyDate, lots, utils.FormatRupiah(totalModal), utils.FormatRupiah(plan.TakeProfit), 
            utils.FormatRupiah(math.Max(initialTSL, plan.CutLoss)), utils.FormatRupiah(plan.CutLoss))
            
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
	
	// 🔒 [LOCK READ] Intip plan
	config.DataMutex.RLock()
	plan, ada := config.MyStocks[symbol]
	config.DataMutex.RUnlock()

	if ada {
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

		netPNL := utils.CalculateNetPNL(plan.EntryPrice, sellPrice, config.BuyFee, config.SellFee)
		totalPenjualan := sellPrice * float64(sellLots) * 100 * (1 - config.SellFee)
		totalModal := plan.EntryPrice * float64(sellLots) * 100 * (1 + config.BuyFee)
		rupiahPNL := totalPenjualan - totalModal

		rupiahLabel := "Cuan Bersih"
		statusEmoji := "🟢"
		if rupiahPNL < 0 {
			rupiahLabel = "Rugi Bersih"
			statusEmoji = "🔴"
			rupiahPNL = -rupiahPNL 
		}

		catatan := "Manual Sell"
		if netPNL >= config.TPPercent*100 {
			catatan = "Take Profit"
		} else if netPNL <= -config.CLPercent*100 {
			catatan = "Cut Loss"
		}
		if isPartial {
			catatan += " (Partial)"
		} else {
			catatan += " (Clear)"
		}

		storage.LogTrade("SELL", symbol, sellPrice, sellLots, netPNL, catatan)

		sisaLot := plan.Lots - sellLots

		// 🔒 [LOCK WRITE] Waktunya menghapus/mengubah sisa lot
		config.DataMutex.Lock()
		if sisaLot == 0 {
			delete(config.MyStocks, symbol) 
		} else {
			plan.Lots = sisaLot
			config.MyStocks[symbol] = plan 
		}
		config.DataMutex.Unlock() // 🔓 Buka gembok

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
			netPNL, rupiahLabel, utils.FormatRupiah(rupiahPNL), 
			statusPorto, catatan)
			
		utils.SendMarkdownMessage(bot, pesan)
	} else {
		utils.SendSimpleMessage(bot, "❌ Saham tidak ditemukan di portofolio.")
	}
}

func ProcessStatusCommand(bot *tgbotapi.BotAPI) {
	// 🔒 [LOCK READ] Clone Map untuk diproses agar tidak bentrok saat membaca
	config.DataMutex.RLock()
	myStocksCopy := make(map[string]models.TradingPlan)
	for k, v := range config.MyStocks { myStocksCopy[k] = v }
	
	pendingOrdersCopy := make(map[string]models.ActiveOrder)
	for k, v := range config.PendingOrders { pendingOrdersCopy[k] = v }
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

			perfYahoo := utils.CalculateNetPNL(plan.EntryPrice, yahooPrice, config.BuyFee, config.SellFee)
			perfGoogle := utils.CalculateNetPNL(plan.EntryPrice, googlePrice, config.BuyFee, config.SellFee)

			totalBuyValue := plan.EntryPrice * (1 + config.BuyFee) * float64(plan.Lots) * 100
			totalSellValue := googlePrice * (1 - config.SellFee) * float64(plan.Lots) * 100
			totalPNL := totalSellValue - totalBuyValue

			var tslLimit float64
			if plan.HighestPrice <= plan.EntryPrice {
				tslLimit = plan.CutLoss
			} else {
				kalkulasiTSL := plan.HighestPrice * (1 - config.TrailingStopPercent)
				tslLimit = math.Max(kalkulasiTSL, plan.CutLoss)
			}

			tslLimit = math.Round(tslLimit)
			takeProfit := math.Round(plan.TakeProfit)
			cutLoss := math.Round(plan.CutLoss)

			trendEmoji := "📈"
			if totalPNL < 0 {
				trendEmoji = "📉"
			}

			sb.WriteString(fmt.Sprintf("🔹 **%s** (%d Lot)\n", plan.Symbol, plan.Lots))
			sb.WriteString(fmt.Sprintf("   Avg Beli: %s\n", utils.FormatRupiah(plan.EntryPrice)))
			sb.WriteString(fmt.Sprintf("   [Y] Now : %s (%.2f%%)\n", utils.FormatRupiah(yahooPrice), perfYahoo))
			sb.WriteString(fmt.Sprintf("   %s Now : %s (%.2f%%)\n", sourceMarker, utils.FormatRupiah(googlePrice), perfGoogle))
			sb.WriteString(fmt.Sprintf("   🏔️ Pucuk : %s\n", utils.FormatRupiah(plan.HighestPrice)))
			sb.WriteString(fmt.Sprintf("   🎯 TP    : %s\n", utils.FormatRupiah(takeProfit)))
			sb.WriteString(fmt.Sprintf("   🩸 CL    : %s\n", utils.FormatRupiah(cutLoss)))
			sb.WriteString(fmt.Sprintf("   🛡️ TSL   : %s\n", utils.FormatRupiah(tslLimit)))

			pnlLabel := "Cuan Bersih"
			if totalPNL < 0 {
				pnlLabel = "Rugi Bersih"
				totalPNL = -totalPNL 
			}

			sb.WriteString(fmt.Sprintf("   👉 **%s: %s %s**\n\n", pnlLabel, utils.FormatRupiah(totalPNL), trendEmoji))
		}
		sb.WriteString("_ket: PNL sudah dipotong pajak Bibit (Beli 0.15%, Jual 0.25%)._\n\n")
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

func ProcessResetCommand(bot *tgbotapi.BotAPI) {
	// 🔒 [LOCK WRITE] Bersihkan map
	config.DataMutex.Lock()
	config.MyStocks = make(map[string]models.TradingPlan)
	config.DataMutex.Unlock() // 🔓 Buka gembok

	storage.SaveData()
	utils.SendSimpleMessage(bot, "🧹 Semua pantauan dihapus!")
}