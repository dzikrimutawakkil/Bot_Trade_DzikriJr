package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"learn-go/internal/config"
)

var marketHolidays []string

// FormatRupiah mengubah float64 ke format Rp. 1.234
func FormatRupiah(amount float64) string {
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

// LoadHolidays memuat daftar hari libur dari file JSON opsional.
func LoadHolidays(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("⚠️ Gagal memuat holidays.json: %v", err)
		}
		return
	}

	if err := json.Unmarshal(data, &marketHolidays); err != nil {
		log.Printf("⚠️ Format holidays.json tidak valid: %v", err)
		return
	}

	log.Printf("✅ Memuat %d hari libur bursa.", len(marketHolidays))
}

// IsMarketOpen mengecek jam buka bursa (WIB)
func IsMarketOpen() bool {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	now := time.Now().In(loc)

	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		return false
	}
	if now.Hour() < 9 || now.Hour() >= 16 {
		return false
	}
	return IsTradingDay(now)
}

// IsTradingDay mengembalikan true jika t bukan Sabtu/Minggu dan bukan hari libur.
func IsTradingDay(t time.Time) bool {
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return false
	}
	dateStr := t.Format("2006-01-02")
	for _, h := range marketHolidays {
		if h == dateStr {
			return false
		}
	}
	return true
}

var mainKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("📊 Status"),
		tgbotapi.NewKeyboardButton("❓ Recomend"),
	),
)

func SendSimpleMessage(bot *tgbotapi.BotAPI, text string) {
	msg := tgbotapi.NewMessage(config.MyChatID, text)
	msg.ReplyMarkup = mainKeyboard
	bot.Send(msg)
}

func SendMarkdownMessage(bot *tgbotapi.BotAPI, text string) {
	msg := tgbotapi.NewMessage(config.MyChatID, text)
	msg.ParseMode = "Markdown"

	msg.ReplyMarkup = mainKeyboard
	_, err := bot.Send(msg)

	if err != nil {
		log.Printf("⚠️ Gagal kirim Markdown: %v. Mengirim teks biasa...", err)

		msg.ParseMode = ""
		bot.Send(msg)
	}
}
