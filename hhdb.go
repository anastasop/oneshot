package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/anastasop/gochess"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font/gofont/gomono"
)

const (
	setFile = "./ChessPiecesArray.png"
)

var (
	pieces     = make(map[string]draw.Image)
	white      = image.NewUniform(color.Gray16{0xffff})
	black      = image.NewUniform(color.Gray16{0x8f8f})
	background = image.NewUniform(color.RGBA{0xCA, 0xA4, 0x72, 255})

	font *truetype.Font

	sigRe     = regexp.MustCompile("[wd]_[0-9]{6}_[0-9]{2,3}_[0-9]{2,3}_[0-9]{5}")
	sqz   int = 30
)

func loadChessSet(fname string) error {
	data, err := ioutil.ReadFile(fname)
	if err != nil {
		return err
	}

	src, err := png.Decode(bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	set := image.NewRGBA(src.Bounds())
	draw.Draw(set, set.Bounds(), src, src.Bounds().Min, draw.Src)

	ff := func(piece string, r image.Rectangle) {
		img := image.NewRGBA(image.Rect(0, 0, sqz, sqz))
		xdraw.BiLinear.Scale(img, img.Bounds(), set.SubImage(r), r, xdraw.Over, nil)
		pieces[piece] = img
	}

	ff("q", image.Rect(0, 0, 60, 60))
	ff("k", image.Rect(60, 0, 120, 60))
	ff("r", image.Rect(120, 0, 180, 60))
	ff("n", image.Rect(180, 0, 240, 60))
	ff("b", image.Rect(240, 0, 300, 60))
	ff("p", image.Rect(300, 0, 360, 60))
	ff("Q", image.Rect(0, 60, 60, 120))
	ff("K", image.Rect(60, 60, 120, 120))
	ff("R", image.Rect(120, 60, 180, 120))
	ff("N", image.Rect(180, 60, 240, 120))
	ff("B", image.Rect(240, 60, 300, 120))
	ff("P", image.Rect(300, 60, 360, 120))

	return nil
}

func writeBoard(game *gochess.Game, fname string) error {
	img := image.NewRGBA(image.Rect(0, 0, 9*sqz, 10*sqz))
	draw.Draw(img, img.Bounds(), background, image.Pt(0, 0), draw.Over)
	board := img.SubImage(image.Rect(sqz/2, sqz+sqz/2, sqz/2+sqz*8, sqz/2+sqz*9)).(*image.RGBA)

	fenParts := strings.Split(game.Tags["FEN"], " ")
	ranks := strings.Split(fenParts[0]+"////////", "/")[0:8]
	colors := [2]image.Image{white, black}
	replacer := strings.NewReplacer(
		"2", "11", "3", "111", "4", "1111", "5", "11111", "6", "111111", "7", "1111111", "8", "11111111")
	for r, rank := range ranks {
		for f, square := range replacer.Replace(rank + "11111111") {
			sq := board.Bounds().Add(image.Pt(f, r).Mul(sqz))
			draw.Draw(board, sq, colors[(f+(r%2))%2], image.ZP, draw.Src)
			if square != '1' {
				if piece, present := pieces[string(square)]; present {
					draw.Draw(board, sq, piece, piece.Bounds().Min, draw.Over)
				} else {
					log.Fatal("Illegal characted in fen: ", square)
				}
			}
		}
	}

	const (
		fontSize = 8
		po       = 14 // page offset
		vs       = 18 // vertical line spacing
		dpi      = 120.0
	)

	ctx := freetype.NewContext()
	ctx.SetFont(font)
	ctx.SetFontSize(fontSize)
	ctx.SetSrc(image.Black)
	ctx.SetDst(img)
	ctx.SetDPI(dpi)
	ctx.SetClip(img.Bounds())

	author := strings.ReplaceAll(game.Tags["White"], "=", " ")
	var summary string
	switch fenParts[1] + game.Tags["Black"][1:2] {
	case "w+":
		summary = "Win. White to play"
	case "b+":
		summary = "Win. Black to play"
	case "w=":
		summary = "Draw. White to play"
	case "b=":
		summary = "Draw. Black to play"
	default:
		summary = "?"
	}

	this, total := variationsCount(game.Moves)
	summary += fmt.Sprintf(" (%d/%d)", this, total)

	lines := [2]string{author, summary}
	for i, line := range lines {
		p := freetype.Pt(po, (i+1)*vs)
		if _, err := ctx.DrawString(line, p); err != nil {
			log.Fatal(err)
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		return err
	}

	if err := ioutil.WriteFile(fname, buf.Bytes(), 0644); err != nil {
		return err
	}

	return nil
}

func variationsCount(variation gochess.Variation) (int, int) {
	total := 0
	thisOnly := 0
	if variation.Plies != nil {
		total = 1
		thisOnly = 1
		for _, ply := range variation.Plies {
			thisOnly += len(ply.Variations)
			for _, v := range ply.Variations {
				_, t := variationsCount(v)
				total += t
			}
		}
	}
	return thisOnly, total
}

func gameSig(game *gochess.Game, numGame int) string {
	gbr := game.Tags["Black"]

	var result string
	switch gbr[1] {
	case '=':
		result = "d"
	case '+':
		result = "w"
	default:
		result = "_"
	}

	this, total := variationsCount(game.Moves)
	sig := fmt.Sprintf("%s_%s_%02d_%02d_%05d", result, gbr[2:6]+gbr[7:9], this, total, numGame)

	if !sigRe.MatchString(sig) {
		log.Println("Game", numGame, "has an incompatible signature")
	}

	return sig
}

func emitGame(game *gochess.Game, numGame int, w io.Writer) {
	sig := gameSig(game, numGame)

	if err := ioutil.WriteFile("./pgn/"+sig+".pgn", game.PGNText, 0644); err != nil {
		log.Fatal("Failed to write pgn: ", err)
	}

	if err := writeBoard(game, "./jpg/"+sig+".jpg"); err != nil {
		log.Fatal("Failed to write png: ", err)
	}

	fen := strings.Join(strings.Fields(game.Tags["FEN"]), "_")

	if _, err := fmt.Fprintf(w, "%s https://lichess.org/analysis/standard/%s\n", sig, fen); err != nil {
		log.Fatal("Failed to write index entry ", err)
	}
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("")
	flag.IntVar(&sqz, "s", 30, "square size")
	flag.Parse()

	f, err := freetype.ParseFont(gomono.TTF)
	if err != nil {
		log.Fatal(err)
	}
	font = f

	if err := loadChessSet(setFile); err != nil {
		log.Fatal(err)
	}

	fin, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatal("Failed to open: ", err)
	}
	defer fin.Close()

	fout, err := os.Create("for_lichess.txt")
	if err != nil {
		log.Fatal("Failed to creage game index")
	}
	defer fout.Close()

	ngame := 0
	parser := gochess.NewParser(fin)
	for {
		game, err := parser.NextGame()
		if err != nil {
			log.Fatal("Failed to parse: ", err)
		}
		if game == nil {
			break
		}
		ngame++
		if err := game.ParseMovesText(); err != nil {
			log.Println("Failed to parse game ", ngame)
			continue
		}

		emitGame(game, ngame, fout)
	}
}
