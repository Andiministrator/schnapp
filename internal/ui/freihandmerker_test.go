// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"math"
	"testing"
	"time"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
)

// F-1038 (Kritik #333 K-4): Der Editor merkt sich Zerlegung und Pfad eines Freihand-Strichs je Objekt.

// spiralStriche legt n Freihand-Striche zu je punkte Punkten an.
func spiralStriche(t *testing.T, d *document.Dokument, n, punkte int) []int {
	t.Helper()
	var ids []int
	for s := 0; s < n; s++ {
		pts := make([]document.Punkt, punkte)
		for i := range pts {
			a := float64(i) * 0.05
			pts[i] = document.Pt(600+(50+float64(i)*500/float64(punkte))*math.Cos(a+float64(s)),
				400+(30+float64(i)*350/float64(punkte))*math.Sin(a+float64(s)))
		}
		id, err := d.Hinzufuegen(document.NeuerFreihand(pts, document.Strich{Farbe: color.NRGBA{R: 200, A: 255}, Breite: 6}))
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	return ids
}

// Das zweite Bild ohne Änderung zerlegt nichts neu, baut seine Befehle unter freihandZweitesBildZeit und sieht genauso
// aus; ein geänderter Strich wird allein neu zerlegt.
func TestFreihandMerkerZweitesBild(t *testing.T) {
	i18n.Setze(nil)
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 1200, 800)))
	if err != nil {
		t.Fatal(err)
	}
	ids := spiralStriche(t, d, 20, 4096)
	e := &editor{s: canvas.NeueSitzung(d)}
	var erstes, zweites time.Duration
	a := headlessBild(t, image.Pt(1200, 800), func(gtx layout.Context) {
		t0 := time.Now()
		e.zeichneObjekte(gtx)
		erstes = time.Since(t0)
	})
	b := headlessBild(t, image.Pt(1200, 800), func(gtx layout.Context) {
		t0 := time.Now()
		e.zeichneObjekte(gtx)
		zweites = time.Since(t0)
	})
	t.Logf("Befehlsaufbau erstes Bild %v, zweites %v", erstes, zweites)
	if e.freihandNeu != 20 {
		t.Errorf("zwei Bilder: %d Zerlegungen, Soll 20", e.freihandNeu)
	}
	if freihandZweitesBildZeit > 0 && zweites > freihandZweitesBildZeit {
		t.Errorf("zweites Bild ohne Änderung: %v, Grenze %v", zweites, freihandZweitesBildZeit)
	}
	if a.Bounds() != b.Bounds() || string(a.Pix) != string(b.Pix) {
		t.Error("das zweite Bild aus dem Merker sieht anders aus als das erste")
	}
	o, _ := d.Objekt(ids[3])
	if err := d.Ersetzen(canvas.Verschiebe(o, document.Pt(5, 0))); err != nil {
		t.Fatal(err)
	}
	headlessBild(t, image.Pt(1200, 800), func(gtx layout.Context) { e.zeichneObjekte(gtx) })
	if e.freihandNeu != 21 {
		t.Errorf("ein geänderter Strich: %d Zerlegungen, Soll 21", e.freihandNeu)
	}
	// Das Ende jedes Bilds räumt Striche ab, die es nicht mehr gibt.
	if err := d.Entfernen(ids[0]); err != nil {
		t.Fatal(err)
	}
	headlessBild(t, image.Pt(1200, 800), func(gtx layout.Context) { e.zeichneObjekte(gtx) })
	if _, ok := e.freihand[freihandSchluessel{id: ids[0]}]; ok || len(e.freihand) != 19 {
		t.Errorf("nach dem Entfernen: %d Striche gemerkt, der entfernte noch da: %v", len(e.freihand), ok)
	}
}

// Die Bereiche zwischen den Nähten gelten nur für ihre Ansicht: Ein Streifen teilt den Strich in zwei Bereiche, ohne
// dass er neu zerlegt wird; Zuschnitt und Streifen zurück ergeben wieder einen.
func TestFreihandMerkerBereicheFolgenDerAnsicht(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 300, 100)))
	if err != nil {
		t.Fatal(err)
	}
	id, err := d.Hinzufuegen(document.NeuerFreihand([]document.Punkt{document.Pt(10, 50), document.Pt(150, 40),
		document.Pt(290, 50)}, document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 4}))
	if err != nil {
		t.Fatal(err)
	}
	e := &editor{s: canvas.NeueSitzung(d)}
	gtx := layout.Context{Ops: new(op.Ops)}
	zahl := func() int {
		o, _ := d.Objekt(id)
		return len(e.stueckeFuer(e.freihandFuer(gtx, o.(document.Freihand), false)))
	}
	if n := zahl(); n != 1 {
		t.Fatalf("ohne Naht: %d Bereiche", n)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Von: 100, Bis: 200}}); err != nil {
		t.Fatal(err)
	}
	if n := zahl(); n != 2 {
		t.Errorf("mit Streifen: %d Bereiche, Soll 2", n)
	}
	if err := d.SetzeStreifen(nil); err != nil {
		t.Fatal(err)
	}
	if n := zahl(); n != 1 {
		t.Errorf("Streifen zurück: %d Bereiche, Soll 1", n)
	}
	if err := d.SetzeZuschnitt(image.Rect(0, 0, 120, 100)); err != nil {
		t.Fatal(err)
	}
	o, _ := d.Objekt(id)
	b := e.freihandFuer(gtx, o.(document.Freihand), false)
	if st := e.stueckeFuer(b); len(st) != 1 || st[0].Quelle.Max.X > 120.5 {
		t.Errorf("Zuschnitt: %+v", st)
	}
	if e.freihandNeu != 1 {
		t.Errorf("Ansicht geändert: %d Zerlegungen, Soll 1", e.freihandNeu)
	}
}

// Über der Speichergrenze fallen nur Striche früherer Runden weg; Striche, die es nicht mehr gibt, räumt das Ende des
// Bilds ab; der Wechsel der Aufnahme leert den Merker.
func TestFreihandMerkerVerdraengtUndRaeumtAuf(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 1200, 800)))
	if err != nil {
		t.Fatal(err)
	}
	ids := spiralStriche(t, d, 4, 200)
	e := &editor{s: canvas.NeueSitzung(d)}
	gtx := layout.Context{Ops: new(op.Ops)}
	fuer := func(id int) *freihandBild {
		o, _ := d.Objekt(id)
		return e.freihandFuer(gtx, o.(document.Freihand), false)
	}
	eins := fuer(ids[0]).bytes
	if eins <= 0 || e.freihandBytes != eins {
		t.Fatalf("Bytes %d / %d", eins, e.freihandBytes)
	}
	alt := freihandSpeicher
	defer func() { freihandSpeicher = alt }()
	e.freihand, e.freihandBytes, e.freihandNeu = nil, 0, 0
	freihandSpeicher = 2*eins + eins/2 // Platz für zweieinhalb Striche
	for _, id := range ids {
		fuer(id)
	}
	e.freihandAufraeumen()
	if len(e.freihand) != 4 {
		t.Fatalf("eine Runde verdrängte ihre eigenen Striche: %d", len(e.freihand))
	}
	fuer(ids[0])
	fuer(ids[1])
	if err := d.Ersetzen(canvas.Verschiebe(func() document.Objekt { o, _ := d.Objekt(ids[2]); return o }(),
		document.Pt(3, 0))); err != nil {
		t.Fatal(err)
	}
	fuer(ids[2]) // geändert: neu, die nicht gezeichneten der vorigen Runde machen Platz
	if _, ok := e.freihand[freihandSchluessel{id: ids[3]}]; ok || len(e.freihand) != 3 || e.freihandBytes != 3*eins {
		t.Errorf("%d Striche, %d Bytes (Soll 3, %d)", len(e.freihand), e.freihandBytes, 3*eins)
	}
	if err := d.Entfernen(ids[0]); err != nil {
		t.Fatal(err)
	}
	e.freihandAufraeumen()
	if _, ok := e.freihand[freihandSchluessel{id: ids[0]}]; ok || e.freihandBytes != 2*eins {
		t.Errorf("entfernter Strich bleibt gemerkt: %d Bytes", e.freihandBytes)
	}

	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.e.freihand, l.e.freihandBytes = e.freihand, e.freihandBytes
	l.e.uebernimm(Geoeffnet{Titel: "neu.png", Dokument: freiTestDokument(t), Rettung: &leistenRettung{}})
	if l.e.freihand != nil || l.e.freihandBytes != 0 {
		t.Errorf("Wechsel der Aufnahme: %d Striche, %d Bytes gemerkt", len(l.e.freihand), l.e.freihandBytes)
	}
}
