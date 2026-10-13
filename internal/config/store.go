package config

import (
	"learn-go/internal/models"
)

// UpdatePlan applies fn to the plan at key if it exists.
// Returns true if the key was found and fn was applied; false otherwise.
// Thread-safe: acquires write lock for the duration.
func UpdatePlan(symbol string, fn func(p *models.TradingPlan)) bool {
	DataMutex.Lock()
	defer DataMutex.Unlock()
	plan, ok := MyStocks[symbol]
	if !ok {
		return false
	}
	fn(&plan)
	MyStocks[symbol] = plan
	return true
}

// UpdateOrder applies fn to the order at key if it exists.
// Returns true if the key was found and fn was applied; false otherwise.
// Thread-safe: acquires write lock for the duration.
func UpdateOrder(symbol string, fn func(o *models.ActiveOrder)) bool {
	DataMutex.Lock()
	defer DataMutex.Unlock()
	order, ok := PendingOrders[symbol]
	if !ok {
		return false
	}
	fn(&order)
	PendingOrders[symbol] = order
	return true
}
