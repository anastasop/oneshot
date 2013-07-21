
package main

import (
	"errors"
	"io"
	"log"
	"os"
	"github.com/anastasop/gochess"
)

var ErrGamesLimit = errors.New("too much games")

func main() {
	parsePGN(os.Stdin, 50)
}

func parsePGN(pgnReader io.Reader, limit int) error {
	parser := gochess.NewParser(pgnReader)
	var err error
	var pgngame *gochess.Game
	ngames := 0
	for {
		if pgngame, err = parser.NextGame(); err == nil && pgngame != nil  {
			ngames++;
//			log.Printf("%s", pgngame.PGNText)
			if err = pgngame.ParseMovesText(); err == nil {
				board := gochess.NewBoard()
				for _, ply := range pgngame.Moves.Plies {
					if err = board.MakeMove(ply.SAN); err != nil {
						log.Printf("Move Error: %s", err)
						return err
					}
				}
				if ngames >= limit {
					return ErrGamesLimit
				}
			}
			log.Println("Parsed", pgngame.Tags["White"], " - ", pgngame.Tags["Black"])
		}
		if err != nil || pgngame == nil {
			break
		}
	}
	if err != nil {
		log.Printf("PGN parser error: game %d error \"%s\"", ngames, err)
	}
	return err

}
