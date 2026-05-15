package research

import (
	"fmt"
	"math"
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

	var cleanOpens, cleanHighs, cleanLows, cleanCloses, cleanVolumes []float64

	// Bersihkan data OHLCV yang kosong/libur
	for i, p := range data.Prices {
		if p > 0 && i < len(data.Volumes) && data.Volumes[i] > 0 &&
		   i < len(data.Opens) && data.Opens[i] > 0 &&
		   i < len(data.Highs) && data.Highs[i] > 0 &&
		   i < len(data.Lows) && data.Lows[i] > 0 {

			cleanOpens = append(cleanOpens, data.Opens[i])
			cleanHighs = append(cleanHighs, data.Highs[i])
			cleanLows = append(cleanLows, data.Lows[i])
			cleanCloses = append(cleanCloses, p)
			cleanVolumes = append(cleanVolumes, data.Volumes[i])
		}
	}

	if len(cleanCloses) < 20 { return -1, "Data kurang", -1, 0 }

	// --- VARIABEL HARI INI & KEMARIN ---
	idxToday := len(cleanCloses) - 1
	idxYest := len(cleanCloses) - 2

	todayOpen := cleanOpens[idxToday]
	todayHigh := cleanHighs[idxToday]
	todayLow := cleanLows[idxToday]
	todayClose := cleanCloses[idxToday]
	todayVol := cleanVolumes[idxToday]

	yestOpen := cleanOpens[idxYest]
	yestClose := cleanCloses[idxYest]
	yestVol := cleanVolumes[idxYest]

	// --- KALKULASI INDIKATOR ---
	ma20 := calculateMA(cleanCloses, 20)
	ma5 := calculateMA(cleanCloses, 5)
	avgVol := calculateMA(cleanVolumes, 20)

	distToMA := ((todayClose - ma20) / ma20) * 100

	verdict := ""

	// =========================================================
	// FILTER C-BoW (urutan HARD BLOCK — urutannya KUNCIA)
	// =========================================================

	// 🔥 FILTER 1: ANTI-ARB — yesterday change < -10%
	percentDropYesterday := ((yestClose - yestOpen) / yestOpen) * 100
	if percentDropYesterday <= -10.0 {
		return -1, "⛔ **ANTI-ARB —昨天的 stock turun >10%, risiko lanjut dump**\nAlasan: Saham ini baru saja dibanting kemarin. Sangat berisiko menangkap pisau jatuh walaupun ada pantulan semu hari ini. Jangan disentuh!", distToMA, ma20
	}

	// 🔥 FILTER 2: ANTI-DISTRIBUTION TRAP — pump from low > 3.5%
	pumpFromLow := ((todayClose - todayLow) / todayLow) * 100
	if pumpFromLow > 3.5 {
		return -1, fmt.Sprintf("⛔ **ANTI-DISTRIBUTION TRAP — pump dari low terlalu besar (+%.2f%%)**\nAlasan: Harga sudah ditarik naik lebih dari 3.5%% dari titik terendahnya hari ini. Risiko ARB (ARB = Airborne Rescue — tiba-tiba hilang naiknya) sangat tinggi. Jangan entry!", pumpFromLow), distToMA, ma20
	}

	// --- Kalkulasi pola teknikal (hanya dijalankan jika tidak ter-blokir di atas) ---
	score := 0.0

	// 1. HIGH-QUALITY HAMMER FILTER
	body := math.Abs(todayClose - todayOpen)
	lowerWick := math.Min(todayOpen, todayClose) - todayLow
	dailyRange := todayHigh - todayLow
	if dailyRange == 0 { dailyRange = 1 }

	isHammer := todayLow <= ma20 &&
				todayClose > ma20 &&
				lowerWick >= (2 * body) &&
				(todayHigh - todayClose) <= (dailyRange * 0.3)

	// 2. STRONG GREEN BOUNCE FILTER
	yestBodyMidpoint := (yestOpen + yestClose) / 2.0
	isGreenBounce := todayClose > todayOpen &&
					 todayClose > yestBodyMidpoint &&
					 todayVol > yestVol &&
					 distToMA <= 4.0 &&
					 todayLow >= ma20*0.98

	// --- LOGIKA SKORING FINAL (hanya untuk saham yang tidak ter-blokir) ---

	if isHammer {
		score = 10
		verdict = "🟢 **CONFIRMED BoW (HIGH-QUALITY HAMMER)**\nAlasan: Terdapat perlawanan sangat kuat dari *buyer*! Ekor bawah > 2x body dan ditutup di area tertinggi hariannya. Tembok Bandar di MA20 tervalidasi."

	} else if isGreenBounce {
		score = 10
		verdict = "🟢 **CONFIRMED BoW (STRONG GREEN BOUNCE)**\nAlasan: Candlestick hari ini berhasil menelan separuh lebih *body* candle kemarin dengan dukungan volume membesar. Smart Money sedang akumulasi di MA20."

	} else if todayClose >= ma20 && todayClose < ma5 && distToMA <= 3.0 && todayVol < avgVol {
		score = 7
		verdict = "🟡 **SIAGA PANTULAN (BELUM KONFIRMASI)**\nAlasan: Harga koreksi di MA20 dengan volume kering, TAPI belum ada bukti perlawanan buyer (tidak ada Hammer/Green Bounce). JANGAN BELI, tunggu konfirmasi besok."

	} else if distToMA > 5.0 {
		score = 4
		verdict = "🔴 **RAWAN PUCUK / FOMO**\nAlasan: Harga terlalu jauh dari MA20. Berisiko besar terkena bantingan. Jangan dikejar!"

	} else if todayClose < ma20 {
		score = 1
		verdict = "💀 **PISAU JATUH (DOWNTREND)**\nAlasan: Harga sudah jebol dan ditutup di bawah MA20. Tren utama rusak. Hindari saham ini!"

	} else {
		score = 5
		verdict = "⚪ **TANGGUNG / SIDEWAYS**\nAlasan: Harga nanggung, tidak dekat support dan tidak terlalu pucuk. Skip cari saham lain."
	}

	return score, verdict, distToMA, ma20
}

// GetMarketFilterStatus mengecek tren IHSG saat ini
func GetMarketFilterStatus() (bool, string) {
	data, err := market.GetHistoricalPrices("^JKSE")
	if err != nil {
		return true, "⚠️ Gagal mengecek IHSG, asumsikan pasar normal."
	}

	var cleanPrices []float64
	for _, p := range data.Prices {
		if p > 0 {
			cleanPrices = append(cleanPrices, p)
		}
	}

	if len(cleanPrices) < 20 {
		return true, "⚠️ Data IHSG tidak lengkap, asumsikan pasar normal."
	}

	currentIHSG := cleanPrices[len(cleanPrices)-1]
	ma20IHSG := calculateMA(cleanPrices, 20)
	ma5IHSG := calculateMA(cleanPrices, 5)

	if currentIHSG < ma20IHSG {
		return false, fmt.Sprintf("🚨 **MARKET DOWNTREND / CRASH!** 🚨\nIHSG saat ini (%s) berada di bawah tren MA20 (%s).\n\n_Bot menyarankan: **CASH IS KING**. Jangan paksakan entry saat badai!_", utils.FormatRupiah(currentIHSG), utils.FormatRupiah(ma20IHSG))
	}

	if currentIHSG < ma5IHSG {
		return true, fmt.Sprintf("⚠️ **MARKET SEDANG KOREKSI WAJAR** ⚠️\nIHSG (%s) di bawah MA5, namun tren MA20 masih terjaga.\n\n_Status: Boleh trading, tapi kurangi agresivitas._", utils.FormatRupiah(currentIHSG), utils.FormatRupiah(ma5IHSG))
	}

	return true, fmt.Sprintf("🟢 **MARKET UPTREND (BULLISH)** 🟢\nIHSG (%s) berada kuat di atas MA5 dan MA20.\n\n_Status: Kondisi ideal untuk Fast Swing! Gas poll! 🚀_", utils.FormatRupiah(currentIHSG))
}