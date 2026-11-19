package editor

import "slices"

// Symbol is a named part of a document: a heading, a function or a type,
// as an outline lists and GoToSymbol finds. The app gives the editor
// the State's Symbols, as its SemanticTokens.
type Symbol struct {
	// Name is what the symbol is called, which the text typed matches.
	Name string
	// Start is where the symbol begins, and End where it ends: its whole
	// body, as a function's.
	Start, End int
}

// symbols is the bar going to a symbol: the symbols fuzzy matched to
// their names as one is typed, as VS Code's go to symbol.
type symbols struct {
	open  bool
	focus bool
	text  string
	// queried is the text the matches were found for, which search
	// remakes them when the typing changes it.
	queried  string
	matches  []Symbol
	chosen   int // the index in matches the arrows chose
	scrollTo bool
}

func (s *symbols) show(ed *editor) {
	s.open, s.focus = true, true
	s.text, s.queried = "", ""
	s.search(ed)
	ed.needsBuild = true
}

func (s *symbols) close(ed *editor) {
	s.open = false
	s.matches = nil
	ed.refocus = true
}

// search finds the symbols the text typed matches, best first: every
// symbol, in the order of the text, while the text is empty.
func (s *symbols) search(ed *editor) {
	s.queried = s.text
	type hit struct {
		sym   Symbol
		score int
	}
	var hits []hit
	for _, sym := range ed.o.symbols {
		if score, ok := fuzzyScore(sym.Name, s.text); ok {
			hits = append(hits, hit{sym, score})
		}
	}
	if s.text != "" {
		slices.SortStableFunc(hits, func(a, b hit) int { return b.score - a.score })
	}
	s.matches = make([]Symbol, len(hits))
	for i, h := range hits {
		s.matches[i] = h.sym
	}
	s.chosen = 0
	s.scrollTo = true
}

// step moves the choice by d, wrapping around the matches, as the
// completions' do.
func (s *symbols) step(d int) {
	if n := len(s.matches); n > 0 {
		s.chosen = (s.chosen + d + n) % n
		s.scrollTo = true
	}
}
