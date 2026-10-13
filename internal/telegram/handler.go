package telegram

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"learn-go/internal/config"
	"learn-go/internal/portfolio"
	"learn-go/internal/research"
	"learn-go/internal/shadow"
	"learn-go/internal/storage"
	"learn-go/internal/utils"
)

func HandleMessages(bot *tgbotapi.BotAPI) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
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
		case "/start", "/menu":
			msg := tgbotapi.NewMessage(config.MyChatID, "🤖 DzikriJr siap membantu, Bos! Silakan pilih menu di bawah ini 👇")

			keyboard := tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("📊 Status"),
					tgbotapi.NewKeyboardButton("❓ Recomend"),
				),
			)
			keyboard.ResizeKeyboard = true

			msg.ReplyMarkup = keyboard
			bot.Send(msg)
			continue

		case "/status", "📊 Status":
			portfolio.ProcessStatusCommand(bot)
			continue
		case "/recommend", "❓ Recomend":
			go research.ProcessRecommendation(bot)
			continue
		}

		// Perintah dengan argumen
		command := strings.ToLower(args[0])
		switch command {
		case "/buy":
			portfolio.ProcessBuyCommand(bot, args)
		case "/sell":
			portfolio.ProcessSellCommand(bot, args)
		case "/research":
			go research.ProcessResearchCommand(bot, args)
		case "/antre":
			portfolio.ProcessAntreCommand(bot, args)
		case "/cancel_antre":
			portfolio.ProcessCancelAntreCommand(bot, args)
		case "/evaluate":
			go portfolio.ProcessPortfolioEvaluation(bot)
		case "/stats":
			ProcessStatsCommand(bot)
		case "/reset":
			if len(args) >= 2 && strings.ToLower(args[1]) == "confirm" {
				portfolio.ProcessResetCommand(bot, true)
			} else {
				portfolio.ProcessResetCommand(bot, false)
			}
		case "/shadow":
			ProcessShadowCommand(bot)
		default:
			utils.SendSimpleMessage(bot, "Gunakan perintah:\n`/buy [KODE] [HARGA] [LOT]`\n`/status` | `/recommend`")
		}
	}
}

// ProcessStatsCommand menampilkan statistik trading ke Telegram
func ProcessStatsCommand(bot *tgbotapi.BotAPI) {
	stats, err := storage.CalculateStatistics()
	if err != nil {
		utils.SendSimpleMessage(bot, "❌ Gagal menghitung statistik: "+err.Error())
		return
	}

	if stats.TotalTrades == 0 {
		utils.SendSimpleMessage(bot, "📊Belum ada data trading untuk dianalisa.\nMulai trading dulu, Bos!")
		return
	}

	message := storage.FormatStatsMessage(stats)
	utils.SendMarkdownMessage(bot, message)
}

// ProcessShadowCommand menampilkan ringkasan shadow log sinyal.
func ProcessShadowCommand(bot *tgbotapi.BotAPI) {
	rows, err := shadow.LoadRows()
	if err != nil {
		utils.SendSimpleMessage(bot, "❌ Gagal memuat data shadow log: "+err.Error())
		return
	}
	if rows == nil {
		utils.SendSimpleMessage(bot, "📊 Shadow log masih kosong. Coba lagi setelah market tutup.")
		return
	}

	sum := shadow.Summarize(rows)
	msg := sum.Format()
	utils.SendMarkdownMessage(bot, msg)
}
