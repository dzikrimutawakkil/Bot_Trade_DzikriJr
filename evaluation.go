package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

func getPortfolioEvaluation(plan TradingPlan, newsContent string, technicalContent string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Asumsi kamu sudah punya inisialisasi client Gemini seperti di handler_research.go
	client, err := genai.NewClient(ctx, option.WithAPIKey(GeminiAPIKey)) 
	if err != nil {
		return "", err
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-flash-latest")

	prompt := fmt.Sprintf(`
		Bertindaklah sebagai Pengawas Portofolio Saham Profesional.
		Saya saat ini sedang memegang saham %s dengan Harga Beli (Average) di Rp %.0f.
		Target Take Profit (TP) saya di Rp %.0f dan batas Cut Loss (CL) di Rp %.0f.

		Berdasarkan data hari ini:
		[DATA FUNDAMENTAL & BERITA TERBARU]
		%s

		[DATA TEKNIKAL & HARGA TERAKHIR]
		%s

		Evaluasi apakah saham ini masih memiliki momentum untuk mencapai target TP, atau justru menunjukkan sinyal pelemahan yang berisiko menyentuh CL.

		WAJIB gunakan format persis seperti di bawah ini dengan Markdown:

		🛡️ **Saham:** %s (Avg: Rp %.0f)
		🚦 **Status:** [AMAN (Hold) / WASPADA (Siap Jual) / BAHAYA (Cut Loss)]
		🎯 **Peluang ke TP:** [Tinggi / Sedang / Rendah]
		📉 **Risiko ke CL:** [Tinggi / Sedang / Rendah]

		📝 **Analisis Kondisi:**
		[Tulis 2-3 kalimat analisis mengapa statusnya demikian. Apakah ada berita buruk baru? Atau teknikal patah tren? Langsung to the point.]
	`, plan.Symbol, plan.EntryPrice, plan.TakeProfit, plan.CutLoss, newsContent, technicalContent, plan.Symbol, plan.EntryPrice)

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

func processPortfolioEvaluation(bot *tgbotapi.BotAPI) {
	if len(myStocks) == 0 {
		log.Println("Portofolio kosong, skip evaluasi harian.")
		return
	}

	sendSimpleMessage(bot, "🔄 _Sedang menganalisa kondisi portofolio hari ini..._")

	var finalReport strings.Builder
	finalReport.WriteString("📊 **LAPORAN EARLY WARNING SYSTEM PORTOFOLIO** 📊\n\n")

	for symbol, plan := range myStocks {
		// Asumsi fetchNewsRSS dan fetchTechnicalData sudah ada di file lain (scrapper.go/analyst.go)
		newsContent, err := fetchNewsRSS(symbol)
		if err != nil {
			newsContent = "Tidak ada berita terbaru."
		}

		technicalContent := fetchTechnicalData(symbol)

		// Evaluasi via Gemini
		eval, err := getPortfolioEvaluation(plan, newsContent, technicalContent)
		if err != nil {
			log.Println("❌ Gagal mendapatkan evaluasi AI untuk", symbol, ":", err)
			finalReport.WriteString(fmt.Sprintf("⚠️ **%s**: Gagal mendapatkan evaluasi AI.\n\n", symbol))
			continue
		}

		finalReport.WriteString(eval)
		finalReport.WriteString("\n━━━━━━━━━━━━━━━━━━━━\n\n")
		
		// Jeda agar tidak terkena limit API Gemini
		time.Sleep(2 * time.Second)
	}

	sendMarkdownMessage(bot, finalReport.String())
}