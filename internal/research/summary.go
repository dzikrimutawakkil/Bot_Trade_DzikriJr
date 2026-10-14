package research

import (
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/robfig/cron/v3"

	"learn-go/internal/config"
	"learn-go/internal/market"
	"learn-go/internal/models"
	"learn-go/internal/signals"
	"learn-go/internal/utils"
)

// CheckTrendBreak scans all live positions and sends alerts when MA5 < MA20.
func CheckTrendBreak(bot *tgbotapi.BotAPI) {
	config.DataMutex.RLock()
	symbols := make([]string, 0, len(config.MyStocks))
	for sym := range config.MyStocks {
		symbols = append(symbols, sym)
	}
	config.DataMutex.RUnlock()

	if len(symbols) == 0 {
		return
	}

	for _, sym := range symbols {
		bars, err := market.GetBars(sym, "1mo")
		if err != nil || len(bars) < 50 {
			continue
		}
		clean := market.CleanBars(bars)
		if len(clean) < 50 {
			continue
		}

		if signals.TrendBroken(clean, time.Now()) {
			msg := fmt.Sprintf("📉 **TREND BREAK — %s!**\n\nMA5 < MA20 terdeteksi. Pertimbangkan jual manual: `/sell %s`.", sym, sym)
			utils.SendMarkdownMessage(bot, msg)
		}
	}
}

// RegisterCronJobs adds the daily summary cron to the shared cron instance.
func RegisterCronJobs(bot *tgbotapi.BotAPI, c *cron.Cron) {
	_, err := c.AddFunc("20 16 * * 1-5", func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("⚠️ runDailySummary panic recovered: %v\n", r)
			}
		}()
		runDailySummary(bot, "Laporan Penutupan Pasar")
	})

	if err != nil {
		fmt.Printf("❌ Gagal mendaftarkan cron summary: %v\n", err)
		return
	}

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
