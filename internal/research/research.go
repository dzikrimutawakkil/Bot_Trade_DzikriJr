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
	// Panggil API Yahoo Chart (1 bulan terakhir, interval harian)
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s.JK?interval=1d&range=1mo", symbol)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("User-Agent", "Mozilla/5.0")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gagal mengambil data teknikal untuk %s: %w", symbol, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("gagal membaca response body untuk %s: %w", symbol, err)
	}

	var data models.YahooChartResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("gagal parsing JSON response untuk %s: %w", symbol, err)
	}

	if len(data.Chart.Result) == 0 || len(data.Chart.Result[0].Indicators.Quote) == 0 {
		return "", fmt.Errorf("data teknikal tidak ditemukan di bursa untuk %s", symbol)
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
		return "", fmt.Errorf("data historis kurang dari 20 hari untuk %s, indikator tidak valid", symbol)
	}

	// 1. Hitung MA20 (Harga)
	last20Price := validCloses[len(validCloses)-20:]
	var sumPrice float64
	for _, val := range last20Price {
		sumPrice += val
	}
	ma20 := sumPrice / 20.0
	currentPrice := validCloses[len(validCloses)-1]

	last5Price := validCloses[len(validCloses)-5:]
	var sum5Price float64
	for _, val := range last5Price {
		sum5Price += val
	}
	ma5 := sum5Price / 5.0

	// 2. Hitung Rata-rata Volume 20 Hari (Volume MA20)
	last20Vol := validVolumes[len(validVolumes)-20:]
	var sumVol float64
	for _, val := range last20Vol {
		sumVol += val
	}
	avgVol20 := sumVol / 20.0
	currentVol := validVolumes[len(validVolumes)-1]

	// 3. LOGIKA BARU: Analisis Volume untuk Buy on Weakness
	statusVolume := "✅ VOLUME NORMAL"
	if currentVol > (avgVol20 * 1.5) {
		// Dulu ini bagus, sekarang ini bahaya (rawan distribusi pucuk)
		statusVolume = "🔥 LONJAKAN VOLUME (Waspada Puncak Distribusi / Guyuran)"
	} else if currentVol < (avgVol20 * 0.7) {
		// Ini incaran kita: Harga turun tapi yang jual sudah habis
		statusVolume = "📉 VOLUME KERING (Tekanan jual mereda, ritel sudah habis barang)"
	}

	// 4. LOGIKA BARU: Deteksi Jarak ke Support MA20 & Setup BoW
	jarakKeMA20 := ((currentPrice - ma20) / ma20) * 100
	statusBoW := "TIDAK ADA SETUP (Sideways / Tanggung)"

	if currentPrice < ma20 {
		// Harga tembus ke bawah MA20
		statusBoW = "💀 PISAU JATUH (Di bawah MA20, Hindari!)"
	} else if currentPrice < ma5 && jarakKeMA20 <= 3.0 && jarakKeMA20 > 0 {
		// Harga lagi turun (di bawah MA5), tapi jaraknya dekat banget sama MA20 (maksimal 3%)
		statusBoW = "🟢 SETUP BUY ON WEAKNESS (Harga koreksi sehat, sangat dekat Support MA20)"
	} else if jarakKeMA20 > 5.0 {
		// Harga terlalu jauh dari MA20, rawan dibanting turun
		statusBoW = "🔴 RAWAN PUCUK (Harga terlalu tinggi dari MA20, rawan koreksi)"
	}

	// Ambil pergerakan 5 hari terakhir
	last5 := validCloses[len(validCloses)-5:]
	var trendStr []string
	for _, val := range last5 {
		trendStr = append(trendStr, fmt.Sprintf("%.0f", val))
	}

	// Rangkuman Laporan Baru untuk diumpankan ke AI
	report := fmt.Sprintf("Harga Terakhir: Rp %.0f\nMA5 (Tren Pendek): Rp %.0f\nMA20 (Support Utama): Rp %.0f\nJarak ke MA20: +%.2f%%\nStatus Setup: %s\nStatus Volume: %s\nHarga 5 Hari Terakhir: %s",
		currentPrice, ma5, ma20, jarakKeMA20, statusBoW, statusVolume, strings.Join(trendStr, " -> "))

	return report, nil
}

// Fungsi AI yang sudah di-UPGRADE (Menerima input Teknikal)
func GetDeepAnalysis(symbol string, newsContent string, technicalContent string) (string, error) {
	const maxAttempts = 3

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()

		client, err := genai.NewClient(ctx, option.WithAPIKey(config.GeminiAPIKey))
		if err != nil {
			log.Printf("[Gemini] Attempt %d/%d gagal init client untuk %s: %v", attempt, maxAttempts, symbol, err)
			if attempt < maxAttempts {
				time.Sleep(time.Duration(1<<(attempt-1)) * time.Second)
			}
			continue
		}
		defer client.Close()

		model := client.GenerativeModel("gemini-flash-latest")
		model.Temperature = genai.Ptr(float32(0.0))
		log.Printf("[AI] Mengirim data BERITA dan TEKNIKAL %s ke Gemini... (attempt %d/%d)", symbol, attempt, maxAttempts)

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
			log.Printf("[AI] Attempt %d/%d gagal untuk %s: %v", attempt, maxAttempts, symbol, err)
			if attempt < maxAttempts {
				time.Sleep(time.Duration(1<<(attempt-1)) * time.Second)
			}
			continue
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

	// Semua attempt gagal
	errAllFailed := fmt.Errorf("[Gemini] semua %d attempt gagal untuk %s", maxAttempts, symbol)
	log.Printf("[AI] FATAL: %v", errAllFailed)
	return "", errAllFailed
}