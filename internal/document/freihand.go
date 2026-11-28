// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"errors"
	"fmt"
	"math"
	"slices"
)

// MaxFreihandPunkte begrenzt die Punkte eines Freihand-Strichs (ADR-0049 Punkt 1).
const MaxFreihandPunkte = 4096

// MaxFreihandGesamt begrenzt die Punkte aller Freihand-Striche eines Dokuments (F-1049): Je Strich 4096 bei bis zu
// MaxObjekte Strichen ergäben über 40 Mio. Punkte (16 Byte je Punkt); eine stark komprimierbare Datei aus „0,0 “
// forderte beim Lesen über 800 MiB an. Echte Striche sind vereinfacht (einige Hundert Punkte).
const MaxFreihandGesamt = 1 << 20

// ErrFreihandVoll meldet, dass ein Objekt die Gesamtgrenze der Freihand-Punkte überschritte (F-1051).
var ErrFreihandVoll = errors.New("document: Grenze der Freihand-Punkte erreicht")

// FreihandPlatz liefert, wie viele Freihand-Punkte das Dokument noch aufnimmt (MaxFreihandGesamt, F-1051).
func (d *Dokument) FreihandPlatz() int { return MaxFreihandGesamt - FreihandPunkteGesamt(d.objekte) }

// pruefeFreihandDazu lehnt neue Objekte ab, deren Freihand-Punkte nicht mehr ins Dokument passen (F-1051): Doppeln,
// Einfügen und --annotate scheitern so beim Hinzufügen, nicht erst beim Speichern oder Lesen der Rettungskopie.
// Gezählt wird nur, wenn Freihand dazukommt, dann über alle Objekte: Wer 10 000 Freihand-Striche einzeln hinzufügt
// (Laden an der Objektgrenze), zahlt quadratisch, gemessen 0,22 s; echte Bilder tragen einige Hundert Striche.
func (d *Dokument) pruefeFreihandDazu(neu ...Objekt) error {
	if n := FreihandPunkteGesamt(neu); n > 0 && n > d.FreihandPlatz() {
		return fmt.Errorf("%w: %d Punkte mehr, Platz für %d", ErrFreihandVoll, n, max(d.FreihandPlatz(), 0))
	}
	return nil
}

// pruefeFreihandStatt lehnt ab, wenn neu an der Stelle von alt mehr Freihand-Punkte trüge, als Platz ist (Kritik
// F-1055: Ersetzen und Zug); weniger oder gleich viele gehen immer. Gezählt über alle Objekte wird nur bei einem Zuwachs.
func (d *Dokument) pruefeFreihandStatt(alt, neu []Objekt) error {
	if dazu := FreihandPunkteGesamt(neu) - FreihandPunkteGesamt(alt); dazu > 0 && dazu > d.FreihandPlatz() {
		return fmt.Errorf("%w: %d Punkte mehr, Platz für %d", ErrFreihandVoll, dazu, max(d.FreihandPlatz(), 0))
	}
	return nil
}

// FreihandPunkteGesamt zählt die Punkte aller Freihand-Striche in objekte.
func FreihandPunkteGesamt(objekte []Objekt) int {
	n := 0
	for _, o := range objekte {
		if f, ok := o.(Freihand); ok {
			n += len(f.Punkte())
		}
	}
	return n
}

// Freihand ist ein freier Strich: ein offener Linienzug im Original (ADR-0049). Zug zeigt auf die Punkte; wie die Spitze
// der Sprechblase wird er nie durchgeschrieben, sondern neu gesetzt (MitPunkten), damit Objekte Werte bleiben.
type Freihand struct {
	Ebene
	ID     int
	Zug    *[]Punkt
	Strich Strich
}

// NeuerFreihand liefert einen Freihand-Strich mit einer eigenen Kopie der Punkte.
func NeuerFreihand(pts []Punkt, s Strich) Freihand {
	return Freihand{Strich: s}.MitPunkten(pts)
}

// Punkte liefert die Punkte (nicht verändern; nil bei leerem Zug).
func (f Freihand) Punkte() []Punkt {
	if f.Zug == nil {
		return nil
	}
	return *f.Zug
}

// MitPunkten liefert f mit einer eigenen Kopie von pts, jede Lage auf 1/100 px gerundet (F-1045): Freihand-Punkte liegen
// immer auf ganzen Hundertsteln, damit die Datei sie als Abstände in Hundertsteln verlustfrei trägt (docs/format.md
// 8.9). Gleich gewordene Nachbarn bleiben stehen; wer sie loswerden will, nimmt Gerundet.
func (f Freihand) MitPunkten(pts []Punkt) Freihand {
	k := make([]Punkt, len(pts))
	for i, p := range pts {
		k[i] = aufHundertstel(p)
	}
	f.Zug = &k
	return f
}

// Verschoben liefert f um d verschoben (neuer Zug). Gerundet wird nur d auf 1/100 px (F-1036: ein Zoom wie 0,73
// macht sonst aus jeder Lage eine Zahl mit 17 Stellen); die Punkte liegen auf Hundertsteln (MitPunkten), jede neue Lage
// wird wieder auf Hundertstel gerundet, damit Speichern und Lesen sie gleich zurückgeben (F-1053); Verschieben um 0 ändert
// nichts (Verify #333).
func (f Freihand) Verschoben(d Punkt) Freihand {
	d = aufHundertstel(d)
	if d == (Punkt{}) {
		return f
	}
	pts := slices.Clone(f.Punkte())
	for i := range pts {
		// Auf Hundertsteln plus Hundertstel bleibt auf Hundertsteln, nur das Gleitkomma nicht (0,1 + 0,2): jede Lage
		// runden, die Bewegung bleibt starr (Kritik F-1053).
		pts[i] = aufHundertstel(pts[i].Plus(d))
	}
	f.Zug = &pts
	return f
}

func (f Freihand) Kennung() int { return f.ID }

// Grenzen ist die Hülle der Punkte um die halbe Strichbreite.
func (f Freihand) Grenzen() Rechteck {
	pts := f.Punkte()
	if len(pts) == 0 {
		return Rechteck{}
	}
	r := Rechteck{Min: pts[0], Max: pts[0]}
	for _, p := range pts[1:] {
		r = r.Vereint(Rechteck{Min: p, Max: p})
	}
	return aufgeweitet(r, f.Strich.Breite)
}

func (f Freihand) mitKennung(id int) Objekt { f.ID = id; return f }
func (f Freihand) mitEbene(e Ebene) Objekt  { f.Ebene = e; return f }

func (f Freihand) pruefe() error {
	pts := f.Punkte()
	if len(pts) < 2 || len(pts) > MaxFreihandPunkte {
		return fmt.Errorf("document: Freihand-Strich mit %d Punkten (2 bis %d)", len(pts), MaxFreihandPunkte)
	}
	for _, p := range pts {
		if !endlich(p.X, p.Y) {
			return fmt.Errorf("document: Freihand-Strich mit ungültigen Koordinaten")
		}
	}
	// Lauter gleiche Punkte sind ein Strich ohne Länge: gestrichelt unsichtbar, aber treffbar (F-1040).
	if f.Laenge() <= 0 {
		return fmt.Errorf("document: Freihand-Strich ohne Länge")
	}
	if err := f.Ebene.pruefe(); err != nil {
		return err
	}
	return f.Strich.pruefe()
}

// gleicheFreihand vergleicht zwei Freihand-Striche dem Wert nach (Punkte statt Adresse, wie die Spitze, F-694).
func gleicheFreihand(a, b Freihand) bool {
	if !slices.Equal(a.Punkte(), b.Punkte()) {
		return false
	}
	a.Zug, b.Zug = nil, nil
	return a == b
}

// Laenge ist die Länge des Linienzugs.
func (f Freihand) Laenge() float64 {
	pts := f.Punkte()
	var l float64
	for i := 1; i < len(pts); i++ {
		l += pts[i].Minus(pts[i-1]).Laenge()
	}
	return l
}

// --- Glätten (ADR-0049 Punkt 3) ---

// Vereinfacht liefert pts nach Ramer–Douglas–Peucker mit der Toleranz tol (Anfang und Ende bleiben).
func Vereinfacht(pts []Punkt, tol float64) []Punkt {
	if len(pts) < 3 {
		return slices.Clone(pts)
	}
	behalten := make([]bool, len(pts))
	behalten[0], behalten[len(pts)-1] = true, true
	type spanne struct{ a, b int }
	stapel := []spanne{{0, len(pts) - 1}}
	for len(stapel) > 0 {
		s := stapel[len(stapel)-1]
		stapel = stapel[:len(stapel)-1]
		weit, wo := 0.0, -1
		for i := s.a + 1; i < s.b; i++ {
			if d := abstandStrecke(pts[i], pts[s.a], pts[s.b]); d > weit {
				weit, wo = d, i
			}
		}
		if wo >= 0 && weit > tol {
			behalten[wo] = true
			stapel = append(stapel, spanne{s.a, wo}, spanne{wo, s.b})
		}
	}
	var r []Punkt
	for i, b := range behalten {
		if b {
			r = append(r, pts[i])
		}
	}
	return r
}

// abstandStrecke ist der Abstand von p zur Strecke a–b.
func abstandStrecke(p, a, b Punkt) float64 {
	d := b.Minus(a)
	l2 := d.X*d.X + d.Y*d.Y
	if l2 == 0 {
		return p.Minus(a).Laenge()
	}
	t := math.Max(0, math.Min(1, ((p.X-a.X)*d.X+(p.Y-a.Y)*d.Y)/l2))
	return p.Minus(a.Plus(d.Mal(t))).Laenge()
}

// Geglaettet liefert pts nach n Runden Chaikin: Jede Strecke wird durch die Punkte bei 1/4 und 3/4 ersetzt, Anfang und
// Ende bleiben. Ecken werden rund, gerade Stücke bleiben gerade.
func Geglaettet(pts []Punkt, n int) []Punkt {
	r := slices.Clone(pts)
	for ; n > 0 && len(r) >= 3; n-- {
		neu := make([]Punkt, 0, 2*len(r))
		neu = append(neu, r[0])
		for i := 0; i+1 < len(r); i++ {
			a, b := r[i], r[i+1]
			q := a.Mal(0.75).Plus(b.Mal(0.25))
			s := a.Mal(0.25).Plus(b.Mal(0.75))
			if i > 0 {
				neu = append(neu, q)
			}
			if i+2 < len(r) {
				neu = append(neu, s)
			}
		}
		neu = append(neu, r[len(r)-1])
		r = neu
	}
	return r
}

// Ausgeduennt liefert höchstens max Punkte: gleichmäßig ausgewählt, Anfang und Ende bleiben.
func Ausgeduennt(pts []Punkt, max int) []Punkt {
	if len(pts) <= max || max < 2 {
		return slices.Clone(pts)
	}
	r := make([]Punkt, max)
	for i := range r {
		r[i] = pts[i*(len(pts)-1)/(max-1)]
	}
	return r
}

// FreihandAusZug macht aus den rohen Punkten eines Zugs den gespeicherten Linienzug (ADR-0049 Punkt 3, Nachtrag 1):
// vereinfachen mit der Toleranz tol (Original), zweimal glätten, danach noch einmal mit tol/2 vereinfachen (Chaikin
// vervierfacht die Punkte, auf flachen Stücken liegen sie fast auf einer Geraden, F-1036) und auf 1/100 px runden.
// Gespeichert werden höchstens so viele Punkte wie roh, sonst bleibt der ungeglättete Linienzug; MaxFreihandPunkte
// begrenzt zuletzt.
func FreihandAusZug(roh []Punkt, tol float64) []Punkt {
	v := Vereinfacht(roh, tol)
	g := Gerundet(Vereinfacht(Geglaettet(v, 2), tol/2))
	if len(g) > len(roh) {
		g = Gerundet(v)
	}
	return Ausgeduennt(g, MaxFreihandPunkte)
}

// Gerundet liefert pts auf 1/100 px gerundet; dabei gleich gewordene Nachbarn fallen weg, mindestens zwei Punkte
// bleiben (F-1036: kurze Zahlen in der Datei).
func Gerundet(pts []Punkt) []Punkt {
	r := make([]Punkt, 0, len(pts))
	for _, p := range pts {
		q := aufHundertstel(p)
		if len(r) > 0 && r[len(r)-1] == q {
			continue
		}
		r = append(r, q)
	}
	if len(r) == 1 && len(pts) >= 2 {
		r = append(r, r[0])
	}
	return r
}

// aufHundertstel rundet p auf 1/100 px.
func aufHundertstel(p Punkt) Punkt { return Pt(math.Round(p.X*100)/100, math.Round(p.Y*100)/100) }
