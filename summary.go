package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// RegisterCronJobs mendaftarkan jadwal laporan otomatis ke Telegram
func RegisterCronJobs(bot *tgbotapi.BotAPI) {
	// Gunakan lokasi waktu Jakarta (WIB)
	jakartaTime, _ := time.LoadLocation("Asia/Jakarta")
	c := cron.New(cron.WithLocation(jakartaTime))

	// 1. Jadwal Laporan Makan Siang (Jam 12:00 WIB, Senin-Jumat)
	_, err := c.AddFunc("0 12 * * 1-5", func() {
		runDailySummary(bot, "Laporan Makan Siang Portofolio")
	})

	// 2. Jadwal Laporan Penutupan Pasar (Jam 16:00 WIB, Senin-Jumat)
	_, err = c.AddFunc("0 16 * * 1-5", func() {
		runDailySummary(bot, "Laporan Penutupan Pasar")
	})

	if err != nil {
		fmt.Printf("❌ Gagal mendaftarkan cron summary: %v\n", err)
		return
	}

	c.Start()
	fmt.Println("⏰ Cron Jobs untuk Summary (12:00 & 16:00) telah aktif!")
}

// runDailySummary mengolah data portofolio menjadi pesan rangkuman
func runDailySummary(bot *tgbotapi.BotAPI, title string) {
	if len(myStocks) == 0 {
		return // Tidak kirim apa-apa jika portofolio kosong
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🥗 **%s**\n", title))
	sb.WriteString(fmt.Sprintf("_%s_\n\n", time.Now().Format("Monday, 02 Jan 2006")))

	var totalPNL float64
	
	for _, plan := range myStocks {
		// Ambil harga real-time (Google) dengan fallback ke Yahoo
		price := getGooglePrice(plan.Symbol)
		if price == 0 {
			price = getLivePrice(plan.Symbol)
		}

		// Hitung Profit & Loss
		pnl := (price - plan.EntryPrice) * float64(plan.Lots) * 100
		perf := ((price - plan.EntryPrice) / plan.EntryPrice) * 100
		totalPNL += pnl

		// Tentukan Emoji
		emoji := "📈"
		if pnl < 0 {
			emoji = "📉"
		}

		sb.WriteString(fmt.Sprintf("🔹 **%s**\n", plan.Symbol))
		sb.WriteString(fmt.Sprintf("   PNL: %s (%.2f%%) %s\n\n", 
			formatRupiah(pnl), 
			perf,
			emoji,
		))
	}

	sb.WriteString("---")
	sb.WriteString(fmt.Sprintf("\n💰 **Total Profit/Loss: %s**", formatRupiah(totalPNL)))
	
	// Pesan penutup santai
	sb.WriteString("\n\n_Bot tetap siaga memantau. Lanjut kerja di Avian Brands lagi, Dzik!_")

	// Kirim pesan menggunakan MyChatID yang ada di config/main
	msg := tgbotapi.NewMessage(MyChatID, sb.String()) 
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}