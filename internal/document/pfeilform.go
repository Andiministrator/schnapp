// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"fmt"
	"math"
)

// Kopfart ist die Form der Spitze eines Pfeils (ADR-0021 Nachtrag 19). Der Nullwert ist das gefüllte Dreieck.
type Kopfart uint8

const (
	KopfDreieck Kopfart = iota // gefülltes Dreieck (Voreinstellung)
	KopfWinkel                 // offene Spitze: Umriss des Dreiecks, Schenkel so dick wie der Strich
	KopfPunkt                  // Scheibe um das Ende
	KopfQuer                   // Balken quer zur Richtung
)

// Kopfarten sind alle Spitzenarten in der Reihenfolge der Bedienung.
var Kopfarten = []Kopfart{KopfDreieck, KopfWinkel, KopfPunkt, KopfQuer}

// PfeilEnden sagt, an welchen Enden ein Pfeil eine Spitze trägt. Der Nullwert ist die Spitze bei B.
type PfeilEnden uint8

const (
	EndenEnde   PfeilEnden = iota // Spitze bei B (Voreinstellung)
	EndenAnfang                   // Spitze bei A
	EndenBeide                    // an beiden Enden
	EndenKeine                    // keine Spitze: eine Linie
)

// PfeilEndenAlle sind alle Möglichkeiten in der Reihenfolge der Bedienung.
var PfeilEndenAlle = []PfeilEnden{EndenKeine, EndenEnde, EndenAnfang, EndenBeide}

// pruefeForm meldet unbekannte Werte von Kopf und Enden.
func (p Pfeil) pruefeForm() error {
	if p.Kopf > KopfQuer || p.Enden > EndenKeine {
		return fmt.Errorf("document: Pfeil mit unbekannter Spitze %d/%d", p.Kopf, p.Enden)
	}
	return nil
}

// Kopfform ist eine Spitze: am Ende Spitze (B oder A) in Richtung Richtung (normiert, vom Schaft nach außen). Schaftende
// ist der Punkt auf der Kurve, an dem der Schaft endet. Kontur läuft von rechts (bezogen auf Richtung) über vorn nach
// links; der Umriss des Kopfs ist Schaftende − n·hw, Kontur, Schaftende + n·hw mit n = links der Richtung. Die beiden
// Schaftpunkte sind dieselben wie die Enden der runden Kappe des Linienstrichs, die der Kopf ersetzt.
type Kopfform struct {
	Spitze, Richtung, Schaftende Punkt
	// SchaftNormale ist die Normale links der letzten Schaftstrecke, nach außen gerichtet (bei einem stark gebogenen
	// Pfeil mit großem Kopf weicht sie von der Sehne ab); ohne Schaft die links der Richtung.
	SchaftNormale Punkt
	Kontur        []Punkt
	AmAnfang      bool // der Kopf sitzt bei A (sonst bei B)
}

// Umriss ist das geschlossene Vieleck des Kopfs zum Strich der halben Breite hw: Schaftende rechts, Kontur, Schaftende
// links — dieselben Punkte wie die runde Kappe des Linienstrichs, die der Kopf ersetzt.
func (k Kopfform) Umriss(hw float64) []Punkt {
	n := k.SchaftNormale
	u := make([]Punkt, 0, len(k.Kontur)+2)
	u = append(u, k.Schaftende.Minus(n.Mal(hw)))
	u = append(u, k.Kontur...)
	return append(u, k.Schaftende.Plus(n.Mal(hw)))
}

// PfeilForm ist die gezeichnete Gestalt eines Pfeils: Schaft als Linienzug (Original) und die Köpfe.
type PfeilForm struct {
	Schaft []Punkt
	Koepfe []Kopfform
	TA, TB float64 // Kurvenparameter von Anfang und Ende des Schafts (TB <= TA: kein Schaft)
}

// pfeilSchritte ist die Zahl der Strecken, in die Renderer, Editor und Treffer die Kurve teilen.
const pfeilSchritte = 48

// links ist die Normale links der Richtung r (wie die Normale des Linienstrichs, ADR-0023 Nachtrag 22).
func links(r Punkt) Punkt { return Pt(-r.Y, r.X) }

// KopfRueckstand ist der Abstand vom Ende des Pfeils bis zum Ende des Schafts bei Strichstärke w.
func KopfRueckstand(art Kopfart, w float64) float64 {
	hw := w / 2
	switch art {
	case KopfWinkel:
		l, h := 3*w+5, (4.5*w+6)/2
		sin, tan := h/math.Hypot(l, h), h/l
		return w/sin + hw/tan
	case KopfPunkt:
		r := 1.5*w + 3
		return math.Sqrt(max(0, r*r-hw*hw))
	case KopfQuer:
		return hw
	}
	return 3*w + 5
}

// Form liefert Schaft und Köpfe des Pfeils p (ADR-0021 Nachtrag 19). Der Schaft endet unter jedem Kopf auf der Kurve;
// die Richtung eines Kopfs ist die Sehne vom Schaftende zum Ende des Pfeils. Ist der Pfeil kürzer als ein Kopf, endet
// der Schaft am anderen Ende bzw. entfällt.
func (p Pfeil) Form() PfeilForm {
	w := p.Strich.Breite
	rueck := KopfRueckstand(p.Kopf, w)
	ta, tb := 0.0, 1.0
	var koepfe []Kopfform
	if p.Enden == EndenEnde || p.Enden == EndenBeide {
		t := p.suchePunkt(true, rueck)
		koepfe = append(koepfe, p.kopf(p.B, p.Punkt(t), w))
		tb = t
	}
	if p.Enden == EndenAnfang || p.Enden == EndenBeide {
		t := p.suchePunkt(false, rueck)
		k := p.kopf(p.A, p.Punkt(t), w)
		k.AmAnfang = true
		koepfe = append(koepfe, k)
		ta = t
	}
	if tb <= ta && len(koepfe) > 0 {
		// Kürzer als die Köpfe: kein Schaft. Jeder Kopf bekommt seine volle Form mit einem gedachten Schaftende auf der
		// Sehne zum anderen Ende (sonst läge es näher an der Spitze als der Kopf lang ist).
		for i, k := range koepfe {
			r := k.Spitze.Minus(p.andereEnde(k.Spitze))
			if l := r.Laenge(); l >= 1e-9 {
				r = r.Mal(1 / l)
			} else {
				r = Pt(1, 0)
			}
			neu := p.kopf(k.Spitze, k.Spitze.Minus(r.Mal(rueck)), w)
			neu.AmAnfang = k.AmAnfang
			koepfe[i] = neu
		}
	}
	var schaft []Punkt
	if tb > ta {
		schaft = make([]Punkt, 0, pfeilSchritte+1)
		for i := 0; i <= pfeilSchritte; i++ {
			t := ta + (tb-ta)*float64(i)/pfeilSchritte
			if i == pfeilSchritte {
				t = tb // genau das Schaftende der Köpfe
			}
			schaft = append(schaft, p.Punkt(t))
		}
	}
	for i := range koepfe {
		koepfe[i].SchaftNormale = schaftNormale(schaft, koepfe[i])
	}
	return PfeilForm{Schaft: schaft, Koepfe: koepfe, TA: ta, TB: tb}
}

// schaftNormale ist die Normale links der nach außen gerichteten Schaftstrecke am Kopf k, gerechnet wie im Linienstrich
// (Richtung = Differenz mal 1/Länge); ohne Schaftstrecke die links der Richtung des Kopfs.
func schaftNormale(schaft []Punkt, k Kopfform) Punkt {
	var a, b Punkt
	gefunden := false
	if k.AmAnfang {
		for i := 1; i < len(schaft); i++ { // erste Strecke mit Länge, wie der Linienstrich doppelte Punkte überspringt
			if d := schaft[i].Minus(schaft[0]); d.Laenge() > 1e-9 {
				a, b, gefunden = schaft[0], schaft[i], true
				break
			}
		}
	} else {
		n := len(schaft)
		for i := n - 2; i >= 0; i-- {
			if d := schaft[n-1].Minus(schaft[i]); d.Laenge() > 1e-9 {
				a, b, gefunden = schaft[i], schaft[n-1], true
				break
			}
		}
	}
	if !gefunden {
		return links(k.Richtung)
	}
	d := b.Minus(a)
	r := d.Mal(1 / d.Laenge())
	if k.AmAnfang {
		return Pt(r.Y, -r.X) // links der Gegenrichtung
	}
	return links(r)
}

// suchePunkt sucht vom Ende B (amEnde) bzw. A her den ersten Kurvenparameter t, dessen Punkt mindestens abstand vom Ende
// entfernt liegt, auf etwa 1e-9 genau. Gibt es keinen, das andere Ende.
func (p Pfeil) suchePunkt(amEnde bool, abstand float64) float64 {
	ende, t0, schritt := p.A, 0.0, 1.0/256
	if amEnde {
		ende, t0, schritt = p.B, 1.0, -1.0/256
	}
	weit := func(t float64) bool { return p.Punkt(t).Minus(ende).Laenge() >= abstand }
	nah := t0
	for i := 1; i <= 256; i++ {
		t := t0 + schritt*float64(i)
		if weit(t) {
			fern := t
			for range 60 {
				m := (nah + fern) / 2
				if weit(m) {
					fern = m
				} else {
					nah = m
				}
			}
			return fern
		}
		nah = t
	}
	return 1 - t0
}

// kopf baut die Spitze am Ende e mit dem Schaftende s (auf der Kurve) zur Strichstärke w.
func (p Pfeil) kopf(e, s Punkt, w float64) Kopfform {
	r := e.Minus(s)
	if l := r.Laenge(); l >= 1e-9 {
		r = r.Mal(1 / l)
	} else if r = e.Minus(p.andereEnde(e)); r.Laenge() >= 1e-9 {
		r = r.Mal(1 / r.Laenge())
	} else {
		r = Pt(1, 0)
	}
	n := links(r)
	hw := w / 2
	k := Kopfform{Spitze: e, Richtung: r, Schaftende: s}
	switch p.Kopf {
	case KopfWinkel:
		l, h := 3*w+5, (4.5*w+6)/2
		d := w / (h / math.Hypot(l, h)) // Versatz des inneren Dreiecks: Schenkel senkrecht w dick
		fuss := e.Minus(r.Mal(l))
		re, li := fuss.Minus(n.Mal(h)), fuss.Plus(n.Mal(h))
		k.Kontur = []Punkt{re.Minus(r.Mal(d)), re, e, li, li.Minus(r.Mal(d))}
	case KopfPunkt:
		rad := 1.5*w + 3
		// Bogen um e von rechts über vorn nach links; Anfang und Ende sind genau die Schaftpunkte s ∓ n·hw.
		von, bis := s.Minus(n.Mal(hw)).Minus(e), s.Plus(n.Mal(hw)).Minus(e)
		a0, a1 := math.Atan2(von.Y, von.X), math.Atan2(bis.Y, bis.X)
		// Drehsinn: über vorn, also über r; der Bogen ist der längere (mehr als π, solange hw < rad).
		bw := a1 - a0
		for bw <= 0 {
			bw += 2 * math.Pi
		}
		vorn := math.Atan2(r.Y, r.X) - a0
		for vorn < 0 {
			vorn += 2 * math.Pi
		}
		if vorn > bw { // vorn liegt auf der anderen Seite: andersherum
			bw -= 2 * math.Pi
		}
		schritte := max(8, min(1024, int(math.Ceil(math.Abs(bw)/(2*math.Acos(max(-1, 1-0.02/rad)))))))
		for j := 1; j < schritte; j++ {
			a := a0 + bw*float64(j)/float64(schritte)
			k.Kontur = append(k.Kontur, Pt(e.X+rad*math.Cos(a), e.Y+rad*math.Sin(a)))
		}
	case KopfQuer:
		q := 2.5*w + 3
		vorn := e.Plus(r.Mal(hw))
		k.Kontur = []Punkt{s.Minus(n.Mal(q)), vorn.Minus(n.Mal(q)), vorn.Plus(n.Mal(q)), s.Plus(n.Mal(q))}
	default:
		l, h := 3*w+5, (4.5*w+6)/2
		fuss := e.Minus(r.Mal(l))
		k.Kontur = []Punkt{fuss.Minus(n.Mal(h)), e, fuss.Plus(n.Mal(h))}
	}
	return k
}

// andereEnde ist A zu B und umgekehrt.
func (p Pfeil) andereEnde(e Punkt) Punkt {
	if e == p.B {
		return p.A
	}
	return p.B
}

// Huelle umfasst Schaft (mit halber Strichstärke) und Köpfe.
func (f PfeilForm) Huelle(w float64) Rechteck {
	hw := w / 2
	h := Rechteck{Min: Pt(math.Inf(1), math.Inf(1)), Max: Pt(math.Inf(-1), math.Inf(-1))}
	dazu := func(q Punkt, rand float64) {
		h.Min = Pt(math.Min(h.Min.X, q.X-rand), math.Min(h.Min.Y, q.Y-rand))
		h.Max = Pt(math.Max(h.Max.X, q.X+rand), math.Max(h.Max.Y, q.Y+rand))
	}
	for _, q := range f.Schaft {
		dazu(q, hw)
	}
	for _, k := range f.Koepfe {
		for _, q := range k.Umriss(hw) {
			dazu(q, 0)
		}
	}
	return h
}

var kopfNamen = [...]string{"triangle", "open", "dot", "bar"}
var endenNamen = [...]string{"end", "start", "both", "none"}

// Name ist der Name der Spitzenart in Datei, --annotate und MCP (triangle, open, dot, bar).
func (k Kopfart) Name() string {
	if int(k) < len(kopfNamen) {
		return kopfNamen[k]
	}
	return kopfNamen[0]
}

// KopfartAus liest einen Namen; unbekannt: Dreieck, false.
func KopfartAus(s string) (Kopfart, bool) {
	for i, n := range kopfNamen {
		if n == s {
			return Kopfart(i), true
		}
	}
	return KopfDreieck, false
}

// Name ist der Name der Enden in Datei, --annotate und MCP (end, start, both, none).
func (e PfeilEnden) Name() string {
	if int(e) < len(endenNamen) {
		return endenNamen[e]
	}
	return endenNamen[0]
}

// PfeilEndenAus liest einen Namen; unbekannt: Ende, false.
func PfeilEndenAus(s string) (PfeilEnden, bool) {
	for i, n := range endenNamen {
		if n == s {
			return PfeilEnden(i), true
		}
	}
	return EndenEnde, false
}

// Teilstueck liefert Anfang, Kontrollpunkt und Ende der Kurve zwischen den Parametern ta und tb: wieder eine
// quadratische Bézier-Kurve (Polarform). Der SVG-Export schreibt damit den gekürzten Schaft.
func (p Pfeil) Teilstueck(ta, tb float64) (a, k, b Punkt) {
	k = p.A.Mal((1 - ta) * (1 - tb)).Plus(p.K.Mal((1-ta)*tb + ta*(1-tb))).Plus(p.B.Mal(ta * tb))
	return p.Punkt(ta), k, p.Punkt(tb)
}
