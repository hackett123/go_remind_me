package tui

type LayoutMode int

const (
	LayoutCompact LayoutMode = iota
	LayoutCard
)

var layoutNames = []string{"Compact", "Card"}

var currentLayout LayoutMode
