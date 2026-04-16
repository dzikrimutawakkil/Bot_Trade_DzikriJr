package research

import (
	"fmt"
	"sort"
	"strings"
	"strconv"
	"regexp"
	"time"
	"math"
	"log"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"learn-go/internal/config"
	"learn-go/internal/utils"
	"learn-go/internal/models"
	"learn-go/internal/market"
)

func ProcessResearchCommand(bot *tgbotapi.BotAPI, args []string) {
	if len(args) != 2 {
		utils.SendSimpleMessage(bot, "❌ Format salah! Gunakan: `/research [KODE]`")
		return
	}
	symbol := strings.ToUpper(args[1])
	utils.SendSimpleMessage(bot, fmt.Sprintf("🧐 Memulai Deep Research untuk %s... (Mohon tunggu sebentar)", symbol))
	
	// 1. Fetch Berita
	news, err := FetchNewsRSS(symbol)
	if err != nil {
		utils.SendSimpleMessage(bot, "❌ Gagal mengambil berita.")
		return
	}

	// 2. Fetch Teknikal untuk AI & Ambil Angka MA20
	technicalData := FetchTechnicalData(symbol)
	_, _, _, ma20 := GetStockScore(symbol) // <-- AMBIL MA20 UNTUK LOGIKA BoW

	// 3. Analisis AI Gemini
	analysis, err := GetDeepAnalysis(symbol, news, technicalData)
	if err != nil {
		utils.SendSimpleMessage(bot, "❌ Gagal melakukan analisis AI.")
		return
	}

	// 4. KALKULATOR POSITION SIZING (SINKRON DENGAN RECOMMENDATION)
	currentPrice := market.GetLivePrice(symbol)
	if currentPrice == 0 {
		currentPrice = market.GetGooglePrice(symbol) // Fallback
	}

	var planText string
	if currentPrice > 0 {
		// Fallback jika API MA20 gagal/kosong
		if ma20 <= 0 {
			ma20 = currentPrice 
		}

		// --- LOGIKA BARU: CONFIRMED BoW (C-BoW) ---
		// Cari titik ekor bawah (Low) untuk Cut Loss Struktural
		hist, errHist := market.GetHistoricalPrices(symbol)
		signalLow := ma20 * 0.98 // Fallback jika gagal ambil historis
		if errHist == nil && len(hist.Lows) > 0 {
			signalLow = hist.Lows[len(hist.Lows)-1]
		}

		// LOGIKA CUT LOSS STRUKTURAL: 1% di bawah ekor candle
		cutLossRaw := signalLow * 0.99
		cutLossPrice := utils.RoundToFraction(cutLossRaw)

		// AREA BELI: Karena sudah konfirmasi, mepet harga sekarang (Buy On Close)
		idealBuyMin := utils.RoundToFraction(currentPrice * 0.99)
		idealBuyMax := utils.RoundToFraction(currentPrice)

		lossPerLot := (currentPrice - cutLossPrice) * 100
		if lossPerLot <= 0 {
			lossPerLot = 1
		}

		// 1. Hitung Jatah Risiko (Risk Limit)
		maxRiskRupiah := config.TotalModalTrading * config.MaxRiskPerTrade
		isMarketSafe, _ := GetMarketFilterStatus()
		warningDefensif := ""
		if !isMarketSafe {
			maxRiskRupiah = maxRiskRupiah / 2.0
			warningDefensif = " 🛡️ _(Defensive Mode)_"
		}

		maxLotsByRisk := math.Floor(maxRiskRupiah / lossPerLot)
		if maxLotsByRisk < 0 {
			maxLotsByRisk = 0
		}

		// 2. Hitung berdasarkan sisa Cash Limit
		hargaSatuLot := currentPrice * 100 * (1 + config.BuyFee)
		maxLotsByCash := math.Floor(config.TotalModalTrading / hargaSatuLot)
		if maxLotsByCash < 0 {
			maxLotsByCash = 0
		}

		maxLots := int(math.Min(maxLotsByRisk, maxLotsByCash))

		// 3. Hitung Target Profit (Hit & Run 4%)
		tp1Price := utils.RoundToFraction(currentPrice * 1.04)

		planText = fmt.Sprintf(`
		━━━━━━━━━━━━━━━━━━
		📐 **TRADING PLAN (Max %d LOT)**%s
		📍 **Harga Terkonfirmasi** : Rp %.0f
		🧱 **Support Ekor (Low)** : Rp %.0f

		🎯 **AREA BELI : Rp %.0f - Rp %.0f**
		_(Sinyal Valid! Hajar Kanan sebagian sebelum market tutup)_

		🚨 **Batas Cut Loss** : Rp %.0f (Jebol Ekor Bawah)
		🚀 **TP 1 (Jual 50%% Lot)** : Rp %.0f (Amankan Modal)
		🛡️ **TP 2 (Sisa 50%%)** : Let It Ride (Trailing Stop 3%%)
		━━━━━━━━━━━━━━━━━━

		👉 Ketik /buy %s %.0f %d jika ingin eksekusi.`, 
		maxLots, warningDefensif, currentPrice, signalLow, 
		idealBuyMin, idealBuyMax, cutLossPrice, 
		tp1Price, symbol, currentPrice, maxLots)
	} else {
		planText = "\n\n_(Gagal menarik harga live untuk kalkulasi Position Sizing)_\n"
	}

	// 5. Gabungkan Hasil AI dengan Kalkulator
	response := fmt.Sprintf("🔍 **Hasil Deep Research: %s**\n\n%s\n%s", symbol, analysis, planText)
	
	utils.SendMarkdownMessage(bot, response)
}

func ProcessRecommendation(bot *tgbotapi.BotAPI) {
	// --- FITUR BARU: MARKET FILTER IHSG ---
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

	// SORTING SEMENTARA: Urutkan teknikalnya dulu untuk mencari Top 5
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].DistToMA < results[j].DistToMA
		}
		return results[i].Score > results[j].Score
	})

	// 🔥 FILTERING PINTAR (The Hunter Algorithm) 🔥
	var finalCandidates []models.Recommendation
	aiCallCount := 0
	maxAICalls := 10      // Sabuk pengaman 1: Maksimal nanya AI 10 kali agar API tidak limit
	targetSetups := 3     // Sabuk pengaman 2: Kita cukup cari 3 saham "BELI" terbaik

	for _, res := range results {

		log.Printf("Evaluasi %s - Score: %.0f, Status: %s, Jarak ke MA20: %.2f%%\n", res.Symbol, res.Score, res.Status, res.DistToMA)

		// Hanya proses saham yang secara teknikal bagus (Skor >= 8)
		if res.Score >= 8 {

			// Kalau sudah terlalu banyak nanya AI, hentikan pencarian
			if aiCallCount >= maxAICalls {
				break
			}

			// 1. Tanya AI (Deep Research)
			news, _ := FetchNewsRSS(res.Symbol)
			tech := FetchTechnicalData(res.Symbol)
			analysis, err := GetDeepAnalysis(res.Symbol, news, tech)
			aiCallCount++

			if err == nil && analysis != "" {
				upperAnalysis := strings.ToUpper(analysis)

				// 2. Langsung Cek Apakah AI merekomendasikan "BELI"
				if strings.Contains(upperAnalysis, "REKOMENDASI: BELI") || strings.Contains(analysis, "🟢") {
					res.DeepAnalysis = analysis
					res.Sentiment = extractSentimentScore(analysis)
					finalCandidates = append(finalCandidates, res)

					// 3. EARLY EXIT: Kalau sudah dapat 3 saham incaran, LANGSUNG BERHENTI!
					if len(finalCandidates) >= targetSetups {
						break
					}
				}
			} else {
				fmt.Printf("⚠️ Gagal mendapat AI untuk %s: %v\n", res.Symbol, err)
			}

			// 4. Jeda untuk menghindari Error 429 dari Google
			if len(finalCandidates) < targetSetups && aiCallCount < maxAICalls {
				time.Sleep(30 * time.Second) // Gunakan 15 detik jika pakai API Key baru
			}
		}
	}

	// Jika setelah di-filter ternyata tidak ada satupun yang layak beli
	if len(finalCandidates) == 0 {
		pesanKosong := "📉 **TIDAK ADA SETUP BELI HARI INI**\n\nBerdasarkan filter teknikal dan validasi AI, tidak ada satupun saham yang memberikan sinyal pantulan / Hammer valid hari ini.\n\n_Cash is King!_ Simpan pelurumu untuk hari esok. 👑"
		utils.SendMarkdownMessage(bot, pesanKosong)
		return
	}

	// SORTING FINAL: Urutkan ulang berdasarkan Sentimen AI
	sort.Slice(finalCandidates, func(i, j int) bool {
		if finalCandidates[i].Sentiment != finalCandidates[j].Sentiment {
			return finalCandidates[i].Sentiment > finalCandidates[j].Sentiment
		}
		return finalCandidates[i].DistToMA < finalCandidates[j].DistToMA
	})

	// RANGKUM PESAN BALASAN
	var sb strings.Builder
	var topSymbols []string
	sb.WriteString("💰 **TOP Saham Rekomendasi (CONFIRMED BoW)** 💰")

	count := 0
	for _, res := range finalCandidates {
		if count >= 3 {
			break
		}

		emoji := "🔥" // Karena finalCandidates pasti score >= 8
		sb.WriteString(fmt.Sprintf("\n\n━━━━━━ %s **%s** %s ━━━━━━\n", emoji, res.Symbol, emoji))

		cleanAnalysis := strings.Replace(res.DeepAnalysis, "Hasil Deep Research:", "", 1)
		cleanAnalysis = strings.Replace(cleanAnalysis, "🔍", "", 1)
		sb.WriteString(fmt.Sprintf("%s\n", strings.TrimSpace(cleanAnalysis)))

		// --- KALKULATOR POSITION SIZING (VERSI CONFIRMED BoW) ---
		currentPrice := market.GetLivePrice(res.Symbol)
		if currentPrice == 0 {
			currentPrice = market.GetGooglePrice(res.Symbol)
		}

		if currentPrice > 0 {
			// Cari titik ekor bawah (Low) hari ini
			hist, errHist := market.GetHistoricalPrices(res.Symbol)
			signalLow := res.MA20 * 0.98 // Fallback
			if errHist == nil && len(hist.Lows) > 0 {
				signalLow = hist.Lows[len(hist.Lows)-1]
			}

			// LOGIKA CUT LOSS STRUKTURAL
			cutLossRaw := signalLow * 0.99
			cutLossPrice := utils.RoundToFraction(cutLossRaw) 

			// AREA BELI: Mepet harga saat ini
			idealBuyMin := utils.RoundToFraction(currentPrice * 0.99) 
			idealBuyMax := utils.RoundToFraction(currentPrice) 

			lossPerLot := (currentPrice - cutLossPrice) * 100
			if lossPerLot <= 0 {
				lossPerLot = 1
			}

			// 1. Hitung Jatah Risiko (Risk Limit)
			maxRiskRupiah := config.TotalModalTrading * config.MaxRiskPerTrade

			warningDefensif := ""
			if !isMarketSafe {
				maxRiskRupiah = maxRiskRupiah / 2.0
				warningDefensif = " 🛡️ _(Defensive Mode)_"
			}

			maxLotsByRisk := math.Floor(maxRiskRupiah / lossPerLot)
			if maxLotsByRisk < 0 {
				maxLotsByRisk = 0
			}

			// 2. Hitung berdasarkan sisa Cash Limit
			hargaSatuLot := currentPrice * 100 * (1 + config.BuyFee)
			maxLotsByCash := math.Floor(config.TotalModalTrading / hargaSatuLot)
			if maxLotsByCash < 0 {
				maxLotsByCash = 0
			}

			maxLots := int(math.Min(maxLotsByRisk, maxLotsByCash))
			
			// 3. Hitung Take Profit Hit & Run
			tp1Price := utils.RoundToFraction(currentPrice * 1.04)

			tradingPlanText := fmt.Sprintf(`
			📐 **TRADING PLAN (Max %d LOT)**%s
			━━━━━━━━━━━━━━━━━━
			📍 **Harga Terkonfirmasi** : Rp %.0f
			🧱 **Support Ekor (Low)** : Rp %.0f

			🎯 **AREA BELI : Rp %.0f - Rp %.0f**
			_(Sinyal Valid! Hajar Kanan sebagian sebelum market tutup)_

			🚨 **Batas Cut Loss** : Rp %.0f (Jebol Ekor Bawah)
			🚀 **TP 1 (Jual 50%% Lot)** : Rp %.0f (Amankan Modal)
			🛡️ **TP 2 (Sisa 50%% Lot)** : Trailing Stop 3%%
			━━━━━━━━━━━━━━━━━━
			`, maxLots, warningDefensif, currentPrice, signalLow, idealBuyMin, idealBuyMax, cutLossPrice, tp1Price)

			sb.WriteString(tradingPlanText)
		} else {
			sb.WriteString("\n_(Gagal menghitung Plan, data harga offline)_\n\n")
		}
		// --------------------------------------------------------

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

func extractSentimentScore(text string) float64 {
	re := regexp.MustCompile(`(?i)Skor Sentimen[:\*]*\s*([0-9]+(?:\.[0-9]+)?)/10`)
	match := re.FindStringSubmatch(text)
	if len(match) > 1 {
		val, _ := strconv.ParseFloat(match[1], 64)
		return val
	}
	return 0 // Jika gagal ekstrak, sentimen dianggap 0
}