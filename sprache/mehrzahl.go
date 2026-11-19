// SPDX-License-Identifier: GPL-3.0-or-later

package sprache

import (
	"fmt"
	"strconv"
	"strings"
)

// Mehrzahlregel ordnet einer Anzahl n den Index der Mehrzahlform zu (msgstr[Index]).
type Mehrzahlregel struct {
	Formen int
	wurzel knoten
}

// Index liefert die Mehrzahlform für n, begrenzt auf [0, Formen-1]. Der Nullwert verhält sich wie die Regel des
// Englischen.
func (r Mehrzahlregel) Index(n int) int {
	if r.wurzel == nil {
		return englischeRegel.Index(n)
	}
	i := r.wurzel.wert(n)
	if i < 0 || i >= r.Formen {
		return 0
	}
	return i
}

// maxAusdruck ist die größte erlaubte Länge des plural-Ausdrucks.
const maxAusdruck = 1024

// englischeRegel gilt ohne Plural-Forms-Kopf: Einzahl nur bei n == 1.
var englischeRegel = Mehrzahlregel{Formen: 2, wurzel: vergleich{"!=", variable{}, zahl(1)}}

// LeseMehrzahlregel liest den Wert des Kopffelds Plural-Forms, z. B. "nplurals=2; plural=(n != 1);".
// Erlaubt sind die C-Ausdrücke, die gettext dort kennt: n, ganze Zahlen, Klammern, ! % * / + - < <= > >= == != &&
// || und ?:.
func LeseMehrzahlregel(kopf string) (Mehrzahlregel, error) {
	var formen int
	var ausdruck string
	for _, teil := range strings.Split(kopf, ";") {
		name, wert, ok := strings.Cut(strings.TrimSpace(teil), "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(name) {
		case "nplurals":
			n, err := strconv.Atoi(strings.TrimSpace(wert))
			if err != nil || n < 1 || n > 16 {
				return Mehrzahlregel{}, fmt.Errorf("nplurals ungültig: %q", wert)
			}
			formen = n
		case "plural":
			ausdruck = wert
		}
	}
	if formen == 0 || strings.TrimSpace(ausdruck) == "" {
		return Mehrzahlregel{}, fmt.Errorf("Plural-Forms unvollständig: %q", kopf)
	}
	// Die Länge begrenzt zugleich die Tiefe von Lesen und Auswerten (Rekursion); echte Formeln sind kürzer als
	// 300 Zeichen. Ohne Grenze bringt eine Datei mit einer Million Klammern das Programm zum Absturz (F-36).
	if len(ausdruck) > maxAusdruck {
		return Mehrzahlregel{}, fmt.Errorf("Plural-Forms länger als %d Zeichen", maxAusdruck)
	}
	l := &leser{text: ausdruck}
	w, err := l.bedingung()
	if err != nil {
		return Mehrzahlregel{}, err
	}
	if l.zeichen() != 0 {
		return Mehrzahlregel{}, fmt.Errorf("Plural-Forms: unerwartet %q", l.text[l.pos:])
	}
	return Mehrzahlregel{Formen: formen, wurzel: w}, nil
}

type knoten interface{ wert(n int) int }

type zahl int
type variable struct{}
type nicht struct{ a knoten }
type vergleich struct {
	op   string
	a, b knoten
}
type wahl struct{ bed, ja, nein knoten }

func (z zahl) wert(int) int     { return int(z) }
func (variable) wert(n int) int { return n }
func (k nicht) wert(n int) int  { return janein(k.a.wert(n) == 0) }
func (k wahl) wert(n int) int {
	if k.bed.wert(n) != 0 {
		return k.ja.wert(n)
	}
	return k.nein.wert(n)
}
func (k vergleich) wert(n int) int {
	a := k.a.wert(n)
	if k.op == "&&" {
		return janein(a != 0 && k.b.wert(n) != 0)
	}
	if k.op == "||" {
		return janein(a != 0 || k.b.wert(n) != 0)
	}
	b := k.b.wert(n)
	switch k.op {
	case "==":
		return janein(a == b)
	case "!=":
		return janein(a != b)
	case "<":
		return janein(a < b)
	case "<=":
		return janein(a <= b)
	case ">":
		return janein(a > b)
	case ">=":
		return janein(a >= b)
	case "+":
		return a + b
	case "-":
		return a - b
	case "*":
		return a * b
	case "/", "%":
		if b == 0 {
			return 0
		}
		if k.op == "/" {
			return a / b
		}
		return a % b
	}
	return 0
}

func janein(b bool) int {
	if b {
		return 1
	}
	return 0
}

// leser ist ein rekursiver Abstiegsleser mit den Vorrangstufen von C.
type leser struct {
	text string
	pos  int
}

func (l *leser) leer() {
	for l.pos < len(l.text) && strings.IndexByte(" \t\n\r", l.text[l.pos]) >= 0 {
		l.pos++
	}
}

func (l *leser) zeichen() byte {
	l.leer()
	if l.pos < len(l.text) {
		return l.text[l.pos]
	}
	return 0
}

// nimm verbraucht einen der Operatoren (längere zuerst angeben) und liefert ihn, sonst "".
func (l *leser) nimm(ops ...string) string {
	l.leer()
	for _, op := range ops {
		if strings.HasPrefix(l.text[l.pos:], op) {
			l.pos += len(op)
			return op
		}
	}
	return ""
}

func (l *leser) bedingung() (knoten, error) {
	bed, err := l.stufe(0)
	if err != nil {
		return nil, err
	}
	if l.nimm("?") == "" {
		return bed, nil
	}
	ja, err := l.bedingung()
	if err != nil {
		return nil, err
	}
	if l.nimm(":") == "" {
		return nil, fmt.Errorf("Plural-Forms: „:“ fehlt bei Position %d", l.pos)
	}
	nein, err := l.bedingung()
	if err != nil {
		return nil, err
	}
	return wahl{bed, ja, nein}, nil
}

// stufen von schwach nach stark bindend; je Stufe die Operatoren, längere vor kürzeren.
var stufen = [][]string{
	{"||"},
	{"&&"},
	{"==", "!="},
	{"<=", ">=", "<", ">"},
	{"+", "-"},
	{"*", "/", "%"},
}

func (l *leser) stufe(i int) (knoten, error) {
	if i == len(stufen) {
		return l.einstellig()
	}
	a, err := l.stufe(i + 1)
	if err != nil {
		return nil, err
	}
	for {
		op := l.nimm(stufen[i]...)
		if op == "" {
			return a, nil
		}
		b, err := l.stufe(i + 1)
		if err != nil {
			return nil, err
		}
		a = vergleich{op, a, b}
	}
}

func (l *leser) einstellig() (knoten, error) {
	switch c := l.zeichen(); {
	case c == '!' && !strings.HasPrefix(l.text[l.pos:], "!="):
		l.pos++
		a, err := l.einstellig()
		if err != nil {
			return nil, err
		}
		return nicht{a}, nil
	case c == '(':
		l.pos++
		a, err := l.bedingung()
		if err != nil {
			return nil, err
		}
		if l.nimm(")") == "" {
			return nil, fmt.Errorf("Plural-Forms: „)“ fehlt bei Position %d", l.pos)
		}
		return a, nil
	case c == 'n':
		l.pos++
		return variable{}, nil
	case c >= '0' && c <= '9':
		start := l.pos
		for l.pos < len(l.text) && l.text[l.pos] >= '0' && l.text[l.pos] <= '9' {
			l.pos++
		}
		z, err := strconv.Atoi(l.text[start:l.pos])
		if err != nil {
			return nil, fmt.Errorf("Plural-Forms: Zahl %q: %w", l.text[start:l.pos], err)
		}
		return zahl(z), nil
	default:
		return nil, fmt.Errorf("Plural-Forms: unerwartet bei Position %d in %q", l.pos, l.text)
	}
}
