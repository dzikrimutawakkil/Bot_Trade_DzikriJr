package shadow

import (
	"fmt"
	"sort"
	"strings"
)

// GroupKey groups rows by a combination of ai_reco and code.
type GroupKey struct {
	AIReco string
	Code   string
}

// GroupStats holds aggregated statistics for a group.
type GroupStats struct {
	Key        GroupKey
	N          int
	WinRate3D  float64
	AvgRet3D   float64
	AvgMaxDD5D float64
	AvgRet1D   float64
	AvgRet5D   float64
	AvgMaxUp5D float64
}

// Summary holds the full shadow log summary.
type Summary struct {
	FilledRows           int
	DateRange            string
	Groups               []GroupStats
	CompareBELIvsHindari float64
	SmallSampleWarning   bool
}

// Summarize computes a summary from filled rows.
// Fee round-trip = BuyFee + SellFee (0.40%).
func Summarize(rows []Row) Summary {
	var filled []Row
	var minDate, maxDate string
	first := true
	for _, r := range rows {
		if !r.Filled {
			continue
		}
		filled = append(filled, r)
		if first {
			minDate = r.Date
			maxDate = r.Date
			first = false
		} else {
			if r.Date < minDate {
				minDate = r.Date
			}
			if r.Date > maxDate {
				maxDate = r.Date
			}
		}
	}

	sort.Slice(filled, func(i, j int) bool {
		return filled[i].Date < filled[j].Date
	})

	groups := make(map[GroupKey]*GroupStats)
	compareBELI := []float64{}
	compareHindari := []float64{}

	for _, r := range filled {
		key := GroupKey{AIReco: r.AIReco, Code: r.Code}
		if _, ok := groups[key]; !ok {
			groups[key] = &GroupStats{Key: key}
		}
		g := groups[key]
		g.N++
		g.AvgRet3D += r.Ret3D
		g.AvgMaxDD5D += r.MaxDD5D
		g.AvgRet1D += r.Ret1D
		g.AvgRet5D += r.Ret5D
		g.AvgMaxUp5D += r.MaxUp5D
		if NetReturn(r.Ret3D) > 0 {
			g.WinRate3D++
		}

		isHammerGreen := r.Code == "HAMMER" || r.Code == "GREEN_BOUNCE"
		if isHammerGreen {
			nr3d := NetReturn(r.Ret3D)
			if r.AIReco == "BELI" {
				compareBELI = append(compareBELI, nr3d)
			} else if r.AIReco == "HINDARI" {
				compareHindari = append(compareHindari, nr3d)
			}
		}
	}

	var result []GroupStats
	smallSample := false
	for _, g := range groups {
		if g.N > 0 {
			g.AvgRet3D = round(g.AvgRet3D/float64(g.N), 2)
			g.AvgMaxDD5D = round(g.AvgMaxDD5D/float64(g.N), 2)
			g.AvgRet1D = round(g.AvgRet1D/float64(g.N), 2)
			g.AvgRet5D = round(g.AvgRet5D/float64(g.N), 2)
			g.AvgMaxUp5D = round(g.AvgMaxUp5D/float64(g.N), 2)
			g.WinRate3D = round(g.WinRate3D/float64(g.N)*100, 1)
		}
		result = append(result, *g)
		if g.N < 30 {
			smallSample = true
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Key.AIReco != result[j].Key.AIReco {
			return result[i].Key.AIReco < result[j].Key.AIReco
		}
		return result[i].N > result[j].N
	})

	avgBELI := avg(compareBELI)
	avgHindari := avg(compareHindari)
	compareVal := round(avgBELI-avgHindari, 2)

	return Summary{
		FilledRows:           len(filled),
		DateRange:            minDate + " → " + maxDate,
		Groups:               result,
		CompareBELIvsHindari: compareVal,
		SmallSampleWarning:   smallSample,
	}
}

// Format builds a readable Telegram message from a Summary.
func (s Summary) Format() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("📊 *Shadow Log Summary*\n"))
	sb.WriteString(fmt.Sprintf("Sinyal filled: %d\n", s.FilledRows))
	sb.WriteString(fmt.Sprintf("Rentang: %s\n\n", s.DateRange))

	if s.FilledRows == 0 {
		sb.WriteString("_Belum ada data filled._\n")
		return sb.String()
	}

	sb.WriteString("*Tabel per ai_reco + code:*\n")
	sb.WriteString("```\n")
	sb.WriteString(fmt.Sprintf("%-6s %-12s %4s %6s %7s %7s\n",
		"ai_reco", "code", "n", "wr_3d%", "avg_ret3d", "avg_dd5d"))
	sb.WriteString("------------------------------------------------\n")
	for _, g := range s.Groups {
		sb.WriteString(fmt.Sprintf("%-6s %-12s %4d %6.1f%% %7.2f%% %7.2f%%\n",
			g.Key.AIReco, g.Key.Code, g.N, g.WinRate3D, g.AvgRet3D, g.AvgMaxDD5D))
	}
	sb.WriteString("```\n")

	if s.CompareBELIvsHindari != 0 || len(s.Groups) > 0 {
		sb.WriteString(fmt.Sprintf("\n*Perbandingan BELI vs HINDARI (HAMMER+GREEN_BOUNCE):*\n"))
		sb.WriteString(fmt.Sprintf("Avg net ret_3d BELI - HINDARI: %+.2f%%\n", s.CompareBELIvsHindari))
	}

	if s.SmallSampleWarning {
		sb.WriteString("\n⚠️ *Sampel kecil (n < 30), jangan disimpulkan.*\n")
	}

	return sb.String()
}

func round(v float64, prec int) float64 {
	m := 1.0
	for i := 0; i < prec; i++ {
		m *= 10
	}
	return float64(int(v*m+0.5)) / m
}

func avg(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	s := 0.0
	for _, v := range vals {
		s += v
	}
	return s / float64(len(vals))
}
