// cmd/bot/main.go
package main

import (
	"log"
	"time"

	"github.com/robfig/cron/v3"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"learn-go/internal/config"
	"learn-go/internal/portfolio"
	"learn-go/internal/storage"
	"learn-go/internal/telegram"
)

func main() {
	storage.LoadData()  // Load portfolio awal

	bot, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Bot %s siap melayani Swing Trade!", bot.Self.UserName)

	// Inisialisasi Cron
	jakartaTime, _ := time.LoadLocation("Asia/Jakarta")
	c := cron.New(cron.WithLocation(jakartaTime))

	c.AddFunc("45 8 * * 1-5", func() {
		portfolio.ProcessPortfolioEvaluation(bot)
	})

	c.AddFunc("30 16 * * 5", func() {
		telegram.ProcessRecommendation(bot) // Laporan Jumat sore
	})

	c.Start()

	// Jalankan Monitor & Handler
	go portfolio.StartPriceMonitor(bot)
	telegram.HandleMessages(bot)
}