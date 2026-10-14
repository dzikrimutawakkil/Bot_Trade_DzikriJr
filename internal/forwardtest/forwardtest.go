package forwardtest

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"learn-go/internal/config"
	"learn-go/internal/market"
	"learn-go/internal/models"
	"learn-go/internal/rules"
	"learn-go/internal/signals"
)

var (
	ftMutex sync.RWMutex
	ftState models.FTState
)

const stateFile = "forward_test.json"

// IsEnabled returns whether forward test mode is active.
func IsEnabled() bool {
	ftMutex.RLock()
	defer ftMutex.RUnlock()
	return os.Getenv("FORWARD_TEST_MODE") == "true"
}

// Init loads state and parses env vars.
func Init() {
	ftMutex.Lock()
	defer ftMutex.Unlock()

	loadStateLocked()

	if ftState.StartEquity == 0 {
		modal := 1_000_000.0
		if m := os.Getenv("FORWARD_TEST_MODAL"); m != "" {
			if v, err := strconv.ParseFloat(m, 64); err == nil && v > 0 {
				modal = v
			}
		}
		ftState = models.FTState{
			StartDate:    time.Now().Format("2006-01-02"),
			StartEquity:  modal,
			Equity:       modal,
			PeakEquity:   modal,
			Positions:    make(map[string]models.FTPosition),
			ClosedTrades: make([]models.FTClosedTrade, 0),
		}
		saveStateLocked()
	}
}

// ProcessWeeklyEntry runs every Friday at 18:05.
// It selects top momentum candidates and simulates entry positions.
func ProcessWeeklyEntry() {
	if !IsEnabled() {
		return
	}
	log.Println("[FT] Processing weekly entry signal...")

	today := time.Now()

	allBars := make(map[string][]models.Bar)
	for _, sym := range config.Watchlist {
		bars, err := market.GetBars(sym, "3mo")
		if err != nil {
			continue
		}
		clean := market.CleanBars(bars)
		if len(clean) >= 50 {
			allBars[sym] = clean
		}
	}

	candidates := signals.SelectWeeklyCandidates(allBars, config.Watchlist, today, 4)
	if len(candidates) == 0 {
		log.Println("[FT] No candidates passed weekly filter.")
		return
	}

	for i := range candidates {
		p := market.GetLivePrice(candidates[i].Symbol)
		if p == 0 {
			p = candidates[i].Close
		}
		candidates[i].Close = p
	}

	ftMutex.Lock()
	defer ftMutex.Unlock()

	toClose := make([]string, 0)
	for sym := range ftState.Positions {
		found := false
		for _, c := range candidates {
			if c.Symbol == sym {
				found = true
				break
			}
		}
		if !found {
			toClose = append(toClose, sym)
		}
	}

	for _, sym := range toClose {
		closeTradeLocked(sym, candidates[0].Close, "REBALANCE")
	}

	available := ftState.Equity
	for _, p := range ftState.Positions {
		posValue := p.EntryPrice * float64(p.Lots) * 100 * (1 + rules.Default.BuyFee)
		available -= posValue
	}
	if available < 0 {
		available = 0
	}

	n := len(candidates)
	if n == 0 {
		return
	}
	alloc := available / float64(n)

	for _, c := range candidates {
		if _, has := ftState.Positions[c.Symbol]; has {
			continue
		}
		if c.Close <= 0 {
			continue
		}

		costPerLot := c.Close * 100
		lots := int(alloc / (costPerLot * (1 + rules.Default.BuyFee)))
		if lots < 1 {
			continue
		}

		ftState.Positions[c.Symbol] = models.FTPosition{
			Symbol:       c.Symbol,
			EntryPrice:   c.Close,
			HighestPrice: c.Close,
			Lots:         lots,
			BuyDate:      today.Format("2006-01-02"),
			HoldDays:     0,
			TP1Done:      false,
		}

		LogEntry(c.Symbol, c.Close, lots, today.Format("2006-01-02"))
		log.Printf("[FT] Entry: %s @ %.0f (%d lot)", c.Symbol, c.Close, lots)
	}

	saveStateLocked()
	log.Printf("[FT] Weekly entry done. Active positions: %d", len(ftState.Positions))
}

// EvaluateDaily runs every trading day at 16:20.
// It checks all open positions for exit conditions (CL, TSL, TP1, TIME).
// Also updates PeakEquity and MaxDD.
func EvaluateDaily() {
	if !IsEnabled() {
		return
	}

	ftMutex.Lock()
	defer ftMutex.Unlock()

	toClose := make(map[string]string)

	for sym, pos := range ftState.Positions {
		price := market.GetLivePrice(sym)
		if price == 0 {
			price = market.GetGooglePrice(sym)
		}
		if price == 0 {
			continue
		}

		if price > pos.HighestPrice {
			pos.HighestPrice = price
		}
		pos.HoldDays++
		ftState.Positions[sym] = pos

		if reason := evaluateExit(sym, pos, price); reason != "" {
			toClose[sym] = reason
		}
	}

	for sym, reason := range toClose {
		price := market.GetLivePrice(sym)
		if price == 0 {
			price = market.GetGooglePrice(sym)
		}
		closeTradeLocked(sym, price, reason)
	}

	updateEquityMetricsLocked()
	saveStateLocked()
}

// updateEquityMetricsLocked recalculates PeakEquity and MaxDD under mutex.
func updateEquityMetricsLocked() {
	var unrealized float64
	for sym, pos := range ftState.Positions {
		price := market.GetLivePrice(sym)
		if price == 0 {
			price = pos.EntryPrice
		}
		sellValue := price * float64(pos.Lots) * 100 * (1 - rules.Default.SellFee)
		buyCost := pos.EntryPrice * float64(pos.Lots) * 100 * (1 + rules.Default.BuyFee)
		unrealized += sellValue - buyCost
	}

	currentEquity := ftState.Equity + unrealized

	if currentEquity > ftState.PeakEquity {
		ftState.PeakEquity = currentEquity
	}

	if ftState.PeakEquity > 0 {
		dd := (ftState.PeakEquity - currentEquity) / ftState.PeakEquity * 100
		if dd > ftState.MaxDD {
			ftState.MaxDD = dd
		}
	}
}

// evaluateExit returns exit reason or "".
func evaluateExit(sym string, pos models.FTPosition, price float64) string {
	cl := rules.Default.DefaultCutLoss(pos.EntryPrice)
	if price <= cl {
		return "CL"
	}

	if rules.Default.TrailingActive(models.TradingPlan{
		EntryPrice:   pos.EntryPrice,
		HighestPrice: pos.HighestPrice,
	}) {
		tsl := rules.Default.TrailingStopPrice(pos.HighestPrice)
		if price <= tsl {
			return "TSL"
		}
	}

	tp1 := rules.Default.TP1Price(pos.EntryPrice)
	if !pos.TP1Done && price >= tp1 {
		sellLots := pos.Lots / 2
		if sellLots < 1 {
			sellLots = pos.Lots
		}

		sellValue := price * float64(sellLots) * 100 * (1 - rules.Default.SellFee)
		buyCost := pos.EntryPrice * float64(sellLots) * 100 * (1 + rules.Default.BuyFee)
		pnlRp := sellValue - buyCost
		pnlPct := rules.Default.NetPNL(pos.EntryPrice, price)

		trade := models.FTClosedTrade{
			Symbol:     sym,
			EntryPrice: pos.EntryPrice,
			ExitPrice:  price,
			Lots:       sellLots,
			EntryDate:  pos.BuyDate,
			ExitDate:   time.Now().Format("2006-01-02"),
			HoldDays:   pos.HoldDays,
			PNLPercent: pnlPct,
			PNLRp:      pnlRp,
			ExitReason: "TP1",
		}
		ftState.ClosedTrades = append(ftState.ClosedTrades, trade)
		ftState.Equity += pnlRp
		LogTrade(trade)

		pos.Lots -= sellLots
		pos.TP1Done = true

		if pos.Lots <= 0 {
			delete(ftState.Positions, sym)
		} else {
			ftState.Positions[sym] = pos
		}

		log.Printf("[FT] TP1 partial: %s @ %.0f (%.2f%%)", sym, price, pnlPct)
		return ""
	}

	if pos.HoldDays >= 20 {
		return "TIME"
	}

	return ""
}

// closeTradeLocked closes a position and updates equity.
func closeTradeLocked(sym string, price float64, reason string) {
	pos, ok := ftState.Positions[sym]
	if !ok {
		return
	}

	sellValue := price * float64(pos.Lots) * 100 * (1 - rules.Default.SellFee)
	buyCost := pos.EntryPrice * float64(pos.Lots) * 100 * (1 + rules.Default.BuyFee)
	pnlRp := sellValue - buyCost
	pnlPct := rules.Default.NetPNL(pos.EntryPrice, price)

	trade := models.FTClosedTrade{
		Symbol:     sym,
		EntryPrice: pos.EntryPrice,
		ExitPrice:  price,
		Lots:       pos.Lots,
		EntryDate:  pos.BuyDate,
		ExitDate:   time.Now().Format("2006-01-02"),
		HoldDays:   pos.HoldDays,
		PNLPercent: pnlPct,
		PNLRp:      pnlRp,
		ExitReason: reason,
	}

	ftState.ClosedTrades = append(ftState.ClosedTrades, trade)
	ftState.Equity += pnlRp

	delete(ftState.Positions, sym)

	LogTrade(trade)
	log.Printf("[FT] Exit: %s @ %.0f (%s) -> PnL: %.2f%% / Rp %.0f", sym, price, reason, pnlPct, pnlRp)
}

// GetState returns a copy of the current forward test state.
func GetState() models.FTState {
	ftMutex.RLock()
	defer ftMutex.RUnlock()
	return ftState
}

// FormatSummary returns a formatted Telegram message with the current summary.
func FormatSummary() string {
	ftMutex.RLock()
	defer ftMutex.RUnlock()

	state := ftState

	if state.StartEquity == 0 {
		return "📊 **FORWARD TEST**\n\nForward test belum dimulai. Aktifkan dengan `FORWARD_TEST_MODE=true`."
	}

	totalPnL := state.Equity - state.StartEquity
	totalPnLPct := (totalPnL / state.StartEquity) * 100

	var wins, total int
	var realizedPnL float64
	for _, t := range state.ClosedTrades {
		total++
		if t.PNLRp > 0 {
			wins++
		}
		realizedPnL += t.PNLRp
	}

	winRate := 0.0
	if total > 0 {
		winRate = float64(wins) / float64(total) * 100
	}

	var unrealized float64
	for sym, pos := range state.Positions {
		price := market.GetLivePrice(sym)
		if price == 0 {
			price = pos.EntryPrice
		}
		sellValue := price * float64(pos.Lots) * 100 * (1 - rules.Default.SellFee)
		buyCost := pos.EntryPrice * float64(pos.Lots) * 100 * (1 + rules.Default.BuyFee)
		unrealized += sellValue - buyCost
	}

	currentEquity := state.Equity + unrealized

	startDate, _ := time.Parse("2006-01-02", state.StartDate)
	weeks := int(time.Since(startDate).Hours() / 24 / 7)
	if weeks < 1 {
		weeks = 1
	}

	msg := fmt.Sprintf("📊 **FORWARD TEST — Week %d**\n", weeks)
	msg += "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
	msg += fmt.Sprintf("Modal Awal : Rp %.0f\n", state.StartEquity)
	msg += fmt.Sprintf("Equity     : Rp %.0f (%+.2f%%)\n", currentEquity, totalPnLPct)
	msg += fmt.Sprintf("Realized   : Rp %+.0f\n", realizedPnL)
	msg += fmt.Sprintf("Unrealized : Rp %+.0f\n", unrealized)
	msg += fmt.Sprintf("Win Rate   : %d/%d (%.1f%%)\n", wins, total, winRate)
	msg += fmt.Sprintf("Closed     : %d trades\n", len(state.ClosedTrades))
	msg += fmt.Sprintf("Max DD     : %.2f%%\n\n", state.MaxDD)

	if len(state.Positions) > 0 {
		msg += fmt.Sprintf("📁 **Posisi Aktif (%d):**\n", len(state.Positions))
		for sym, pos := range state.Positions {
			price := market.GetLivePrice(sym)
			if price == 0 {
				price = pos.EntryPrice
			}
			pnlPct := rules.Default.NetPNL(pos.EntryPrice, price)
			msg += fmt.Sprintf("  • %s: %.0f (%.2f%%, %d hari)\n", sym, price, pnlPct, pos.HoldDays)
		}
	} else {
		msg += "📁 **Posisi Aktif:** none\n"
	}

	if len(state.ClosedTrades) > 0 {
		msg += fmt.Sprintf("\n📈 **Recent Closed (%d):**\n", len(state.ClosedTrades))
		start := len(state.ClosedTrades) - 5
		if start < 0 {
			start = 0
		}
		for i := start; i < len(state.ClosedTrades); i++ {
			t := state.ClosedTrades[i]
			emoji := "🟢"
			if t.PNLRp < 0 {
				emoji = "🔴"
			}
			msg += fmt.Sprintf("%s %s: %.2f%% / Rp %+.0f (%s)\n", emoji, t.Symbol, t.PNLPercent, t.PNLRp, t.ExitReason)
		}
	}

	return msg
}

func loadStateLocked() {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		ftState = models.FTState{
			StartDate:    time.Now().Format("2006-01-02"),
			StartEquity:  1_000_000,
			Equity:       1_000_000,
			PeakEquity:   1_000_000,
			Positions:    make(map[string]models.FTPosition),
			ClosedTrades: make([]models.FTClosedTrade, 0),
		}
		return
	}

	var s models.FTState
	if err := json.Unmarshal(data, &s); err != nil {
		ftState = models.FTState{
			StartDate:    time.Now().Format("2006-01-02"),
			StartEquity:  1_000_000,
			Equity:       1_000_000,
			PeakEquity:   1_000_000,
			Positions:    make(map[string]models.FTPosition),
			ClosedTrades: make([]models.FTClosedTrade, 0),
		}
		return
	}

	if s.Positions == nil {
		s.Positions = make(map[string]models.FTPosition)
	}
	if s.ClosedTrades == nil {
		s.ClosedTrades = make([]models.FTClosedTrade, 0)
	}
	if s.PeakEquity == 0 {
		s.PeakEquity = s.Equity
	}
	ftState = s
}

func saveStateLocked() {
	data, err := json.MarshalIndent(ftState, "", "  ")
	if err != nil {
		log.Printf("[FT] Gagal marshal state: %v", err)
		return
	}

	tmp := stateFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		log.Printf("[FT] Gagal write tmp: %v", err)
		return
	}
	if err := os.Rename(tmp, stateFile); err != nil {
		log.Printf("[FT] Gagal rename: %v", err)
	}
}
