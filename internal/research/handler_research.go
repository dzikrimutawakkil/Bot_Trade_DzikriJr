package research

import (
	"fmt"
	"sort"
	"strings"
	"strconv"
	"regexp"
	"time"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"learn-go/internal/config"
	"learn-go/internal/utils"
	"learn-go/internal/models"
)

// Logika /research yang tadinya di dalam loop
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

	technicalData := FetchTechnicalData(symbol)

	analysis, err := GetDeepAnalysis(symbol, news, technicalData)
	if err != nil {
		utils.SendSimpleMessage(bot, "❌ Gagal melakukan analisis AI.")
		return
	}

	response := fmt.Sprintf("🔍 **Hasil Deep Research: %s**\n\n%s", symbol, analysis)
	utils.SendMarkdownMessage(bot, response)
}

func ProcessRecommendation(bot *tgbotapi.BotAPI) {
	var results []models.Recommendation

	// Ganti pesan loading karena prosesnya sekarang lebih berat
	utils.SendSimpleMessage(bot, "⏳ Proses sortir LQ45 sedang berlangsung...")

	for _, s := range config.Watchlist {
		score, status, distToMA := GetStockScore(s) //
		if score > 0 {
			results = append(results, models.Recommendation{
				Symbol:   s,
				Score:    score,
				Status:   status,
				DistToMA: distToMA,
			})
		}
	}

	// SORTING SEMENTARA: Urutkan teknikalnya dulu untuk mencari Top 5
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].DistToMA < results[j].DistToMA //
		}
		return results[i].Score > results[j].Score //
	})

	// Potong maksimal 5 saham teratas agar call API AI tidak kepanjangan
	limit := 5
	if len(results) < limit {
		limit = len(results)
	}
	topCandidates := results[:limit]

	// INTEGRASI DEEP RESEARCH (Hanya untuk yang Hijau / Skor >= 8)
	for i, res := range topCandidates {
		if res.Score >= 8 {
			news, _ := FetchNewsRSS(res.Symbol)
			tech := FetchTechnicalData(res.Symbol)
			analysis, err := GetDeepAnalysis(res.Symbol, news, tech)
			
			if err == nil && analysis != "" {
				topCandidates[i].DeepAnalysis = analysis
				topCandidates[i].Sentiment = extractSentimentScore(analysis)
			} else {
				fmt.Printf("⚠️ Gagal mendapat AI untuk %s: %v\n", res.Symbol, err)
			}
			
			// WAJIB: Kasih jeda 3 detik agar server AI tidak memblokir bot karena spam
			time.Sleep(3 * time.Second)
		}
	}

	// SORTING FINAL: Urutkan ulang berdasarkan Sentimen AI
	sort.Slice(topCandidates, func(i, j int) bool {
		// Jika keduanya Hijau (Skor 10), sentimen terbesar menang
		if topCandidates[i].Score >= 8 && topCandidates[j].Score >= 8 {
			if topCandidates[i].Sentiment != topCandidates[j].Sentiment {
				return topCandidates[i].Sentiment > topCandidates[j].Sentiment
			}
			return topCandidates[i].DistToMA < topCandidates[j].DistToMA
		}
		// Kalau selain Hijau, kembalikan ke aturan teknikal biasa
		if topCandidates[i].Score == topCandidates[j].Score {
			return topCandidates[i].DistToMA < topCandidates[j].DistToMA
		}
		return topCandidates[i].Score > topCandidates[j].Score
	})

	// RANGKUM PESAN BALASAN
	var sb strings.Builder
	var topSymbols []string
	sb.WriteString("💰 **TOP 3 Saham Rekomendasi** 💰\n\n")

	count := 0
	for _, res := range topCandidates {
		if count >= 3 {
			break
		}

		emoji := "⭐"
		if res.Score >= 8 {
			emoji = "🔥"
		}

		// Header saham yang minimalis seperti seleramu
		sb.WriteString(fmt.Sprintf("━━━━━━ %s **%s** %s ━━━━━━\n\n", emoji, res.Symbol, emoji,))
		
		if res.Score >= 8 && res.DeepAnalysis != "" {
			cleanAnalysis := strings.Replace(res.DeepAnalysis, "Hasil Deep Research:", "", 1)
			cleanAnalysis = strings.Replace(cleanAnalysis, "🔍", "", 1)
			sb.WriteString(fmt.Sprintf("%s\n\n\n", strings.TrimSpace(cleanAnalysis)))
		} else {
			sb.WriteString(fmt.Sprintf("%s\n\n\n", res.Status))
		}

		topSymbols = append(topSymbols, res.Symbol)
		count++
	}

	if count == 0 {
		utils.SendSimpleMessage(bot, "Pasar lagi kurang oke, Dzik. Pantau RDPU dulu.")
		return
	}

	dataBerita := "news:" + strings.Join(topSymbols, ",") //
	btn := tgbotapi.NewInlineKeyboardButtonData("📰 Cek Berita Top 3", dataBerita) //
	keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(btn)) //

	msg := tgbotapi.NewMessage(config.MyChatID, sb.String()) //
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