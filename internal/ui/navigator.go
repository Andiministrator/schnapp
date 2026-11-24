// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Navigator (ADR-0031 Nachtrag 9, Karte #197): Mini-Übersicht des ganzen Bilds unten rechts auf der Leinwand, solange das
// Bild bei dieser Zoomstufe nicht ins Fenster passt. Ein Rechteck zeigt den sichtbaren Ausschnitt; Klick und Ziehen
// verschieben ihn, das Mausrad zoomt. Die Rechnung (Sichtbarkeit, Abbildung, Zentrieren) liegt in canvas/navigator.go.

import (
	"image"
	"math"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// navigatorZustand ist, was der Editor vom Navigator merkt.
type navigatorZustand struct {
	zieht   bool            // die Taste ist im Kasten gedrückt; Ziehen führt den Ausschnitt
	rahmen  image.Rectangle // Lage des Kastens beim letzten Zeichnen in Leinwandpixeln (leer: nicht gezeigt; Tests)
	ausschn image.Rectangle // Lage des Ausschnitt-Rechtecks beim letzten Zeichnen (Tests)
}

// zeichneNavigator wertet die Zeigerereignisse des Kastens aus und zeichnet ihn über der Leinwand. Er meldet sich mit
// eigenem Zeigerziel an, das über dem der Leinwand liegt: Druck, Ziehen und Rad erreichen die Werkzeuge nicht.
func (e *editor) zeichneNavigator(gtx layout.Context) {
	seite, rand := gtx.Dp(canvas.NavigatorSeite), gtx.Dp(canvas.NavigatorRand)
	n, ok := e.s.Navigator(seite, rand)
	if ok {
		e.navigatorEreignisse(gtx, n)
		n, ok = e.s.Navigator(seite, rand) // das Rad kann den Zoom bis zum Einpassen geändert haben
	}
	if !ok {
		e.nav = navigatorZustand{}
		return
	}
	r := n.Flaeche
	e.nav.rahmen = r
	// Anmeldung, Zeiger und Inhalt im Ausschnitt des Kastens.
	c := clip.Rect(r).Push(gtx.Ops)
	event.Op(gtx.Ops, &e.nav)
	pointer.CursorPointer.Add(gtx.Ops)
	paint.Fill(gtx.Ops, farbeLeisteVoll)
	e.zeichneNavigatorBild(gtx, n)
	// 1-dp-Rand innen, dann der Ausschnitt (2 dp, orange) darüber; beides auf den Kasten begrenzt.
	rand1 := max(1, gtx.Dp(1))
	for _, s := range randStreifen(r, rand1) {
		paint.FillShape(gtx.Ops, farbeLinie, clip.Rect(s).Op())
	}
	e.nav.ausschn = navigatorAusschnitt(n, e.s.Abb, e.s.Fenster())
	for _, s := range randStreifen(e.nav.ausschn, max(1, gtx.Dp(2))) {
		paint.FillShape(gtx.Ops, farbeAkzent, clip.Rect(s).Op())
	}
	c.Pop()
}

// navigatorAusschnitt ist das Ausschnitt-Rechteck in ganzen Leinwandpixeln, mindestens ein Pixel groß.
func navigatorAusschnitt(n canvas.Navigator, abb canvas.Abbildung, fenster image.Point) image.Rectangle {
	a := n.Ausschnitt(abb, fenster)
	r := image.Rect(int(math.Round(a.Min.X)), int(math.Round(a.Min.Y)), int(math.Round(a.Max.X)), int(math.Round(a.Max.Y)))
	if r.Dx() < 1 {
		r.Max.X = r.Min.X + 1
	}
	if r.Dy() < 1 {
		r.Max.Y = r.Min.Y + 1
	}
	// Flache oder schmale Bilder (F-613, KS-7): Deckt der Ausschnitt das ganze Bild in einer Achse, in der das Bild
	// dünner ist als der Kasten, füllt er den Kasten in dieser Achse, sonst bliebe er ein Strich.
	if n.Inhalt.Dx() < n.Flaeche.Dx() && r.Min.X <= n.Inhalt.Min.X && r.Max.X >= n.Inhalt.Max.X {
		r.Min.X, r.Max.X = n.Flaeche.Min.X, n.Flaeche.Max.X
	}
	if n.Inhalt.Dy() < n.Flaeche.Dy() && r.Min.Y <= n.Inhalt.Min.Y && r.Max.Y >= n.Inhalt.Max.Y {
		r.Min.Y, r.Max.Y = n.Flaeche.Min.Y, n.Flaeche.Max.Y
	}
	return r
}

// randStreifen sind die vier Streifen eines Rahmens der Breite w innen an r (kleiner als 2w: die ganze Fläche).
func randStreifen(r image.Rectangle, w int) []image.Rectangle {
	if r.Dx() <= 2*w || r.Dy() <= 2*w {
		return []image.Rectangle{r}
	}
	return []image.Rectangle{
		{Min: r.Min, Max: image.Pt(r.Max.X, r.Min.Y+w)},
		{Min: image.Pt(r.Min.X, r.Max.Y-w), Max: r.Max},
		{Min: image.Pt(r.Min.X, r.Min.Y+w), Max: image.Pt(r.Min.X+w, r.Max.Y-w)},
		{Min: image.Pt(r.Max.X-w, r.Min.Y+w), Max: image.Pt(r.Max.X, r.Max.Y-w)},
	}
}

// zeichneNavigatorBild zeichnet das Bild der Leinwand mit Objekten und Nähten unter einer Verkleinerung in den Kasten:
// dieselben Zeichenaufrufe wie leinwand, keine zweite Renderkette.
func (e *editor) zeichneNavigatorBild(gtx layout.Context, n canvas.Navigator) {
	f := float32(n.Faktor)
	af := op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(f, f)).
		Offset(f32.Pt(float32(n.Inhalt.Min.X)-float32(n.Bild.Min.X)*f, float32(n.Inhalt.Min.Y)-float32(n.Bild.Min.Y)*f))).Push(gtx.Ops)
	bo := op.Offset(e.bildOrt).Push(gtx.Ops)
	bi := clip.Rect(image.Rectangle{Max: e.bildOp.Size()}).Push(gtx.Ops)
	e.bildOp.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	bi.Pop()
	bo.Pop()
	bc := clip.Rect(image.Rectangle{Max: e.s.D.Groesse()}).Push(gtx.Ops)
	e.zeichneObjekte(gtx)
	e.zeichneNaehte(gtx)
	bc.Pop()
	af.Pop()
}

// navigatorEreignisse wertet Druck, Ziehen, Loslassen und Rad im Kasten aus. Positionen sind Leinwandpixel (die Anmeldung
// steht im Ursprung der Leinwand). Während des Speicherns ruht das Bearbeiten wie auf der Leinwand (ADR-0020 Nachtrag 6).
func (e *editor) navigatorEreignisse(gtx layout.Context, n canvas.Navigator) {
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target:  &e.nav,
			Kinds:   pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel | pointer.Scroll,
			ScrollY: pointer.ScrollRange{Min: -1e6, Max: 1e6},
		})
		if !ok {
			return
		}
		pe, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		gtx.Execute(op.InvalidateCmd{})
		// Loslassen und Abbruch gelten immer, auch während des Speicherns: Sie ändern nichts am Dokument, und ein
		// verlorenes Loslassen ließe `zieht` stehen (F-613, KS-3).
		if pe.Kind == pointer.Release || pe.Kind == pointer.Cancel {
			e.nav.zieht = false
			continue
		}
		if e.s.SpeichernLaeuft() {
			continue
		}
		p := document.Pt(float64(pe.Position.X), float64(pe.Position.Y))
		switch pe.Kind {
		case pointer.Press:
			// Nur die linke Taste führt den Ausschnitt; jeder andere Druck beendet ein stehengebliebenes Ziehen.
			e.nav.zieht = pe.Buttons.Contain(pointer.ButtonPrimary)
			if e.nav.zieht {
				e.s.NavigatorZu(n, p)
				gtx.Execute(key.FocusCmd{Tag: e}) // wie ein Klick auf die Leinwand: Tasten gehören ihr
			}
		case pointer.Drag:
			if e.nav.zieht {
				e.s.NavigatorZu(n, p)
			}
		case pointer.Scroll:
			e.s.NavigatorZoom(radSchritt(pe.Scroll.Y))
		}
	}
}
