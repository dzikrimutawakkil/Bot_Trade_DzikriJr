package utils

import (
	"fmt"
	"strings"
	"time"
	"log"
	"math"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"learn-go/internal/config"
)

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

// CalculateNetPNL menghitung persentase profit bersih setelah dipotong fee beli dan jual
func CalculateNetPNL(entryPrice float64, currentPrice float64, buyFee float64, sellFee float64) float64 {
	// Modal asli = Harga beli + fee beli
	totalBuyCapital := entryPrice * (1 + buyFee)
	
	// Uang diterima = Harga jual - fee jual
	netSellValue := currentPrice * (1 - sellFee)
	
	// Rumus untung bersih
	return ((netSellValue - totalBuyCapital) / totalBuyCapital) * 100
}

func RoundToFraction(price float64) float64 {
	var fraction float64

	switch {
	case price < 50:
		// Saham gocap atau di bawahnya (aturan papan pemantauan khusus)
		fraction = 1 
	case price < 200:
		fraction = 1
	case price < 500:
		fraction = 2
	case price < 2000:
		fraction = 5
	case price < 5000:
		fraction = 10
	default:
		// Harga di atas 5000
		fraction = 25
	}

	// Rumus pembulatan ke kelipatan terdekat
	return math.Round(price/fraction) * fraction
}