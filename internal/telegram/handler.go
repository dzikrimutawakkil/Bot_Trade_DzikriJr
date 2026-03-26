package telegram

import (
	"strings"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"learn-go/internal/config"
	"learn-go/internal/portfolio"
	"learn-go/internal/utils"
	"learn-go/internal/research"
)

func HandleMessages(bot *tgbotapi.BotAPI) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		// --- 1. LOGIKA UNTUK KLIK TOMBOL INLINE (Berita) ---
		if update.CallbackQuery != nil {
			if update.CallbackQuery.From.ID != config.MyChatID {
				continue
			}
			data := update.CallbackQuery.Data
			if strings.HasPrefix(data, "news:") {
				stockString := strings.TrimPrefix(data, "news:")
				listSaham := strings.Split(stockString, ",")
				research.ProcessNews(bot, listSaham)
				bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			}
			continue
		}

		// --- 2. LOGIKA UNTUK PESAN TEKS & TOMBOL KEYBOARD ---
		if update.Message == nil || update.Message.From.ID != config.MyChatID {
			continue
		}

		text := update.Message.Text
		args := strings.Fields(text)
		if len(args) == 0 {
			continue
		}

		// Cek teks UTUH terlebih dahulu (Untuk tombol keyboard)
		switch text {
		case "/status", "📊 Status":
			portfolio.ProcessStatusCommand(bot)
			continue
		case "/recommend", "❓ Recomend":
			research.ProcessRecommendation(bot)
			continue
		case "/reset":
			portfolio.ProcessResetCommand(bot)
			continue
		}

		// Jika bukan tombol, cek perintah yang memakai argumen (/buy, /sell, /research)
		command := strings.ToLower(args[0])
		switch command {
		case "/buy":
			portfolio.ProcessBuyCommand(bot, args)
		case "/sell":
			portfolio.ProcessSellCommand(bot, args)
		case "/research":
			research.ProcessResearchCommand(bot, args)
		case "/evaluate":
			portfolio.ProcessPortfolioEvaluation(bot)
		default:
			utils.SendSimpleMessage(bot, "Gunakan perintah:\n`/buy [KODE] [HARGA] [LOT]`\n`/status` | `/recommend`")
		}
	}
}