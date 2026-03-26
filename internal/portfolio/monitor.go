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
	log.Println("📡 Monitor harga (Dual-Check) aktif dengan konfigurasi eksternal...")

	for range ticker.C {
		if !utils.IsMarketOpen() {
			continue
		}

		for symbol, plan := range config.MyStocks {
			yahooPrice := market.GetLivePrice(symbol)
			if yahooPrice == 0 {
				continue
			}

			// Hitung persentase PNL dari Yahoo
			yahooPNL := (yahooPrice - plan.EntryPrice) / plan.EntryPrice * 100

			// STEP 1: Gunakan variabel Config untuk Trigger Yahoo
			isTPTrigger := yahooPNL >= config.YahooTPTrigger
			isCLTrigger := yahooPNL <= -config.YahooCLTrigger // Kita pakai minus karena CL itu turun

			if isTPTrigger || isCLTrigger {
				// STEP 2: Cross-check Google
				realPrice := market.GetGooglePrice(symbol)
				if realPrice == 0 {
					realPrice = yahooPrice
				}

				realPNL := (realPrice - plan.EntryPrice) / plan.EntryPrice * 100

				// STEP 3: Verifikasi Final menggunakan variabel Config
				conditionMet := false
				var msg string

				if realPNL >= config.GoogleTPTarget {
					// KONDISI TAKE PROFIT
					msg = fmt.Sprintf("🚀 **TAKE PROFIT CONFIRMED!**\n\n"+
						"Emiten: **%s**\n"+
						"Target: `+%.1f%%`\n"+
						"Real PNL: `+%.2f%%`\n"+
						"Harga Google: `%s`\n\n"+
						"Ketik `/sell %s` jika sudah eksekusi.",
						symbol, config.GoogleTPTarget, realPNL, utils.FormatRupiah(realPrice), symbol)
					conditionMet = true
				} else if realPNL <= -config.GoogleCLTarget {
					// KONDISI CUT LOSS
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