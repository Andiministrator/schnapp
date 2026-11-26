// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Regressionstests aus der Sitzungs-QA 23.11.2026 abends (Kritiker A, Karte #189): schwebende Eingabe der Felder bei
// Klick woanders und Fensterwechsel (A-2), Tasten bei offenem Menü und offener Hilfe (A-3, A-7), Umschalt+Enter (A-5),
// Klick auf eine zuletzt benutzte Farbe.

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/io/key"
	"gioui.org/io/pointer"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

var qaGruen = color.NRGBA{G: 255, A: 255}

// A-2a: Eingabe im Hex-Feld, dann Klick auf die Leinwand neben das Bild (Drücken und Loslassen in getrennten Bildern,
// wie eine echte Maus). ADR-0031 Nachtrag 5: „ein Klick woanders übernimmt ebenfalls“.
func TestSitzungsQAHexDannKlickLeinwand(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	fw := &l.e.seite.fw
	l.klick(mittenPunkt(fw.hex.rahmen))
	l.tippe("00ff00ff") // acht Ziffern: deckend (das Schild des Testtexts ist durchsichtig)
	f := l.e.leinwandFlaeche(l.gtx())
	p := f.Min.Add(image.Pt(6, 6))
	l.maus(pointer.Move, p, 0)
	l.setle()
	l.maus(pointer.Press, p, pointer.ButtonPrimary)
	l.setle()
	l.maus(pointer.Release, p, 0)
	l.setle()
	t.Logf("getrennte Bilder: Schild %v, Stil %v, Auswahl %d, Fokus Leinwand %v", l.textHintergrund(), l.e.s.Stil.Farbe, l.e.s.Auswahl, l.fokus(l.e))
	if l.textHintergrund() != qaGruen {
		t.Errorf("Klick auf die Leinwand übernahm die Eingabe nicht: Schild %v, Werkzeugfarbe %v", l.textHintergrund(), l.e.s.Stil.Farbe)
	}
}

// A-2a': dasselbe, Drücken und Loslassen im selben Bild (Helfer klick).
func TestSitzungsQAHexDannKlickLeinwandEinBild(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	fw := &l.e.seite.fw
	l.klick(mittenPunkt(fw.hex.rahmen))
	l.tippe("00ff00ff") // acht Ziffern: deckend (das Schild des Testtexts ist durchsichtig)
	f := l.e.leinwandFlaeche(l.gtx())
	l.klick(f.Min.Add(image.Pt(6, 6)))
	t.Logf("ein Bild: Schild %v, Stil %v, Auswahl %d", l.textHintergrund(), l.e.s.Stil.Farbe, l.e.s.Auswahl)
	if l.textHintergrund() != qaGruen {
		t.Errorf("Klick auf die Leinwand färbte das bearbeitete Objekt nicht: Schild %v, Werkzeugfarbe %v", l.textHintergrund(), l.e.s.Stil.Farbe)
	}
}

// A-2b: Eingabe im Hex-Feld, dann Klick auf ein Farbfeld. Das Feld behält Fokus und alten Text; ein späteres Tab
// übernimmt den alten Text über die Wahl des Farbfelds hinweg.
func TestSitzungsQAHexDannFarbfeld(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	fw := &l.e.seite.fw
	l.klick(mittenPunkt(fw.hex.rahmen))
	l.tippe("00ff00")
	l.klick(mittenPunkt(l.e.seite.ff.rahmen[1]))
	want := canvas.Beschriftungsfarben[1]
	t.Logf("nach Farbfeld: Schild %v, Fokus Hex %v, Feld zeigt %q, aktuelle Farbe %s", l.textHintergrund(), l.fokus(&fw.hex), fw.hex.gezeigt, canvas.HexText(l.e.s.AktuelleFarbe()))
	if fw.hex.gezeigt != canvas.HexText(l.e.s.AktuelleFarbe()) {
		t.Errorf("Hex-Feld zeigt %q, die Farbe ist %s", fw.hex.gezeigt, canvas.HexText(l.e.s.AktuelleFarbe()))
	}
	l.r.MoveFocus(key.FocusForward)
	l.setle()
	t.Logf("nach Tab: Schild %v", l.textHintergrund())
	if l.textHintergrund() != want {
		t.Errorf("nach Tab: Schild %v, das Farbfeld hatte %v gewählt", l.textHintergrund(), want)
	}
}

// A-2c: Eingabe im Hex-Feld, dann Pipette. Die Pipette nimmt Schwarz (deckend); danach übernimmt der Fokuswechsel die
// alte Eingabe darüber.
func TestSitzungsQAHexDannPipette(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	fw := &l.e.seite.fw
	l.klick(mittenPunkt(fw.hex.rahmen))
	l.tippe("00ff00ff")
	l.e.s.D.OriginalBild().SetNRGBA(20, 15, color.NRGBA{A: 255}) // durchsichtige Bildpunkte nimmt die Pipette nicht (A-9)
	l.e.bildNeu()
	l.klick(mittenPunkt(fw.pipRahmen))
	if !l.e.s.PipetteLaeuft() {
		t.Fatal("Pipette nicht an")
	}
	f := l.e.leinwandFlaeche(l.gtx())
	p := l.e.s.Abb.ZuFenster(document.Pt(20, 15))
	l.klick(f.Min.Add(image.Pt(int(p.X), int(p.Y))))
	t.Logf("nach Pipette: Schild %v, Stil %v", l.textHintergrund(), l.e.s.Stil.Farbe)
	if l.textHintergrund() != (color.NRGBA{A: 255}) {
		t.Errorf("Schild %v, die Pipette nahm Schwarz deckend", l.textHintergrund())
	}
}

// A-3: Menü per Maus geöffnet, während das Hex-Feld den Fokus hat: Esc muss das Menü schließen.
func TestSitzungsQAHexFokusMenueEsc(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	fw := &l.e.seite.fw
	l.klick(mittenPunkt(fw.hex.rahmen))
	l.klick(mittenPunkt(l.e.menue.titelRahmen[0]))
	if !l.e.menue.offen {
		t.Fatal("Menü nicht offen")
	}
	t.Logf("Menü offen, Fokus Hex %v", l.fokus(&fw.hex))
	l.taste(key.NameEscape, 0)
	if l.e.menue.offen {
		t.Errorf("Esc schließt das per Maus geöffnete Menü nicht, solange das Hex-Feld den Fokus hat")
	}
}

// A-5: Umschalt+Enter im Hex-Feld bei gewähltem Text.
func TestSitzungsQAHexUmschaltEnter(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	fw := &l.e.seite.fw
	l.klick(mittenPunkt(fw.hex.rahmen))
	l.tippe("00ff00")
	l.taste(key.NameReturn, key.ModShift)
	t.Logf("Umschalt+Enter: Texteingabe %v, Fokus Hex %v, Schild %v", l.e.s.TextLaeuft(), l.fokus(&fw.hex), l.textHintergrund())
	if l.e.s.TextLaeuft() {
		t.Errorf("Umschalt+Enter im Hex-Feld beginnt die Texteingabe am gewählten Text, der Fokus bleibt im Feld (%v)", l.fokus(&fw.hex))
	}
}

// A-7: F1 bei Fokus im Feld, dann Esc: Die Hilfetafel soll mit einem Esc schließen.
func TestSitzungsQAHexFokusHilfeEsc(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	fw := &l.e.seite.fw
	l.klick(mittenPunkt(fw.hex.rahmen))
	l.taste(key.NameF1, 0)
	if !l.e.hilfeOffen {
		t.Fatal("Hilfe nicht offen")
	}
	l.taste(key.NameEscape, 0)
	if l.e.hilfeOffen {
		t.Errorf("erstes Esc schließt die Hilfetafel nicht (Fokus Hex %v)", l.fokus(&fw.hex))
	}
}

// A-8: Fenster verliert den Fokus mitten in der Eingabe (Blick in ein anderes Fenster).
func TestSitzungsQAHexFensterFokusWeg(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	fw := &l.e.seite.fw
	l.klick(mittenPunkt(fw.hex.rahmen))
	vor := l.e.s.D.Stand()
	l.tippe("336") // unterwegs zu 336699
	l.r.Queue(key.FocusEvent{Focus: false})
	l.setle()
	t.Logf("Fenster ohne Fokus: Schild %v, Stand geändert %v, Feld zeigt %q", l.textHintergrund(), l.e.s.D.Stand() != vor, fw.hex.gezeigt)
	if l.e.s.D.Stand() != vor {
		t.Errorf("halb getippte Eingabe wurde beim Fokusverlust des Fensters übernommen: %v", l.textHintergrund())
	}
}

// Lücke: Klick auf eine zuletzt benutzte Farbe (fünftes Feld) — erwartet grün.
func TestSitzungsQAZuletztFeldKlick(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	fw := &l.e.seite.fw
	l.klick(mittenPunkt(fw.hex.rahmen))
	l.tippe("123456")
	l.taste(key.NameReturn, 0)
	l.taste(key.NameEscape, 0)
	l.klick(mittenPunkt(l.e.seite.ff.rahmen[0]))
	l.setle()
	if l.e.seite.farbRahmen[0].Empty() {
		t.Fatal("kein fünftes Feld")
	}
	l.klick(mittenPunkt(l.e.seite.farbRahmen[0]))
	if l.textHintergrund() != (color.NRGBA{R: 0x12, G: 0x34, B: 0x56, A: l.textHintergrund().A}) {
		t.Errorf("Klick auf zuletzt benutzte Farbe: Schild %v", l.textHintergrund())
	}
}
