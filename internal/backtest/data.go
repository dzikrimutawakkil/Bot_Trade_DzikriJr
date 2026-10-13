package backtest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"learn-go/internal/market"
	"learn-go/internal/models"
)


func LoadBars(symbol string, years int, refresh bool) ([]models.Bar, error) {
	cacheDir := "data/cache"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir cache: %w", err)
	}

	cacheFile := filepath.Join(cacheDir, symbol+".json")
	if !refresh {
		if bars, err := loadFromCache(cacheFile); err == nil && len(bars) > 0 {
			return bars, nil
		}
	}

	bars, err := market.GetBars(symbol, fmt.Sprintf("%dy", years))
	if err != nil {
		return nil, fmt.Errorf("GetBars: %w", err)
	}

	clean := market.CleanBars(bars)
	if len(clean) == 0 {
		return nil, fmt.Errorf("no clean bars for %s", symbol)
	}

	if err := saveToCache(cacheFile, clean); err != nil {
		fmt.Printf("warning: failed to cache %s: %v\n", symbol, err)
	}

	return clean, nil
}

func loadFromCache(path string) ([]models.Bar, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var bars []models.Bar
	if err := json.Unmarshal(data, &bars); err != nil {
		return nil, err
	}
	return bars, nil
}

func saveToCache(path string, bars []models.Bar) error {
	data, err := json.Marshal(bars)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func FilterTradingDays(bars []models.Bar) []models.Bar {
	trading := make([]models.Bar, 0, len(bars))
	for _, b := range bars {
		if b.Time.Weekday() != time.Saturday && b.Time.Weekday() != time.Sunday {
			trading = append(trading, b)
		}
	}
	return trading
}
