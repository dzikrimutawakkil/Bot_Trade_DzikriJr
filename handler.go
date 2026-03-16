package main

import (
	"strings"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func handleMessages(bot *tgbotapi.BotAPI) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		// --- 1. LOGIKA UNTUK KLIK TOMBOL INLINE (Berita) ---
		if update.CallbackQuery != nil {
			if update.CallbackQuery.From.ID != MyChatID {
				continue
			}
			data := update.CallbackQuery.Data
			if strings.HasPrefix(data, "news:") {
				stockString := strings.TrimPrefix(data, "news:")
				listSaham := strings.Split(stockString, ",")
				processNews(bot, listSaham)
				bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			}
			continue
		}

		// --- 2. LOGIKA UNTUK PESAN TEKS & TOMBOL KEYBOARD ---
		if update.Message == nil || update.Message.From.ID != MyChatID {
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
			processStatusCommand(bot)
			continue
		case "/recommend", "❓ Recomend":
			processRecommendation(bot)
			continue
		case "/reset":
			processResetCommand(bot)
			continue
		}

		// Jika bukan tombol, cek perintah yang memakai argumen (/buy, /sell, /research)
		command := strings.ToLower(args[0])
		switch command {
		case "/buy":
			processBuyCommand(bot, args)
		case "/sell":
			processSellCommand(bot, args)
		case "/research":
			processResearchCommand(bot, args)
		default:
			sendSimpleMessage(bot, "Gunakan perintah:\n`/buy [KODE] [HARGA] [LOT]`\n`/status` | `/recommend`")
		}
	}
}