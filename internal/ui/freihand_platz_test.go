// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"fmt"
	"image"
	"testing"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
)

// F-1051: Ist das Bild an der Gesamtgrenze der Freihand-Punkte, sagt die Fußzeile es beim Zeichnen und beim Einfügen,
// statt still nichts zu tun oder erst beim Speichern zu scheitern.
func TestFreihandGesamtgrenzeMeldung(t *testing.T) {
	e := testEditor(t)
	pts := make([]document.Punkt, document.MaxFreihandPunkte)
	for i := range pts {
		pts[i] = document.Pt(float64(i%30), float64(i%2))
	}
	strich := document.NeuerFreihand(pts, document.Strich{Breite: 2})
	var os []document.Objekt
	for range document.MaxFreihandGesamt / document.MaxFreihandPunkte {
		os = append(os, strich)
	}
	if _, err := e.s.D.HinzufuegenMehrere(os); err != nil {
		t.Fatal(err)
	}
	n := len(e.s.D.Objekte())

	q, err := document.OhnePixel(image.Pt(40, 30), image.Rect(0, 0, 40, 30), nil, document.KanteAbriss)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.Hinzufuegen(document.NeuerFreihand(pts[:2], document.Strich{Breite: 2})); err != nil {
		t.Fatal(err)
	}
	e.einfuegenAnwenden(q)
	soll := fmt.Sprintf(i18n.T("Could not paste: freehand strokes in this image would exceed %d points."),
		document.MaxFreihandGesamt)
	if len(e.s.D.Objekte()) != n || e.meldung.Text != soll {
		t.Errorf("Einfügen: %d Objekte (vorher %d), Meldung %q", len(e.s.D.Objekte()), n, e.meldung.Text)
	}

	if g := e.grenzText(canvas.AblehnungFreihand); g == "" {
		t.Error("kein Grenztext für AblehnungFreihand")
	}
	e.s.SetzeWerkzeug(canvas.WerkzeugFreihand)
	if h := e.hinweis(); h != e.grenzText(canvas.AblehnungFreihand) { // F-1056: kein Strich versprochen
		t.Errorf("Hinweis Freihand: %q", h)
	}
	e.s.SetzeWerkzeug(canvas.WerkzeugTextmarker)
	if !e.s.TextmarkerFrei() {
		e.s.SchalteTextmarkerFrei()
	}
	if h := e.hinweis(); h != e.grenzText(canvas.AblehnungFreihand) {
		t.Errorf("Hinweis freier Textmarker: %q", h)
	}
	e.s.SchalteTextmarkerFrei()
	if h := e.hinweis(); h == e.grenzText(canvas.AblehnungFreihand) {
		t.Error("gerader Textmarker meldet die Freihand-Grenze")
	}
	e.s.SetzeWerkzeug(canvas.WerkzeugAuswahl)
	if !e.s.WaehleAlle() {
		t.Fatal("nichts gewählt")
	}
	e.meldung.Text = ""
	e.doppeln() // F-1056: Meldung statt still
	if e.meldung.Text != e.grenzText(canvas.AblehnungFreihand) {
		t.Errorf("Doppeln an der Grenze: Meldung %q", e.meldung.Text)
	}
	e.s.SetzeWerkzeug(canvas.WerkzeugFreihand)
	e.s.SetzeFenster(image.Pt(88, 78))
	e.s.Druecken(document.Pt(30, 30), canvas.TastePrimaer)
	e.s.Loslassen(document.Pt(40, 40))
	e.meldeAbgelehnt()
	if e.meldung.Text != e.grenzText(canvas.AblehnungFreihand) || len(e.s.D.Objekte()) != n {
		t.Errorf("Zeichnen: Meldung %q, %d Objekte", e.meldung.Text, len(e.s.D.Objekte()))
	}
}
