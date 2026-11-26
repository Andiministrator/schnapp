// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op/paint"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/render"
)

// ADR-0021 Nachtrag 26 Punkt 4: Der Editor zeigt den Schatten wie der Renderer — weich, versetzt, unter dem Objekt, mit
// der Deckkraft der Ebene über Objekt und Schatten zusammen. Verglichen wird das Zeichnen ohne Fenster bei Zoom 100 %
// mit render.Bild über einem bunten Bild.
func TestEditorZeichnetSchattenWieRenderer(t *testing.T) {
	i18n.Setze(nil)
	bild := image.NewNRGBA(image.Rect(0, 0, 240, 160))
	for y := range 160 {
		for x := range 240 {
			bild.SetNRGBA(x, y, color.NRGBA{R: uint8(60 + x/2), G: uint8(220 - y), B: uint8(90 + (x+y)/4), A: 255})
		}
	}
	d, err := document.Neu(bild)
	if err != nil {
		t.Fatal(err)
	}
	mit := func(o document.Objekt, sch document.Schatten, deckkraft int) document.Objekt {
		return document.MitEbene(o, document.Ebene{Schatten: sch}.MitDeckkraft(deckkraft))
	}
	rot := document.Strich{Farbe: color.NRGBA{R: 210, G: 40, B: 40, A: 255}, Breite: 5}
	for _, o := range []document.Objekt{
		mit(document.Kasten{Bereich: document.R(document.Pt(20, 20), document.Pt(100, 70)), Strich: rot,
			Fuellung: color.NRGBA{R: 255, G: 255, B: 255, A: 255}}, document.StandardSchatten(1), 100),
		mit(document.GeraderPfeil(document.Pt(120, 30), document.Pt(220, 90), rot),
			document.Schatten{An: true, Winkel: 90, Laenge: 8, Weich: 4, Farbe: color.NRGBA{B: 120, A: 160}}, 100),
		mit(document.Kasten{Bereich: document.R(document.Pt(30, 95), document.Pt(110, 140)),
			Fuellung: color.NRGBA{R: 20, G: 20, B: 20, A: 255}}, document.Schatten{An: true, Winkel: 45, Laenge: 10,
			Farbe: color.NRGBA{A: 255}}, 50),
	} {
		if _, err := d.Hinzufuegen(o); err != nil {
			t.Fatal(err)
		}
	}
	e := &editor{s: canvas.NeueSitzung(d)}
	soll, err := render.Bild(d)
	if err != nil {
		t.Fatal(err)
	}
	ist := headlessBild(t, image.Pt(240, 160), func(gtx layout.Context) {
		paint.NewImageOp(e.s.Leinwandbild()).Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		e.zeichneObjekte(gtx)
	})
	groesst, grob := 0, 0
	for y := 0; y < 160; y++ {
		for x := 0; x < 240; x++ {
			a, b := ist.RGBAAt(x, y), soll.NRGBAAt(x, y)
			diff := max(absInt(int(a.R)-int(b.R)), absInt(int(a.G)-int(b.G)), absInt(int(a.B)-int(b.B)))
			groesst = max(groesst, diff)
			if diff > 12 {
				grob++
				if grob <= 8 {
					t.Logf("bei %d,%d: Editor %v, Renderer %v", x, y, a, b)
				}
			}
		}
	}
	t.Logf("größte Abweichung %d, %d Pixel über 12", groesst, grob)
	if grob > 20 {
		t.Errorf("%d Pixel weichen um mehr als 12 ab", grob)
	}
	// Der Schatten ist da: unter dem gefüllten Kasten rechts unten dunkler als das Bild.
	if a, b := ist.RGBAAt(104, 76), bild.NRGBAAt(104, 76); int(a.G) > int(b.G)-10 {
		t.Errorf("kein Schatten rechts unter dem Kasten: Editor %v, Bild %v", a, b)
	}
}

// F-843: Ein Schattenbild wird nur neu gerechnet, wenn sich sein Objekt oder die Ansicht ändert — nicht bei jeder
// Änderung eines anderen Objekts; die Pixel aller gemerkten Bilder bleiben unter schattenSpeicher.
func TestSchattenZwischenspeicher(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 200, 120)))
	if err != nil {
		t.Fatal(err)
	}
	k := document.Kasten{Bereich: document.R(document.Pt(20, 20), document.Pt(80, 60)), Fuellung: color.NRGBA{A: 255}}
	idA, _ := d.Hinzufuegen(document.MitEbene(k, document.Ebene{Schatten: document.StandardSchatten(1)}))
	idB, _ := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(100, 20), document.Pt(150, 60)),
		Fuellung: color.NRGBA{R: 255, A: 255}})
	e := &editor{s: canvas.NeueSitzung(d)}
	a, _ := d.Objekt(idA)
	e.schattenFuer(a)
	e.schattenFuer(a)
	if e.schattenNeu != 1 {
		t.Fatalf("zweimal dasselbe: %d Neuberechnungen", e.schattenNeu)
	}
	b, _ := d.Objekt(idB)
	if err := d.Ersetzen(canvas.Verschiebe(b, document.Pt(5, 0))); err != nil {
		t.Fatal(err)
	}
	e.schattenFuer(a)
	if e.schattenNeu != 1 {
		t.Errorf("Änderung an einem anderen Objekt rechnete den Schatten neu (%d)", e.schattenNeu)
	}
	if err := d.SetzeZuschnitt(image.Rect(10, 0, 200, 120)); err != nil {
		t.Fatal(err)
	}
	e.schattenFuer(a)
	if e.schattenNeu != 2 {
		t.Errorf("Zuschnitt rechnete den Schatten nicht neu (%d)", e.schattenNeu)
	}
	a2 := canvas.Verschiebe(a, document.Pt(3, 3))
	if err := d.Ersetzen(a2); err != nil {
		t.Fatal(err)
	}
	e.schattenFuer(a2)
	if e.schattenNeu != 3 || e.schattenBytes <= 0 || e.schattenBytes > schattenSpeicher {
		t.Errorf("geändertes Objekt: %d Neuberechnungen, %d Bytes", e.schattenNeu, e.schattenBytes)
	}
}

// V-2 der Sitzungs-Kritik 26.11. (5): Der Schatten einer Lupe hängt an Pixeln und Kante; er wird nach jeder
// Dokumentänderung neu gerechnet (Stand im Schlüssel), der eines Kastens nicht.
func TestSchattenLupeNeuNachAenderung(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 300, 200)))
	if err != nil {
		t.Fatal(err)
	}
	l := document.Lupe{Quelle: document.R(document.Pt(10, 10), document.Pt(40, 40)), Ziel: document.R(document.Pt(100, 20),
		document.Pt(160, 80)), Strich: document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 2}}
	idL, _ := d.Hinzufuegen(document.MitEbene(l, document.Ebene{Schatten: document.StandardSchatten(1)}))
	idK, _ := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(200, 100), document.Pt(250, 150)),
		Fuellung: color.NRGBA{A: 255}})
	e := &editor{s: canvas.NeueSitzung(d)}
	lo, _ := d.Objekt(idL)
	e.schattenFuer(lo)
	k, _ := d.Objekt(idK)
	if err := d.Ersetzen(canvas.Verschiebe(k, document.Pt(3, 0))); err != nil {
		t.Fatal(err)
	}
	e.schattenFuer(lo)
	if e.schattenNeu != 2 {
		t.Errorf("Lupe nach einer Dokumentänderung: %d Neuberechnungen, Soll 2", e.schattenNeu)
	}
}

// V-3: Über der Speichergrenze fallen nur Bilder früherer Runden weg; die Schatten eines Bilds bleiben gemerkt, auch
// wenn sie zusammen über der Grenze liegen — das zweite Bild rechnet nichts neu.
func TestSchattenVerdraengtEinzeln(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 400, 300)))
	if err != nil {
		t.Fatal(err)
	}
	var ids []int
	for i := range 4 {
		k := document.Kasten{Bereich: document.R(document.Pt(float64(20+90*i), 20), document.Pt(float64(80+90*i), 80)),
			Fuellung: color.NRGBA{A: 255}}
		id, _ := d.Hinzufuegen(document.MitEbene(k, document.Ebene{Schatten: document.StandardSchatten(1)}))
		ids = append(ids, id)
	}
	e := &editor{s: canvas.NeueSitzung(d)}
	alt := schattenSpeicher
	defer func() { schattenSpeicher = alt }()
	o0, _ := d.Objekt(ids[0])
	eins := e.schattenFuer(o0).bytes
	e.schatten, e.schattenBytes, e.schattenNeu = nil, 0, 0
	schattenSpeicher = 2*eins + eins/2 // Platz für zweieinhalb Bilder
	runde := func() {
		for _, id := range ids {
			o, _ := d.Objekt(id)
			e.schattenFuer(o)
		}
		e.schattenAufraeumen()
	}
	runde()
	if e.schattenNeu != 4 {
		t.Fatalf("erste Runde: %d", e.schattenNeu)
	}
	runde()
	if e.schattenNeu != 4 {
		t.Errorf("zweite Runde rechnete neu: %d", e.schattenNeu)
	}
	// Nur zwei Objekte gezeichnet, dann ein fünftes: Die zwei nicht gezeichneten fallen weg; die drei der laufenden Runde
	// bleiben, auch wenn sie zusammen über der Grenze liegen.
	k := document.Kasten{Bereich: document.R(document.Pt(20, 150), document.Pt(80, 210)), Fuellung: color.NRGBA{A: 255}}
	id5, _ := d.Hinzufuegen(document.MitEbene(k, document.Ebene{Schatten: document.StandardSchatten(1)}))
	for _, id := range ids[:2] {
		o, _ := d.Objekt(id)
		e.schattenFuer(o)
	}
	o5, _ := d.Objekt(id5)
	e.schattenFuer(o5)
	if _, ok := e.schatten[ids[0]]; !ok {
		t.Error("ein Bild der laufenden Runde wurde verdrängt")
	}
	_, drei := e.schatten[ids[2]]
	_, vier := e.schatten[ids[3]]
	if len(e.schatten) != 3 || drei || vier || e.schattenBytes != 3*eins {
		t.Errorf("%d Bilder (nicht gezeichnete noch da: %v %v), %d Bytes", len(e.schatten), drei, vier, e.schattenBytes)
	}
}
