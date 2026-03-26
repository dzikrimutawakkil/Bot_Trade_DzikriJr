package research

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/mmcdole/gofeed"
	"google.golang.org/api/option"
    
    "learn-go/internal/config"
    "learn-go/internal/models"
)

// Fungsi untuk ambil berita terbaru via RSS Google News (Fundamental)
func FetchNewsRSS(symbol string) (string, error) {
	fp := gofeed.NewParser()
	url := fmt.Sprintf("https://news.google.com/rss/search?q=saham+%s&hl=id-ID&gl=ID&ceid=ID:id", symbol)

	feed, err := fp.ParseURL(url)
	if err != nil {
		return "", err
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

func FetchTechnicalData(symbol string) string {
    // Panggil API Yahoo Chart (1 bulan terakhir, interval harian)
    url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s.JK?interval=1d&range=1mo", symbol)
    req, _ := http.NewRequest("GET", url, nil)
    req.Header.Add("User-Agent", "Mozilla/5.0") 

    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return "Data teknikal gagal diambil."
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    // Asumsi struct struct YahooChart sudah disesuaikan
    var data models.YahooChartResponse
    json.Unmarshal(body, &data)

    if len(data.Chart.Result) == 0 || len(data.Chart.Result[0].Indicators.Quote) == 0 {
        return "Data teknikal tidak ditemukan di bursa."
    }

    closes := data.Chart.Result[0].Indicators.Quote[0].Close
    volumes := data.Chart.Result[0].Indicators.Quote[0].Volume

    var validCloses []float64
    var validVolumes []float64

    // Filter data kosong (hari libur) dan sinkronkan harga dengan volume
    for i, c := range closes {
        if c > 0 { 
            validCloses = append(validCloses, c)
            if i < len(volumes) {
                validVolumes = append(validVolumes, volumes[i])
            }
        }
    }

    if len(validCloses) < 20 || len(validVolumes) < 20 {
        return "Data historis kurang dari 20 hari, indikator tidak valid."
    }

    // 1. Hitung MA20 (Harga)
    last20Price := validCloses[len(validCloses)-20:]
    var sumPrice float64
    for _, val := range last20Price {
        sumPrice += val
    }
    ma20 := sumPrice / 20.0
    currentPrice := validCloses[len(validCloses)-1]

    statusMA := "DI BAWAH MA20 (Downtrend/Lemah)"
    if currentPrice > ma20 {
        statusMA = "DI ATAS MA20 (Uptrend/Kuat)"
    }

    // 2. Hitung Rata-rata Volume 20 Hari (Volume MA20)
    last20Vol := validVolumes[len(validVolumes)-20:]
    var sumVol float64
    for _, val := range last20Vol {
        sumVol += val
    }
    avgVol20 := sumVol / 20.0
    currentVol := validVolumes[len(validVolumes)-1]

    // 3. Analisis Lonjakan Volume
    statusVolume := "⚠️ VOLUME RENDAH (Kurang Konfirmasi)"
    if currentVol > (avgVol20 * 1.5) { // Volume melonjak 50% di atas rata-rata
        statusVolume = "🔥 LONJAKAN VOLUME (Validasi Kuat/Akumulasi)"
    } else if currentVol > avgVol20 {
        statusVolume = "✅ VOLUME NORMAL (Di Atas Rata-rata)"
    }

    // Ambil pergerakan 5 hari terakhir
    last5 := validCloses[len(validCloses)-5:]
    var trendStr []string
    for _, val := range last5 {
        trendStr = append(trendStr, fmt.Sprintf("%.0f", val))
    }

    report := fmt.Sprintf("Harga Terakhir: Rp %.0f\nMA20: Rp %.0f\nStatus Teknikal: %s\nStatus Volume: %s\nHarga 5 Hari Terakhir: %s",
        currentPrice, ma20, statusMA, statusVolume, strings.Join(trendStr, " -> "))

    return report
}

// Fungsi AI yang sudah di-UPGRADE (Menerima input Teknikal)
func GetDeepAnalysis(symbol string, newsContent string, technicalContent string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := genai.NewClient(ctx, option.WithAPIKey(config.GeminiAPIKey))
	if err != nil {
		return "", err
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-flash-latest")
	log.Printf("[AI] Mengirim data BERITA dan TEKNIKAL %s ke Gemini...", symbol)

	// Prompt yang jauh lebih canggih & tajam
	// Prompt yang sudah di-UPGRADE untuk tampilan Telegram yang cantik
	prompt := fmt.Sprintf(`
		Bertindaklah sebagai Analis Saham Profesional.
		Analisis saham %s berdasarkan data berikut:

		[DATA FUNDAMENTAL & SENTIMEN BERITA]
		%s

		[DATA TEKNIKAL]
		%s

		WAJIB gunakan format persis seperti di bawah ini. Gunakan pemformatan Markdown:

		🎯 **Skor Sentimen:** [Angka 1-10]/10
		📊 **Tren Teknikal:** [Bullish / Bearish / Sideways] (Berikan emoji 📈/📉/↔️)
		🌊 **Volume:** [Tuliskan apakah akumulasi kuat atau sepi]
		🔑 **Kata Kunci:** [3-5 kata kunci]

		📝 **Kesimpulan Analisis:**
		[Tulis 2-3 kalimat padat. Jangan buat satu paragraf panjang yang sumpek, gunakan enter/baris baru jika perlu agar nyaman dibaca.]

		🟢 **REKOMENDASI: [BELI / TAHAN / JUAL]** Alasan: [Satu kalimat penjelasan yang solid]
		`, symbol, newsContent, technicalContent)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Printf("[AI] Gagal dapat respon: %v", err)
		return "", err
	}
	log.Printf("[AI] Respon berhasil diterima!")

	if len(resp.Candidates) > 0 {
		var sb strings.Builder
		for _, part := range resp.Candidates[0].Content.Parts {
			sb.WriteString(fmt.Sprintf("%v", part))
		}
		return sb.String(), nil
	}

	return "AI terdiam tanpa kata.", nil
}