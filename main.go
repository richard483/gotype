package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	yellow = lipgloss.Color("#E2B714")
	cream  = lipgloss.Color("#D1D0C5")
	dim    = lipgloss.Color("#646669")
	red    = lipgloss.Color("#CA4754")
	bg     = lipgloss.Color("#323437")

	titleStyle   = lipgloss.NewStyle().Foreground(yellow).Bold(true)
	dimStyle     = lipgloss.NewStyle().Foreground(dim)
	normalStyle  = lipgloss.NewStyle().Foreground(cream)
	correctStyle = lipgloss.NewStyle().Foreground(cream)
	errorStyle   = lipgloss.NewStyle().Foreground(red).Underline(true)
	caretStyle   = lipgloss.NewStyle().Background(yellow).Foreground(bg)
	metricStyle  = lipgloss.NewStyle().Foreground(yellow).Bold(true)
)

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

type keyMap struct {
	restart key.Binding
	quit    key.Binding
}

func (k keyMap) ShortHelp() []key.Binding  { return []key.Binding{k.restart, k.quit} }
func (k keyMap) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }

type model struct {
	game   *game
	mode   testMode
	limit  int
	pool   []string
	seed   int64
	width  int
	height int
	now    time.Time
	help   help.Model
	keys   keyMap
}

func newModel(mode testMode, limit int, pool []string, seed int64) model {
	keys := keyMap{
		restart: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "restart")),
		quit:    key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "quit")),
	}
	return model{game: newGame(mode, limit, pool, seed), mode: mode, limit: limit, pool: pool, seed: seed, now: time.Now(), help: help.New(), keys: keys}
}

func (m model) Init() tea.Cmd { return tick() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.help.Width = msg.Width
	case tickMsg:
		m.now = time.Time(msg)
		m.game.expire(m.now)
		return m, tick()
	case tea.KeyMsg:
		m.now = time.Now()
		if key.Matches(msg, m.keys.quit) {
			return m, tea.Quit
		}
		if key.Matches(msg, m.keys.restart) {
			m.seed++
			m.game = newGame(m.mode, m.limit, m.pool, m.seed)
			return m, nil
		}
		m.game.expire(m.now)
		if m.game.done() {
			if msg.String() == "r" {
				m.seed++
				m.game = newGame(m.mode, m.limit, m.pool, m.seed)
			}
			return m, nil
		}
		switch msg.Type {
		case tea.KeyBackspace, tea.KeyDelete:
			m.game.backspace(m.now)
		case tea.KeyEnter:
			m.game.submitWord(m.now)
		case tea.KeyRunes:
			for _, r := range msg.Runes {
				// Bubble Tea represents the spacebar as a printable rune on
				// terminals where it does not arrive as a dedicated key event.
				// Handling it here also makes pasted phrases advance correctly.
				if r == ' ' {
					m.game.submitWord(m.now)
				} else {
					m.game.typeRune(r, m.now)
				}
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return "loading..."
	}
	contentWidth := min(max(m.width-6, 30), 100)
	var body string
	if m.game.done() {
		body = m.results(contentWidth)
	} else {
		body = m.test(contentWidth)
	}
	content := lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render("gotype_"), "", body, "", dimStyle.Render(m.help.View(m.keys)))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content, lipgloss.WithWhitespaceBackground(bg))
}

func (m model) test(width int) string {
	statusLabel := "words"
	if m.mode == modeTime {
		statusLabel = "seconds"
	}
	status := fmt.Sprintf("%s %s    wpm %s    acc %s", metricStyle.Render(fmt.Sprint(m.game.remaining(m.now))), dimStyle.Render(statusLabel), metricStyle.Render(fmt.Sprintf("%.0f", m.game.wpm(m.now))), metricStyle.Render(fmt.Sprintf("%.0f%%", m.game.accuracy())))
	return lipgloss.JoinVertical(lipgloss.Left, status, "", m.renderWords(width), "", dimStyle.Render("start typing · space/enter next word · backspace correct"))
}

func (m model) renderWords(width int) string {
	lines := m.wordLines(width)
	currentLine := 0
	for lineIndex, line := range lines {
		if len(line) > 0 && m.game.wordIndex >= line[0] && m.game.wordIndex <= line[len(line)-1] {
			currentLine = lineIndex
			break
		}
	}

	visible := make([]string, 0, 3)
	for _, line := range lines[currentLine:min(currentLine+3, len(lines))] {
		words := make([]string, 0, len(line))
		for _, wordIndex := range line {
			words = append(words, m.renderWord(wordIndex))
		}
		visible = append(visible, strings.Join(words, " "))
	}
	return strings.Join(visible, "\n")
}

// wordLines assigns every word to a stable terminal row before applying ANSI
// styling. Advancing within a row therefore never reflows the remaining words.
func (m model) wordLines(width int) [][]int {
	lines := make([][]int, 0)
	lineWidth := 0
	for i, word := range m.game.words {
		wordWidth := utf8.RuneCountInString(word)
		if len(lines) == 0 || (lineWidth > 0 && lineWidth+1+wordWidth > width) {
			lines = append(lines, []int{})
			lineWidth = 0
		}
		lines[len(lines)-1] = append(lines[len(lines)-1], i)
		if lineWidth > 0 {
			lineWidth++
		}
		lineWidth += wordWidth
	}
	return lines
}

func (m model) renderWord(index int) string {
	target := []rune(m.game.words[index])
	if index < m.game.wordIndex {
		return dimStyle.Render(string(target))
	}
	if index > m.game.wordIndex {
		return dimStyle.Render(string(target))
	}
	typed := m.game.input
	var out strings.Builder
	for i, r := range target {
		if i < len(typed) {
			if typed[i] == r {
				out.WriteString(correctStyle.Render(string(r)))
			} else {
				out.WriteString(errorStyle.Render(string(typed[i])))
			}
		} else if i == len(typed) {
			out.WriteString(caretStyle.Render(string(r)))
		} else {
			out.WriteString(dimStyle.Render(string(r)))
		}
	}
	return out.String()
}

func (m model) results(width int) string {
	result := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(width/3).Render(dimStyle.Render("wpm\n")+metricStyle.Render(fmt.Sprintf("%.1f", m.game.wpm(m.now)))),
		lipgloss.NewStyle().Width(width/3).Render(dimStyle.Render("accuracy\n")+metricStyle.Render(fmt.Sprintf("%.1f%%", m.game.accuracy()))),
		lipgloss.NewStyle().Width(width/3).Render(dimStyle.Render("raw\n")+normalStyle.Render(fmt.Sprintf("%.1f", m.game.rawWPM(m.now)))),
	)
	details := fmt.Sprintf("characters %d/%d    errors %d    time %.1fs", m.game.correct, m.game.keyStrokes, m.game.errors, m.game.elapsed(m.now).Seconds())
	return lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render("test complete"), "", result, "", dimStyle.Render(details), "", normalStyle.Render("press r or tab to try again"))
}

type options struct {
	mode  testMode
	limit int
	seed  int64
}

func parseOptions(args []string, stderr io.Writer) (options, error) {
	fs := flag.NewFlagSet("gotype", flag.ContinueOnError)
	fs.SetOutput(stderr)
	modeFlag := fs.String("mode", "words", "test mode: words or time")
	wordsFlag := fs.Int("words", 25, "number of words in words mode")
	timeFlag := fs.Int("time", 30, "seconds in time mode")
	seedFlag := fs.Int64("seed", time.Now().UnixNano(), "random seed")
	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	mode := normalizeMode(*modeFlag)
	if mode != modeWords && mode != modeTime {
		return options{}, errors.New("--mode must be words or time")
	}
	limit := *wordsFlag
	if mode == modeTime {
		limit = *timeFlag
	}
	if limit <= 0 {
		return options{}, errors.New("test length must be greater than zero")
	}
	return options{mode: mode, limit: limit, seed: *seedFlag}, nil
}

func run(args []string) error {
	opts, err := parseOptions(args, os.Stderr)
	if err != nil {
		return err
	}
	words, err := loadWords()
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(newModel(opts.mode, opts.limit, words, opts.seed), tea.WithAltScreen()).Run()
	return err
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, "gotype:", err)
		os.Exit(1)
	}
}
