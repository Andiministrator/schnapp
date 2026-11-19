// SPDX-License-Identifier: GPL-3.0-or-later

// Paket geo rechnet so, wie Python es tut: math.dist/hypot nach dem Verfahren von
// CPython (vector_norm, fast immer korrekt gerundet), sum() mit Neumaier-Ausgleich, Modulo mit dem Vorzeichen des
// Teilers, radians/degrees über die doppelt genau gerundete Konstante. So bleiben erzeugte SVGs bytegleich zu
// Python-Werkzeugen; wo Go und die C-Bibliothek in der letzten Binärstelle abweichen (sin, cos, atan2, exp),
// verschwindet das beim Runden auf zwei Nachkommastellen.
package geo

import (
	"math"
	"strconv"
	"strings"
)

// P ist ein Punkt.
type P struct{ X, Y float64 }

var (
	piD       = math.Pi
	gradZuRad = piD / 180.0
	radZuGrad = 180.0 / piD
)

// Radians wie math.radians.
func Radians(g float64) float64 { return g * gradZuRad }

// Degrees wie math.degrees.
func Degrees(r float64) float64 { return r * radZuGrad }

// Mod wie % für float in Python (Ergebnis mit dem Vorzeichen des Teilers).
func Mod(a, b float64) float64 {
	m := math.Mod(a, b)
	if m != 0 {
		if (b < 0) != (m < 0) {
			m += b
		}
	} else {
		m = math.Copysign(0, b)
	}
	return m
}

type dl struct{ hi, lo float64 }

func dlMul(x, y float64) dl {
	h := x * y
	return dl{h, math.FMA(x, y, -h)}
}

func dlFastSum(a, b float64) dl {
	h := a + b
	return dl{h, (a - h) + b}
}

// norm wie vector_norm in CPython (Modules/mathmodule.c).
func norm(vec []float64, max float64) float64 {
	if math.IsInf(max, 0) {
		return max
	}
	if max == 0 || len(vec) <= 1 {
		return max
	}
	_, maxE := math.Frexp(max)
	if maxE < -1023 {
		const dblMin = 2.2250738585072014e-308
		v := make([]float64, len(vec))
		for i, x := range vec {
			v[i] = x / dblMin
		}
		return dblMin * norm(v, max/dblMin)
	}
	scale := math.Ldexp(1.0, -maxE)
	csum, frac1, frac2 := 1.0, 0.0, 0.0
	for _, x := range vec {
		x *= scale
		pr := dlMul(x, x)
		sm := dlFastSum(csum, pr.hi)
		csum = sm.hi
		frac1 += pr.lo
		frac2 += sm.lo
	}
	h := math.Sqrt(csum - 1.0 + (frac1 + frac2))
	pr := dlMul(-h, h)
	sm := dlFastSum(csum, pr.hi)
	csum = sm.hi
	frac1 += pr.lo
	frac2 += sm.lo
	x := csum - 1.0 + (frac1 + frac2)
	h += x / (2.0 * h)
	return h / scale
}

// Hypot wie math.hypot(x, y).
func Hypot(x, y float64) float64 {
	ax, ay := math.Abs(x), math.Abs(y)
	if math.IsNaN(ax) || math.IsNaN(ay) {
		if math.IsInf(ax, 0) || math.IsInf(ay, 0) {
			return math.Inf(1)
		}
		return math.NaN()
	}
	return norm([]float64{ax, ay}, math.Max(ax, ay))
}

// Dist wie math.dist(p, q).
func Dist(p, q P) float64 { return Hypot(p.X-q.X, p.Y-q.Y) }

// Summe wie sum() über Kommazahlen in Python 3.12+ (Startwert 0, Neumaier-Ausgleich).
type Summe struct {
	hi, lo float64
	ganz   bool
	i      int64
}

// NeueSumme beginnt bei der ganzen Zahl 0 wie sum().
func NeueSumme() *Summe { return &Summe{ganz: true} }

// AddGanz addiert eine ganze Zahl.
func (s *Summe) AddGanz(v int64) {
	if s.ganz {
		s.i += v
		return
	}
	s.add(float64(v))
}

// Add addiert eine Kommazahl.
func (s *Summe) Add(x float64) {
	if s.ganz {
		s.ganz = false
		s.hi = float64(s.i) + x
		return
	}
	s.add(x)
}

func (s *Summe) add(x float64) {
	t := s.hi + x
	if math.Abs(s.hi) >= math.Abs(x) {
		s.lo += (s.hi - t) + x
	} else {
		s.lo += (x - t) + s.hi
	}
	s.hi = t
}

// Wert: Ergebnis als Kommazahl (ganz: float der Summe) und ob es ganz blieb.
func (s *Summe) Wert() (float64, bool) {
	if s.ganz {
		return float64(s.i), true
	}
	if s.lo != 0 && !math.IsInf(s.lo, 0) && !math.IsNaN(s.lo) {
		return s.hi + s.lo, false
	}
	return s.hi, false
}

// SummeF: sum() über Kommazahlen.
func SummeF(xs ...float64) float64 {
	s := NeueSumme()
	for _, x := range xs {
		s.Add(x)
	}
	v, _ := s.Wert()
	return v
}

// Ceil wie math.ceil (ganze Zahl).
func Ceil(x float64) int { return int(math.Ceil(x)) }

// Pow wie x ** y für Kommazahlen.
func Pow(x, y float64) float64 { return math.Pow(x, y) }

// F2 schreibt wie f"{v:.2f}".
func F2(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }

// F1 schreibt wie f"{v:.1f}".
func F1(v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) }

// Zahl: zwei Nachkommastellen, Nullen am Ende weg, "-0" wird "0".
func Zahl(v float64) string {
	s := strings.TrimRight(F2(v), "0")
	s = strings.TrimRight(s, ".")
	if s == "-0" || s == "" {
		return "0"
	}
	return s
}

// Zahl1: eine Nachkommastelle, ".0" am Ende weg ("-0" bleibt).
func Zahl1(v float64) string {
	s := F1(v)
	if strings.HasSuffix(s, ".0") {
		return s[:len(s)-2]
	}
	return s
}

// Rund wie round(v, n) in Python (Kommazahl).
func Rund(v float64, n int) float64 {
	f, _ := strconv.ParseFloat(strconv.FormatFloat(v, 'f', n, 64), 64)
	return f
}

// Escape wie html.escape(s, quote=True).
func Escape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#x27;")
	return r.Replace(s)
}

// EscapeOhneQuote wie html.escape(s, quote=False).
func EscapeOhneQuote(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

// Repr schreibt eine Kommazahl wie repr() in Python: kürzeste eindeutige Darstellung, ".0" bei ganzen Zahlen,
// Exponentschreibweise unter 1e-4 und ab 1e16, "inf"/"-inf"/"nan".
func Repr(f float64) string {
	switch {
	case math.IsInf(f, 1):
		return "inf"
	case math.IsInf(f, -1):
		return "-inf"
	case math.IsNaN(f):
		return "nan"
	}
	e := strconv.FormatFloat(f, 'e', -1, 64)
	exp, _ := strconv.Atoi(e[strings.LastIndexByte(e, 'e')+1:])
	if exp < -4 || exp >= 16 {
		return e
	}
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}
