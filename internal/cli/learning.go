package cli

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/MiaoDX/verse-driven/internal/packs"
	"github.com/MiaoDX/verse-driven/internal/schema"
)

type learningState struct {
	Version int                     `json:"version"`
	Cards   map[string]learningCard `json:"cards"`
}

type learningCard struct {
	Repetitions  int       `json:"repetitions"`
	IntervalDays int       `json:"interval_days"`
	EaseFactor   float64   `json:"ease_factor"`
	Due          time.Time `json:"due"`
	LastSeen     time.Time `json:"last_seen,omitempty"`
}

type userConfig struct {
	Version         int  `json:"version"`
	LearningEnabled bool `json:"learning_enabled"`
}

func userConfigPath(home string) string {
	return filepath.Join(home, ".config", "scripture-mcp", "config.json")
}

func readUserConfig(home string) (userConfig, bool, error) {
	path := userConfigPath(home)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return userConfig{Version: 1}, false, nil
		}
		return userConfig{}, false, fmt.Errorf("read user config: %w", err)
	}
	var cfg userConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return userConfig{}, true, fmt.Errorf("parse user config: %w", err)
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	return cfg, true, nil
}

func writeUserConfig(home string, cfg userConfig) error {
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	path := userConfigPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir user config: %w", err)
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode user config: %w", err)
	}
	b = append(b, '\n')
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("write user config: %w", err)
	}
	return nil
}

func learningEnabledFromConfig(s Streams) (bool, error) {
	home, err := homeDir(s)
	if err != nil {
		return false, err
	}
	cfg, exists, err := readUserConfig(home)
	if err != nil {
		return false, err
	}
	return exists && cfg.LearningEnabled, nil
}

func pickLearningRecapVerse(tradition string, seed int64, s Streams) (schema.Verse, bool, error) {
	home, err := homeDir(s)
	if err != nil {
		return schema.Verse{}, false, err
	}
	path := learningStatePath(home)
	state, err := readLearningState(path)
	if err != nil {
		return schema.Verse{}, false, err
	}
	pool := recapPool(tradition)
	if len(pool) == 0 {
		return schema.Verse{}, false, nil
	}
	now := time.Now().UTC()
	v := selectLearningVerse(pool, state, now, seed)
	card := state.Cards[v.ID]
	state.Cards[v.ID] = advanceSM2(card, now, 4)
	if err := writeLearningState(path, state); err != nil {
		return schema.Verse{}, false, err
	}
	return v, true, nil
}

func learningStatePath(home string) string {
	return filepath.Join(home, ".config", "scripture-mcp", "learning.json")
}

func readLearningState(path string) (learningState, error) {
	state := learningState{Version: 1, Cards: make(map[string]learningCard)}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return state, fmt.Errorf("read learning state: %w", err)
	}
	if len(b) == 0 {
		return state, nil
	}
	if err := json.Unmarshal(b, &state); err != nil {
		return state, fmt.Errorf("parse learning state: %w", err)
	}
	if state.Version == 0 {
		state.Version = 1
	}
	if state.Cards == nil {
		state.Cards = make(map[string]learningCard)
	}
	return state, nil
}

func writeLearningState(path string, state learningState) error {
	if state.Version == 0 {
		state.Version = 1
	}
	if state.Cards == nil {
		state.Cards = make(map[string]learningCard)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir learning state: %w", err)
	}
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode learning state: %w", err)
	}
	b = append(b, '\n')
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("write learning state: %w", err)
	}
	return nil
}

func recapPool(tradition string) []schema.Verse {
	var pool []schema.Verse
	r := packs.All()
	for _, name := range r.Names() {
		p := r.Pack(name)
		if p.Meta.InclusionMode != "" && p.Meta.InclusionMode != "bundled" {
			continue
		}
		if tradition != "" && p.Meta.Tradition != tradition {
			continue
		}
		pool = append(pool, p.Verses()...)
	}
	sort.Slice(pool, func(i, j int) bool { return pool[i].ID < pool[j].ID })
	return pool
}

func selectLearningVerse(pool []schema.Verse, state learningState, now time.Time, seed int64) schema.Verse {
	byID := make(map[string]schema.Verse, len(pool))
	for _, v := range pool {
		byID[v.ID] = v
	}
	var dueIDs []string
	for id, card := range state.Cards {
		if _, ok := byID[id]; ok && !card.Due.After(now) {
			dueIDs = append(dueIDs, id)
		}
	}
	if len(dueIDs) > 0 {
		sort.Slice(dueIDs, func(i, j int) bool {
			a, b := state.Cards[dueIDs[i]], state.Cards[dueIDs[j]]
			if !a.Due.Equal(b.Due) {
				return a.Due.Before(b.Due)
			}
			return dueIDs[i] < dueIDs[j]
		})
		return byID[dueIDs[0]]
	}

	var unseen []schema.Verse
	for _, v := range pool {
		if _, ok := state.Cards[v.ID]; !ok {
			unseen = append(unseen, v)
		}
	}
	if len(unseen) > 0 {
		if seed == 0 {
			seed = now.UnixNano()
		}
		rng := rand.New(rand.NewSource(seed))
		return unseen[rng.Intn(len(unseen))]
	}

	sort.Slice(pool, func(i, j int) bool {
		a, b := state.Cards[pool[i].ID], state.Cards[pool[j].ID]
		if !a.Due.Equal(b.Due) {
			return a.Due.Before(b.Due)
		}
		return pool[i].ID < pool[j].ID
	})
	return pool[0]
}

func advanceSM2(card learningCard, now time.Time, quality int) learningCard {
	if card.EaseFactor == 0 {
		card.EaseFactor = 2.5
	}
	if quality < 0 {
		quality = 0
	}
	if quality > 5 {
		quality = 5
	}
	if quality < 3 {
		card.Repetitions = 0
		card.IntervalDays = 1
	} else {
		card.Repetitions++
		switch card.Repetitions {
		case 1:
			card.IntervalDays = 1
		case 2:
			card.IntervalDays = 6
		default:
			card.IntervalDays = int(math.Round(float64(card.IntervalDays) * card.EaseFactor))
			if card.IntervalDays < 1 {
				card.IntervalDays = 1
			}
		}
	}
	delta := 5 - quality
	card.EaseFactor = card.EaseFactor + (0.1 - float64(delta)*(0.08+float64(delta)*0.02))
	if card.EaseFactor < 1.3 {
		card.EaseFactor = 1.3
	}
	card.LastSeen = now
	card.Due = now.AddDate(0, 0, card.IntervalDays)
	return card
}
