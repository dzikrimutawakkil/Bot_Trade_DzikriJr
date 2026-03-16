package main

import (
	"context"
	"fmt"
	"strings"
	"log"
	"time"
	"github.com/mmcdole/gofeed"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// Fungsi untuk ambil berita terbaru via RSS Google News
func fetchNewsRSS(symbol string) (string, error) {
	fp := gofeed.NewParser()
	// URL RSS Google News untuk keyword saham tertentu
	url := fmt.Sprintf("https://news.google.com/rss/search?q=saham+%s&hl=id-ID&gl=ID&ceid=ID:id", symbol)
	
	feed, err := fp.ParseURL(url)
	if err != nil {
		return "", err
	}

	var newsList []string
	// Ambil 10 berita teratas
	for i, item := range feed.Items {
		if i >= 10 {
			break
		}
		newsList = append(newsList, fmt.Sprintf("- %s", item.Title))
	}

	return strings.Join(newsList, "\n"), nil
}

func getDeepAnalysis(symbol string, newsContent string) (string, error) {
	// 1. Tambahkan Timeout 30 Detik (Biar nggak nunggu selamanya)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := genai.NewClient(ctx, option.WithAPIKey(GeminiAPIKey))
	if err != nil {
		return "", err
	}
	defer client.Close()

	// 2. Gunakan model yang paling "lincah" di daftar tadi
	model := client.GenerativeModel("gemini-flash-latest")

	log.Printf("[AI] Mengirim data berita %s ke Gemini...", symbol)
	
	prompt := fmt.Sprintf(`
    Analisis sentimen saham %s dari berita berikut.
    Berikan jawaban dalam teks biasa (jangan pakai simbol aneh atau garis bawah berlebihan).
    
    Format:
    Skor Sentimen: [angka]
    Keyword: [kata1, kata2]
    Kesimpulan: [1 kalimat]
    
    Berita: %s`, symbol, newsContent)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		// Jika kena timeout, ini akan tercetak di terminal
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