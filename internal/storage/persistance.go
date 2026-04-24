package storage

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	"learn-go/internal/config"
	"learn-go/internal/models"
)

const StorageFile = "stocks.json"

type StorageData struct {
	MyStocks      map[string]models.TradingPlan `json:"my_stocks"`
	PendingOrders map[string]models.ActiveOrder `json:"pending_orders"`
}

// SaveData menyimpan data ke JSON dengan pengaman Mutex
func SaveData() {
	// 🔒 [LOCK] Gunakan RLock (Read Lock) karena kita hanya ingin membaca data untuk di-marshal
	config.DataMutex.RLock() 
	dataToSave := StorageData{
		MyStocks:      config.MyStocks,
		PendingOrders: config.PendingOrders,
	}
	data, err := json.MarshalIndent(dataToSave, "", "  ")
	config.DataMutex.RUnlock() // 🔓 [UNLOCK] Segera buka gembok setelah marshal selesai
	
	if err != nil {
		log.Printf("❌ Gagal menukar data ke JSON: %v", err)
		return
	}

	err = ioutil.WriteFile(StorageFile, data, 0644)
	if err != nil {
		log.Printf("❌ Gagal menulis file: %v", err)
	}
}

// LoadData memuat data saat bot startup
func LoadData() {
	if _, err := os.Stat(StorageFile); os.IsNotExist(err) {
		log.Println("ℹ️ File penyimpanan belum ada, memulai data baru.")
		return
	}

	data, err := ioutil.ReadFile(StorageFile)
	if err != nil {
		log.Printf("❌ Gagal membaca file: %v", err)
		return
	}

	var storageData StorageData
	err = json.Unmarshal(data, &storageData)
	
	if err == nil && (storageData.MyStocks != nil || storageData.PendingOrders != nil) {
		// 🔒 [LOCK] Gunakan Lock (Write Lock) karena kita akan mengubah isi variabel global
		config.DataMutex.Lock() 
		if storageData.MyStocks != nil {
			config.MyStocks = storageData.MyStocks
		}
		if storageData.PendingOrders != nil {
			config.PendingOrders = storageData.PendingOrders
		}
		config.DataMutex.Unlock() // 🔓 [UNLOCK]
		
		log.Printf("✅ Berhasil memuat %d saham dan %d antrean.", len(config.MyStocks), len(config.PendingOrders))
		return
	}

	// Migrasi format lama (Backward Compatibility)
	var oldFormat map[string]models.TradingPlan
	errOld := json.Unmarshal(data, &oldFormat)
	if errOld == nil && oldFormat != nil {
		config.DataMutex.Lock() // 🔒 [LOCK]
		config.MyStocks = oldFormat
		config.DataMutex.Unlock() // 🔓 [UNLOCK] wajib dibuka sebelum panggil SaveData!
		
		SaveData() // Simpan ulang ke format baru
		log.Printf("✅ Migrasi format lama berhasil.")
	}
}