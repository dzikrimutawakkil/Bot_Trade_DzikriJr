// internal/storage/logger.go
package storage

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"time"
)

// LogTrade akan mencatat setiap transaksi ke dalam file CSV
func LogTrade(action, symbol string, price float64, lots int, pnlPercent float64, notes string) {
	fileName := "trade_history.csv"
	
	// Cek apakah file sudah ada
	fileExists := true
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		fileExists = false
	}

	// Buka file dengan mode Append (tambahkan di baris bawah)
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("❌ Gagal membuka file log:", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Jika file baru dibuat, tulis Header (judul kolom) terlebih dahulu
	if !fileExists {
		header := []string{"Waktu", "Aksi", "Saham", "Harga", "Lot", "PNL (%)", "Catatan"}
		writer.Write(header)
	}

	// Siapkan data baris baru
	record := []string{
		time.Now().Format("2006-01-02 15:04:05"), // Waktu eksekusi
		action,                                   // BUY / SELL
		symbol,                                   // Kode Saham (ADRO, EMTK, dll)
		fmt.Sprintf("%.0f", price),               // Harga eksekusi
		fmt.Sprintf("%d", lots),                  // Jumlah Lot
		fmt.Sprintf("%.2f", pnlPercent),          // Untung/Rugi dalam persen
		notes,                                    // Catatan khusus
	}

	err = writer.Write(record)
	if err != nil {
		log.Println("❌ Gagal menulis ke CSV:", err)
	}
}