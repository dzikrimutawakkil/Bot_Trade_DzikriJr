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
	"learn-go/internal/signals"
	"learn-go/internal/storage"
	"learn-go/internal/utils"
)

// ProcessWeeklyRebalance runs every Friday at 18:05.
// It exits broken-trend positions and opens new S08 momentum candidates in the live portfolio.
func ProcessWeeklyRebalance(bot *tgbotapi.BotAPI) {
	log.Println("[Portfolio] Processing weekly rebalance (S08 Momentum)...")

	today := time.Now()

	// 1. Exit broken-trend positions
	config.DataMutex.RLock()
	symbols := make([]string, 0, len(config.MyStocks))
	for sym := range config.MyStocks {
		symbols = append(symbols, sym)
	}
	config.DataMutex.RUnlock()

	for _, sym := range symbols {
		bars, err := market.GetBars(sym, "1mo")
		if err != nil || len(bars) < 50 {
			continue
		}
		clean := market.CleanBars(bars)
		if len(clean) < 50 {
			continue
		}

		if signals.TrendBroken(clean, today) {
			price := market.GetLivePrice(sym)
			if price == 0 {
				price = market.GetGooglePrice(sym)
			}
			if price == 0 {
				continue
			}

			config.DataMutex.RLock()
			plan := config.MyStocks[sym]
			config.DataMutex.RUnlock()

			netPNL := rules.Default.NetPNL(plan.EntryPrice, price)
			totalSell := price * float64(plan.Lots) * 100 * (1 - rules.Default.SellFee)
			totalBuy := plan.EntryPrice * float64(plan.Lots) * 100 * (1 + rules.Default.BuyFee)
			pnlRp := totalSell - totalBuy

			storage.LogTrade(storage.TradeRecord{
				Action:     "SELL",
				Symbol:     sym,
				Price:      price,
				Lots:       plan.Lots,
				PNLPercent: netPNL,
				PNLRp:      pnlRp,
				ExitReason: "REBALANCE",
				Note:       "Trend Break (MA5 < MA20)",
			})

			config.DataMutex.Lock()
			delete(config.MyStocks, sym)
			config.DataMutex.Unlock()
			storage.SaveData()

			emoji := "🟢"
			if pnlRp < 0 {
				emoji = "🔴"
			}
			msg := emoji + " **TREND BREAK — %s di-exit!**\nPnL: %.2f%% / Rp %+.0f"
			utils.SendMarkdownMessage(bot, fmt.Sprintf(msg, sym, netPNL, pnlRp))
		}
	}

	// 2. Scan for new S08 momentum candidates
	allBars := make(map[string][]models.Bar)
	for _, sym := range config.Watchlist {
		bars, err := market.GetBars(sym, "3mo")
		if err != nil {
			continue
		}
		clean := market.CleanBars(bars)
		if len(clean) >= 50 {
			allBars[sym] = clean
		}
	}

	candidates := signals.SelectWeeklyCandidates(allBars, config.Watchlist, today, 4)
	if len(candidates) == 0 {
		log.Println("[Portfolio] No S08 candidates this week.")
		return
	}

	// Update prices
	for i := range candidates {
		p := market.GetLivePrice(candidates[i].Symbol)
		if p == 0 {
			p = candidates[i].Close
		}
		candidates[i].Close = p
	}

	// 3. Open positions for candidates not already held
	for _, c := range candidates {
		config.DataMutex.RLock()
		_, alreadyHeld := config.MyStocks[c.Symbol]
		config.DataMutex.RUnlock()

		if alreadyHeld {
			continue
		}

		// Equal weight allocation
		available := config.RemainingCapital()
		n := len(candidates)
		alloc := available / float64(n)

		costPerLot := c.Close * 100 * (1 + rules.Default.BuyFee)
		lots := int(alloc / costPerLot)
		if lots < 1 {
			continue
		}

		if c.Close <= 0 {
			continue
		}

		plan := models.TradingPlan{
			Symbol:       c.Symbol,
			EntryPrice:   c.Close,
			HighestPrice: c.Close,
			Lots:         lots,
			BuyDate:      today.Format("2006-01-02"),
			TakeProfit:   rules.Default.TP1Price(c.Close),
			CutLoss:      rules.Default.DefaultCutLoss(c.Close),
			HoldDays:     0,
			TP1Done:      false,
		}

		config.DataMutex.Lock()
		config.MyStocks[c.Symbol] = plan
		config.DataMutex.Unlock()

		storage.LogTrade(storage.TradeRecord{
			Action:     "BUY",
			Symbol:     c.Symbol,
			Price:      c.Close,
			Lots:       lots,
			PNLPercent: 0,
			PNLRp:      0,
			ExitReason: "",
			Note:       "S08 Weekly Entry",
		})
		storage.SaveData()

		modal := c.Close * float64(lots) * 100 * (1 + rules.Default.BuyFee)
		msg := "🚀 **S08 MOMENTUM ENTRY!**\n\n" +
			"Emiten: **" + c.Symbol + "**\n" +
			"Harga  : Rp " + utils.FormatRupiah(c.Close) + "\n" +
			"Lot    : " + fmt.Sprintf("%d", lots) + "\n" +
			"Modal  : " + utils.FormatRupiah(modal) + "\n\n" +
			"📈 Uptrend: MA5 > MA20 > MA50 confirmed\n" +
			"🛡️ CL: " + utils.FormatRupiah(plan.CutLoss) + " | TP1: " + utils.FormatRupiah(plan.TakeProfit)
		utils.SendMarkdownMessage(bot, msg)

		log.Printf("[Portfolio] S08 Entry: %s @ %.0f (%d lot)", c.Symbol, c.Close, lots)
	}

	log.Printf("[Portfolio] Weekly rebalance done. Active positions: %d", len(config.MyStocks))
}
