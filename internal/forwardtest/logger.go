package forwardtest

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"learn-go/internal/models"
)

const ftCSVFile = "forward_test_history.csv"

var csvMutex sync.Mutex

// LogTrade appends a closed trade to forward_test_history.csv.
func LogTrade(trade models.FTClosedTrade) {
	csvMutex.Lock()
	defer csvMutex.Unlock()

	f, err := openOrCreateCSV()
	if err != nil {
		log.Printf("[FT] Gagal buka CSV logger: %v", err)
		return
	}
	defer f.Close()

	rec := []string{
		time.Now().Format("2006-01-02 15:04:05"),
		trade.Symbol,
		"EXIT",
		strconv.FormatFloat(trade.EntryPrice, 'f', 0, 64),
		strconv.FormatFloat(trade.ExitPrice, 'f', 0, 64),
		strconv.Itoa(trade.Lots),
		trade.EntryDate,
		trade.ExitDate,
		strconv.Itoa(trade.HoldDays),
		strconv.FormatFloat(trade.PNLPercent, 'f', 2, 64),
		strconv.FormatFloat(trade.PNLRp, 'f', 0, 64),
		trade.ExitReason,
	}
	if err := csv.NewWriter(f).Write(rec); err != nil {
		log.Printf("[FT] Gagal write CSV: %v", err)
	}
}

// LogEntry logs an opening entry.
func LogEntry(symbol string, price float64, lots int, date string) {
	csvMutex.Lock()
	defer csvMutex.Unlock()

	f, err := openOrCreateCSV()
	if err != nil {
		log.Printf("[FT] Gagal buka CSV logger: %v", err)
		return
	}
	defer f.Close()

	rec := []string{
		time.Now().Format("2006-01-02 15:04:05"),
		symbol,
		"ENTRY",
		strconv.FormatFloat(price, 'f', 0, 64),
		"-",
		strconv.Itoa(lots),
		date,
		"-",
		"0",
		"-",
		"-",
		"-",
	}
	if err := csv.NewWriter(f).Write(rec); err != nil {
		log.Printf("[FT] Gagal write CSV entry: %v", err)
	}
}

func openOrCreateCSV() (*os.File, error) {
	csvMutex.Lock()
	defer csvMutex.Unlock()

	absPath, err := filepath.Abs(ftCSVFile)
	if err != nil {
		absPath = ftCSVFile
	}

	f, err := os.OpenFile(absPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("stat file: %w", err)
	}

	if info.Size() == 0 {
		header := []string{
			"timestamp", "symbol", "action", "entry_price", "exit_price",
			"lots", "entry_date", "exit_date", "hold_days",
			"pnl_percent", "pnl_rp", "exit_reason",
		}
		if err := csv.NewWriter(f).Write(header); err != nil {
			f.Close()
			return nil, fmt.Errorf("write header: %w", err)
		}
	}

	if _, err := f.Seek(0, os.SEEK_END); err != nil {
		f.Close()
		return nil, fmt.Errorf("seek end: %w", err)
	}

	return f, nil
}

// WeeklyStats summarises closed trades for the current week.
type WeeklyStats struct {
	Total  int
	Wins   int
	PnLRp  float64
	AvgPct float64
}

func GetWeeklyStats() WeeklyStats {
	ftMutex.RLock()
	defer ftMutex.RUnlock()

	startDate, _ := time.Parse("2006-01-02", ftState.StartDate)
	weekStart := startDate.AddDate(0, 0, -int(startDate.Weekday()))

	var s WeeklyStats
	for _, t := range ftState.ClosedTrades {
		tExit, err := time.Parse("2006-01-02", t.ExitDate)
		if err != nil {
			continue
		}
		if tExit.After(weekStart) {
			s.Total++
			if t.PNLRp > 0 {
				s.Wins++
			}
			s.PnLRp += t.PNLRp
			s.AvgPct += t.PNLPercent
		}
	}
	if s.Total > 0 {
		s.AvgPct /= float64(s.Total)
	}
	return s
}
