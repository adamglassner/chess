// Built-in art, fonts, music and sounds. The images were drawn by Chess's
// original art code (tools/assets) at 4K-ready sizes; nothing is loaded from
// disk at run time.
//
// (c) 2026 Ironite
package main

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"io/fs"
	"path"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font/opentype"
)

//go:embed assets/pieces assets/boards assets/people assets/fonts assets/audio assets/icon_*.png assets/ironite-logo.png
var assetFS embed.FS

//go:embed CHANGELOG.md
var changelogMD string

var pieceSizes = []int{90, 135, 180, 225}

type artT struct {
	pieces  map[int8][]*ebiten.Image // by code, one per pieceSizes entry
	surface map[string]*ebiten.Image // the 8x8 squares per board theme
	desk    map[string]*ebiten.Image // desk, leather pad and board frame per theme
	people  map[string]*ebiten.Image
	logo    *ebiten.Image
	icons   []image.Image
	shadow  map[int8]*ebiten.Image // soft silhouettes for a lifted piece
}

var art artT

func mustImage(name string) image.Image {
	b, err := assetFS.ReadFile(name)
	if err != nil {
		panic(err)
	}
	im, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		panic(fmt.Errorf("%s: %w", name, err))
	}
	return im
}

func loadAssets() error {
	art.pieces = map[int8][]*ebiten.Image{}
	art.shadow = map[int8]*ebiten.Image{}
	for code := int8(-6); code <= 6; code++ {
		if code == 0 {
			continue
		}
		side := "w"
		if code < 0 {
			side = "b"
		}
		letter := string(" PNBRQK"[absI8(code)])
		for _, sz := range pieceSizes {
			im := mustImage(fmt.Sprintf("assets/pieces/%s%s_%d.png", side, letter, sz))
			art.pieces[code] = append(art.pieces[code], ebiten.NewImageFromImage(im))
			if sz == 225 {
				art.shadow[code] = ebiten.NewImageFromImage(silhouette(im, 7))
			}
		}
	}
	art.surface, art.desk = map[string]*ebiten.Image{}, map[string]*ebiten.Image{}
	for _, t := range []string{"walnut", "marble", "club"} {
		art.surface[t] = ebiten.NewImageFromImage(mustImage("assets/boards/" + t + ".jpg"))
		art.desk[t] = ebiten.NewImageFromImage(mustImage("assets/boards/desk_" + t + ".jpg"))
	}
	art.people = map[string]*ebiten.Image{}
	ents, _ := fs.ReadDir(assetFS, "assets/people")
	for _, e := range ents {
		n := e.Name()
		art.people[n[:len(n)-len(path.Ext(n))]] = ebiten.NewImageFromImage(mustImage("assets/people/" + n))
	}
	art.logo = ebiten.NewImageFromImage(mustImage("assets/ironite-logo.png"))
	for _, s := range []int{16, 32, 48, 64, 256} {
		art.icons = append(art.icons, mustImage(fmt.Sprintf("assets/icon_%d.png", s)))
	}
	for i, n := range []string{"Lora-Regular.ttf", "Lora-Italic.ttf", "Lora-SemiBold.ttf", "Lora-SemiBoldItalic.ttf"} {
		b, err := assetFS.ReadFile("assets/fonts/" + n)
		if err != nil {
			return err
		}
		f, err := opentype.Parse(b)
		if err != nil {
			return err
		}
		fonts[i] = f
	}
	return nil
}

func absI8(v int8) int {
	if v < 0 {
		return int(-v)
	}
	return int(v)
}

// pieceImg picks the stored size closest above the size needed on screen
// (the GPU's mipmaps take care of the rest).
func pieceImg(code int8, logical float64) *ebiten.Image {
	need := dv(logical)
	list := art.pieces[code]
	for i, sz := range pieceSizes {
		if float64(sz) >= need*.98 {
			return list[i]
		}
	}
	return list[len(list)-1]
}

// silhouette makes a soft, blurred black shape of an image's alpha (for shadows).
func silhouette(src image.Image, radius int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	a := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			_, _, _, al := src.At(b.Min.X+x, b.Min.Y+y).RGBA()
			a[y*w+x] = float64(al) / 0xffff
		}
	}
	tmp := make([]float64, w*h)
	for pass := 0; pass < 3; pass++ { // three box blurs ~ a gaussian
		for y := 0; y < h; y++ {
			sum, n := 0.0, 0
			for x := -radius; x < w+radius; x++ {
				if x+radius < w {
					sum += a[y*w+x+radius]
					n++
				}
				if x-radius-1 >= 0 {
					sum -= a[y*w+x-radius-1]
					n--
				}
				if x >= 0 && x < w {
					tmp[y*w+x] = sum / float64(2*radius+1)
				}
			}
		}
		for x := 0; x < w; x++ {
			sum := 0.0
			for y := -radius; y < h+radius; y++ {
				if y+radius < h {
					sum += tmp[(y+radius)*w+x]
				}
				if y-radius-1 >= 0 {
					sum -= tmp[(y-radius-1)*w+x]
				}
				if y >= 0 && y < h {
					a[y*w+x] = sum / float64(2*radius+1)
				}
			}
		}
	}
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for i, v := range a {
		out.Pix[i*4+3] = uint8(clamp01(v) * 255)
	}
	return out
}

// pngBytes encodes an image (used for screenshots in tests).
func pngBytes(im image.Image) []byte {
	var b bytes.Buffer
	_ = png.Encode(&b, im)
	return b.Bytes()
}
