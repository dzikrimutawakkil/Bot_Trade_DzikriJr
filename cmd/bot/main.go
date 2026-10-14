package main

import (
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/robfig/cron/v3"

	"learn-go/internal/config"
	"learn-go/internal/forwardtest"
	"learn-go/internal/portfolio"
	"learn-go/internal/research"
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
	forwardtest.Init()

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

	safePortfolioEval := func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("⚠️ ProcessPortfolioEvaluation panic recovered: %v", r)
			}
		}()
		portfolio.ProcessPortfolioEvaluation(bot)
	}

	if _, err := c.AddFunc("45 8 * * 1-5", func() {
		if !utils.IsTradingDay(time.Now().In(jakartaTime)) {
			return
		}
		safePortfolioEval()
	}); err != nil {
		log.Fatalf("Gagal menambahkan cron 08:45: %v", err)
	}

	if _, err := c.AddFunc("30 12 * * 1-5", func() {
		if !utils.IsTradingDay(time.Now().In(jakartaTime)) {
			return
		}
		safePortfolioEval()
	}); err != nil {
		log.Fatalf("Gagal menambahkan cron 12:30: %v", err)
	}

	if _, err := c.AddFunc("40 15 * * 1-5", func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("⚠️ ProcessS08Recommendation panic recovered: %v", r)
			}
		}()
		if !utils.IsTradingDay(time.Now().In(jakartaTime)) {
			return
		}
		research.ProcessS08Recommendation(bot)
	}); err != nil {
		log.Fatalf("Gagal menambahkan cron 15:40 S08: %v", err)
	}

	// Trend Break check every Friday at 16:20
	if _, err := c.AddFunc("20 16 * * 5", func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("⚠️ CheckTrendBreak panic recovered: %v", r)
			}
		}()
		if !utils.IsTradingDay(time.Now().In(jakartaTime)) {
			return
		}
		go research.CheckTrendBreak(bot)
	}); err != nil {
		log.Fatalf("Gagal menambahkan cron Trend Break 16:20 Jumat: %v", err)
	}

	// Shadow Log: catat sinyal harian (16:30) dan isi hasil (16:50)
	if _, err := c.AddFunc("30 16 * * 1-5", func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("⚠️ RecordSignals panic recovered: %v", r)
			}
		}()
		if !utils.IsTradingDay(time.Now().In(jakartaTime)) {
			return
		}
		go shadow.RecordSignals()
	}); err != nil {
		log.Fatalf("Gagal menambahkan cron 16:30 shadow: %v", err)
	}
	if _, err := c.AddFunc("50 16 * * 1-5", func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("⚠️ FillOutcomes panic recovered: %v", r)
			}
		}()
		if !utils.IsTradingDay(time.Now().In(jakartaTime)) {
			return
		}
		go shadow.FillOutcomes()
	}); err != nil {
		log.Fatalf("Gagal menambahkan cron 16:50 shadow: %v", err)
	}

	// Forward Test: evaluate exits daily (16:20), entry every Friday (18:05)
	if _, err := c.AddFunc("20 16 * * 1-5", func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("⚠️ EvaluateDaily panic recovered: %v", r)
			}
		}()
		if !utils.IsTradingDay(time.Now().In(jakartaTime)) {
			return
		}
		go forwardtest.EvaluateDaily()
	}); err != nil {
		log.Fatalf("Gagal menambahkan cron 16:20 forwardtest: %v", err)
	}
	if _, err := c.AddFunc("5 18 * * 5", func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("⚠️ ProcessWeeklyRebalance panic recovered: %v", r)
			}
		}()
		if !utils.IsTradingDay(time.Now().In(jakartaTime)) {
			return
		}
		go portfolio.ProcessWeeklyRebalance(bot)
		go forwardtest.ProcessWeeklyEntry()
	}); err != nil {
		log.Fatalf("Gagal menambahkan cron 18:05 weekly: %v", err)
	}

	c.Start()
	defer func() {
		if r := recover(); r != nil {
			log.Printf("⚠️ Cron scheduler recovered from panic: %v", r)
		}
	}()

	research.RegisterCronJobs(bot, c)

	// StartPriceMonitor + HandleMessages — each goroutine has its own panic recovery
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("⚠️ Price monitor recovered from panic: %v", r)
			}
		}()
		portfolio.StartPriceMonitor(bot)
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("⚠️ Message handler recovered from panic: %v", r)
			}
		}()
		telegram.HandleMessages(bot)
	}()

	// Block forever — let goroutines run
	select {}
}
