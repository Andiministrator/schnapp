// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/io/key"

	"github.com/Andiministrator/schnapp/internal/canvas"
)

// Farbpaar in Werkzeugleiste und rechter Leiste, Tauschen per Knopf, Menü und Strg+Umschalt+X (ADR-0021 Nachtrag 11).

// TestFarbpaarWerkzeugleiste: Ein Klick auf das Feld Füllung macht es aktiv, beim Pfeil (ohne Füllung) nicht; der Knopf
// ⇄ tauscht die Farben des gewählten Texts als einen Rückgängig-Schritt.
func TestFarbpaarWerkzeugleiste(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	_, farbe := werkzeugRahmen(l.gtx(), l.e.werkzeugKnoepfe())
	vorn, fuel, tausch := farbpaarRahmen(farbe)
	if vorn.Overlaps(fuel) || vorn.Overlaps(tausch) || fuel.Overlaps(tausch) || !tausch.In(farbe) {
		t.Fatalf("Lage: vorn %v, Füllung %v, Tauschen %v in %v", vorn, fuel, tausch, farbe)
	}
	l.klick(mittenPunkt(vorn)) // zeigt den Farbwähler der Leiste, keine Liste (F-817)
	if l.e.s.AktivesFeld() != canvas.FeldVordergrund || l.e.menue.offen {
		t.Fatalf("Klick auf Vordergrund: Feld %v, Liste offen %v", l.e.s.AktivesFeld(), l.e.menue.offen)
	}
	l.klick(mittenPunkt(fuel))
	if l.e.s.AktivesFeld() != canvas.FeldFuellung {
		t.Fatalf("Klick auf Füllung: Feld %v", l.e.s.AktivesFeld())
	}
	alt, vor := l.text(), l.e.s.D.Stand()
	l.klick(mittenPunkt(tausch))
	if tx := l.text(); tx.Farbe != alt.Hintergrund || tx.Hintergrund != alt.Farbe {
		t.Errorf("nach ⇄: Schrift %v, Schild %v", tx.Farbe, tx.Hintergrund)
	}
	if n := schritteSeit(l, vor); n != 1 {
		t.Errorf("%d Schritte, erwartet 1", n)
	}
	// Pfeil ohne Auswahl: Füllung grau, Klick wirkungslos, Tauschen aus.
	l.e.s.Auswahl = 0
	l.e.s.SetzeWerkzeug(canvas.WerkzeugPfeil)
	l.setle()
	l.klick(mittenPunkt(fuel))
	if l.e.s.AktivesFeld() != canvas.FeldVordergrund || l.e.menue.offen {
		t.Errorf("Füllung beim Pfeil: Feld %v, Liste offen %v", l.e.s.AktivesFeld(), l.e.menue.offen)
	}
	stil := l.e.s.Stil
	l.klick(mittenPunkt(tausch))
	if l.e.s.Stil != stil {
		t.Errorf("⇄ beim Pfeil änderte den Stil: %+v", l.e.s.Stil)
	}
}

// TestFarbpaarRechteLeiste: Die Zeile über dem Farbwähler wählt das Feld, „Keine“ macht die Füllung durchsichtig.
func TestFarbpaarRechteLeiste(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	k := &l.e.seite
	for i, r := range k.paarRahmen {
		if r.Empty() {
			t.Fatalf("Knopf %d der Zeile nicht gezeichnet", i)
		}
	}
	l.klick(mittenPunkt(k.farbRahmen[1])) // Füllung (Voreinstellung der Vorrichtung) auf Blau
	if l.text().Hintergrund != canvas.Beschriftungsfarben[1] {
		t.Fatalf("Schild %v", l.text().Hintergrund)
	}
	l.klick(mittenPunkt(k.paarRahmen[paarKeine]))
	if a := l.text().Hintergrund.A; a != 0 {
		t.Errorf("„Keine“: Deckkraft %d", a)
	}
	l.klick(mittenPunkt(k.paarRahmen[paarVorn]))
	l.klick(mittenPunkt(k.farbRahmen[2]))
	if tx := l.text(); tx.Farbe != canvas.Beschriftungsfarben[2] || tx.Hintergrund.A != 0 {
		t.Errorf("Vordergrund: Schrift %v, Schild %v", tx.Farbe, tx.Hintergrund)
	}
	// „Keine“ wirkt nur bei aktiver Füllung.
	l.klick(mittenPunkt(k.paarRahmen[paarKeine]))
	if l.text().Farbe.A == 0 {
		t.Error("„Keine“ machte den Vordergrund durchsichtig")
	}
	l.klick(mittenPunkt(k.paarRahmen[paarFuellung]))
	if l.e.s.AktivesFeld() != canvas.FeldFuellung {
		t.Errorf("Feld %v", l.e.s.AktivesFeld())
	}
	vor := l.e.s.D.Stand()
	l.klick(mittenPunkt(k.paarRahmen[paarTausch]))
	if tx := l.text(); tx.Hintergrund != canvas.Beschriftungsfarben[2] || schritteSeit(l, vor) != 1 {
		t.Errorf("⇄: Schild %v", tx.Hintergrund)
	}
}

// TestFarbenTauschenTasteUndMenue: Strg+Umschalt+X tauscht; X allein bleibt Stauchen; der Menüeintrag ist ohne
// Füllung grau.
func TestFarbenTauschenTasteUndMenue(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.e.s.SetzeFarbe(color.NRGBA{G: 200, A: 255}) // Schild grün
	alt := l.text()
	l.taste("X", key.ModShortcut|key.ModShift)
	if tx := l.text(); tx.Farbe != alt.Hintergrund || tx.Hintergrund != alt.Farbe {
		t.Errorf("Strg+Umschalt+X: Schrift %v, Schild %v", tx.Farbe, tx.Hintergrund)
	}
	if l.e.s.Werkzeug == canvas.WerkzeugStauchen {
		t.Error("Strg+Umschalt+X wählte Stauchen")
	}
	an := func() bool {
		for _, en := range l.e.menueEintraegeVon(menueBearbeiten) {
			if en.akt == aktFarbenTauschen {
				return en.an
			}
		}
		t.Fatal("kein Eintrag „Farben tauschen“ im Menü Bearbeiten")
		return false
	}
	if !an() {
		t.Error("Eintrag grau, obwohl der Text eine Füllung hat")
	}
	l.e.s.Auswahl = 0
	l.e.s.SetzeWerkzeug(canvas.WerkzeugNummer)
	if an() {
		t.Error("Eintrag an bei der Nummer (ohne Füllung)")
	}
}

// TestZiehenAufKeinerFuellung: Ein Zug in der Farbfläche macht eine ganz durchsichtige Füllung deckend (ADR-0021
// Nachtrag 11 Punkt 7); der Deckkraft-Regler setzt sie wie bisher.
func TestZiehenAufKeinerFuellung(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	fw := &l.e.seite.fw
	if l.text().Hintergrund.A != 0 || l.e.s.AktivesFeld() != canvas.FeldFuellung {
		t.Fatalf("Vorbedingung: Schild %v, Feld %v", l.text().Hintergrund, l.e.s.AktivesFeld())
	}
	l.ziehe(lageIn(fw.flaeche, 0.5, 0.5), lageIn(fw.flaeche, 0.9, 0.1))
	if a := l.text().Hintergrund.A; a != 255 {
		t.Errorf("nach der Farbfläche: Deckkraft %d", a)
	}
	l.ziehe(lageIn(fw.deck, 0.5, 0.5), image.Pt(fw.deck.Min.X-50, fw.deck.Min.Y))
	if a := l.text().Hintergrund.A; a != 0 {
		t.Errorf("Deckkraft-Regler links: %d", a)
	}
}
