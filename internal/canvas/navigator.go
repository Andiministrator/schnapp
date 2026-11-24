// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"math"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Navigator: Mini-Übersicht des ganzen Bilds mit dem sichtbaren Ausschnitt (ADR-0031 Nachtrag 9, Karte #197). Dieses
// Paket rechnet nur; Zeichnen und Zeigerereignisse stehen in internal/ui/navigator.go.

// Maße des Navigators in Geräteunabhängigen Punkten (dp): längste Seite und Abstand zu den Leinwandkanten.
const (
	NavigatorSeite = 160
	NavigatorRand  = 12
	// NavigatorMinSeite ist das Mindestmaß der kurzen Kastenseite in dp (F-613, KS-7): Sehr flache oder schmale Bilder
	// bekommen einen Kasten dieser Dicke, das Bild steht darin mittig.
	NavigatorMinSeite = 16
)

// navigatorToleranz ist der Überstand in Pixeln, ab dem das Bild „nicht ins Fenster passt“: Beim Einpassen runden Zoom
// und Fenster nicht auf den Pixel genau.
const navigatorToleranz = 0.5

// NavigatorSichtbar sagt, ob das Bild (Bildfläche in Ansichtspixeln) bei diesem Zoom in einer Achse über das Fenster
// hinausragt. Nur dann braucht es die Übersicht.
func NavigatorSichtbar(bild image.Rectangle, zoom float64, fenster image.Point) bool {
	if bild.Empty() || fenster.X <= 0 || fenster.Y <= 0 || !(zoom > 0) {
		return false
	}
	return float64(bild.Dx())*zoom > float64(fenster.X)+navigatorToleranz ||
		float64(bild.Dy())*zoom > float64(fenster.Y)+navigatorToleranz
}

// Navigator ist die Lage des Kastens und die Abbildung zwischen Bildfläche und Kasten.
type Navigator struct {
	Flaeche image.Rectangle // Lage des Kastens in Leinwandpixeln
	Inhalt  image.Rectangle // Lage des Bilds im Kasten (Seitenverhältnis gewahrt; bei flachen Bildern kleiner als Flaeche)
	Bild    image.Rectangle // gezeigte Bildfläche in Ansichtskoordinaten (Sitzung.Bildflaeche)
	Faktor  float64         // Kastenpixel je Ansichtspixel
}

// NeuerNavigator legt den Kasten unten rechts in ein Fenster der Größe fenster: längste Seite seite, Abstand rand,
// Seitenverhältnis der Bildfläche. ok ist falsch bei leerer Bildfläche oder wenn der Kasten nicht ins Fenster passt.
func NeuerNavigator(bild image.Rectangle, fenster image.Point, seite, rand int) (Navigator, bool) {
	if bild.Empty() || seite < 1 {
		return Navigator{}, false
	}
	f := float64(seite) / float64(max(bild.Dx(), bild.Dy()))
	b := max(1, int(math.Round(float64(bild.Dx())*f)))
	h := max(1, int(math.Round(float64(bild.Dy())*f)))
	// Kurze Seite mindestens seite/10 (16 dp bei 160 dp): das Bild bleibt in seinem Maßstab, der Kasten wird dicker.
	mind := max(1, seite*NavigatorMinSeite/NavigatorSeite)
	kb, kh := max(b, mind), max(h, mind)
	ende := image.Pt(fenster.X-rand, fenster.Y-rand)
	r := image.Rectangle{Min: ende.Sub(image.Pt(kb, kh)), Max: ende}
	if r.Min.X < rand || r.Min.Y < rand {
		return Navigator{}, false
	}
	in := image.Rectangle{Min: r.Min.Add(image.Pt((kb-b)/2, (kh-h)/2))}
	in.Max = in.Min.Add(image.Pt(b, h))
	return Navigator{Flaeche: r, Inhalt: in, Bild: bild, Faktor: f}, true
}

// ZuKasten rechnet einen Ansichtspunkt in Leinwandpixel im Kasten um.
func (n Navigator) ZuKasten(a document.Punkt) document.Punkt {
	return document.Pt(float64(n.Inhalt.Min.X)+(a.X-float64(n.Bild.Min.X))*n.Faktor,
		float64(n.Inhalt.Min.Y)+(a.Y-float64(n.Bild.Min.Y))*n.Faktor)
}

// ZuAnsicht rechnet einen Leinwandpunkt im Kasten in einen Ansichtspunkt um; Punkte außerhalb des Kastens liegen am
// nächsten Rand der Bildfläche (Ziehen darf den Kasten verlassen).
func (n Navigator) ZuAnsicht(p document.Punkt) document.Punkt {
	a := document.Pt(float64(n.Bild.Min.X)+(p.X-float64(n.Inhalt.Min.X))/n.Faktor,
		float64(n.Bild.Min.Y)+(p.Y-float64(n.Inhalt.Min.Y))/n.Faktor)
	return document.Pt(math.Max(float64(n.Bild.Min.X), math.Min(float64(n.Bild.Max.X), a.X)),
		math.Max(float64(n.Bild.Min.Y), math.Min(float64(n.Bild.Max.Y), a.Y)))
}

// Enthaelt meldet, ob der Leinwandpunkt im Kasten liegt (Rand eingeschlossen, Ereignisse am Rand gehören ihm).
func (n Navigator) Enthaelt(p document.Punkt) bool {
	return p.X >= float64(n.Flaeche.Min.X) && p.X <= float64(n.Flaeche.Max.X) &&
		p.Y >= float64(n.Flaeche.Min.Y) && p.Y <= float64(n.Flaeche.Max.Y)
}

// Ausschnitt ist der sichtbare Teil des Bilds im Kasten (Leinwandpixel): das Fenster in Ansichtskoordinaten, auf die
// Bildfläche begrenzt.
func (n Navigator) Ausschnitt(abb Abbildung, fenster image.Point) document.Rechteck {
	a := abb.ZuAnsicht(document.Pt(0, 0))
	b := abb.ZuAnsicht(document.Pt(float64(fenster.X), float64(fenster.Y)))
	klemme := func(p document.Punkt) document.Punkt {
		return document.Pt(math.Max(float64(n.Bild.Min.X), math.Min(float64(n.Bild.Max.X), p.X)),
			math.Max(float64(n.Bild.Min.Y), math.Min(float64(n.Bild.Max.Y), p.Y)))
	}
	return document.R(n.ZuKasten(klemme(a)), n.ZuKasten(klemme(b)))
}

// ZentriertAuf liefert die Abbildung, die den Ansichtspunkt mitte in die Fenstermitte legt, begrenzt auf das Bild: Ragt
// das Bild in einer Achse über das Fenster hinaus, bleibt der Ausschnitt im Bild; ist es dort kleiner, steht es mittig.
func ZentriertAuf(abb Abbildung, mitte document.Punkt, bild image.Rectangle, fenster image.Point) Abbildung {
	ein := func(m, von, bis, flaeche float64) float64 {
		halb := flaeche / abb.Zoom / 2
		if bis-von <= 2*halb {
			return (von + bis) / 2
		}
		return math.Max(von+halb, math.Min(bis-halb, m))
	}
	c := document.Pt(
		ein(mitte.X, float64(bild.Min.X), float64(bild.Max.X), float64(fenster.X)),
		ein(mitte.Y, float64(bild.Min.Y), float64(bild.Max.Y), float64(fenster.Y)))
	abb.Versatz = document.Pt(float64(fenster.X)/2, float64(fenster.Y)/2).Minus(c.Mal(abb.Zoom))
	return abb
}

// Navigator liefert den Kasten für die Leinwand dieser Sitzung, wenn er gezeigt wird: eingeschaltet (Menü Ansicht),
// das Bild ragt bei diesem Zoom über das Fenster hinaus und der Kasten passt hinein. seite und rand in Pixeln.
func (s *Sitzung) Navigator(seite, rand int) (Navigator, bool) {
	bild := s.Bildflaeche()
	if s.navigatorAus || !NavigatorSichtbar(bild, s.Abb.Zoom, s.fenster) {
		return Navigator{}, false
	}
	return NeuerNavigator(bild, s.fenster, seite, rand)
}

// Fenster ist die zuletzt gemeldete Größe der Leinwand in Pixeln (SetzeFenster).
func (s *Sitzung) Fenster() image.Point { return s.fenster }

// NavigatorAn meldet, ob der Navigator eingeschaltet ist (Menü Ansicht; Voreinstellung an, nicht gespeichert).
func (s *Sitzung) NavigatorAn() bool { return !s.navigatorAus }

// SchalteNavigator schaltet den Navigator um.
func (s *Sitzung) SchalteNavigator() { s.navigatorAus = !s.navigatorAus }

// NavigatorZu zentriert die Ansicht auf die Stelle p des Kastens (Leinwandpixel; außerhalb gilt der nächste Rand). Während
// eines Zugs und ohne Kasten geschieht nichts, wie bei Zoom und Einpassen.
func (s *Sitzung) NavigatorZu(n Navigator, p document.Punkt) {
	if s.Zieht() {
		return
	}
	s.eingepasst = false
	s.Abb = ZentriertAuf(s.Abb, n.ZuAnsicht(p), n.Bild, s.fenster)
}

// NavigatorZoom zoomt wie das Mausrad über der Leinwand, aber um die Fenstermitte: Der Zeiger steht im Kasten in der
// Ecke, ein Zoom um ihn schöbe das Bild aus dem Fenster.
func (s *Sitzung) NavigatorZoom(dy float64) {
	s.Scrollen(document.Pt(float64(s.fenster.X)/2, float64(s.fenster.Y)/2), dy)
}
