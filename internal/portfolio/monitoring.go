package portfolio

import (
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"learn-go/internal/config"
	"learn-go/internal/market"
	"learn-go/internal/models"
	"learn-go/internal/rules"
	"learn-go/internal/storage"
	"learn-go/internal/utils"
)

func StartPriceMonitor(bot *tgbotapi.BotAPI) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("⚠️ Price monitor ticker recovered: %v", r)
		}
	}()
	ticker := time.NewTicker(config.CheckPeriod)
	log.Println("📡 Monitor harga (Dual-Check) aktif dengan Trailing Stop & Order Match...")

	for range ticker.C {
		if !utils.IsMarketOpen() {
			continue
		}

		// ==========================================
		// 1. PANTAU PORTOFOLIO AKTIF (MyStocks)
		// ==========================================
		config.DataMutex.RLock()
		symbols := make([]string, 0, len(config.MyStocks))
		for sym := range config.MyStocks {
			symbols = append(symbols, sym)
		}
		config.DataMutex.RUnlock()

		for _, symbol := range symbols {
			yahooPrice := market.GetLivePrice(symbol)
			if yahooPrice == 0 {
				continue
			}

			// Update HighestPrice via UpdatePlan (thread-safe, only writes if changed)
			changed := config.UpdatePlan(symbol, func(p *models.TradingPlan) {
				if p.HighestPrice == 0 {
					p.HighestPrice = p.EntryPrice
				}
				if yahooPrice > p.HighestPrice {
					p.HighestPrice = yahooPrice
				}
			})
			if !changed {
				continue
			}
			storage.SaveData()

			// Evaluate dengan harga Yahoo
			config.DataMutex.RLock()
			plan := config.MyStocks[symbol]
			config.DataMutex.RUnlock()

			// TIME STOP: Exit if hold days >= 20
			config.DataMutex.RLock()
			currentPlan := config.MyStocks[symbol]
			config.DataMutex.RUnlock()
			if currentPlan.HoldDays >= 20 {
				if time.Since(currentPlan.LastNotified) >= config.EmergencyDelay {
					msg := fmt.Sprintf("⏰ **TIME STOP — %s sudah 20 hari bursa!**\n\nSegera jual manual dengan `/sell %s`.", symbol, symbol)
					utils.SendMarkdownMessage(bot, msg)
					config.UpdatePlan(symbol, func(p *models.TradingPlan) {
						p.LastNotified = time.Now()
					})
					storage.SaveData()
				}
			}

			d := rules.Default.Evaluate(plan, yahooPrice)
			if d == rules.None {
				continue
			}

			// Cross-check Google (fallback Yahoo)
			realPrice := market.GetGooglePrice(symbol)
			if realPrice == 0 {
				realPrice = yahooPrice
			}

			d2 := rules.Default.Evaluate(plan, realPrice)
			if d2 == rules.None {
				continue
			}

			// Dedupe: lewati jika recent notification
			config.DataMutex.RLock()
			lastPlan := config.MyStocks[symbol]
			if time.Since(lastPlan.LastNotified) < config.EmergencyDelay {
				config.DataMutex.RUnlock()
				continue
			}
			config.DataMutex.RUnlock()

			// Bangun pesan
			msg := buildAlertMessage(symbol, plan, d2, realPrice)

			utils.SendMarkdownMessage(bot, msg)

			// Update LastNotified
			config.UpdatePlan(symbol, func(p *models.TradingPlan) {
				p.LastNotified = time.Now()
			})
			storage.SaveData()
		}

		// ==========================================
		// 2. PANTAU ANTREAN BARU (PendingOrders)
		// ==========================================
		config.DataMutex.RLock()
		orderSymbols := make([]string, 0, len(config.PendingOrders))
		for sym := range config.PendingOrders {
			orderSymbols = append(orderSymbols, sym)
		}
		config.DataMutex.RUnlock()

		for _, symbol := range orderSymbols {
			currentPrice := market.GetLivePrice(symbol)
			if currentPrice == 0 {
				continue
			}

			// --- KONDISI 1: AUTO MATCH (LANTAI TERSENTUH/JEBOL) ---
			config.DataMutex.RLock()
			order, exists := config.PendingOrders[symbol]
			config.DataMutex.RUnlock()
			if !exists {
				continue
			}

			if currentPrice <= order.OrderPrice {
				// Hapus dari antrean
				config.DataMutex.Lock()
				delete(config.PendingOrders, symbol)
				config.DataMutex.Unlock()

				// Cek apakah sudah punya saham ini
				config.DataMutex.RLock()
				existingPlan, hasExisting := config.MyStocks[symbol]
				config.DataMutex.RUnlock()

				if hasExisting {
					// Averaging
					totalOldCost := existingPlan.EntryPrice * float64(existingPlan.Lots)
					totalNewCost := order.OrderPrice * float64(order.Lot)
					newTotalLots := existingPlan.Lots + order.Lot
					newAvgPrice := (totalOldCost + totalNewCost) / float64(newTotalLots)

					plan := models.TradingPlan{
						Symbol:       symbol,
						EntryPrice:   newAvgPrice,
						Lots:         newTotalLots,
						HighestPrice: existingPlan.HighestPrice,
						BuyDate:      existingPlan.BuyDate,
						TakeProfit:   rules.Default.TP1Price(newAvgPrice),
						CutLoss:      rules.Default.DefaultCutLoss(newAvgPrice),
						TP1Done:      existingPlan.TP1Done,
					}

					config.DataMutex.Lock()
					config.MyStocks[symbol] = plan
					config.DataMutex.Unlock()
					storage.LogTrade(storage.TradeRecord{
						Action:     "BUY",
						Symbol:     symbol,
						Price:      order.OrderPrice,
						Lots:       order.Lot,
						PNLPercent: 0,
						PNLRp:      0,
						ExitReason: "",
						Note:       "Auto-Match (Averaging Down)",
					})
					storage.SaveData()

					totalModalBaru := newAvgPrice * float64(newTotalLots) * 100 * (1 + rules.Default.BuyFee)
					msg := fmt.Sprintf("🎯 **ANTREAN MATCH & AVERAGED!**\n\n"+
						"Emiten: **%s**\n"+
						"Antrean Beli: `Rp. %.0f` (%d Lot)\n\n"+
						"⚖️ **Status Portofolio Baru:**\n"+
						"Total Lot: `%d Lot`\n"+
						"Harga Rata-rata: `Rp. %.0f`\n"+
						"Total Modal: `%s` _(Termasuk Fee)_\n\n"+
						"_Radar Cut Loss & Trailing Stop telah disesuaikan!_ 🛡️",
						symbol, order.OrderPrice, order.Lot, newTotalLots, newAvgPrice, utils.FormatRupiah(totalModalBaru))
					utils.SendMarkdownMessage(bot, msg)
				} else {
					// Entry baru
					plan := models.TradingPlan{
						Symbol:       symbol,
						EntryPrice:   order.OrderPrice,
						HighestPrice: order.OrderPrice,
						Lots:         order.Lot,
						BuyDate:      time.Now().Format("2006-01-02"),
						TakeProfit:   rules.Default.TP1Price(order.OrderPrice),
						CutLoss:      rules.Default.DefaultCutLoss(order.OrderPrice),
					}

					config.DataMutex.Lock()
					config.MyStocks[symbol] = plan
					config.DataMutex.Unlock()
					storage.LogTrade(storage.TradeRecord{
						Action:     "BUY",
						Symbol:     symbol,
						Price:      order.OrderPrice,
						Lots:       order.Lot,
						PNLPercent: 0,
						PNLRp:      0,
						ExitReason: "",
						Note:       "Auto-Match dari Antrean",
					})
					storage.SaveData()

					totalModal := order.OrderPrice * float64(order.Lot) * 100 * (1 + rules.Default.BuyFee)
					msg := fmt.Sprintf("✅ **ANTREAN MATCHED!**\n\n"+
						"Emiten: **%s**\n"+
						"Harga Beli: `Rp. %.0f`\n"+
						"Jumlah: `%d Lot`\n"+
						"Modal Terpakai: `%s` _(Termasuk Fee)_\n\n"+
						"_Saham telah otomatis masuk ke portofolio aktif. Radar Cut Loss & Trailing Stop sekarang MENYALA._ 🛡️",
						symbol, order.OrderPrice, order.Lot, utils.FormatRupiah(totalModal))
					utils.SendMarkdownMessage(bot, msg)
				}
				continue
			}

			// --- KONDISI 2: HARGA KABUR (OPPORTUNITY COST) ---
			diffPercent := (currentPrice - order.OrderPrice) / order.OrderPrice

			if diffPercent >= rules.Default.RunawayPct/100 {
				config.DataMutex.RLock()
				currentOrder := config.PendingOrders[symbol]
				config.DataMutex.RUnlock()

				if time.Since(currentOrder.LastNotified) > 60*time.Minute {
					msg := fmt.Sprintf("🏃‍♂️💨 **HARGA KABUR BOS!**\n\n"+
						"Emiten: **%s**\n"+
						"Antreanmu: `Rp. %.0f`\n"+
						"Harga Sekarang: `Rp. %.0f` (Naik +%.1f%%)\n\n"+
						"Uangnya nganggur nih. Mending ditarik aja antreannya di aplikasi sekuritas.\n\n"+
						"👉 Ketik `/cancel_antre %s` untuk hapus dari pantauan bot.",
						symbol, order.OrderPrice, currentPrice, diffPercent*100, symbol)

					utils.SendMarkdownMessage(bot, msg)

					config.UpdateOrder(symbol, func(o *models.ActiveOrder) {
						o.LastNotified = time.Now()
					})
				}
			}
		}
	}
}

func buildAlertMessage(symbol string, plan models.TradingPlan, d rules.Decision, price float64) string {
	netPNL := rules.Default.NetPNL(plan.EntryPrice, price)

	switch d {
	case rules.CutLoss:
		return fmt.Sprintf("🚨 **CUT LOSS CONFIRMED!**\n\nEmiten: **%s**\nReal PNL: `%.2f%%`\nKetik `/sell %s`.", symbol, netPNL, symbol)

	case rules.TrailingStop:
		tslPrice := rules.Default.TrailingStopPrice(plan.HighestPrice)
		return fmt.Sprintf("🛡️ **TRAILING STOP TERSENTUH!** ✅\n\nEmiten: **%s**\nBatas TSL: `%s`\nCuan Aman: `+%.2f%%`\nKetik `/sell %s`.", symbol, utils.FormatRupiah(tslPrice), netPNL, symbol)

	case rules.TakeProfit1:
		// Jual 50% lot (minimal 1)
		sellLots := plan.Lots / 2
		if sellLots < 1 {
			sellLots = plan.Lots
		}
		if sellLots == plan.Lots {
			return fmt.Sprintf("🎯 **TAKE PROFIT CONFIRMED!**\n\nEmiten: **%s**\nReal PNL: `+%.2f%%`\nLot tersisa sedikit —建议 jual semua.\nKetik `/sell %s`.", symbol, netPNL, symbol)
		}
		return fmt.Sprintf("🎯 **TAKE PROFIT CONFIRMED!**\n\nEmiten: **%s**\nReal PNL: `+%.2f%%`\nKetik `/sell %s %d`.", symbol, netPNL, symbol, sellLots)
	}

	return ""
}
