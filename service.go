package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// Struktur JSON untuk endpoint /v8/finance/chart
type YahooChartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				RegularMarketPrice float64 `json:"regularMarketPrice"`
				PreviousClose      float64 `json:"previousClose"`
			} `json:"meta"`
			Indicators struct {
				Quote []struct {
					Close []float64 `json:"close"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
	} `json:"chart"`
}

func getLivePrice(symbol string) float64 {
	ticker := symbol
	if !strings.HasSuffix(symbol, ".JK") && symbol != "AAPL" {
		ticker = symbol + ".JK"
	}

	// Menggunakan endpoint /v8/finance/chart (Lebih stabil)
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

	var data YahooChartResponse
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

func getHistoricalPrices(symbol string) (HistoricalData, error) {
	ticker := symbol
	if !strings.HasSuffix(symbol, ".JK") { ticker = symbol + ".JK" }

	// Ambil data 3 bulan (range=3mo) dengan interval harian (interval=1d)
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?range=3mo&interval=1d", ticker)

	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil { return HistoricalData{}, err }
	defer resp.Body.Close()

	var data YahooChartResponse // Pakai struct yang sudah kita buat sebelumnya
	json.NewDecoder(resp.Body).Decode(&data)

	if len(data.Chart.Result) > 0 {
		return HistoricalData{
			Prices: data.Chart.Result[0].Indicators.Quote[0].Close,
			Symbol: symbol,
		}, nil
	}
	return HistoricalData{}, fmt.Errorf("data kosong")
}