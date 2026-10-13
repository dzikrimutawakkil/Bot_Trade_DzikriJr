package backtest

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const weeklyOOSFrom = "2024-01-01"

func splitTradesByOOS(trades []Trade, fromDate time.Time) (isTrades, oosTrades []Trade) {
	for _, t := range trades {
		if t.EntryDate.Before(fromDate) {
			isTrades = append(isTrades, t)
		} else {
			oosTrades = append(oosTrades, t)
		}
	}
	return
}

func splitEquityByOOS(points []EquityPoint, fromDate time.Time) (isPoints, oosPoints []EquityPoint) {
	for _, p := range points {
		if p.Date.Before(fromDate) {
			isPoints = append(isPoints, p)
		} else {
			oosPoints = append(oosPoints, p)
		}
	}
	return
}

// GenerateWeeklyReport writes a report.md covering criteria K1-K7 and the
// per-year breakdown, using trades split by 2024-01-01 for OOS.
func GenerateWeeklyReport(res Result, cfg Config) (string, error) {
	var sb strings.Builder

	strategyName := cfg.Strategy
	if strategyName == "" {
		strategyName = "momentum-weekly"
	}

	sb.WriteString(fmt.Sprintf("# Backtest Report — %s (Weekly DCA)\n\n", strings.ToUpper(strategyName)))

	sb.WriteString("## Parameter\n")
	sb.WriteString("| Parameter | Nilai |\n|---|---|\n")
	sb.WriteString(fmt.Sprintf("| Strategy | %s |\n", strategyName))
	sb.WriteString(fmt.Sprintf("| Symbols | %s |\n", strings.Join(cfg.Symbols, ", ")))
	sb.WriteString(fmt.Sprintf("| Capital Awal | Rp %.0f |\n", cfg.Capital))
	sb.WriteString(fmt.Sprintf("| DCA Bulanan | Rp %.0f |\n", cfg.DCAMonthly))
	sb.WriteString(fmt.Sprintf("| Max Positions | %d |\n", cfg.MaxPositions))
	sb.WriteString(fmt.Sprintf("| Slippage | %.2f%% |\n", cfg.SlippagePct))
	sb.WriteString(fmt.Sprintf("| CL (fixed) | -%.1f%% |\n", cfg.CLFixedPct))
	sb.WriteString(fmt.Sprintf("| TP | +%.1f%% |\n", cfg.TPPct))
	sb.WriteString(fmt.Sprintf("| Max Hold (hari bursa) | %d |\n", cfg.MaxHoldDays))
	sb.WriteString(fmt.Sprintf("| Years | %d |\n", cfg.Years))
	sb.WriteString(fmt.Sprintf("| Seed | %d |\n", cfg.Seed))
	sb.WriteString("\n**WARNING:** Yahoo Finance data tidak include dividen, tidak dilakukan survivorship-bias adjustment, dan asumsi fill di harga open. Gemini AI tidak terlibat dalam backtest ini.\n\n")

	oosFrom, _ := time.Parse("2006-01-02", weeklyOOSFrom)
	isTrades, oosTrades := splitTradesByOOS(res.Trades, oosFrom)

	isMetrics := calcMetrics(isTrades)
	oosMetrics := calcMetrics(oosTrades)
	fullMetrics := res.Metrics

	years := res.EquityCurve
	if len(years) > 0 {
		sb.WriteString(fmt.Sprintf("- Rentang data: %s → %s\n", years[0].Date.Format("2006-01-02"), years[len(years)-1].Date.Format("2006-01-02")))
	}
	sb.WriteString(fmt.Sprintf("- Total hari tercatat: %d\n", len(res.EquityCurve)))
	sb.WriteString(fmt.Sprintf("- Total modal di-inject: Rp %.0f (modal awal + %d kali DCA Rp %.0f)\n",
		cfg.Capital+cfg.DCAMonthly*float64(len(uniqueMonths(res.EquityCurve))),
		len(uniqueMonths(res.EquityCurve))-1,
		cfg.DCAMonthly))
	sb.WriteString("\n")

	sb.WriteString("## Hasil Akhir\n")
	sb.WriteString(fmt.Sprintf("- Modal Akhir: **Rp %.0f**\n", res.FinalCapital))
	sb.WriteString(fmt.Sprintf("- Total Injected: Rp %.0f\n", cfg.Capital+cfg.DCAMonthly*float64(len(uniqueMonths(res.EquityCurve))-1)))
	sb.WriteString(fmt.Sprintf("- Profit (vs injected): **Rp %.0f** (%.2f%%)\n", res.FinalCapital-(cfg.Capital+cfg.DCAMonthly*float64(len(uniqueMonths(res.EquityCurve))-1)), res.TotalReturnPct))
	maxDD := 0.0
	for _, p := range res.EquityCurve {
		if p.Drawdown > maxDD {
			maxDD = p.Drawdown
		}
	}
	sb.WriteString(fmt.Sprintf("- Max Drawdown: %.2f%%\n", maxDD))
	sb.WriteString("\n")

	sb.WriteString("## Kriteria Lolos (K1–K7)\n")
	sb.WriteString("| # | Kriteria | Ambang | Hasil | Lolos? |\n|---|---|---|---|---|\n")

	yearsEligible := 0
	yearsPositive := 0
	if len(oosTrades) > 0 {
		byYear := make(map[int][]Trade)
		for _, t := range oosTrades {
			byYear[t.ExitDate.Year()] = append(byYear[t.ExitDate.Year()], t)
		}
		for _, ytr := range byYear {
			if len(ytr) >= 5 {
				yearsEligible++
				if calcMetrics(ytr).ExpectancyPct > 0 {
					yearsPositive++
				}
			}
		}
	}
	k1 := len(res.Trades) >= 100
	k2 := oosMetrics.ExpectancyPct > 0
	k3 := oosMetrics.WinRate > 45
	k5 := maxDD < 50
	k6 := yearsEligible == 0 || yearsPositive*3 >= yearsEligible*2

	cell := func(b bool) string {
		if b {
			return "✅"
		}
		return "❌"
	}
	sb.WriteString(fmt.Sprintf("| K1 | Total trades | ≥ 100 | %d | %s |\n", len(res.Trades), cell(k1)))
	sb.WriteString(fmt.Sprintf("| K2 | **Expectancy OOS net** | **> 0%%** | **%.2f%%** | **%s** |\n", oosMetrics.ExpectancyPct, cell(k2)))
	sb.WriteString(fmt.Sprintf("| K3 | Win rate OOS | > 45%% | %.1f%% | %s |\n", oosMetrics.WinRate, cell(k3)))
	sb.WriteString("| K4 | R1 (slippage 0.3%%) | expectancy OOS > 0 | (lihat compare.md) | — |\n")
	sb.WriteString(fmt.Sprintf("| K5 | Max DD | < 50%% | %.2f%% | %s |\n", maxDD, cell(k5)))
	sb.WriteString(fmt.Sprintf("| K6 | Konsisten | ≥ 2/3 tahun OOS exp > 0 | %d/%d tahun | %s |\n", yearsPositive, yearsEligible, cell(k6)))
	sb.WriteString("| K7 | Beat Baseline | > random3 AND > DCA naif | (lihat compare.md) | — |\n\n")

	sb.WriteString("## Metrik Lengkap\n")
	sb.WriteString("| Window | Trades | Win Rate | Expectancy | PF (Rp) | PF (%%) | Avg Hold |\n|---|---:|---:|---:|---:|---:|---:|\n")
	sb.WriteString(fmt.Sprintf("| Full | %d | %.1f%% | %.2f%% | %.2f | %.2f | %.1f |\n",
		fullMetrics.TotalTrades, fullMetrics.WinRate, fullMetrics.ExpectancyPct,
		fullMetrics.ProfitFactor, fullMetrics.ProfitFactorPct, fullMetrics.AvgHoldDays))
	sb.WriteString(fmt.Sprintf("| In-Sample (sebelum %s) | %d | %.1f%% | %.2f%% | %.2f | %.2f | %.1f |\n",
		weeklyOOSFrom, isMetrics.TotalTrades, isMetrics.WinRate, isMetrics.ExpectancyPct,
		isMetrics.ProfitFactor, isMetrics.ProfitFactorPct, isMetrics.AvgHoldDays))
	sb.WriteString(fmt.Sprintf("| OOS (%s+) | %d | %.1f%% | %.2f%% | %.2f | %.2f | %.1f |\n",
		weeklyOOSFrom, oosMetrics.TotalTrades, oosMetrics.WinRate, oosMetrics.ExpectancyPct,
		oosMetrics.ProfitFactor, oosMetrics.ProfitFactorPct, oosMetrics.AvgHoldDays))
	sb.WriteString("\n")

	if len(res.Trades) > 0 {
		byYear := make(map[int][]Trade)
		for _, t := range res.Trades {
			byYear[t.EntryDate.Year()] = append(byYear[t.EntryDate.Year()], t)
		}
		var ykeys []int
		for y := range byYear {
			ykeys = append(ykeys, y)
		}
		sort.Ints(ykeys)

		sb.WriteString("## Per Tahun (berdasarkan entry date)\n")
		sb.WriteString("| Tahun | Trades | Win Rate | Expectancy | PF (Rp) | Injected | Net PnL |\n|---|---:|---:|---:|---:|---:|---:|\n")
		for _, y := range ykeys {
			ysm := calcMetrics(byYear[y])
			yearInjected := cfg.DCAMonthly * 12
			if y == ykeys[0] {
				yearInjected = cfg.Capital + cfg.DCAMonthly*12
			}
			yearPNL := 0.0
			for _, t := range byYear[y] {
				yearPNL += t.PNLRpNet
			}
			sb.WriteString(fmt.Sprintf("| %d | %d | %.1f%% | %.2f%% | %.2f | Rp %.0f | Rp %.0f |\n",
				y, ysm.TotalTrades, ysm.WinRate, ysm.ExpectancyPct, ysm.ProfitFactor,
				yearInjected, yearPNL))
		}
		sb.WriteString("\n")
	}

	if len(res.Trades) > 0 {
		sb.WriteString("## Exit Reasons\n")
		sb.WriteString("| Reason | Count | Avg PNL %% |\n|---|---:|---:|\n")
		reasons := make([]string, 0, len(fullMetrics.ExitReasons))
		for r := range fullMetrics.ExitReasons {
			reasons = append(reasons, r)
		}
		sort.Strings(reasons)
		for _, r := range reasons {
			es := fullMetrics.ExitReasons[r]
			sb.WriteString(fmt.Sprintf("| %s | %d | %.2f%% |\n", r, es.Count, es.AvgPNLPct))
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// GenerateCompareWeeklyReport writes a compare.md contrasting the momentum
// strategy vs the three baselines (RANDOM3, DCA_NAIF, BUY_HOLD). It also
// prints the R1 (slippage 0.3%) robustness outcome if the caller supplies
// the re-run Result via r1Res. Pass nil to skip K4.
func GenerateCompareWeeklyReport(res Result, cfg Config, baselines []BaselineResult, r1Res *Result) (string, error) {
	var sb strings.Builder

	sb.WriteString("# Compare Report — Weekly Momentum vs Baselines\n\n")
	sb.WriteString(fmt.Sprintf("Mode: --strategy %s --years %d --capital %.0f --dca-monthly %.0f --max-positions %d --slippage-pct %.1f --seed %d\n\n",
		cfg.Strategy, cfg.Years, cfg.Capital, cfg.DCAMonthly, cfg.MaxPositions, cfg.SlippagePct, cfg.Seed))

	oosFrom, _ := time.Parse("2006-01-02", weeklyOOSFrom)
	_, oosTrades := splitTradesByOOS(res.Trades, oosFrom)
	_, oosEquity := splitEquityByOOS(res.EquityCurve, oosFrom)
	oosMetrics := calcMetrics(oosTrades)
	totalInjectedMain := cfg.Capital + cfg.DCAMonthly*float64(len(uniqueMonths(res.EquityCurve))-1)
	years := res.EquityCurve
	span := 0.0
	if len(years) > 1 {
		span = years[len(years)-1].Date.Sub(years[0].Date).Hours() / 24 / 365.0
	}
	annualized := 0.0
	if span > 0 {
		annualized = res.TotalReturnPct / span
	}
	maxDD := 0.0
	for _, p := range res.EquityCurve {
		if p.Drawdown > maxDD {
			maxDD = p.Drawdown
		}
	}

	momentumOOSFinal := finalEquityInPeriod(oosEquity)
	momentumOOSReturn := 0.0
	if totalInjectedMain > 0 {
		momentumOOSReturn = (momentumOOSFinal - totalInjectedMain) / totalInjectedMain * 100
	}

	sb.WriteString("## Tabel Perbandingan\n\n")
	sb.WriteString("Catatan: kolom *Equity* dan *Profit* dihitung dari rentang OOS saja (2024-01-01 → akhir backtest). *Return vs Injected* dan *Expectancy per Trade* membandingkan strategi dengan apples-to-apples.\n\n")
	sb.WriteString("| Strategi | Equity (OOS akhir) | Injected (OOS) | Profit (OOS) | Return vs Injected | Expectancy per Trade | Annualized | Max DD | Trades |\n")
	sb.WriteString("|---|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	sb.WriteString(fmt.Sprintf("| **%s (OOS)** | Rp %.0f | Rp %.0f | Rp %.0f | **%.2f%%** | **%.2f%%** | %.2f%% | %.2f%% | %d |\n",
		strings.ToUpper(cfg.Strategy), momentumOOSFinal, totalInjectedMain,
		momentumOOSFinal-totalInjectedMain,
		momentumOOSReturn, oosMetrics.ExpectancyPct, annualized, maxDD, len(oosTrades)))

	for _, b := range baselines {
		_, bOOSEq := splitEquityByOOS(b.EquityCurve, oosFrom)
		final := finalEquityInPeriod(bOOSEq)
		ret := 0.0
		if b.TotalInjected > 0 {
			ret = (final - b.TotalInjected) / b.TotalInjected * 100
		}
		sb.WriteString(fmt.Sprintf("| %s | Rp %.0f | Rp %.0f | Rp %.0f | %.2f%% | %.2f%% | %.2f%% | %.2f%% | %d |\n",
			b.Name, final, b.TotalInjected, final-b.TotalInjected, ret, b.ExpectancyPct, b.AnnualizedPct, b.MaxDrawdownPct, b.TradeCount))
	}
	sb.WriteString("\n")

	sb.WriteString("## Kriteria Lolos (K1–K7)\n\n")
	sb.WriteString("| # | Kriteria | Ambang | Hasil |\n|---|---|---|---|\n")
	sb.WriteString(fmt.Sprintf("| K1 | Total trades | ≥ 100 | %d → %s |\n", len(res.Trades), cell(len(res.Trades) >= 100)))

	k2 := oosMetrics.ExpectancyPct > 0
	sb.WriteString(fmt.Sprintf("| K2 | **Expectancy OOS net** | **> 0%%** | **%.2f%%** → %s |\n", oosMetrics.ExpectancyPct, cell(k2)))

	k3 := oosMetrics.WinRate > 45
	sb.WriteString(fmt.Sprintf("| K3 | Win rate OOS | > 45%% | %.1f%% → %s |\n", oosMetrics.WinRate, cell(k3)))

	r1Pass := false
	r1Exp := 0.0
	if r1Res != nil {
		_, r1OOS := splitTradesByOOS(r1Res.Trades, oosFrom)
		r1Metrics := calcMetrics(r1OOS)
		r1Exp = r1Metrics.ExpectancyPct
		r1Pass = r1Exp > 0
	}
	sb.WriteString(fmt.Sprintf("| K4 | R1 (slippage 0.3%%) | expectancy OOS > 0 | %.2f%% → %s |\n", r1Exp, cell(r1Pass)))

	k5 := maxDD < 50
	sb.WriteString(fmt.Sprintf("| K5 | Max DD | < 50%% | %.2f%% → %s |\n", maxDD, cell(k5)))

	yearsEligible := 0
	yearsPositive := 0
	if len(oosTrades) > 0 {
		byYear := make(map[int][]Trade)
		for _, t := range oosTrades {
			byYear[t.EntryDate.Year()] = append(byYear[t.EntryDate.Year()], t)
		}
		for _, ytr := range byYear {
			if len(ytr) >= 5 {
				yearsEligible++
				if calcMetrics(ytr).ExpectancyPct > 0 {
					yearsPositive++
				}
			}
		}
	}
	k6 := yearsEligible == 0 || yearsPositive*3 >= yearsEligible*2
	sb.WriteString(fmt.Sprintf("| K6 | Konsisten | ≥ 2/3 tahun OOS exp > 0 | %d/%d tahun → %s |\n", yearsPositive, yearsEligible, cell(k6)))

	var rand3, dcaNaif *BaselineResult
	for i := range baselines {
		if baselines[i].Name == "RANDOM3" {
			rand3 = &baselines[i]
		}
		if baselines[i].Name == "DCA_NAIF" {
			dcaNaif = &baselines[i]
		}
	}
	beatRand := rand3 != nil && oosMetrics.ExpectancyPct > rand3.ExpectancyPct
	beatDCA := dcaNaif != nil && res.TotalReturnPct > dcaNaif.ReturnPct
	k7 := beatRand && beatDCA
	sb.WriteString(fmt.Sprintf("| K7 | Beat baseline | > random3 AND > DCA naif | %s |\n", cell(k7)))
	if rand3 != nil {
		sb.WriteString(fmt.Sprintf("  - momentum exp=%.2f%% vs random3 exp=%.2f%% → %s\n", oosMetrics.ExpectancyPct, rand3.ExpectancyPct, cell(beatRand)))
	}
	if dcaNaif != nil {
		sb.WriteString(fmt.Sprintf("  - momentum return=%.2f%% vs DCA naif return=%.2f%% → %s\n", res.TotalReturnPct, dcaNaif.ReturnPct, cell(beatDCA)))
	}
	sb.WriteString("\n")

	sb.WriteString("**⚠️ Catatan penting:** Backtest ini tidak include dividen, ada survivorship bias (watchlist diseleksi setelah tahu performa), dan asumsi fill di harga open. Hasil OOS positif di IDX dengan fee IDX masih harus divalidasi paper trading.\n")

	return sb.String(), nil
}

func cell(b bool) string {
	if b {
		return "✅"
	}
	return "❌"
}

func uniqueMonths(points []EquityPoint) []string {
	seen := make(map[string]bool)
	out := make([]string, 0)
	for _, p := range points {
		m := p.Date.Format("2006-01")
		if !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	return out
}

func finalEquityInPeriod(points []EquityPoint) float64 {
	if len(points) == 0 {
		return 0
	}
	return points[len(points)-1].Equity
}

// SaveWeeklyOutput writes trades.csv, equity.csv, report.md, and compare.md
// for the weekly momentum run.
func SaveWeeklyOutput(cfg Config, res Result, report, compareReport string) error {
	if err := os.MkdirAll(cfg.OutDir, 0755); err != nil {
		return fmt.Errorf("mkdir outdir: %w", err)
	}

	tradesPath := filepath.Join(cfg.OutDir, "trades.csv")
	f, err := os.Create(tradesPath)
	if err != nil {
		return err
	}
	f.WriteString("entry_date,symbol,signal_code,entry,exit_date,exit_reason,lots_sold,lots_total,pnl_pct_net,pnl_rp_net,hold_days,hold_trading_days\n")
	for _, t := range res.Trades {
		f.WriteString(fmt.Sprintf("%s,%s,%s,%.2f,%s,%s,%d,%d,%.4f,%.2f,%d,%d\n",
			t.EntryDate.Format("2006-01-02"), t.Symbol, t.SignalCode,
			t.EntryPrice, t.ExitDate.Format("2006-01-02"), t.ExitReason,
			t.LotsSold, t.LotsTotal, t.PNLPctNet, t.PNLRpNet, t.HoldDays, t.HoldTradingDays))
	}
	f.Close()

	equityPath := filepath.Join(cfg.OutDir, "equity.csv")
	ef, err := os.Create(equityPath)
	if err != nil {
		return err
	}
	ef.WriteString("date,equity,drawdown_pct\n")
	for _, p := range res.EquityCurve {
		ef.WriteString(fmt.Sprintf("%s,%.2f,%.4f\n",
			p.Date.Format("2006-01-02"), p.Equity, p.Drawdown))
	}
	ef.Close()

	if err := os.WriteFile(filepath.Join(cfg.OutDir, "report.md"), []byte(report), 0644); err != nil {
		return err
	}
	if compareReport != "" {
		if err := os.WriteFile(filepath.Join(cfg.OutDir, "compare.md"), []byte(compareReport), 0644); err != nil {
			return err
		}
	}
	return nil
}
