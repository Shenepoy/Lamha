package annotate

import (
	"image"
	"image/color"
	"math"
)

var digitGlyphs = [10][7]string{
	{
		"01110",
		"10001",
		"10001",
		"10001",
		"10001",
		"10001",
		"01110",
	},
	{
		"00100",
		"01100",
		"00100",
		"00100",
		"00100",
		"00100",
		"01110",
	},
	{
		"01110",
		"10001",
		"00001",
		"00010",
		"00100",
		"01000",
		"11111",
	},
	{
		"01110",
		"10001",
		"00001",
		"00110",
		"00001",
		"10001",
		"01110",
	},
	{
		"00010",
		"00110",
		"01010",
		"10010",
		"11111",
		"00010",
		"00010",
	},
	{
		"11111",
		"10000",
		"11110",
		"00001",
		"00001",
		"10001",
		"01110",
	},
	{
		"01110",
		"10000",
		"11110",
		"10001",
		"10001",
		"10001",
		"01110",
	},
	{
		"11111",
		"00001",
		"00010",
		"00100",
		"01000",
		"01000",
		"01000",
	},
	{
		"01110",
		"10001",
		"10001",
		"01110",
		"10001",
		"10001",
		"01110",
	},
	{
		"01110",
		"10001",
		"10001",
		"01111",
		"00001",
		"00001",
		"01110",
	},
}

func drawNumber(dst *image.NRGBA, cx, cy float64, number int, ink color.NRGBA, diameter float64) {
	if number < 0 {
		number = 0
	}
	digits := []int{}
	if number == 0 {
		digits = []int{0}
	}
	for n := number; n > 0; n /= 10 {
		digits = append([]int{n % 10}, digits...)
	}

	cell := math.Max(4.2, diameter*0.072)
	gap := cell * 0.35
	glyphW := 5 * cell
	totalW := float64(len(digits))*glyphW + float64(len(digits)-1)*gap
	totalH := 7 * cell
	x0 := cx - totalW/2
	y0 := cy - totalH/2
	radius := cell * 0.62
	for i, digit := range digits {
		ox := x0 + float64(i)*(glyphW+gap)
		drawGlyph(dst, ox, y0, digit, cell, radius, ink)
	}
}

func drawGlyph(dst *image.NRGBA, x0, y0 float64, digit int, cell, radius float64, ink color.NRGBA) {
	if digit < 0 || digit > 9 {
		return
	}
	for row, line := range digitGlyphs[digit] {
		for col, ch := range line {
			if ch != '1' {
				continue
			}
			stampDisc(dst, x0+(float64(col)+0.5)*cell, y0+(float64(row)+0.5)*cell, radius, ink)
		}
	}
}
