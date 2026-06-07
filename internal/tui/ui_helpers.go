package tui

import btable "github.com/evertras/bubble-table/table"

var normalBorder = btable.Border{
	Top:            "─",
	Left:           "│",
	Right:          "│",
	Bottom:         "─",
	TopRight:       "┐",
	TopLeft:        "┌",
	BottomRight:    "┘",
	BottomLeft:     "└",
	TopJunction:    "┬",
	LeftJunction:   "├",
	RightJunction:  "┤",
	BottomJunction: "┴",
	InnerJunction:  "┼",
	InnerDivider:   "│",
}
