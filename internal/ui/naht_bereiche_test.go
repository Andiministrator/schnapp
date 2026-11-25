// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"fmt"
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
	"github.com/Andiministrator/schnapp/internal/render"
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
		dt := time.Since(s)
		t.Logf("%s: zeichneObjekte %v", fall.name, dt)
		if editorAufbauZeit > 0 && dt > editorAufbauZeit {
			t.Errorf("%s: zeichneObjekte %v, Grenze %v", fall.name, dt, editorAufbauZeit)
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
	kaesten := func(breite float64) []document.Objekt {
		var k []document.Objekt
		for i := 0; i < 200; i++ {
			k = append(k, document.Kasten{Bereich: document.R(document.Pt(2, 2), document.Pt(250, 250)),
				Eckradius: 20, Strich: document.Strich{Farbe: schwarz, Breite: breite}})
		}
		return k
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
		budget  bool // render.Pruefe muss das Dokument annehmen (Fall knapp unter dem Budget)
	}{
		{"200 runde Kästen ohne Füllung, 4096 Kacheln", 256, 256, kacheln, kaesten(1), false},
		// F-782 (ADR-0023 Nachtrag 19): breiter Strich knapp unter KantenBudget (Strich 44 lehnt Pruefe ab); vorher
		// 1,6 bis 1,8 s je Bild mit llvmpipe, jeder Bereich spielte alle Kanten seiner Vierecke.
		{"200 runde Kästen, Strich 40, 4096 Kacheln", 256, 256, kacheln, kaesten(40), true},
		{"4 Texte, 1 lange und 399 kurze Zeilen, 1023 Spaltenstreifen", 2048, 1000, spalten(1023, 2), texte, false},
		{"50 Texte à 200 kurze Zeilen, 1023 Spaltenstreifen", 2048, 1000, spalten(1023, 2), kurze, false},
		{"5 Texte 6×60 @ Grad 74, 1023 Spaltenstreifen", 4200, 520, spalten(1023, 4), at, false},
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
		if err := render.Pruefe(d); fall.budget && err != nil {
			t.Fatalf("%s: %v", fall.name, err)
		}
		e := &editor{s: canvas.NeueSitzung(d)}
		headlessBild(t, d.Groesse(), func(gtx layout.Context) { e.zeichneObjekte(gtx) }) // Fläche, Zwischenspeicher
		s := time.Now()
		headlessBild(t, d.Groesse(), func(gtx layout.Context) { e.zeichneObjekte(gtx) })
		dt := time.Since(s)
		t.Logf("%s: ganzes Bild %v", fall.name, dt)
		if editorBildZeit > 0 && dt > editorBildZeit {
			t.Errorf("%s: ganzes Bild %v, Grenze %v", fall.name, dt, editorBildZeit)
		}
	}
}

// F-783 (ADR-0023 Nachtrag 19): Ist der halbe Strich breiter als der Eckradius, kehren sich die inneren Kanten der
// Gehrungsvierecke um. Gefüllt als Schleife hoben sich die Umlaufzahlen auf, und ein kleiner runder Kasten zeigte im
// Editor ein Loch, wo der Renderer füllt (Gios eigener Strich hatte es auch). Verglichen wird mit dem Renderer: Wo der
// Renderer ein Pixel samt allen Pixeln bis zwei Schritte weit gleich deckt (ganz oder gar nicht), muss der Editor
// dasselbe zeigen (rgbaNah); an den Kanten darf die Glättung abweichen (der Renderer addiert sie an den Gelenken seiner
// Kreise und Rechtecke, der Editor glättet die Gehrungsvierecke einfach), aber kein Pixel darf voll gegen leer stehen.
// Über Nähten (volle Bereiche als ein Rechteck, Hüllen je Bereich beschnitten; F-782) zeigt jedes Pixel der Ansicht,
// was der Editor ohne Naht an seiner Stelle im Original zeigt: Der Renderer zeichnet dort die Kante der Naht mit.
func TestEditorDickerStrichKleinerKastenWieRenderer(t *testing.T) {
	gruen := color.NRGBA{G: 160, A: 255}
	for _, fall := range []struct {
		seiteX, seiteY int
		radius, breite float64
	}{{40, 40, 20, 60}, {30, 30, 10, 50}, {20, 20, 10, 24}, {42, 42, 20, 60}, {200, 130, 8, 24}, {60, 50, 12, 20}} {
		min := image.Pt(80, 80)
		bereich := image.Rectangle{Min: min, Max: min.Add(image.Pt(fall.seiteX, fall.seiteY))}
		name := fmt.Sprintf("Kasten %d×%d Radius %v Strich %v", fall.seiteX, fall.seiteY, fall.radius, fall.breite)
		bild := func(st []document.Streifen) (*image.RGBA, *document.Dokument) {
			d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 360, 300)))
			if err != nil {
				t.Fatal(err)
			}
			if err := d.SetzeStreifen(st); err != nil {
				t.Fatal(err)
			}
			if _, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(float64(bereich.Min.X),
				float64(bereich.Min.Y)), document.Pt(float64(bereich.Max.X), float64(bereich.Max.Y))),
				Eckradius: fall.radius, Strich: document.Strich{Farbe: gruen, Breite: fall.breite}}); err != nil {
				t.Fatal(err)
			}
			e := &editor{s: canvas.NeueSitzung(d)}
			return headlessBild(t, d.Groesse(), func(gtx layout.Context) { e.zeichneObjekte(gtx) }), d
		}
		editor, d := bild(nil)
		rend, err := render.Bild(d)
		if err != nil {
			t.Fatal(err)
		}
		rgba := func(x, y int) color.RGBA { return color.RGBAModel.Convert(rend.At(x, y)).(color.RGBA) }
		innen, ab, loch, deckung := 0, 0, 0, 0
		for y := 2; y < 298; y++ {
			for x := 2; x < 358; x++ {
				ed, r := editor.RGBAAt(x, y), rgba(x, y)
				if r.A > 0 {
					deckung++
				}
				if (ed.A < 64 && r.A > 192) || (ed.A > 192 && r.A < 64) {
					loch++
				}
				gleich := r.A == 0 || r.A == 255
				for dy := -2; dy <= 2 && gleich; dy++ {
					for dx := -2; dx <= 2 && gleich; dx++ {
						gleich = rgba(x+dx, y+dy).A == r.A
					}
				}
				if gleich {
					innen++
					if !rgbaNah(ed, r) {
						ab++
					}
				}
			}
		}
		if deckung == 0 || innen == 0 {
			t.Fatalf("%s: Renderer hat nichts gezeichnet", name)
		}
		if ab != 0 || loch != 0 {
			m := bereich.Min.Add(bereich.Size().Div(2))
			t.Errorf("%s: %d von %d Pixeln abseits der Kanten weichen vom Renderer ab, %d stehen voll gegen leer "+
				"(Mitte: Editor %d, Renderer %d)", name, ab, innen, loch, editor.RGBAAt(m.X, m.Y).A, rgba(m.X, m.Y).A)
		}

		// Über Nähten: eine Spalten- und eine Zeilennaht durch die Mitte, links davon schmale Spalten alle 4 Pixel.
		m := bereich.Min.Add(bereich.Size().Div(2))
		st := []document.Streifen{{Von: m.X - 2, Bis: m.X + 2}, {Zeilen: true, Von: m.Y - 2, Bis: m.Y + 2}}
		imStreifenX := map[int]bool{}
		for x := m.X - 2; x < m.X+2; x++ {
			imStreifenX[x] = true
		}
		for x := 30; x < m.X-8; x += 4 {
			st = append(st, document.Streifen{Von: x, Bis: x + 1})
			imStreifenX[x] = true
		}
		var qx, qy []int // Spalte bzw. Zeile des Originals je Spalte bzw. Zeile der Ansicht
		for x := 0; x < 360; x++ {
			if !imStreifenX[x] {
				qx = append(qx, x)
			}
		}
		for y := 0; y < 300; y++ {
			if y < m.Y-2 || y >= m.Y+2 {
				qy = append(qy, y)
			}
		}
		mit, dm := bild(st)
		if g := dm.Groesse(); g != image.Pt(len(qx), len(qy)) {
			t.Fatalf("%s: Ansicht %v, erwartet %d×%d", name, g, len(qx), len(qy))
		}
		abNaht := 0
		for vy, y := range qy {
			for vx, x := range qx {
				if !rgbaNah(mit.RGBAAt(vx, vy), editor.RGBAAt(x, y)) {
					abNaht++
				}
			}
		}
		if abNaht != 0 {
			t.Errorf("%s: über den Nähten weichen %d Pixel vom Bild ohne Naht ab", name, abNaht)
		}
	}
}
