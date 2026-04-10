package research

import (
	"fmt"
	"learn-go/internal/market"
	"learn-go/internal/utils"
)

func calculateMA(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}
	sum := 0.0
	for i := len(prices) - period; i < len(prices); i++ {
		sum += prices[i]
	}
	return sum / float64(period)
}

func calculateRSI(prices []float64) float64 {
	period := 14
	if len(prices) < period+1 {
		return 0
	}

	var gains, losses float64
	for i := len(prices) - period; i < len(prices); i++ {
		diff := prices[i] - prices[i-1]
		if diff > 0 {
			gains += diff
		} else {
			losses -= diff
		}
	}

	if losses == 0 {
		return 100
	}
	rs := (gains / float64(period)) / (losses / float64(period))
	return 100 - (100 / (1 + rs))
}

// 1. STRATEGI SIDEWAYS (Buy on Weakness) + Filter Likuiditas
func GetScoreBoW(symbol string) (float64, string, float64, float64) {
	data, err := market.GetHistoricalPrices(symbol)
	if err != nil { return -1, "", -1, 0 }

	var cleanPrices, cleanVolumes []float64
	for i, p := range data.Prices {
		if p > 0 && i < len(data.Volumes) && data.Volumes[i] > 0 {
			cleanPrices = append(cleanPrices, p)
			cleanVolumes = append(cleanVolumes, data.Volumes[i])
		}
	}

	if len(cleanPrices) < 20 { return -1, "Data kurang", -1, 0 }

	lastPrice := cleanPrices[len(cleanPrices)-1]
	ma20 := calculateMA(cleanPrices, 20)
	ma5 := calculateMA(cleanPrices, 5)

	lastVol := cleanVolumes[len(cleanVolumes)-1]
	avgVol := calculateMA(cleanVolumes, 20)
	rsiToday := calculateRSI(cleanPrices)

	distToMA := ((lastPrice - ma20) / ma20) * 100

	// 🛑 FILTER 1: LIKUIDITAS (Anti Saham Sepi / Gorengan)
	if avgVol < 2000000 {
		return 1, "🔴 **SKIP (TIDAK LIKUID)**\nAlasan: Transaksi terlalu sepi. Rawan dimanipulasi bandar dan sulit jualan.", distToMA, ma20
	}

	score := 0.0
	verdict := ""

	if lastPrice >= ma20 && lastPrice < ma5 && distToMA <= 3.0 && lastVol < avgVol {
		score = 10
		verdict = "🟢 **SETUP BUY ON WEAKNESS (GOLDEN)**\nAlasan: Harga terkoreksi mendekati Support MA20 dengan volume kering. Area beli risiko rendah."
	} else if lastPrice >= ma20 && lastPrice < ma5 && distToMA <= 5.0 {
		score = 8
		verdict = "🟠 **SIAGA PANTULAN (BoW)**\nAlasan: Harga sedang turun mendekati Support MA20. Pantau ketat."
	} else if distToMA > 5.0 || rsiToday > 70 {
		score = 4
		verdict = "🟡 **RAWAN PUCUK / FOMO**\nAlasan: Harga terbang terlalu jauh dari Support MA20. Jangan dikejar!"
	} else if lastPrice < ma20 {
		score = 1
		verdict = "🔴 **PISAU JATUH (DOWNTREND)**\nAlasan: Harga jebol ke bawah MA20. Tren utama rusak. Hindari!"
	} else {
		score = 5
		verdict = "⚪ **TANGGUNG / SIDEWAYS**\nAlasan: Harga nanggung, tidak dekat support dan tidak terlalu pucuk. Skip."
	}

	return score, verdict, distToMA, ma20
}

// 2. STRATEGI BULLISH (Breakout Momentum) + Fake Breakout Protection
func GetScoreBreakout(symbol string) (float64, string, float64, float64) {
	data, err := market.GetHistoricalPrices(symbol)
	if err != nil { return -1, "", -1, 0 }

	var cleanPrices, cleanVolumes, cleanHighs, cleanOpens []float64
	for i, p := range data.Prices {
		if p > 0 && i < len(data.Volumes) && data.Volumes[i] > 0 && i < len(data.Highs) && i < len(data.Opens) {
			cleanPrices = append(cleanPrices, p)
			cleanVolumes = append(cleanVolumes, data.Volumes[i])
			cleanHighs = append(cleanHighs, data.Highs[i])
			cleanOpens = append(cleanOpens, data.Opens[i])
		}
	}

	if len(cleanPrices) < 20 { return -1, "Data kurang", -1, 0 }

	lastPrice := cleanPrices[len(cleanPrices)-1]
	lastHigh := cleanHighs[len(cleanHighs)-1]
	lastOpen := cleanOpens[len(cleanOpens)-1]
	ma20 := calculateMA(cleanPrices, 20)
	ma5 := calculateMA(cleanPrices, 5)
	
	lastVol := cleanVolumes[len(cleanVolumes)-1]
	avgVol := calculateMA(cleanVolumes, 20)

	distToMA5 := ((lastPrice - ma5) / ma5) * 100

	// 🛑 FILTER 1: LIKUIDITAS (Anti Saham Sepi / Gorengan)
	// Jika rata-rata volume di bawah 2 juta lembar (20.000 lot) sehari
	if avgVol < 2000000 {
		return 1, "🔴 **SKIP (TIDAK LIKUID)**\nAlasan: Transaksi terlalu sepi. Rawan dimanipulasi bandar dan sulit jualan.", distToMA5, ma5
	}

	// 🛑 FILTER 2: FAKE BREAKOUT (Jarum Atas)
	// Jika bayangan atas (upper wick) lebih panjang 1.5x lipat dari body candle
	upperWick := lastHigh - lastPrice
	bodySize := lastPrice - lastOpen
	if bodySize < 0 { bodySize = lastOpen - lastPrice } // Nilai absolut
	if bodySize == 0 { bodySize = 1 } // Mencegah error dibagi nol

	if upperWick > (bodySize * 1.5) && lastPrice > ma5 {
		return 3, "🟡 **FAKE BREAKOUT (JARUM ATAS)**\nAlasan: Harga ditarik naik tapi dibanting lagi ke bawah (tekanan jual besar). Menghindari jebakan pucuk!", distToMA5, ma5
	}

	// LOGIKA SKORING BREAKOUT UTAMA
	score := 0.0
	verdict := ""

	if lastPrice > ma5 && lastPrice > ma20 && lastVol > (avgVol*1.5) {
		score = 10
		verdict = "🟢 **SETUP BREAKOUT MOMENTUM**\nAlasan: Harga naik di atas MA5 dengan lonjakan volume solid (tanpa jarum atas panjang). Bandar akumulasi!"
	} else if lastPrice > ma5 && lastPrice > ma20 && distToMA5 <= 3.0 {
		score = 8
		verdict = "🟠 **BUY ON STRENGTH**\nAlasan: Harga merayap naik perlahan di atas MA5 dengan jarak aman. Boleh antre."
	} else if distToMA5 > 5.0 {
		score = 4
		verdict = "🟡 **RAWAN PUCUK (OVEREXTENDED)**\nAlasan: Sudah terbang terlalu jauh dari MA5. Berisiko dibanting."
	} else {
		score = 2
		verdict = "🔴 **TIDAK ADA MOMENTUM**\nAlasan: Harga di bawah MA5 atau pergerakan kurang agresif."
	}

	return score, verdict, distToMA5, ma5
}

// 3. THE ROUTER: Memilih strategi berdasarkan cuaca IHSG
func EvaluateStockAdaptive(symbol string, marketRegime string) (float64, string, float64, float64) {
	switch marketRegime {
	case "BULLISH":
		return GetScoreBreakout(symbol)

	case "BEARISH":
		score, verdict, dist, ma := GetScoreBoW(symbol)
		if score < 10 {
			score = 1
			verdict = "🔴 **SKIP (MARKET CRASH)**\nAlasan: Setup kurang kuat untuk melawan arus IHSG yang sedang hancur."
		}
		return score, verdict, dist, ma

	case "SIDEWAYS":
		fallthrough
	default:
		return GetScoreBoW(symbol)
	}
}

// 4. SENSOR PASAR (IHSG Tracker)
func GetMarketFilterStatus() (string, string) {
	data, err := market.GetHistoricalPrices("^JKSE")
	if err != nil {
		return "SIDEWAYS", "⚠️ Gagal mengecek IHSG, asumsikan pasar normal (SIDEWAYS)."
	}

	var cleanPrices []float64
	for _, p := range data.Prices {
		if p > 0 {
			cleanPrices = append(cleanPrices, p)
		}
	}

	if len(cleanPrices) < 20 {
		return "SIDEWAYS", "⚠️ Data IHSG tidak lengkap, asumsikan pasar normal (SIDEWAYS)."
	}

	currentIHSG := cleanPrices[len(cleanPrices)-1]
	ma20IHSG := calculateMA(cleanPrices, 20)
	ma5IHSG := calculateMA(cleanPrices, 5)

	if currentIHSG < ma20IHSG {
		return "BEARISH", fmt.Sprintf("🚨 **MARKET DOWNTREND (BEARISH)** 🚨\nIHSG saat ini (%s) berada di bawah tren MA20 (%s).\n\n_Bot beralih ke Mode Defensif. Hanya mencari saham anti-badai._", utils.FormatRupiah(currentIHSG), utils.FormatRupiah(ma20IHSG))
	}

	if currentIHSG < ma5IHSG {
		return "SIDEWAYS", fmt.Sprintf("⚠️ **MARKET KOREKSI WAJAR (SIDEWAYS)** ⚠️\nIHSG (%s) di bawah MA5, namun tren MA20 masih terjaga.\n\n_Bot beralih ke Mode Normal (Buy on Weakness)._", utils.FormatRupiah(currentIHSG))
	}

	return "BULLISH", fmt.Sprintf("🟢 **MARKET UPTREND (BULLISH)** 🟢\nIHSG (%s) berada kuat di atas MA5 dan MA20.\n\n_Bot beralih ke Mode Agresif (Momentum/Breakout)! Gas poll! 🚀_", utils.FormatRupiah(currentIHSG))
}