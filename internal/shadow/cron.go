package shadow

import (
	"log"
	"time"

	"learn-go/internal/config"
	"learn-go/internal/market"
	"learn-go/internal/models"
	"learn-go/internal/research"
	"learn-go/internal/rules"
	"learn-go/internal/signals"
	"learn-go/internal/utils"
)

// RecordSignals runs at 16:30 WIB on trading days.
// It evaluates all watchlist stocks + IHSG, calls Gemini for Score >= 7,
// and appends rows to signals_log.csv.
func RecordSignals() {
	jakarta, _ := time.LoadLocation("Asia/Jakarta")
	now := time.Now().In(jakarta)

	if !utils.IsTradingDay(now) {
		return
	}

	todayStr := now.Format("2006-01-02")
	log.Printf("[Shadow] RecordSignals mulai untuk %s", todayStr)

	barsCache := make(map[string][]models.Bar)

	ihsgBars, err := market.GetBars(IHSG, "3mo")
	if err != nil {
		log.Printf("[Shadow] Gagal ambil IHSG: %v", err)
		return
	}
	ihsgClean := market.CleanBars(ihsgBars)
	barsCache[IHSG] = ihsgClean

	_, marketState := signals.MarketSafe(toCloses(ihsgClean))

	symbols := config.Watchlist
	logged := 0
	aiCalls := 0

	for _, sym := range symbols {
		bars, ok := barsCache[sym]
		if !ok {
			b, err := market.GetBars(sym, "3mo")
			if err != nil {
				log.Printf("[Shadow] Gagal ambil %s: %v", sym, err)
				continue
			}
			bars = market.CleanBars(b)
			barsCache[sym] = bars
		}

		if len(bars) < 20 {
			continue
		}

		lastBar := bars[len(bars)-1]
		lastBarDate := lastBar.Time.In(jakarta).Format("2006-01-02")
		if lastBarDate != todayStr {
			log.Printf("[Shadow] Skip %s: bar terakhir %s bukan hari ini (%s)", sym, lastBarDate, todayStr)
			continue
		}

		sig := signals.Evaluate(bars, signals.Params{
			MinScore:      MinScore,
			PumpFilterPct: signals.DefaultParams().PumpFilterPct,
			ARBDropPct:    signals.DefaultParams().ARBDropPct,
		})

		if sig.Score < MinScore {
			continue
		}

		planCL := rules.Default.StructuralCutLoss(sig.Low)
		planTP1 := rules.Default.TP1Price(bars[len(bars)-1].Close)

		row := Row{
			Date:        todayStr,
			Symbol:      sym,
			Score:       sig.Score,
			Code:        sig.Code,
			MarketState: marketState,
			Close:       bars[len(bars)-1].Close,
			SignalLow:   sig.Low,
			PlanCL:      planCL,
			PlanTP1:     planTP1,
			AIReco:      "NA",
			Model:       config.GeminiModel,
		}

		if aiCalls < MaxAICallsPerDay {
			news, _ := research.FetchNewsRSS(sym)
			tech, _ := research.FetchTechnicalData(sym)
			analysis, err := research.GetDeepAnalysis(sym, news, tech)

			if err == nil && analysis != "" {
				row.AIReco = ParseAIReco(analysis)
				row.AISentiment = ParseSentiment(analysis)
				row.AIConfidence = ParseConfidence(analysis)
				log.Printf("[Shadow] %s: ai_reco=%s, sentiment=%.1f, conf=%.1f%%",
					sym, row.AIReco, row.AISentiment, row.AIConfidence)
			} else {
				log.Printf("[Shadow] Gemini gagal untuk %s: %v", sym, err)
			}
			aiCalls++

			if aiCalls < MaxAICallsPerDay {
				time.Sleep(time.Duration(AICallDelaySec) * time.Second)
			}
		} else {
			log.Printf("[Shadow] Maks AI calls (%d) tercapai, skip Gemini untuk %s", MaxAICallsPerDay, sym)
		}

		rows, err := LoadRows()
		if err != nil {
			log.Printf("[Shadow] Gagal LoadRows: %v", err)
			continue
		}
		if rows == nil {
			rows = []Row{}
		}

		if HasRow(todayStr, sym, rows) {
			log.Printf("[Shadow] Skip %s %s: sudah ada", todayStr, sym)
			continue
		}

		rows = AddOrUpdateRow(rows, row)
		if err := SaveRows(rows); err != nil {
			log.Printf("[Shadow] Gagal SaveRows untuk %s: %v", sym, err)
			continue
		}
		logged++
	}

	log.Printf("[Shadow] RecordSignals selesai: %d sinyal dicatat, %d panggilan AI", logged, aiCalls)
}

// FillOutcomes runs at 16:50 WIB on trading days.
// It fills outcome columns for rows that are >= 5 trading days old.
func FillOutcomes() {
	jakarta, _ := time.LoadLocation("Asia/Jakarta")
	now := time.Now().In(jakarta)

	if !utils.IsTradingDay(now) {
		return
	}

	rows, err := LoadRows()
	if err != nil {
		log.Printf("[Shadow] FillOutcomes: gagal LoadRows: %v", err)
		return
	}
	if rows == nil {
		return
	}

	updated := false
	for i := range rows {
		if rows[i].Filled {
			continue
		}

		sigDate, err := time.Parse("2006-01-02", rows[i].Date)
		if err != nil {
			continue
		}

		tradingDays := TradingDaysBetween(sigDate, now)
		if tradingDays < 5 {
			continue
		}

		bars, err := market.GetBars(rows[i].Symbol, "3mo")
		if err != nil {
			log.Printf("[Shadow] FillOutcomes: gagal ambil bar %s: %v", rows[i].Symbol, err)
			continue
		}
		clean := market.CleanBars(bars)
		if len(clean) < 5 {
			continue
		}

		idx := -1
		for j := len(clean) - 1; j >= 0; j-- {
			barDate := clean[j].Time.In(jakarta).Format("2006-01-02")
			if barDate == rows[i].Date {
				idx = j
				break
			}
		}
		if idx < 0 {
			continue
		}

		startIdx := idx + 1
		if startIdx+5 > len(clean) {
			continue
		}
		futureBars := clean[startIdx : startIdx+5]

		rows[i] = FillRow(rows[i], futureBars)
		updated = true
	}

	if updated {
		if err := SaveRows(rows); err != nil {
			log.Printf("[Shadow] FillOutcomes: gagal SaveRows: %v", err)
			return
		}
		log.Printf("[Shadow] FillOutcomes: %d baris di-update", countFilled(rows))
	}
}

func toCloses(bars []models.Bar) []float64 {
	out := make([]float64, len(bars))
	for i, b := range bars {
		out[i] = b.Close
	}
	return out
}

func countFilled(rows []Row) int {
	n := 0
	for _, r := range rows {
		if r.Filled {
			n++
		}
	}
	return n
}
