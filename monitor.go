package main

import (
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func startPriceMonitor(bot *tgbotapi.BotAPI) {
	ticker := time.NewTicker(CheckPeriod)
	log.Println("📡 Monitor harga (Dual-Check) aktif dengan konfigurasi eksternal...")

	for range ticker.C {
		if !isMarketOpen() {
			continue
		}

		for symbol, plan := range myStocks {
			yahooPrice := getLivePrice(symbol)
			if yahooPrice == 0 {
				continue
			}

			// Hitung persentase PNL dari Yahoo
			yahooPNL := (yahooPrice - plan.EntryPrice) / plan.EntryPrice * 100

			// STEP 1: Gunakan variabel Config untuk Trigger Yahoo
			isTPTrigger := yahooPNL >= YahooTPTrigger
			isCLTrigger := yahooPNL <= -YahooCLTrigger // Kita pakai minus karena CL itu turun

			if isTPTrigger || isCLTrigger {
				// STEP 2: Cross-check Google
				realPrice := getGooglePrice(symbol)
				if realPrice == 0 {
					realPrice = yahooPrice
				}

				realPNL := (realPrice - plan.EntryPrice) / plan.EntryPrice * 100

				// STEP 3: Verifikasi Final menggunakan variabel Config
				conditionMet := false
				var msg string

				if realPNL >= GoogleTPTarget {
					// KONDISI TAKE PROFIT
					msg = fmt.Sprintf("🚀 **TAKE PROFIT CONFIRMED!**\n\n"+
						"Emiten: **%s**\n"+
						"Target: `+%.1f%%`\n"+
						"Real PNL: `+%.2f%%`\n"+
						"Harga Google: `%s`\n\n"+
						"Ketik `/sell %s` jika sudah eksekusi.",
						symbol, GoogleTPTarget, realPNL, formatRupiah(realPrice), symbol)
					conditionMet = true
				} else if realPNL <= -GoogleCLTarget {
					// KONDISI CUT LOSS
					msg = fmt.Sprintf("🚨 **CUT LOSS CONFIRMED!**\n\n"+
						"Emiten: **%s**\n"+
						"Batas: `-%.1f%%`\n"+
						"Real PNL: `%.2f%%`\n"+
						"Harga Google: `%s`\n\n"+
						"Ketik `/sell %s` jika sudah eksekusi.",
						symbol, GoogleCLTarget, realPNL, formatRupiah(realPrice), symbol)
					conditionMet = true
				}

				// STEP 4: Kirim Notifikasi
				if conditionMet {
					// Gunakan EmergencyDelay dari Config
					if time.Since(plan.LastNotified) < EmergencyDelay {
						continue
					}

					sendMarkdownMessage(bot, msg)
					plan.LastNotified = time.Now()
					myStocks[symbol] = plan
					saveData()
				}
			}
		}
	}
}