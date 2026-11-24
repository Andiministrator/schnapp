// SPDX-License-Identifier: GPL-3.0-or-later

package main

// Logo „schnapp“ (gewählt: Runde 6 „e6“ = e4 mit „SCREENSHOTS FOR LINUX“, Andi 19.11.2026). Übernommen aus den
// Entwurfs-Skripten runde1/4/5/6 (nur, was das Paket braucht); deren Ergebnisse liegen weiter unter brand/entwuerfe/.
// Schriftzug: Monoline mit runden Enden, Bögen als abgerundete Rechtecke. Maße: x-Höhe 100 (y 100–200), Oberlänge
// bis y 40, Unterlänge bis y 262, Strich 22.

import (
	"fmt"
	"strings"
)

const (
	sw   = 22      // Strichstärke
	bw   = 76      // Breite eines Bogens (Mittellinie)
	br   = 30      // Eckenradius der Bögen
	lauf = bw + 40 // Vorschub je Buchstabe
)

// farben einer Fassung (Runde 4).
type farben map[string]string

var (
	dunkel = farben{"grund": "#1E2124", "schrift": "#ECEAE5", "kroko": "#4F9479", "maul": "#2C5545",
		"akzent": "#F5A65B", "pupille": "#1E2124", "weiss": "#FFFFFF"}
	hell = farben{"grund": "#FFFFFF", "schrift": "#1E2124", "kroko": "#3B7A5F", "maul": "#24473A",
		"akzent": "#E0893A", "pupille": "#1E2124", "weiss": "#FFFFFF"}
	// neben: Farbe der Unterzeile; grau-400 auf dunkel, auf Weiß dunkler für Kontrast.
	neben = map[string]string{"dunkel": "#92989F", "hell": "#5E646B"}
)

func kasten(x int) string {
	const oben, unten, w, r = 100, 200, bw, br
	return fmt.Sprintf("M%d,%d H%d A%d,%d 0 0 1 %d,%d V%d A%d,%d 0 0 1 %d,%d H%d A%d,%d 0 0 1 %d,%d V%d A%d,%d 0 0 1 %d,%d Z",
		x+r, oben, x+w-r, r, r, x+w, oben+r, unten-r, r, r, x+w-r, unten, x+r, r, r, x, unten-r, oben+r, r, r, x+r, oben)
}

func buchstabe(z rune, x int) []string {
	const w, r = bw, br
	switch z {
	case 's':
		q := 25
		return []string{fmt.Sprintf("M%d,%d H%d A%d,%d 0 0 0 %d,%d H%d A%d,%d 0 0 1 %d,%d H%d",
			x+w, 100, x+q, q, q, x+q, 150, x+w-q, q, q, x+w-q, 200, x)}
	case 'c':
		return []string{fmt.Sprintf("M%d,%d H%d A%d,%d 0 0 0 %d,%d V%d A%d,%d 0 0 0 %d,%d H%d",
			x+w, 100, x+r, r, r, x, 100+r, 200-r, r, r, x+r, 200, x+w)}
	case 'h':
		return []string{fmt.Sprintf("M%d,40 V200", x),
			fmt.Sprintf("M%d,%d A%d,%d 0 0 1 %d,100 H%d A%d,%d 0 0 1 %d,%d V200", x, 100+r, r, r, x+r, x+w-r, r, r, x+w, 100+r)}
	case 'n':
		return []string{fmt.Sprintf("M%d,200 V100 M%d,%d A%d,%d 0 0 1 %d,100 H%d A%d,%d 0 0 1 %d,%d V200",
			x, x, 100+r, r, r, x+r, x+w-r, r, r, x+w, 100+r)}
	case 'a':
		return []string{kasten(x), fmt.Sprintf("M%d,100 V200", x+w)}
	case 'p':
		return []string{kasten(x), fmt.Sprintf("M%d,100 V262", x)}
	}
	panic(fmt.Sprintf("buchstabe %q", z))
}

func zug(d, farbe string) string {
	return fmt.Sprintf(`<path d="%s" fill="none" stroke="%s" stroke-width="%d" stroke-linecap="round" stroke-linejoin="round"/>`,
		d, farbe, sw)
}

// sMitAuge ist das orange s; blick verschiebt die Pupille nach rechts (0 mittig, 1 ganz rechts).
func sMitAuge(x int, f farben, blick float64) string {
	var b strings.Builder
	for _, d := range buchstabe('s', x) {
		b.WriteString(zug(d, f["akzent"]))
	}
	// Auge seit Karte #217 (Variante B, Andi 24.11.2026) größer: Ring r 15, Pupille r 7,5.
	fmt.Fprintf(&b, `<circle cx="%d" cy="125" r="15" fill="%s" stroke="%s" stroke-width="2"/>`, x+30, f["weiss"], f["akzent"])
	fmt.Fprintf(&b, `<circle cx="%.1f" cy="125" r="7.5" fill="%s"/>`, float64(x)+27.5+7.5*blick, f["pupille"])
	return b.String()
}

// schriftzug: „schnapp“ mit dem s mit Auge (Blick nach rechts); versatz verschiebt einzelne Buchstaben senkrecht.
func schriftzug(f farben, versatz map[rune]int) (string, int) {
	var b strings.Builder
	b.WriteString(sMitAuge(40, f, 1.0))
	x := 40 + lauf
	for _, z := range "chnapp" {
		var neu strings.Builder
		for _, d := range buchstabe(z, x) {
			neu.WriteString(zug(d, f["schrift"]))
		}
		if dy := versatz[z]; dy != 0 {
			fmt.Fprintf(&b, `<g transform="translate(0,%d)">%s</g>`, dy, neu.String())
		} else {
			b.WriteString(neu.String())
		}
		x += lauf
	}
	return b.String(), x - 40
}
