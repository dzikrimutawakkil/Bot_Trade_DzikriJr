package rules

import (
	"learn-go/internal/models"
	"learn-go/internal/pricing"
)

type Rules struct {
	Strategy                 string
	BuyFee, SellFee          float64
	DefaultCutLossPct        float64
	StructuralCLBufferPct    float64
	BuyZoneBelowPct          float64
	TP1Pct                   float64
	TP1SellFraction          float64
	TrailingStopPct          float64
	TrailingActivationNetPct float64
	RunawayPct               float64
	MaxRiskPerTradePct       float64
	DefensiveRiskDivisor     float64
	MinPurchaseAmount        float64
}

var Default = Rules{
	Strategy:                 "C-BoW",
	BuyFee:                   0.0015,
	SellFee:                  0.0025,
	DefaultCutLossPct:        3.0,
	StructuralCLBufferPct:    1.0,
	BuyZoneBelowPct:          1.0,
	TP1Pct:                   5.0,
	TP1SellFraction:          0.5,
	TrailingStopPct:          3.0,
	TrailingActivationNetPct: 2.0,
	RunawayPct:               3.0,
	MaxRiskPerTradePct:       5.0,
	DefensiveRiskDivisor:     2.0,
	MinPurchaseAmount:        250000.0,
}

func (r Rules) DefaultCutLoss(entry float64) float64 {
	return pricing.RoundToFraction(entry * (1 - r.DefaultCutLossPct/100))
}

func (r Rules) StructuralCutLoss(signalLow float64) float64 {
	return pricing.RoundToFraction(signalLow * (1 - r.StructuralCLBufferPct/100))
}

func (r Rules) TP1Price(entry float64) float64 {
	return pricing.RoundToFraction(entry * (1 + r.TP1Pct/100))
}

func (r Rules) TrailingStopPrice(highest float64) float64 {
	return pricing.RoundToFraction(highest * (1 - r.TrailingStopPct/100))
}

func (r Rules) NetPNL(entry, price float64) float64 {
	return pricing.CalculateNetPNL(entry, price, r.BuyFee, r.SellFee)
}

func (r Rules) TrailingActive(p models.TradingPlan) bool {
	hp := p.HighestPrice
	if hp == 0 {
		hp = p.EntryPrice
	}
	return r.NetPNL(p.EntryPrice, hp) >= r.TrailingActivationNetPct
}

func (r Rules) EffectiveStop(p models.TradingPlan) float64 {
	if r.TrailingActive(p) {
		tsl := r.TrailingStopPrice(p.HighestPrice)
		if tsl > p.CutLoss {
			return tsl
		}
	}
	return p.CutLoss
}

type Decision int

const (
	None Decision = iota
	CutLoss
	TrailingStop
	TakeProfit1
)

func (r Rules) Evaluate(p models.TradingPlan, price float64) Decision {
	highest := p.HighestPrice
	if highest == 0 {
		highest = p.EntryPrice
	}

	if price <= p.CutLoss {
		return CutLoss
	}

	if r.TrailingActive(p) && price <= r.TrailingStopPrice(highest) {
		return TrailingStop
	}

	if !p.TP1Done && p.TakeProfit > 0 && price >= p.TakeProfit {
		return TakeProfit1
	}

	return None
}
