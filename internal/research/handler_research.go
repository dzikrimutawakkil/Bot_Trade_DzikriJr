package research

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"learn-go/internal/config"
	"learn-go/internal/market"
	"learn-go/internal/models"
	"learn-go/internal/rules"
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

func ProcessRecommendation(bot *tgbotapi.BotAPI) {
	utils.SendSimpleMessage(bot, "🔎 Mengecek kondisi angin IHSG (Market Trend)...")

	isMarketSafe, marketStatusMsg := GetMarketFilterStatus()
	utils.SendMarkdownMessage(bot, marketStatusMsg)

	utils.SendSimpleMessage(bot, "⏳ Proses sortir watchlist sedang berlangsung...")

	var results []models.Recommendation

	for _, s := range config.Watchlist {
		score, status, distToMA, ma20 := GetStockScore(s)
		if score > 0 {
			results = append(results, models.Recommendation{
				Symbol:   s,
				Score:    score,
				Status:   status,
				DistToMA: distToMA,
				MA20:     ma20,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].DistToMA < results[j].DistToMA
		}
		return results[i].Score > results[j].Score
	})

	var finalCandidates []models.Recommendation
	aiCallCount := 0
	maxAICalls := 10
	targetSetups := 3

	for _, res := range results {
		log.Printf("Evaluasi %s - Score: %.0f, Status: %s, Jarak ke MA20: %.2f%%\n", res.Symbol, res.Score, res.Status, res.DistToMA)

		if res.Score >= 8 {
			if aiCallCount >= maxAICalls {
				break
			}

			news, _ := FetchNewsRSS(res.Symbol)
			tech, _ := FetchTechnicalData(res.Symbol)
			analysis, err := GetDeepAnalysis(res.Symbol, news, tech)
			aiCallCount++

			if err == nil && analysis != "" {
				upperAnalysis := strings.ToUpper(analysis)
				if strings.Contains(upperAnalysis, "REKOMENDASI: BELI") || strings.Contains(analysis, "🟢") {
					res.DeepAnalysis = analysis
					res.Sentiment = ExtractSentimentScore(analysis)
					finalCandidates = append(finalCandidates, res)

					if len(finalCandidates) >= targetSetups {
						break
					}
				}
			} else {
				utils.SendSimpleMessage(bot, fmt.Sprintf("⚠️ Gagal mengambil analisis untuk %s. Coba lagi nanti.", res.Symbol))
			}

			if len(finalCandidates) < targetSetups && aiCallCount < maxAICalls {
				time.Sleep(30 * time.Second)
			}
		}
	}

	if len(finalCandidates) == 0 {
		pesanKosong := "📉 **TIDAK ADA SETUP BELI HARI INI**\n\nBerdasarkan filter teknikal dan validasi AI, tidak ada satupun saham yang memberikan sinyal pantulan / Hammer valid hari ini.\n\n_Cash is King!_ Simpan pelurumu untuk hari esok. 👑"
		utils.SendMarkdownMessage(bot, pesanKosong)
		return
	}

	sort.Slice(finalCandidates, func(i, j int) bool {
		if finalCandidates[i].Sentiment != finalCandidates[j].Sentiment {
			return finalCandidates[i].Sentiment > finalCandidates[j].Sentiment
		}
		return finalCandidates[i].DistToMA < finalCandidates[j].DistToMA
	})

	var sb strings.Builder
	var topSymbols []string
	sb.WriteString("💰 **TOP Saham Rekomendasi (CONFIRMED BoW)** 💰")

	count := 0
	for _, res := range finalCandidates {
		if count >= 3 {
			break
		}

		emoji := "🔥"
		sb.WriteString(fmt.Sprintf("\n\n━━━━━━ %s **%s** %s ━━━━━━\n", emoji, res.Symbol, emoji))

		cleanAnalysis := strings.Replace(res.DeepAnalysis, "Hasil Deep Research:", "", 1)
		cleanAnalysis = strings.Replace(cleanAnalysis, "🔍", "", 1)
		sb.WriteString(fmt.Sprintf("%s\n", strings.TrimSpace(cleanAnalysis)))

		currentPrice := market.GetLivePrice(res.Symbol)
		if currentPrice == 0 {
			currentPrice = market.GetGooglePrice(res.Symbol)
		}

		if currentPrice > 0 {
			signalLow := res.MA20 * 0.98
			hist, _ := market.GetHistoricalPrices(res.Symbol)
			if len(hist.Lows) > 0 {
				signalLow = hist.Lows[len(hist.Lows)-1]
			}

			in := rules.PlanInput{
				CurrentPrice: currentPrice,
				SignalLow:    signalLow,
				TotalCapital: config.TotalModalTrading,
				CapitalInUse: config.CapitalInUse(),
				MarketSafe:   isMarketSafe,
			}
			out := rules.Default.SuggestPlan(in)
			sb.WriteString(buildPlanText(res.Symbol, in, out, true))
		} else {
			sb.WriteString("\n_(Gagal menghitung Plan, data harga offline)_\n\n")
		}

		topSymbols = append(topSymbols, res.Symbol)
		count++
	}

	dataBerita := "news:" + strings.Join(topSymbols, ",")
	btn := tgbotapi.NewInlineKeyboardButtonData("📰 Cek Berita Rekomendasi", dataBerita)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(btn))

	msg := tgbotapi.NewMessage(config.MyChatID, sb.String())
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
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
