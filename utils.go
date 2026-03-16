package main

import (
	"fmt"
	"strings"
	"time"
	"log"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// formatRupiah mengubah float64 ke format Rp. 1.234
func formatRupiah(amount float64) string {
	s := fmt.Sprintf("%.0f", amount)
	if len(s) <= 3 {
		return "Rp. " + s
	}

	var result []string
	for len(s) > 3 {
		result = append([]string{s[len(s)-3:]}, result...)
		s = s[:len(s)-3]
	}
	result = append([]string{s}, result...)

	return "Rp. " + strings.Join(result, ".")
}

// isMarketOpen mengecek jam buka bursa (WIB)
func isMarketOpen() bool {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	now := time.Now().In(loc)

	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		return false
	}
	if now.Hour() < 9 || now.Hour() >= 16 {
		return false
	}
	return true
}

var mainKeyboard = tgbotapi.NewReplyKeyboard(
    tgbotapi.NewKeyboardButtonRow(
        tgbotapi.NewKeyboardButton("📊 Status"),
        tgbotapi.NewKeyboardButton("❓ Recomend"),
    ),
)

func sendSimpleMessage(bot *tgbotapi.BotAPI, text string) {
	msg := tgbotapi.NewMessage(MyChatID, text)
	msg.ReplyMarkup = mainKeyboard
	bot.Send(msg)
}

func sendMarkdownMessage(bot *tgbotapi.BotAPI, text string) {
    msg := tgbotapi.NewMessage(MyChatID, text)
    msg.ParseMode = "Markdown"

    msg.ReplyMarkup = mainKeyboard 
    _, err := bot.Send(msg)
    
    if err != nil {
        log.Printf("⚠️ Gagal kirim Markdown: %v. Mengirim teks biasa...", err)
        
        msg.ParseMode = "" 
        bot.Send(msg)
    }
}