package main

import (
	"math/rand"
	"strings"
	"time"
)

type testMode string

const (
	modeWords testMode = "words"
	modeTime  testMode = "time"
)

type game struct {
	mode       testMode
	limit      int
	pool       []string
	words      []string
	wordIndex  int
	input      []rune
	startedAt  time.Time
	finishedAt time.Time
	keyStrokes int
	correct    int
	errors     int
	rng        *rand.Rand
}

func newGame(mode testMode, limit int, pool []string, seed int64) *game {
	g := &game{mode: mode, limit: limit, pool: pool, rng: rand.New(rand.NewSource(seed))}
	g.refill()
	return g
}

func (g *game) refill() {
	target := g.limit
	if g.mode == modeTime {
		target = 250
	}
	for len(g.words) < target {
		candidate := g.pool[g.rng.Intn(len(g.pool))]
		if len(g.pool) > 1 && len(g.words) > 0 && candidate == g.words[len(g.words)-1] {
			continue
		}
		g.words = append(g.words, candidate)
	}
}

func (g *game) start(now time.Time) {
	if g.startedAt.IsZero() {
		g.startedAt = now
	}
}

func (g *game) typeRune(r rune, now time.Time) {
	if g.done() || r < 32 || r == 127 {
		return
	}
	g.start(now)
	g.keyStrokes++
	target := []rune(g.words[g.wordIndex])
	if len(g.input) < len(target) && r == target[len(g.input)] {
		g.correct++
	} else {
		g.errors++
	}
	g.input = append(g.input, r)
	g.finishExactFinalWord(now)
}

func (g *game) backspace(now time.Time) {
	if g.done() || len(g.input) == 0 {
		return
	}
	g.input = g.input[:len(g.input)-1]
	g.finishExactFinalWord(now)
}

func (g *game) finishExactFinalWord(now time.Time) {
	if g.mode != modeWords || g.wordIndex != g.limit-1 || string(g.input) != g.currentWord() {
		return
	}
	g.wordIndex++
	g.input = nil
	g.finishedAt = now
}

func (g *game) submitWord(now time.Time) {
	if g.done() || len(g.input) == 0 {
		return
	}
	g.start(now)
	g.keyStrokes++
	if string(g.input) == g.words[g.wordIndex] {
		g.correct++ // the separating space
	} else {
		g.errors++
	}
	g.wordIndex++
	g.input = nil
	if g.mode == modeWords && g.wordIndex >= g.limit {
		g.finishedAt = now
	}
	if g.mode == modeTime && len(g.words)-g.wordIndex < 50 {
		for range 200 {
			candidate := g.pool[g.rng.Intn(len(g.pool))]
			g.words = append(g.words, candidate)
		}
	}
}

func (g *game) expire(now time.Time) {
	if g.mode == modeTime && !g.startedAt.IsZero() && !g.done() && now.Sub(g.startedAt) >= time.Duration(g.limit)*time.Second {
		g.finishedAt = g.startedAt.Add(time.Duration(g.limit) * time.Second)
	}
}

func (g *game) done() bool { return !g.finishedAt.IsZero() }

func (g *game) elapsed(now time.Time) time.Duration {
	if g.startedAt.IsZero() {
		return 0
	}
	if g.done() {
		return g.finishedAt.Sub(g.startedAt)
	}
	return now.Sub(g.startedAt)
}

func (g *game) wpm(now time.Time) float64 {
	minutes := g.elapsed(now).Minutes()
	if minutes <= 0 {
		return 0
	}
	return float64(g.correct) / 5 / minutes
}

func (g *game) rawWPM(now time.Time) float64 {
	minutes := g.elapsed(now).Minutes()
	if minutes <= 0 {
		return 0
	}
	return float64(g.keyStrokes) / 5 / minutes
}

func (g *game) accuracy() float64 {
	if g.keyStrokes == 0 {
		return 100
	}
	return float64(g.correct) / float64(g.keyStrokes) * 100
}

func (g *game) remaining(now time.Time) int {
	if g.mode == modeWords {
		return g.limit - g.wordIndex
	}
	left := time.Duration(g.limit)*time.Second - g.elapsed(now)
	if left <= 0 {
		return 0
	}
	return int((left + time.Second - 1) / time.Second)
}

func (g *game) currentWord() string {
	if g.wordIndex >= len(g.words) {
		return ""
	}
	return g.words[g.wordIndex]
}

func normalizeMode(value string) testMode { return testMode(strings.ToLower(value)) }
