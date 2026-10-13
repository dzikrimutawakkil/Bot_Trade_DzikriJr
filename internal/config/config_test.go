package config

import (
	"strings"
	"testing"

	"learn-go/internal/models"
	"learn-go/internal/rules"
)

func TestLoadEnv_EmptyEnv_ReturnsErrorWithAllVariables(t *testing.T) {
	err := LoadEnv()
	if err == nil {
		t.Fatal("LoadEnv() expected error for empty env, got nil")
	}
	errMsg := err.Error()
	for _, name := range []string{"BOT_TOKEN", "MY_CHAT_ID", "GEMINI_API_KEY"} {
		if !strings.Contains(errMsg, name) {
			t.Errorf("LoadEnv() error should mention %q, got: %s", name, errMsg)
		}
	}
}

func TestCapitalInUse_TwoStocks(t *testing.T) {
	DataMutex.Lock()
	MyStocks["A"] = models.TradingPlan{EntryPrice: 1000, Lots: 2}
	MyStocks["B"] = models.TradingPlan{EntryPrice: 500, Lots: 10}
	DataMutex.Unlock()

	t.Cleanup(func() {
		DataMutex.Lock()
		delete(MyStocks, "A")
		delete(MyStocks, "B")
		DataMutex.Unlock()
	})

	got := CapitalInUse()
	// 1000*2*100*(1+0.0015) + 500*10*100*(1+0.0015) = 200300 + 500750 = 701050
	expected := 701050.0
	diff := got - expected
	if diff < 0 {
		diff = -diff
	}
	if diff > 1 {
		t.Errorf("CapitalInUse() = %v; want ~%v (diff %v > tolerance 1)", got, expected, diff)
	}
}

func TestUpdatePlan(t *testing.T) {
	DataMutex.Lock()
	MyStocks["A"] = models.TradingPlan{EntryPrice: 100, Lots: 1, HighestPrice: 50}
	DataMutex.Unlock()

	t.Cleanup(func() {
		DataMutex.Lock()
		delete(MyStocks, "A")
		DataMutex.Unlock()
	})

	t.Run("key not found → return false", func(t *testing.T) {
		ok := UpdatePlan("MISSING", func(p *models.TradingPlan) {
			t.Error("fn should not be called when key not found")
		})
		if ok {
			t.Error("UpdatePlan should return false for missing key")
		}
	})

	t.Run("key found → fn applied and map updated", func(t *testing.T) {
		ok := UpdatePlan("A", func(p *models.TradingPlan) {
			p.HighestPrice = 200
		})
		if !ok {
			t.Fatal("UpdatePlan should return true for existing key")
		}
		DataMutex.RLock()
		if MyStocks["A"].HighestPrice != 200 {
			t.Errorf("HighestPrice = %v; want 200", MyStocks["A"].HighestPrice)
		}
		DataMutex.RUnlock()
	})

	t.Run("map unchanged when key not found", func(t *testing.T) {
		DataMutex.RLock()
		beforeLen := len(MyStocks)
		DataMutex.RUnlock()

		UpdatePlan("MISSING", func(p *models.TradingPlan) {})
		_ = beforeLen

		DataMutex.RLock()
		afterLen := len(MyStocks)
		DataMutex.RUnlock()

		if afterLen != 1 {
			t.Errorf("Map should still have 1 entry, got %d", afterLen)
		}
	})
}

func TestUpdateOrder(t *testing.T) {
	DataMutex.Lock()
	PendingOrders["B"] = models.ActiveOrder{Symbol: "B", OrderPrice: 500, Lot: 5}
	DataMutex.Unlock()

	t.Cleanup(func() {
		DataMutex.Lock()
		delete(PendingOrders, "B")
		DataMutex.Unlock()
	})

	t.Run("key not found → return false", func(t *testing.T) {
		ok := UpdateOrder("MISSING", func(o *models.ActiveOrder) {})
		if ok {
			t.Error("UpdateOrder should return false for missing key")
		}
	})

	t.Run("key found → fn applied", func(t *testing.T) {
		ok := UpdateOrder("B", func(o *models.ActiveOrder) {
			o.OrderPrice = 600
		})
		if !ok {
			t.Fatal("UpdateOrder should return true for existing key")
		}
		DataMutex.RLock()
		if PendingOrders["B"].OrderPrice != 600 {
			t.Errorf("OrderPrice = %v; want 600", PendingOrders["B"].OrderPrice)
		}
		DataMutex.RUnlock()
	})
}

func TestBuyFeeInRules(t *testing.T) {
	if rules.Default.BuyFee != 0.0015 {
		t.Errorf("rules.Default.BuyFee = %v; want 0.0015", rules.Default.BuyFee)
	}
	if rules.Default.SellFee != 0.0025 {
		t.Errorf("rules.Default.SellFee = %v; want 0.0025", rules.Default.SellFee)
	}
}
