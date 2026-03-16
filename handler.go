package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func handleMessages(bot *tgbotapi.BotAPI) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		// --- 1. LOGIKA UNTUK KLIK TOMBOL (CallbackQuery) ---
		if update.CallbackQuery != nil {
			// Keamanan: Hanya MyChatID yang bisa eksekusi
			if update.CallbackQuery.From.ID != MyChatID {
				continue
			}

			data := update.CallbackQuery.Data
			if strings.HasPrefix(data, "news:") {
				stockString := strings.TrimPrefix(data, "news:")
				listSaham := strings.Split(stockString, ",")

				// Panggil tanpa oper chatID
				processNews(bot, listSaham)

				bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			}
			continue
		}

		// --- 2. LOGIKA UNTUK PESAN TEKS ---
		if update.Message == nil || update.Message.From.ID != MyChatID {
			continue
		}

		text := update.Message.Text
		args := strings.Fields(text)

		if len(args) == 4 && args[0] == "/buy" {
			processBuyCommand(bot, args)
			continue
		}

		if len(args) == 2 && args[0] == "/sell" {
			symbol := strings.ToUpper(args[1])
			if _, ada := myStocks[symbol]; ada {
				delete(myStocks, symbol)
				saveData()
				sendSimpleMessage(bot, fmt.Sprintf("✅ Pantauan %s dihentikan.", symbol))
			} else {
				sendSimpleMessage(bot, "❌ Saham tidak ditemukan.")
			}
			continue
		}

		if len(args) == 2 && args[0] == "/research" {
			symbol := strings.ToUpper(args[1])
			sendSimpleMessage(bot, fmt.Sprintf("🧐 Memulai Deep Research untuk %s... (Mohon tunggu sebentar)", symbol))
			
			news, err := fetchNewsRSS(symbol)
			if err != nil {
				sendSimpleMessage(bot, "❌ Gagal mengambil berita.")
				continue
			}

			analysis, err := getDeepAnalysis(symbol, news)
			if err != nil {
				sendSimpleMessage(bot, "❌ Gagal melakukan analisis AI.")
				continue
			}

			// 3. Kirim Hasil
			response := fmt.Sprintf("🔍 **Hasil Deep Research: %s**\n\n%s", symbol, analysis)
			sendMarkdownMessage(bot, response)
			continue
		}

		if text == "/status" || text == "📊 Status" {
			processStatusCommand(bot)
			continue
		}

		if text == "/recommend" || text == "❓ Recomend" {
			processRecommendation(bot)
			continue
		}

		if text == "/reset" {
			myStocks = make(map[string]TradingPlan)
			saveData()
			sendSimpleMessage(bot, "🧹 Semua pantauan dihapus!")
			continue
		}

		sendSimpleMessage(bot, "Gunakan perintah:\n`/buy [KODE] [HARGA] [LOT]`\n`/status` | `/recommend`")
	}
}


func processBuyCommand(bot *tgbotapi.BotAPI, args []string) {
	if len(args) < 4 {
		sendSimpleMessage(bot, "❌ Format salah! Gunakan: `/buy [KODE] [HARGA] [LOT]`")
		return
	}

	symbol := strings.ToUpper(args[1])
	entry, _ := strconv.ParseFloat(args[2], 64)
	lots, _ := strconv.Atoi(args[3]) 

	plan := TradingPlan{
		Symbol:     symbol,
		EntryPrice: entry,
		TakeProfit: entry * (1 + TPPercent),
		CutLoss:    entry * (1 - CLPercent),
		Lots:       lots,
	}
	myStocks[symbol] = plan
	saveData()

	totalModal := entry * float64(lots) * 100
	response := fmt.Sprintf("✅ **%s Terpasang!**\nLot: %d\nModal: %s\nTP: %s | CL: %s", 
		symbol, lots, formatRupiah(totalModal), formatRupiah(plan.TakeProfit), formatRupiah(plan.CutLoss))
	sendMarkdownMessage(bot, response)
}

func processStatusCommand(bot *tgbotapi.BotAPI) {
	if len(myStocks) == 0 {
		sendSimpleMessage(bot, "Belum ada saham yang dipantau.")
		return
	}

	var sb strings.Builder
	sb.WriteString("📋 **Status Portofolio (Dual-Check):**\n\n")

	for _, plan := range myStocks {
		// 1. Ambil data dari kedua sumber
		yahooPrice := getLivePrice(plan.Symbol)
		googlePrice := getGooglePrice(plan.Symbol)

		// Fallback: Jika Google gagal scrape, samakan dengan Yahoo agar tidak 0
		sourceMarker := "[G]"
		if googlePrice == 0 {
			googlePrice = yahooPrice
			sourceMarker = "[Y-fallback]"
		}

		// 2. Hitung performa (Kita pakai Google sebagai acuan PNL karena lebih real-time)
		totalPNL := (googlePrice - plan.EntryPrice) * float64(plan.Lots) * 100
		perfYahoo := ((yahooPrice - plan.EntryPrice) / plan.EntryPrice) * 100
		perfGoogle := ((googlePrice - plan.EntryPrice) / plan.EntryPrice) * 100

		trendEmoji := "📈"
		if googlePrice < plan.EntryPrice {
			trendEmoji = "📉"
		}

		// 3. Susun Pesan
		sb.WriteString(fmt.Sprintf("🔹 **%s** (%d Lot)\n", plan.Symbol, plan.Lots))
		sb.WriteString(fmt.Sprintf("   Entry : %s\n", formatRupiah(plan.EntryPrice)))
		
		// Baris Yahoo (Radar Jauh)
		sb.WriteString(fmt.Sprintf("   [Y] Now : %s (%.2f%%)\n", 
			formatRupiah(yahooPrice), perfYahoo))
		
		// Baris Google (Sniper/Real-time)
		sb.WriteString(fmt.Sprintf("   %s Now : %s (%.2f%%)\n", 
   			 sourceMarker, formatRupiah(googlePrice), perfGoogle))

		// Total Profit/Loss (Berdasarkan harga Google)
		pnlLabel := "Cuan"
		if totalPNL < 0 {
			pnlLabel = "Rugi"
		}
		sb.WriteString(fmt.Sprintf("   👉 **%s: %s %s**\n\n", 
			pnlLabel, formatRupiah(totalPNL), trendEmoji))
	}

	sb.WriteString("_ket: [Y]=Yahoo (Delay 15m), [G]=Google (Real-time)_")

	sendMarkdownMessage(bot, sb.String())
}


// --- FUNGSI REKOMENDASI (Tanpa Parameter ChatID) ---
func processRecommendation(bot *tgbotapi.BotAPI) {
	pool := []string{
		"ACES", "ADRO", "AKRA", "AMRT", "ANKM", "ASII", "BBCA", "BBNI", "BBRI", "BBTN",
		"BMRI", "BRIS", "BRPT", "BUKA", "CPIN", "EMTK", "ESSA", "EXCL", "GOTO", "HRUM",
		"ICBP", "INCO", "INDY", "INKP", "INTP", "ITMG", "KLBF", "MAPI", "MBMA", "MDKA",
		"MEDC", "MIKA", "PGAS", "PGEO", "PTBA", "SIDO", "SMGR", "SRTG", "TLKM", "TPIA",
		"UNTR", "UNVR",
	}

	type Recommendation struct {
		Symbol string
		Score  float64
		Status string
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

	// --- LOGIKA SORTIR AGRESIF ---
    sort.Slice(results, func(i, j int) bool {
        // Jika skor sama (misal sama-sama 10)
        if results[i].Score == results[j].Score {
            // Pilih yang jarak ke MA20-nya paling kecil (paling mepet garis)
            // Kita pakai absolute (math.Abs) supaya jarak di bawah MA20 pun terhitung dekat
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