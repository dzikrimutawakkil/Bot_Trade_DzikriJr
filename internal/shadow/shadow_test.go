package shadow

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"learn-go/internal/models"
)

func TestFillRow(t *testing.T) {
	// Synthetic bars:
	// entry = 1000
	// day+1 close=1010 → ret_1d = (1010-1000)/1000*100 = 1.0
	// day+3 close=1030 → ret_3d = (1030-1000)/1000*100 = 3.0
	// day+5 close=1020 → ret_5d = (1020-1000)/1000*100 = 2.0
	// max High = 1050 → max_up_5d = (1050-1000)/1000*100 = 5.0
	// min Low = 985 → max_dd_5d = (985-1000)/1000*100 = -1.5
	futureBars := []models.Bar{
		{Close: 1010, High: 1020, Low: 1000},
		{Close: 1020, High: 1030, Low: 1010},
		{Close: 1030, High: 1050, Low: 1020},
		{Close: 1015, High: 1035, Low: 1005},
		{Close: 1020, High: 1030, Low: 985},
	}

	row := Row{Close: 1000}
	result := FillRow(row, futureBars)

	if result.Ret1D != 1.0 {
		t.Errorf("Ret1D = %.2f, want 1.0", result.Ret1D)
	}
	if result.Ret3D != 3.0 {
		t.Errorf("Ret3D = %.2f, want 3.0", result.Ret3D)
	}
	if result.Ret5D != 2.0 {
		t.Errorf("Ret5D = %.2f, want 2.0", result.Ret5D)
	}
	if result.MaxUp5D != 5.0 {
		t.Errorf("MaxUp5D = %.2f, want 5.0", result.MaxUp5D)
	}
	if result.MaxDD5D != -1.5 {
		t.Errorf("MaxDD5D = %.2f, want -1.5", result.MaxDD5D)
	}
	if !result.Filled {
		t.Error("Filled should be true")
	}
}

func TestFillRowLessThan5Bars(t *testing.T) {
	// With < 5 bars, FillRow returns row unchanged (Filled stays false)
	futureBars := []models.Bar{
		{Close: 1010, High: 1020, Low: 1000},
	}
	row := Row{Close: 1000}
	result := FillRow(row, futureBars)

	if result.Filled {
		t.Error("Filled should be false when less than 5 bars")
	}
	if result.Ret1D != 0 {
		t.Errorf("Ret1D = %.2f, want 0 (not filled)", result.Ret1D)
	}
}

func TestHasRow(t *testing.T) {
	rows := []Row{
		{Date: "2026-10-01", Symbol: "BBRI"},
		{Date: "2026-10-01", Symbol: "BMRI"},
		{Date: "2026-10-02", Symbol: "BBRI"},
	}

	if !HasRow("2026-10-01", "BBRI", rows) {
		t.Error("HasRow should return true for existing (date, symbol)")
	}
	if HasRow("2026-10-03", "BBRI", rows) {
		t.Error("HasRow should return false for non-existing date")
	}
	if HasRow("2026-10-01", "UNTR", rows) {
		t.Error("HasRow should return false for different symbol")
	}
}

func TestAddOrUpdateRow(t *testing.T) {
	rows := []Row{
		{Date: "2026-10-01", Symbol: "BBRI", Score: 8},
	}

	// Updating existing (same date+symbol) should not increase len
	newRow := Row{Date: "2026-10-01", Symbol: "BBRI", Score: 10}
	result := AddOrUpdateRow(rows, newRow)
	if len(result) != 1 {
		t.Errorf("len = %d, want 1 (update, not append)", len(result))
	}
	if result[0].Score != 10 {
		t.Errorf("Score = %.0f, want 10", result[0].Score)
	}

	// Adding new symbol should increase len
	newRow.Symbol = "BMRI"
	result = AddOrUpdateRow(rows, newRow)
	if len(result) != 2 {
		t.Errorf("len = %d, want 2 (new symbol appends)", len(result))
	}
}

func TestTradingDaysBetween(t *testing.T) {
	// Mon Oct 18, 2026 → Tue Oct 19: 1 weekday
	mon := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC) // Sunday (weekday=0)
	// But we need Monday: let me check: Oct 18, 2026 is Sunday
	// Let me use a known Monday
	mon = time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC) // Monday
	tue := mon.AddDate(0, 0, 1)                         // Tuesday
	fri := mon.AddDate(0, 0, 4)                         // Friday
	nextMon := mon.AddDate(0, 0, 7)                     // next Monday

	if n := TradingDaysBetween(mon, tue); n != 1 {
		t.Errorf("Mon->Tue = %d, want 1", n)
	}
	// Mon->Fri: increment past Mon, count Tue(1),Wed(2),Thu(3),Fri(4) = 4
	if n := TradingDaysBetween(mon, fri); n != 4 {
		t.Errorf("Mon->Fri = %d, want 4 (Tue+Wed+Thu+Fri)", n)
	}
	// Fri->nextMon: Sat(1),Sun(2) → next Mon = end → stops at Sat → 1
	if n := TradingDaysBetween(fri, nextMon); n != 1 {
		t.Errorf("Fri->nextMon = %d, want 1", n)
	}
}

func TestNetReturn(t *testing.T) {
	// RoundTripFee = 0.0015 + 0.0025 = 0.0040
	// NetReturn(gross) = gross - 0.40
	// Gross 3.0% → net 2.60%
	if r := NetReturn(3.0); r != 2.60 {
		t.Errorf("NetReturn(3.0) = %.2f, want 2.60", r)
	}
	// Gross 0.5% → net 0.10%
	if r := NetReturn(0.5); r != 0.10 {
		t.Errorf("NetReturn(0.5) = %.2f, want 0.10", r)
	}
	// Gross -2.0% → net -2.40%
	if r := NetReturn(-2.0); r != -2.40 {
		t.Errorf("NetReturn(-2.0) = %.2f, want -2.40", r)
	}
}

func TestSaveAndLoadRows(t *testing.T) {
	tmpDir := t.TempDir()
	SetDataDir(tmpDir)

	rows := []Row{
		{
			Date:         "2026-10-13",
			Symbol:       "BBRI",
			Score:        10,
			Code:         "HAMMER",
			MarketState:  "UPTREND",
			Close:        4500.0,
			SignalLow:    4400.0,
			PlanCL:       4356.0,
			PlanTP1:      4680.0,
			AIReco:       "BELI",
			AISentiment:  8,
			AIConfidence: 85,
			Model:        "gemini-flash-latest",
			Ret1D:        0,
			Ret3D:        0,
			Ret5D:        0,
			MaxUp5D:      0,
			MaxDD5D:      0,
			Filled:       false,
		},
	}

	err := SaveRows(rows)
	if err != nil {
		t.Fatalf("SaveRows: %v", err)
	}

	loaded, err := LoadRows()
	if err != nil {
		t.Fatalf("LoadRows: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("len(loaded) = %d, want 1", len(loaded))
	}
	if loaded[0].Symbol != "BBRI" {
		t.Errorf("Symbol = %s, want BBRI", loaded[0].Symbol)
	}
	if loaded[0].AIReco != "BELI" {
		t.Errorf("AIReco = %s, want BELI", loaded[0].AIReco)
	}
	if loaded[0].Filled {
		t.Error("Filled should be false")
	}
}

func TestUniqueKeyNoDuplicate(t *testing.T) {
	// Verify HasRow and AddOrUpdateRow work correctly for duplicate prevention
	rows := []Row{
		{Date: "2026-10-13", Symbol: "BBRI", Score: 8},
	}

	if HasRow("2026-10-13", "BBRI", rows) {
		// AddOrUpdateRow should update (not append) when key matches
		result := AddOrUpdateRow(rows, Row{Date: "2026-10-13", Symbol: "BBRI", Score: 12})
		if len(result) != 1 {
			t.Errorf("Duplicate added: len = %d, want 1", len(result))
		}
		if result[0].Score != 12 {
			t.Errorf("Score not updated: %.0f, want 12", result[0].Score)
		}
	}
}

func TestFilePathCreatesDir(t *testing.T) {
	tmpDir := filepath.Join(t.TempDir(), "subdir", "nested")
	SetDataDir(tmpDir)
	p := filePath()
	if _, err := os.Stat(filepath.Dir(p)); os.IsNotExist(err) {
		t.Error("filePath should create parent directories")
	}
}
