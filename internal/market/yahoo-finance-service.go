package market

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"learn-go/internal/models"
)

func GetLivePrice(symbol string) float64 {
	ticker := symbol
	if !strings.HasSuffix(symbol, ".JK") && symbol != "AAPL" && !strings.HasPrefix(symbol, "^") {
		ticker = symbol + ".JK"
	}

	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1m&range=1d", ticker)

	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)

	// Header wajib agar tidak dianggap bot
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ Koneksi Error: %v", err)
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ Yahoo Return Status: %d", resp.StatusCode)
		return 0
	}

	var data models.YahooChartResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Printf("❌ Gagal Decode JSON: %v", err)
		return 0
	}

	if len(data.Chart.Result) > 0 {
		result := data.Chart.Result[0]
		
		// Coba ambil harga terakhir dari array 'close' (biasanya lebih update beberapa detik)
		quotes := result.Indicators.Quote[0].Close
		if len(quotes) > 0 {
			// Ambil data non-null terakhir
			for i := len(quotes) - 1; i >= 0; i-- {
				if quotes[i] > 0 {
					return quotes[i]
				}
			}
		}
		
		// Fallback ke meta price
		return result.Meta.RegularMarketPrice
	}

	log.Printf("⚠️ Ticker %s tidak ditemukan di data chart", ticker)
	return 0
}

func GetHistoricalPrices(symbol string) (models.HistoricalData, error) {
	ticker := symbol
	if !strings.HasSuffix(symbol, ".JK") && !strings.HasPrefix(symbol, "^") { 
		ticker = symbol + ".JK" 
	}

	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?range=3mo&interval=1d", ticker)

	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil { return models.HistoricalData{}, err }
	defer resp.Body.Close()

	var data models.YahooChartResponse // Pakai struct yang sudah kita buat sebelumnya
	json.NewDecoder(resp.Body).Decode(&data)

	if len(data.Chart.Result) > 0 {
		return models.HistoricalData{
			Prices:  data.Chart.Result[0].Indicators.Quote[0].Close,
			Volumes: data.Chart.Result[0].Indicators.Quote[0].Volume,
			Highs:   data.Chart.Result[0].Indicators.Quote[0].High, // <-- TAMBAHKAN INI
			Opens:   data.Chart.Result[0].Indicators.Quote[0].Open, // <-- TAMBAHKAN INI
			Symbol:  symbol,
		}, nil
	}
	return models.HistoricalData{}, fmt.Errorf("data kosong")
}