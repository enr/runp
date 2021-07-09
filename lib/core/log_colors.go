package core

import (
	ct "github.com/daviddengcn/go-colortext"
)

type labelColor struct {
	background ct.Color
	foreground ct.Color
}

var labelColors = []labelColor{
	//
	// {background: ct.None, foreground: ct.None},
	{background: ct.None, foreground: ct.White},
	// {background: ct.None, foreground: ct.Black},
	{background: ct.None, foreground: ct.Green},
	{background: ct.None, foreground: ct.Cyan},
	{background: ct.None, foreground: ct.Magenta},
	{background: ct.None, foreground: ct.Yellow},
	{background: ct.None, foreground: ct.Blue},
	{background: ct.None, foreground: ct.Red},
	//
	// {background: ct.White, foreground: ct.None},
	// {background: ct.White, foreground: ct.White},
	{background: ct.White, foreground: ct.Black},
	{background: ct.White, foreground: ct.Green},
	// {background: ct.White, foreground: ct.Cyan},
	{background: ct.White, foreground: ct.Magenta},
	// {background: ct.White, foreground: ct.Yellow},
	{background: ct.White, foreground: ct.Blue},
	{background: ct.White, foreground: ct.Red},
	//
	// {background: ct.Black, foreground: ct.None},
	{background: ct.Black, foreground: ct.White},
	// {background: ct.Black, foreground: ct.Black},
	{background: ct.Black, foreground: ct.Green},
	{background: ct.Black, foreground: ct.Cyan},
	{background: ct.Black, foreground: ct.Magenta},
	{background: ct.Black, foreground: ct.Yellow},
	{background: ct.Black, foreground: ct.Blue},
	{background: ct.Black, foreground: ct.Red},
	//
	// {background: ct.Green, foreground: ct.None},
	{background: ct.Green, foreground: ct.White},
	{background: ct.Green, foreground: ct.Black},
	// {background: ct.Green, foreground: ct.Green},
	// {background: ct.Green, foreground: ct.Cyan},
	// {background: ct.Green, foreground: ct.Magenta},
	{background: ct.Green, foreground: ct.Yellow},
	// {background: ct.Green, foreground: ct.Blue},
	{background: ct.Green, foreground: ct.Red},
	//
	// {background: ct.Cyan, foreground: ct.None},
	// {background: ct.Cyan, foreground: ct.White},
	{background: ct.Cyan, foreground: ct.Black},
	// {background: ct.Cyan, foreground: ct.Green},
	// {background: ct.Cyan, foreground: ct.Cyan},
	{background: ct.Cyan, foreground: ct.Magenta},
	{background: ct.Cyan, foreground: ct.Yellow},
	// {background: ct.Cyan, foreground: ct.Blue},
	// {background: ct.Cyan, foreground: ct.Red},
	//
	// {background: ct.Magenta, foreground: ct.None},
	{background: ct.Magenta, foreground: ct.White},
	{background: ct.Magenta, foreground: ct.Black},
	// {background: ct.Magenta, foreground: ct.Green},
	{background: ct.Magenta, foreground: ct.Cyan},
	// {background: ct.Magenta, foreground: ct.Magenta},
	{background: ct.Magenta, foreground: ct.Yellow},
	{background: ct.Magenta, foreground: ct.Blue},
	// {background: ct.Magenta, foreground: ct.Red},
	//
	// {background: ct.Yellow, foreground: ct.None},
	// {background: ct.Yellow, foreground: ct.White},
	{background: ct.Yellow, foreground: ct.Black},
	{background: ct.Yellow, foreground: ct.Green},
	{background: ct.Yellow, foreground: ct.Cyan},
	{background: ct.Yellow, foreground: ct.Magenta},
	// {background: ct.Yellow, foreground: ct.Yellow},
	{background: ct.Yellow, foreground: ct.Blue},
	{background: ct.Yellow, foreground: ct.Red},
	//
	// {background: ct.Blue, foreground: ct.None},
	{background: ct.Blue, foreground: ct.White},
	{background: ct.Blue, foreground: ct.Black},
	// {background: ct.Blue, foreground: ct.Green},
	// {background: ct.Blue, foreground: ct.Cyan},
	{background: ct.Blue, foreground: ct.Magenta},
	{background: ct.Blue, foreground: ct.Yellow},
	// {background: ct.Blue, foreground: ct.Blue},
	{background: ct.Blue, foreground: ct.Red},
	//
	// {background: ct.Red, foreground: ct.None},
	{background: ct.Red, foreground: ct.White},
	{background: ct.Red, foreground: ct.Black},
	{background: ct.Red, foreground: ct.Green},
	// {background: ct.Red, foreground: ct.Cyan},
	// {background: ct.Red, foreground: ct.Magenta},
	{background: ct.Red, foreground: ct.Yellow},
	{background: ct.Red, foreground: ct.Blue},
	// {background: ct.Red, foreground: ct.Red},
}
