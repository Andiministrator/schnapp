// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"errors"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/portal"
	"github.com/Andiministrator/schnapp/internal/render"
)

// Kritik Karte #193 — Erkundung.

// K193: Bei Zoom 4 bleibt jedes Zielpixel ein scharfes Quadrat (nächster Nachbar, ADR-0033 Punkt 3). Der Test der
// Karte prüft nur Zoom 1; dort unterscheiden sich FilterNearest und FilterLinear nicht.
func TestK193EditorZoomNaechsterNachbar(t *testing.T) {
	e, l := lupenEditor(t, document.Strich{Farbe: color.NRGBA{G: 255, A: 255}})
	const zoom = 4
	b := headlessBild(t, image.Pt(40*zoom, 30*zoom), func(gtx layout.Context) {
		defer op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(zoom, zoom))).Push(gtx.Ops).Pop()
		e.zeichneObjekte(gtx)
	})
	soll := render.LupenInhalt(e.s.D, l)
	falsch := 0
	for y := 10; y < 16; y++ {
		for x := 20; x < 28; x++ {
			for dy := 0; dy < zoom; dy++ {
				for dx := 0; dx < zoom; dx++ {
					if p := b.RGBAAt(x*zoom+dx, y*zoom+dy); !nahFarbe(p, soll.NRGBAAt(x, y)) {
						falsch++
					}
				}
			}
		}
	}
	if falsch > 0 {
		t.Errorf("Zoom %d: %d Fensterpixel im Ziel weichen vom Quellpixel ab (geglättet?)", zoom, falsch)
	}
}

// K193: Im laufenden Zug (Stand unverändert) folgt das Bild dem verschobenen Ziel; nach Schwärzen, Rückgängig und
// Wiederholen zeigt der Editor jeweils den Stand des Originals.
func TestK193EditorMerkerFolgt(t *testing.T) {
	e, l := lupenEditor(t, document.Strich{Farbe: color.NRGBA{G: 255, A: 255}})
	zeichne := func() *image.RGBA {
		return headlessBild(t, image.Pt(64, 48), func(gtx layout.Context) { e.zeichneObjekte(gtx) })
	}
	b := zeichne()
	klar := b.RGBAAt(21, 11)
	// Zug: Ziel um (5, 3) verschieben, ohne den Zug zu beenden.
	stand := e.s.D.Stand()
	if err := e.s.D.BeginneZug(l.ID); err != nil {
		t.Fatal(err)
	}
	m := l
	m.Ziel = document.R(document.Pt(25, 13), document.Pt(33, 19))
	if err := e.s.D.ZugSetze(m); err != nil {
		t.Fatal(err)
	}
	if e.s.D.Stand() != stand {
		t.Fatal("Stand ändert sich im Zug")
	}
	b = zeichne()
	if p := b.RGBAAt(26, 14); p != klar {
		t.Errorf("Zug: Bild folgt dem Ziel nicht: (26,14) %v, Soll %v", p, klar)
	}
	if p := b.RGBAAt(21, 11); p.A != 0 {
		t.Errorf("Zug: altes Ziel noch gezeichnet: %v", p)
	}
	if err := e.s.D.BeendeZug(); err != nil {
		t.Fatal(err)
	}
	schwarz := color.NRGBA{A: 255}
	if err := e.s.D.Schwaerzen(image.Rect(2, 3, 4, 5), document.Schwaerzung{Farbe: schwarz}); err != nil {
		t.Fatal(err)
	}
	if p := zeichne().RGBAAt(26, 14); !nahFarbe(p, schwarz) {
		t.Errorf("geschwärzt: %v", p)
	}
	if err := e.s.D.Rueckgaengig(); err != nil {
		t.Fatal(err)
	}
	if p := zeichne().RGBAAt(26, 14); p != klar {
		t.Errorf("Rückgängig: %v, Soll %v", p, klar)
	}
	if err := e.s.D.Wiederholen(); err != nil {
		t.Fatal(err)
	}
	if p := zeichne().RGBAAt(26, 14); !nahFarbe(p, schwarz) {
		t.Errorf("Wiederholen: %v, Soll schwarz (kein Klartext aus dem Merker)", p)
	}
}

// K193-1 (ADR-0033 Nachtrag 2 Punkt 3): Der Export aus dem Editor prüft das Rechenbudget vor dem Kodieren. Lupen, deren
// Quellen zusammen das Budget überschreiten, werden nicht exportiert; die Leiste sagt es, kein Ziel entsteht.
func TestK193EditorExportPrueftBudget(t *testing.T) {
	i18n.Setze(nil)
	fehlerAbfangen(t)
	const w, h = 2000, 1000
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, w, h)))
	if err != nil {
		t.Fatal(err)
	}
	n := int(render.Budget/(25*w*h)) + 1 // 25: Gewicht je Pixel einer Füllung
	for i := 0; i < n; i++ {
		if _, err := d.Hinzufuegen(document.Lupe{Quelle: document.R(document.Pt(0, 0), document.Pt(w, h)),
			Ziel: document.R(document.Pt(1, 1), document.Pt(2, 2)), Strich: document.Strich{Farbe: color.NRGBA{A: 255}}}); err != nil {
			t.Fatal(err)
		}
	}
	e := &editor{s: canvas.NeueSitzung(d)}
	for _, name := range []string{"export.svg", "export.png"} {
		ziel := filepath.Join(t.TempDir(), name)
		e.exportFertig(exportErgebnis{ergebnis: portal.Ergebnis{URI: "file://" + ziel}})
		if _, err := os.Lstat(ziel); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s: trotz Budget geschrieben (%v)", name, err)
		}
		if !strings.Contains(e.meldung.Text, "too many or too large") {
			t.Errorf("%s: Leiste zeigt %q", name, e.meldung.Text)
		}
	}
}

// K193-4: Der Merker der Lupenbilder wird nie geleert: Bilder gelöschter Lupen bleiben im Speicher.
func TestK193EditorMerkerWaechst(t *testing.T) {
	e, _ := lupenEditor(t, document.Strich{Farbe: color.NRGBA{G: 255, A: 255}})
	zeichne := func() *image.RGBA {
		return headlessBild(t, image.Pt(64, 48), func(gtx layout.Context) { e.zeichneObjekte(gtx) })
	}
	for i := 0; i < 5; i++ {
		id, err := e.s.D.Hinzufuegen(document.Lupe{Quelle: document.R(document.Pt(2, 3), document.Pt(6, 6)),
			Ziel: document.R(document.Pt(20, 10), document.Pt(28, 16)), Strich: document.Strich{Farbe: color.NRGBA{G: 255, A: 255}}})
		if err != nil {
			t.Fatal(err)
		}
		zeichne()
		if err := e.s.D.Entfernen(id); err != nil {
			t.Fatal(err)
		}
		zeichne()
	}
	if n, m := len(e.lupe.bilder), len(e.s.D.Objekte()); n > m {
		t.Errorf("Merker hält %d Bilder bei %d Lupen im Dokument (gelöschte Lupen bleiben im Speicher)", n, m)
	}
}
