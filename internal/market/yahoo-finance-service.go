package market

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

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
		
		quotes := result.Indicators.Quote[0].Close
		if len(quotes) > 0 {
			for i := len(quotes) - 1; i >= 0; i-- {
				if quotes[i] > 0 {
					return quotes[i]
				}
			}
		}
		
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

	const maxAttempts = 3
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		client := &http.Client{}
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			log.Printf("[Yahoo] Attempt %d/%d gagal koneksi untuk %s: %v", attempt, maxAttempts, symbol, err)
			if attempt < maxAttempts {
				sleepSec := 1 << (attempt - 1) // 1s, 2s, 4s
				time.Sleep(time.Duration(sleepSec) * time.Second)
			}
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("status code %d", resp.StatusCode)
			log.Printf("[Yahoo] Attempt %d/%d gagal untuk %s: HTTP %d", attempt, maxAttempts, symbol, resp.StatusCode)
			if attempt < maxAttempts {
				sleepSec := 1 << (attempt - 1)
				time.Sleep(time.Duration(sleepSec) * time.Second)
			}
			continue
		}

		var data models.YahooChartResponse
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			lastErr = err
			log.Printf("[Yahoo] Attempt %d/%d gagal decode untuk %s: %v", attempt, maxAttempts, symbol, err)
			if attempt < maxAttempts {
				sleepSec := 1 << (attempt - 1)
				time.Sleep(time.Duration(sleepSec) * time.Second)
			}
			continue
		}

		if len(data.Chart.Result) > 0 && len(data.Chart.Result[0].Indicators.Quote) > 0 {
			quote := data.Chart.Result[0].Indicators.Quote[0]
			return models.HistoricalData{
				Opens:   quote.Open,
				Highs:   quote.High,
				Lows:    quote.Low,
				Prices:  quote.Close,
				Volumes: quote.Volume,
				Symbol:  symbol,
			}, nil
		}

		lastErr = fmt.Errorf("data kosong")
		log.Printf("[Yahoo] Attempt %d/%d gagal untuk %s: data kosong", attempt, maxAttempts, symbol)
		if attempt < maxAttempts {
			sleepSec := 1 << (attempt - 1)
			time.Sleep(time.Duration(sleepSec) * time.Second)
		}
	}

	return models.HistoricalData{}, fmt.Errorf("[Yahoo] semua attempt gagal untuk %s: %v", symbol, lastErr)
}