// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"strings"
	"testing"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// Zeit je Editor-Bild über vielen Nähten (F-747, F-750, ADR-0023 Nachtrag 16): Gemessen wird nur der Aufbau der
// Zeichenbefehle ohne GPU. Vorher lief der Editor je Bereich über alle Zeilen (5,3 s je Bild bei 1024 Zeilenstreifen)
// und setzte jede Zeile je Bereich neu (149 ms bei 1023 Spaltenstreifen); jetzt rund 4 ms. Grenze 1 s auch unter -race
// auf einem langsamen Runner.
func TestEditorBildUeberVielenNaehten(t *testing.T) {
	schwarz := color.NRGBA{A: 255}
	for _, fall := range []struct {
		name   string
		w, h   int
		inhalt string
		grad   float64
	}{
		{"5 Texte je 9999 Zeilen, 1024 Zeilenstreifen", 64, 4096, "x" + strings.Repeat("\n", 9998) + "x", 8},
		{"5 Texte 6×60 @ Grad 74, 1023 Spaltenstreifen", 4200, 520,
			strings.TrimSuffix(strings.Repeat(strings.Repeat("@", 60)+"\n", 6), "\n"), 74},
	} {
		d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, fall.w, fall.h)))
		if err != nil {
			t.Fatal(err)
		}
		var st []document.Streifen
		if fall.w == 64 {
			for i := 0; i < document.MaxStreifen; i++ {
				st = append(st, document.Streifen{Zeilen: true, Von: 2 * i, Bis: 2*i + 1})
			}
		} else {
			for i := 0; i < 1023; i++ {
				st = append(st, document.Streifen{Von: 4 * i, Bis: 4*i + 1})
			}
		}
		if err := d.SetzeStreifen(st); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 5; i++ {
			if _, err := d.Hinzufuegen(document.Text{Ort: document.Pt(4, 5), Inhalt: fall.inhalt, Schriftgrad: fall.grad,
				Farbe: schwarz}); err != nil {
				t.Fatal(err)
			}
		}
		e := &editor{s: canvas.NeueSitzung(d)}
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Constraints: layout.Exact(d.Groesse())}
		e.zeichneObjekte(gtx) // Zwischenspeicher (Tinte, Umrisse) füllen
		ops.Reset()
		s := time.Now()
		e.zeichneObjekte(gtx)
		if dt := time.Since(s); dt > time.Second {
			t.Errorf("%s: zeichneObjekte %v, Grenze 1 s", fall.name, dt)
		}
	}
}

// F-765 (ADR-0023 Nachtrag 18): Den Strich eines runden Kastens zeichnet der Editor aus Vierecken je Strecke des Umrisses
// (document.RundesRechteck) statt mit dem Strich von Gio. Ohne Naht sieht er aus wie Gios Strich um dasselbe abgerundete
// Rechteck: keine Lücken an den Gelenken, keine Ausbuchtungen. Verglichen wird auf ganzzahligem Rechteck und Radius,
// wo Gio genau dasselbe rundet; Kantenglättung darf abweichen, nicht ganze Pixel.
func TestEditorRunderKastenStrichWieGio(t *testing.T) {
	gruen := color.NRGBA{G: 160, A: 255}
	for _, fall := range []struct{ breite, radius float64 }{{1, 50}, {4, 50}, {13, 50}, {24, 8}, {40, 30}} {
		breite, radius := fall.breite, fall.radius
		bereich := image.Rect(30, 20, 230, 150)
		d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 260, 170)))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(30, 20), document.Pt(230, 150)),
			Eckradius: radius, Strich: document.Strich{Farbe: gruen, Breite: breite}}); err != nil {
			t.Fatal(err)
		}
		e := &editor{s: canvas.NeueSitzung(d)}
		editor := headlessBild(t, d.Groesse(), func(gtx layout.Context) { e.zeichneObjekte(gtx) })
		gio := headlessBild(t, d.Groesse(), func(gtx layout.Context) {
			rr := clip.UniformRRect(bereich, int(radius))
			paint.FillShape(gtx.Ops, gruen, clip.Stroke{Path: rr.Path(gtx.Ops), Width: float32(breite)}.Op())
		})
		ab, deckung := 0, 0
		for y := 0; y < 170; y++ {
			for x := 0; x < 260; x++ {
				a, b := editor.RGBAAt(x, y), gio.RGBAAt(x, y)
				if b.A > 0 {
					deckung++
				}
				if int(a.A)-int(b.A) > 96 || int(b.A)-int(a.A) > 96 {
					ab++
				}
			}
		}
		if deckung == 0 {
			t.Fatalf("Breite %v: Gio hat nichts gezeichnet", breite)
		}
		if ab != 0 {
			t.Errorf("Breite %v: %d Pixel weichen um mehr als 96 Stufen Deckung von Gios Strich ab (%d mit Deckung)", breite,
				ab, deckung)
		}
	}
}

// F-765 (ADR-0023 Nachtrag 18): Text über vielen schmalen Spalten. Der Editor zeichnet jede Zeile in Läufen von Glyphen
// und spielt jeden Lauf nur in den Spalten, die seine Tinte berührt. Jede Spalte zeigt trotzdem genau das, was ohne
// Streifen an ihrer Stelle im Original steht: Spalte für Spalte verglichen, auch an den Grenzen der Läufe und am Ende
// jeder Zeile (die Tinte der letzten Glyphe reicht über ihren Ursprung hinaus).
func TestEditorTextUeberSchmalenSpalten(t *testing.T) {
	const w, h, periode = 640, 150, 5
	schrift := color.NRGBA{R: 200, A: 255}
	tx := document.Text{Ort: document.Pt(7, 12), Inhalt: "WAWAVWWMMWWAW mmm WWW\nWm\nAVAVAWWWWMW", Schriftgrad: 30,
		Farbe: schrift}
	neu := func(st []document.Streifen) *image.RGBA {
		d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, w, h)))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := d.Hinzufuegen(tx); err != nil {
			t.Fatal(err)
		}
		if st != nil {
			if err := d.SetzeStreifen(st); err != nil {
				t.Fatal(err)
			}
		}
		e := &editor{s: canvas.NeueSitzung(d)}
		return headlessBild(t, image.Pt(w, h), func(gtx layout.Context) { e.zeichneObjekte(gtx) })
	}
	ohne := neu(nil)
	var st []document.Streifen
	var quelle []int // Spalte des Originals je Spalte der Ansicht
	for x := 0; x < w; x++ {
		if x%periode == 0 {
			st = append(st, document.Streifen{Von: x, Bis: x + 1})
		} else {
			quelle = append(quelle, x)
		}
	}
	mit := neu(st)
	ab, tinte := 0, 0
	for vx, qx := range quelle {
		for y := 0; y < h; y++ {
			if !rgbaNah(mit.RGBAAt(vx, y), ohne.RGBAAt(qx, y)) {
				ab++
			}
			if mit.RGBAAt(vx, y).A > 0 {
				tinte++
			}
		}
	}
	if tinte == 0 {
		t.Fatal("keine Schrift gezeichnet")
	}
	if ab != 0 {
		t.Errorf("%d Pixel weichen von der Spalte des Originals ab", ab)
	}
}

// F-765 (ADR-0023 Nachtrag 18): Der Befehlsaufbau allein sagt nichts über die Zeit der GPU: Der Editor spielte einen
// runden Kasten ohne Füllung in jedem Bereich seines Rechtecks ab und jede Textzeile in allen Spalten rechts ihres
// Anfangs. 200 runde Kästen über 4096 Kacheln brauchten so 21,6 s je Bild mit llvmpipe. Gemessen wird deshalb ein
// ganzes Bild über die Zeichenfläche ohne Fenster (Aufbau, GPU, Auslesen; die Fläche wird vorher einmal angelegt und
// die Zwischenspeicher gefüllt). Die Grenze 2 s ist großzügig für llvmpipe auf einem langsamen Runner.
func TestEditorGanzesBildUeberVielenNaehten(t *testing.T) {
	schwarz := color.NRGBA{A: 255}
	spalten := func(n, abstand int) []document.Streifen {
		var st []document.Streifen
		for i := 0; i < n; i++ {
			st = append(st, document.Streifen{Von: abstand * i, Bis: abstand*i + 1})
		}
		return st
	}
	var kacheln []document.Streifen
	for i := 0; i < 63; i++ {
		kacheln = append(kacheln, document.Streifen{Von: 4 * i, Bis: 4*i + 1},
			document.Streifen{Zeilen: true, Von: 4 * i, Bis: 4*i + 1})
	}
	var kaesten []document.Objekt
	for i := 0; i < 200; i++ {
		kaesten = append(kaesten, document.Kasten{Bereich: document.R(document.Pt(2, 2), document.Pt(250, 250)),
			Eckradius: 20, Strich: document.Strich{Farbe: schwarz, Breite: 1}})
	}
	langKurz := strings.Repeat("x", 1700) + strings.Repeat("\nx", 399)
	var texte []document.Objekt
	for i := 0; i < 4; i++ {
		texte = append(texte, document.Text{Ort: document.Pt(4, 2), Inhalt: langKurz, Schriftgrad: 2, Farbe: schwarz})
	}
	kurz := strings.TrimSuffix(strings.Repeat("x\n", 200), "\n")
	var kurze []document.Objekt
	for i := 0; i < 50; i++ {
		kurze = append(kurze, document.Text{Ort: document.Pt(4, 2), Inhalt: kurz, Schriftgrad: 4, Farbe: schwarz})
	}
	lang := strings.TrimSuffix(strings.Repeat(strings.Repeat("@", 60)+"\n", 6), "\n")
	var at []document.Objekt
	for i := 0; i < 5; i++ {
		at = append(at, document.Text{Ort: document.Pt(4, 5), Inhalt: lang, Schriftgrad: 74, Farbe: schwarz})
	}
	for _, fall := range []struct {
		name    string
		w, h    int
		st      []document.Streifen
		objekte []document.Objekt
	}{
		{"200 runde Kästen ohne Füllung, 4096 Kacheln", 256, 256, kacheln, kaesten},
		{"4 Texte, 1 lange und 399 kurze Zeilen, 1023 Spaltenstreifen", 2048, 1000, spalten(1023, 2), texte},
		{"50 Texte à 200 kurze Zeilen, 1023 Spaltenstreifen", 2048, 1000, spalten(1023, 2), kurze},
		{"5 Texte 6×60 @ Grad 74, 1023 Spaltenstreifen", 4200, 520, spalten(1023, 4), at},
	} {
		d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, fall.w, fall.h)))
		if err != nil {
			t.Fatal(err)
		}
		if err := d.SetzeStreifen(fall.st); err != nil {
			t.Fatal(err)
		}
		for _, o := range fall.objekte {
			if _, err := d.Hinzufuegen(o); err != nil {
				t.Fatal(err)
			}
		}
		e := &editor{s: canvas.NeueSitzung(d)}
		headlessBild(t, d.Groesse(), func(gtx layout.Context) { e.zeichneObjekte(gtx) }) // Fläche, Zwischenspeicher
		s := time.Now()
		headlessBild(t, d.Groesse(), func(gtx layout.Context) { e.zeichneObjekte(gtx) })
		dt := time.Since(s)
		t.Logf("%s: ganzes Bild %v", fall.name, dt)
		if dt > 2*time.Second {
			t.Errorf("%s: ganzes Bild %v, Grenze 2 s", fall.name, dt)
		}
	}
}
