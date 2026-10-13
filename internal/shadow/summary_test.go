package shadow

import (
	"testing"
)

func makeRow(reco, code string, ret3d float64) Row {
	return Row{
		Date:    "2026-10-01",
		Symbol:  "BBRI",
		Score:   10,
		Code:    code,
		Ret3D:   ret3d,
		MaxDD5D: -1.0,
		Filled:  true,
	}
}

func TestSummarizeEmpty(t *testing.T) {
	sum := Summarize([]Row{})
	if sum.FilledRows != 0 {
		t.Errorf("FilledRows = %d, want 0", sum.FilledRows)
	}
}

func TestSummarizeWinRate(t *testing.T) {
	// 5 rows: ret3d = 3, 4, -2, 5, -1  (net after 0.40% fee: 2.6, 3.6, -2.4, 4.6, -1.4)
	rows := []Row{
		makeRow("BELI", "HAMMER", 3.0),
		makeRow("BELI", "HAMMER", 4.0),
		makeRow("BELI", "HAMMER", -2.0),
		makeRow("BELI", "HAMMER", 5.0),
		makeRow("BELI", "HAMMER", -1.0),
	}

	sum := Summarize(rows)

	if sum.FilledRows != 5 {
		t.Errorf("FilledRows = %d, want 5", sum.FilledRows)
	}
	if len(sum.Groups) != 1 {
		t.Fatalf("len(Groups) = %d, want 1", len(sum.Groups))
	}
	g := sum.Groups[0]
	if g.N != 5 {
		t.Errorf("N = %d, want 5", g.N)
	}
	if g.WinRate3D != 60.0 {
		t.Errorf("WinRate3D = %.1f%%, want 60.0%%", g.WinRate3D)
	}

	avgRet3d := (3.0 + 4.0 + -2.0 + 5.0 + -1.0) / 5.0
	if g.AvgRet3D != avgRet3d {
		t.Errorf("AvgRet3D = %.2f, want %.2f", g.AvgRet3D, avgRet3d)
	}
}

func TestSummarizeMultipleGroups(t *testing.T) {
	rows := []Row{
		{Date: "2026-10-01", Symbol: "BBRI", Code: "HAMMER", AIReco: "BELI", Ret3D: 3.0, MaxDD5D: -1.0, Filled: true},
		{Date: "2026-10-01", Symbol: "BMRI", Code: "HAMMER", AIReco: "BELI", Ret3D: 5.0, MaxDD5D: -2.0, Filled: true},
		{Date: "2026-10-01", Symbol: "ADRO", Code: "GREEN_BOUNCE", AIReco: "HINDARI", Ret3D: -3.0, MaxDD5D: -5.0, Filled: true},
	}

	sum := Summarize(rows)

	if sum.FilledRows != 3 {
		t.Errorf("FilledRows = %d, want 3", sum.FilledRows)
	}
	if len(sum.Groups) != 2 {
		t.Errorf("Groups = %d, want 2", len(sum.Groups))
	}
}

func TestSummarizeSmallSample(t *testing.T) {
	rows := []Row{
		makeRow("BELI", "HAMMER", 3.0),
		makeRow("BELI", "HAMMER", 2.0),
	}

	sum := Summarize(rows)
	if !sum.SmallSampleWarning {
		t.Error("SmallSampleWarning should be true when n < 30")
	}

	// Add 30 rows
	for i := 0; i < 30; i++ {
		rows = append(rows, makeRow("BELI", "HAMMER", 2.0))
	}
	sum2 := Summarize(rows)
	if sum2.SmallSampleWarning {
		t.Error("SmallSampleWarning should be false when n >= 30")
	}
}

func TestSummarizeCompareBELIvsHindari(t *testing.T) {
	// BELI on HAMMER: ret3d 3.0 → net 2.6
	// HINDARI on HAMMER: ret3d -2.0 → net -2.4
	// diff = 2.6 - (-2.4) = 5.0
	rows := []Row{
		{Date: "2026-10-01", Symbol: "BBRI", Code: "HAMMER", AIReco: "BELI", Ret3D: 3.0, Filled: true},
		{Date: "2026-10-01", Symbol: "BMRI", Code: "HAMMER", AIReco: "HINDARI", Ret3D: -2.0, Filled: true},
	}

	sum := Summarize(rows)

	// avg BELI net = 3.0 - 0.40 = 2.6; avg HINDARI net = -2.0 - 0.40 = -2.4
	// diff = 2.6 - (-2.4) = 5.0
	if sum.CompareBELIvsHindari != 5.0 {
		t.Errorf("CompareBELIvsHindari = %.2f, want 5.00", sum.CompareBELIvsHindari)
	}
}

func TestSummaryFormat(t *testing.T) {
	rows := []Row{
		{Date: "2026-10-01", Symbol: "BBRI", Code: "HAMMER", AIReco: "BELI", Ret3D: 3.0, MaxDD5D: -1.0, Filled: true},
	}
	sum := Summarize(rows)
	output := sum.Format()

	if sum.FilledRows != 1 {
		t.Errorf("FilledRows = %d", sum.FilledRows)
	}
	if sum.DateRange == "" {
		t.Error("DateRange should not be empty")
	}
	if output == "" {
		t.Error("Format() should return non-empty string")
	}
}
