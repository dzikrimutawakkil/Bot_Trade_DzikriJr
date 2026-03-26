package market

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"log"
	"github.com/PuerkitoBio/goquery"
)

func GetGooglePrice(symbol string) float64 {
	url := fmt.Sprintf("https://www.google.com/finance/quote/%s:IDX", symbol)

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ Google Connection Error: %v", err)
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("❌ Google Return Status: %d", resp.StatusCode)
		return 0
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return 0
	}

	priceStr := ""

	// Trik 1: Cari elemen yang punya atribut 'data-last-price' (Paling Akurat)
	doc.Find("[data-last-price]").Each(func(i int, s *goquery.Selection) {
		val, exists := s.Attr("data-last-price")
		if exists && priceStr == "" {
			priceStr = val
		}
	})

	// Trik 2: Kalau Trik 1 gagal, cari class umum lainnya (fxKbKc atau YMl77)
	if priceStr == "" {
		doc.Find(".fxKbKc, .YMl77").Each(func(i int, s *goquery.Selection) {
			if priceStr == "" {
				priceStr = s.Text()
			}
		})
	}

	// Clean up: Hapus Rp, titik, koma, dsb
	priceStr = strings.ReplaceAll(priceStr, "Rp", "")
	priceStr = strings.ReplaceAll(priceStr, ".", "")
	priceStr = strings.ReplaceAll(priceStr, ",", "")
	priceStr = strings.TrimSpace(priceStr)

	price, _ := strconv.ParseFloat(priceStr, 64)
	
	if price == 0 {
		log.Printf("⚠️ Google Scraper gagal dapet angka buat %s", symbol)
	}

	return price
}