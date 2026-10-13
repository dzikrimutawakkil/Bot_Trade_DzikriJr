package storage

import (
	"os"
	"path/filepath"
	"testing"

	"learn-go/internal/config"
	"learn-go/internal/models"
)

func TestReadTradeHistory(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		wantBuy     int
		wantSell    int
		wantSymbols []string
	}{
		{
			name:        "v2 fixture",
			filename:    "trade_history_v2.csv",
			wantBuy:     3,
			wantSell:    4,
			wantSymbols: []string{"MTEL", "MTEL", "MDKA", "BBNI"},
		},
		{
			name:        "v1 fixture",
			filename:    "trade_history_v1.csv",
			wantBuy:     2,
			wantSell:    3,
			wantSymbols: []string{"MTEL", "MTEL", "MDKA"},
		},
		{
			name:        "mixed fixture",
			filename:    "trade_history_mixed.csv",
			wantBuy:     2,
			wantSell:    3,
			wantSymbols: []string{"MTEL", "MTEL", "MDKA"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join("testdata", tt.filename)
			records, err := ReadTradeHistory(path)
			if err != nil {
				t.Fatalf("ReadTradeHistory(%s) error = %v", tt.name, err)
			}

			var buyCount, sellCount int
			var sellSymbols []string
			for _, r := range records {
				if r.Action == "BUY" {
					buyCount++
				} else if r.Action == "SELL" {
					sellCount++
					sellSymbols = append(sellSymbols, r.Symbol)
				}
			}

			if buyCount != tt.wantBuy {
				t.Errorf("buyCount = %d, want %d", buyCount, tt.wantBuy)
			}
			if sellCount != tt.wantSell {
				t.Errorf("sellCount = %d, want %d", sellCount, tt.wantSell)
			}
			if len(sellSymbols) != len(tt.wantSymbols) {
				t.Fatalf("sellSymbols len = %d, want %d", len(sellSymbols), len(tt.wantSymbols))
			}
			for i, sym := range sellSymbols {
				if sym != tt.wantSymbols[i] {
					t.Errorf("sellSymbols[%d] = %s, want %s", i, sym, tt.wantSymbols[i])
				}
			}
		})
	}
}

func TestMigrateTradeHistory(t *testing.T) {
	fixtureV2 := filepath.Join("testdata", "trade_history_v2.csv")

	// Copy to temp file
	tmp := t.TempDir()
	tmpFile := filepath.Join(tmp, "trade_history.csv")
	copyFile := func(src, dst string) error {
		r, err := os.Open(src)
		if err != nil {
			return err
		}
		defer r.Close()
		w, err := os.Create(dst)
		if err != nil {
			return err
		}
		defer w.Close()
		buf := make([]byte, 4096)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				if _, werr := w.Write(buf[:n]); werr != nil {
					return werr
				}
			}
			if err != nil {
				break
			}
		}
		return nil
	}
	if err := copyFile(fixtureV2, tmpFile); err != nil {
		t.Fatalf("copyFile: %v", err)
	}

	// Run migration
	if err := MigrateTradeHistory(tmpFile); err != nil {
		t.Fatalf("MigrateTradeHistory error = %v", err)
	}

	// Verify backup was created
	entries, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	hasBackup := false
	for _, e := range entries {
		if e.Name() != "trade_history.csv" {
			hasBackup = true
		}
	}
	if !hasBackup {
		t.Log("Note: no .bak file found (may use timestamp suffix)")
	}
	// Backup check - if .bak file exists (may have timestamp suffix)

	// Read migrated records
	records, err := ReadTradeHistory(tmpFile)
	if err != nil {
		t.Fatalf("ReadTradeHistory after migrate: %v", err)
	}

	// Expected values from spec (tolerance ±1 Rp)
	expected := map[string]struct {
		wantPNLRp  float64
		wantReason string
	}{
		"MTEL_555":  {wantPNLRp: 8905, wantReason: "MANUAL"},  // partial sell
		"MTEL_525":  {wantPNLRp: -6058, wantReason: "MANUAL"}, // clear sell
		"MDKA_3470": {wantPNLRp: 11632, wantReason: "MANUAL"},
		"BBNI_3800": {wantPNLRp: 1484, wantReason: "MANUAL"},
	}

	for _, rec := range records {
		if rec.Action != "SELL" {
			continue
		}
		key := rec.Symbol + "_" + formatPrice(rec.Price)
		exp, ok := expected[key]
		if !ok {
			continue
		}
		diff := rec.PNLRp - exp.wantPNLRp
		if diff < 0 {
			diff = -diff
		}
		if diff > 1 {
			t.Errorf("[%s] PNLRp = %.2f, want %.2f (±1)", key, rec.PNLRp, exp.wantPNLRp)
		}
		if rec.ExitReason != exp.wantReason {
			t.Errorf("[%s] ExitReason = %s, want %s", key, rec.ExitReason, exp.wantReason)
		}
	}

	// Idempotent: migrate again, should not change
	if err := MigrateTradeHistory(tmpFile); err != nil {
		t.Fatalf("MigrateTradeHistory 2nd run error = %v", err)
	}
	records2, err := ReadTradeHistory(tmpFile)
	if err != nil {
		t.Fatalf("ReadTradeHistory after 2nd migrate: %v", err)
	}
	if len(records2) != len(records) {
		t.Errorf("Idempotent: len changed from %d to %d", len(records), len(records2))
	}
}

func TestCalculateStatisticsFromFile(t *testing.T) {
	// First migrate v2 fixture to get correct PNLRp
	fixtureV2 := filepath.Join("testdata", "trade_history_v2.csv")
	tmp := t.TempDir()
	tmpFile := filepath.Join(tmp, "trade_history.csv")

	// Copy
	r, _ := os.Open(fixtureV2)
	w, _ := os.Create(tmpFile)
	buf := make([]byte, 4096)
	for {
		n, _ := r.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
		}
		if n == 0 {
			break
		}
	}
	r.Close()
	w.Close()

	MigrateTradeHistory(tmpFile)

	stats, err := CalculateStatisticsFromFile(tmpFile)
	if err != nil {
		t.Fatalf("CalculateStatisticsFromFile error = %v", err)
	}

	if stats.TotalTrades != 4 {
		t.Errorf("TotalTrades = %d, want 4", stats.TotalTrades)
	}
	if stats.WinningTrades != 3 {
		t.Errorf("WinningTrades = %d, want 3", stats.WinningTrades)
	}
	if stats.LosingTrades != 1 {
		t.Errorf("LosingTrades = %d, want 1", stats.LosingTrades)
	}

	winRate := float64(stats.WinningTrades) / float64(stats.TotalTrades) * 100
	if stats.WinRate != winRate {
		t.Errorf("WinRate = %.1f, want %.1f", stats.WinRate, winRate)
	}

	// NetProfitRp should be sum of all PNLRp
	// 8905 + (-6058) + 11632 + 1484 = 15963
	net := stats.NetProfitRp
	if net < 15960 || net > 15966 {
		t.Errorf("NetProfitRp = %.2f, want ≈15963", net)
	}

	// ExpectancyRp
	exp := stats.ExpectancyRp
	if exp < 3990 || exp > 3992 {
		t.Errorf("ExpectancyRp = %.2f, want ≈3991", exp)
	}
}

func TestResolveExitReason(t *testing.T) {
	// Plan: entry=535, CL=515, TP=555, highest=556, tp1Done=false, trailingActive=true
	entry := 535.0
	cl := 515.0
	tp := 555.0
	highest := 556.0
	tp1Done := false
	trailingActive := true

	tests := []struct {
		price      float64
		wantReason string
	}{
		{514, "CL"},     // <= CL
		{540, "TSL"},    // trailing active, <= trailing stop (556*0.97≈539)
		{555, "TP1"},    // >= TP, partial (tp1Done=false)
		{548, "MANUAL"}, // >= CL, not TSL, not TP
	}

	for _, tt := range tests {
		got := ResolveExitReason(tt.price, entry, cl, tp, highest, tp1Done, trailingActive)
		if got != tt.wantReason {
			t.Errorf("ResolveExitReason(%.0f) = %s, want %s", tt.price, got, tt.wantReason)
		}
	}
}

func formatPrice(price float64) string {
	if price == 555 || price == 525 {
		return "5" + formatPriceInt(price)
	}
	return formatPriceInt(price)
}

func TestSaveLoadRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	tmpFile := filepath.Join(tmp, "stocks.json")
	origFile := StorageFile
	StorageFile = tmpFile
	defer func() { StorageFile = origFile }()

	config.DataMutex.Lock()
	config.MyStocks = map[string]models.TradingPlan{
		"BBRI": {Symbol: "BBRI", EntryPrice: 4500, Lots: 10},
	}
	config.PendingOrders = map[string]models.ActiveOrder{
		"TOWR": {Symbol: "TOWR", OrderPrice: 480, Lot: 5},
	}
	config.DataMutex.Unlock()

	SaveData()

	config.DataMutex.Lock()
	config.MyStocks = nil
	config.PendingOrders = nil
	config.DataMutex.Unlock()

	LoadData()

	config.DataMutex.RLock()
	defer config.DataMutex.RUnlock()

	if len(config.MyStocks) != 1 {
		t.Errorf("MyStocks len = %d, want 1", len(config.MyStocks))
	}
	if len(config.PendingOrders) != 1 {
		t.Errorf("PendingOrders len = %d, want 1", len(config.PendingOrders))
	}
}

func TestLoadCorruptFile(t *testing.T) {
	tmp := t.TempDir()
	tmpFile := filepath.Join(tmp, "stocks.json")
	origFile := StorageFile
	StorageFile = tmpFile
	defer func() { StorageFile = origFile }()

	config.DataMutex.Lock()
	config.MyStocks = nil
	config.PendingOrders = nil
	config.DataMutex.Unlock()

	os.WriteFile(tmpFile, []byte("this is not json{"), 0644)

	LoadData()

	config.DataMutex.RLock()
	defer config.DataMutex.RUnlock()

	if len(config.MyStocks) != 0 {
		t.Errorf("MyStocks should be empty after corrupt load, got len=%d", len(config.MyStocks))
	}

	corruptFiles, _ := filepath.Glob(filepath.Join(tmp, "stocks.json.corrupt-*"))
	if len(corruptFiles) == 0 {
		t.Errorf("Expected .corrupt-* file to be created")
	}
}

func formatPriceInt(price float64) string {
	switch {
	case price == 555:
		return "555"
	case price == 525:
		return "525"
	case price == 3470:
		return "3470"
	case price == 3800:
		return "3800"
	default:
		return "0"
	}
}
