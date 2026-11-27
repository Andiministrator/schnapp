// SPDX-License-Identifier: GPL-3.0-or-later

package libview

import (
	"image"
	"sort"
	"time"

	"github.com/Andiministrator/schnapp/internal/library"
)

// Raster ist der Zustand der vollen Ansicht der Bibliothek (ADR-0036 Punkte 3 und 4): die gefundenen Aufnahmen nach
// Monaten gruppiert (neueste zuerst), die offene Aufnahme, der Tastaturfokus und der senkrechte Versatz. Ohne Gio; die
// Oberfläche (internal/ui/bibliothek.go) fragt Lage, Treffer und den sichtbaren Bereich ab.
//
// Schritte des Fokus über Monatsgrenzen hinweg laufen über die Zeilen des ganzen Rasters: Pfeil hoch aus der ersten
// Zeile eines Monats trifft die letzte Zeile des Monats davor in derselben Spalte (oder deren letzte Kachel).

// RasterMasse sind die Maße des Rasters in Pixeln, die die Oberfläche aus dp rechnet.
type RasterMasse struct {
	Rand    int // links und rechts der Kacheln
	Oben    int // über der ersten Monatsüberschrift
	KachelB int // Breite einer Kachel
	KachelH int // Höhe einer Kachel samt Text darunter
	LueckeX int // waagerecht zwischen zwei Kacheln
	LueckeY int // senkrecht zwischen zwei Zeilen
	Kopf    int // Höhe einer Monatsüberschrift samt Abstand zu den Kacheln
	Breite  int // sichtbare Breite
	Hoehe   int // sichtbare Höhe
}

// Gruppe ist ein Monat im Raster: Aufnahmen von Von bis Bis (ausschließlich) in Eintraege.
type Gruppe struct {
	Jahr     int
	Monat    time.Month
	Von, Bis int
}

// Anzahl ist die Zahl der Aufnahmen des Monats (Überschrift).
func (g Gruppe) Anzahl() int { return g.Bis - g.Von }

// rasterZeile ist eine Zeile von Kacheln: Stellen von bis bis (ausschließlich), Oberkante y im Inhalt, Monat gruppe.
type rasterZeile struct{ von, bis, y, gruppe int }

// Raster hält Aufnahmen, Fokus und Rollen der vollen Ansicht.
type Raster struct {
	eintraege []library.Eintrag
	gruppen   []Gruppe
	zone      *time.Location
	aktuell   string  // Pfad der offenen Aufnahme relativ zur Ablage; leer: keine aus der Bibliothek
	fokus     int     // Stelle mit Tastaturfokus; -1: keine
	rollen    int     // Versatz in Pixeln, 0 = ganz oben
	rest      float32 // Radweg unter einem Pixel (Touchpad)
	mark      Markierung

	// Anordnung für masse (gueltig), neu gerechnet nach Setze oder anderen Maßen.
	masse   RasterMasse
	gueltig bool
	zeilen  []rasterZeile
	zeileZu []int // Zeile je Stelle
	koepfe  []int // Oberkante der Überschrift je Monat
	inhalt  int   // Höhe des ganzen Inhalts
}

// NeuesRaster liefert ein leeres Raster ohne Fokus; Monate gelten in der Zone zone (nil: lokale Zeit).
func NeuesRaster(zone *time.Location) *Raster {
	if zone == nil {
		zone = time.Local
	}
	return &Raster{zone: zone, fokus: -1}
}

// Setze übernimmt die gefundenen Aufnahmen (neueste zuerst, wie library.Suchen sie liefert) und gruppiert sie nach
// Monaten. Der Fokus bleibt bei derselben Aufnahme; gibt es sie nicht mehr, bleibt er an derselben Stelle (die nächste
// rückt nach) bzw. an der letzten.
func (r *Raster) Setze(es []library.Eintrag) {
	var alt string
	if r.fokus >= 0 && r.fokus < len(r.eintraege) {
		alt = r.eintraege[r.fokus].Pfad
	}
	r.eintraege = append([]library.Eintrag(nil), es...)
	r.gruppen = r.gruppen[:0]
	for i, e := range r.eintraege {
		j, m, _ := e.Erstellt.In(r.zone).Date()
		if n := len(r.gruppen); n == 0 || r.gruppen[n-1].Jahr != j || r.gruppen[n-1].Monat != m {
			r.gruppen = append(r.gruppen, Gruppe{Jahr: j, Monat: m, Von: i})
		}
		r.gruppen[len(r.gruppen)-1].Bis = i + 1
	}
	r.gueltig = false
	r.mark.Bereinige(r.eintraege)
	if r.fokus < 0 {
		return
	}
	if i := r.Stelle(alt); i >= 0 {
		r.fokus = i
		return
	}
	r.fokus = min(r.fokus, len(r.eintraege)-1)
}

// Markierung sind die markierten Vorschauen der Ansicht (ADR-0036 Zusatz Karte #361).
func (r *Raster) Markierung() *Markierung { return &r.mark }

// Eintraege sind die gezeigten Aufnahmen, neueste zuerst.
func (r *Raster) Eintraege() []library.Eintrag { return r.eintraege }

// Gruppen sind die Monate in Reihenfolge des Rasters.
func (r *Raster) Gruppen() []Gruppe { return r.gruppen }

// Stelle liefert die Stelle der Aufnahme rel, -1 ohne.
func (r *Raster) Stelle(rel string) int {
	if rel == "" {
		return -1
	}
	for i, e := range r.eintraege {
		if e.Pfad == rel {
			return i
		}
	}
	return -1
}

// SetzeAktuell merkt die offene Aufnahme (Pfad relativ zur Ablage; leer: keine aus der Bibliothek).
func (r *Raster) SetzeAktuell(rel string) { r.aktuell = rel }

// Aktuell ist der Pfad der offenen Aufnahme relativ zur Ablage.
func (r *Raster) Aktuell() string { return r.aktuell }

// IstAktuell meldet, ob die Stelle i die offene Aufnahme ist (grün umrandet; nicht in den Papierkorb).
func (r *Raster) IstAktuell(i int) bool {
	return r.aktuell != "" && i >= 0 && i < len(r.eintraege) && r.eintraege[i].Pfad == r.aktuell
}

// Fokus ist die Stelle mit Tastaturfokus; -1: keine.
func (r *Raster) Fokus() int { return r.fokus }

// SetzeFokus setzt den Fokus auf die Stelle i; außerhalb der Liste: kein Fokus.
func (r *Raster) SetzeFokus(i int) {
	if i < 0 || i >= len(r.eintraege) {
		i = -1
	}
	r.fokus = i
}

// Spalten ist die Zahl der Kacheln je Zeile für die Maße m, mindestens eine.
func Spalten(m RasterMasse) int {
	schritt := m.KachelB + m.LueckeX
	if schritt <= 0 {
		return 1
	}
	return max(1, (m.Breite-2*m.Rand+m.LueckeX)/schritt)
}

// anordnen rechnet Zeilen und Überschriften für die Maße m (nur nach einer Änderung).
func (r *Raster) anordnen(m RasterMasse) {
	if r.gueltig && r.masse == m {
		return
	}
	r.masse, r.gueltig = m, true
	sp := Spalten(m)
	r.zeilen, r.koepfe = r.zeilen[:0], r.koepfe[:0]
	r.zeileZu = make([]int, len(r.eintraege))
	y := m.Oben
	for g, gr := range r.gruppen {
		r.koepfe = append(r.koepfe, y)
		y += m.Kopf
		for von := gr.Von; von < gr.Bis; von += sp {
			bis := min(von+sp, gr.Bis)
			for i := von; i < bis; i++ {
				r.zeileZu[i] = len(r.zeilen)
			}
			r.zeilen = append(r.zeilen, rasterZeile{von: von, bis: bis, y: y, gruppe: g})
			y += m.KachelH + m.LueckeY
		}
	}
	r.inhalt = y
	if len(r.gruppen) == 0 {
		r.inhalt = 0
	}
}

// Inhalt ist die Höhe des ganzen Rasters in Pixeln.
func (r *Raster) Inhalt(m RasterMasse) int {
	r.anordnen(m)
	return r.inhalt
}

// RollMax ist der größte Versatz: wie weit der Inhalt über die sichtbare Höhe hinausragt.
func (r *Raster) RollMax(m RasterMasse) int { return max(0, r.Inhalt(m)-m.Hoehe) }

// Rollen ist der Versatz, begrenzt auf 0 … RollMax (kleineres Fenster, kürzere Liste).
func (r *Raster) Rollen(m RasterMasse) int {
	r.rollen = max(0, min(r.rollen, r.RollMax(m)))
	return r.rollen
}

// Lage liefert das Rechteck der Kachel i in Pixeln ab der linken oberen Ecke des sichtbaren Bereichs (nach dem Rollen).
func (r *Raster) Lage(i int, m RasterMasse) image.Rectangle {
	r.anordnen(m)
	if i < 0 || i >= len(r.eintraege) {
		return image.Rectangle{}
	}
	z := r.zeilen[r.zeileZu[i]]
	x := m.Rand + (i-z.von)*(m.KachelB+m.LueckeX)
	y := z.y - r.Rollen(m)
	return image.Rect(x, y, x+m.KachelB, y+m.KachelH)
}

// KopfLage liefert die Oberkante der Überschrift des Monats g im sichtbaren Bereich (nach dem Rollen).
func (r *Raster) KopfLage(g int, m RasterMasse) int {
	r.anordnen(m)
	if g < 0 || g >= len(r.koepfe) {
		return 0
	}
	return r.koepfe[g] - r.Rollen(m)
}

// Sichtbar liefert die Stellen von bis bis (ausschließlich), deren Zeile ganz oder teilweise sichtbar ist, dazu eine
// Zeile Vorlauf über und unter dem sichtbaren Bereich (ADR-0036 Punkt 6: nur sie werden geladen).
func (r *Raster) Sichtbar(m RasterMasse) (von, bis int) {
	r.anordnen(m)
	if len(r.zeilen) == 0 {
		return 0, 0
	}
	ro := r.Rollen(m)
	schritt := m.KachelH + m.LueckeY
	oben, unten := ro-schritt, ro+m.Hoehe+schritt
	a := sort.Search(len(r.zeilen), func(k int) bool { return r.zeilen[k].y+m.KachelH > oben })
	b := sort.Search(len(r.zeilen), func(k int) bool { return r.zeilen[k].y >= unten })
	if a >= b {
		return 0, 0
	}
	return r.zeilen[a].von, r.zeilen[b-1].bis
}

// SichtbareGruppen liefert die Monate von bis bis (ausschließlich), deren Überschrift ganz oder teilweise sichtbar ist.
func (r *Raster) SichtbareGruppen(m RasterMasse) (von, bis int) {
	r.anordnen(m)
	ro := r.Rollen(m)
	von, bis = len(r.koepfe), 0
	for g, y := range r.koepfe {
		if y+m.Kopf > ro && y < ro+m.Hoehe {
			von, bis = min(von, g), max(bis, g+1)
		}
	}
	if bis == 0 {
		return 0, 0
	}
	return von, bis
}

// Treffer liefert die Stelle der Kachel unter dem Punkt p (Pixel ab der linken oberen Ecke des sichtbaren Bereichs),
// -1 zwischen den Kacheln, auf einer Überschrift oder außerhalb.
func (r *Raster) Treffer(p image.Point, m RasterMasse) int {
	r.anordnen(m)
	if p.X < 0 || p.Y < 0 || p.X >= m.Breite || p.Y >= m.Hoehe {
		return -1
	}
	y := p.Y + r.Rollen(m)
	k := sort.Search(len(r.zeilen), func(k int) bool { return r.zeilen[k].y+m.KachelH > y })
	if k >= len(r.zeilen) || y < r.zeilen[k].y {
		return -1
	}
	x := p.X - m.Rand
	schritt := m.KachelB + m.LueckeX
	if x < 0 || schritt <= 0 || x%schritt >= m.KachelB {
		return -1
	}
	i := r.zeilen[k].von + x/schritt
	if i >= r.zeilen[k].bis {
		return -1
	}
	return i
}

// Bewege liefert die Stelle, auf die der Fokus von i aus mit der Taste s rückt: links/rechts in der Reihenfolge,
// hoch/runter eine Zeile (dieselbe Spalte oder die letzte Kachel einer kürzeren Zeile, über Monatsgrenzen hinweg),
// Bild hoch/runter eine sichtbare Höhe, Pos1/Ende an den Anfang bzw. das Ende. An den Rändern bleibt er stehen. Ohne
// Fokus (i < 0) trifft jede Taste die erste Aufnahme; -1 bei leerem Raster.
func (r *Raster) Bewege(i int, s Schritt, m RasterMasse) int {
	r.anordnen(m)
	n := len(r.eintraege)
	switch {
	case n == 0:
		return -1
	case i < 0 || i >= n:
		return 0
	}
	zeilen := 0
	switch s {
	case SchrittLinks:
		return max(0, i-1)
	case SchrittRechts:
		return min(n-1, i+1)
	case SchrittAnfang:
		return 0
	case SchrittEnde:
		return n - 1
	case SchrittHoch:
		zeilen = -1
	case SchrittRunter:
		zeilen = 1
	case SchrittSeiteHoch:
		zeilen = -r.seitenZeilen(m)
	case SchrittSeiteRunter:
		zeilen = r.seitenZeilen(m)
	}
	k := r.zeileZu[i]
	spalte := i - r.zeilen[k].von
	ziel := r.zeilen[max(0, min(k+zeilen, len(r.zeilen)-1))]
	return min(ziel.von+spalte, ziel.bis-1)
}

// seitenZeilen ist die Zahl der Zeilen, um die Bild hoch/runter blättert: so viele, wie ganz sichtbar sind, mindestens
// eine.
func (r *Raster) seitenZeilen(m RasterMasse) int {
	schritt := m.KachelH + m.LueckeY
	if schritt <= 0 {
		return 1
	}
	return max(1, m.Hoehe/schritt)
}

// Zeige rollt so wenig wie möglich, dass die Kachel i ganz sichtbar ist; in der ersten Zeile eines Monats samt dessen
// Überschrift (Tastaturfokus, offene Aufnahme beim Öffnen).
func (r *Raster) Zeige(i int, m RasterMasse) {
	r.anordnen(m)
	if i < 0 || i >= len(r.eintraege) {
		return
	}
	z := r.zeilen[r.zeileZu[i]]
	oben, unten := z.y, z.y+m.KachelH
	if r.zeileZu[i] == 0 || r.zeilen[r.zeileZu[i]-1].gruppe != z.gruppe {
		oben = max(0, r.koepfe[z.gruppe]-m.Oben) // samt Überschrift und dem Abstand darüber
	}
	ro := r.Rollen(m)
	switch {
	case oben < ro:
		ro = oben
	case unten > ro+m.Hoehe:
		ro = min(oben, unten+m.LueckeY-m.Hoehe) // ist die Kachel höher als die Fläche, gilt ihre Oberkante
	}
	r.rollen = max(0, min(ro, r.RollMax(m)))
}

// Rad rollt um den Radweg d (Pixel, nach unten positiv). Der Bruchteil unter einem Pixel bleibt als Rest; am Anschlag
// verfällt er (wie die rechte Leiste, F-612).
func (r *Raster) Rad(d float32, m RasterMasse) {
	rmax := r.RollMax(m)
	r.rest += d
	n := int(r.rest)
	r.rest -= float32(n)
	neu := max(0, min(r.Rollen(m)+n, rmax))
	if neu != r.rollen+n || (neu == 0 && r.rest < 0) || (neu == rmax && r.rest > 0) {
		r.rest = 0
	}
	r.rollen = neu
}

// SetzeRollen setzt den Versatz (Ziehen am Balken, Tests); begrenzt beim nächsten Gebrauch.
func (r *Raster) SetzeRollen(y int) { r.rollen = max(0, y) }
