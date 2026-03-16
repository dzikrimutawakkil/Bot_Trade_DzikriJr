package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

const StorageFile = "stocks.json"

// saveData menyimpan map myStocks ke dalam file JSON
func saveData() {
	data, err := json.MarshalIndent(myStocks, "", "  ")
	if err != nil {
		log.Printf("❌ Gagal menukar data ke JSON: %v", err)
		return
	}

	err = ioutil.WriteFile(StorageFile, data, 0644)
	if err != nil {
		log.Printf("❌ Gagal menulis file: %v", err)
	}
}

// loadData membaca data dari file JSON saat bot baru dinyalakan
func loadData() {
	if _, err := os.Stat(StorageFile); os.IsNotExist(err) {
		log.Println("ℹ️ File penyimpanan belum ada, memulai data baru.")
		return
	}

	data, err := ioutil.ReadFile(StorageFile)
	if err != nil {
		log.Printf("❌ Gagal membaca file: %v", err)
		return
	}

	err = json.Unmarshal(data, &myStocks)
	if err != nil {
		log.Printf("❌ Gagal decode JSON: %v", err)
	} else {
		log.Printf("✅ Berhasil memuat %d saham dari penyimpanan.", len(myStocks))
	}
}