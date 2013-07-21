
package main

import (
	"fmt"
)

const (
	kInitialPosition =
		"XXXXXXXXXXX" +
		"XXXXXXXXXXX" +
		"XXXX***XXXX" +
		"XXXX***XXXX" +
		"XX*******XX" +
		"XX***_***XX" +
		"XX*******XX" +
		"XXXX***XXXX" +
		"XXXX***XXXX" +
		"XXXXXXXXXXX" +
		"XXXXXXXXXXX"
	kPawn = '*'
	kHole = '_'
)

var examinedPositions = make(map[string]int)

type position []byte

func (p position) String() string {
	s := "|"
	for i := 2; i < 9; i++ {
		s += string(p[i * 11 + 2: i * 11 + 9])
		s += "|"
	}
	return s
}

func updatedPosition(pos position, pawn, hole, captured int) position {
	p := make([]byte, len(pos))
	copy(p, pos)
	p[pawn] = kHole
	p[hole] = kPawn
	p[captured] = kHole
	return p
}

func generatePositions(pos position) {
	if _, seen := examinedPositions[string(pos)]; seen {
		return
	} else {
		examinedPositions[string(pos)] = 1
	}
	
	for i := 22; i < 99; i++ {
		if pos[i] == kPawn && pos[i - 2] == kHole && pos[i - 1] == kPawn {
			upos := updatedPosition(pos, i, i - 2, i - 1)
			fmt.Println(pos, upos, "L")
			generatePositions(upos)
		}
		if pos[i] == kPawn && pos[i + 2] == kHole && pos[i + 1] == kPawn {
			upos := updatedPosition(pos, i, i + 2, i + 1)
			fmt.Println(pos, upos, "R")
			generatePositions(upos)
		}
		if pos[i] == kPawn && pos[i - 22] == kHole && pos[i - 11] == kPawn {
			upos := updatedPosition(pos, i, i - 22, i - 11)
			fmt.Println(pos, upos, "U")
			generatePositions(upos)
		}
		if pos[i] == kPawn && pos[i + 22] == kHole && pos[i + 11] == kPawn {
			upos := updatedPosition(pos, i, i + 22, i + 11)
			fmt.Println(pos, upos, "D")
			generatePositions(upos)
		}
	}
}

func main() {
	generatePositions([]byte(kInitialPosition))
}
