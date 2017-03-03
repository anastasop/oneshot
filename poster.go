package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strings"

	svg "github.com/ajstarks/svgo"
	"github.com/nfnt/resize"
)

var imgFile = flag.String("i", "./sherlock.jpg", "image file")
var txtFile = flag.String("t", "./advs.txt", "text file")
var svgFile = flag.String("o", "./image.svg", "svg file")
var ratio = flag.Float64("s", 0.6, "ratio")
var xscale = flag.Float64("x", 1.667, "x scale")

func readImage(fname string) image.Image {
	fin, err := os.Open(fname)
	if err != nil {
		log.Fatal(err)
	}
	defer fin.Close()

	img, _, err := image.Decode(fin)
	if err != nil {
		log.Fatal(err)
	}

	img500 := resize.Resize(0, 500, img, resize.Lanczos3)
	return resize.Resize(uint(float64(img.Bounds().Dx())**xscale), uint(img.Bounds().Dy()), img500, resize.Bilinear)
}

func readText(fname string) string {
	buf, err := ioutil.ReadFile(fname)
	if err != nil {
		log.Fatal(err)
	}

	return strings.Join(strings.Fields(string(buf)), " ")
}

func sameColor(a, b color.Color) bool {
	ar, ag, ab, _ := a.RGBA()
	br, bg, bb, _ := b.RGBA()

	return ar == br && ag == bg && ab == bb
}

func emitSVG(canvas *svg.SVG, img image.Image, txt string, ratio float64) {
	b := img.Bounds()
	ti := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		ix := b.Min.X
		prev := img.At(b.Min.X, y)
		var curr color.Color
		for x := b.Min.X + 1; x < b.Max.X; x++ {
			curr = color.RGBAModel.Convert(img.At(x, y))
			if !sameColor(prev, curr) {
				r, g, b, _ := prev.RGBA()
				canvas.Text(int(math.Ceil(float64(ix)*ratio)), y, txt[ti:ti+x-ix],
					fmt.Sprintf("fill:rgb(%d, %d, %d)", uint8(r), uint8(g), uint8(b)))
				ti += x - ix
				ix = x
				prev = curr
			}
		}

		r, g, bb, _ := curr.RGBA()
		if b.Max.X-ix > 0 {
			canvas.Text(int(math.Ceil(float64(ix)*ratio)), y, txt[ti:ti+b.Max.X-ix],
				fmt.Sprintf("fill:rgb(%d, %d, %d)", uint8(r), uint8(g), uint8(bb)))
			ti += b.Max.X - ix
		}
	}
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("poster: ")
	flag.Parse()

	img := readImage(*imgFile)
	txt := readText(*txtFile)

	fout, err := os.Create(*svgFile)
	if err != nil {
		log.Fatal(err)
	}
	defer fout.Close()

	canvas := svg.New(fout)
	canvas.Start(int(float64(img.Bounds().Dx())**ratio), img.Bounds().Dy(),
		`xml:space="preserve"`,
		fmt.Sprintf(`viewbox="0 0 %d %d"`, int(float64(img.Bounds().Dx())**ratio), img.Bounds().Dy()),
		`style="font-family: 'Source Code Pro'; font-size: 1; font-weight: 900;"`)
	emitSVG(canvas, img, txt, *ratio)
	canvas.End()
}
