package main

import (
	"bytes"
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"
)

const (
	W           = 78
	H           = 26
	groundY     = 21
	trainTop    = 12
	frameDur    = time.Second / 30
	introFrames = 90
	restX       = 14
	totalScroll = 200.0
	cruise      = 3.2
	carLen      = 13
	bodyLen     = 39
	noseLen     = 19
	trainLen    = bodyLen + noseLen
)

type RGB struct{ R, G, B uint8 }

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }
func lerp(a, b RGB, t float64) RGB {
	t = clamp(t, 0, 1)
	f := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t + 0.5) }
	return RGB{f(a.R, b.R), f(a.G, b.G), f(a.B, b.B)}
}
func easeOutCubic(t float64) float64 { t = 1 - t; return 1 - t*t*t }
func mod(a, n int) int               { return ((a % n) + n) % n }
func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func hash(n int) int {
	x := uint32(n)*2654435761 + 0x9e3779b9
	x ^= x >> 15
	x *= 2246822519
	x ^= x >> 13
	return int(x & 0x7fffffff)
}

type Canvas struct {
	w, h int
	px   []RGB
}

func NewCanvas(w, h int) *Canvas { return &Canvas{w: w, h: h, px: make([]RGB, w*h)} }
func (c *Canvas) Set(x, y int, col RGB) {
	if x >= 0 && x < c.w && y >= 0 && y < c.h {
		c.px[y*c.w+x] = col
	}
}
func (c *Canvas) At(x, y int) RGB { return c.px[y*c.w+x] }
func (c *Canvas) Blend(x, y int, col RGB, a float64) {
	if x >= 0 && x < c.w && y >= 0 && y < c.h {
		i := y*c.w + x
		c.px[i] = lerp(c.px[i], col, a)
	}
}
func (c *Canvas) Fill(col RGB) {
	for i := range c.px {
		c.px[i] = col
	}
}
func (c *Canvas) Render(b *bytes.Buffer, y0 int) {
	for y := y0; y < c.h; y += 2 {
		var fg, bg RGB
		hasFg, hasBg := false, false
		for x := 0; x < c.w; x++ {
			top, bot := c.At(x, y), c.At(x, y+1)
			if top == bot {
				if !hasBg || bg != top {
					fmt.Fprintf(b, "\x1b[48;2;%d;%d;%dm", top.R, top.G, top.B)
					bg, hasBg = top, true
				}
				b.WriteByte(' ')
				continue
			}
			if !hasFg || fg != top {
				fmt.Fprintf(b, "\x1b[38;2;%d;%d;%dm", top.R, top.G, top.B)
				fg, hasFg = top, true
			}
			if !hasBg || bg != bot {
				fmt.Fprintf(b, "\x1b[48;2;%d;%d;%dm", bot.R, bot.G, bot.B)
				bg, hasBg = bot, true
			}
			b.WriteString("▀")
		}
		b.WriteString("\x1b[0m\n")
	}
}

var (
	bgDark   = RGB{14, 15, 30}
	ridgeCol = RGB{96, 74, 136}
	fujiCol  = RGB{76, 62, 124}
	snowCol  = RGB{240, 226, 244}
	hillCol  = RGB{46, 38, 92}
	treeCol  = RGB{18, 20, 46}
	poleCol  = RGB{28, 30, 56}
	railCol  = RGB{150, 158, 184}
	ballast  = RGB{36, 36, 58}
)

func skyColor(y int) RGB {
	stops := []RGB{{10, 12, 38}, {52, 40, 104}, {176, 86, 132}, {255, 168, 124}}
	pos := float64(y) / float64(groundY) * float64(len(stops)-1)
	i := int(pos)
	if i >= len(stops)-1 {
		return stops[len(stops)-1]
	}
	return lerp(stops[i], stops[i+1], pos-float64(i))
}

func drawScene(c *Canvas, frame int, scroll, v float64) {
	blur := clamp(v/6, 0, 1)
	for y := 0; y < H; y++ {
		col := skyColor(minInt(y, groundY))
		for x := 0; x < W; x++ {
			c.Set(x, y, col)
		}
	}
	for i := 0; i < 28; i++ {
		sx := mod(hash(i*7+1)%W-int(scroll*0.02), W)
		sy := hash(i*13+5) % 11
		tw := 0.5 + 0.5*math.Sin(float64(frame)*0.15+float64(i)*1.7)
		c.Blend(sx, sy, RGB{255, 255, 255}, 0.25+0.55*tw)
	}
	for dy := -8; dy <= 8; dy++ {
		for dx := -8; dx <= 8; dx++ {
			d := math.Hypot(float64(dx), float64(dy))
			switch {
			case d <= 5:
				c.Blend(62+dx, 13+dy, RGB{255, 214, 150}, 0.95)
			case d <= 8:
				c.Blend(62+dx, 13+dy, RGB{255, 214, 150}, 0.22*(8-d)/3)
			}
		}
	}
	fill := func(x, top int, col RGB) {
		for y := top; y <= groundY; y++ {
			c.Set(x, y, col)
		}
	}
	for x := 0; x < W; x++ {
		u := float64(x) + scroll*0.05 + 7
		h := 6 + 2.5*math.Sin(u*0.09) + 1.5*math.Sin(u*0.23+1)
		fill(x, groundY-int(h+0.5), ridgeCol)
	}
	off := scroll*0.08 + 18
	for x := 0; x < W; x++ {
		u := math.Mod(float64(x)+off, 130)
		d := math.Max(math.Abs(u-65), 1.5)
		if d >= 34 {
			continue
		}
		h := 16 * math.Pow(1-d/34, 0.85)
		top := groundY - int(h)
		fill(x, top, fujiCol)
		if h > 8 {
			snow := int(h*0.3 + 1.3*math.Sin(u*1.1))
			for y := top; y < top+snow && y <= groundY; y++ {
				c.Set(x, y, snowCol)
			}
		}
	}
	for x := 0; x < W; x++ {
		u := float64(x) + scroll*0.25
		h := 5 + 2*math.Sin(u*0.13+2) + 1.2*math.Sin(u*0.31)
		fill(x, groundY-int(h), hillCol)
	}
	toff := int(scroll * 0.55)
	for x := 0; x < W; x++ {
		ix := x + toff
		cell, k := ix/7, ix%7
		h := 1
		if hash(cell*3+1)%5 != 0 {
			h = 1 + hash(cell)%3 + maxInt(0, 3-abs(k-3))
		}
		fill(x, groundY-h, treeCol)
	}
	for x := 0; x < W; x++ {
		c.Blend(x, 8, RGB{110, 104, 150}, 0.55)
	}
	po := int(scroll * 0.85)
	for x := 0; x < W; x++ {
		if (x+po)%40 < 2 {
			for y := 5; y <= groundY; y++ {
				c.Set(x, y, poleCol)
			}
			for dx := -3; dx <= 4; dx++ {
				c.Set(x+dx, 6, poleCol)
			}
		}
	}
	for x := 0; x < W; x++ {
		c.Set(x, groundY, railCol)
		c.Set(x, groundY+1, ballast)
		c.Set(x, groundY+2, ballast)
		c.Set(x, groundY+3, bgDark)
		c.Set(x, groundY+4, bgDark)
	}
	sleeper := lerp(RGB{88, 72, 88}, ballast, blur*0.8)
	so := int(scroll)
	for x := 0; x < W; x++ {
		if (x+so)%8 < 4 {
			c.Set(x, groundY+1, sleeper)
			c.Set(x, groundY+2, sleeper)
		}
	}
}

func drawSpeedLines(c *Canvas, scroll, intensity float64) {
	if intensity < 0.05 {
		return
	}
	for i := 0; i < 12; i++ {
		y := 4 + hash(i*11+3)%17
		length := 6 + hash(i*5+1)%9
		base := hash(i*17+9) % W
		x0 := mod(base-int(scroll*1.6), W+length) - length
		for k := 0; k < length; k++ {
			a := intensity * 0.5 * float64(k+1) / float64(length)
			c.Blend(x0+k, y, RGB{220, 228, 250}, a)
		}
	}
}

func drawBeam(c *Canvas, nx, ny int) {
	for d := 1; d <= 14; d++ {
		spread := d / 5
		for dy := -spread; dy <= spread; dy++ {
			a := 0.30 * (1 - float64(d)/15)
			if dy != 0 {
				a *= 0.5
			}
			c.Blend(nx+d, ny+dy, RGB{255, 236, 170}, a)
		}
	}
}

func train(c *Canvas, x0, y0 int) {
	noseTop := func(x int) int {
		if x < bodyLen {
			return 0
		}
		d := float64(x-bodyLen+1) / float64(noseLen)
		return int(math.Round(math.Pow(d, 1.8) * 5.0))
	}
	trainPixel := func(x, ry, top int) (RGB, bool) {
		inNose := x >= bodyLen
		m := x % carLen
		switch {
		case ry == 8:
			if x < 3 || x > trainLen-6 {
				return RGB{}, false
			}
			if m == 2 || m == 3 || m == 9 || m == 10 {
				return RGB{26, 28, 42}, true
			}
			return RGB{}, false
		case ry == 7:
			if x < 1 || x > trainLen-3 {
				return RGB{}, false
			}
			return RGB{56, 62, 84}, true
		case x >= trainLen-2 && ry == 5:
			return RGB{255, 238, 160}, true
		case x == 0 && ry == top:
			return RGB{}, false
		case ry == 5:
			return RGB{0, 112, 222}, true
		case ry == top:
			if ry == 0 {
				return RGB{190, 198, 216}, true
			}
			return RGB{232, 238, 248}, true
		case m == 0 && x > 0 && x <= bodyLen:
			return RGB{150, 158, 180}, true
		case ry == 2 || ry == 3:
			if !inNose && m%4 != 0 && m < 12 {
				if ry == 2 {
					return RGB{110, 166, 224}, true
				}
				return RGB{60, 108, 176}, true
			}
			if inNose && x >= bodyLen+2 && x <= bodyLen+11 {
				if ry == 2 {
					return RGB{44, 70, 126}, true
				}
				return RGB{24, 40, 86}, true
			}
		case ry == 6:
			return RGB{204, 212, 228}, true
		}
		return RGB{244, 247, 252}, true
	}
	for x := 0; x < trainLen; x++ {
		top := noseTop(x)
		for ry := top; ry <= 8; ry++ {
			if col, ok := trainPixel(x, ry, top); ok {
				c.Set(x0+x, y0+ry, col)
			}
		}
	}
	for y := y0 - 3; y < y0; y++ {
		c.Set(x0+21, y, RGB{70, 76, 100})
	}
	for dx := 19; dx <= 23; dx++ {
		c.Set(x0+dx, y0-3, RGB{88, 94, 120})
	}
}

var glyphs = map[byte][]string{
	'M': {"X...X", "XX.XX", "X.X.X", "X...X", "X...X"},
	'E': {"XXXX", "X...", "XXX.", "X...", "XXXX"},
	'T': {"XXXXX", "..X..", "..X..", "..X..", "..X.."},
	'R': {"XXXX.", "X...X", "XXXX.", "X..X.", "X...X"},
	'O': {".XXX.", "X...X", "X...X", "X...X", ".XXX."},
	'L': {"X...", "X...", "X...", "X...", "XXXX"},
	'I': {"XXX", ".X.", ".X.", ".X.", "XXX"},
	'N': {"X...X", "XX..X", "X.X.X", "X..XX", "X...X"},
}

func titleGradient(t float64) RGB {
	cyan, blue, violet := RGB{90, 225, 255}, RGB{80, 140, 255}, RGB{190, 120, 255}
	if t < 0.5 {
		return lerp(cyan, blue, t*2)
	}
	return lerp(blue, violet, (t-0.5)*2)
}

func titleCanvas(reveal float64) *Canvas {
	c := NewCanvas(W, 6)
	c.Fill(bgDark)
	const text = "METROLIN"
	total := -1
	for i := 0; i < len(text); i++ {
		total += len(glyphs[text[i]][0]) + 1
	}
	x0 := (W - total) / 2
	limit := x0 + int(reveal*float64(total)+0.5)
	cx := x0
	for i := 0; i < len(text); i++ {
		g := glyphs[text[i]]
		for gy, row := range g {
			for gx := 0; gx < len(row); gx++ {
				px := cx + gx
				if row[gx] != 'X' || px >= limit {
					continue
				}
				col := titleGradient(float64(px-x0) / float64(total))
				if reveal < 1 && limit-px <= 2 {
					col = lerp(col, RGB{255, 255, 255}, 0.7)
				}
				c.Set(px, 1+gy, col)
			}
		}
		cx += len(g[0]) + 1
	}
	return c
}

func tagline(show, loop bool) string {
	text := "metrolin-cli v1.0.0"
	if loop {
		text += "  ·  Ctrl+C to exit"
	}
	n := len([]rune(text))
	line := strings.Repeat(" ", W)
	if show {
		pad := (W - n) / 2
		line = strings.Repeat(" ", pad) + text + strings.Repeat(" ", W-pad-n)
	}
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm\x1b[38;2;118;132;178m%s\x1b[0m\n", bgDark.R, bgDark.G, bgDark.B, line)
}

func isTTY() bool {
	fi, err := os.Stdout.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func main() {
	loop := flag.Bool("loop", false, "keep the train running until Ctrl+C")
	flag.Parse()
	if !isTTY() || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		fmt.Println("metrolin-cli v1.0.0")
		return
	}
	cols, rows, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		cols, rows = 80, 24
	}
	avail := rows - 1
	sceneRows, titleRows := H/2, 3
	if avail < sceneRows+titleRows+1 {
		sceneRows = avail - titleRows - 1
	}
	if sceneRows < 10 {
		titleRows = 0
		sceneRows = minInt(H/2, avail-1)
	}
	if cols < W || sceneRows < 7 {
		fmt.Println("metrolin-cli v1.0.0")
		return
	}
	skip := (H/2 - sceneRows) * 2
	lines := sceneRows + titleRows + 1
	fmt.Print("\x1b[?25l")
	defer fmt.Print("\x1b[?25h")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	ticker := time.NewTicker(frameDur)
	defer ticker.Stop()
	var buf bytes.Buffer
	var prevScroll float64
	for f := 0; ; f++ {
		var scroll, v, reveal float64
		trainX := restX
		if *loop {
			scroll, v, reveal = cruise*float64(f), cruise, 1
		} else {
			p := math.Min(1, float64(f)/introFrames)
			e := easeOutCubic(p)
			scroll = totalScroll * e
			v = scroll - prevScroll
			trainX = int(math.Round(-trainLen + float64(restX+trainLen)*e))
			reveal = clamp((p-0.5)/0.45, 0, 1)
		}
		prevScroll = scroll
		scene := NewCanvas(W, H)
		drawScene(scene, f, scroll, v)
		drawSpeedLines(scene, scroll, clamp(v/2.5, 0, 1))
		drawBeam(scene, trainX+trainLen, trainTop+5)
		train(scene, trainX, trainTop)
		buf.Reset()
		if f > 0 {
			fmt.Fprintf(&buf, "\x1b[%dA\r", lines)
		}
		buf.WriteString("\x1b[?2026h")
		scene.Render(&buf, skip)
		if titleRows > 0 {
			titleCanvas(reveal).Render(&buf, 0)
		}
		buf.WriteString(tagline(reveal >= 0.6, *loop))
		buf.WriteString("\x1b[?2026l")
		os.Stdout.Write(buf.Bytes())
		if !*loop && f >= introFrames {
			return
		}
		select {
		case <-stop:
			return
		case <-ticker.C:
			SelectOption()
		}
	}
}
