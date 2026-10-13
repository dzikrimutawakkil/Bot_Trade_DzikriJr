package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"learn-go/internal/backtest"
	"learn-go/internal/config"
	"learn-go/internal/models"
)

func main() {
	symbolsStr := flag.String("symbols", "", "Comma-separated symbols (default: watchlist)")
	years := flag.Int("years", 5, "Years of historical data")
	capital := flag.Float64("capital", 1000000, "Starting capital")
	maxPositions := flag.Int("max-positions", 3, "Max simultaneous positions")
	slippage := flag.Float64("slippage-pct", 0.2, "Slippage percentage")
	maxHold := flag.Int("max-hold", 5, "Max hold days (0=disabled)")
	minScore := flag.Float64("min-score", 8, "Minimum signal score")
	noPump := flag.Bool("no-pump-filter", false, "Disable pump/distribution filter")
	noArb := flag.Bool("no-arb-filter", false, "Disable ARB filter")
	ignoreMarket := flag.Bool("ignore-market", false, "Ignore IHSG market filter")
	oosFromStr := flag.String("oos-from", "", "Out-of-sample start date (YYYY-MM-DD)")
	seed := flag.Int64("seed", 42, "Random seed")
	outDir := flag.String("out", "backtest_out", "Output directory")
	refresh := flag.Bool("refresh", false, "Ignore cache, refetch data")
	strategy := flag.String("strategy", "cbow", "Strategy: cbow, breakout20, pullback20, momentum-weekly")
	clMode := flag.String("cl-mode", "structural", "Cut loss mode: structural or fixed")
	compare := flag.Bool("compare", false, "Run all strategies and generate compare report")
	dcaMonthly := flag.Float64("dca-monthly", 0, "Monthly DCA injection in Rp (0=disabled, only for momentum-weekly)")
	clFixedPct := flag.Float64("cl-fixed-pct", 3.0, "CL percentage for momentum-weekly (entry-relative)")
	tpPct := flag.Float64("tp-pct", 5.0, "TP percentage for momentum-weekly (entry-relative)")
	noBaseline := flag.Bool("no-baseline", false, "Skip baseline (random3, dca-naif, buy-hold) for momentum-weekly")
	flag.Parse()

	// Resolve outDir to absolute BEFORE chdir (chdir changes cwd to temp build dir).
	// Cwd at this point is still the project folder where go run was invoked.
	absOutDir, _ := filepath.Abs(*outDir)
	*outDir = absOutDir

	// chdir to binary dir so relative data/ paths resolve correctly at runtime.
	if err := os.Chdir(filepath.Dir(os.Args[0])); err != nil {
		log.Printf("warning: could not chdir: %v", err)
	}

	cfg := backtest.Config{
		Capital:       *capital,
		MaxPositions:  *maxPositions,
		SlippagePct:   *slippage,
		MaxHoldDays:   *maxHold,
		MinScore:      *minScore,
		PumpFilter:    !*noPump,
		ARBFilter:     !*noArb,
		IgnoreMarket:  *ignoreMarket,
		Seed:          *seed,
		OutDir:        *outDir,
		Refresh:       *refresh,
		Years:         *years,
		Strategy:      *strategy,
		CLMode:        *clMode,
		DCAMonthly:    *dcaMonthly,
		CLFixedPct:    *clFixedPct,
		TPPct:         *tpPct,
	}

	symbols := config.Watchlist
	if *symbolsStr != "" {
		symbols = strings.Split(*symbolsStr, ",")
		for i := range symbols {
			symbols[i] = strings.TrimSpace(symbols[i])
		}
	}
	cfg.Symbols = symbols

	var oosFrom time.Time
	if *oosFromStr != "" {
		var err error
		oosFrom, err = time.Parse("2006-01-02", *oosFromStr)
		if err != nil {
			log.Fatalf("invalid oos-from date: %v", err)
		}
		cfg.OOSFrom = oosFrom
	}

	fmt.Printf("Loading data for %d symbols (%d years)...\n", len(symbols), *years)
	allBars := make(map[string][]models.Bar)
	for _, sym := range symbols {
		fmt.Printf("  Fetching %s...\n", sym)
		bars, err := backtest.LoadBars(sym, *years, *refresh)
		if err != nil {
			fmt.Printf("  WARNING: failed to load %s: %v\n", sym, err)
			continue
		}
		allBars[sym] = bars
		fmt.Printf("  %s: %d bars loaded\n", sym, len(bars))
	}

	if cfg.Strategy == "momentum-weekly" {
		if *maxHold == 5 {
			// Default the weekly strategy to 20 trading days per spec
			// (caller did not pass --max-hold explicitly).
			cfg.MaxHoldDays = 20
		}
		fmt.Printf("Running momentum-weekly backtest (DCA monthly=%.0f, max-positions=%d, slippage=%.2f%%, max-hold=%d)...\n",
			cfg.DCAMonthly, cfg.MaxPositions, cfg.SlippagePct, cfg.MaxHoldDays)
		res, err := backtest.RunWeekly(cfg, allBars)
		if err != nil {
			log.Fatalf("weekly run failed: %v", err)
		}
		fmt.Printf("  %d trades executed\n", len(res.Trades))

		var baselines []backtest.BaselineResult
		var r1Res *backtest.Result
		if !*noBaseline {
			fmt.Println("Running baselines (random3, dca-naif, buy-hold)...")
			baselines = append(baselines, backtest.RunBaselineRandom3(cfg, allBars, cfg.MaxPositions))
			baselines = append(baselines, backtest.RunBaselineDCA(cfg, allBars))
			baselines = append(baselines, backtest.RunBaselineBuyHold(cfg, allBars))

			fmt.Println("Running R1 robustness (slippage 0.3%)...")
			cfgR1 := cfg
			cfgR1.SlippagePct = 0.3
			r1, err := backtest.RunWeekly(cfgR1, allBars)
			if err == nil {
				r1Res = &r1
			} else {
				log.Printf("R1 run failed: %v", err)
			}
		}

		report, err := backtest.GenerateWeeklyReport(res, cfg)
		if err != nil {
			log.Fatalf("generate weekly report failed: %v", err)
		}
		var compareReport string
		if len(baselines) > 0 {
			compareReport, err = backtest.GenerateCompareWeeklyReport(res, cfg, baselines, r1Res)
			if err != nil {
				log.Fatalf("generate compare report failed: %v", err)
			}
		}
		if err := backtest.SaveWeeklyOutput(cfg, res, report, compareReport); err != nil {
			log.Fatalf("save weekly output failed: %v", err)
		}

		absPath, _ := filepath.Abs(*outDir)
		fmt.Printf("\nDone! Output saved to %s/\n", absPath)
		fmt.Printf("  - %s/trades.csv\n", absPath)
		fmt.Printf("  - %s/equity.csv\n", absPath)
		fmt.Printf("  - %s/report.md\n", absPath)
		if compareReport != "" {
			fmt.Printf("  - %s/compare.md\n", absPath)
		}
		return
	}

	fmt.Printf("  Fetching IHSG (^JKSE)...\n")
	ihsgBars, err := backtest.LoadBars("^JKSE", *years, *refresh)
	if err != nil {
		log.Fatalf("failed to load IHSG: %v", err)
	}
	fmt.Printf("  IHSG: %d bars loaded\n", len(ihsgBars))

	if *compare {
		fmt.Println("Running compare mode...")
		entries, err := backtest.RunCompare(cfg, allBars, ihsgBars)
		if err != nil {
			log.Fatalf("compare run failed: %v", err)
		}

		compareReport := backtest.GenerateCompareReport(entries, cfg)
		if err := backtest.SaveCompareOutput(cfg, entries, compareReport); err != nil {
			log.Fatalf("save compare output failed: %v", err)
		}

		absPath, _ := filepath.Abs(*outDir)
		fmt.Printf("\nDone! Compare report saved to %s/\n", absPath)
		fmt.Printf("  - %s/compare.md\n", absPath)
		for _, e := range entries {
			fmt.Printf("  - %s/%s/report.md\n", absPath, e.Name)
		}
		return
	}

	oosIdx := len(symbols) * 70 / 100
	labels := []string{"In-Sample", "Out-of-Sample"}

	fmt.Printf("Running backtest (strategy=%s, cl-mode=%s)...\n", cfg.Strategy, cfg.CLMode)
	res, err := backtest.Run(cfg, allBars, ihsgBars)
	if err != nil {
		log.Fatalf("backtest run failed: %v", err)
	}

	fmt.Printf("  %d trades found\n", len(res.Trades))

	fmt.Println("Running ablation...")
	ablResults, err := backtest.RunAblation(cfg, allBars, ihsgBars)
	if err != nil {
		log.Printf("ablation failed: %v", err)
	}

	report, err := backtest.GenerateReport(res, cfg, oosIdx, labels)
	if err != nil {
		log.Fatalf("generate report failed: %v", err)
	}

	report += "\n" + backtest.CompareAblation(res, ablResults)
	report += "\n## Kesimpulan\n" + backtest.Conclusion(res)

	if err := backtest.SaveOutput(cfg, res.Trades, report); err != nil {
		log.Fatalf("save output failed: %v", err)
	}

	absPath, _ := filepath.Abs(*outDir)
	fmt.Printf("\nDone! Output saved to %s/\n", absPath)
	fmt.Printf("  - %s/trades.csv\n", absPath)
	fmt.Printf("  - %s/report.md\n", absPath)
}
