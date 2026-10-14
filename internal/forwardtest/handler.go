package forwardtest

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Handler processes the /forwardtest command.
func Handler(bot *tgbotapi.BotAPI) {
	if !IsEnabled() {
		bot.Send(tgbotapi.NewMessage(bot.Self.ID,
			"⚠️ **Forward Test belum aktif.**\n\n"+
				"Tambahkan ke `.env`:\n"+
				"`FORWARD_TEST_MODE=true`\n"+
				"`FORWARD_TEST_MODAL=1000000`\n\n"+
				"Lalu restart bot."))
		return
	}

	msg := FormatSummary()
	send := tgbotapi.NewMessage(bot.Self.ID, msg)
	send.ParseMode = "Markdown"
	bot.Send(send)
}
