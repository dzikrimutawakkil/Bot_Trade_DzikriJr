package research

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"learn-go/internal/config"
	"learn-go/internal/market"
	"learn-go/internal/models"
	"learn-go/internal/rules"
	"learn-go/internal/signals"
	"learn-go/internal/utils"
)

func ProcessResearchCommand(bot *tgbotapi.BotAPI, args []string) {
	if len(args) != 2 {
		utils.SendSimpleMessage(bot, "❌ Format salah! Gunakan: `/research [KODE]`")
		return
	}
	symbol := strings.ToUpper(args[1])
	utils.SendSimpleMessage(bot, fmt.Sprintf("🧐 Memulai Deep Research untuk %s... (Mohon tunggu sebentar)", symbol))

	news, err := FetchNewsRSS(symbol)
	if err != nil {
		utils.SendSimpleMessage(bot, "❌ Gagal mengambil berita.")
		return
	}

	technicalData, errTech := FetchTechnicalData(symbol)
	if errTech != nil {
		utils.SendSimpleMessage(bot, fmt.Sprintf("❌ Gagal mengambil data teknikal untuk %s.", symbol))
		return
	}
	_, _, _, ma20 := GetStockScore(symbol)

	analysis, err := GetDeepAnalysis(symbol, news, technicalData)
	if err != nil {
		utils.SendSimpleMessage(bot, fmt.Sprintf("⚠️ Gagal mengambil analisis untuk %s.\nCoba lagi nanti atau hubungi admin.", symbol))
		return
	}

	currentPrice := market.GetLivePrice(symbol)
	if currentPrice == 0 {
		currentPrice = market.GetGooglePrice(symbol)
	}

	planText := ""
	if currentPrice > 0 {
		if ma20 <= 0 {
			ma20 = currentPrice
		}

		hist, _ := market.GetHistoricalPrices(symbol)
		signalLow := ma20 * 0.98
		if len(hist.Lows) > 0 {
			signalLow = hist.Lows[len(hist.Lows)-1]
		}

		_, out, ok := buildResearchPlanText(symbol, currentPrice, signalLow, ma20)
		if ok {
			in := rules.PlanInput{
				CurrentPrice: currentPrice,
				SignalLow:    signalLow,
				TotalCapital: config.TotalModalTrading,
				CapitalInUse: config.CapitalInUse(),
				MarketSafe:   true,
			}
			isMarketSafe, _ := GetMarketFilterStatus()
			in.MarketSafe = isMarketSafe

			planText = buildPlanText(symbol, in, out, true)
		} else {
			planText = "\n\n⚠️ **WARNING: DATA HARGA OFFLINE**\n_(Kalkulator Position Sizing tidak dapat ditampilkan.)_\n"
		}
	} else {
		planText = "\n\n⚠️ **WARNING: DATA HARGA OFFLINE**\n_(Kalkulator Position Sizing tidak dapat ditampilkan karena bot gagal menarik harga real-time dari bursa saat ini. Silakan coba lagi nanti.)_\n"
	}

	response := fmt.Sprintf("🔍 **Hasil Deep Research: %s**\n\n%s\n%s", symbol, analysis, planText)
	utils.SendMarkdownMessage(bot, response)
}

func ProcessS08Recommendation(bot *tgbotapi.BotAPI) {
	utils.SendSimpleMessage(bot, "📊 Scanning S08 Momentum (MA5 > MA20 > MA50)...")

	today := time.Now()

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
		msg := "📉 **TIDAK ADA SETUP S08 HARI INI**\n\n" +
			"Tidak ada saham yang memenuhi filter:\n" +
			"• MA5 > MA20 > MA50 (uptrend)\n" +
			"• Volatilitas < 6%\n\n" +
			"_Cash is King!_ 👑"
		utils.SendMarkdownMessage(bot, msg)
		return
	}

	// Update with live prices
	for i := range candidates {
		p := market.GetLivePrice(candidates[i].Symbol)
		if p == 0 {
			p = candidates[i].Close
		}
		candidates[i].Close = p
	}

	var sb strings.Builder
	sb.WriteString("💰 **S08 MOMENTUM SCAN** 💰\n\n")
	sb.WriteString("Filter: MA5 > MA20 > MA50 | Volatilitas < 6%\n\n")

	for i, c := range candidates {
		perf := rules.Default.NetPNL(c.Close, c.Close) // 0 at entry
		_ = perf
		ema := fmt.Sprintf("MA5: %.0f | MA20: %.0f | MA50: %.0f\n", c.MA5, c.MA20, c.MA50)
		momentum := fmt.Sprintf("Momentum: +%.2f%% dari MA5\n", c.MomentumPct)

		sb.WriteString(fmt.Sprintf("━━━━━━ 🔥 %d. %s ━━━━━━\n", i+1, c.Symbol))
		sb.WriteString(fmt.Sprintf("Harga  : Rp %.0f\n", c.Close))
		sb.WriteString(ema)
		sb.WriteString(momentum)

		alloc := config.RemainingCapital() / float64(len(candidates))
		costPerLot := c.Close * 100 * (1 + rules.Default.BuyFee)
		lots := int(alloc / costPerLot)
		if lots < 1 {
			lots = 1
		}
		modal := c.Close * float64(lots) * 100 * (1 + rules.Default.BuyFee)
		tp1 := rules.Default.TP1Price(c.Close)
		cl := rules.Default.DefaultCutLoss(c.Close)

		sb.WriteString(fmt.Sprintf("Lot    : %d (Modal ~%s)\n", lots, utils.FormatRupiah(modal)))
		sb.WriteString(fmt.Sprintf("TP1    : %s (+5%%)\n", utils.FormatRupiah(tp1)))
		sb.WriteString(fmt.Sprintf("CL     : %s (-3%%)\n\n", utils.FormatRupiah(cl)))
	}

	bot.Send(tgbotapi.NewMessage(config.MyChatID, sb.String()))
}

func ProcessNews(bot *tgbotapi.BotAPI, topStocks []string) {
	var sb strings.Builder
	sb.WriteString("📰 **Berita Khusus Saham Rekomendasi** 📰\n\n")

	for i, symbol := range topStocks {
		stockbitURL := fmt.Sprintf("https://stockbit.com/symbol/%s", symbol)
		googleNewsURL := fmt.Sprintf("https://www.google.com/search?q=berita+saham+%s&tbm=nws", symbol)

		sb.WriteString(fmt.Sprintf("🔥 **%d. %s**\n", i+1, symbol))
		sb.WriteString(fmt.Sprintf("   🔗 [Stockbit](%s) | [Google News](%s)\n\n", stockbitURL, googleNewsURL))
	}

	sb.WriteString("----------\n_Tips: Cek sentimen pasar dulu!_")

	msg := tgbotapi.NewMessage(config.MyChatID, sb.String())
	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = true
	bot.Send(msg)
}

func ExtractSentimentScore(text string) float64 {
	re := regexp.MustCompile(`(?i)Skor Sentimen[:\*]*\s*([0-9]+(?:\.[0-9]+)?)/10`)
	match := re.FindStringSubmatch(text)
	if len(match) > 1 {
		val, _ := strconv.ParseFloat(match[1], 64)
		return val
	}
	return 0
}
