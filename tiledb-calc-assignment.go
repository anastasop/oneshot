package main

import (
	"fmt"
	"regexp"
	"strconv"
	"unicode"
	"unicode/utf8"
)

// a recursive descent parser for the problem
// I could have used reverse polish notation but because
// i solved the same problem in winter for AdventOfCode2020
// see https://github.com/anastasop/aoc2020/blob/main/18/B.java
// i was familiar with the algorithms
//
// The grammar is
//
// expr -> expr * term | expr / term | term
// term -> term + power | term - power | power
// power -> power ** factor | factor
// factor -> NUMBER | ( expr )
//
// of course for a recursive descent parser it needs left factoring
// and becomes
//
// expr -> term expr1
// expr1 -> * term expr1 | / term expr1 | ε
// term -> power term1
// term1 -> + power term1 | - power term1 | ε
// power -> factor power1
// power1 -> ** factor power1 | ε
// factor -> NUMBER | ( expr )
//
// a expression is composed of white space, digits, + - * / ** ()
//
// all expression parsers return (int, bool)
// int is the value of the expression and bool is true
// if successfully matched and parsed an expression

var verifier = regexp.MustCompile(`^(\d+|\*\*|[*+-/()]|\s+)+$`)
var tokenizer = regexp.MustCompile(`\d+|\*\*|[*+-/()]`)

type Cursor struct {
	tokens []string
	pos    int
}

func NewCursor(str string) *Cursor {
	if !verifier.MatchString(str) {
		return nil
	}
	cur := new(Cursor)
	cur.tokens = tokenizer.FindAllString(str, -1)
	cur.pos = 0
	return cur
}

func (c *Cursor) lookahead() string {
	if c.pos < len(c.tokens) {
		return c.tokens[c.pos]
	}
	return ""
}

func (c *Cursor) advance() {
	c.pos++
}

// follows returns true if the next token is s.
// To match against an arbitrary number use "0"
func (c *Cursor) follows(s string) bool {
	if s == "0" {
		r, _ := utf8.DecodeRuneInString(c.lookahead())
		return unicode.IsDigit(r)
	}
	return s == c.lookahead()
}

func evalExpr(c *Cursor) (int, bool) {
	v, ok := evalTerm(c)
	if !ok {
		return 0, false
	}

	return evalExpr1(v, c)
}

func evalExpr1(v int, c *Cursor) (int, bool) {
	if c.follows("+") {
		c.advance()
		a, okA := evalTerm(c)
		if !okA {
			return 0, false
		}
		v += a
		return evalExpr1(v, c)
	} else if c.follows("-") {
		c.advance()
		a, okA := evalTerm(c)
		if !okA {
			return 0, false
		}
		v -= a
		return evalExpr1(v, c)
	}

	return v, true
}

func evalTerm(c *Cursor) (int, bool) {
	a, ok := evalPower(c)
	if !ok {
		return 0, false
	}

	return evalTerm1(a, c)
}

func evalTerm1(v int, c *Cursor) (int, bool) {
	if c.follows("*") {
		c.advance()
		a, okA := evalPower(c)
		if !okA {
			return 0, false
		}
		v *= a
		return evalTerm1(v, c)
	} else if c.follows("/") {
		c.advance()
		a, okA := evalPower(c)
		if !okA {
			return 0, false
		}
		v /= a
		return evalTerm1(v, c)
	}

	return v, true
}

func evalPower(c *Cursor) (int, bool) {
	a, ok := evalFactor(c)
	if !ok {
		return 0, false
	}

	return evalPower1(a, c)
}

func evalPower1(v int, c *Cursor) (int, bool) {
	if c.follows("**") {
		c.advance()
		a, okA := evalFactor(c)
		if !okA {
			return 0, false
		}
		v = power(v, a)
		return evalPower1(v, c)
	}

	return v, true
}

func evalFactor(c *Cursor) (int, bool) {
	if c.follows("0") {
		s := c.lookahead()
		v, _ := strconv.Atoi(s) // tokenizer ensures this is number
		c.advance()
		return v, true
	} else if c.follows("(") {
		c.advance()
		v, ok := evalExpr(c)
		if !ok {
			return 0, false
		}
		if !c.follows(")") {
			return 0, false
		}
		c.advance()
		return v, true
	} else {
		return 0, false
	}
}

func power(a, b int) int {
	p := 1
	for i := 0; i < b; i++ {
		p *= a
	}
	return p
}

func StringCalculate(str string) string {
	c := NewCursor(str)
	if c == nil {
		return "failed to tokenize input"
	}

	if res, ok := evalExpr(c); ok {
		return strconv.Itoa(res)
	}
	return "failed to parse expression"
}

func main() {
	fmt.Println(StringCalculate("(1+2**2)*(3+4**2)"))
}
