
package main

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

type UCIEngine struct {
	cmd *exec.Cmd
	r *bufio.Reader
	w io.Writer
}

type tUCIInfo struct {
	MultiPV int
	PV []string
	Score float32
	MateInMoves int

	Nodes int
	Depth int
	SelectiveDepth int
	Time int

	CurrMove string
	CurrMoveNumber int
	CurrLine []string
	Refutation []string

	Hashfull float32
	NodesPerSec int
	TableBaseHits int
	TableBaseHitsShredder int
	Cpuload float32
	Informational string
}

type UCIVariation struct {
	Score float32
	MateInMoves int
	Moves []string
}

type UCIPositionEvaluation struct {
	BestMove string
	PonderMove string
	Variations []UCIVariation
}

func parseInfoLine(line string) tUCIInfo {
	tokens := append(strings.Fields(line), "", "", "", "", "")
	var info tUCIInfo
	info.MultiPV = 1
	for i := 1; i < len(tokens); {
	    switch tokens[i] {
	    case "depth":
		i++
		if v, err := strconv.Atoi(tokens[i]); err == nil {
			info.Depth = v
			i++
		}
	    case "seldepth":
		i++
		if v, err := strconv.Atoi(tokens[i]); err == nil {
			info.SelectiveDepth = v
			i++
		}
	    case "time":
		i++
		if v, err := strconv.Atoi(tokens[i]); err == nil {
			info.Time = v
			i++
		}
	    case "nodes":
		i++
		if v, err := strconv.Atoi(tokens[i]); err == nil {
			info.Nodes = v
			i++
		}
	    case "pv":
		s := make([]string, 0)
		for i += 1; i < len(tokens) && strings.IndexAny(tokens[i], "12345678") >= 0; i++ {
			s = append(s, tokens[i])
		}
		if len(s) > 0 {
			info.PV = s
		}
	    case "multipv":
		i++
		if v, err := strconv.Atoi(tokens[i]); err == nil {
			info.MultiPV = v
			i++
		}
	    case "score":
		i++
		loop: for {
			switch tokens[i] {
			case "lowerbound", "upperbound":
				i++
			case "cp":
				i++
				if v, err := strconv.ParseFloat(tokens[i], 32); err == nil {
					info.Score = float32(v)
					i++
				}
			case "mate":
				i++
				if v, err := strconv.Atoi(tokens[i]); err == nil {
					info.MateInMoves = v
					i++
				}
			default:
				break loop
			}
		}
	    case "currmove":
		info.CurrMove = tokens[i + 1]
		i += 2
	    case "currmovenumber":
		i++
		if v, err := strconv.Atoi(tokens[i]); err == nil {
			info.CurrMoveNumber = v
			i++
		}
	    case "hashfull":
		i++
		if v, err := strconv.ParseFloat(tokens[i], 32); err == nil {
			info.Hashfull = float32(v)
			i++
		}
	    case "nps":
		i++
		if v, err := strconv.Atoi(tokens[i]); err == nil {
			info.NodesPerSec = v
			i++
		}
	    case "tbhits":
		i++
		if v, err := strconv.Atoi(tokens[i]); err == nil {
			info.TableBaseHits = v
			i++
		}
	    case "sbhits":
		i++
		if v, err := strconv.Atoi(tokens[i]); err == nil {
			info.TableBaseHitsShredder = v
			i++
		}
	    case "cpuload":
		i++
		if v, err := strconv.ParseFloat(tokens[i], 32); err == nil {
			info.Cpuload = float32(v)
			i++
		}
	    case "string":
		info.Informational = tokens[i + 1]
		i += 2
	    case "refutation":
		s := make([]string, 0)
		for i += 1; i < len(tokens) && strings.IndexAny(tokens[i], "12345678") >= 0; i++ {
			s = append(s, tokens[i])
		}
		if len(s) > 0 {
			info.Refutation = s
		}
	    case "currline":
		s := make([]string, 0)
		for i += 1; i < len(tokens) && strings.IndexAny(tokens[i], "12345678") >= 0; i++ {
			s = append(s, tokens[i])
		}
		if len(s) > 0 {
			info.CurrLine = s
		}
	    default:
		i++
	    }
	}
	return info
}

func parseBestMoveLine(line string) (bestmove, ponder string) {
	tokens := append(strings.Fields(line), "", "", "", "", "")
	if tokens[0] == "bestmove" {
		if tokens[2] == "ponder" {
			return tokens[1], tokens[3]
		}
		return tokens[1], ""
	}
	return "", ""
}

func (eng *UCIEngine) waitEngineToGetReady() (err error) {
	var line string
	if _, err = fmt.Fprintln(eng.w, "isready"); err == nil {
		if line, err = eng.r.ReadString('\n'); err == nil {
			if !strings.HasPrefix(line, "readyok") {
				err = fmt.Errorf("waitEngineToGetReady did not return 'readyok' but '%s'", line)
			}
		}
	}
	return
}

func (eng *UCIEngine) resetEngine(options map[string]string) error {
	if _, err := fmt.Fprintln(eng.w, "uci"); err != nil {
		return err
	}
	for {
		line, err := eng.r.ReadString('\n')
		if err != nil {
			return err
		}
		if strings.HasPrefix(line, "uciok") {
			break
		}
	}
	if options != nil {
		for k, v := range options {
			if _, err := fmt.Fprintln(eng.w, "setoption name", k, "value", v); err != nil {
				return err
			}
		}
	}
	if err := eng.waitEngineToGetReady(); err != nil {
		return err
	}
	return nil
}


func NewUCIEngine(path string, options map[string]string) (*UCIEngine, error) {
	eng := new(UCIEngine)
	eng.cmd = exec.Command(path)
	if r, err := eng.cmd.StdoutPipe(); err == nil {
		eng.r = bufio.NewReader(r)
	} else {
		return nil, err
	}
	if w, err := eng.cmd.StdinPipe(); err == nil {
		eng.w = w
	} else {
		return nil, err
	}

	if err := eng.cmd.Start(); err != nil {
		return nil, err
	}
	eng.resetEngine(options)
	return eng, nil
}

func (eng *UCIEngine) EvaluatePosition(newgame bool, fen string, moves []string, msec int) (UCIPositionEvaluation, error) {
	var eval UCIPositionEvaluation
	if newgame {
		if _, err := fmt.Fprintln(eng.w, "ucinewgame"); err != nil {
			return eval, err
		}
	}
	cmd := ""
	if fen == "startpos" {
		cmd += "position startpos"
	} else {
		cmd += "position fen " + fen
	}
	if moves != nil {
		cmd += " moves"
		for _, move := range moves {
			cmd += " " + move
		}
	}
	if _, err := fmt.Fprintln(eng.w, cmd); err != nil {
		return eval, err
	}

	if err := eng.waitEngineToGetReady(); err != nil {
		return eval, err
	}
	if _, err := fmt.Fprintln(eng.w, "go movetime", msec); err != nil {
		return eval, err
	}
 
	var pvs [32]tUCIInfo
	for {
		line, err := eng.r.ReadString('\n')
		if err != nil {
			return eval, err
		}
		if strings.HasPrefix(line, "info") {
			info := parseInfoLine(line[4:])
			if info.PV != nil {
				pvs[info.MultiPV] = info
			}
		} else if strings.HasPrefix(line, "bestmove") {
			b, p := parseBestMoveLine(line)
			eval.BestMove = b
			eval.PonderMove = p
			break
		} else {
			return eval, fmt.Errorf("cannot understand line: '%s'", line)
		}
	}
	for i := 1; pvs[i].MultiPV != 0; i++ {
		variant := UCIVariation{Score: pvs[i].Score, MateInMoves: pvs[i].MateInMoves, Moves: pvs[i].PV}
		eval.Variations = append(eval.Variations, variant)
	}
	return eval, nil
}

func (eng *UCIEngine) Close() error {
	fmt.Fprintln(eng.w, "quit")
	return eng.cmd.Wait()
}

func main() {
//	eng, err := NewUCIEngine("C:\\home\\bin\\Houdini_3_w32.exe", nil)
//	eng, err := NewUCIEngine("C:\\home\\bin\\stockfish-231-32-ja.exe", nil)
	eng, err := NewUCIEngine("/home/spyros/bin/stockfish", nil)
	if err != nil {
		fmt.Println("error: ", err)
		return
	}
	fen := "kbK5/pp6/1P6/8/8/8/8/R7 w - - 0 1"
	eval, err := eng.EvaluatePosition(true, fen, nil, 5000)
	if err == nil {
		for _, v := range eval.Variations {
			fmt.Printf("Evaluation: score: %v mate: %v variation: %v\n", v.Score, v.MateInMoves, v.Moves)
		}
	} else {
		fmt.Println("error: ", err)
	}
	if err := eng.Close(); err != nil {
		fmt.Println("error: ", err)
	}
}
