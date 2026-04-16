package portfolio

import (
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"learn-go/internal/config"
	"learn-go/internal/market"
	"learn-go/internal/storage"
	"learn-go/internal/utils"
	"learn-go/internal/models"
)
func StartPriceMonitor(bot *tgbotapi.BotAPI) {
	ticker := time.NewTicker(config.CheckPeriod)
	log.Println("📡 Monitor harga (Dual-Check) aktif dengan Trailing Stop & Order Match...")

	for range ticker.C {
		if !utils.IsMarketOpen() {
			continue
		}

		// ==========================================
		// 1. PANTAU PORTOFOLIO AKTIF (MyStocks)
		// ==========================================
		for symbol, plan := range config.MyStocks {
			yahooPrice := market.GetLivePrice(symbol)
			if yahooPrice == 0 {
				continue
			}

			// --- UPDATE HIGHEST PRICE ---
			if plan.HighestPrice == 0 {
				plan.HighestPrice = plan.EntryPrice
			}

			if yahooPrice > plan.HighestPrice {
				plan.HighestPrice = yahooPrice
				config.MyStocks[symbol] = plan
				storage.SaveData()
			}

			// 1. Hitung PNL Saat Ini
			yahooPNL := utils.CalculateNetPNL(plan.EntryPrice, yahooPrice, config.BuyFee, config.SellFee)

			// 2. Tentukan Batas TSL (Harga Pucuk - 1.5%)
			tslPrice := plan.HighestPrice * (1 - config.TrailingStopPercent)

			// --- TRIGGER PENGECEKAN ---
			isTPTrigger := yahooPNL >= config.YahooTPTrigger
			isCLTrigger := yahooPNL <= -config.YahooCLTrigger

			// 🔥 TSL TRIGGER: Terhubung ke TSLActivationTrigger di Config
			isTSLTrigger := false
			if yahooPNL >= config.TSLActivationTrigger {
				isTSLTrigger = yahooPrice <= tslPrice
			}

			if isTPTrigger || isCLTrigger || isTSLTrigger {
				// STEP 2: Cross-check Google Finance
				realPrice := market.GetGooglePrice(symbol)
				if realPrice == 0 {
					realPrice = yahooPrice
				}

				realPNL := utils.CalculateNetPNL(plan.EntryPrice, realPrice, config.BuyFee, config.SellFee)

				// Verifikasi TSL ulang dengan harga real (Tetap cek syarat aktivasi)
				realTslTriggered := false
				if realPNL >= config.TSLActivationTrigger {
					realTslTriggered = realPrice <= tslPrice
				}

				// STEP 3: Verifikasi Final & Notifikasi
				conditionMet := false
				var msg string

				if realPNL >= config.GoogleTPTarget {
					msg = fmt.Sprintf("🎯 **TAKE PROFIT CONFIRMED!**\n\nEmiten: **%s**\nReal PNL: `+%.2f%%`\nKetik `/sell %s`.", symbol, realPNL, symbol)
					conditionMet = true
				} else if realTslTriggered {
					msg = fmt.Sprintf("🛡️ **TRAILING STOP TERSENTUH!** ✅\n\nEmiten: **%s**\nBatas TSL: `%s`\nCuan Aman: `+%.2f%%`\nKetik `/sell %s`.",
						symbol, utils.FormatRupiah(tslPrice), realPNL, symbol)
					conditionMet = true
				} else if realPNL <= -config.GoogleCLTarget {
					msg = fmt.Sprintf("🚨 **CUT LOSS CONFIRMED!**\n\nEmiten: **%s**\nReal PNL: `%.2f%%`\nKetik `/sell %s`.", symbol, realPNL, symbol)
					conditionMet = true
				}

				if conditionMet {
					if time.Since(plan.LastNotified) < config.EmergencyDelay {
						continue
					}
					utils.SendMarkdownMessage(bot, msg)
					plan.LastNotified = time.Now()
					config.MyStocks[symbol] = plan
					storage.SaveData()
				}
			}
		}

		// ==========================================
		// 2. PANTAU ANTREAN BARU (PendingOrders)
		// ==========================================
		for symbol, order := range config.PendingOrders {
			currentPrice := market.GetLivePrice(symbol)
			if currentPrice == 0 {
				continue
			}

			// --- KONDISI 1: AUTO MATCH (LANTAI TERSENTUH/JEBOL) ---
			// Jika harga market sudah sama dengan atau lebih rendah dari harga antrean kita
			if currentPrice <= order.OrderPrice {

				// Langsung hapus dari antrean agar tidak tereksekusi dua kali
				delete(config.PendingOrders, symbol)

				// --- CEK APAKAH SUDAH PUNYA SAHAMNYA (AVERAGING DOWN) ---
				if existingPlan, exists := config.MyStocks[symbol]; exists {
					// 1. Hitung total modal lama dan baru
					totalOldCost := existingPlan.EntryPrice * float64(existingPlan.Lots)
					totalNewCost := order.OrderPrice * float64(order.Lot)

					// 2. Hitung harga rata-rata baru
					newTotalLots := existingPlan.Lots + order.Lot
					newAveragePrice := (totalOldCost + totalNewCost) / float64(newTotalLots)

					// 3. Update Plan yang sudah ada
					existingPlan.EntryPrice = newAveragePrice
					existingPlan.Lots = newTotalLots
					existingPlan.TakeProfit = utils.RoundToFraction(newAveragePrice * (1 + config.TPPercent))
					existingPlan.CutLoss = utils.RoundToFraction(newAveragePrice * (1 - config.CLPercent))
					existingPlan.HighestPrice = newAveragePrice // Reset

					config.MyStocks[symbol] = existingPlan
					storage.LogTrade("BUY", symbol, order.OrderPrice, order.Lot, 0.0, "Auto-Match (Averaging Down)")
					storage.SaveData()

					totalModalBaru := newAveragePrice * float64(newTotalLots) * 100 * (1 + config.BuyFee)

					msg := fmt.Sprintf("🎯 **ANTREAN MATCH & AVERAGED!**\n\n"+
						"Emiten: **%s**\n"+
						"Antrean Beli: `Rp. %.0f` (%d Lot)\n\n"+
						"⚖️ **Status Portofolio Baru:**\n"+
						"Total Lot: `%d Lot`\n"+
						"Harga Rata-rata: `Rp. %.0f`\n"+
						"Total Modal: `%s` _(Termasuk Fee)_\n\n"+
						"_Radar Cut Loss & Trailing Stop telah disesuaikan!_ 🛡️",
						symbol, order.OrderPrice, order.Lot, newTotalLots, newAveragePrice, utils.FormatRupiah(totalModalBaru))
					utils.SendMarkdownMessage(bot, msg)

				} else {
					// --- ENTRY BARU (BELUM PUNYA SAHAMNYA) ---
					plan := models.TradingPlan{
						Symbol:       symbol,
						EntryPrice:   order.OrderPrice,
						HighestPrice: order.OrderPrice,
						Lots:         order.Lot,
						TakeProfit:   utils.RoundToFraction(order.OrderPrice * (1 + config.TPPercent)),
						CutLoss:      utils.RoundToFraction(order.OrderPrice * (1 - config.CLPercent)),
					}
					
					config.MyStocks[symbol] = plan
					storage.LogTrade("BUY", symbol, order.OrderPrice, order.Lot, 0.0, "Auto-Match dari Antrean")
					storage.SaveData()

					totalModal := order.OrderPrice * float64(order.Lot) * 100 * (1 + config.BuyFee)

					msg := fmt.Sprintf("✅ **ANTREAN MATCHED!**\n\n"+
						"Emiten: **%s**\n"+
						"Harga Beli: `Rp. %.0f`\n"+
						"Jumlah: `%d Lot`\n"+
						"Modal Terpakai: `%s` _(Termasuk Fee)_\n\n"+
						"_Saham telah otomatis masuk ke portofolio aktif. Radar Cut Loss & Trailing Stop sekarang MENYALA._ 🛡️",
						symbol, order.OrderPrice, order.Lot, utils.FormatRupiah(totalModal))
					utils.SendMarkdownMessage(bot, msg)
				}

				continue // Lanjut cek antrean berikutnya
			}

			// --- KONDISI 2: HARGA KABUR (OPPORTUNITY COST) ---
			diffPercent := (currentPrice - order.OrderPrice) / order.OrderPrice

			if diffPercent >= config.RunawayPercent {
				if time.Since(order.LastNotified) > 60*time.Minute {
					msg := fmt.Sprintf("🏃‍♂️💨 **HARGA KABUR BOS!**\n\n"+
						"Emiten: **%s**\n"+
						"Antreanmu: `Rp. %.0f`\n"+
						"Harga Sekarang: `Rp. %.0f` (Naik +%.1f%%)\n\n"+
						"Uangnya nganggur nih. Mending ditarik aja antreannya di aplikasi sekuritas.\n\n"+
						"👉 Ketik `/cancel_antre %s` untuk hapus dari pantauan bot.",
						symbol, order.OrderPrice, currentPrice, diffPercent*100, symbol)

					utils.SendMarkdownMessage(bot, msg)

					order.LastNotified = time.Now()
					config.PendingOrders[symbol] = order
				}
			}
		}
	}
}