// Drawing helpers. The game is laid out on a fixed 1600x900 "stage" (the
// same layout as before) that is scaled to fit the window. Everything is
// drawn at the screen's real pixel resolution: text, panels and shapes are
// rasterised at device size, and the big images are drawn with mipmaps, so
// the game stays sharp from 720p up to 4K.
//
// (c) 2026 Ironite
package main

import (
	"image"
	"image/color"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	stageW = 1600.0
	stageH = 900.0
)

// stage maps logical stage coordinates to device pixels.
type stageT struct {
	s, ox, oy float64 // scale and offset in device pixels
	frame     int     // frame counter, used to age caches
}

var st stageT

func (t *stageT) fit(w, h float64) bool {
	s := math.Min(w/stageW, h/stageH)
	ox, oy := math.Round((w-stageW*s)/2), math.Round((h-stageH*s)/2)
	changed := math.Abs(s-t.s) > 1e-6
	t.s, t.ox, t.oy = s, ox, oy
	return changed
}
func dx(x float64) float64 { return st.ox + x*st.s }
func dy(y float64) float64 { return st.oy + y*st.s }
func dv(v float64) float64 { return v * st.s }

// toLogical converts a device-pixel position to stage coordinates.
func toLogical(x, y float64) (float64, float64) { return (x - st.ox) / st.s, (y - st.oy) / st.s }

type rect struct{ x, y, w, h float64 }

func (r rect) has(x, y float64) bool { return x >= r.x && y >= r.y && x < r.x+r.w && y < r.y+r.h }
func (r rect) cx() float64           { return r.x + r.w/2 }
func (r rect) cy() float64           { return r.y + r.h/2 }

func rgba(r, g, b uint8, a float64) color.RGBA {
	return color.RGBA{uint8(float64(r) * a), uint8(float64(g) * a), uint8(float64(b) * a), uint8(255 * a)} // premultiplied
}
func hexc(s string) color.RGBA {
	s = strings.TrimPrefix(s, "#")
	var v [3]uint8
	for i := 0; i < 3; i++ {
		var n uint8
		for _, c := range s[i*2 : i*2+2] {
			n <<= 4
			switch {
			case c >= '0' && c <= '9':
				n |= uint8(c - '0')
			case c >= 'a' && c <= 'f':
				n |= uint8(c-'a') + 10
			case c >= 'A' && c <= 'F':
				n |= uint8(c-'A') + 10
			}
		}
		v[i] = n
	}
	return color.RGBA{v[0], v[1], v[2], 255}
}
func withA(c color.RGBA, a float64) color.RGBA {
	return color.RGBA{uint8(float64(c.R) * a), uint8(float64(c.G) * a), uint8(float64(c.B) * a), uint8(float64(c.A) * a)}
}

/* ------------------------------------------------------------ images */

// drawImage draws img into the logical rect (x, y, w, h).
func drawImage(dst, img *ebiten.Image, x, y, w, h, alpha float64) {
	b := img.Bounds()
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(dv(w)/float64(b.Dx()), dv(h)/float64(b.Dy()))
	op.GeoM.Translate(dx(x), dy(y))
	if alpha < 1 {
		op.ColorScale.ScaleAlpha(float32(alpha))
	}
	dst.DrawImage(img, op)
}

// drawImageTinted draws img with a colour multiplied in (white images become clr).
func drawImageTinted(dst, img *ebiten.Image, x, y, w, h float64, clr color.Color) {
	b := img.Bounds()
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(dv(w)/float64(b.Dx()), dv(h)/float64(b.Dy()))
	op.GeoM.Translate(dx(x), dy(y))
	op.ColorScale.ScaleWithColor(clr)
	dst.DrawImage(img, op)
}

// drawImageRot draws img centred on (cx, cy), w x h logical, rotated by ang.
func drawImageRot(dst, img *ebiten.Image, cx, cy, w, h, ang float64, clr color.Color) {
	b := img.Bounds()
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-float64(b.Dx())/2, -float64(b.Dy())/2)
	op.GeoM.Scale(dv(w)/float64(b.Dx()), dv(h)/float64(b.Dy()))
	op.GeoM.Rotate(ang)
	op.GeoM.Translate(dx(cx), dy(cy))
	if clr != nil {
		op.ColorScale.ScaleWithColor(clr)
	}
	dst.DrawImage(img, op)
}

var whiteImg = func() *ebiten.Image {
	im := ebiten.NewImage(3, 3)
	pix := make([]byte, 4*9)
	for i := range pix {
		pix[i] = 0xff
	}
	im.WritePixels(pix)
	return im
}()
var whiteSub = whiteImg.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)

func fillRect(dst *ebiten.Image, x, y, w, h float64, clr color.Color) {
	vector.DrawFilledRect(dst, float32(dx(x)), float32(dy(y)), float32(dv(w)), float32(dv(h)), clr, false)
}
func fillCircle(dst *ebiten.Image, cx, cy, r float64, clr color.Color) {
	vector.DrawFilledCircle(dst, float32(dx(cx)), float32(dy(cy)), float32(dv(r)), clr, true)
}
func strokeCircle(dst *ebiten.Image, cx, cy, r, w float64, clr color.Color) {
	vector.StrokeCircle(dst, float32(dx(cx)), float32(dy(cy)), float32(dv(r)), float32(dv(w)), clr, true)
}

// fillPath fills a path given in logical coordinates (clr is premultiplied).
func fillPath(dst *ebiten.Image, build func(p *vector.Path), clr color.Color, rule ebiten.FillRule) {
	var p vector.Path
	build(&p)
	vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
	r, g, b, a := clr.RGBA()
	for i := range vs {
		vs[i].DstX = float32(dx(float64(vs[i].DstX)))
		vs[i].DstY = float32(dy(float64(vs[i].DstY)))
		vs[i].SrcX, vs[i].SrcY = 1, 1
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = float32(r)/0xffff, float32(g)/0xffff, float32(b)/0xffff, float32(a)/0xffff
	}
	op := &ebiten.DrawTrianglesOptions{ColorScaleMode: ebiten.ColorScaleModePremultipliedAlpha, AntiAlias: true, FillRule: rule}
	dst.DrawTriangles(vs, is, whiteSub, op)
}

/* ------------------------------------------------------------ cache */

type cacheEntry struct {
	img  *ebiten.Image
	w, h float64 // logical size
	base float64 // logical baseline offset (text)
	used int
}

type imgCache struct{ m map[string]*cacheEntry }

func newCache() *imgCache { return &imgCache{m: map[string]*cacheEntry{}} }

func (c *imgCache) get(key string, make func() *cacheEntry) *cacheEntry {
	if e, ok := c.m[key]; ok {
		e.used = st.frame
		return e
	}
	e := make()
	e.used = st.frame
	c.m[key] = e
	return e
}

// sweep drops images that haven't been drawn for a while.
func (c *imgCache) sweep(age int) {
	for k, e := range c.m {
		if st.frame-e.used > age {
			if e.img != nil {
				e.img.Deallocate()
			}
			delete(c.m, k)
		}
	}
}
func (c *imgCache) clear() {
	for k, e := range c.m {
		if e.img != nil {
			e.img.Deallocate()
		}
		delete(c.m, k)
	}
}

var (
	textCache = newCache()
	boxCache  = newCache()
)

/* ------------------------------------------------------------- fonts */

type fontID int

const (
	fRegular fontID = iota
	fItalic
	fSemi
	fSemiItalic
)

var fonts [4]*opentype.Font

type faceKey struct {
	f  fontID
	px float64
}

var faces = map[faceKey]font.Face{}

func face(f fontID, px float64) font.Face {
	px = math.Round(px*4) / 4
	k := faceKey{f, px}
	if fc, ok := faces[k]; ok {
		return fc
	}
	fc, err := opentype.NewFace(fonts[f], &opentype.FaceOptions{Size: px, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		panic(err)
	}
	faces[k] = fc
	return fc
}

type textStyle struct {
	f       fontID
	size    float64 // logical px
	spacing float64 // letter spacing, logical px
	upper   bool
}

func (t textStyle) str(s string) string {
	if t.upper {
		return strings.ToUpper(s)
	}
	return s
}

// measure returns the logical width of s.
func measure(t textStyle, s string) float64 {
	s = t.str(s)
	fc := face(t.f, t.size*4) // measure at a fixed large size for stability
	w := fixed.Int26_6(0)
	prev := rune(-1)
	for _, r := range s {
		if prev >= 0 {
			w += fc.Kern(prev, r)
		}
		a, _ := fc.GlyphAdvance(r)
		w += a
		prev = r
	}
	n := float64(utf8.RuneCountInString(s))
	return float64(w)/64/4 + t.spacing*n
}

// rasterText draws s into an RGBA image (white, alpha = coverage) at device scale.
func rasterText(t textStyle, s string, clr color.Color) *cacheEntry {
	s = t.str(s)
	px := t.size * st.s
	fc := face(t.f, px)
	m := fc.Metrics()
	asc, desc := float64(m.Ascent)/64, float64(m.Descent)/64
	wl := measure(t, s)
	W := int(math.Ceil(wl*st.s)) + 4
	H := int(math.Ceil(asc+desc)) + 4
	rgbaImg := image.NewRGBA(image.Rect(0, 0, W, H))
	d := &font.Drawer{Dst: rgbaImg, Src: image.NewUniform(clr), Face: fc}
	x := 2.0
	prev := rune(-1)
	for _, r := range s {
		if prev >= 0 {
			x += float64(fc.Kern(prev, r)) / 64
		}
		d.Dot = fixed.Point26_6{X: fixed.Int26_6(x * 64), Y: fixed.Int26_6((asc + 2) * 64)}
		d.DrawString(string(r))
		a, _ := fc.GlyphAdvance(r)
		x += float64(a)/64 + t.spacing*st.s
		prev = r
	}
	return &cacheEntry{img: ebiten.NewImageFromImage(rgbaImg), w: float64(W) / st.s, h: float64(H) / st.s, base: (asc + 2) / st.s}
}

type align int

const (
	alignLeft align = iota
	alignCenter
	alignRight
)

// text draws s with its baseline at y (logical). Returns the logical width.
func text(dst *ebiten.Image, t textStyle, s string, x, y float64, al align, clr color.Color) float64 {
	if s == "" {
		return 0
	}
	key := textKey(t, s)
	e := textCache.get(key, func() *cacheEntry { return rasterText(t, s, color.White) })
	w := e.w - 4/st.s
	switch al {
	case alignCenter:
		x -= w / 2
	case alignRight:
		x -= w
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(math.Round(dx(x)-2), math.Round(dy(y-e.base)))
	op.ColorScale.ScaleWithColor(clr)
	dst.DrawImage(e.img, op)
	return w
}

// textMid draws s vertically centred on cy (using the cap height).
func textMid(dst *ebiten.Image, t textStyle, s string, x, cy float64, al align, clr color.Color) float64 {
	return text(dst, t, s, x, cy+t.size*0.35, al, clr)
}

func textKey(t textStyle, s string) string {
	return string(rune('A'+int(t.f))) + "|" + ftoa(t.size) + "|" + ftoa(t.spacing) + "|" + ftoa(st.s) + "|" + t.str(s)
}

func ftoa(f float64) string {
	return strings.TrimRight(strings.TrimRight(strconvF(f), "0"), ".")
}

func strconvF(f float64) string {
	// small, allocation-light float formatting for cache keys
	neg := f < 0
	if neg {
		f = -f
	}
	ip := int64(f)
	fp := int64(math.Round((f - float64(ip)) * 1000))
	if fp == 1000 {
		ip++
		fp = 0
	}
	s := itoa64(ip) + "." + pad3(fp)
	if neg {
		s = "-" + s
	}
	return s
}
func itoa64(n int64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
func pad3(n int64) string {
	s := itoa64(n)
	for len(s) < 3 {
		s = "0" + s
	}
	return s
}

// wrap splits s into lines no wider than w (logical).
func wrap(t textStyle, s string, w float64) []string {
	var lines []string
	for _, para := range strings.Split(s, "\n") {
		words := strings.Fields(para)
		cur := ""
		for _, wd := range words {
			try := wd
			if cur != "" {
				try = cur + " " + wd
			}
			if cur != "" && measure(t, try) > w {
				lines = append(lines, cur)
				cur = wd
			} else {
				cur = try
			}
		}
		lines = append(lines, cur)
	}
	return lines
}

/* ------------------------------------------------------------- boxes */

type stop struct {
	at  float64
	clr color.RGBA // straight (non-premultiplied) colour with alpha
}

// boxStyle describes a rounded panel, button or plaque, like the CSS boxes
// the game used before: a vertical gradient, a border, a drop shadow, an
// outer glow and a soft inner highlight along the top edge.
type boxStyle struct {
	radius      float64
	fill        []stop
	border      color.RGBA
	borderW     float64
	shadow      color.RGBA
	shadowBlur  float64
	shadowY     float64
	ring        color.RGBA // 0 0 0 Npx ring outside the border
	ringW       float64
	glow        color.RGBA
	glowBlur    float64
	innerTop    color.RGBA // inset 0 1px highlight
	key         string
	dashedBelow bool
}

func lerpC(a, b color.RGBA, t float64) [4]float64 {
	return [4]float64{
		float64(a.R) + (float64(b.R)-float64(a.R))*t,
		float64(a.G) + (float64(b.G)-float64(a.G))*t,
		float64(a.B) + (float64(b.B)-float64(a.B))*t,
		float64(a.A) + (float64(b.A)-float64(a.A))*t,
	}
}

func gradAt(stops []stop, t float64) [4]float64 {
	if len(stops) == 1 {
		c := stops[0].clr
		return [4]float64{float64(c.R), float64(c.G), float64(c.B), float64(c.A)}
	}
	if t <= stops[0].at {
		return lerpC(stops[0].clr, stops[0].clr, 0)
	}
	for i := 1; i < len(stops); i++ {
		if t <= stops[i].at {
			a, b := stops[i-1], stops[i]
			return lerpC(a.clr, b.clr, (t-a.at)/(b.at-a.at))
		}
	}
	c := stops[len(stops)-1].clr
	return [4]float64{float64(c.R), float64(c.G), float64(c.B), float64(c.A)}
}

// sdRound is the signed distance from (px, py) to a rounded rect of size w x h at the origin.
func sdRound(px, py, w, h, r float64) float64 {
	qx := math.Abs(px-w/2) - (w/2 - r)
	qy := math.Abs(py-h/2) - (h/2 - r)
	ox, oy := math.Max(qx, 0), math.Max(qy, 0)
	return math.Hypot(ox, oy) + math.Min(math.Max(qx, qy), 0) - r
}

func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }

// over composites a premultiplied colour (0-255 channels, alpha a in 0-1) over dst.
func over(dst *[4]float64, r, g, b, a float64) {
	dst[0] = r + dst[0]*(1-a)
	dst[1] = g + dst[1]*(1-a)
	dst[2] = b + dst[2]*(1-a)
	dst[3] = a*255 + dst[3]*(1-a)
}

func renderBox(bs boxStyle, w, h float64) *cacheEntry {
	s := st.s
	m := math.Max(bs.shadowBlur*1.6+math.Abs(bs.shadowY), bs.glowBlur*1.6) + bs.ringW + 2
	W, H := int(math.Ceil((w+2*m)*s)), int(math.Ceil((h+2*m)*s))
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	r := math.Min(bs.radius, math.Min(w, h)/2)
	aa := 1 / s // one device pixel in logical units
	for j := 0; j < H; j++ {
		ly := (float64(j)+.5)/s - m
		fillC := gradAt(bs.fill, clamp01(ly/h))
		for i := 0; i < W; i++ {
			lx := (float64(i)+.5)/s - m
			d := sdRound(lx, ly, w, h, r)
			if d > m {
				continue
			}
			var px [4]float64
			if bs.shadow.A > 0 {
				ds := sdRound(lx, ly-bs.shadowY, w, h, r)
				sig := math.Max(bs.shadowBlur/2, .5)
				a := .5 * math.Erfc(ds/(sig*math.Sqrt2)) * float64(bs.shadow.A) / 255
				c := bs.shadow
				over(&px, float64(c.R)*a, float64(c.G)*a, float64(c.B)*a, a)
			}
			if bs.glow.A > 0 {
				sig := math.Max(bs.glowBlur/2, .5)
				a := .5 * math.Erfc(d/(sig*math.Sqrt2)) * float64(bs.glow.A) / 255
				c := bs.glow
				over(&px, float64(c.R)*a, float64(c.G)*a, float64(c.B)*a, a)
			}
			if bs.ringW > 0 {
				cov := clamp01(.5-(d-bs.ringW)/aa) - clamp01(.5-d/aa)
				a := cov * float64(bs.ring.A) / 255
				c := bs.ring
				over(&px, float64(c.R)*a, float64(c.G)*a, float64(c.B)*a, a)
			}
			cov := clamp01(.5 - d/aa)
			if cov > 0 {
				a := cov * fillC[3] / 255
				over(&px, fillC[0]*a, fillC[1]*a, fillC[2]*a, a)
				if bs.innerTop.A > 0 && ly >= 0 && ly < 1.4 && d < -.5 {
					t := (1 - ly/1.4) * cov * float64(bs.innerTop.A) / 255
					c := bs.innerTop
					over(&px, float64(c.R)*t, float64(c.G)*t, float64(c.B)*t, t)
				}
				if bs.borderW > 0 {
					bc := clamp01(.5-d/aa) - clamp01(.5-(d+bs.borderW)/aa)
					a := bc * float64(bs.border.A) / 255
					c := bs.border
					over(&px, float64(c.R)*a, float64(c.G)*a, float64(c.B)*a, a)
				}
			}
			o := img.PixOffset(i, j)
			img.Pix[o] = uint8(math.Min(255, px[0]))
			img.Pix[o+1] = uint8(math.Min(255, px[1]))
			img.Pix[o+2] = uint8(math.Min(255, px[2]))
			img.Pix[o+3] = uint8(math.Min(255, px[3]))
		}
	}
	return &cacheEntry{img: ebiten.NewImageFromImage(img), w: w + 2*m, h: h + 2*m, base: m}
}

// box draws a styled rounded box filling the logical rect r.
func box(dst *ebiten.Image, bs boxStyle, r rect, alpha float64) {
	key := bs.key + "|" + ftoa(r.w) + "x" + ftoa(r.h) + "@" + ftoa(st.s)
	e := boxCache.get(key, func() *cacheEntry { return renderBox(bs, r.w, r.h) })
	m := e.base
	b := e.img.Bounds()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(dv(e.w)/float64(b.Dx()), dv(e.h)/float64(b.Dy()))
	op.GeoM.Translate(math.Round(dx(r.x-m)), math.Round(dy(r.y-m)))
	if alpha < 1 {
		op.ColorScale.ScaleAlpha(float32(alpha))
	}
	dst.DrawImage(e.img, op)
}

// radialImage makes a w x h (device px) image from a function of the
// normalised distance from (cx, cy) (fractions of w, h) over radius r (fraction of w).
func radialImage(w, h int, cx, cy, r float64, f func(t float64) [4]float64) *ebiten.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			d := math.Hypot(float64(i)+.5-cx*float64(w), float64(j)+.5-cy*float64(h)) / (r * float64(w))
			c := f(d)
			o := img.PixOffset(i, j)
			a := c[3]
			img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = uint8(c[0]*a), uint8(c[1]*a), uint8(c[2]*a), uint8(a*255)
		}
	}
	return ebiten.NewImageFromImage(img)
}
