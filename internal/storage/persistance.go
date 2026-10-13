package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"learn-go/internal/config"
	"learn-go/internal/models"
)

var StorageFile = "stocks.json"

type StorageData struct {
	MyStocks      map[string]models.TradingPlan `json:"my_stocks"`
	PendingOrders map[string]models.ActiveOrder `json:"pending_orders"`
}

// SaveData menyimpan data ke JSON secara atomic (write ke .tmp lalu rename).
func SaveData() {
	config.DataMutex.RLock()
	dataToSave := StorageData{
		MyStocks:      config.MyStocks,
		PendingOrders: config.PendingOrders,
	}
	config.DataMutex.RUnlock()

	data, err := json.MarshalIndent(dataToSave, "", "  ")
	if err != nil {
		log.Printf("❌ Gagal menukar data ke JSON: %v", err)
		return
	}

	tmpFile := StorageFile + ".tmp"
	err = os.WriteFile(tmpFile, data, 0644)
	if err != nil {
		log.Printf("❌ Gagal menulis file: %v", err)
		return
	}

	if err := os.Rename(tmpFile, StorageFile); err != nil {
		log.Printf("❌ Gagal rename file atomic: %v", err)
		return
	}
}

// LoadData memuat data saat bot startup.
func LoadData() {
	if _, err := os.Stat(StorageFile); os.IsNotExist(err) {
		log.Println("ℹ️ File penyimpanan belum ada, memulai data baru.")
		return
	}

	data, err := os.ReadFile(StorageFile)
	if err != nil {
		log.Printf("❌ Gagal membaca file: %v", err)
		return
	}

	var storageData StorageData
	err = json.Unmarshal(data, &storageData)

	if err == nil && (storageData.MyStocks != nil || storageData.PendingOrders != nil) {
		config.DataMutex.Lock()
		if storageData.MyStocks != nil {
			config.MyStocks = storageData.MyStocks
		}
		if storageData.PendingOrders != nil {
			config.PendingOrders = storageData.PendingOrders
		}
		config.DataMutex.Unlock()

		log.Printf("✅ Berhasil memuat %d saham dan %d antrean.", len(config.MyStocks), len(config.PendingOrders))
		return
	}

	// Migrasi format lama (Backward Compatibility)
	var oldFormat map[string]models.TradingPlan
	errOld := json.Unmarshal(data, &oldFormat)
	if errOld == nil && oldFormat != nil {
		config.DataMutex.Lock()
		config.MyStocks = oldFormat
		config.DataMutex.Unlock()

		SaveData()
		log.Printf("✅ Migrasi format lama berhasil.")
		return
	}

	// Semua format gagal → file korup
	corruptName := fmt.Sprintf("%s.corrupt-%d", StorageFile, time.Now().Unix())
	if copyErr := os.WriteFile(corruptName, data, 0644); copyErr == nil {
		log.Printf("❌ File penyimpanan korup (%v). Disalin ke %s dan mulai kosong.", err, corruptName)
	} else {
		log.Printf("❌ File penyimpanan korup (%v). Gagal menyalin ke backup.", err)
	}
}
