// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"math"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// ziehe drückt bei a, zieht nach b und lässt dort los (Fensterpixel).
func (l *menueLauf) ziehe(a, b image.Point) {
	pa, pb := f32.Pt(float32(a.X), float32(a.Y)), f32.Pt(float32(b.X), float32(b.Y))
	l.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: pa})
	l.setle()
	l.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: pa})
	l.setle()
	l.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: pb}) // Gio macht daraus Drag
	l.setle()
	l.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: pb})
	l.setle()
}

// textHintergrund ist das Schild des Texts im Test.
func (l *menueLauf) textHintergrund() color.NRGBA {
	o, _ := l.e.s.D.Objekt(l.id)
	return o.(document.Text).Hintergrund
}

func lageIn(r image.Rectangle, fx, fy float64) image.Point {
	return image.Pt(r.Min.X+int(math.Round(fx*float64(r.Dx()))), r.Min.Y+int(math.Round(fy*float64(r.Dy()))))
}

// TestFarbflaecheUndRegler: Ziehen in Fläche, Farbton- und Deckkraft-Regler färbt den gewählten Text; jedes Ziehen ist
// genau ein Rückgängig-Schritt (ADR-0021 Nachtrag 9 Punkt 2).
func TestFarbflaecheUndRegler(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	fw := &l.e.seite.fw
	if fw.flaeche.Empty() || fw.ton.Empty() || fw.deck.Empty() || fw.pipRahmen.Empty() {
		t.Fatalf("Farbwähler nicht gezeichnet: %v %v %v %v", fw.flaeche, fw.ton, fw.deck, fw.pipRahmen)
	}
	l.klick(mittenPunkt(l.e.seite.farbRahmen[0])) // Rot, deckend: Ausgang für die Regler
	vor := l.e.s.D.Stand()
	// Farbton: ein Drittel → Grün (120°). Gedrückt wird innen, losgelassen darf außerhalb sein.
	l.ziehe(lageIn(fw.ton, 0.9, 0.5), lageIn(fw.ton, 1.0/3, 0.5))
	l.ziehe(lageIn(fw.flaeche, 0.2, 0.8), lageIn(fw.flaeche, 1.2, -0.2)) // volle Sättigung und Helligkeit
	h := canvas.HSVAus(l.textHintergrund())
	if math.Abs(h.H-120) > 2 || h.S < 0.99 || h.V < 0.99 {
		t.Errorf("nach Farbton und Fläche: %v (%+v)", l.textHintergrund(), h)
	}
	l.ziehe(lageIn(fw.deck, 0.95, 0.5), lageIn(fw.deck, 0.5, 0.5))
	if a := l.textHintergrund().A; a < 125 || a > 130 {
		t.Errorf("Deckkraft halb: Alpha %d", a)
	}
	// Über den Rand hinaus: begrenzt.
	l.ziehe(lageIn(fw.deck, 0.5, 0.5), image.Pt(fw.deck.Min.X-300, fw.deck.Min.Y))
	if a := l.textHintergrund().A; a != 0 {
		t.Errorf("Deckkraft links außerhalb: Alpha %d", a)
	}
	if n := schritteSeit(l, vor); n != 4 {
		t.Errorf("%d Rückgängig-Schritte, erwartet 4", n)
	}
	if l.e.s.Zieht() || fw.zieht != reglerKeiner {
		t.Error("Ziehen endete nicht")
	}
	if z := l.e.s.Zuletzt(); len(z) == 0 || z[0] != l.textHintergrund() {
		t.Errorf("zuletzt %v", z)
	}
	// Die zuletzt benutzte Farbe steht als fünftes Feld und ist wählbar.
	l.setle()
	if l.e.seite.farbRahmen[4].Empty() {
		t.Fatal("kein Feld für die zuletzt benutzte Farbe")
	}
}

// schritteSeit zählt die Rückgängig-Schritte bis zum Stand vor und stellt sie wieder her.
func schritteSeit(l *menueLauf, vor uint64) int {
	n := 0
	for l.e.s.D.Stand() != vor && l.e.s.Rueckgaengig() {
		n++
	}
	for i := 0; i < n; i++ {
		l.e.s.Wiederholen()
	}
	return n
}

// TestFarbwaehlerUnterHilfe: Unter der Hilfetafel wirken Ziehen und Pipette nicht (wie die Knöpfe, A-8).
func TestFarbwaehlerUnterHilfe(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	fw := &l.e.seite.fw
	vor, farbe := l.e.s.D.Stand(), l.e.s.Stil.Farbe
	l.e.hilfeOffen = true
	l.ziehe(lageIn(fw.flaeche, 0.5, 0.5), lageIn(fw.flaeche, 0.9, 0.1))
	l.klick(mittenPunkt(fw.pipRahmen))
	if l.e.s.D.Stand() != vor || l.e.s.Stil.Farbe != farbe || l.e.s.PipetteLaeuft() {
		t.Error("Farbwähler wirkte unter der Hilfetafel")
	}
}

// TestHexEingabe: Klick ins Feld, tippen, Enter übernimmt. Buchstaben sind Eingabe, keine Werkzeugkürzel (A = Pfeil,
// B = Schwärzen, C = Zuschneiden); Rücktaste und Entf löschen nicht das gewählte Objekt.
func TestHexEingabe(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	fw := &l.e.seite.fw
	l.klick(mittenPunkt(fw.hex.rahmen))
	if !l.fokus(&fw.hex) {
		t.Fatalf("Klick fokussiert das Hex-Feld nicht (%s)", menueFokusName(l))
	}
	w := l.e.s.Werkzeug
	for _, z := range []string{"A", "B", "C"} { // Gio liefert die Taste und den Text
		l.r.Queue(key.Event{Name: key.Name(z), State: key.Press})
		l.r.Queue(key.EditEvent{Text: z})
		l.setle()
	}
	l.taste(key.NameDeleteBackward, 0)
	l.taste(key.NameDeleteForward, 0)
	if _, ok := l.e.s.D.Objekt(l.id); !ok {
		t.Fatal("Rücktaste oder Entf im Feld löschte das gewählte Objekt")
	}
	if l.e.s.Werkzeug != w || fw.hex.gezeigt != "AB" {
		t.Fatalf("Werkzeug %v (vorher %v), Feld %q", l.e.s.Werkzeug, w, fw.hex.gezeigt)
	}
	l.tippe("#zz1") // z wird nicht angenommen
	if fw.hex.gezeigt != "AB#1" {
		t.Errorf("Feld %q", fw.hex.gezeigt)
	}
	l.taste(key.NameReturn, 0)
	if !fw.hex.fehler || !l.fokus(&fw.hex) {
		t.Fatalf("ungültig: Fehler %v, Fokus %v", fw.hex.fehler, l.fokus(&fw.hex))
	}
	for range 4 {
		l.taste(key.NameDeleteBackward, 0)
	}
	vor := l.e.s.D.Stand()
	l.tippe("#336699")
	l.taste(key.NameReturn, 0)
	want := color.NRGBA{R: 0x33, G: 0x66, B: 0x99, A: l.e.s.Stil.Farbe.A}
	if fw.hex.fehler || l.textHintergrund() != want || l.e.s.D.Stand() == vor {
		t.Fatalf("nach Enter: Fehler %v, Schild %v, erwartet %v", fw.hex.fehler, l.textHintergrund(), want)
	}
	if fw.hex.gezeigt != canvas.HexText(want) {
		t.Errorf("Anzeige nach Enter %q", fw.hex.gezeigt)
	}
	// Esc verwirft und gibt den Fokus an die Leinwand.
	l.tippe("ff0000")
	l.taste(key.NameEscape, 0)
	if l.textHintergrund() != want || !l.fokus(l.e) || l.e.s.Auswahl != l.id {
		t.Errorf("Esc: Schild %v, Fokus %s, Auswahl %d", l.textHintergrund(), menueFokusName(l), l.e.s.Auswahl)
	}
}

// TestDeckkraftEingabeUndTab: Tab aus dem Feld übernimmt die Eingabe; ungültig bleibt alles beim Alten.
func TestDeckkraftEingabeUndTab(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	fw := &l.e.seite.fw
	l.fokusiere(&fw.deckkraft)
	l.tippe("50")
	l.r.MoveFocus(key.FocusForward)
	l.setle()
	if a := l.textHintergrund().A; a != 128 {
		t.Errorf("Deckkraft 50 nach Tab: Alpha %d", a)
	}
	vor := l.e.s.D.Stand()
	l.fokusiere(&fw.deckkraft)
	l.tippe("999") // höchstens 4 Zeichen, aber über 100: ungültig
	l.r.MoveFocus(key.FocusForward)
	l.setle()
	if l.e.s.D.Stand() != vor || fw.deckkraft.fehler {
		t.Errorf("ungültige Deckkraft wirkte (Stand), Fehler stehen geblieben %v", fw.deckkraft.fehler)
	}
	// Während des Speicherns übernimmt Enter nichts.
	l.fokusiere(&fw.deckkraft)
	l.tippe("10")
	l.e.s.SpeichernJetzt()
	l.taste(key.NameReturn, 0)
	if l.e.s.D.Stand() != vor {
		t.Error("Eingabe während des Speicherns übernommen")
	}
}

// TestPipetteKnopf: Der Knopf schaltet die Pipette, die Fußzeile sagt es, ein Klick ins Bild nimmt die Farbe auf.
func TestPipetteKnopf(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	fw := &l.e.seite.fw
	l.klick(mittenPunkt(fw.pipRahmen))
	if !l.e.s.PipetteLaeuft() || l.e.hinweis() != "Click in the image to pick its color. Esc cancels." {
		t.Fatalf("Pipette %v, Hinweis %q", l.e.s.PipetteLaeuft(), l.e.hinweis())
	}
	l.taste(key.NameEscape, 0)
	if l.e.s.PipetteLaeuft() || l.e.s.Auswahl != l.id {
		t.Fatal("Esc beendet die Pipette nicht oder hob die Auswahl auf")
	}
	l.fokusiere(&fw.pipette)
	l.taste(key.NameReturn, 0)
	if !l.e.s.PipetteLaeuft() {
		t.Fatal("Enter auf der Pipette schaltet sie nicht ein")
	}
	// Das Testbild ist 40×30 und durchsichtig; durchsichtige Bildpunkte nimmt die Pipette nicht (A-9).
	l.e.s.D.OriginalBild().SetNRGBA(20, 15, color.NRGBA{A: 255})
	l.e.bildNeu()
	f := l.e.leinwandFlaeche(l.gtx())
	p := l.e.s.Abb.ZuFenster(document.Pt(20, 15))
	l.klick(f.Min.Add(image.Pt(int(p.X), int(p.Y))))
	if l.e.s.PipetteLaeuft() || l.e.s.Stil.Farbe != (color.NRGBA{A: 255}) {
		t.Errorf("Klick ins Bild: Pipette %v, Farbe %v", l.e.s.PipetteLaeuft(), l.e.s.Stil.Farbe)
	}
	if l.textHintergrund() != l.e.s.Stil.Farbe {
		t.Errorf("Schild %v, Farbe %v", l.textHintergrund(), l.e.s.Stil.Farbe)
	}
}
