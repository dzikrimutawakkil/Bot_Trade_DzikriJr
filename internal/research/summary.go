package research

import (
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	
	"learn-go/internal/market"
	"learn-go/internal/config"
	"learn-go/internal/models"
	"learn-go/internal/utils"
)

// RegisterCronJobs mendaftarkan jadwal laporan otomatis ke Telegram
func RegisterCronJobs(bot *tgbotapi.BotAPI) {
	// Gunakan lokasi waktu Jakarta (WIB)
	jakartaTime, _ := time.LoadLocation("Asia/Jakarta")
	c := cron.New(cron.WithLocation(jakartaTime))

	// HANYA 1 JADWAL: Laporan Penutupan Pasar (Jam 16:20 WIB, Senin-Jumat)
	// Catatan: err menggunakan := karena ini sekarang inisialisasi pertama
	_, err := c.AddFunc("20 16 * * 1-5", func() {
		runDailySummary(bot, "Laporan Penutupan Pasar")
	})

	if err != nil {
		fmt.Printf("❌ Gagal mendaftarkan cron summary: %v\n", err)
		return
	}

	c.Start()
	fmt.Println("⏰ Cron Jobs untuk Summary (16:20) telah aktif!")
}

// runDailySummary mengolah data portofolio menjadi pesan rangkuman
func runDailySummary(bot *tgbotapi.BotAPI, title string) {
	// 🔒 [LOCK READ] Kunci dan Salin Map agar tidak crash saat membaca portofolio
	config.DataMutex.RLock()
	stocksCopy := make(map[string]models.TradingPlan)
	for k, v := range config.MyStocks { 
		stocksCopy[k] = v 
	}
	config.DataMutex.RUnlock()

	if len(stocksCopy) == 0 {
		return // Tidak kirim apa-apa jika portofolio kosong
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🥗 **%s**\n", title))
	sb.WriteString(fmt.Sprintf("_%s_\n\n", time.Now().Format("Monday, 02 Jan 2006")))

	var totalPNL float64
	var sourceMarker string 

	// Loop menggunakan salinan Map (stocksCopy) yang sudah aman
	for _, plan := range stocksCopy {
		// Ambil harga real-time (Google) dengan fallback ke Yahoo
		price := market.GetGooglePrice(plan.Symbol)
		
		sourceMarker = "Real-Time" 
		if price == 0 {
			price = market.GetLivePrice(plan.Symbol)
			sourceMarker = "Kemungkinan Delay 15m"
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
			utils.FormatRupiah(pnl),
			perf,
			emoji,
		))
	}

	sb.WriteString("---")
	sb.WriteString(fmt.Sprintf("\n💰 **Total Profit/Loss: %s**", utils.FormatRupiah(totalPNL)))
	sb.WriteString("\n\nData diambil " + sourceMarker + ".")

	// Pesan penutup santai
	sb.WriteString("\n\n_Bot tetap siaga memantau. Lanjutkan aktivitasmu, dan semoga cuan selalu menyertai!_")

	// Kirim pesan menggunakan MyChatID yang ada di config/main
	msg := tgbotapi.NewMessage(config.MyChatID, sb.String())
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}