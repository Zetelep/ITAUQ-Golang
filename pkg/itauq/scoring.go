package itauq

import (
	"errors"
	"math"
	"sort"
)

// ErrEmptyScores is returned when a category or overall computation is asked
// for but no answered items were supplied for it.
var ErrEmptyScores = errors.New("itauq: no scores provided")

// CategoryAggregate holds the raw inputs needed to render one category score.
type CategoryAggregate struct {
	Category string
	Scores   []int // raw 1..ScaleMax scores for the 3 items in this category
}

// VariableScore implements the ITAUQ variable (per-item) score formula:
//
//	Skor Variabel = (x1 + x2 + x3) / 3
//
// where x1..x3 are the 3 Likert answers belonging to one item-variable
// (operationally: one category, since every ITAUQ item has exactly one
// variable = its category). The result is rounded to two decimals to match the
// 0.00–7.00 scale reported in the API.
func VariableScore(scores []int) (float64, error) {
	if len(scores) == 0 {
		return 0, ErrEmptyScores
	}
	var sum int
	for _, s := range scores {
		sum += s
	}
	return round2(float64(sum) / float64(len(scores))), nil
}

// CategoryScore averages all raw scores within one category (the mean of the
// three variables/items that share a category name).
func CategoryScore(agg CategoryAggregate) (float64, error) {
	return VariableScore(agg.Scores)
}

// NormalizedScore converts an average raw score on the 1..ScaleMax Likert
// scale into a 0..100 usability percentage using the standard 7-point formula:
//
//	normalized = (raw - 1) / (ScaleMax - 1) * 100
//
// The result is rounded to two decimals.
func NormalizedScore(raw float64, instrument Instrument) float64 {
	if instrument.ScaleMax <= instrument.ScaleMin {
		return 0
	}
	v := (raw - float64(instrument.ScaleMin)) / float64(instrument.ScaleMax-instrument.ScaleMin) * 100.0
	return round2(v)
}

// OverallScore computes the overall usability score (x-bar) as the mean of the
// category-level averages, and returns both the raw mean and the normalized
// 0..100 percentage. An empty input yields (0, 0, nil) — callers that want to
// distinguish "no answers at all" should check beforehand.
func OverallScore(perCategoryRaw map[string]float64, instrument Instrument) (rawAvg, normalized float64, ok bool) {
	if len(perCategoryRaw) == 0 {
		return 0, 0, false
	}
	// Stable ordering by category name so that the average is deterministic
	// across runs even though we only use the sum/count internally.
	keys := make([]string, 0, len(perCategoryRaw))
	for k := range perCategoryRaw {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sum float64
	for _, k := range keys {
		sum += perCategoryRaw[k]
	}
	rawAvg = round2(sum / float64(len(keys)))
	normalized = NormalizedScore(rawAvg, instrument)
	ok = true
	return
}

// CategoryStats bundles the raw and normalized scores for one category as
// returned by the evaluation endpoints.
type CategoryStats struct {
	Category         string  `json:"category"`
	AvgRawScore      float64 `json:"avg_raw_score"`
	NormalizedScore  float64 `json:"normalized_score"`
	AnsweredItemCount int    `json:"answered_item_count"`
}

// ComputeCategoryStats groups item scores by their instrument category and
// returns the per-category aggregates in stable, alphabetical order.
func ComputeCategoryStats(instrument *Loaded, scores map[int]int) []CategoryStats {
	byCat := make(map[string][]int)
	for itemID, score := range scores {
		cat, ok := instrument.ItemCategory(itemID)
		if !ok {
			continue
		}
		byCat[cat] = append(byCat[cat], score)
	}
	cats := make([]string, 0, len(byCat))
	for c := range byCat {
		cats = append(cats, c)
	}
	sort.Strings(cats)

	out := make([]CategoryStats, 0, len(cats))
	for _, c := range cats {
		raw, _ := VariableScore(byCat[c])
		out = append(out, CategoryStats{
			Category:          c,
			AvgRawScore:       raw,
			NormalizedScore:   NormalizedScore(raw, instrument.Instrument),
			AnsweredItemCount: len(byCat[c]),
		})
	}
	return out
}

func round2(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*100) / 100
}
