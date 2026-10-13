package shadow

import (
	"regexp"
	"strconv"
	"strings"

	"learn-go/internal/research"
)

var (
	recoRE = regexp.MustCompile(`(?i)REKOMENDASI[:\s\*]*\s*(BELI|TAHAN|HINDARI|NA)`)
	confRE = regexp.MustCompile(`(?i)AI Confidence[:\*]*\s*([0-9]+(?:\.[0-9]+)?)\s*%`)
)

// ParseAIReco extracts recommendation from Gemini output text.
// Returns "BELI", "TAHAN", "HINDARI", or "NA" if parsing fails.
func ParseAIReco(text string) string {
	m := recoRE.FindStringSubmatch(text)
	if len(m) < 2 {
		return "NA"
	}
	switch strings.ToUpper(m[1]) {
	case "BELI":
		return "BELI"
	case "TAHAN":
		return "TAHAN"
	case "HINDARI":
		return "HINDARI"
	default:
		return "NA"
	}
}

// ParseSentiment extracts sentiment score (0-10) from Gemini output text.
// Returns 0 if parsing fails.
func ParseSentiment(text string) float64 {
	return research.ExtractSentimentScore(text)
}

// ParseConfidence extracts AI confidence percentage from Gemini output text.
// Returns 0 if parsing fails.
func ParseConfidence(text string) float64 {
	m := confRE.FindStringSubmatch(text)
	if len(m) < 2 {
		return 0
	}
	val, _ := strconv.ParseFloat(m[1], 64)
	return val
}
