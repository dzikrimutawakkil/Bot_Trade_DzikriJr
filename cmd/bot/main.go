// cmd/bot/main.go
package main

import (
	"log"
	"time"

	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"learn-go/internal/config"
	"learn-go/internal/portfolio"
	"learn-go/internal/storage"
	"learn-go/internal/telegram"
	"learn-go/internal/research"
)

func main() {
	// Load environment variables dari .env file SEBELUM inisialisasi lain
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: Tidak dapat load .env file (file mungkin tidak ada): %v", err)
	} else {
		log.Println("Environment variables berhasil dimuat dari .env")
	}

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
		portfolio.ProcessPortfolioEvaluation(bot) // Rapat Pagi
	})

	c.AddFunc("30 12 * * 1-5", func() {
		portfolio.ProcessPortfolioEvaluation(bot) // Noon Briefing
	})

	// 🔥 THE GOLDEN HOUR: Tiap Senin-Jumat jam 15:40 WIB
	c.AddFunc("40 15 * * 1-5", func() {
		log.Println("Memeriksa sisa peluru sebelum mengeksekusi Hunter Algorithm...")

		// Hitung total modal yang sedang terpakai di pasar
		var modalTerpakai float64
		for _, plan := range config.MyStocks {
			modalTerpakai += plan.EntryPrice * float64(plan.Lots) * 100 * (1 + config.BuyFee)
		}

		sisaModal := config.TotalModalTrading - modalTerpakai

		// Syarat: Jalan kalau portofolio KOSONG atau SISA MODAL masih di atas Rp 250.000 
		// (Ubah angka 250000 ini sesuai batas minimal kamu buat beli 1 saham)
		if len(config.MyStocks) == 0 || sisaModal > 250000 {
			research.ProcessRecommendation(bot)
		} else {
			log.Printf("Skip Hunter Algorithm. Peluru tersisa (Rp %.0f) terlalu tipis.\n", sisaModal)
		}
	})

	c.Start()
	
	// Panggil Cron Job untuk Summary dari package research
	research.RegisterCronJobs(bot)

	// Jalankan Monitor & Handler
	go portfolio.StartPriceMonitor(bot)
	telegram.HandleMessages(bot)
}