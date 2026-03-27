package portfolio

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
	"learn-go/internal/config"
	"learn-go/internal/models"
	"learn-go/internal/utils"
	"learn-go/internal/research"
	"learn-go/internal/market" // Tambahkan import market untuk get harga live
)

func getPortfolioEvaluation(plan models.TradingPlan, currentPrice float64, newsContent string, technicalContent string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := genai.NewClient(ctx, option.WithAPIKey(config.GeminiAPIKey)) 
	if err != nil {
		return "", err
	}
	defer client.Close()

	// Gunakan versi 1.5-flash yang lebih stabil
	model := client.GenerativeModel("gemini-flash-latest")

	// 1. Hitung Floating PNL
	floatingPNL := utils.CalculateNetPNL(plan.EntryPrice, currentPrice, config.BuyFee, config.SellFee)

	// 2. Hitung Batas TSL Saat Ini
	tslLimit := plan.HighestPrice * (1 - config.TrailingStopPercent)

	prompt := fmt.Sprintf(`
		Bertindaklah sebagai Manajer Portofolio Saham Profesional dengan spesialisasi strategi FAST SWING TRADING (target hold 1-5 hari kerja).
		Evaluasi posisi saham %s yang sedang saya pegang saat ini dari kacamata seorang Fast Swing Trader.

		[STATUS POSISI SAYA]
		- Harga Beli (Avg): Rp %.0f
		- Harga Saat Ini: Rp %.0f (Floating: %.2f%%)
		- Rekor Harga Pucuk: Rp %.0f
		- Batas Trailing Stop (TSL): Rp %.0f (Batas aman pengunci profit / cut loss dinamis)
		- Target Take Profit Awal: Rp %.0f

		[DATA FUNDAMENTAL & BERITA]
		%s

		[DATA TEKNIKAL]
		%s

		Sebagai Fast Swing Trader, fokuslah pada momentum JANGKA SANGAT PENDEK. Evaluasi apakah saham ini masih punya "bensin" untuk lanjut naik besok, atau momentumnya sudah hilang sehingga lebih baik bungkus profit/cut loss HARI INI JUGA sebelum menyentuh batas Trailing Stop.

		WAJIB gunakan format persis seperti di bawah ini dengan Markdown:

		🛡️ **Saham:** %s (Avg: Rp %.0f | Now: Rp %.0f)
		💰 **Floating:** %.2f%%
		🚦 **Tindakan:** [AMAN (Hold) / WASPADA (Siap Jual) / KUNCI PROFIT (Jual Sekarang) / BAHAYA (Cut Loss)]
		🎯 **Potensi Lanjut Naik:** [Tinggi / Sedang / Rendah]

		📝 **Saran Strategi:**
		[Berikan 2-3 kalimat tajam ala Fast Swing Trader. Evaluasi posisi harga saat ini terhadap batas TSL (Rp %.0f) dan MA5 (Momentum Harga Mingguan). Beritahu apakah lebih baik bungkus cuan sekarang atau tahan maksimal 1-2 hari lagi.]
	`, 
		plan.Symbol, plan.EntryPrice, currentPrice, floatingPNL, plan.HighestPrice, tslLimit, plan.TakeProfit, 
		newsContent, technicalContent, 
		plan.Symbol, plan.EntryPrice, currentPrice, floatingPNL, tslLimit)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", err
	}

	if len(resp.Candidates) > 0 {
		var sb strings.Builder
		for _, part := range resp.Candidates[0].Content.Parts {
			sb.WriteString(fmt.Sprintf("%v", part))
		}
		return sb.String(), nil
	}

	return "AI tidak memberikan respon evaluasi.", nil
}

func ProcessPortfolioEvaluation(bot *tgbotapi.BotAPI) {
	if len(config.MyStocks) == 0 {
		log.Println("Portofolio kosong, skip evaluasi harian.")
		return
	}

	utils.SendSimpleMessage(bot, "🔄 _Menyiapkan Rapat Pagi: Evaluasi Portofolio..._")

	var finalReport strings.Builder
	finalReport.WriteString("📋 **MORNING BRIEFING: EVALUASI PORTOFOLIO** 📋\n\n")

	for symbol, plan := range config.MyStocks {
		// Ambil data harga terbaru pagi ini
		currentPrice := market.GetLivePrice(symbol)
		if currentPrice == 0 {
			currentPrice = plan.EntryPrice // Fallback jika API gagal
		}

		newsContent, err := research.FetchNewsRSS(symbol)
		if err != nil {
			newsContent = "Tidak ada berita terbaru."
		}

		// Ambil technical string
		technicalContent := research.FetchTechnicalData(symbol)

		// Evaluasi via Gemini dengan tambahan parameter currentPrice
		eval, err := getPortfolioEvaluation(plan, currentPrice, newsContent, technicalContent)
		if err != nil {
			log.Println("❌ Gagal mendapatkan evaluasi AI untuk", symbol, ":", err)
			finalReport.WriteString(fmt.Sprintf("⚠️ **%s**: Gagal mendapatkan AI.\n\n", symbol))
			continue
		}

		finalReport.WriteString(eval)
		finalReport.WriteString("\n━━━━━━━━━━━━━━━━━━━━\n\n")
		
		// Jeda agar tidak terkena limit API Gemini
		time.Sleep(3 * time.Second)
	}

	utils.SendMarkdownMessage(bot, finalReport.String())
}