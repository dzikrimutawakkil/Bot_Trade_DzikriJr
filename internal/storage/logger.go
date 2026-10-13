// internal/storage/logger.go
package storage

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"learn-go/internal/pricing"
	"learn-go/internal/rules"
)

const CSVFile = "trade_history.csv"

// TradeRecord merepresentasikan satu baris transaksi di CSV v3.
type TradeRecord struct {
	Time       time.Time
	Action     string // "BUY" | "SELL"
	Symbol     string
	Price      float64
	Lots       int
	Strategy   string // default: rules.Default.Strategy
	PNLPercent float64
	PNLRp      float64
	ExitReason string // "TP1" | "TP" | "TSL" | "CL" | "MANUAL"; kosong untuk BUY
	Note       string
}

// LogTrade mencatat transaksi ke CSV.
// Jika rec.Time kosong, otomatis di-set ke time.Now().
func LogTrade(rec TradeRecord) {
	if rec.Time.IsZero() {
		rec.Time = time.Now()
	}
	if rec.Strategy == "" {
		rec.Strategy = rules.Default.Strategy
	}

	file, err := os.OpenFile(CSVFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("❌ Gagal membuka file log:", err)
		return
	}
	defer file.Close()

	// Tulis header v3 jika file baru
	stat, _ := file.Stat()
	if stat.Size() == 0 {
		writeHeader(file)
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()

	record := []string{
		rec.Time.Format("2006-01-02 15:04:05"),
		rec.Action,
		rec.Symbol,
		fmt.Sprintf("%.0f", rec.Price),
		fmt.Sprintf("%d", rec.Lots),
		rec.Strategy,
		fmt.Sprintf("%.2f", rec.PNLPercent),
		fmt.Sprintf("%.0f", rec.PNLRp),
		rec.ExitReason,
		rec.Note,
	}
	if err := writer.Write(record); err != nil {
		log.Println("❌ Gagal menulis ke CSV:", err)
	}
}

func writeHeader(f *os.File) {
	header := []string{
		"Waktu", "Aksi", "Saham", "Harga", "Lot",
		"Strategi", "PNL (%)", "PNL (Rp)", "AlasanExit", "Catatan",
	}
	writer := csv.NewWriter(f)
	writer.Write(header)
	writer.Flush()
}

// detectLayout membaca jumlah kolom dan menentukan versi CSV.
func detectLayout(cols int) int {
	switch cols {
	case 7:
		return 1 // v1
	case 8:
		return 2 // v2
	case 10:
		return 3 // v3
	default:
		return 0 // unknown
	}
}

// ReadTradeHistory membaca file CSV dan mengembalikan semua TradeRecord.
// Layout dideteksi per-baris agar file lama (campuran v1/v2/v3) tetap bisa dibaca.
func ReadTradeHistory(path string) ([]TradeRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	var records []TradeRecord
	skippedUnknown := false

	for {
		fields, err := reader.Read()
		if err == io.EOF {
			break
		}

		// Skip blank rows
		if len(fields) == 0 || (len(fields) == 1 && strings.TrimSpace(fields[0]) == "") {
			continue
		}

		trim0 := strings.TrimSpace(fields[0])

		// Skip separator bars
		if strings.Contains(trim0, "---") || strings.Contains(trim0, "===") {
			continue
		}

		// Header detection: kolom pertama "Waktu" — set global layout
		if trim0 == "Waktu" {
			continue
		}

		// Per-bar layout detection berdasarkan jumlah kolom aktual baris ini.
		layout := detectLayout(len(fields))

		switch layout {
		case 1:
			rec, ok := parseV1(fields)
			if !ok {
				if !skippedUnknown {
					log.Printf("[ReadTradeHistory] Baris tidak recognised, dilewati: %v", fields)
					skippedUnknown = true
				}
				continue
			}
			records = append(records, rec)
		case 2:
			rec, ok := parseV2(fields)
			if !ok {
				if !skippedUnknown {
					log.Printf("[ReadTradeHistory] Baris tidak recognised, dilewati: %v", fields)
					skippedUnknown = true
				}
				continue
			}
			records = append(records, rec)
		case 3:
			rec, ok := parseV3(fields)
			if !ok {
				if !skippedUnknown {
					log.Printf("[ReadTradeHistory] Baris tidak recognised, dilewati: %v", fields)
					skippedUnknown = true
				}
				continue
			}
			records = append(records, rec)
		default:
			// Baris tidak dikenali
			if !skippedUnknown {
				log.Printf("[ReadTradeHistory] Layout tidak dikenali, dilewati: %v", fields)
				skippedUnknown = true
			}
			continue
		}
	}

	return records, nil
}

// parseV1: Waktu,Aksi,Saham,Harga,Lot,PNL (%),Catatan
func parseV1(fields []string) (TradeRecord, bool) {
	if len(fields) < 7 {
		return TradeRecord{}, false
	}
	action := strings.TrimSpace(fields[1])
	if action != "BUY" && action != "SELL" {
		return TradeRecord{}, false
	}
	t, _ := time.ParseInLocation("2006-01-02 15:04:05", strings.TrimSpace(fields[0]), time.Local)
	lots, _ := parseInt(fields[4])
	pnlPct, _ := parseFloat(fields[5])
	return TradeRecord{
		Time:       t,
		Action:     action,
		Symbol:     strings.TrimSpace(fields[2]),
		Price:      parseFloatOr(fields[3], 0),
		Lots:       lots,
		Strategy:   rules.Default.Strategy,
		PNLPercent: pnlPct,
		PNLRp:      0,
		ExitReason: "",
		Note:       strings.TrimSpace(fields[6]),
	}, true
}

// parseV2: Waktu,Aksi,Saham,Harga,Lot,Strategi,PNL (%),Catatan
func parseV2(fields []string) (TradeRecord, bool) {
	if len(fields) < 8 {
		return TradeRecord{}, false
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", strings.TrimSpace(fields[0]), time.Local)
	if err != nil {
		t = time.Time{}
	}
	lots, _ := parseInt(fields[4])
	pnlPct, _ := parseFloat(fields[6])
	strategy := strings.TrimSpace(fields[5])
	if strategy == "" {
		strategy = rules.Default.Strategy
	}
	return TradeRecord{
		Time:       t,
		Action:     strings.TrimSpace(fields[1]),
		Symbol:     strings.TrimSpace(fields[2]),
		Price:      parseFloatOr(fields[3], 0),
		Lots:       lots,
		Strategy:   strategy,
		PNLPercent: pnlPct,
		PNLRp:      0,
		ExitReason: "",
		Note:       strings.TrimSpace(fields[7]),
	}, true
}

// parseV3: Waktu,Aksi,Saham,Harga,Lot,Strategi,PNL (%),PNL (Rp),AlasanExit,Catatan
func parseV3(fields []string) (TradeRecord, bool) {
	if len(fields) < 10 {
		return TradeRecord{}, false
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", strings.TrimSpace(fields[0]), time.Local)
	if err != nil {
		t = time.Time{}
	}
	lots, _ := parseInt(fields[4])
	pnlPct, _ := parseFloat(fields[6])
	pnlRp, _ := parseFloat(fields[7])
	strategy := strings.TrimSpace(fields[5])
	if strategy == "" {
		strategy = rules.Default.Strategy
	}
	return TradeRecord{
		Time:       t,
		Action:     strings.TrimSpace(fields[1]),
		Symbol:     strings.TrimSpace(fields[2]),
		Price:      parseFloatOr(fields[3], 0),
		Lots:       lots,
		Strategy:   strategy,
		PNLPercent: pnlPct,
		PNLRp:      pnlRp,
		ExitReason: strings.TrimSpace(fields[8]),
		Note:       strings.TrimSpace(fields[9]),
	}, true
}

func parseFloat(s string) (float64, bool) {
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", ".")
	s = strings.TrimSuffix(s, "%")
	v, err := strconv.ParseFloat(s, 64)
	return v, err == nil
}

func parseFloatOr(s string, fallback float64) float64 {
	v, ok := parseFloat(s)
	if ok {
		return v
	}
	return fallback
}

func parseInt(s string) (int, bool) {
	v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return int(v), err == nil
}

// MigrateTradeHistory memigrasi file ke format v3.
// Idempotent: jika file tidak ada atau sudah v3 semua, tidak melakukan apa-apa.
// v1/v2 SELL tanpa PNLRp dihitung ulang via position replay.
func MigrateTradeHistory(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // tidak ada file, tidak perlu migrasi
	}

	// Baca semua record
	records, err := ReadTradeHistory(path)
	if err != nil {
		return fmt.Errorf("gagal membaca file: %w", err)
	}

	// Cek apakah sudah semua v3
	allV3 := true
	for _, r := range records {
		if r.PNLRp == 0 && r.Action == "SELL" {
			allV3 = false
			break
		}
	}
	if allV3 && len(records) > 0 {
		return nil // sudah v3, tidak perlu migrasi
	}

	// Replay posisi per saham untuk menghitung PNLRp yang belum ada
	records = replayPositions(records)

	// Backup file lama
	backupPath := path + ".bak"
	if _, err := os.Stat(backupPath); err == nil {
		// Backup sudah ada, tambah suffix timestamp
		backupPath = fmt.Sprintf("%s.%s.bak", path, time.Now().Format("20060102_150405"))
	}
	if err := copyFile(path, backupPath); err != nil {
		log.Printf("[MigrateTradeHistory] Gagal backup ke %s: %v", backupPath, err)
	} else {
		log.Printf("[MigrateTradeHistory] Backup disimpan: %s", backupPath)
	}

	// Tulis ulang sebagai v3
	file, err := os.OpenFile(path, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("gagal menulis file: %w", err)
	}
	defer file.Close()

	writeHeader(file)
	writer := csv.NewWriter(file)
	for _, rec := range records {
		writer.Write([]string{
			rec.Time.Format("2006-01-02 15:04:05"),
			rec.Action,
			rec.Symbol,
			fmt.Sprintf("%.0f", rec.Price),
			fmt.Sprintf("%d", rec.Lots),
			rec.Strategy,
			fmt.Sprintf("%.2f", rec.PNLPercent),
			fmt.Sprintf("%.0f", rec.PNLRp),
			rec.ExitReason,
			rec.Note,
		})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("gagal menulis CSV: %w", err)
	}

	log.Printf("[MigrateTradeHistory] Migrasi selesai: %d record", len(records))
	return nil
}

// replayPositions menghitung ulang PNLRp untuk SELL yang belum punya.
// BUY/SELL dibaca berurutan per saham; PNLRp dihitung dari avg price saat itu.
func replayPositions(records []TradeRecord) []TradeRecord {
	type pos struct {
		avgPrice   float64
		totalLots  int
		totalCost  float64
		exitReason string
		note       string
	}

	positions := make(map[string]*pos)
	result := make([]TradeRecord, 0, len(records))

	for i := range records {
		rec := &records[i]
		if rec.Action == "BUY" {
			// Update posisi
			p, ok := positions[rec.Symbol]
			if !ok {
				p = &pos{}
				positions[rec.Symbol] = p
			}
			p.totalCost += rec.Price * float64(rec.Lots) * 100
			p.totalLots += rec.Lots
			if p.totalLots > 0 {
				p.avgPrice = p.totalCost / float64(p.totalLots) / 100
			}
			rec.PNLRp = 0
			result = append(result, *rec)
		} else if rec.Action == "SELL" {
			p, ok := positions[rec.Symbol]
			if !ok || p.totalLots == 0 {
				// Fallback: hitung dari PNLPct saja
				rec.PNLRp = calculateFallbackRp(rec)
				rec.ExitReason = inferExitReasonFromNote(rec.Note)
				if rec.ExitReason == "" {
					rec.ExitReason = "MANUAL"
				}
				result = append(result, *rec)
				continue
			}
			// Hitung PNLRp dari avg price
			sellValue := rec.Price * (1 - rules.Default.SellFee) * float64(rec.Lots) * 100
			buyCost := p.avgPrice * (1 + rules.Default.BuyFee) * float64(rec.Lots) * 100
			rec.PNLRp = sellValue - buyCost
			rec.ExitReason = inferExitReasonFromNote(rec.Note)
			if rec.ExitReason == "" {
				rec.ExitReason = "MANUAL"
			}
			if rec.Strategy == "" {
				rec.Strategy = rules.Default.Strategy
			}
			result = append(result, *rec)

			// Kurangi posisi
			p.totalLots -= rec.Lots
			if p.totalLots > 0 {
				p.totalCost = p.avgPrice * float64(p.totalLots) * 100
			} else {
				delete(positions, rec.Symbol)
			}
		}
	}

	return result
}

func calculateFallbackRp(rec *TradeRecord) float64 {
	// Fallback: PNLRp = Harga*(1-SellFee)*Lot*100 * (1 - 1/(1+pnl%/100))
	if rec.PNLPercent == 0 {
		return 0
	}
	base := rec.Price * (1 - rules.Default.SellFee) * float64(rec.Lots) * 100
	return base * (1 - 1/(1+rec.PNLPercent/100))
}

func inferExitReasonFromNote(note string) string {
	n := strings.ToLower(note)
	if strings.Contains(n, "take profit") {
		return "TP"
	}
	if strings.Contains(n, "cut loss") {
		return "CL"
	}
	if strings.Contains(n, "trailing") {
		return "TSL"
	}
	return "MANUAL"
}

func ResolveExitReason(price float64, entry, cutLoss, takeProfit, highest float64, tp1Done, trailingActive bool) string {
	if price <= cutLoss {
		return "CL"
	}
	if trailingActive && price <= pricing.RoundToFraction(highest*(1-rules.Default.TrailingStopPct/100)) {
		return "TSL"
	}
	if !tp1Done && takeProfit > 0 && price >= takeProfit {
		return "TP1"
	}
	return "MANUAL"
}

func copyFile(src, dst string) error {
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

	_, err = io.Copy(w, r)
	return err
}
