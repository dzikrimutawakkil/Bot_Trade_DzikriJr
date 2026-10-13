package research

import (
	"fmt"

	"learn-go/internal/config"
	"learn-go/internal/rules"
)

func buildPlanText(symbol string, in rules.PlanInput, out rules.PlanOutput, withBuyCommand bool) string {
	if len(out.Notes) > 0 && out.MaxLots == 0 {
		// CL invalid — tampilkan notes tanpa /buy
		header := fmt.Sprintf("━━━━━━━━━━━━━━━━━━\n📐 **TRADING PLAN**\n━━━━━━━━━━━━━━━━━━\n📍 **Harga:** Rp %.0f\n🧱 **Support Ekor:** Rp %.0f\n", in.CurrentPrice, in.SignalLow)
		noteLine := ""
		for _, n := range out.Notes {
			noteLine += "\n⚠️ " + n
		}
		header += noteLine + "\n━━━━━━━━━━━━━━━━━━\n"
		return header
	}

	tp1Fraction := int(rules.Default.TP1SellFraction * 100)
	tsPct := rules.Default.TrailingStopPct

	text := fmt.Sprintf(`━━━━━━━━━━━━━━━━━━
📐 **TRADING PLAN (Max %d LOT)**
📍 **Harga Terkonfirmasi** : Rp %.0f
🧱 **Support Ekor (Low)** : Rp %.0f

🎯 **AREA BELI : Rp %.0f - Rp %.0f**
_(Sinyal Valid! Hajar Kanan sebagian sebelum market tutup)_

🚨 **Batas Cut Loss** : Rp %.0f (Jebol Ekor Bawah)
🚀 **TP 1 (Jual %d%% Lot)** : Rp %.0f (Amankan Modal)
🛡️ **TP 2 (Sisa %d%% Lot)** : Trailing Stop %.0f%%
━━━━━━━━━━━━━━━━━━`,
		out.MaxLots, in.CurrentPrice, in.SignalLow,
		out.BuyMin, out.BuyMax, out.CutLoss,
		tp1Fraction, out.TP1, 100-tp1Fraction, tsPct)

	if withBuyCommand && out.MaxLots > 0 {
		text += fmt.Sprintf("\n\n👉 Ketik /buy %s %.0f %d %.0f jika ingin eksekusi.",
			symbol, in.CurrentPrice, out.MaxLots, out.CutLoss)
	}

	return text
}

func buildResearchPlanText(symbol string, currentPrice float64, signalLow float64, ma20 float64) (string, rules.PlanOutput, bool) {
	if currentPrice <= 0 {
		return "\n\n⚠️ **WARNING: DATA HARGA OFFLINE**\n_(Kalkulator Position Sizing tidak dapat ditampilkan karena bot gagal menarik harga real-time dari bursa saat ini. Silakan coba lagi nanti.)_\n", rules.PlanOutput{}, false
	}

	if signalLow <= 0 {
		signalLow = ma20 * 0.98
	}

	isMarketSafe, _ := GetMarketFilterStatus()

	in := rules.PlanInput{
		CurrentPrice: currentPrice,
		SignalLow:    signalLow,
		TotalCapital: config.TotalModalTrading,
		CapitalInUse: config.CapitalInUse(),
		MarketSafe:   isMarketSafe,
	}

	out := rules.Default.SuggestPlan(in)

	return "", out, true
}
