package core

import (
	ct "github.com/daviddengcn/go-colortext"
)

type labelColor struct {
	background ct.Color
	foreground ct.Color
}

var labelColors = []labelColor{
	// No background — bright foregrounds readable on both dark and light terminals
	{background: ct.None, foreground: ct.Yellow},
	{background: ct.None, foreground: ct.Cyan},
	{background: ct.None, foreground: ct.Green},
	{background: ct.None, foreground: ct.Magenta},
	{background: ct.None, foreground: ct.White},
	// High-contrast solid backgrounds
	{background: ct.Black, foreground: ct.White},
	{background: ct.Black, foreground: ct.Yellow},
	{background: ct.White, foreground: ct.Black},
	{background: ct.Yellow, foreground: ct.Black},
	{background: ct.Blue, foreground: ct.White},
	{background: ct.Blue, foreground: ct.Yellow},
	// Good contrast
	{background: ct.Black, foreground: ct.Cyan},
	{background: ct.Black, foreground: ct.Magenta},
	{background: ct.Black, foreground: ct.Green},
	{background: ct.White, foreground: ct.Blue},
	{background: ct.White, foreground: ct.Red},
	{background: ct.White, foreground: ct.Magenta},
	{background: ct.Cyan, foreground: ct.Black},
	{background: ct.Cyan, foreground: ct.Yellow},
	{background: ct.Magenta, foreground: ct.Black},
	{background: ct.Magenta, foreground: ct.White},
	{background: ct.Magenta, foreground: ct.Yellow},
	{background: ct.Red, foreground: ct.White},
	{background: ct.Red, foreground: ct.Black},
	{background: ct.Red, foreground: ct.Yellow},
	{background: ct.Green, foreground: ct.Black},
	{background: ct.Green, foreground: ct.White},
	{background: ct.Green, foreground: ct.Yellow},
	{background: ct.Yellow, foreground: ct.Blue},
	{background: ct.Yellow, foreground: ct.Magenta},
	{background: ct.Yellow, foreground: ct.Red},
}
