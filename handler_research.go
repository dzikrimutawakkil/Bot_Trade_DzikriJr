package main

import (
	"fmt"
	"sort"
	"strings"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Logika /research yang tadinya di dalam loop
func processResearchCommand(bot *tgbotapi.BotAPI, args []string) {
	if len(args) != 2 {
		sendSimpleMessage(bot, "❌ Format salah! Gunakan: `/research [KODE]`")
		return
	}
	symbol := strings.ToUpper(args[1])
	sendSimpleMessage(bot, fmt.Sprintf("🧐 Memulai Deep Research untuk %s... (Mohon tunggu sebentar)", symbol))
	
	news, err := fetchNewsRSS(symbol)
	if err != nil {
		sendSimpleMessage(bot, "❌ Gagal mengambil berita.")
		return
	}

	analysis, err := getDeepAnalysis(symbol, news)
	if err != nil {
		sendSimpleMessage(bot, "❌ Gagal melakukan analisis AI.")
		return
	}

	response := fmt.Sprintf("🔍 **Hasil Deep Research: %s**\n\n%s", symbol, analysis)
	sendMarkdownMessage(bot, response)
}

func processRecommendation(bot *tgbotapi.BotAPI) {
	pool := []string{
		"ACES", "ADRO", "AKRA", "AMRT", "ANKM", "ASII", "BBCA", "BBNI", "BBRI", "BBTN",
		"BMRI", "BRIS", "BRPT", "BUKA", "CPIN", "EMTK", "ESSA", "EXCL", "GOTO", "HRUM",
		"ICBP", "INCO", "INDY", "INKP", "INTP", "ITMG", "KLBF", "MAPI", "MBMA", "MDKA",
		"MEDC", "MIKA", "PGAS", "PGEO", "PTBA", "SIDO", "SMGR", "SRTG", "TLKM", "TPIA",
		"UNTR", "UNVR",
	}

	type Recommendation struct {
		Symbol   string
		Score    float64
		Status   string
		DistToMA float64
	}
	var results []Recommendation

	sendSimpleMessage(bot, "⏳ Siap Bos! Lagi sortir saham LQ45 buat kamu...")

	for _, s := range pool {
		score, status, distToMA := getStockScore(s)
		if score > 0 {
			results = append(results, Recommendation{s, score, status, distToMA})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].DistToMA < results[j].DistToMA
		}
		return results[i].Score > results[j].Score
	})

	var sb strings.Builder
	var topSymbols []string
	sb.WriteString("💰 **Rekomendasi Belanja Saham** 💰\n\n")

	count := 0
	for _, res := range results {
		if count >= 3 {
			break
		}
		emoji := "⭐"
		if res.Score == 10 {
			emoji = "🔥"
		}
		sb.WriteString(fmt.Sprintf("%s **%s**\n%s\n\n", emoji, res.Symbol, res.Status))
		topSymbols = append(topSymbols, res.Symbol)
		count++
	}

	if count == 0 {
		msg := tgbotapi.NewMessage(MyChatID, "Pasar lagi kurang oke, Dzik. Pantau RDPU dulu.")
		bot.Send(msg)
		return
	}

	dataBerita := "news:" + strings.Join(topSymbols, ",")
	btn := tgbotapi.NewInlineKeyboardButtonData("📰 Cek Berita Top 3", dataBerita)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(btn))

	msg := tgbotapi.NewMessage(MyChatID, sb.String())
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

func processNews(bot *tgbotapi.BotAPI, topStocks []string) {
	var sb strings.Builder
	sb.WriteString("📰 **Berita Khusus Saham Rekomendasi** 📰\n\n")

	for i, symbol := range topStocks {
		stockbitURL := fmt.Sprintf("https://stockbit.com/symbol/%s", symbol)
		googleNewsURL := fmt.Sprintf("https://www.google.com/search?q=berita+saham+%s&tbm=nws", symbol)

		sb.WriteString(fmt.Sprintf("🔥 **%d. %s**\n", i+1, symbol))
		sb.WriteString(fmt.Sprintf("   🔗 [Stockbit](%s) | [Google News](%s)\n\n", stockbitURL, googleNewsURL))
	}

	sb.WriteString("----------\n_Tips: Cek sentimen pasar dulu!_")

	msg := tgbotapi.NewMessage(MyChatID, sb.String())
	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = true
	bot.Send(msg)
}