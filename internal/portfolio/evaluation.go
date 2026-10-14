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
	"learn-go/internal/market"
	"learn-go/internal/models"
	"learn-go/internal/research"
	"learn-go/internal/rules"
	"learn-go/internal/storage"
	"learn-go/internal/utils"
)

func getPortfolioEvaluation(plan models.TradingPlan, currentPrice float64, newsContent string, technicalContent string, chartIcon string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := genai.NewClient(ctx, option.WithAPIKey(config.GeminiAPIKey))
	if err != nil {
		return "", err
	}
	defer client.Close()

	model := client.GenerativeModel(config.GeminiModel)
	model.Temperature = genai.Ptr(float32(0.0))
	log.Printf("[AI] Evaluasi portofolio %s (model=%s)", plan.Symbol, config.GeminiModel)

	floatingPNL := rules.Default.NetPNL(plan.EntryPrice, currentPrice)

	tslDisplay := rules.Default.EffectiveStop(plan)
	tslActive := rules.Default.TrailingActive(plan)
	tslLabel := fmt.Sprintf("Rp %.0f", tslDisplay)
	if !tslActive {
		tslLabel = "belum aktif"
	}

	strategyContext := "1. **JANGAN PANIK KARENA MA5!** Saham ini sengaja dibeli saat harganya merah/turun ke area Support (MA20).\n2. **FOKUS EVALUASI:** Apakah Support MA20 atau batas Cut Loss masih kuat menahan harga? Jika iya, suruh HOLD (Aman) menunggu pantulan.\n3. **TARGET FAST SWING:** Target profit adalah +4%% hingga +7%% dalam 1-5 hari bursa. Berikan saran terkait pencapaian target ini berdasarkan waktu beli."

	prompt := fmt.Sprintf(`
        Bertindaklah sebagai Manajer Portofolio Saham Profesional khusus FAST SWING TRADING.
        Evaluasi posisi saham %s.

        [STATUS POSISI SAYA]
        - Tanggal Beli: %s
        - Harga Beli (Avg): Rp %.0f
        - Harga Saat Ini: Rp %.0f (Floating: %.2f%%)
        - Rekor Harga Pucuk: Rp %.0f
        - Batas Trailing Stop (TSL): %s
        - Batas Cut Loss: Rp %.0f

        ⚠️ **ATURAN EVALUASI** ⚠️
        %s

        [DATA FUNDAMENTAL & BERITA]
        %s

        [DATA TEKNIKAL]
        %s

        WAJIB gunakan format Markdown:

        🛡️ **Saham:** %s (Avg: Rp %.0f | Now: Rp %.0f)
        %s **Floating:** %.2f%%
        🚦 **Tindakan:** [AMAN (Hold) / WASPADA (Siap Jual) / KUNCI PROFIT (Jual) / BAHAYA (Cut Loss)]
        🎯 **Potensi Lanjut Naik:** [Tinggi / Sedang / Rendah]

        📝 **Saran Strategi:**
        [Berikan 2-3 kalimat tajam sesuai strategi BOW. Sebutkan posisi harga terhadap batas CL atau TSL (%s). Evaluasi juga apakah pergerakan harga sejalan dengan target waktu 1-5 hari dan target profit +4%%.]
    `,
		plan.Symbol, plan.BuyDate, plan.EntryPrice, currentPrice, floatingPNL, plan.HighestPrice, tslLabel, plan.CutLoss,
		strategyContext, newsContent, technicalContent,
		plan.Symbol, plan.EntryPrice, currentPrice, chartIcon, floatingPNL,
		tslLabel)

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
	config.DataMutex.RLock()
	if len(config.MyStocks) == 0 {
		config.DataMutex.RUnlock()
		log.Println("Portofolio kosong, skip evaluasi harian.")
		return
	}
	stocksToEval := make(map[string]models.TradingPlan)
	for k, v := range config.MyStocks {
		stocksToEval[k] = v
	}
	config.DataMutex.RUnlock()

	utils.SendSimpleMessage(bot, "🔄 _Menyiapkan Rapat: Evaluasi Portofolio & Keuangan..._")

	var finalReport strings.Builder
	finalReport.WriteString("📋 **EVALUASI PORTOFOLIO** 📋\n\n")

	var totalPNLRupiah float64

	for _, plan := range stocksToEval {
		// Increment HoldDays (TIME stop tracking)
		config.UpdatePlan(plan.Symbol, func(p *models.TradingPlan) {
			p.HoldDays++
		})
		storage.SaveData()

		currentPrice := market.GetLivePrice(plan.Symbol)
		if currentPrice == 0 {
			currentPrice = market.GetGooglePrice(plan.Symbol)
			if currentPrice == 0 {
				currentPrice = plan.EntryPrice
			}
		}

		totalBuyValue := plan.EntryPrice * (1 + rules.Default.BuyFee) * float64(plan.Lots) * 100
		totalSellValue := currentPrice * (1 - rules.Default.SellFee) * float64(plan.Lots) * 100
		pnlRupiah := totalSellValue - totalBuyValue
		totalPNLRupiah += pnlRupiah

		chartIcon := "📈"
		if currentPrice < plan.EntryPrice {
			chartIcon = "📉"
		}

		newsContent, err := research.FetchNewsRSS(plan.Symbol)
		if err != nil {
			newsContent = "Tidak ada berita terbaru."
		}
		technicalContent, errTech := research.FetchTechnicalData(plan.Symbol)
		if errTech != nil {
			technicalContent = "Tidak ada data teknikal terbaru."
		}

		eval, err := getPortfolioEvaluation(plan, currentPrice, newsContent, technicalContent, chartIcon)
		if err != nil {
			log.Println("❌ Gagal evaluasi", plan.Symbol, ":", err)
			finalReport.WriteString(fmt.Sprintf("⚠️ **%s**: Gagal mendapatkan AI.\n\n", plan.Symbol))
			continue
		}

		finalReport.WriteString(eval)
		finalReport.WriteString("\n━━━━━━━━━━━━━━━━━━━━\n\n")

		time.Sleep(3 * time.Second)
	}

	totalEmoji := "📈"
	pnlLabel := "Cuan Bersih"
	if totalPNLRupiah < 0 {
		totalEmoji = "📉"
		pnlLabel = "Rugi Bersih"
	}

	finalReport.WriteString(fmt.Sprintf("\n %s **Status Keseluruhan:** _%s_", pnlLabel, totalEmoji))
	finalReport.WriteString(fmt.Sprintf("\n💰 **Total Floating PNL: %s**", utils.FormatRupiah(totalPNLRupiah)))
	finalReport.WriteString("\n\n_Bot tetap siaga memantau. Gunakan evaluasi ini untuk pertimbangan eksekusi hari ini._")

	utils.SendMarkdownMessage(bot, finalReport.String())
}
