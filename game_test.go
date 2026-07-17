package main

import (
	"math"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCorrectWordStats(t *testing.T) {
	g := newGame(modeWords, 1, []string{"go"}, 1)
	start := time.Unix(100, 0)
	g.typeRune('g', start)
	g.typeRune('o', start.Add(time.Second))

	if !g.done() {
		t.Fatal("game should finish on the final character")
	}
	if g.correct != 2 || g.keyStrokes != 2 || g.errors != 0 {
		t.Fatalf("stats = correct %d, keys %d, errors %d", g.correct, g.keyStrokes, g.errors)
	}
	if got := g.wpm(start.Add(time.Second)); math.Abs(got-24) > 0.001 {
		t.Fatalf("wpm = %v, want 24", got)
	}
}

func TestMistakesRemainInAccuracyAfterBackspace(t *testing.T) {
	g := newGame(modeWords, 1, []string{"go"}, 1)
	now := time.Unix(100, 0)
	g.typeRune('x', now)
	g.backspace(now)
	g.typeRune('g', now)
	g.typeRune('o', now)
	g.submitWord(now.Add(time.Second))

	if got := g.accuracy(); math.Abs(got-(2.0/3.0*100)) > 0.001 {
		t.Fatalf("accuracy = %v, want about 66.67", got)
	}
}

func TestTimedGameExpires(t *testing.T) {
	g := newGame(modeTime, 30, []string{"go", "type"}, 1)
	start := time.Unix(100, 0)
	g.typeRune('g', start)
	g.expire(start.Add(29 * time.Second))
	if g.done() {
		t.Fatal("game ended too early")
	}
	g.expire(start.Add(30 * time.Second))
	if !g.done() || g.remaining(start.Add(30*time.Second)) != 0 {
		t.Fatal("game should expire at 30 seconds")
	}
}

func TestNoConsecutiveInitialWords(t *testing.T) {
	g := newGame(modeWords, 100, []string{"a", "b"}, 1)
	for i := 1; i < len(g.words); i++ {
		if g.words[i] == g.words[i-1] {
			t.Fatalf("consecutive duplicate at %d", i)
		}
	}
}

func TestSingleWordPoolDoesNotStall(t *testing.T) {
	g := newGame(modeWords, 3, []string{"go"}, 1)
	if len(g.words) != 3 {
		t.Fatalf("generated %d words, want 3", len(g.words))
	}
}

func TestSpaceRuneAdvancesToNextWord(t *testing.T) {
	m := newModel(modeWords, 3, []string{"go", "find", "some"}, 1)
	// Pin the sequence so this is the exact phrase from the bug report. A
	// KeyRunes message also covers terminals that deliver spaces this way and
	// multi-word text pasted in a single event.
	m.game.words = []string{"go", "find", "some"}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("go find some ")})
	got := updated.(model)
	if got.game.wordIndex != 3 || !got.game.done() {
		t.Fatalf("word index = %d, done = %v; want index 3 and done", got.game.wordIndex, got.game.done())
	}
	if got.game.errors != 0 {
		t.Fatalf("errors = %d, want 0", got.game.errors)
	}
}

func TestWordRowsAreStableWithinLine(t *testing.T) {
	m := newModel(modeWords, 5, []string{"one"}, 1)
	m.game.words = []string{"one", "two", "three", "four", "five"}

	lines := m.wordLines(13)
	want := [][]int{{0, 1, 2}, {3, 4}}
	if len(lines) != len(want) {
		t.Fatalf("lines = %v, want %v", lines, want)
	}
	for i := range want {
		if len(lines[i]) != len(want[i]) {
			t.Fatalf("lines = %v, want %v", lines, want)
		}
		for j := range want[i] {
			if lines[i][j] != want[i][j] {
				t.Fatalf("lines = %v, want %v", lines, want)
			}
		}
	}
}
