// Package sus loads the System Usability Scale (SUS) instrument from the
// embedded sus.json file. SUS is a fixed 10-item Likert questionnaire used to
// rate the evaluation website itself (not the app being evaluated).
package sus

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
)

//go:embed sus.json
var raw []byte

// Polarity indicates whether a SUS item contributes positively or negatively
// to the overall score. Mirrors the v_respondent_sus_score view's item_id % 2
// rule (odd = positive, even = negative), but we keep the field explicit so
// scoring code doesn't have to rely on parity alone.
type Polarity string

const (
	PolarityPositive Polarity = "positive"
	PolarityNegative Polarity = "negative"
)

// Question is a single SUS item.
type Question struct {
	ID       int      `json:"id"`
	Text     string   `json:"text"`
	Polarity Polarity `json:"polarity"`
}

// Instrument is the full SUS definition as exposed over the wire.
type Instrument struct {
	Version   string     `json:"version"`
	ScaleMin  int        `json:"scale_min"`
	ScaleMax  int        `json:"scale_max"`
	Questions []Question `json:"questions"`
}

// Loaded holds the parsed instrument together with a lookup map for fast
// (item_id) -> question validation when respondents submit answers.
type Loaded struct {
	Instrument
	byItem map[int]Question
}

var (
	once    sync.Once
	loaded  *Loaded
	loadErr error
)

// Load parses and returns the embedded instrument. Safe to call repeatedly;
// the JSON is parsed once on first call.
func Load() (*Loaded, error) {
	once.Do(func() {
		var meta struct {
			Metadata struct {
				ID       string `json:"id"`
				ScaleMin int    `json:"scaleMin"`
				ScaleMax int    `json:"scaleMax"`
			} `json:"metadata"`
			Questions []Question `json:"questions"`
		}
		if err := json.Unmarshal(raw, &meta); err != nil {
			loadErr = fmt.Errorf("parse sus.json: %w", err)
			return
		}
		byItem := make(map[int]Question, len(meta.Questions))
		for _, q := range meta.Questions {
			byItem[q.ID] = q
		}
		loaded = &Loaded{
			Instrument: Instrument{
				Version:   meta.Metadata.ID,
				ScaleMin:  meta.Metadata.ScaleMin,
				ScaleMax:  meta.Metadata.ScaleMax,
				Questions: meta.Questions,
			},
			byItem: byItem,
		}
	})
	if loadErr != nil {
		return nil, loadErr
	}
	return loaded, nil
}

// Item returns the question for a given item id. The second return value
// reports whether the id is part of the instrument.
func (l *Loaded) Item(itemID int) (Question, bool) {
	q, ok := l.byItem[itemID]
	return q, ok
}