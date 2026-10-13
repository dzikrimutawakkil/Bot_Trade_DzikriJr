package rules

import (
	"math"

	"learn-go/internal/pricing"
)

type PlanInput struct {
	CurrentPrice float64
	SignalLow    float64
	TotalCapital float64
	CapitalInUse float64
	MarketSafe   bool
}

type PlanOutput struct {
	BuyMin, BuyMax, CutLoss, TP1, LossPerLot float64
	MaxLotsByRisk, MaxLotsByCash, MaxLots    int
	Notes                                    []string
}

func (r Rules) SuggestPlan(in PlanInput) PlanOutput {
	out := PlanOutput{}

	out.CutLoss = r.StructuralCutLoss(in.SignalLow)
	out.BuyMin = pricing.RoundToFraction(in.CurrentPrice * (1 - r.BuyZoneBelowPct/100))
	out.BuyMax = pricing.RoundToFraction(in.CurrentPrice)
	out.TP1 = r.TP1Price(in.CurrentPrice)

	out.LossPerLot = (in.CurrentPrice - out.CutLoss) * 100

	if out.LossPerLot <= 0 {
		out.MaxLots = 0
		out.Notes = append(out.Notes, "CL di atas/sama dengan harga, tidak ada setup valid")
		return out
	}

	riskBudget := in.TotalCapital * r.MaxRiskPerTradePct / 100
	if !in.MarketSafe {
		riskBudget = riskBudget / r.DefensiveRiskDivisor
	}
	out.MaxLotsByRisk = int(math.Floor(riskBudget / out.LossPerLot))

	hargaSatuLot := in.CurrentPrice * 100 * (1 + r.BuyFee)
	sisaModal := in.TotalCapital - in.CapitalInUse
	if hargaSatuLot > 0 {
		out.MaxLotsByCash = int(math.Floor(sisaModal / hargaSatuLot))
	}

	out.MaxLots = out.MaxLotsByRisk
	if out.MaxLotsByCash < out.MaxLots {
		out.MaxLots = out.MaxLotsByCash
	}
	if out.MaxLots < 0 {
		out.MaxLots = 0
	}

	return out
}
