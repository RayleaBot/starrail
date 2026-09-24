package app

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// ark-plugin's and StarRail-plugin's pages draw their charts with ECharts in
// the browser. Plugin templates do not run scripts, so these build the same
// charts as SVG shapes for the templates to place.

// EChartsDistribution lays out the ECharts line ark-plugin's profile-detail
// draws of a ranking distribution, in an 800 x 400 box with ECharts' default
// grid (80 left and right, 60 top and bottom): x runs over 0–100 with a tick
// every 20, y from 0 to yMax with a tick every interval and at yMax, the
// points join as smooth: true draws them and the area below them is
// filled. at places a data point in the box.
func EChartsDistribution(xs, values []float64, yMax, interval float64) (chart map[string]any, at func(x, y float64) (float64, float64)) {
	left, top, right, bottom := 80.0, 60.0, 720.0, 340.0
	at = func(x, y float64) (float64, float64) {
		if yMax <= 0 {
			return left + x/100*(right-left), bottom
		}
		return left + x/100*(right-left), bottom - y/yMax*(bottom-top)
	}
	points := make([][2]float64, len(values))
	for index, value := range values {
		points[index][0], points[index][1] = at(xs[index], value)
	}
	line := echartsSmooth(points, 0.5)
	area := ""
	if len(points) > 0 {
		area = fmt.Sprintf("%s L %s %s L %s %s Z", line, svgNumber(points[len(points)-1][0]), svgNumber(bottom), svgNumber(points[0][0]), svgNumber(bottom))
	}
	xTicks, yTicks := []any{}, []any{}
	for tick := 0.0; tick <= 100; tick += 20 {
		x, _ := at(tick, 0)
		xTicks = append(xTicks, map[string]any{"x": svgNumber(x), "value": tick})
	}
	for tick := 0.0; interval > 0 && tick <= yMax+interval*1e-9; tick += interval {
		_, y := at(0, tick)
		yTicks = append(yTicks, map[string]any{"y": svgNumber(y), "value": tick})
	}
	if last := len(yTicks) - 1; yMax > 0 && (last < 0 || yTicks[last].(map[string]any)["value"].(float64) < yMax) {
		yTicks = append(yTicks, map[string]any{"y": svgNumber(top), "value": yMax})
	}
	chart = map[string]any{"width": "800", "height": "400", "left": svgNumber(left), "right": svgNumber(right), "top": svgNumber(top), "bottom": svgNumber(bottom),
		"line": line, "area": area, "x_ticks": xTicks, "y_ticks": yTicks}
	return chart, at
}

// echartsNice is ECharts' numberUtil.nice with rounding.
func echartsNice(value float64) float64 {
	exponent := math.Floor(math.Log10(value))
	exp10 := math.Pow(10, exponent)
	f := value / exp10
	nf := 10.0
	switch {
	case f < 1.5:
		nf = 1
	case f < 2.5:
		nf = 2
	case f < 4:
		nf = 3
	case f < 7:
		nf = 5
	}
	return nf * exp10
}

// echartsSmooth is the path ECharts' poly shape draws through points with a
// smooth factor and no monotone direction: each control pair follows the
// neighbours' direction, split by the lengths of the two segments and kept
// within the neighbouring points.
func echartsSmooth(points [][2]float64, smooth float64) string {
	if len(points) == 0 {
		return ""
	}
	path := []string{"M " + svgNumber(points[0][0]) + " " + svgNumber(points[0][1])}
	cp0 := points[0]
	prev := points[0]
	for index := 1; index < len(points); index++ {
		point := points[index]
		if index == len(points)-1 {
			path = append(path, fmt.Sprintf("C %s %s %s %s %s %s", svgNumber(cp0[0]), svgNumber(cp0[1]), svgNumber(point[0]), svgNumber(point[1]), svgNumber(point[0]), svgNumber(point[1])))
			break
		}
		next := points[index+1]
		lenPrev := math.Hypot(point[0]-prev[0], point[1]-prev[1])
		lenNext := math.Hypot(next[0]-point[0], next[1]-point[1])
		ratio := lenNext / (lenNext + lenPrev)
		vx, vy := next[0]-prev[0], next[1]-prev[1]
		nextCp := [2]float64{point[0] + vx*smooth*ratio, point[1] + vy*smooth*ratio}
		nextCp[0] = math.Max(math.Min(nextCp[0], math.Max(next[0], point[0])), math.Min(next[0], point[0]))
		nextCp[1] = math.Max(math.Min(nextCp[1], math.Max(next[1], point[1])), math.Min(next[1], point[1]))
		vx, vy = nextCp[0]-point[0], nextCp[1]-point[1]
		cp1 := [2]float64{point[0] - vx*lenPrev/lenNext, point[1] - vy*lenPrev/lenNext}
		cp1[0] = math.Max(math.Min(cp1[0], math.Max(prev[0], point[0])), math.Min(prev[0], point[0]))
		cp1[1] = math.Max(math.Min(cp1[1], math.Max(prev[1], point[1])), math.Min(prev[1], point[1]))
		vx, vy = point[0]-cp1[0], point[1]-cp1[1]
		nextCp = [2]float64{point[0] + vx*lenNext/lenPrev, point[1] + vy*lenNext/lenPrev}
		path = append(path, fmt.Sprintf("C %s %s %s %s %s %s", svgNumber(cp0[0]), svgNumber(cp0[1]), svgNumber(cp1[0]), svgNumber(cp1[1]), svgNumber(point[0]), svgNumber(point[1])))
		cp0, prev = nextCp, point
	}
	return strings.Join(path, " ")
}

// echartsPalette is ECharts' default series palette.
var echartsPalette = []string{"#5470c6", "#91cc75", "#fac858", "#ee6666", "#73c0de", "#3ba272", "#fc8452", "#9a60b4", "#ea7ccc"}

// EChartsBar lays out an ECharts bar chart of one series on a category axis
// in a width x height box with the grid's (top, right, bottom, left)
// margins: the value axis runs from 0 to the nice maximum ECharts picks for
// splitNumber, each bar takes 80% of its band with the band's background
// behind it, colored by data from the default palette, and its label sits
// inside the bar.
func EChartsBar(labels []string, values []float64, width, height float64, grid [4]float64, splitNumber int) map[string]any {
	left, top := grid[3], grid[0]
	right, bottom := width-grid[1], height-grid[2]
	maximum := 0.0
	for _, value := range values {
		maximum = math.Max(maximum, value)
	}
	if maximum == 0 {
		maximum = 1
	}
	interval := echartsNice(maximum / float64(splitNumber))
	yMax := math.Ceil(maximum/interval-1e-9) * interval
	y := func(value float64) float64 { return bottom - value/yMax*(bottom-top) }
	ticks := []any{}
	for tick := 0.0; tick <= yMax+interval*1e-9; tick = math.Round((tick+interval)*1e9) / 1e9 {
		ticks = append(ticks, map[string]any{"y": svgNumber(y(tick)), "label": strconv.FormatFloat(tick, 'f', -1, 64)})
	}
	band := (right - left) / float64(max(len(values), 1))
	bars, separators := []any{}, []any{map[string]any{"x": svgNumber(left)}}
	for index, value := range values {
		x := left + band*float64(index) + band*0.1
		separators = append(separators, map[string]any{"x": svgNumber(left + band*float64(index+1))})
		bars = append(bars, map[string]any{"x": svgNumber(x), "y": svgNumber(y(value)), "width": svgNumber(band * 0.8), "height": svgNumber(bottom - y(value)),
			"color": echartsPalette[index%len(echartsPalette)], "center": svgNumber(x + band*0.4), "middle": svgNumber((y(value) + bottom) / 2), "label": labels[index], "value": strconv.FormatFloat(value, 'f', -1, 64)})
	}
	return map[string]any{"width": svgNumber(width), "height": svgNumber(height), "left": svgNumber(left), "right": svgNumber(right), "top": svgNumber(top), "bottom": svgNumber(bottom),
		"background_height": svgNumber(bottom - top), "bars": bars, "ticks": ticks, "separators": separators, "tick_bottom": svgNumber(bottom + 5), "label_y": svgNumber(bottom + 8)}
}

// EChartsCalendar lays out an ECharts calendar heatmap of one year, weeks
// from Sunday, with a piecewise visual map of splitNumber pieces over
// [low, high]: the pieces' colors and labels, every day's cell, the cells of
// the dates given filled by their piece, each month's outline, and the
// month, weekday and year labels as ECharts places them. rect is the
// calendar's (left, top, right margin) and cellHeight its row height.
func EChartsCalendar(year int, values map[string]float64, low, high float64, splitNumber int, width float64, rect [3]float64, cellHeight float64) map[string]any {
	first := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	days := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC).Sub(first).Hours() / 24
	offset := int(first.Weekday())
	weeks := (offset + int(days) + 6) / 7
	left, top := rect[0], rect[1]
	cell := (width - rect[0] - rect[2]) / float64(weeks)
	step := (high - low) / float64(splitNumber)
	precision := 0
	for math.Abs(math.Round(step*math.Pow(10, float64(precision)))/math.Pow(10, float64(precision))-step) > 1e-12 && precision < 5 {
		precision++
	}
	colors := []string{"#f6efa6", "#d88273", "#bf444c"}
	pieces := []any{}
	pieceColors := []string{}
	for index := range splitNumber {
		from, to := low+step*float64(index), low+step*float64(index+1)
		if index == splitNumber-1 {
			to = high
		}
		color := echartsLerp(float64(index)/float64(max(splitNumber-1, 1)), colors)
		pieceColors = append(pieceColors, color)
		pieces = append(pieces, map[string]any{"color": color, "label": strconv.FormatFloat(from, 'f', precision, 64) + " - " + strconv.FormatFloat(to, 'f', precision, 64)})
	}
	// The visual map's items, centred in a row at its top of 65 plus its
	// padding: a 20x14 round rectangle, a 10px gap, the label, 10px apart.
	widths, total := []float64{}, -10.0
	for _, raw := range pieces {
		label := raw.(map[string]any)["label"].(string)
		widths = append(widths, 20+10+echartsTextWidth(label, 12))
		total += widths[len(widths)-1] + 10
	}
	x := (width - total) / 2
	for index, raw := range pieces {
		piece := raw.(map[string]any)
		piece["x"], piece["text_x"] = svgNumber(x), svgNumber(x+30)
		x += widths[index] + 10
	}
	month := func(index int) time.Month { return first.AddDate(0, 0, index).Month() }
	place := func(index int) (float64, float64) {
		n := offset + index
		return left + float64(n/7)*cell, top + float64(n%7)*cellHeight
	}
	cells, filled, outlines, monthLabels := []any{}, []any{}, []any{}, []any{}
	for index := range int(days) {
		x, y := place(index)
		date := first.AddDate(0, 0, index)
		cells = append(cells, map[string]any{"x": svgNumber(x), "y": svgNumber(y)})
		if value, ok := values[date.Format(time.DateOnly)]; ok {
			piece := min(int((value-low)/step), splitNumber-1)
			// A value on a boundary belongs to the lower piece, as ECharts'
			// closed intervals are searched from the first.
			if piece > 0 && value == low+step*float64(piece) {
				piece--
			}
			filled = append(filled, map[string]any{"x": svgNumber(x), "y": svgNumber(y), "color": pieceColors[max(piece, 0)]})
		}
		if date.Day() == 1 {
			monthLabels = append(monthLabels, map[string]any{"x": svgNumber(x), "label": []string{"一月", "二月", "三月", "四月", "五月", "六月", "七月", "八月", "九月", "十月", "十一月", "十二月"}[date.Month()-1]})
		}
		// A cell's edge where the neighbour is another month or outside the
		// year draws the month outlines.
		n := offset + index
		differs := func(other int) bool { return other < 0 || other >= int(days) || month(other) != date.Month() }
		edge := func(x1, y1, x2, y2 float64) {
			outlines = append(outlines, map[string]any{"x1": svgNumber(x1), "y1": svgNumber(y1), "x2": svgNumber(x2), "y2": svgNumber(y2)})
		}
		if n%7 == 0 || differs(index-1) {
			edge(x, y, x+cell, y)
		}
		if n%7 == 6 || differs(index+1) {
			edge(x, y+cellHeight, x+cell, y+cellHeight)
		}
		if differs(index - 7) {
			edge(x, y, x, y+cellHeight)
		}
		if differs(index + 7) {
			edge(x+cell, y, x+cell, y+cellHeight)
		}
	}
	dayLabels := []any{}
	for row, label := range []string{"日", "一", "二", "三", "四", "五", "六"} {
		dayLabels = append(dayLabels, map[string]any{"x": svgNumber(left - cell/2), "y": svgNumber(top + (float64(row)+0.5)*cellHeight), "label": label})
	}
	return map[string]any{"pieces": pieces, "cell_width": svgNumber(cell), "cell_height": svgNumber(cellHeight), "cells": cells, "filled": filled, "outlines": outlines,
		"month_labels": monthLabels, "month_y": svgNumber(top - 5), "day_labels": dayLabels, "year": strconv.Itoa(year), "year_x": svgNumber(left - 30), "year_y": svgNumber(top + 3.5*cellHeight)}
}

// echartsLerp is ECharts' color.fastLerp over a color list.
func echartsLerp(at float64, colors []string) string {
	value := at * float64(len(colors)-1)
	low := int(math.Floor(value))
	high := int(math.Ceil(value))
	parse := func(hex string) [3]float64 {
		n, _ := strconv.ParseUint(hex[1:], 16, 32)
		return [3]float64{float64(n >> 16 & 255), float64(n >> 8 & 255), float64(n & 255)}
	}
	a, b := parse(colors[low]), parse(colors[high])
	t := value - float64(low)
	return fmt.Sprintf("rgba(%d,%d,%d,1)", int(math.Round(a[0]+(b[0]-a[0])*t)), int(math.Round(a[1]+(b[1]-a[1])*t)), int(math.Round(a[2]+(b[2]-a[2])*t)))
}

// echartsTextWidth estimates the width of a label of digits, points, spaces
// and dashes at a font size, as the canvas measures such text.
func echartsTextWidth(text string, size float64) float64 {
	width := 0.0
	for _, r := range text {
		switch {
		case r >= '0' && r <= '9':
			width += size * 0.56
		case r == '-':
			width += size * 0.33
		default:
			width += size * 0.28
		}
	}
	return width
}
