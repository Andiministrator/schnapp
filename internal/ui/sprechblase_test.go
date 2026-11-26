// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/widget/material"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/icons"
	"github.com/Andiministrator/schnapp/internal/render"
)

// fensterPt rechnet einen Punkt des Bilds (Original = Ansicht) in Fensterpixel um.
func (l *menueLauf) fensterPt(p document.Punkt) image.Point {
	f := l.e.leinwandFlaeche(l.gtx())
	q := l.e.s.Abb.ZuFenster(p)
	return f.Min.Add(image.Pt(int(q.X), int(q.Y)))
}

// Taste K wählt das Werkzeug Sprechblase; Knopf, Menü Werkzeuge, Hilfe und Symbol kennen es (ADR-0021 Nachtrag 13 Punkt 5).
func TestSprechblaseTasteKnopfMenueHilfe(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	if werkzeugKuerzel(canvas.WerkzeugSprechblase) != "K" || l.e.werkzeugName(canvas.WerkzeugSprechblase) != "Speech bubble" {
		t.Fatalf("Kürzel %q, Name %q", werkzeugKuerzel(canvas.WerkzeugSprechblase), l.e.werkzeugName(canvas.WerkzeugSprechblase))
	}
	gefunden := false
	for _, z := range l.e.hilfeZeilen() {
		gefunden = gefunden || (z.text == "Speech bubble" && z.taste == "K")
	}
	if !gefunden {
		t.Error("Sprechblase fehlt in der Übersicht der Tastenkürzel")
	}
	im := false
	for _, en := range l.e.menueEintraegeVon(menueWerkzeuge) {
		im = im || (en.hatWerkzeug && en.werkzeug == canvas.WerkzeugSprechblase)
	}
	if !im {
		t.Error("Sprechblase fehlt im Menü Werkzeuge")
	}
	// Reihenfolge der Leiste: direkt unter Text.
	ws := l.e.werkzeugKnoepfe()
	for i, w := range ws {
		if w.w == canvas.WerkzeugText && ws[i+1].w != canvas.WerkzeugSprechblase {
			t.Errorf("auf Text folgt %v, Soll die Sprechblase", ws[i+1].w)
		}
	}
	if werkzeugSymbol(canvas.WerkzeugSprechblase).Name != icons.Sprechblase.Name || icons.Sprechblase.Name == icons.Auswahl.Name {
		t.Errorf("Symbol der Sprechblase %q", werkzeugSymbol(canvas.WerkzeugSprechblase).Name)
	}
	l.e.s.Auswahl = 0
	l.fokusiere(l.e)
	l.taste("K", 0)
	if l.e.s.Werkzeug != canvas.WerkzeugSprechblase {
		t.Fatalf("nach K: Werkzeug %v", l.e.s.Werkzeug)
	}
	if h := l.e.hinweis(); h == "" {
		t.Error("kein Hinweis der Fußzeile für die Sprechblase")
	}
}

// Aufziehen mit der Maus, tippen, bestätigen; dann der Schalter „Spitze“ der Leiste (aus, an), je ein Schritt.
func TestSprechblaseMausUndSchalter(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.e.s.Auswahl = 0
	l.e.s.SetzeWerkzeug(canvas.WerkzeugSprechblase)
	l.setle()
	if err := l.e.s.D.Entfernen(l.id); err != nil { // der Testtext läge unter dem Druckpunkt
		t.Fatal(err)
	}
	vorher := len(l.e.s.D.Objekte())
	l.ziehe(l.fensterPt(document.Pt(8, 26)), l.fensterPt(document.Pt(14, 6)))
	if !l.e.s.TextLaeuft() {
		t.Fatal("nach dem Aufziehen keine Texteingabe")
	}
	l.fokusiere(l.e)
	l.tippe("Hi")
	l.taste(key.NameReturn, 0)
	if l.e.s.TextLaeuft() || len(l.e.s.D.Objekte()) != vorher+1 {
		t.Fatalf("Eingabe %v, Objekte %d, Soll %d", l.e.s.TextLaeuft(), len(l.e.s.D.Objekte()), vorher+1)
	}
	o, _ := l.e.s.D.Objekt(l.e.s.Auswahl)
	tx, ok := o.(document.Text)
	if !ok || tx.Inhalt != "Hi" || tx.Spitze == nil {
		t.Fatalf("Objekt %+v", o)
	}
	if !l.e.spitze.rahmen.Empty() && !l.e.s.SpitzeAn() {
		t.Error("Schalter zeigt „aus“ bei einer Blase mit Spitze")
	}
	if l.e.spitze.rahmen.Empty() {
		t.Fatal("Schalter „Spitze“ nicht gezeichnet")
	}
	mitte := l.e.spitze.rahmen.Min.Add(l.e.spitze.rahmen.Max).Div(2)
	l.klick(mitte)
	if l.e.s.SpitzeAn() {
		t.Error("Klick auf den Schalter nahm die Spitze nicht weg")
	}
	l.klick(mitte)
	if !l.e.s.SpitzeAn() {
		t.Error("zweiter Klick setzte die Spitze nicht wieder")
	}
}

// Ein Objekt ohne Text zeigt den Schalter nicht (nur Werkzeug Sprechblase ohne Auswahl, Auswählen ohne Objekt).
func TestSprechblaseSchalterNurBeiText(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.e.s.Auswahl = 0
	l.e.s.SetzeWerkzeug(canvas.WerkzeugSprechblase)
	l.setle()
	if !l.e.spitze.rahmen.Empty() {
		t.Error("Schalter ohne gewählten Text")
	}
	l.e.s.SetzeWerkzeug(canvas.WerkzeugAuswahl)
	l.e.s.Auswahl = l.id
	l.setle()
	if l.e.spitze.rahmen.Empty() {
		t.Error("Schalter fehlt bei einem gewählten Text")
	}
}

// blasenEditor: Editor mit bunten Streifen und einer Blase (Schriftgrad 40, Ort (60,20), Spitze (20,110)).
func blasenEditor(t *testing.T, fuellung color.NRGBA, deckkraft int) *editor {
	t.Helper()
	i18n.Setze(nil)
	bild := image.NewNRGBA(image.Rect(0, 0, 160, 120))
	for y := 0; y < 120; y++ {
		for x := 0; x < 160; x++ {
			bild.SetNRGBA(x, y, color.NRGBA{R: uint8(40 + 20*(x/20)), G: uint8(220 - 25*(y/15)), B: uint8(30 + 25*((x+y)/30)), A: 255})
		}
	}
	d, err := document.Neu(bild)
	if err != nil {
		t.Fatal(err)
	}
	e := &editor{s: canvas.NeueSitzung(d)}
	sp := document.Pt(20, 110)
	id, err := d.Hinzufuegen(document.Text{Ort: document.Pt(60, 20), Inhalt: "M", Schriftgrad: 40,
		Farbe: color.NRGBA{R: 255, G: 255, B: 255, A: 255}, Hintergrund: fuellung, Spitze: &sp})
	if err != nil {
		t.Fatal(err)
	}
	if deckkraft < 100 && !e.s.SetzeDeckkraft(id, deckkraft) {
		t.Fatal("Deckkraft nicht gesetzt")
	}
	return e
}

// vergleiche zeichnet den Editor und den Renderer und liefert die größte Abweichung je Kanal in r.
func vergleiche(t *testing.T, e *editor, r image.Rectangle) int {
	t.Helper()
	soll, err := render.Bild(e.s.D)
	if err != nil {
		t.Fatal(err)
	}
	g := e.s.D.Groesse()
	ist := headlessBild(t, g, func(gtx layout.Context) {
		paint.NewImageOp(e.s.Leinwandbild()).Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		e.zeichneObjekte(gtx)
	})
	groesst := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			a, b := ist.RGBAAt(x, y), soll.NRGBAAt(x, y)
			groesst = max(groesst, absInt(int(a.R)-int(b.R)), absInt(int(a.G)-int(b.G)), absInt(int(a.B)-int(b.B)))
		}
	}
	return groesst
}

// dreieckInnen ist ein Rechteck von Pixeln, die sicher im Dreieck der Blase und außerhalb des Schilds liegen.
func dreieckInnen(t *testing.T, e *editor) image.Rectangle {
	t.Helper()
	o, _ := e.s.D.Objekt(e.s.D.Objekte()[0].Kennung())
	tx := o.(document.Text)
	a, sp, b, ok := tx.Spitzendreieck()
	if !ok {
		t.Fatal("kein Dreieck")
	}
	// Schwerpunkt der Fläche aus Basismitte und Spitze, ein kleines Quadrat um die Stelle 60 % zur Spitze.
	m := a.Plus(b).Mal(0.5)
	p := m.Plus(sp.Minus(m).Mal(0.5))
	if tx.Grenzen().Enthaelt(p) {
		t.Fatalf("Probepunkt %v liegt im Schild %v", p, tx.Grenzen())
	}
	return image.Rect(int(p.X)-1, int(p.Y)-1, int(p.X)+1, int(p.Y)+1)
}

// ADR-0021 Nachtrag 13 Punkt 3: Der Editor füllt Schild und Spitze als ein Vieleck, wie der Renderer; gedeckt, halb
// durchsichtig und unter Ebenen-Deckkraft (Punkt 4, ADR-0020 Nachtrag 14: Mischung im sRGB-Raum).
func TestEditorZeichnetSprechblaseGefuellt(t *testing.T) {
	for _, fall := range []struct {
		name    string
		farbe   color.NRGBA
		deckung int
	}{
		{"deckend", color.NRGBA{R: 200, G: 30, B: 30, A: 255}, 100},
		{"halbdurchsichtig", color.NRGBA{R: 200, G: 30, B: 30, A: 120}, 100},
		{"Ebene 50 %", color.NRGBA{R: 200, G: 30, B: 30, A: 255}, 50},
	} {
		e := blasenEditor(t, fall.farbe, fall.deckung)
		r := dreieckInnen(t, e)
		if d := vergleiche(t, e, r); d > 2 {
			t.Errorf("%s: größte Abweichung %d je Kanal im Dreieck %v, Soll höchstens 2", fall.name, d, r)
		}
		// Das Dreieck ist wirklich gefüllt (nicht der Untergrund).
		b := headlessBild(t, e.s.D.Groesse(), func(gtx layout.Context) { e.zeichneObjekte(gtx) })
		if p := b.RGBAAt(r.Min.X, r.Min.Y); p.A == 0 {
			t.Errorf("%s: Dreieck bei %v bleibt leer", fall.name, r.Min)
		}
	}
}

// Ohne Füllung zeichnet der Editor nur die beiden Schenkel in der Schriftfarbe (Punkt 3), nicht das Schild.
func TestEditorZeichnetSprechblaseOhneFuellung(t *testing.T) {
	e := blasenEditor(t, color.NRGBA{}, 100)
	o, _ := e.s.D.Objekt(e.s.D.Objekte()[0].Kennung())
	tx := o.(document.Text)
	a, sp, _, ok := tx.Spitzendreieck()
	if !ok {
		t.Fatal("kein Dreieck")
	}
	b := headlessBild(t, e.s.D.Groesse(), func(gtx layout.Context) { e.zeichneObjekte(gtx) })
	mitte := a.Plus(sp).Mal(0.5) // Mitte des einen Schenkels
	if p := b.RGBAAt(int(mitte.X), int(mitte.Y)); p.R < 200 || p.G < 200 || p.B < 200 || p.A < 200 {
		t.Errorf("Mitte des Schenkels bei %v: Pixel %v, Soll nahe weiß (Schriftfarbe, Kantenglättung)", mitte, p)
	}
	innen := tx.Grenzen().Max.Minus(document.Pt(3, 3)) // im Schild, aber nicht auf der Schrift
	if p := b.RGBAAt(int(innen.X), int(innen.Y)); p.A != 0 {
		t.Errorf("Schild bei %v gefüllt (%v), Soll durchsichtig", innen, p)
	}
	if d := vergleiche(t, e, image.Rect(int(mitte.X)-1, int(mitte.Y)-1, int(mitte.X)+1, int(mitte.Y)+1)); d > 4 {
		t.Errorf("Abweichung vom Renderer auf dem Schenkel: %d", d)
	}
}

// Die Vorschau der Ebenenliste zeigt die ganze Blase samt Spitze: Die Hülle bestimmt den Maßstab, das Schild rückt in
// die obere rechte Ecke, die obere linke bleibt leer (mit dem Schild allein füllte es die Vorschau aus).
func TestEbenenVorschauZeigtSpitze(t *testing.T) {
	e := blasenEditor(t, color.NRGBA{R: 200, G: 30, B: 30, A: 255}, 100)
	o, _ := e.s.D.Objekt(e.s.D.Objekte()[0].Kennung())
	v := image.Rect(0, 0, 80, 60)
	b := headlessBild(t, v.Size(), func(gtx layout.Context) { e.zeichneVorschau(gtx, o, v) })
	if p := b.RGBAAt(5, 5); p.R != 255 || p.G != 255 || p.B != 255 {
		t.Errorf("Pixel oben links %v, Soll Weiß: Schild und Spitze sollten zusammen in die Vorschau passen", p)
	}
	rot, links, untenRechts := 0, 0, 0
	for y := 0; y < v.Dy(); y++ {
		for x := 0; x < v.Dx(); x++ {
			p := b.RGBAAt(x, y)
			if p.R > 150 && p.G < 100 {
				rot++
				if x >= 35 && y >= 45 {
					untenRechts++
				}
			}
			// Unten links liegt die Spitze der Blase (Original: unterer linker Rand der Hülle).
			if x < 25 && y >= 40 && (p.R != 255 || p.G != 255 || p.B != 255) {
				links++
			}
		}
	}
	if rot == 0 {
		t.Error("kein Rot in der Vorschau")
	}
	t.Logf("nicht weiße Pixel unten links: %d, rote unten rechts: %d", links, untenRechts)
	if untenRechts != 0 {
		t.Errorf("%d rote Pixel unten rechts: das Schild füllt die Vorschau, die Spitze bestimmt den Maßstab nicht mit", untenRechts)
	}
	if links == 0 {
		t.Error("die Spitze fehlt unten links in der Vorschau")
	}
}

// Das Kästchen „Spitze“ der Leiste zeigt den Zustand (seit Karte #369 ein Kästchen mit Haken wie „Shadow“): grün bei
// einer Blase, leer bei einem Text. Gemessen oben links im Kästchen, abseits des Hakens.
func TestSpitzenSchalterZeigtZustand(t *testing.T) {
	e := blasenEditor(t, color.NRGBA{R: 200, A: 255}, 100)
	e.thema = material.NewTheme()
	id := e.s.D.Objekte()[0].Kennung()
	e.s.Auswahl = id
	kaestchenFarbe := func() color.NRGBA {
		b := headlessBild(t, image.Pt(160, 60), func(gtx layout.Context) {
			e.zeichneSpitze(gtx, 4, 4, 100)
		})
		p := b.RGBAAt(4+4, 4+(26-18)/2+4) // Kästchen 18 dp ab (4, 8), Haken ab 22 % der Breite
		return color.NRGBA{R: p.R, G: p.G, B: p.B, A: p.A}
	}
	if got := kaestchenFarbe(); got != farbeAuswahl {
		t.Errorf("Kästchen bei einer Blase: %v, Soll die Auswahlfarbe %v", got, farbeAuswahl)
	}
	e.s.SchalteSpitze()
	if got := kaestchenFarbe(); got == farbeAuswahl {
		t.Errorf("Kästchen bei einem Text: %v, Soll leer", got)
	}
	if r := e.spitze.rahmen; r.Min.X != 4 || r.Dx() < 100 {
		t.Errorf("Klickfläche %v umfasst nicht Kästchen und Wort", r)
	}
}

// Auswahlrahmen und Griff: Der Rahmen umschließt Schild und Spitze (Huelle), der Griff sitzt an der Spitze.
func TestEditorAuswahlrahmenSprechblase(t *testing.T) {
	e := blasenEditor(t, color.NRGBA{R: 200, A: 255}, 100)
	o, _ := e.s.D.Objekt(e.s.D.Objekte()[0].Kennung())
	tx := o.(document.Text)
	rs := auswahlRechtecke(tx)
	if len(rs) != 1 || rs[0] != tx.Huelle() {
		t.Errorf("Rahmen %v, Soll die Hülle %v", rs, tx.Huelle())
	}
	if g := canvas.Griffe(tx); len(g) != 1 || g[0].Ort != *tx.Spitze {
		t.Errorf("Griffe %v", g)
	}
}
