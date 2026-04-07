package storage

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	"learn-go/internal/config"
	"learn-go/internal/models" // Jangan lupa import models jika ActiveOrder ada di sana
)

const StorageFile = "stocks.json"

// StorageData adalah pembungkus untuk menyimpan portofolio dan antrean sekaligus
type StorageData struct {
	MyStocks      map[string]models.TradingPlan `json:"my_stocks"`
	PendingOrders map[string]models.ActiveOrder `json:"pending_orders"`
}

// SaveData menyimpan config.MyStocks dan config.PendingOrders ke dalam file JSON
func SaveData() {
	// Masukkan kedua data ke dalam wadah pembungkus
	dataToSave := StorageData{
		MyStocks:      config.MyStocks,
		PendingOrders: config.PendingOrders,
	}

	data, err := json.MarshalIndent(dataToSave, "", "  ")
	if err != nil {
		log.Printf("❌ Gagal menukar data ke JSON: %v", err)
		return
	}

	err = ioutil.WriteFile(StorageFile, data, 0644)
	if err != nil {
		log.Printf("❌ Gagal menulis file: %v", err)
	}
}

// LoadData membaca data dari file JSON saat bot baru dinyalakan
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

	// 1. Coba decode menggunakan struktur baru (StorageData)
	var storageData StorageData
	err = json.Unmarshal(data, &storageData)
	
	if err == nil && (storageData.MyStocks != nil || storageData.PendingOrders != nil) {
		// Jika berhasil pakai struktur baru
		if storageData.MyStocks != nil {
			config.MyStocks = storageData.MyStocks
		}
		if storageData.PendingOrders != nil {
			config.PendingOrders = storageData.PendingOrders
		}
		log.Printf("✅ Berhasil memuat %d saham dan %d antrean dari penyimpanan.", len(config.MyStocks), len(config.PendingOrders))
		return
	}

	// 2. [BACKWARD COMPATIBILITY] Jika JSON masih menggunakan format lama (hanya MyStocks murni)
	log.Println("🔄 Mendeteksi format JSON lama, mencoba migrasi...")
	var oldFormat map[string]models.TradingPlan
	errOld := json.Unmarshal(data, &oldFormat)
	
	if errOld == nil && oldFormat != nil {
		config.MyStocks = oldFormat
		// PendingOrders dibiarkan kosong karena memang tidak ada di format lama
		log.Printf("✅ Berhasil memuat %d saham dari format penyimpanan lama. Format baru akan terbentuk saat save berikutnya.", len(config.MyStocks))
		
		// Langsung save agar file stocks.json ter-update ke struktur baru
		SaveData() 
	} else {
		log.Printf("❌ Gagal decode JSON (File rusak atau format tidak dikenal): %v", err)
	}
}