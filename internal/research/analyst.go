package research

import (
	"fmt"
    "learn-go/internal/market"
    "learn-go/internal/utils"
)

func calculateMA(prices []float64, period int) float64 {
	if len(prices) < period { return 0 }
	sum := 0.0
	for i := len(prices) - period; i < len(prices); i++ {
		sum += prices[i]
	}
	return sum / float64(period)
}

func calculateRSI(prices []float64) float64 {
	period := 14
	if len(prices) < period+1 { return 0 }

	var gains, losses float64
	for i := len(prices) - period; i < len(prices); i++ {
		diff := prices[i] - prices[i-1]
		if diff > 0 { gains += diff } else { losses -= diff }
	}
	
	if losses == 0 { return 100 }
	rs := (gains / float64(period)) / (losses / float64(period))
	return 100 - (100 / (1 + rs))
}

func GetStockScore(symbol string) (float64, string, float64, float64) {
	data, err := market.GetHistoricalPrices(symbol)
	if err != nil { return -1, "", -1, 0}

	var cleanPrices []float64
	var cleanVolumes []float64
	for i, p := range data.Prices {
		if p > 0 && i < len(data.Volumes) && data.Volumes[i] > 0 { 
			cleanPrices = append(cleanPrices, p)
			cleanVolumes = append(cleanVolumes, data.Volumes[i])
		}
	}
	
	if len(cleanPrices) < 20 { return -1, "Data kurang", -1, 0 }

	// --- KALKULASI INDIKATOR ---
	lastPrice := cleanPrices[len(cleanPrices)-1]
	ma20 := calculateMA(cleanPrices, 20)
	ma5 := calculateMA(cleanPrices, 5)
	
	lastVol := cleanVolumes[len(cleanVolumes)-1]
	avgVol := calculateMA(cleanVolumes, 20)
	
	// RSI masih bisa dipakai sekadar untuk filter tambahan
	rsiToday := calculateRSI(cleanPrices)

	// Hitung jarak harga ke MA20 dalam persen (Sangat Krusial untuk BoW)
	distToMA := ((lastPrice - ma20) / ma20) * 100

	score := 0.0
	verdict := ""

	// --- LOGIKA SKORING BUY ON WEAKNESS (BoW) ---

	// 1. 🟢 HIJAU SANGAT KUAT (Golden Setup BoW)
	// Syarat: Harga di atas MA20 (Masih Uptrend Utama), tapi sedang koreksi di bawah MA5, 
	// jarak ke MA20 sangat dekat (0% s/d 3%), DAN Volume Kering (Seller habis).
	if lastPrice >= ma20 && lastPrice < ma5 && distToMA <= 3.0 && lastVol < avgVol {
		score = 10
		verdict = "🟢 **SETUP BUY ON WEAKNESS (GOLDEN)**\nAlasan: Harga sedang koreksi mendekati Support MA20 dengan volume kering (Tekanan jual ritel sudah habis). Ini adalah area beli risiko rendah."
	
	// 2. 🟠 SIAGA (Hampir Menyentuh Support / Koreksi dengan Volume Normal)
	// Syarat: Harga masih di atas MA20, jarak < 5%, tapi volume belum benar-benar kering.
	} else if lastPrice >= ma20 && lastPrice < ma5 && distToMA <= 5.0 {
		score = 8
		verdict = "🟠 **SIAGA PANTULAN (BoW)**\nAlasan: Harga sedang turun mendekati Support MA20. Pantau ketat, siap-siap entry jika besok muncul pantulan."

	// 3. 🟡 RAWAN PUCUK / KEMAHALAN (Mantan Setup Breakout)
	// Syarat: Harga terbang jauh di atas MA20 (> 5%) atau RSI sudah kepanasan.
	} else if distToMA > 5.0 || rsiToday > 70 {
		score = 4 // Skor kita turunkan drastis agar tidak direkomendasikan AI
		verdict = "🟡 **RAWAN PUCUK / FOMO**\nAlasan: Harga sudah terbang terlalu jauh dari Support MA20. Berisiko besar terkena bantingan (Take Profit bandar). Jangan dikejar!"

	// 4. 🔴 PISAU JATUH (Downtrend Parah)
	// Syarat: Harga tembus ke bawah MA20 (Support jebol).
	} else if lastPrice < ma20 {
		score = 1
		verdict = "🔴 **PISAU JATUH (DOWNTREND)**\nAlasan: Harga sudah jebol ke bawah MA20. Tren utama rusak. Hindari saham ini sampai dia bisa naik lagi ke atas MA20."
	
	// 5. Kondisi Sideways / Tanggung
	} else {
		score = 5
		verdict = "⚪ **TANGGUNG / SIDEWAYS**\nAlasan: Harga nanggung, tidak dekat support dan tidak terlalu pucuk. Skip cari saham lain."
	}

	return score, verdict, distToMA, ma20
}

// GetMarketFilterStatus mengecek tren IHSG saat ini
func GetMarketFilterStatus() (bool, string) {
	// 1. Tarik data IHSG (^JKSE) - Ingat, kembaliannya adalah Struct
	data, err := market.GetHistoricalPrices("^JKSE")
	if err != nil {
		return true, "⚠️ Gagal mengecek IHSG, asumsikan pasar normal." // Fallback
	}

	// 2. Ekstrak dan bersihkan array harganya (seperti di GetStockScore)
	var cleanPrices []float64
	for _, p := range data.Prices {
		if p > 0 {
			cleanPrices = append(cleanPrices, p)
		}
	}

	// 3. Pastikan data cukup
	if len(cleanPrices) < 20 {
		return true, "⚠️ Data IHSG tidak lengkap, asumsikan pasar normal."
	}

	// 4. Kalkulasi Indikator
	currentIHSG := cleanPrices[len(cleanPrices)-1]
	ma20IHSG := calculateMA(cleanPrices, 20)
	ma5IHSG := calculateMA(cleanPrices, 5)

	// KONDISI 1: MARKET CRASH (IHSG di bawah MA20) -> DILARANG TRADING
	if currentIHSG < ma20IHSG {
		return false, fmt.Sprintf("🚨 **MARKET DOWNTREND / CRASH!** 🚨\nIHSG saat ini (%s) berada di bawah tren MA20 (%s).\n\n_Bot menyarankan: **CASH IS KING**. Jangan paksakan entry saat badai!_", utils.FormatRupiah(currentIHSG), utils.FormatRupiah(ma20IHSG))
	}

	// KONDISI 2: MARKET KOREKSI (IHSG di bawah MA5) -> HATI-HATI
	if currentIHSG < ma5IHSG {
		return true, fmt.Sprintf("⚠️ **MARKET SEDANG KOREKSI WAJAR** ⚠️\nIHSG (%s) di bawah MA5, namun tren MA20 masih terjaga.\n\n_Status: Boleh trading, tapi kurangi agresivitas._", utils.FormatRupiah(currentIHSG))
	}

	// KONDISI 3: MARKET UPTREND
	return true, fmt.Sprintf("🟢 **MARKET UPTREND (BULLISH)** 🟢\nIHSG (%s) berada kuat di atas MA5 dan MA20.\n\n_Status: Kondisi ideal untuk Fast Swing! Gas poll! 🚀_", utils.FormatRupiah(currentIHSG))
}