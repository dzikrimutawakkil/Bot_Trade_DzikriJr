package main

import (
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/robfig/cron/v3"

	"learn-go/internal/config"
	"learn-go/internal/portfolio"
	"learn-go/internal/research"
	"learn-go/internal/rules"
	"learn-go/internal/shadow"
	"learn-go/internal/storage"
	"learn-go/internal/telegram"
	"learn-go/internal/utils"
)

func main() {
	if err := config.LoadEnv(); err != nil {
		log.Fatalf("Gagal memuat environment: %v", err)
	}
	log.Println("Environment variables berhasil dimuat.")

	storage.LoadData()

	if err := storage.MigrateTradeHistory("trade_history.csv"); err != nil {
		log.Printf("Migrasi CSV warning: %v", err)
	}

	bot, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Bot %s siap melayani Swing Trade!", bot.Self.UserName)

	jakartaTime, _ := time.LoadLocation("Asia/Jakarta")
	c := cron.New(cron.WithLocation(jakartaTime))

	if _, err := c.AddFunc("45 8 * * 1-5", func() {
		if !utils.IsTradingDay(time.Now().In(jakartaTime)) {
			return
		}
		portfolio.ProcessPortfolioEvaluation(bot)
	}); err != nil {
		log.Fatalf("Gagal menambahkan cron 08:45: %v", err)
	}

	if _, err := c.AddFunc("30 12 * * 1-5", func() {
		if !utils.IsTradingDay(time.Now().In(jakartaTime)) {
			return
		}
		portfolio.ProcessPortfolioEvaluation(bot)
	}); err != nil {
		log.Fatalf("Gagal menambahkan cron 12:30: %v", err)
	}

	if _, err := c.AddFunc("40 15 * * 1-5", func() {
		if !utils.IsTradingDay(time.Now().In(jakartaTime)) {
			return
		}
		log.Println("Memeriksa sisa peluru sebelum mengeksekusi Hunter Algorithm...")

		sisaModal := config.RemainingCapital()

		if len(config.MyStocks) == 0 || sisaModal > rules.Default.MinPurchaseAmount {
			research.ProcessRecommendation(bot)
		} else {
			log.Printf("Skip Hunter Algorithm. Peluru tersisa (Rp %.0f) terlalu tipis.\n", sisaModal)
		}
	}); err != nil {
		log.Fatalf("Gagal menambahkan cron 15:40: %v", err)
	}

	// Shadow Log: catat sinyal harian (16:30) dan isi hasil (16:50)
	if _, err := c.AddFunc("30 16 * * 1-5", func() {
		if !utils.IsTradingDay(time.Now().In(jakartaTime)) {
			return
		}
		go shadow.RecordSignals()
	}); err != nil {
		log.Fatalf("Gagal menambahkan cron 16:30 shadow: %v", err)
	}
	if _, err := c.AddFunc("50 16 * * 1-5", func() {
		if !utils.IsTradingDay(time.Now().In(jakartaTime)) {
			return
		}
		go shadow.FillOutcomes()
	}); err != nil {
		log.Fatalf("Gagal menambahkan cron 16:50 shadow: %v", err)
	}

	c.Start()

	research.RegisterCronJobs(bot)

	go portfolio.StartPriceMonitor(bot)
	telegram.HandleMessages(bot)
}
