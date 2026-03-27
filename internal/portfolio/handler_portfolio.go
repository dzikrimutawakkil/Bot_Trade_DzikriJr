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

	storage.LogTrade("BUY", symbol, entry, lots, 0.0, "Entry awal")

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
		utils.FormatRupiah(func() float64 {
			if initialTSL > plan.CutLoss {
				return initialTSL
			}
			return plan.CutLoss
		}()), // Pastikan TSL awal tidak lebih dalam dari Cut Loss
		utils.FormatRupiah(plan.CutLoss))
		
	utils.SendMarkdownMessage(bot, response)
}

// Logika /sell dengan fitur Pencatatan Otomatis (Auto-Logger) dan Harga Manual
func ProcessSellCommand(bot *tgbotapi.BotAPI, args []string) {
	// Format sekarang wajib pakai harga: /sell [KODE] [HARGA]
	if len(args) < 3 {
		utils.SendSimpleMessage(bot, "❌ Format salah! Gunakan: `/sell [KODE] [HARGA]`")
		return
	}
	
	symbol := strings.ToUpper(args[1])
	sellPrice, err := strconv.ParseFloat(args[2], 64)
	if err != nil {
		utils.SendSimpleMessage(bot, "❌ Harga harus berupa angka!")
		return
	}
	
	if plan, ada := config.MyStocks[symbol]; ada {
		
		// 1. Hitung persentase cuan/rugi (netPNL) dengan harga manualmu
		netPNL := utils.CalculateNetPNL(plan.EntryPrice, sellPrice, config.BuyFee, config.SellFee)

		// 2. Hitung Rupiah Bersih (Cash flow sebenarnya di rekening)
		totalPenjualan := sellPrice * float64(plan.Lots) * 100 * (1 - config.SellFee)
		totalModal := plan.EntryPrice * float64(plan.Lots) * 100 * (1 + config.BuyFee)
		rupiahPNL := totalPenjualan - totalModal

		rupiahLabel := "Cuan Bersih"
		statusEmoji := "🟢"
		if rupiahPNL < 0 {
			rupiahLabel = "Rugi Bersih"
			statusEmoji = "🔴"
			rupiahPNL = -rupiahPNL // Ubah ke positif untuk keperluan tampilan format Rupiah
		}

		// 3. Tentukan Catatan (Take Profit / Cut Loss / Manual)
		catatan := "Manual Sell"
		if netPNL >= config.TPPercent*100 {
			catatan = "Take Profit"
		} else if netPNL <= -config.CLPercent*100 {
			catatan = "Cut Loss"
		}

		// 4. Catat ke dalam CSV sebelum datanya dihapus
		storage.LogTrade("SELL", symbol, sellPrice, plan.Lots, netPNL, catatan)

		// 5. Hapus pantauan portofolio dan simpan
		delete(config.MyStocks, symbol)
		storage.SaveData()
		
		// 6. Format Pesan Laporan Penjualan
		pesan := fmt.Sprintf("%s **%s BERHASIL DIJUAL!**\n\n"+
			"🤝 **Harga Jual:** %s\n"+
			"🛒 **Lot Terjual:** %d\n"+
			"📊 **PNL Persentase:** **%.2f%%**\n"+
			"💰 **%s:** %s\n\n"+
			"📝 _Tercatat di History sebagai: %s_", 
			statusEmoji, symbol, 
			utils.FormatRupiah(sellPrice), plan.Lots, 
			netPNL, rupiahLabel, utils.FormatRupiah(rupiahPNL), 
			catatan)
			
		utils.SendMarkdownMessage(bot, pesan)
		
	} else {
		utils.SendSimpleMessage(bot, "❌ Saham tidak ditemukan di portofolio.")
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

		// 3. Hitung Batas Trailing Stop Saat Ini (Diperbarui dengan Logika Anti-Bocor)
		var tslLimit float64
		if plan.HighestPrice <= plan.EntryPrice {
			// Jika saham belum pernah naik (pucuk = harga beli), gunakan Cut Loss awal
			tslLimit = plan.CutLoss
		} else {
			// Jika saham sudah naik, hitung TSL Dinamis
			kalkulasiTSL := plan.HighestPrice * (1 - config.TrailingStopPercent)
			
			// Pastikan TSL dinamis tidak lebih dalam (rendah) dari Cut Loss awal
			tslLimit = math.Max(kalkulasiTSL, plan.CutLoss)
		}
		
		// 4. PEMBULATAN SEMUA BATAS HARGA (Sesuai Fraksi Pasar)
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
		
		// Info Pucuk, TP, CL & TSL yang sudah dibulatkan
		sb.WriteString(fmt.Sprintf("   🏔️ Pucuk : %s\n", utils.FormatRupiah(plan.HighestPrice)))
		sb.WriteString(fmt.Sprintf("   🎯 TP    : %s\n", utils.FormatRupiah(takeProfit)))
		sb.WriteString(fmt.Sprintf("   🩸 CL    : %s\n", utils.FormatRupiah(cutLoss)))
		sb.WriteString(fmt.Sprintf("   🛡️ TSL   : %s\n", utils.FormatRupiah(tslLimit)))

		pnlLabel := "Cuan Bersih"
		if totalPNL < 0 {
			pnlLabel = "Rugi Bersih"
			totalPNL = -totalPNL // Ubah ke positif untuk format Rupiah
		}
		
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