package models

import "time"

// Bar merepresentasikan satu candlestick OHLCV.
type Bar struct {
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

// Struktur JSON untuk endpoint /v8/finance/chart
type YahooChartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				RegularMarketPrice float64 `json:"regularMarketPrice"`
				PreviousClose      float64 `json:"previousClose"`
			} `json:"meta"`
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open   []float64 `json:"open"`
					High   []float64 `json:"high"`
					Low    []float64 `json:"low"`
					Close  []float64 `json:"close"`
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
	BuyDate      string
	TP1Done      bool
}

// HistoricalData sekarang menyimpan array OHLC utuh
type HistoricalData struct {
	Opens   []float64
	Highs   []float64
	Lows    []float64
	Prices  []float64 // Ini adalah Close
	Volumes []float64
	Symbol  string
}

type ActiveOrder struct {
	Symbol       string
	OrderPrice   float64
	Lot          int
	LastNotified time.Time
}
