package research

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/mmcdole/gofeed"
	"google.golang.org/api/option"

	"learn-go/internal/config"
	"learn-go/internal/market"
	"learn-go/internal/models"
)

func FetchNewsRSS(symbol string) (string, error) {
	fp := gofeed.NewParser()
	url := fmt.Sprintf("https://news.google.com/rss/search?q=saham+%s&hl=id-ID&gl=ID&ceid=ID:id", symbol)

	feed, err := fp.ParseURL(url)
	if err != nil {
		return "", fmt.Errorf("gagal fetch RSS untuk %s: %w", symbol, err)
	}

	var newsList []string
	for i, item := range feed.Items {
		if i >= 10 {
			break
		}
		newsList = append(newsList, fmt.Sprintf("- %s", item.Title))
	}
	return strings.Join(newsList, "\n"), nil
}

func FetchTechnicalData(symbol string) (string, error) {
	bars, err := market.GetBars(symbol, "1mo")
	if err != nil {
		return "", fmt.Errorf("gagal mengambil data teknikal untuk %s: %w", symbol, err)
	}

	clean := market.CleanBars(bars)
	if len(clean) < 20 {
		return "", fmt.Errorf("data historis kurang dari 20 hari untuk %s, indikator tidak valid", symbol)
	}

	validCloses := extractFieldF64(clean, func(b models.Bar) float64 { return b.Close })
	validVolumes := extractFieldF64(clean, func(b models.Bar) float64 { return b.Volume })

	last20Price := validCloses[len(validCloses)-20:]
	ma20 := sum(last20Price) / 20.0
	currentPrice := validCloses[len(validCloses)-1]

	last5Price := validCloses[len(validCloses)-5:]
	ma5 := sum(last5Price) / 5.0

	last20Vol := validVolumes[len(validVolumes)-20:]
	avgVol20 := sum(last20Vol) / 20.0
	currentVol := validVolumes[len(validVolumes)-1]

	statusVolume := "✅ VOLUME NORMAL"
	if currentVol > (avgVol20 * 1.5) {
		statusVolume = "🔥 LONJAKAN VOLUME (Waspada Puncak Distribusi / Guyuran)"
	} else if currentVol < (avgVol20 * 0.7) {
		statusVolume = "📉 VOLUME KERING (Tekanan jual mereda, ritel sudah habis barang)"
	}

	jarakKeMA20 := ((currentPrice - ma20) / ma20) * 100
	statusBoW := "TIDAK ADA SETUP (Sideways / Tanggung)"

	if currentPrice < ma20 {
		statusBoW = "💀 PISAU JATUH (Di bawah MA20, Hindari!)"
	} else if currentPrice < ma5 && jarakKeMA20 <= 3.0 && jarakKeMA20 > 0 {
		statusBoW = "🟢 SETUP BUY ON WEAKNESS (Harga koreksi sehat, sangat dekat Support MA20)"
	} else if jarakKeMA20 > 5.0 {
		statusBoW = "🔴 RAWAN PUCUK (Harga terlalu tinggi dari MA20, rawan koreksi)"
	}

	last5 := validCloses[len(validCloses)-5:]
	var trendStr []string
	for _, val := range last5 {
		trendStr = append(trendStr, fmt.Sprintf("%.0f", val))
	}

	report := fmt.Sprintf("Harga Terakhir: Rp %.0f\nMA5 (Tren Pendek): Rp %.0f\nMA20 (Support Utama): Rp %.0f\nJarak ke MA20: +%.2f%%\nStatus Setup: %s\nStatus Volume: %s\nHarga 5 Hari Terakhir: %s",
		currentPrice, ma5, ma20, jarakKeMA20, statusBoW, statusVolume, strings.Join(trendStr, " -> "))

	return report, nil
}

func GetDeepAnalysis(symbol string, newsContent string, technicalContent string) (string, error) {
	const maxAttempts = 3

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result, err := doDeepAnalysisAttempt(symbol, newsContent, technicalContent, attempt)
		if err == nil {
			return result, nil
		}
		log.Printf("[AI] Attempt %d gagal untuk %s: %v", attempt, symbol, err)
		if attempt < maxAttempts {
			time.Sleep(time.Duration(1<<(attempt-1)) * time.Second)
		}
	}

	return "", fmt.Errorf("[Gemini] semua %d attempt gagal untuk %s", maxAttempts, symbol)
}

func doDeepAnalysisAttempt(symbol, newsContent, technicalContent string, attempt int) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	client, err := genai.NewClient(ctx, option.WithAPIKey(config.GeminiAPIKey))
	if err != nil {
		return "", fmt.Errorf("init client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel(config.GeminiModel)
	model.Temperature = genai.Ptr(float32(0.0))
	log.Printf("[AI] Mengirim data ke Gemini untuk %s... (attempt %d/3, model=%s)", symbol, attempt, config.GeminiModel)

	prompt := fmt.Sprintf(`
		Bertindaklah sebagai Analis Saham Profesional khusus **FAST SWING (Hold 1-5 Hari)** dengan strategi **BUY ON WEAKNESS (BoW) / Contrarian**.
		Analisis saham %s berdasarkan data berikut:

		[DATA FUNDAMENTAL & SENTIMEN BERITA]
		%s

		[DATA TEKNIKAL]
		%s

		⚠️ **ATURAN TRADING (WAJIB DIIKUTI!)** ⚠️
		1. **STRATEGI UTAMA (BoW):** Rekomendasikan BELI HANYA JIKA "Status Setup" adalah "🟢 SETUP BUY ON WEAKNESS" (harga terkoreksi mendekati MA20) DAN volume menunjukkan "📉 VOLUME KERING". Ini berarti tekanan jual ritel sudah habis.
		2. **HINDARI PUCUK FOMO:** Jika "Status Setup" menunjukkan "🔴 RAWAN PUCUK" atau ada "🔥 LONJAKAN VOLUME" setelah harga naik berhari-hari, rekomendasikan JUAL/TAHAN. Itu adalah jebakan distribusi bandit (Sell on News).
		3. **HARAM MENGAMBIL PISAU JATUH:** Jika statusnya "💀 PISAU JATUH", WAJIB rekomendasikan HINDARI.

		WAJIB gunakan format persis seperti di bawah ini. Gunakan pemformatan Markdown:

		🎯 **Skor Sentimen:** [Angka 1-10]/10
		🤖 **AI Confidence:** [Angka 0-100]%%
		🚀 **Katalis Utama:** [Tulis HANYA 1 kalimat singkat alasan paling kuat untuk beli/hindari]
		📊 **Tren Teknikal:** [Pullback ke Support / Overextended / Downtrend] (Berikan emoji yang sesuai)
		🌊 **Volume:** [Sebutkan apakah Kering (Bagus) atau Lonjakan (Bahaya)]
		🔑 **Kata Kunci:** [3-5 kata kunci]

		📝 **Kesimpulan Analisis:**
		[Tulis 2-3 kalimat. Jelaskan mengapa koreksi saat ini adalah peluang beli murah (BoW) berdasarkan rendahnya volume, ATAU jelaskan mengapa harga saat ini terlalu pucuk untuk dikejar. Abaikan RSI.]

		[PILIH HANYA SALAH SATU FORMAT REKOMENDASI DI BAWAH INI:]
		🟢 **REKOMENDASI: BELI**
		🟡 **REKOMENDASI: TAHAN / PANTAU**
		🔴 **REKOMENDASI: HINDARI (SKIP)**

		_Alasan: [Satu kalimat solid fokus pada risiko (Risk/Reward) dan jarak harga terhadap garis MA20]_
	`, symbol, newsContent, technicalContent)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("generate content: %w", err)
	}
	log.Printf("[AI] Respon berhasil diterima!")

	if len(resp.Candidates) > 0 {
		var sb strings.Builder
		for _, part := range resp.Candidates[0].Content.Parts {
			sb.WriteString(fmt.Sprintf("%v", part))
		}
		result := sb.String()
		if result == "" {
			return "", fmt.Errorf("respons kosong")
		}
		return result, nil
	}

	return "", fmt.Errorf("respons kosong")
}

func extractFieldF64(bars []models.Bar, fn func(models.Bar) float64) []float64 {
	out := make([]float64, len(bars))
	for i, b := range bars {
		out[i] = fn(b)
	}
	return out
}

func sum(vals []float64) float64 {
	s := 0.0
	for _, v := range vals {
		s += v
	}
	return s
}
