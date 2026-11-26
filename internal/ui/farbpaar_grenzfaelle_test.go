// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Tests aus der Kritik zu Karte #219 (B-1, B-2, B-5). Dass Hex und Deckkraft in niedrigen Fenstern unter die
// Fußzeile laufen, bleibt offen (Bericht b219, Finding-Vorschlag): Die Einklapp-Logik kann nur die Ebenen einklappen.

import (
	"image"
	"testing"

	"gioui.org/io/key"

	"github.com/Andiministrator/schnapp/internal/canvas"
)

// TestTonReglerAufKeinerFuellung: Der Farbton-Regler macht eine ganz durchsichtige Füllung („keine“, {0,0,0,0}) zu
// einem sichtbaren Farbton, nicht zu deckendem Schwarz (Kritik B-1).
func TestTonReglerAufKeinerFuellung(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800)) // Schild {0,0,0,0}, Feld Füllung aktiv
	fw := &l.e.seite.fw
	l.ziehe(lageIn(fw.ton, 0.3, 0.5), lageIn(fw.ton, 0.6, 0.5))
	c := l.text().Hintergrund
	if c.A != 255 || (c.R == 0 && c.G == 0 && c.B == 0) {
		t.Errorf("Farbton-Regler auf „keine“: %v, erwartet ein deckender Farbton", c)
	}
}

// TestFuellungPerTastatur: Leertaste am Feld Füllung der Werkzeugleiste öffnet die Farbliste für die Füllung, Esc
// gibt den Fokus an das Feld zurück.
func TestFuellungPerTastatur(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	k := &l.e.knoepfe
	l.e.s.SetzeFeld(canvas.FeldVordergrund)
	l.fokusiere(&k.fuellung)
	l.taste(key.NameSpace, 0)
	if !l.e.menue.offen || l.e.menue.welches != listeFarbe || l.e.s.AktivesFeld() != canvas.FeldFuellung {
		t.Fatalf("Leertaste am Feld Füllung: offen %v, Menü %d, Feld %v", l.e.menue.offen, l.e.menue.welches, l.e.s.AktivesFeld())
	}
	l.taste(key.NameEscape, 0)
	if !l.fokus(&k.fuellung) {
		t.Errorf("nach Esc liegt der Fokus nicht am Feld Füllung")
	}
}

// TestGrauesFeldPerTastatur: Enter und Leertaste am grauen Feld Füllung (Pfeil) tun nichts.
func TestGrauesFeldPerTastatur(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.e.s.Auswahl = 0
	l.e.s.SetzeWerkzeug(canvas.WerkzeugPfeil)
	l.setle()
	for _, n := range []key.Name{key.NameReturn, key.NameSpace} {
		l.fokusiere(&l.e.knoepfe.fuellung)
		l.taste(n, 0)
		if l.e.menue.offen || l.e.s.AktivesFeld() != canvas.FeldVordergrund {
			t.Errorf("%q am grauen Feld Füllung: Liste offen %v", n, l.e.menue.offen)
			l.taste(key.NameEscape, 0)
		}
	}
}

// TestEnterAmTauschKnopf: Enter am Knopf ⇄ tauscht Schrift und Schild in einem Schritt, statt eine Texteingabe zu
// beginnen.
func TestEnterAmTauschKnopf(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.e.s.SetzeFarbe(canvas.Beschriftungsfarben[1])
	alt, vor := l.text(), l.e.s.D.Stand()
	l.fokusiere(&l.e.knoepfe.tausch)
	l.taste(key.NameReturn, 0)
	if l.e.s.TextLaeuft() {
		t.Fatal("Enter am Knopf ⇄ begann eine Texteingabe")
	}
	if tx := l.text(); tx.Farbe != alt.Hintergrund || tx.Hintergrund != alt.Farbe || schritteSeit(l, vor) != 1 {
		t.Errorf("Enter am Knopf ⇄: Schrift %v, Schild %v", tx.Farbe, tx.Hintergrund)
	}
}

// TestFarbwaehlerHoehe bindet farbwaehlerHoeheDp an die gezeichnete Höhe des Farbwählers, von der Zeile des
// Farbpaars bis unter Hex und Deckkraft (Kritik B-5): Die Schwelle für das Einklappen der Ebenen rechnet damit.
func TestFarbwaehlerHoehe(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800)) // PxPerDp 1
	se := &l.e.seite
	if !se.farbeGezeigt {
		t.Fatal("Farbwähler nicht gezeichnet")
	}
	h := se.fw.hex.rahmen.Max.Y - se.paarRahmen[paarVorn].Min.Y
	if d := se.fw.deckkraft.rahmen.Max.Y - se.paarRahmen[paarVorn].Min.Y; d > h {
		h = d
	}
	if h != farbwaehlerHoeheDp {
		t.Errorf("Farbwähler %d dp hoch, farbwaehlerHoeheDp = %d", h, farbwaehlerHoeheDp)
	}
	// Die Farbfelder darunter (ADR-0021 Nachtrag 24 Punkt 7): bis zur Unterkante von „+“.
	breite := se.fw.pipRahmen.Max.X - se.paarRahmen[paarVorn].Min.X // die Pipette steht am rechten Rand
	if ff := se.ff.plusRahmen.Max.Y - h - se.paarRahmen[paarVorn].Min.Y; ff != l.e.farbfelderHoehe(l.gtx(), breite) {
		t.Errorf("Farbfelder %d dp hoch, farbfelderHoehe = %d", ff, l.e.farbfelderHoehe(l.gtx(), breite))
	}
}
