package storage

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"learn-go/internal/config"
)

const StorageFile = "stocks.json"

// SaveData menyimpan map config.MyStocks ke dalam file JSON
func SaveData() {
	data, err := json.MarshalIndent(config.MyStocks, "", "  ")
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

	err = json.Unmarshal(data, &config.MyStocks)
	if err != nil {
		log.Printf("❌ Gagal decode JSON: %v", err)
	} else {
		log.Printf("✅ Berhasil memuat %d saham dari penyimpanan.", len(config.MyStocks))
	}
}