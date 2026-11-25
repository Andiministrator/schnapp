// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"testing"

	"gioui.org/io/key"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0031 Punkt 3, Nachtrag 3: Strg+Umschalt+B und der Menüeintrag klappen die rechte Leiste ein und aus; die
// Leinwand wird um ihre Breite schmaler. In einem schmalen Fenster bleibt sie eingeklappt, die Einstellung bleibt.
func TestSeitenleisteEinUndAus(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	breit := image.Rect(werkzeugeDp, menuezeileDp, 1200-seiteDp, 800-fusszeileDp)
	if f := l.e.leinwandFlaeche(l.gtx()); f != breit || l.e.seite.kopf == "" {
		t.Fatalf("Leiste sichtbar: Leinwand %v, soll %v; Kopf %q", f, breit, l.e.seite.kopf)
	}
	l.taste("B", key.ModShortcut|key.ModShift)
	if f := l.e.leinwandFlaeche(l.gtx()); !l.e.seiteZu || f.Max.X != 1200 || l.e.seite.kopf != "" {
		t.Fatalf("nach Strg+Umschalt+B: zu %v, Leinwand %v, Kopf %q", l.e.seiteZu, f, l.e.seite.kopf)
	}
	if l.e.s.Werkzeug == canvas.WerkzeugSchwaerzen {
		t.Error("Strg+Umschalt+B hat das Werkzeug Schwärzen (B) gewählt")
	}
	l.e.fuehreAus(l.gtx(), aktSeitenleiste, true)
	l.setle()
	if l.e.seiteZu {
		t.Error("Menüeintrag „Seitenleiste“ klappt nicht wieder auf")
	}
	l.fenst = image.Pt(werkzeugeDp+seiteDp+minLeinwandDp-1, 600)
	l.setle()
	if f := l.e.leinwandFlaeche(l.gtx()); f.Max.X != l.fenst.X || l.e.seite.kopf != "" || l.e.seiteZu {
		t.Errorf("schmales Fenster: Leinwand %v, Kopf %q, zu %v", f, l.e.seite.kopf, l.e.seiteZu)
	}
}

// Kopf, Farbe und Kantenform der Leiste: Ein gewählter Text heißt „Auswahl: Text“, ein Klick auf eine Farbe färbt ihn
// als ein Rückgängig-Schritt um; beim Stauchen mit Naht setzen die Knöpfe die Kantenform.
func TestSeitenleisteFarbeUndKante(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	if l.e.seite.kopf != "SELECTION: TEXT" || !l.e.seite.farbeGezeigt {
		t.Fatalf("Kopf %q, Farbe gezeigt %v", l.e.seite.kopf, l.e.seite.farbeGezeigt)
	}
	vor := l.e.s.D.Stand()
	r := l.e.seite.farbRahmen[1]
	l.klick(r.Min.Add(r.Max).Div(2))
	o, _ := l.e.s.D.Objekt(l.id)
	if l.e.s.AktuelleFarbe() != canvas.Beschriftungsfarben[1] || o.(document.Text).Hintergrund != canvas.Beschriftungsfarben[1] {
		t.Errorf("Klick auf Blau: aktuell %v, Text %v", l.e.s.AktuelleFarbe(), o.(document.Text).Hintergrund)
	}
	if l.e.s.D.Stand() == vor {
		t.Error("Umfärben ist kein Rückgängig-Schritt")
	}

	l.e.s.Auswahl = 0
	l.taste("X", 0) // Stauchen
	if l.e.seite.farbeGezeigt || l.e.seite.kantRahmen[0].Empty() {
		t.Fatalf("Stauchen: Farbe gezeigt %v, Kantenknöpfe %v", l.e.seite.farbeGezeigt, l.e.seite.kantRahmen)
	}
	r = l.e.seite.kantRahmen[1]
	l.klick(r.Min.Add(r.Max).Div(2))
	if l.e.s.D.Kante() != document.KanteAbriss {
		t.Error("Kantenform ohne Naht geändert")
	}
	// Graue Kantenknöpfe bleiben fokussierbar; Enter ändert ohne Naht nichts (Verify 2, Fund 5).
	l.fokusiere(&l.e.seite.kanten[2])
	if !l.fokus(&l.e.seite.kanten[2]) {
		t.Error("grauer Kantenknopf nicht fokussierbar")
	}
	l.taste(key.NameReturn, 0)
	if l.e.s.D.Kante() != document.KanteAbriss {
		t.Error("Enter auf grauem Kantenknopf hat die Form geändert")
	}
	if err := l.e.s.D.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 10, Bis: 12}}); err != nil {
		t.Fatal(err)
	}
	l.setle()
	l.klick(r.Min.Add(r.Max).Div(2))
	if l.e.s.D.Kante() != document.KanteWelle || l.e.meldung.Text != kantenMeldung(document.KanteWelle) {
		t.Errorf("Knopf Welle: Kante %v, Meldung %q", l.e.s.D.Kante(), l.e.meldung.Text)
	}
}

// Die Listen der Fußzeile per Tastatur: Enter auf dem Feld öffnet die Liste mit Fokus auf dem ersten Eintrag, Pfeile
// links und rechts wechseln nicht in die Menüzeile, Esc gibt den Fokus an das Feld zurück, Enter auf einem Eintrag
// setzt die Farbe.
func TestFussListePerTastatur(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.e.s.Auswahl = 0
	l.fokusiere(&l.e.knoepfe.fussFarbe)
	l.taste(key.NameReturn, 0)
	if !l.e.menue.offen || l.e.menue.welches != listeFarbe || menueFokusNr(l) != 0 {
		t.Fatalf("Enter auf dem Farbfeld: offen %v (%d), Fokus %s", l.e.menue.offen, l.e.menue.welches, menueFokusName(l))
	}
	l.taste(key.NameRightArrow, 0)
	if !l.e.menue.offen || l.e.menue.welches != listeFarbe {
		t.Errorf("Pfeil rechts verlässt die Liste: offen %v (%d)", l.e.menue.offen, l.e.menue.welches)
	}
	l.taste(key.NameEscape, 0)
	if l.e.menue.offen || !l.fokus(&l.e.knoepfe.fussFarbe) {
		t.Errorf("Esc: offen %v, Fokus %s", l.e.menue.offen, menueFokusName(l))
	}
	l.taste(key.NameReturn, 0)
	l.taste(key.NameEnd, 0)
	l.taste(key.NameReturn, 0)
	if l.e.menue.offen || l.e.s.Stil.Farbe != canvas.Beschriftungsfarben[len(canvas.Beschriftungsfarben)-1] {
		t.Errorf("Ende und Enter: offen %v, Farbe %v", l.e.menue.offen, l.e.s.Stil.Farbe)
	}

	l.fokusiere(&l.e.knoepfe.fussZoom)
	l.taste(key.NameReturn, 0)
	l.taste(key.NameDownArrow, 0) // 25 %
	l.taste(key.NameReturn, 0)
	if l.e.s.Abb.Zoom != 0.25 {
		t.Errorf("Zoomliste per Tastatur: Zoom %v, soll 0,25", l.e.s.Abb.Zoom)
	}

	// Das Farbfeld der Werkzeugleiste öffnet per Tastatur die Liste an sich selbst; Esc gibt ihm den Fokus zurück.
	l.fokusiere(&l.e.knoepfe.farbe)
	l.taste(key.NameReturn, 0)
	if !l.e.menue.offen || l.e.menue.welches != listeFarbe || l.e.listenAnker(listeFarbe) != l.e.farbKnopfRahmen {
		t.Fatalf("Enter am Farbfeld links: offen %v, Anker %v", l.e.menue.offen, l.e.listenAnker(listeFarbe))
	}
	l.taste(key.NameEscape, 0)
	if !l.fokus(&l.e.knoepfe.farbe) {
		t.Errorf("Esc nach der Liste am Farbfeld links: Fokus %s", menueFokusName(l))
	}
}

// F-516: Im kleinsten Fenster (480×320) liegen alle Werkzeugknöpfe und das Farbfeld zwischen Menüzeile und Fußzeile,
// ohne sich zu überlappen; in einem hohen Fenster bleibt die Leiste wie bisher (Knöpfe 40 dp).
func TestWerkzeugleistePasstInKleinesFenster(t *testing.T) {
	for _, gr := range []image.Point{{480, 320}, {1600, 1000}} {
		l := neuerMenueLauf(t, gr)
		ws := l.e.werkzeugKnoepfe()
		knoepfe, farbe := werkzeugRahmen(l.gtx(), ws)
		alle := append(knoepfe, farbe)
		for i, r := range alle {
			if r.Min.Y < menuezeileDp || r.Max.Y > gr.Y-fusszeileDp {
				t.Errorf("%v: Knopf %d bei %v außerhalb der Leiste", gr, i, r)
			}
			if i > 0 && r.Min.Y < alle[i-1].Max.Y {
				t.Errorf("%v: Knopf %d überlappt den vorigen (%v, %v)", gr, i, alle[i-1], r)
			}
		}
		if gr.Y == 1000 && (knoepfe[0].Dx() != werkzeugKnopfDp || knoepfe[0].Min.Y != menuezeileDp+8) {
			t.Errorf("hohes Fenster: erster Knopf %v, soll 40 dp ab 38", knoepfe[0])
		}
	}
}

// ADR-0021 Nachtrag 7: − und + der Strichstärke gehen von Stufe zu Stufe, beim Werkzeug Pfeil für neue Pfeile, bei
// einem gewählten Kasten für ihn (ein Rückgängig-Schritt); ohne Strich (Text) fehlt die Zeile.
func TestSeitenleisteStrichstaerke(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	if !l.e.seite.strRahmen[0].Empty() {
		t.Fatal("Strichstärke beim gewählten Text sichtbar")
	}
	l.e.s.Auswahl = 0
	l.taste("A", 0)
	plus := l.e.seite.strRahmen[1]
	if plus.Empty() {
		t.Fatal("keine Strichstärke beim Pfeil")
	}
	l.klick(plus.Min.Add(plus.Max).Div(2))
	if l.e.s.Stil.Breite != 8 {
		t.Errorf("+ beim Pfeil: %v, soll 8", l.e.s.Stil.Breite)
	}
	id, err := l.e.s.D.Hinzufuegen(document.Kasten{Bereich: document.Rechteck{Min: document.Pt(5, 5), Max: document.Pt(40, 30)},
		Strich: document.Strich{Farbe: canvas.Beschriftungsfarben[0], Breite: 8}})
	if err != nil {
		t.Fatal(err)
	}
	l.e.s.Auswahl = id
	l.setle()
	minus := l.e.seite.strRahmen[0]
	vor := l.e.s.D.Stand()
	l.klick(minus.Min.Add(minus.Max).Div(2))
	o, _ := l.e.s.D.Objekt(id)
	if o.(document.Kasten).Strich.Breite != 6 || l.e.s.D.Stand() == vor {
		t.Errorf("− am gewählten Kasten: %v", o.(document.Kasten).Strich.Breite)
	}
}
