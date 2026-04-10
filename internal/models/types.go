package models

import "time"

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
					Open   []float64 `json:"open"`
					High   []float64 `json:"high"`
					Close []float64 `json:"close"`
					Volume []float64 `json:"volume"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
	} `json:"chart"`
}

type Recommendation struct {
	Symbol       string
	Score        float64
	Status       string
	DistToMA     float64
	DeepAnalysis string
	Sentiment    float64
	MA20         float64
}

// TradingPlan: Struktur data untuk saham yang dipantau
type TradingPlan struct {
	Symbol       string
	EntryPrice   float64
	TakeProfit   float64
	CutLoss      float64
	HighestPrice float64
	Lots         int
	LastNotified time.Time
}

// [FIX 2] Ubah namanya kembali jadi HistoricalData karena service.go nyari nama ini
type HistoricalData struct {
	Prices []float64
	Volumes []float64
	Highs   []float64
	Opens   []float64
	Symbol string
}

type ActiveOrder struct {
	Symbol       string
	OrderPrice   float64
	Lot          int
	LastNotified time.Time
}