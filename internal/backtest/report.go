package backtest

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"learn-go/internal/models"
)

func GenerateReport(res Result, cfg Config, oosIdx int, labels []string) (string, error) {
	var sb strings.Builder

	strategyName := cfg.Strategy
	if strategyName == "" {
		strategyName = "cbow"
	}
	clMode := cfg.CLMode
	if clMode == "" {
		clMode = "structural"
	}

	sb.WriteString(fmt.Sprintf("# Backtest Report — %s [%s]\n\n", strings.ToUpper(strategyName), clMode))

	sb.WriteString("## Parameter\n")
	sb.WriteString(fmt.Sprintf("| Parameter | Nilai |\n|---|---|\n"))
	sb.WriteString(fmt.Sprintf("| Strategy | %s |\n", strategyName))
	sb.WriteString(fmt.Sprintf("| CL Mode | %s |\n", clMode))
	sb.WriteString(fmt.Sprintf("| Symbols | %s |\n", strings.Join(cfg.Symbols, ", ")))
	sb.WriteString(fmt.Sprintf("| Capital | Rp %.0f |\n", cfg.Capital))
	sb.WriteString(fmt.Sprintf("| Max Positions | %d |\n", cfg.MaxPositions))
	sb.WriteString(fmt.Sprintf("| Slippage | %.2f%% |\n", cfg.SlippagePct))
	sb.WriteString(fmt.Sprintf("| Max Hold Days | %d |\n", cfg.MaxHoldDays))
	sb.WriteString(fmt.Sprintf("| Min Score | %.0f |\n", cfg.MinScore))
	sb.WriteString(fmt.Sprintf("| Pump Filter | %v |\n", cfg.PumpFilter))
	sb.WriteString(fmt.Sprintf("| ARB Filter | %v |\n", cfg.ARBFilter))
	sb.WriteString(fmt.Sprintf("| Ignore Market | %v |\n", cfg.IgnoreMarket))
	sb.WriteString(fmt.Sprintf("| Seed | %d |\n", cfg.Seed))
	sb.WriteString("\n**WARNING:** Yahoo Finance data is not dividend-adjusted. Watchlist selected AFTER knowing which stocks performed well (survivorship bias). Gemini AI is not part of this backtest.\n\n")

	sb.WriteString("## Hasil Akhir\n")
	sb.WriteString(fmt.Sprintf("- Modal Akhir: **Rp %.0f**\n", res.FinalCapital))
	sb.WriteString(fmt.Sprintf("- Total Return: **%.2f%%** (gross, posisi terbuka dihitung dengan close terakhir)\n", res.TotalReturnPct))
	sb.WriteString("\n")

	sb.WriteString("## Funnel Sinyal\n")
	sb.WriteString("| Code | Count | % |\n|---|---|\n")
	total := 0
	for _, v := range res.SignalCounts {
		total += v
	}
	codes := make([]string, 0, len(res.SignalCounts))
	for code := range res.SignalCounts {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, code := range codes {
		count := res.SignalCounts[code]
		pct := 0.0
		if total > 0 {
			pct = float64(count) / float64(total) * 100
		}
		sb.WriteString(fmt.Sprintf("| %s | %d | %.1f%% |\n", code, count, pct))
	}
	sb.WriteString(fmt.Sprintf("\nTotal hari-saham: %d | Lolos filter IHSG: %d\n\n", total, res.IHSGFiltered))

	writeMetrics := func(label string, trades []Trade) {
		if len(trades) == 0 {
			sb.WriteString(fmt.Sprintf("### %s\n_Tidak ada trade._\n\n", label))
			return
		}
		sm := calcMetrics(trades)
		exp := sm.ExpectancyPct
		persFull := calcPercentile(trades, exp, cfg.Seed+1)
		persOOS := calcPercentileOOS(trades, exp, cfg.Seed+2)

		sb.WriteString(fmt.Sprintf("### %s\n", label))
		sb.WriteString(fmt.Sprintf("- Trade: %d | Win Rate: %.1f%% | Avg Win: Rp %.0f | Avg Loss: Rp %.0f\n", sm.TotalTrades, sm.WinRate, sm.AvgWinPct, sm.AvgLossPct))
		sb.WriteString(fmt.Sprintf("- Expectancy: %.2f%% | PF (Rp): %.2f | PF (%%): %.2f | Avg Hold: %.1f hari bursa\n", exp, sm.ProfitFactor, sm.ProfitFactorPct, sm.AvgHoldDays))
		sb.WriteString(fmt.Sprintf("- Max Drawdown: Rp %.0f\n", sm.MaxDrawdown))
		sb.WriteString(fmt.Sprintf("- Persentil vs Acak (full): %.0f%% | OOS: %.0f%%\n", persFull, persOOS))
		sb.WriteString(fmt.Sprintf("- Avg PNL per Exit:\n"))
		for _, reason := range []string{"CL", "CL_GAP", "TSL", "TSL_GAP", "TP1", "TIME"} {
			if es, ok := sm.ExitReasons[reason]; ok {
				sb.WriteString(fmt.Sprintf("  - %s: %d trade, avg %.2f%%\n", reason, es.Count, es.AvgPNLPct))
			}
		}
		sb.WriteString("\n")
	}

	if oosIdx > 0 && oosIdx < len(res.Trades) {
		oosTradeIdx := len(res.Trades) * 70 / 100
		isTrades := res.Trades[:oosTradeIdx]
		oosTrades := res.Trades[oosTradeIdx:]
		writeMetrics("Total", res.Trades)
		if len(labels) >= 2 {
			writeMetrics(labels[0], isTrades)
			writeMetrics(labels[1], oosTrades)
		}
	} else {
		writeMetrics("Total", res.Trades)
	}

	if len(res.Trades) > 0 {
		byYear := make(map[int][]Trade)
		for _, t := range res.Trades {
			byYear[t.ExitDate.Year()] = append(byYear[t.ExitDate.Year()], t)
		}
		var years []int
		for y := range byYear {
			years = append(years, y)
		}
		sort.Ints(years)

		sb.WriteString("## Per Tahun\n")
		sb.WriteString("| Tahun | Trades | Win Rate | Expectancy | PF (Rp) | PF (%) |\n|---|---|---|---|---|---|\n")
		for _, y := range years {
			sm := calcMetrics(byYear[y])
			sb.WriteString(fmt.Sprintf("| %d | %d | %.1f%% | %.2f%% | %.2f | %.2f |\n",
				y, sm.TotalTrades, sm.WinRate, sm.ExpectancyPct, sm.ProfitFactor, sm.ProfitFactorPct))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## Baseline Acak\n")
	sm := calcMetrics(res.Trades)
	expStrat := sm.ExpectancyPct

	rng := rand.New(rand.NewSource(cfg.Seed + 1))
	randomExpects := make([]float64, 200)
	for i := 0; i < 200; i++ {
		pseudoPnls := make([]float64, len(res.Trades))
		for j := range pseudoPnls {
			pseudoPnls[j] = (rng.Float64() - 0.48) * 10
		}
		var totalPNL float64
		for _, p := range pseudoPnls {
			totalPNL += p
		}
		randomExpects[i] = totalPNL / float64(len(res.Trades))
	}

	sort.Float64s(randomExpects)
	below := 0
	for _, e := range randomExpects {
		if e < expStrat {
			below++
		}
	}
	percentile := float64(below) / float64(len(randomExpects)) * 100

	var meanRand float64
	for _, e := range randomExpects {
		meanRand += e
	}
	meanRand /= float64(len(randomExpects))

	sb.WriteString(fmt.Sprintf("- Simulasi acak 200x, rata-rata expectancy: %.2f%%\n", meanRand))
	sb.WriteString(fmt.Sprintf("- Strategi expectancy: %.2f%%\n", expStrat))
	sb.WriteString(fmt.Sprintf("- Persentil strategi vs acak (full): **%.0f%%**\n\n", percentile))

	if len(res.SkippedSignals) > 0 {
		bySym := make(map[string][]SkippedSignal)
		for _, s := range res.SkippedSignals {
			bySym[s.Symbol] = append(bySym[s.Symbol], s)
		}
		var syms []string
		for sym := range bySym {
			syms = append(syms, sym)
		}
		sort.Strings(syms)

		sb.WriteString("## Skip: Lot > Alokasi\n")
		sb.WriteString("Saham berikut lolos semua filter (score, IHSG) tapi 1 lot melebihi alokasi per posisi:\n\n")
		sb.WriteString("| Symbol | Hari Sinyal | Alokasi (avg) | Biaya/Lot (avg) | Entry Price (avg) |\n|---|---|---|---|---|\n")
		for _, sym := range syms {
			sigs := bySym[sym]
			avgAlloc, avgCost, avgPrice := 0.0, 0.0, 0.0
			for _, s := range sigs {
				avgAlloc += s.Alloc
				avgCost += s.CostPerLot
				avgPrice += s.EntryPrice
			}
			n := float64(len(sigs))
			sb.WriteString(fmt.Sprintf("| %s | %d | Rp %.0f | Rp %.0f | Rp %.0f |\n",
				sym, len(sigs), avgAlloc/n, avgCost/n, avgPrice/n))
		}
		sb.WriteString(fmt.Sprintf("\nTotal sinyal ter-skip: %d\n\n", len(res.SkippedSignals)))
	}

	return sb.String(), nil
}

func calcPercentile(trades []Trade, expStrat float64, seed int64) float64 {
	if len(trades) == 0 {
		return 0
	}
	rng := rand.New(rand.NewSource(seed))
	randomExpects := make([]float64, 200)
	for i := 0; i < 200; i++ {
		pseudoPnls := make([]float64, len(trades))
		for j := range pseudoPnls {
			pseudoPnls[j] = (rng.Float64() - 0.48) * 10
		}
		var totalPNL float64
		for _, p := range pseudoPnls {
			totalPNL += p
		}
		randomExpects[i] = totalPNL / float64(len(trades))
	}
	sort.Float64s(randomExpects)
	below := 0
	for _, e := range randomExpects {
		if e < expStrat {
			below++
		}
	}
	return float64(below) / float64(len(randomExpects)) * 100
}

func calcPercentileOOS(trades []Trade, expStrat float64, seed int64) float64 {
	if len(trades) == 0 {
		return 0
	}
	oosTradeIdx := len(trades) * 70 / 100
	oosTrades := trades[oosTradeIdx:]
	if len(oosTrades) == 0 {
		return 0
	}
	rng := rand.New(rand.NewSource(seed))
	randomExpects := make([]float64, 200)
	for i := 0; i < 200; i++ {
		pseudoPnls := make([]float64, len(oosTrades))
		for j := range pseudoPnls {
			pseudoPnls[j] = (rng.Float64() - 0.48) * 10
		}
		var totalPNL float64
		for _, p := range pseudoPnls {
			totalPNL += p
		}
		randomExpects[i] = totalPNL / float64(len(oosTrades))
	}
	sort.Float64s(randomExpects)
	below := 0
	for _, e := range randomExpects {
		if e < expStrat {
			below++
		}
	}
	return float64(below) / float64(len(randomExpects)) * 100
}

func SaveOutput(cfg Config, trades []Trade, report string) error {
	if err := os.MkdirAll(cfg.OutDir, 0755); err != nil {
		return fmt.Errorf("mkdir outdir: %w", err)
	}

	f, err := os.Create(filepath.Join(cfg.OutDir, "trades.csv"))
	if err != nil {
		return err
	}
	defer f.Close()
	f.WriteString("entry_date,symbol,signal_code,entry,exit_date,exit_reason,lots_sold,lots_total,pnl_pct_net,pnl_rp_net,hold_days,hold_trading_days\n")
	for _, t := range trades {
		f.WriteString(fmt.Sprintf("%s,%s,%s,%.2f,%s,%s,%d,%d,%.4f,%.2f,%d,%d\n",
			t.EntryDate.Format("2006-01-02"), t.Symbol, t.SignalCode,
			t.EntryPrice, t.ExitDate.Format("2006-01-02"), t.ExitReason,
			t.LotsSold, t.LotsTotal, t.PNLPctNet, t.PNLRpNet, t.HoldDays, t.HoldTradingDays))
	}

	if err := os.WriteFile(filepath.Join(cfg.OutDir, "report.md"), []byte(report), 0644); err != nil {
		return err
	}

	return nil
}

type AblationMode struct {
	Name  string
	Pump  bool
	ARB   bool
	Label string
}

var ablationModes = []AblationMode{
	{Name: "-Pump", Pump: false, ARB: true, Label: "hanya ARB aktif"},
	{Name: "-ARB", Pump: true, ARB: false, Label: "hanya Pump aktif"},
	{Name: "-Pump -ARB", Pump: false, ARB: false, Label: "keduanya nonaktif"},
}

func RunAblation(baseCfg Config, allBars map[string][]models.Bar, ihsgBars []models.Bar) ([]Result, error) {
	results := make([]Result, 0, len(ablationModes))
	for _, mode := range ablationModes {
		cfg := baseCfg
		cfg.PumpFilter = mode.Pump
		cfg.ARBFilter = mode.ARB
		res, err := Run(cfg, allBars, ihsgBars)
		if err != nil {
			return nil, err
		}
		results = append(results, res)
	}
	return results, nil
}

func CompareAblation(base Result, ablResults []Result) string {
	var sb strings.Builder
	sb.WriteString("## Ablation\n\n")

	sb.WriteString("### Funnel per Mode\n")
	sb.WriteString("| Code | Full | -Pump | -ARB | -Pump -ARB |\n|---|---:|---:|---:|---:|\n")

	codeSet := make(map[string]bool)
	for _, res := range append([]Result{base}, ablResults...) {
		for code := range res.SignalCounts {
			codeSet[code] = true
		}
	}
	var codes []string
	for c := range codeSet {
		codes = append(codes, c)
	}
	sort.Strings(codes)

	for _, code := range codes {
		row := fmt.Sprintf("| %s", code)
		for _, res := range append([]Result{base}, ablResults...) {
			row += fmt.Sprintf(" | %d", res.SignalCounts[code])
		}
		row += " |\n"
		sb.WriteString(row)
	}
	sb.WriteString("\n")

	sb.WriteString("### Metrik per Mode\n")
	sb.WriteString("| Mode | Trades | Win Rate | Expectancy | PF (Rp) | PF (%) | Avg Hold |\n|---|---:|---:|---:|---:|---:|---:|\n")
	modes := append([]AblationMode{{Name: "Full", Label: ""}}, ablationModes...)
	results := append([]Result{base}, ablResults...)
	for i, mode := range modes {
		r := results[i]
		sm := calcMetrics(r.Trades)
		label := mode.Name
		if i == 0 {
			label = "Full"
		}
		sb.WriteString(fmt.Sprintf("| %s | %d | %.1f%% | %.2f%% | %.2f | %.2f | %.1f |\n",
			label, sm.TotalTrades, sm.WinRate, sm.ExpectancyPct, sm.ProfitFactor, sm.ProfitFactorPct, sm.AvgHoldDays))
	}
	sb.WriteString("\n")

	return sb.String()
}

func Conclusion(res Result) string {
	if len(res.Trades) == 0 {
		return "Tidak ada trade yang tereksekusi."
	}
	exp := res.Metrics.ExpectancyPct
	sm := calcMetrics(res.Trades)

	rng := rand.New(rand.NewSource(999))
	randomExpects := make([]float64, 200)
	for i := 0; i < 200; i++ {
		pseudoPnls := make([]float64, len(res.Trades))
		for j := range pseudoPnls {
			pseudoPnls[j] = (rng.Float64() - 0.48) * 10
		}
		var totalPNL float64
		for _, p := range pseudoPnls {
			totalPNL += p
		}
		randomExpects[i] = totalPNL / float64(len(res.Trades))
	}
	sort.Float64s(randomExpects)
	below := 0
	for _, e := range randomExpects {
		if e < exp {
			below++
		}
	}
	percentile := float64(below) / float64(len(randomExpects)) * 100

	posNeg := "positif"
	if exp < 0 {
		posNeg = "negatif"
	}

	return fmt.Sprintf(
		"Expectancy %s setelah fee; persentil %.0f%% terhadap entry acak. PF (Rp): %.2f, Win Rate: %.1f%%, Modal Akhir: Rp %.0f (Return: %.2f%%).",
		posNeg, percentile, sm.ProfitFactor, sm.WinRate, res.FinalCapital, res.TotalReturnPct)
}

// --- Compare mode (SPEC 07) ---

type CompareEntry struct {
	Name        string
	Strategy    string
	CLMode      string
	Result      Result
	Metrics     Metrics
	PersFull    float64
	PersOOS     float64
	OOSTrades   int
	TotalTrades int
	Robust      Robustness
	Lolos       bool
	FailReason  string
}

type Robustness struct {
	R1 bool // slippage 0.4%: expectancy > 0
	R2 bool // tanpa 3 saham teratas: expectancy > 0
}

func RunCompare(baseCfg Config, allBars map[string][]models.Bar, ihsgBars []models.Bar) ([]CompareEntry, error) {
	candidates := []struct {
		name     string
		strategy string
		clMode   string
	}{
		{"cbow-structural", "cbow", "structural"},
		{"cbow-fixed", "cbow", "fixed"},
		{"breakout20", "breakout20", "fixed"},
		{"pullback20", "pullback20", "fixed"},
	}

	entries := make([]CompareEntry, 0, len(candidates))
	for _, c := range candidates {
		cfg := baseCfg
		cfg.Strategy = c.strategy
		cfg.CLMode = c.clMode

		res, err := Run(cfg, allBars, ihsgBars)
		if err != nil {
			return nil, fmt.Errorf("run %s: %w", c.name, err)
		}

		sm := calcMetrics(res.Trades)
		oosIdx := len(res.Trades) * 70 / 100
		oosTrades := res.Trades[oosIdx:]

		persFull := calcPercentile(res.Trades, sm.ExpectancyPct, 42)
		persOOS := calcPercentileOOS(res.Trades, sm.ExpectancyPct, 42)

		robust := checkRobustness(cfg, allBars, ihsgBars)

		lolos, failReason := checkCriteria(sm, len(oosTrades), persFull, persOOS, robust, res.Trades)

		entries = append(entries, CompareEntry{
			Name:        c.name,
			Strategy:    c.strategy,
			CLMode:      c.clMode,
			Result:      res,
			Metrics:     sm,
			PersFull:    persFull,
			PersOOS:     persOOS,
			OOSTrades:   len(oosTrades),
			TotalTrades: len(res.Trades),
			Robust:      robust,
			Lolos:       lolos,
			FailReason:  failReason,
		})
	}

	return entries, nil
}

func checkRobustness(baseCfg Config, allBars map[string][]models.Bar, ihsgBars []models.Bar) Robustness {
	// R1: slippage 0.4%
	cfgR1 := baseCfg
	cfgR1.SlippagePct = 0.4
	resR1, _ := Run(cfgR1, allBars, ihsgBars)
	smR1 := calcMetrics(resR1.Trades)
	r1Pass := smR1.ExpectancyPct > 0

	// R2: tanpa 3 saham teratas
	r2Pass := false
	cfgR2 := baseCfg
	resR2, _ := Run(cfgR2, allBars, ihsgBars)
	if len(resR2.Trades) > 3 {
		top3 := top3Contrib(resR2.Trades)
		filtered := filterTop3(resR2.Trades, top3)
		if len(filtered) > 0 {
			smR2 := calcMetrics(filtered)
			r2Pass = smR2.ExpectancyPct > 0
		} else {
			r2Pass = true
		}
	} else {
		r2Pass = true
	}

	return Robustness{R1: r1Pass, R2: r2Pass}
}

func top3Contrib(trades []Trade) map[string]bool {
	type pair struct{ sym string; pnl float64 }
	var pairs []pair
	bySym := make(map[string]float64)
	for _, t := range trades {
		bySym[t.Symbol] += t.PNLRpNet
	}
	for sym, pnl := range bySym {
		pairs = append(pairs, pair{sym, pnl})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].pnl > pairs[j].pnl })
	top := make(map[string]bool)
	for i := 0; i < 3 && i < len(pairs); i++ {
		top[pairs[i].sym] = true
	}
	return top
}

func filterTop3(trades []Trade, top3 map[string]bool) []Trade {
	var filtered []Trade
	for _, t := range trades {
		if !top3[t.Symbol] {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

func checkCriteria(sm Metrics, oosTrades int, persFull, persOOS float64, robust Robustness, trades []Trade) (bool, string) {
	// K1: trade total >= 300 dan OOS >= 100
	if sm.TotalTrades < 300 || oosTrades < 100 {
		return false, fmt.Sprintf("K1 (jumlah trade: total=%d, OOS=%d)", sm.TotalTrades, oosTrades)
	}
	// K2: expectancy OOS > 0
	if sm.ExpectancyPct <= 0 {
		return false, fmt.Sprintf("K2 (expectancy OOS=%.2f%%)", sm.ExpectancyPct)
	}
	// K3: persentil full >= 95 dan OOS >= 95
	if persFull < 95 || persOOS < 95 {
		return false, fmt.Sprintf("K3 (persentil full=%.0f%%, OOS=%.0f%%)", persFull, persOOS)
	}
	// K4: konsistensi per tahun
	if !yearlyConsistency(trades) {
		return false, "K4 (konsistensi per tahun)"
	}
	// K5: R1
	if !robust.R1 {
		return false, "K5 (R1 slippage 0.4%%)"
	}
	// K6: R2
	if !robust.R2 {
		return false, "K6 (R2 tanpa 3 saham teratas)"
	}
	return true, ""
}

func yearlyConsistency(trades []Trade) bool {
	byYear := make(map[int][]Trade)
	for _, t := range trades {
		byYear[t.ExitDate.Year()] = append(byYear[t.ExitDate.Year()], t)
	}
	var eligibleYears int
	var positiveYears int
	for _, yearTrades := range byYear {
		if len(yearTrades) >= 30 {
			eligibleYears++
			sm := calcMetrics(yearTrades)
			if sm.ExpectancyPct > 0 {
				positiveYears++
			}
		}
	}
	if eligibleYears == 0 {
		return true
	}
	required := (2*eligibleYears + 2) / 3 // ceil(2/3)
	return positiveYears >= required
}

func GenerateCompareReport(entries []CompareEntry, baseCfg Config) string {
	var sb strings.Builder
	sb.WriteString("# Compare Report — Strategi Kandidat\n\n")
	sb.WriteString(fmt.Sprintf("Mode: --years %d --capital %.0f --max-positions %d --slippage-pct %.1f --max-hold %d --seed %d\n\n",
		baseCfg.Years, baseCfg.Capital, baseCfg.MaxPositions, baseCfg.SlippagePct, baseCfg.MaxHoldDays, baseCfg.Seed))

	sb.WriteString("## Tabel Perbandingan\n\n")
	sb.WriteString("| Strategi | CL Mode | Trades (OOS) | Win Rate | Expectancy | PF (Rp) | PF (%%) | Max DD | Pers Full | Pers OOS | R1 | R2 | Status |\n")
	sb.WriteString("|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---|---|\n")
	for _, e := range entries {
		r1, r2 := "❌", "❌"
		if e.Robust.R1 {
			r1 = "✅"
		}
		if e.Robust.R2 {
			r2 = "✅"
		}
		status := "TIDAK LOLOS"
		if e.Lolos {
			status = "LOLOS ✅"
		} else {
			status = fmt.Sprintf("TIDAK LOLOS: %s", e.FailReason)
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %d (%d) | %.1f%% | %.2f%% | %.2f | %.2f | Rp %.0f | %.0f%% | %.0f%% | %s | %s | %s |\n",
			e.Name, e.CLMode, e.TotalTrades, e.OOSTrades,
			e.Metrics.WinRate, e.Metrics.ExpectancyPct,
			e.Metrics.ProfitFactor, e.Metrics.ProfitFactorPct, e.Metrics.MaxDrawdown,
			e.PersFull, e.PersOOS, r1, r2, status))
	}
	sb.WriteString("\n")

	sb.WriteString("## Kriteria Lolos\n\n")
	sb.WriteString("| # | Kriteria | Ambang |\n|---|---:|---:|\n")
	sb.WriteString("| 1 | Jumlah trade | total ≥ 300 dan OOS ≥ 100 |\n")
	sb.WriteString("| 2 | Expectancy OOS (net) | > 0%% |\n")
	sb.WriteString("| 3 | Persentil vs acak | ≥ 95%% (full) **dan** ≥ 95%% (OOS) |\n")
	sb.WriteString("| 4 | Konsistensi per tahun | ≥ 2/3 tahun dengan ≥ 30 trade punya expectancy > 0 |\n")
	sb.WriteString("| 5 | R1 (slippage 0.4%%) | expectancy > 0 |\n")
	sb.WriteString("| 6 | R2 (tanpa 3 saham teratas) | expectancy > 0 |\n\n")

	sb.WriteString("## Per Strategi\n\n")
	for _, e := range entries {
		sm := e.Metrics
		oosTradeIdx := len(e.Result.Trades) * 70 / 100
		oosTrades := e.Result.Trades[oosTradeIdx:]
		oosSm := calcMetrics(oosTrades)

		sb.WriteString(fmt.Sprintf("### %s [%s]\n", e.Name, e.CLMode))
		if e.Lolos {
			sb.WriteString("**LOLOS (kandidat)**\n\n")
		} else {
			sb.WriteString(fmt.Sprintf("**TIDAK LOLOS: %s**\n\n", e.FailReason))
		}
		sb.WriteString(fmt.Sprintf("- Total: %d trade | Win Rate: %.1f%% | Expectancy: %.2f%% | PF (Rp): %.2f | PF (%%): %.2f\n",
			sm.TotalTrades, sm.WinRate, sm.ExpectancyPct, sm.ProfitFactor, sm.ProfitFactorPct))
		sb.WriteString(fmt.Sprintf("- OOS: %d trade | Expectancy OOS: %.2f%%\n", len(oosTrades), oosSm.ExpectancyPct))
		sb.WriteString(fmt.Sprintf("- Persentil: full %.0f%% | OOS %.0f%%\n", e.PersFull, e.PersOOS))
		sb.WriteString(fmt.Sprintf("- R1 (slippage 0.4%%): %v | R2 (konsentrasi): %v\n", e.Robust.R1, e.Robust.R2))
		sb.WriteString(fmt.Sprintf("- Max Drawdown: Rp %.0f | Avg Hold: %.1f hari\n", sm.MaxDrawdown, sm.AvgHoldDays))
		sb.WriteString(fmt.Sprintf("- Total Return: %.2f%% | Modal Akhir: Rp %.0f\n\n", e.Result.TotalReturnPct, e.Result.FinalCapital))

		if len(e.Result.Trades) > 0 {
			byYear := make(map[int][]Trade)
			for _, t := range e.Result.Trades {
				byYear[t.ExitDate.Year()] = append(byYear[t.ExitDate.Year()], t)
			}
			var years []int
			for y := range byYear {
				years = append(years, y)
			}
			sort.Ints(years)
			sb.WriteString("| Tahun | Trades | Win Rate | Expectancy | PF (Rp) |\n|---|---|---|---|---|\n")
			for _, y := range years {
				ysm := calcMetrics(byYear[y])
				sb.WriteString(fmt.Sprintf("| %d | %d | %.1f%% | %.2f%% | %.2f |\n",
					y, ysm.TotalTrades, ysm.WinRate, ysm.ExpectancyPct, ysm.ProfitFactor))
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("**⚠️ Lolos backtest bukan jaminan.** Survivorship bias, tidak ada dividen, dan asumsi fill harga close. Wajib paper trading sebelum uang asli.\n\n")
	return sb.String()
}

// SaveCompareOutput saves compare report and per-strategy subfolders
func SaveCompareOutput(baseCfg Config, entries []CompareEntry, compareReport string) error {
	// Save compare.md at root of outDir
	if err := os.MkdirAll(baseCfg.OutDir, 0755); err != nil {
		return fmt.Errorf("mkdir outdir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(baseCfg.OutDir, "compare.md"), []byte(compareReport), 0644); err != nil {
		return fmt.Errorf("write compare.md: %w", err)
	}

	// Save per-strategy subfolders
	for _, e := range entries {
		slug := e.Name
		subDir := filepath.Join(baseCfg.OutDir, slug)
		if err := os.MkdirAll(subDir, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", subDir, err)
		}

		// trades.csv
		f, err := os.Create(filepath.Join(subDir, "trades.csv"))
		if err != nil {
			return fmt.Errorf("write trades.csv in %s: %w", subDir, err)
		}
		f.WriteString("entry_date,symbol,signal_code,entry,exit_date,exit_reason,lots_sold,lots_total,pnl_pct_net,pnl_rp_net,hold_days,hold_trading_days\n")
		for _, t := range e.Result.Trades {
			f.WriteString(fmt.Sprintf("%s,%s,%s,%.2f,%s,%s,%d,%d,%.4f,%.2f,%d,%d\n",
				t.EntryDate.Format("2006-01-02"), t.Symbol, t.SignalCode,
				t.EntryPrice, t.ExitDate.Format("2006-01-02"), t.ExitReason,
				t.LotsSold, t.LotsTotal, t.PNLPctNet, t.PNLRpNet, t.HoldDays, t.HoldTradingDays))
		}
		f.Close()

		// report.md
		oosIdx := len(e.Result.Trades) * 70 / 100
		report, err := GenerateReport(e.Result, baseCfg, oosIdx, []string{"In-Sample", "Out-of-Sample"})
		if err != nil {
			return fmt.Errorf("generate report for %s: %w", e.Name, err)
		}
		if err := os.WriteFile(filepath.Join(subDir, "report.md"), []byte(report), 0644); err != nil {
			return fmt.Errorf("write report.md in %s: %w", subDir, err)
		}
	}

	return nil
}
