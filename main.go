package main

import (
	"log"
	"github.com/robfig/cron/v3"
	"time"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	bot, err := tgbotapi.NewBotAPI(BotToken)
	if err != nil {
		log.Panic(err)
	}

	configs := tgbotapi.NewSetMyCommands(
		tgbotapi.BotCommand{Command: "reset", Description: "Hapus SEMUA pantauan (Mulai dari 0)"},
		tgbotapi.BotCommand{Command: "buy", Description: "Format: /buy [KODE] [HARGA] [LOTS]"},
		tgbotapi.BotCommand{Command: "sell", Description: "Format: /sell [KODE]"},
		tgbotapi.BotCommand{Command: "research", Description: "Format: /research [KODE]"},
		tgbotapi.BotCommand{Command: "evaluate", Description: "Evaluasi Portofolio Saat Ini"},
	)
	bot.Request(configs)

	loadData()

	log.Printf("Bot %s siap!", bot.Self.UserName)


	// 1. Inisialisasi Cron
	jakartaTime, _ := time.LoadLocation("Asia/Jakarta")
	c := cron.New(cron.WithLocation(jakartaTime))

	// Cron job analisis otomatis setiap jam 08:45 WIB (Senin-Jumat)
	// "45 8 * * 1-5" artinya: Jam 08:45, Senin sampai Jumat
	_, err = c.AddFunc("45 8 * * 1-5", func() {
		log.Println("Menjalankan Automatic Daily Scanner...")
		processPortfolioEvaluation(bot)
	})

	// Atur Jadwal Rekomendasi Saham Baru: Jam 16:30, KHUSUS hari Jumat
	// (30 Menit setelah penutupan pasar di akhir pekan)
	_, err = c.AddFunc("30 16 * * 5", func() {
		log.Println("Menjalankan Weekly Stock Recommendation...")
		processRecommendation(bot)
	})

	if err != nil {
		log.Fatal("Gagal setting jadwal:", err)
	}

	// 3. Mulai Cron
	c.Start()

	RegisterCronJobs(bot)

	// Hanya memanggil fungsi dari file lain
	go startPriceMonitor(bot)
	handleMessages(bot)
}