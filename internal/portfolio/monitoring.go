package portfolio

import (
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"learn-go/internal/config"
	"learn-go/internal/storage"
	"learn-go/internal/utils"
	"learn-go/internal/market"
)

func StartPriceMonitor(bot *tgbotapi.BotAPI) {
	ticker := time.NewTicker(config.CheckPeriod)
	log.Println("📡 Monitor harga (Dual-Check) aktif dengan Trailing Stop...")

	for range ticker.C {
		if !utils.IsMarketOpen() {
			continue
		}

		for symbol, plan := range config.MyStocks {
			yahooPrice := market.GetLivePrice(symbol)
			if yahooPrice == 0 {
				continue
			}

			// --- FITUR BARU: UPDATE HIGHEST PRICE UNTUK TRAILING STOP ---
			if plan.HighestPrice == 0 {
				plan.HighestPrice = plan.EntryPrice // Inisialisasi awal
			}

			isNewHigh := false
			if yahooPrice > plan.HighestPrice {
				plan.HighestPrice = yahooPrice
				isNewHigh = true
			}

			if isNewHigh {
				config.MyStocks[symbol] = plan
				storage.SaveData()
			}
			// -------------------------------------------------------------

			// Hitung persentase PNL dari Yahoo
			yahooPNL := utils.CalculateNetPNL(plan.EntryPrice, yahooPrice, config.BuyFee, config.SellFee)

			// Hitung batas Trailing Stop dinamis (Cut Loss berdasarkan pucuk)
			// Misal plan.HighestPrice = 1000, TrailingStopPercent = 0.04 (4%), TSL = 960
			tslPrice := plan.HighestPrice * (1 - config.TrailingStopPercent)

			// TRIGGER PENGECEKAN
			isTPTrigger := yahooPNL >= config.YahooTPTrigger
			isTSLTrigger := yahooPrice <= tslPrice // Trigger baru menggunakan angka TSL
			isCLTrigger := yahooPNL <= -config.YahooCLTrigger 

			if isTPTrigger || isCLTrigger || isTSLTrigger {
				// STEP 2: Cross-check Google Finance
				realPrice := market.GetGooglePrice(symbol)
				if realPrice == 0 {
					realPrice = yahooPrice
				}

				realPNL := utils.CalculateNetPNL(plan.EntryPrice, realPrice, config.BuyFee, config.SellFee)
				
				// Verifikasi TSL dengan harga real
				realTslTriggered := realPrice <= tslPrice

				// STEP 3: Verifikasi Final menggunakan variabel Config
				conditionMet := false
				var msg string

				if realPNL >= config.GoogleTPTarget {
					// KONDISI 1: TAKE PROFIT (Harga Tembus Target Atas)
					msg = fmt.Sprintf("🚀 **TAKE PROFIT CONFIRMED!**\n\n"+
						"Emiten: **%s**\n"+
						"Target: `+%.1f%%`\n"+
						"Real PNL: `+%.2f%%`\n"+
						"Harga Google: `%s`\n\n"+
						"Ketik `/sell %s` jika sudah eksekusi.",
						symbol, config.GoogleTPTarget, realPNL, utils.FormatRupiah(realPrice), symbol)
					conditionMet = true

				} else if realTslTriggered {
					// KONDISI 2: TRAILING STOP (Sabuk Pengaman Tersentuh)
					// Prioritaskan TSL daripada CL biasa karena TSL mengunci profit di pucuk
					securedPNL := ((tslPrice - plan.EntryPrice) / plan.EntryPrice) * 100
					statusEmoji := "💸" // Profit Diamankan
					if securedPNL < 0 {
						statusEmoji = "🩸" // Rugi Dibatasi
					}

					msg = fmt.Sprintf("🚨 **TRAILING STOP TERSENTUH!** %s\n\n"+
						"Emiten: **%s**\n"+
						"Puncak Tertinggi: `%s`\n"+
						"Batas Sabuk (TSL): `%s`\n"+
						"Harga Google: `%s`\n\n"+
						"PNL Saat Ini: `%.2f%%`\n\n"+
						"Ketik `/sell %s` jika sudah jual.",
						statusEmoji, symbol, utils.FormatRupiah(plan.HighestPrice), utils.FormatRupiah(tslPrice), utils.FormatRupiah(realPrice), securedPNL, symbol)
					conditionMet = true

				} else if realPNL <= -config.GoogleCLTarget {
					// KONDISI 3: CUT LOSS STATIS (Sebagai cadangan kalau TSL belum sempat naik)
					msg = fmt.Sprintf("🚨 **CUT LOSS CONFIRMED!**\n\n"+
						"Emiten: **%s**\n"+
						"Batas: `-%.1f%%`\n"+
						"Real PNL: `%.2f%%`\n"+
						"Harga Google: `%s`\n\n"+
						"Ketik `/sell %s` jika sudah eksekusi.",
						symbol, config.GoogleCLTarget, realPNL, utils.FormatRupiah(realPrice), symbol)
					conditionMet = true
				}

				// STEP 4: Kirim Notifikasi
				if conditionMet {
					// Gunakan EmergencyDelay dari Config
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
	}
}