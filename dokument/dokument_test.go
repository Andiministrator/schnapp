// SPDX-License-Identifier: GPL-3.0-or-later

package dokument

import (
	"errors"
	"image"
	"image/color"
	"math"
	"reflect"
	"testing"
)

// testbild ist ein synthetisches Bild, in dem jedes Pixel seine Lage trägt: R = x, G = y.
func testbild(b, h int) *image.NRGBA {
	m := image.NewNRGBA(image.Rect(0, 0, b, h))
	for y := 0; y < h; y++ {
		for x := 0; x < b; x++ {
			m.SetNRGBA(x, y, color.NRGBA{uint8(x), uint8(y), 7, 255})
		}
	}
	return m
}

func neu(t *testing.T, b, h int) *Dokument {
	t.Helper()
	d, err := Neu(testbild(b, h))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

var rot = Strich{Farbe: color.NRGBA{255, 0, 0, 255}, Breite: 4}

func TestNeu(t *testing.T) {
	quelle := testbild(10, 5)
	d, err := Neu(quelle.SubImage(image.Rect(2, 1, 10, 5)))
	if err != nil {
		t.Fatal(err)
	}
	o := d.Original()
	if o.Rect != image.Rect(0, 0, 8, 4) || o.NRGBAAt(0, 0) != (color.NRGBA{2, 1, 7, 255}) {
		t.Errorf("Original %v, Pixel (0,0) %v", o.Rect, o.NRGBAAt(0, 0))
	}
	// Kopien: Änderungen an Quelle oder geliefertem Original berühren das Dokument nicht.
	quelle.SetNRGBA(2, 1, color.NRGBA{})
	o.SetNRGBA(0, 0, color.NRGBA{})
	if d.Original().NRGBAAt(0, 0) != (color.NRGBA{2, 1, 7, 255}) {
		t.Error("Dokument teilt Pixel mit Quelle oder Kopie")
	}
	for _, r := range []image.Rectangle{image.Rect(0, 0, 0, 5), image.Rect(0, 0, MaxSeite+1, 1)} {
		if _, err := Neu(image.NewNRGBA(r)); !errors.Is(err, ErrBildGroesse) {
			t.Errorf("Neu(%v) = %v", r, err)
		}
	}
}

func TestObjekteUndVerlauf(t *testing.T) {
	d := neu(t, 100, 100)
	id1, err := d.Hinzufuegen(GeraderPfeil(Pt(0, 0), Pt(50, 0), rot))
	if err != nil {
		t.Fatal(err)
	}
	id2, _ := d.Hinzufuegen(Kasten{Bereich: R(Pt(10, 10), Pt(20, 20)), Strich: rot})
	if id1 != 1 || id2 != 2 || len(d.Objekte()) != 2 {
		t.Fatalf("Kennungen %d %d, %d Objekte", id1, id2, len(d.Objekte()))
	}
	o, _ := d.Objekt(id2)
	k := o.(Kasten)
	k.Eckradius = 3
	if err := d.Ersetzen(k); err != nil {
		t.Fatal(err)
	}
	if err := d.Entfernen(id1); err != nil {
		t.Fatal(err)
	}
	// Zurück durch alle drei Schritte, dann wieder vor.
	stand := func() []Objekt { return d.Objekte() }
	nachEntfernen := stand()
	d.Rueckgaengig()
	nachErsetzen := stand()
	d.Rueckgaengig()
	d.Rueckgaengig()
	if got := stand(); len(got) != 1 || got[0].Kennung() != id1 {
		t.Fatalf("nach dreimal Rückgängig: %v", got)
	}
	d.Wiederholen()
	d.Wiederholen()
	if !reflect.DeepEqual(stand(), nachErsetzen) {
		t.Errorf("Wiederholen: %v, want %v", stand(), nachErsetzen)
	}
	d.Wiederholen()
	if !reflect.DeepEqual(stand(), nachEntfernen) {
		t.Errorf("Wiederholen: %v, want %v", stand(), nachEntfernen)
	}
	if err := d.Wiederholen(); !errors.Is(err, ErrNichtsDa) {
		t.Errorf("Wiederholen am Ende: %v", err)
	}
	// Neue Änderung verwirft die zurückgenommenen.
	d.Rueckgaengig()
	d.Hinzufuegen(Nummer{Mitte: Pt(5, 5), Wert: 1, Radius: 8})
	if d.KannWiederholen() {
		t.Error("Wiederholen nach neuer Änderung möglich")
	}
	if d.Entfernen(99) != ErrUnbekannt || d.Ersetzen(Kasten{ID: 99}) != ErrUnbekannt || d.Ersetzen(Kasten{}) != ErrUnbekannt {
		t.Error("unbekannte Kennung ohne Fehler")
	}
}

func TestUngueltigeObjekte(t *testing.T) {
	d := neu(t, 10, 10)
	nan := math.NaN()
	for _, o := range []Objekt{
		Pfeil{A: Pt(nan, 0)},
		GeraderPfeil(Pt(0, 0), Pt(1, 1), Strich{Breite: -1}),
		Kasten{Bereich: Rechteck{Pt(5, 5), Pt(1, 1)}},
		Kasten{Bereich: R(Pt(0, 0), Pt(1, 1)), Eckradius: math.Inf(1)},
		Text{Inhalt: "x"},
		Nummer{Radius: 0},
	} {
		if _, err := d.Hinzufuegen(o); err == nil {
			t.Errorf("%#v ohne Fehler", o)
		}
	}
	if len(d.Objekte()) != 0 || d.KannRueckgaengig() {
		t.Error("ungültiges Objekt hinterließ Spuren")
	}
}

// F-18: Rückgängig während eines Zugs ändert nichts und meldet ErrZugLaeuft.
func TestRueckgaengigWaehrendZug(t *testing.T) {
	d := neu(t, 100, 100)
	d.Hinzufuegen(GeraderPfeil(Pt(0, 0), Pt(10, 0), rot))
	id, err := d.BeginneNeuenZug(GeraderPfeil(Pt(20, 20), Pt(21, 21), rot))
	if err != nil {
		t.Fatal(err)
	}
	vorher := d.Objekte()
	if err := d.Rueckgaengig(); !errors.Is(err, ErrZugLaeuft) {
		t.Fatalf("Rückgängig im Zug: %v", err)
	}
	if d.KannRueckgaengig() || d.KannWiederholen() {
		t.Error("Kann… meldet im Zug true")
	}
	if !reflect.DeepEqual(d.Objekte(), vorher) {
		t.Error("Rückgängig im Zug hat Objekte verändert")
	}
	// Alle anderen Änderungen sind ebenso gesperrt.
	sperren := map[string]error{
		"Wiederholen":    d.Wiederholen(),
		"Entfernen":      d.Entfernen(1),
		"Ersetzen":       d.Ersetzen(GeraderPfeil(Pt(0, 0), Pt(1, 1), rot).mitKennung(1)),
		"Zuschnitt":      d.SetzeZuschnitt(image.Rect(0, 0, 5, 5)),
		"Streifen":       d.SetzeStreifen(nil),
		"Schwärzen":      d.Schwaerzen(image.Rect(0, 0, 5, 5), Schwaerzung{}),
		"BeginneZug":     d.BeginneZug(1),
		"VergissVerlauf": d.VergissVerlauf(),
	}
	_, sperren["Hinzufuegen"] = d.Hinzufuegen(Nummer{Radius: 1})
	_, sperren["BeginneNeuenZug"] = d.BeginneNeuenZug(Nummer{Radius: 1})
	for name, err := range sperren {
		if !errors.Is(err, ErrZugLaeuft) {
			t.Errorf("%s im Zug: %v", name, err)
		}
	}
	// Weiterziehen, beenden: genau ein Schritt im Verlauf.
	if err := d.ZugSetze(GeraderPfeil(Pt(20, 20), Pt(80, 60), rot).mitKennung(id)); err != nil {
		t.Fatal(err)
	}
	if err := d.BeendeZug(); err != nil {
		t.Fatal(err)
	}
	if o, _ := d.Objekt(id); o.(Pfeil).B != Pt(80, 60) {
		t.Errorf("Zug nicht übernommen: %v", o)
	}
	d.Rueckgaengig()
	if _, ok := d.Objekt(id); ok || len(d.Objekte()) != 1 {
		t.Error("ein Rückgängig entfernt den neuen Pfeil nicht")
	}
}

func TestZugAnVorhandenemObjekt(t *testing.T) {
	d := neu(t, 100, 100)
	id, _ := d.Hinzufuegen(GeraderPfeil(Pt(0, 0), Pt(100, 0), rot))
	alt, _ := d.Objekt(id)

	// Unveränderter Zug hinterlässt keinen Verlaufsschritt.
	d.BeginneZug(id)
	d.BeendeZug()
	d.Rueckgaengig() // nimmt das Hinzufügen zurück
	if d.KannRueckgaengig() {
		t.Error("unveränderter Zug landete im Verlauf")
	}
	d.Wiederholen()

	// Griff ziehen: ein Schritt, Rückgängig stellt den geraden Pfeil her.
	d.BeginneZug(id)
	for _, y := range []float64{10, 20, 40} {
		if err := d.ZugSetze(alt.(Pfeil).MitGriff(Pt(50, y))); err != nil {
			t.Fatal(err)
		}
	}
	if err := d.ZugSetze(Kasten{ID: id + 1, Bereich: R(Pt(0, 0), Pt(1, 1))}); err == nil {
		t.Error("ZugSetze mit fremder Kennung ohne Fehler")
	}
	d.BeendeZug()
	if o, _ := d.Objekt(id); o.(Pfeil).Griff() != Pt(50, 40) {
		t.Errorf("Griff %v", o.(Pfeil).Griff())
	}
	d.Rueckgaengig()
	if o, _ := d.Objekt(id); o != alt {
		t.Errorf("Rückgängig nach Zug: %v, want %v", o, alt)
	}

	// Abbruch stellt her und hinterlässt nichts; neues Objekt verschwindet.
	d.BeginneZug(id)
	d.ZugSetze(alt.(Pfeil).MitGriff(Pt(50, 90)))
	d.BrecheZugAb()
	if o, _ := d.Objekt(id); o != alt {
		t.Error("Abbruch stellt nicht her")
	}
	nid, _ := d.BeginneNeuenZug(Nummer{Radius: 5, Wert: 1})
	d.BrecheZugAb()
	if _, ok := d.Objekt(nid); ok {
		t.Error("abgebrochenes neues Objekt bleibt")
	}
	if d.BeendeZug() != ErrKeinZug || d.BrecheZugAb() != ErrKeinZug || d.ZugSetze(alt) != ErrKeinZug {
		t.Error("Zug-Befehle ohne Zug ohne Fehler")
	}
	if d.BeginneZug(99) != ErrUnbekannt {
		t.Error("Zug an unbekanntem Objekt")
	}
}

func TestPfeil(t *testing.T) {
	p := GeraderPfeil(Pt(0, 0), Pt(100, 0), rot)
	if p.Griff() != Pt(50, 0) {
		t.Fatalf("Griff %v", p.Griff())
	}
	// Griff nach (50,40): K = 2·(50,40) − (50,0) = (50,80); die Kurve läuft durch den Griff, Enden bleiben.
	q := p.MitGriff(Pt(50, 40))
	if q.K != Pt(50, 80) || q.Griff() != Pt(50, 40) || q.Punkt(0) != q.A || q.Punkt(1) != q.B {
		t.Errorf("MitGriff: K %v, Griff %v", q.K, q.Griff())
	}
	if r := GeraderPfeil(Pt(0, 0), Pt(0, 10), rot).RichtungAmEnde(); r != Pt(0, 1) {
		t.Errorf("Richtung %v", r)
	}
	if r := (Pfeil{B: Pt(3, 4), K: Pt(3, 4)}).RichtungAmEnde(); math.Abs(r.X-0.6) > 1e-12 || math.Abs(r.Y-0.8) > 1e-12 {
		t.Errorf("Richtung bei K = B: %v", r)
	}
	if r := (Pfeil{}).RichtungAmEnde(); r != Pt(1, 0) {
		t.Errorf("Richtung ohne Länge: %v", r)
	}
	if g := q.Grenzen(); g != R(Pt(-2, -2), Pt(102, 82)) {
		t.Errorf("Grenzen %v", g)
	}
}

func TestNaechsteNummer(t *testing.T) {
	d := neu(t, 10, 10)
	if d.NaechsteNummer() != 1 {
		t.Fatal("leeres Dokument")
	}
	d.Hinzufuegen(Nummer{Wert: 1, Radius: 3})
	d.Hinzufuegen(Nummer{Wert: 4, Radius: 3})
	d.Hinzufuegen(Kasten{Bereich: R(Pt(0, 0), Pt(1, 1))})
	if n := d.NaechsteNummer(); n != 5 {
		t.Errorf("NaechsteNummer = %d", n)
	}
}

func TestAnsicht(t *testing.T) {
	d := neu(t, 100, 50)
	// Zuschnitt x 10..90, y 5..45; Spalten 30..40 und 60..65 heraus, Zeilen 20..30 heraus.
	if err := d.SetzeZuschnitt(image.Rect(90, 45, 10, 5)); err != nil { // vertauschte Ecken werden geordnet
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]Streifen{{false, 60, 65}, {true, 20, 30}, {false, 30, 40}}); err != nil {
		t.Fatal(err)
	}
	if g := d.Groesse(); g != image.Pt(80-15, 40-10) {
		t.Fatalf("Groesse %v", g)
	}
	for _, f := range []struct{ orig, ans Punkt }{
		{Pt(10, 5), Pt(0, 0)},
		{Pt(29, 19), Pt(19, 14)},
		{Pt(40, 30), Pt(20, 15)},
		{Pt(66, 31), Pt(66-10-15, 31-5-10)},
	} {
		if a := d.ZuAnsicht(f.orig); a != f.ans {
			t.Errorf("ZuAnsicht(%v) = %v, want %v", f.orig, a, f.ans)
		}
		if o := d.ZuOriginal(f.ans); o != f.orig {
			t.Errorf("ZuOriginal(%v) = %v, want %v", f.ans, o, f.orig)
		}
	}
	// Punkte im Streifen fallen auf seine Kante.
	if a := d.ZuAnsicht(Pt(35, 25)); a != Pt(20, 15) {
		t.Errorf("Punkt im Streifen: %v", a)
	}
	// Pixel der Ansicht kommen aus den richtigen Stellen des Originals (R = x, G = y).
	b := d.Ansichtsbild()
	if b.Rect.Size() != d.Groesse() {
		t.Fatalf("Ansichtsbild %v", b.Rect)
	}
	for _, f := range []struct{ x, y, ox, oy int }{{0, 0, 10, 5}, {19, 14, 29, 19}, {20, 15, 40, 30}, {64, 29, 89, 44}} {
		if c := b.NRGBAAt(f.x, f.y); c.R != uint8(f.ox) || c.G != uint8(f.oy) {
			t.Errorf("Ansicht (%d,%d) = %v, want aus (%d,%d)", f.x, f.y, c, f.ox, f.oy)
		}
	}
	// Rückgängig hebt Streifen, dann Zuschnitt auf.
	d.Rueckgaengig()
	if g := d.Groesse(); g != image.Pt(80, 40) {
		t.Errorf("nach Rückgängig Streifen: %v", g)
	}
	d.Rueckgaengig()
	if g := d.Groesse(); g != image.Pt(100, 50) || d.Zuschnitt() != image.Rect(0, 0, 100, 50) {
		t.Errorf("nach Rückgängig Zuschnitt: %v", g)
	}
}

func TestAnsichtUngueltig(t *testing.T) {
	d := neu(t, 100, 50)
	for name, err := range map[string]error{
		"Zuschnitt außerhalb":   d.SetzeZuschnitt(image.Rect(-1, 0, 10, 10)),
		"Zuschnitt leer":        d.SetzeZuschnitt(image.Rect(5, 5, 5, 10)),
		"Streifen außerhalb":    d.SetzeStreifen([]Streifen{{true, 40, 51}}),
		"Streifen verkehrt":     d.SetzeStreifen([]Streifen{{false, 10, 10}}),
		"Streifen überlappen":   d.SetzeStreifen([]Streifen{{false, 10, 20}, {false, 15, 30}}),
		"Ansicht würde leer":    d.SetzeStreifen([]Streifen{{false, 0, 100}}),
		"Ansicht leer gestückt": d.SetzeStreifen([]Streifen{{true, 0, 25}, {true, 25, 50}}),
	} {
		if err == nil {
			t.Errorf("%s: kein Fehler", name)
		}
	}
	if d.KannRueckgaengig() {
		t.Error("ungültige Änderung im Verlauf")
	}
	// Gleiche Lage auf verschiedenen Achsen überlappt nicht.
	if err := d.SetzeStreifen([]Streifen{{false, 10, 20}, {true, 10, 20}}); err != nil {
		t.Error(err)
	}
}

func TestSchwaerzen(t *testing.T) {
	d := neu(t, 40, 20)
	schwarz := color.NRGBA{0, 0, 0, 255}
	// Bereich ragt über den Rand: wird begrenzt.
	if err := d.Schwaerzen(image.Rect(30, 10, 50, 30), Schwaerzung{Farbe: schwarz}); err != nil {
		t.Fatal(err)
	}
	o := d.Original()
	for _, p := range []image.Point{{30, 10}, {39, 19}} {
		if o.NRGBAAt(p.X, p.Y) != schwarz {
			t.Errorf("Pixel %v nicht geschwärzt: %v", p, o.NRGBAAt(p.X, p.Y))
		}
	}
	if o.NRGBAAt(29, 10) == schwarz || o.NRGBAAt(30, 9) == schwarz {
		t.Error("Pixel außerhalb geschwärzt")
	}
	// Auch die Ansicht (aus dem Original abgeleitet) zeigt nur Schwarz, auch nach Zuschnitt.
	d.SetzeZuschnitt(image.Rect(25, 5, 40, 20))
	if c := d.Ansichtsbild().NRGBAAt(5, 5); c != schwarz {
		t.Errorf("Ansicht zeigt %v", c)
	}
	// Rückgängig (Zuschnitt, dann Schwärzen) holt die Pixel zurück, solange der Verlauf im Speicher ist ...
	d.Rueckgaengig()
	d.Rueckgaengig()
	if c := d.Original().NRGBAAt(35, 15); c != (color.NRGBA{35, 15, 7, 255}) {
		t.Errorf("Rückgängig Schwärzen: %v", c)
	}
	// ... nach VergissVerlauf nicht mehr.
	d.Wiederholen()
	d.VergissVerlauf()
	if d.Rueckgaengig() != ErrNichtsDa || d.Original().NRGBAAt(35, 15) != schwarz {
		t.Error("nach VergissVerlauf zurückholbar")
	}
	if d.Schwaerzen(image.Rect(50, 50, 60, 60), Schwaerzung{}) == nil {
		t.Error("Bereich außerhalb ohne Fehler")
	}
}

func TestVerpixeln(t *testing.T) {
	d := neu(t, 20, 10)
	if d.Schwaerzen(image.Rect(0, 0, 20, 10), Schwaerzung{Block: MinBlock - 1}) == nil {
		t.Error("zu kleine Blöcke ohne Fehler")
	}
	if err := d.Schwaerzen(image.Rect(0, 0, 20, 10), Schwaerzung{Block: 8}); err != nil {
		t.Fatal(err)
	}
	o := d.Original()
	// Restblöcke unter MinBlock werden dem Nachbarn zugeschlagen (F-61): Spalten 0..8 und 8..20 (Rest 16..20
	// verschmolzen), Zeilen 0..10 (Rest 8..10 verschmolzen). Block (0..8, 0..10): Mittel x 0..7 = 3 (3,5 abgerundet),
	// y 0..9 = 4. Block (8..20, 0..10): Mittel x 8..19 = 13, y = 4.
	for _, f := range []struct {
		x, y int
		r, g uint8
	}{{0, 0, 3, 4}, {7, 9, 3, 4}, {8, 0, 13, 4}, {19, 9, 13, 4}} {
		if c := o.NRGBAAt(f.x, f.y); c.R != f.r || c.G != f.g {
			t.Errorf("Pixel (%d,%d) = %v, want R %d G %d", f.x, f.y, c, f.r, f.g)
		}
	}
}

// F-61: Kein Pixel eines verpixelten Bereichs bleibt im Original stehen, auch bei 8k+1 Pixeln Kantenlänge; zu
// schmale Bereiche und durchsichtige Füllungen werden abgelehnt.
func TestVerpixelnOhneRestpixel(t *testing.T) {
	d := neu(t, 40, 40)
	vorher := d.Original()
	if err := d.Schwaerzen(image.Rect(3, 5, 12, 14), Schwaerzung{Block: MinBlock}); err != nil { // 9×9
		t.Fatal(err)
	}
	nachher := d.Original()
	// Ein 9×9-Bereich wird ein einziger Block (Rest 1 verschmolzen): keine Reststruktur, eine Farbe. Die frühere
	// Restecke (11,13) blieb unverändert stehen.
	farbe := nachher.NRGBAAt(3, 5)
	for y := 5; y < 14; y++ {
		for x := 3; x < 12; x++ {
			if nachher.NRGBAAt(x, y) != farbe {
				t.Errorf("Pixel (%d,%d) = %v, Block hat %v", x, y, nachher.NRGBAAt(x, y), farbe)
			}
		}
	}
	if nachher.NRGBAAt(11, 13) == vorher.NRGBAAt(11, 13) {
		t.Error("Restecke (11,13) unverändert")
	}
	for name, s := range map[string]struct {
		r image.Rectangle
		s Schwaerzung
	}{
		"zu schmal":        {image.Rect(0, 0, 7, 20), Schwaerzung{Block: MinBlock}},
		"1×1":              {image.Rect(0, 0, 1, 1), Schwaerzung{Block: MinBlock}},
		"durchsichtig":     {image.Rect(0, 0, 10, 10), Schwaerzung{}},
		"halbdurchsichtig": {image.Rect(0, 0, 10, 10), Schwaerzung{Farbe: color.NRGBA{0, 0, 0, 128}}},
	} {
		if d.Schwaerzen(s.r, s.s) == nil {
			t.Errorf("%s: kein Fehler", name)
		}
	}
}

// F-62: nil, riesige Werte, Steuerzeichen und überlaufende Nummern werden abgewiesen; F-59: Pixelzahl begrenzt.
func TestWertebereiche(t *testing.T) {
	d := neu(t, 10, 10)
	if _, err := d.Hinzufuegen(nil); err == nil {
		t.Error("nil hinzugefügt")
	}
	if d.Ersetzen(nil) == nil {
		t.Error("nil ersetzt")
	}
	if _, err := d.BeginneNeuenZug(nil); err == nil {
		t.Error("nil-Zug begonnen")
	}
	id, _ := d.BeginneNeuenZug(Nummer{Radius: 3, Wert: 1})
	if d.ZugSetze(nil) == nil {
		t.Error("ZugSetze(nil) ohne Fehler")
	}
	d.BrecheZugAb()
	_ = id
	for _, o := range []Objekt{
		Kasten{Bereich: R(Pt(-1e308, 0), Pt(1e308, 1))},
		Kasten{Bereich: R(Pt(0, 0), Pt(1, 1)), Eckradius: 1e6},
		GeraderPfeil(Pt(0, 0), Pt(MaxKoordinate+1, 0), rot),
		Text{Ort: Pt(0, 0), Schriftgrad: 10, Inhalt: "a\x00b"},
		Text{Ort: Pt(0, 0), Schriftgrad: 10, Inhalt: "a\xffb"},
		Text{Ort: Pt(0, 0), Schriftgrad: MaxGroesse + 1, Inhalt: "x"},
		Nummer{Radius: 3, Wert: MaxNummer + 1},
		Nummer{Radius: 3, Wert: -1},
	} {
		if _, err := d.Hinzufuegen(o); err == nil {
			t.Errorf("%#v ohne Fehler", o)
		}
	}
	if _, err := d.Hinzufuegen(Text{Ort: Pt(0, 0), Schriftgrad: 10, Inhalt: "Zeile 1\n\tZeile 2 ÄÖÜ"}); err != nil {
		t.Errorf("gültiger Text abgelehnt: %v", err)
	}
	// Seiten je für sich erlaubt, Fläche zu groß.
	if _, err := Neu(image.NewNRGBA(image.Rect(0, 0, MaxSeite, MaxPixel/MaxSeite+1))); !errors.Is(err, ErrBildGroesse) {
		t.Errorf("Fläche über MaxPixel: %v", err)
	}
}
