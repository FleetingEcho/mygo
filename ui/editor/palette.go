package editor

import "slices"

// palette is the bar running a command chosen from all of them: the
// commands fuzzy matched to their names as one is typed, as Zed's
// command palette.
type palette struct {
	open  bool
	focus bool
	text  string
	// queried is the text the matches were found for, which search
	// remakes them when the typing changes.
	queried  string
	matches  []Command
	chosen   int // the index in matches the arrows chose
	scrollTo bool
}

func (p *palette) show(ed *editor) {
	p.open, p.focus = true, true
	p.text, p.queried = "", ""
	p.search()
	ed.needsBuild = true
}

func (p *palette) close(ed *editor) {
	p.open = false
	p.matches = nil
	ed.refocus = true
}

// search finds the commands the text typed matches, best first: every
// command, in their order, while the text is empty.
func (p *palette) search() {
	p.queried = p.text
	type hit struct {
		cmd   Command
		score int
	}
	var hits []hit
	for c := Command(1); c < commandCount; c++ {
		if s, ok := fuzzyScore(c.String(), p.text); ok {
			hits = append(hits, hit{c, s})
		}
	}
	if p.text != "" {
		slices.SortStableFunc(hits, func(a, b hit) int { return b.score - a.score })
	}
	p.matches = make([]Command, len(hits))
	for i, h := range hits {
		p.matches[i] = h.cmd
	}
	p.chosen = 0
	p.scrollTo = true
}

// step moves the choice by d, wrapping around the matches, as the
// completions' do.
func (p *palette) step(d int) {
	if n := len(p.matches); n > 0 {
		p.chosen = (p.chosen + d + n) % n
		p.scrollTo = true
	}
}
