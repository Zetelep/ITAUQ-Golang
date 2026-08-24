// Package itauq loads the Indonesian Tourism Application Usability Questionnaire
// instrument from the embedded itauq.json file and exposes helpers for validating
// respondent answers and rendering the questionnaire text for a given app name.
package itauq

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

//go:embed itauq.json
var raw []byte

// Question is a single ITAUQ item.
type Question struct {
	ID       int    `json:"id"`
	Category string `json:"category"`
	Text     string `json:"text"`
	MinLabel string `json:"minLabel"`
	MaxLabel string `json:"maxLabel"`
}

// Instrument is the full questionnaire definition.
type Instrument struct {
	Version  string     `json:"version"`
	ScaleMin int        `json:"scale_min"`
	ScaleMax int        `json:"scale_max"`
	Questions []Question `json:"questions"`
}

// Loaded holds the parsed instrument together with a lookup map for fast
// (item_id) -> category validation when respondents submit answers.
type Loaded struct {
	Instrument
	byItem map[int]Question
}

var (
	once    sync.Once
	loaded  *Loaded
	loadErr error
)

// Load parses and returns the embedded instrument. It is safe to call repeatedly;
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
			loadErr = fmt.Errorf("parse itauq.json: %w", err)
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

// RenderQuestion returns the question text with the {AppName} placeholder
// substituted by the supplied app name.
func (l *Loaded) RenderQuestion(q Question, appName string) Question {
	out := q
	out.Text = strings.ReplaceAll(q.Text, "{AppName}", appName)
	return out
}

// ItemCategory looks up the expected category for a given item id. The second
// return value reports whether the id is part of the instrument.
func (l *Loaded) ItemCategory(itemID int) (string, bool) {
	q, ok := l.byItem[itemID]
	return q.Category, ok
}