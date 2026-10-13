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

var httpClient = &http.Client{Timeout: 15 * time.Second}

func GetLivePrice(symbol string) float64 {
	ticker := symbol
	if !strings.HasSuffix(symbol, ".JK") && symbol != "AAPL" && !strings.HasPrefix(symbol, "^") {
		ticker = symbol + ".JK"
	}

	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1m&range=1d", ticker)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")

	resp, err := httpClient.Do(req)
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

		if len(result.Indicators.Quote) == 0 || len(result.Indicators.Quote[0].Close) == 0 {
			return result.Meta.RegularMarketPrice
		}

		quotes := result.Indicators.Quote[0].Close
		for i := len(quotes) - 1; i >= 0; i-- {
			if quotes[i] > 0 {
				return quotes[i]
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
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			log.Printf("[Yahoo] Attempt %d/%d gagal koneksi untuk %s: %v", attempt, maxAttempts, symbol, err)
			if attempt < maxAttempts {
				time.Sleep(time.Duration(1<<(attempt-1)) * time.Second)
			}
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("status code %d", resp.StatusCode)
			log.Printf("[Yahoo] Attempt %d/%d gagal untuk %s: HTTP %d", attempt, maxAttempts, symbol, resp.StatusCode)
			resp.Body.Close()
			if attempt < maxAttempts {
				time.Sleep(time.Duration(1<<(attempt-1)) * time.Second)
			}
			continue
		}

		var data models.YahooChartResponse
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			lastErr = err
			log.Printf("[Yahoo] Attempt %d/%d gagal decode untuk %s: %v", attempt, maxAttempts, symbol, err)
			resp.Body.Close()
			if attempt < maxAttempts {
				time.Sleep(time.Duration(1<<(attempt-1)) * time.Second)
			}
			continue
		}
		resp.Body.Close()

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
			time.Sleep(time.Duration(1<<(attempt-1)) * time.Second)
		}
	}

	return models.HistoricalData{}, fmt.Errorf("[Yahoo] semua attempt gagal untuk %s: %v", symbol, lastErr)
}

// GetBars mengambil OHLCV bar dari Yahoo Finance dengan range tertentu.
func GetBars(symbol string, yahooRange string) ([]models.Bar, error) {
	ticker := symbol
	if !strings.HasSuffix(symbol, ".JK") && !strings.HasPrefix(symbol, "^") {
		ticker = symbol + ".JK"
	}

	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?range=%s&interval=1d", ticker, yahooRange)

	const maxAttempts = 3
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < maxAttempts {
				time.Sleep(time.Duration(1<<(attempt-1)) * time.Second)
			}
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("status code %d", resp.StatusCode)
			resp.Body.Close()
			if attempt < maxAttempts {
				time.Sleep(time.Duration(1<<(attempt-1)) * time.Second)
			}
			continue
		}

		var data models.YahooChartResponse
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			lastErr = err
			resp.Body.Close()
			if attempt < maxAttempts {
				time.Sleep(time.Duration(1<<(attempt-1)) * time.Second)
			}
			continue
		}
		resp.Body.Close()

		if len(data.Chart.Result) == 0 || len(data.Chart.Result[0].Indicators.Quote) == 0 {
			lastErr = fmt.Errorf("data kosong")
			continue
		}

		result := data.Chart.Result[0]
		quote := result.Indicators.Quote[0]

		bars := make([]models.Bar, 0, len(quote.Close))
		timestamps := result.Timestamp

		for i := 0; i < len(quote.Close); i++ {
			ts := int64(0)
			if i < len(timestamps) {
				ts = timestamps[i]
			}
			bars = append(bars, models.Bar{
				Time:   time.Unix(ts, 0).In(time.UTC),
				Open:   quote.Open[i],
				High:   quote.High[i],
				Low:    quote.Low[i],
				Close:  quote.Close[i],
				Volume: quote.Volume[i],
			})
		}

		return bars, nil
	}

	return nil, fmt.Errorf("[Yahoo] GetBars gagal untuk %s: %v", symbol, lastErr)
}

// CleanBars membuang bar dengan Open/High/Low/Close/Volume <= 0.
func CleanBars(raw []models.Bar) []models.Bar {
	clean := make([]models.Bar, 0, len(raw))
	for _, b := range raw {
		if b.Open > 0 && b.High > 0 && b.Low > 0 && b.Close > 0 && b.Volume > 0 {
			clean = append(clean, b)
		}
	}
	return clean
}
