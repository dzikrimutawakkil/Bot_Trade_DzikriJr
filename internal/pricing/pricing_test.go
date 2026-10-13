package pricing

import (
	"math"
	"testing"
)

func TestRoundToFraction(t *testing.T) {
	cases := []struct {
		input    float64
		expected float64
	}{
		{556.4, 555},
		{3481.6, 3480},
		{199.6, 200},
		{5012, 5000},
		{514.8, 515},
		{500, 500},
	}

	for _, c := range cases {
		got := RoundToFraction(c.input)
		if got != c.expected {
			t.Errorf("RoundToFraction(%v) = %v; want %v", c.input, got, c.expected)
		}
	}
}

func TestCalculateNetPNL(t *testing.T) {
	buyFee := 0.0015
	sellFee := 0.0025

	cases := []struct {
		entry     float64
		sell      float64
		expected  float64
		tolerance float64
	}{
		{535, 555, 3.32, 0.005},
		{535, 525, -2.26, 0.005},
		{3340, 3470, 3.48, 0.005},
		{3770, 3800, 0.39, 0.005},
	}

	for _, c := range cases {
		got := CalculateNetPNL(c.entry, c.sell, buyFee, sellFee)
		diff := math.Abs(got - c.expected)
		if diff > c.tolerance {
			t.Errorf("CalculateNetPNL(%v, %v, %.4f, %.4f) = %v; want ~%v (diff %v > tolerance %v)",
				c.entry, c.sell, buyFee, sellFee, got, c.expected, diff, c.tolerance)
		}
	}
}
