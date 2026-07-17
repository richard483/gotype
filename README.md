# gotype

A minimal Monkeytype-inspired typing test for the terminal, written in Go.

## Run

```sh
go run .
```

Word mode is the default (`25` words). You can also choose a timed test:

```sh
go run . --mode words --words 50
go run . --mode time --time 60
```

Build a standalone binary with `go build -o gotype .` (use `gotype.exe` on Windows).

While testing, type the highlighted text, press space or enter between words, use
backspace to correct, tab to restart, and escape to quit. Timing begins on the
first printable character.

## Word list and license

The embedded English word list comes from Monkeytype's canonical
[`english.json`](https://github.com/monkeytypegame/monkeytype/blob/master/frontend/static/languages/english.json).
Monkeytype is licensed under GPL-3.0. This project is therefore distributed under
GPL-3.0 as well; see [LICENSE](LICENSE).
