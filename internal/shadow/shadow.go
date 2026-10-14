package shadow

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"learn-go/internal/models"
	"learn-go/internal/rules"
)

const (
	FileName         = "signals_log.csv"
	IHSG             = "^JKSE"
	RecordHour       = 16
	RecordMinute     = 30
	FillHour         = 16
	FillMinute       = 50
	MaxAICallsPerDay = 25
	AICallDelaySec   = 15
	MinScore         = 7.0
	MinBarsForSignal = 60
)

var (
	mu      sync.Mutex
	dataDir string
)

func init() {
	wd, _ := os.Getwd()
	dataDir = filepath.Join(wd, "data")
}

// SetDataDir overrides the data directory (useful for testing).
func SetDataDir(dir string) {
	mu.Lock()
	dataDir = dir
	mu.Unlock()
}

// filePath returns the absolute path to signals_log.csv.
func filePath() string {
	mu.Lock()
	defer mu.Unlock()
	_ = os.MkdirAll(dataDir, 0755)
	return filepath.Join(dataDir, FileName)
}

// Row represents a single signal log entry.
type Row struct {
	Date         string
	Symbol       string
	Score        float64
	Code         string
	MarketState  string
	Close        float64
	SignalLow    float64
	PlanCL       float64
	PlanTP1      float64
	AIReco       string
	AISentiment  float64
	AIConfidence float64
	Model        string
	Ret1D        float64
	Ret3D        float64
	Ret5D        float64
	MaxUp5D      float64
	MaxDD5D      float64
	Filled       bool
}

// LoadRows reads all rows from the CSV file. Returns nil on empty/missing file.
func LoadRows() ([]Row, error) {
	path := filePath()
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) <= 1 {
		return nil, nil
	}

	rows := make([]Row, 0, len(records)-1)
	for i, rec := range records[1:] {
		row, err := parseRecord(rec, i+2)
		if err != nil {
			continue
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func parseRecord(rec []string, lineNum int) (Row, error) {
	if len(rec) < 17 {
		return Row{}, fmt.Errorf("line %d: expected 17 fields, got %d", lineNum, len(rec))
	}
	row := Row{
		Date:        strings.TrimSpace(rec[0]),
		Symbol:      strings.TrimSpace(rec[1]),
		MarketState: strings.TrimSpace(rec[4]),
		Close:       toFloat(rec[5]),
		SignalLow:   toFloat(rec[6]),
		PlanCL:      toFloat(rec[7]),
		PlanTP1:     toFloat(rec[8]),
		AIReco:      strings.TrimSpace(rec[9]),
		Model:       strings.TrimSpace(rec[12]),
	}
	row.Score, _ = strconv.ParseFloat(strings.TrimSpace(rec[2]), 64)
	row.Code = strings.TrimSpace(rec[3])
	row.AISentiment, _ = strconv.ParseFloat(strings.TrimSpace(rec[10]), 64)
	row.AIConfidence, _ = strconv.ParseFloat(strings.TrimSpace(rec[11]), 64)
	row.Ret1D, _ = strconv.ParseFloat(strings.TrimSpace(rec[13]), 64)
	row.Ret3D, _ = strconv.ParseFloat(strings.TrimSpace(rec[14]), 64)
	row.Ret5D, _ = strconv.ParseFloat(strings.TrimSpace(rec[15]), 64)
	row.MaxUp5D, _ = strconv.ParseFloat(strings.TrimSpace(rec[16]), 64)
	if len(rec) > 17 {
		row.MaxDD5D, _ = strconv.ParseFloat(strings.TrimSpace(rec[17]), 64)
		row.Filled = strings.TrimSpace(rec[18]) == "true"
	}
	return row, nil
}

func toFloat(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}

// SaveRows writes all rows atomically (tmp + rename). Mutex protects concurrent access.
func SaveRows(rows []Row) error {
	path := filePath()
	tmp := path + ".tmp"

	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	w := csv.NewWriter(f)

	header := []string{
		"date", "symbol", "score", "code", "market_state", "close", "signal_low",
		"plan_cl", "plan_tp1", "ai_reco", "ai_sentiment", "ai_confidence", "model",
		"ret_1d", "ret_3d", "ret_5d", "max_up_5d", "max_dd_5d", "filled",
	}
	if err := w.Write(header); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}

	for _, r := range rows {
		rec := []string{
			r.Date, r.Symbol,
			fmt.Sprintf("%.1f", r.Score), r.Code, r.MarketState,
			fmt.Sprintf("%.4f", r.Close),
			fmt.Sprintf("%.4f", r.SignalLow),
			fmt.Sprintf("%.4f", r.PlanCL),
			fmt.Sprintf("%.4f", r.PlanTP1),
			r.AIReco,
			fmt.Sprintf("%.1f", r.AISentiment),
			fmt.Sprintf("%.1f", r.AIConfidence),
			r.Model,
			fmt.Sprintf("%.4f", r.Ret1D),
			fmt.Sprintf("%.4f", r.Ret3D),
			fmt.Sprintf("%.4f", r.Ret5D),
			fmt.Sprintf("%.4f", r.MaxUp5D),
			fmt.Sprintf("%.4f", r.MaxDD5D),
			fmt.Sprintf("%t", r.Filled),
		}
		if err := w.Write(rec); err != nil {
			f.Close()
			os.Remove(tmp)
			return err
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	f.Close()

	mu.Lock()
	defer mu.Unlock()
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	return nil
}

// HasRow returns true if a row for (date, symbol) already exists.
func HasRow(date, symbol string, rows []Row) bool {
	for _, r := range rows {
		if r.Date == date && r.Symbol == symbol {
			return true
		}
	}
	return false
}

// AddOrUpdateRow inserts or updates a row keyed by (date, symbol).
func AddOrUpdateRow(rows []Row, newRow Row) []Row {
	for i, r := range rows {
		if r.Date == newRow.Date && r.Symbol == newRow.Symbol {
			rows[i] = newRow
			return rows
		}
	}
	return append(rows, newRow)
}

// NextTradingDayOffset returns how many trading days (skipping weekends) from startDate
// to reach targetOffset-th trading day after startDate (inclusive of startDate means offset=0).
// E.g. offset=5 means 5 full trading days after (so check on the 5th day from now).
func NextTradingDayOffset(startDate time.Time, offset int) time.Time {
	d := startDate
	skipped := 0
	for {
		d = d.AddDate(0, 0, 1)
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue
		}
		skipped++
		if skipped >= offset {
			return d
		}
	}
}

// TradingDaysBetween returns the number of trading days (Mon-Fri) from start (exclusive) to end (inclusive).
func TradingDaysBetween(start, end time.Time) int {
	if end.Before(start) || end.Equal(start) {
		return 0
	}
	count := 0
	d := start
	for {
		d = d.AddDate(0, 0, 1)
		if d.After(end) {
			break
		}
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			count++
		}
	}
	return count
}

// FillRow fills outcome columns for a row given future bars (next 5 trading days after signal).
func FillRow(row Row, futureBars []models.Bar) Row {
	if len(futureBars) < 5 {
		return row
	}
	entry := row.Close

	bar0 := futureBars[0]
	ret1d := (bar0.Close - entry) / entry * 100
	row.Ret1D = math.Round(ret1d*100) / 100

	bar2 := futureBars[2]
	ret3d := (bar2.Close - entry) / entry * 100
	row.Ret3D = math.Round(ret3d*100) / 100

	bar4 := futureBars[4]
	ret5d := (bar4.Close - entry) / entry * 100
	row.Ret5D = math.Round(ret5d*100) / 100

	maxHigh := entry
	minLow := entry
	for i := 0; i < 5 && i < len(futureBars); i++ {
		if futureBars[i].High > maxHigh {
			maxHigh = futureBars[i].High
		}
		if futureBars[i].Low < minLow {
			minLow = futureBars[i].Low
		}
	}

	row.MaxUp5D = math.Round(((maxHigh-entry)/entry*100)*100) / 100
	row.MaxDD5D = math.Round(((minLow-entry)/entry*100)*100) / 100
	row.Filled = true
	return row
}

// RoundTripFeePct returns the round-trip fee percentage (BuyFee + SellFee) as a percentage number (e.g. 0.40 for 0.40%).
func RoundTripFeePct() float64 {
	return (rules.Default.BuyFee + rules.Default.SellFee) * 100
}

// NetReturn applies round-trip fee to a gross return percentage.
func NetReturn(grossPct float64) float64 {
	return math.Round((grossPct-RoundTripFeePct())*100) / 100
}
