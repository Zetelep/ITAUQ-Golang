package sus

import (
	"errors"
	"math"
	"sort"
)

// ErrIncompleteSUS is returned when fewer than 10 item scores are supplied.
var ErrIncompleteSUS = errors.New("sus: expected 10 answered items")

// ItemContribution is the per-item contribution to the final SUS score, kept
// for debugging / reporting purposes. Positive-polarity items contribute
// (score - 1); negative-polarity items contribute (5 - score).
type ItemContribution struct {
	ItemID       int     `json:"item_id"`
	Polarity     Polarity `json:"polarity"`
	RawScore     int     `json:"raw_score"`
	Contribution float64 `json:"contribution"`
}

// ComputeScore implements the standard System Usability Scale formula:
//
//	contribution = score - 1   for positive items
//	contribution = 5 - score   for negative items
//	final = sum(contributions) * 2.5
//
// The final score is rounded to two decimals (the conventional SUS
// presentation) and lies on a 0..100 scale. Returns ErrIncompleteSUS if the
// input has fewer than 10 items; missing items are simply ignored.
func ComputeScore(instrument *Loaded, scores map[int]int) (float64, []ItemContribution, error) {
	if len(scores) < len(instrument.Questions) {
		return 0, nil, ErrIncompleteSUS
	}
	keys := make([]int, 0, len(scores))
	for k := range scores {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	var sum float64
	parts := make([]ItemContribution, 0, len(keys))
	for _, id := range keys {
		q, ok := instrument.Item(id)
		if !ok {
			continue
		}
		raw := scores[id]
		var contrib float64
		switch q.Polarity {
		case PolarityPositive:
			contrib = float64(raw - 1)
		case PolarityNegative:
			contrib = float64(5 - raw)
		default:
			contrib = 0
		}
		sum += contrib
		parts = append(parts, ItemContribution{
			ItemID:       id,
			Polarity:     q.Polarity,
			RawScore:     raw,
			Contribution: round2(contrib),
		})
	}
	return round2(sum * 2.5), parts, nil
}

func round2(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*100) / 100
}
