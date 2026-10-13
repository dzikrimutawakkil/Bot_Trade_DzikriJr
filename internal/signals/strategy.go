package signals

import (
	"errors"
	"learn-go/internal/models"
)

type Strategy interface {
	Name() string
	MinBars() int
	Evaluate(bars []models.Bar, ihsgSafe bool) Signal
}

var errUnknownStrategy = errors.New("unknown strategy")

func GetStrategy(name string) (Strategy, error) {
	switch name {
	case "cbow":
		return &CBoWStrategy{}, nil
	case "breakout20":
		return &breakout20Strategy{}, nil
	case "pullback20":
		return &pullback20Strategy{}, nil
	default:
		return nil, errUnknownStrategy
	}
}

type CBoWStrategy struct{}

func (s *CBoWStrategy) Name() string   { return "cbow" }
func (s *CBoWStrategy) MinBars() int   { return 60 }
func (s *CBoWStrategy) Evaluate(bars []models.Bar, ihsgSafe bool) Signal {
	if !ihsgSafe {
		return Signal{Score: 0, Code: "NONE", Low: bars[len(bars)-1].Low}
	}
	return Evaluate(bars, DefaultParams())
}
