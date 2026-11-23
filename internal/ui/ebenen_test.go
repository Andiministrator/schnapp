// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"strings"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// ebenenLauf: Editor mit dem Text „Hallo“ (unten) und einem Kasten darüber, Fenster hoch genug für die ganze Leiste.
func ebenenLauf(t *testing.T) (*menueLauf, int) {
	t.Helper()
	l := neuerMenueLauf(t, image.Pt(1200, 1200))
	k, err := l.e.s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(2, 2), document.Pt(20, 20)),
		Strich: document.Strich{Farbe: canvas.Schwaerzfarbe, Breite: 2}})
	if err != nil {
		t.Fatal(err)
	}
	l.setle()
	return l, k
}

func mitte(r image.Rectangle) image.Point { return r.Min.Add(r.Max).Div(2) }

func (l *menueLauf) ebene(id int) document.Ebene {
	o, _ := l.e.s.D.Objekt(id)
	return o.Ebenenwerte()
}

// ADR-0032 Punkt 6: Liste von oben nach unten mit dem Bild zuletzt; Klick aufs Auge blendet aus, aufs Schloss sperrt,
// auf den Namen wählt.
func TestEbenenLeisteKlicks(t *testing.T) {
	l, k := ebenenLauf(t)
	eb := &l.e.ebenen
	if !eb.gezeigt || len(eb.namen) != 3 || eb.namen[0] != "Rectangle" || eb.namen[1] != "Text “Hallo”" || eb.namen[2] != "Image" {
		t.Fatalf("gezeigt %v, Namen %q", eb.gezeigt, eb.namen)
	}
	z := eb.zeilen[0].rahmen
	if z.Empty() || !z.In(eb.liste) {
		t.Fatalf("Zeile %v nicht in der Liste %v", z, eb.liste)
	}
	l.klick(image.Pt(z.Min.X+eb.augeB/2, mitte(z).Y))
	if !l.ebene(k).Ausgeblendet {
		t.Error("Klick aufs Auge blendet nicht aus")
	}
	l.klick(image.Pt(z.Max.X-eb.augeB/2, mitte(z).Y))
	if !l.ebene(k).Gesperrt {
		t.Error("Klick aufs Schloss sperrt nicht")
	}
	l.klick(mitte(z))
	if l.e.s.Auswahl != k {
		t.Errorf("Klick auf die Zeile wählt %d, Soll %d", l.e.s.Auswahl, k)
	}
	// Bild: keine Aktion
	vor := l.e.s.D.Stand()
	l.klick(mitte(eb.zeilen[2].rahmen))
	if l.e.s.D.Stand() != vor || l.e.s.Auswahl != k {
		t.Error("Klick auf die Bild-Zeile wirkt")
	}
	// Überschrift klappt ein und aus.
	l.fokusiere(&eb.kopf)
	l.taste(key.NameReturn, 0)
	if !eb.zu || eb.gezeigt {
		t.Error("Enter auf der Überschrift klappt nicht ein")
	}
}

// Ziehen einer Zeile ordnet um (ein Schritt); Knöpfe ↑ ↓ wirken an der Wahl.
func TestEbenenLeisteZiehenUndKnoepfe(t *testing.T) {
	l, k := ebenenLauf(t)
	eb := &l.e.ebenen
	a, b := mitte(eb.zeilen[0].rahmen), eb.zeilen[1].rahmen.Max.Sub(image.Pt(eb.zeilen[1].rahmen.Dx()/2, 2))
	vor := l.e.s.D.Stand()
	l.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(float32(a.X), float32(a.Y))})
	l.setle()
	l.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(float32(a.X), float32(a.Y))})
	l.setle()
	l.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(float32(b.X), float32(b.Y))})
	l.setle()
	if !eb.druck.zieht || eb.druck.slot != 2 {
		t.Fatalf("Ziehen: %+v", eb.druck)
	}
	l.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(float32(b.X), float32(b.Y))})
	l.setle()
	if l.e.s.D.Stelle(k) != 0 || l.e.s.D.Stand() == vor {
		t.Fatalf("Kasten nach dem Ziehen an Stelle %d", l.e.s.D.Stelle(k))
	}
	if eb.namen[0] != "Text “Hallo”" {
		t.Errorf("Liste nach dem Ziehen: %q", eb.namen)
	}
	l.e.s.WaehleEbene(k)
	l.klick(mitte(eb.knopfRahmen[ebKnopfHoch]))
	if l.e.s.D.Stelle(k) != 1 {
		t.Error("Knopf ↑ bringt nicht nach vorn")
	}
	l.klick(mitte(eb.knopfRahmen[ebKnopfDoppeln]))
	if l.e.s.Auswahl == k || len(l.e.s.D.Objekte()) != 3 {
		t.Error("Knopf Doppeln")
	}
	l.klick(mitte(eb.knopfRahmen[ebKnopfLoeschen]))
	if len(l.e.s.D.Objekte()) != 2 {
		t.Error("Knopf Löschen")
	}
}

// F2 öffnet das Namensfeld mit Fokus; Tippen ersetzt, Enter übernimmt (ein Schritt), Esc verwirft. Werkzeugbuchstaben
// sind im Feld Text.
func TestEbenenUmbenennen(t *testing.T) {
	l, k := ebenenLauf(t)
	l.e.s.WaehleEbene(k)
	l.taste(key.NameF2, 0)
	eb := &l.e.ebenen
	if eb.umbenennen != k || !l.fokus(&eb.name) {
		t.Fatalf("F2: umbenennen %d, Fokus %v", eb.umbenennen, l.fokus(&eb.name))
	}
	l.tippe("Vorsicht")
	if l.e.s.Werkzeug != canvas.WerkzeugAuswahl {
		t.Error("Buchstabe im Namensfeld hat ein Werkzeug gewählt")
	}
	l.taste(key.NameReturn, 0)
	if got := l.ebene(k).Name; got != "Vorsicht" || eb.umbenennen != 0 {
		t.Fatalf("Name %q, umbenennen %d", got, eb.umbenennen)
	}
	if eb.namen[0] != "Vorsicht" {
		t.Errorf("Liste zeigt %q", eb.namen[0])
	}
	vor := l.e.s.D.Stand()
	l.taste(key.NameF2, 0)
	l.tippe("weg")
	l.taste(key.NameEscape, 0)
	if l.ebene(k).Name != "Vorsicht" || l.e.s.D.Stand() != vor || eb.umbenennen != 0 {
		t.Error("Esc verwirft nicht")
	}
	// Deckkraft-Feld
	l.fokusiere(&eb.deck)
	l.tippe("25")
	l.taste(key.NameReturn, 0)
	if l.ebene(k).Deckkraft() != 25 {
		t.Errorf("Deckkraft %d", l.ebene(k).Deckkraft())
	}
}

// Menü „Ebene“ und Kürzel: Strg+H, Strg+L, Strg+D, Strg+Bild↓, Alt+Bild↑.
func TestEbenenKuerzel(t *testing.T) {
	l, k := ebenenLauf(t)
	l.e.s.WaehleEbene(k)
	l.taste("H", key.ModShortcut)
	l.taste("L", key.ModShortcut)
	if e := l.ebene(k); !e.Ausgeblendet || !e.Gesperrt {
		t.Errorf("Strg+H/L: %+v", e)
	}
	l.taste(key.NamePageDown, key.ModShortcut)
	if l.e.s.D.Stelle(k) != 0 {
		t.Error("Strg+Bild↓ bringt nicht nach hinten")
	}
	l.taste(key.NamePageUp, key.ModAlt)
	if l.e.s.Auswahl == k {
		t.Error("Alt+Bild↑ wählt nicht die Ebene darüber")
	}
	l.taste("D", key.ModShortcut)
	if len(l.e.s.D.Objekte()) != 3 {
		t.Error("Strg+D doppelt nicht")
	}
	en := l.e.menueEintraegeVon(menueEbene)
	if len(en) != 8 || !en[0].an {
		t.Errorf("Menü Ebene: %d Einträge", len(en))
	}
}

// Ziehen über zwei Zeilen: Die oberste Ebene landet zwischen der zweiten und dritten (die eigene Zeile zählt nicht mit).
func TestEbenenLeisteZiehenUeberZweiZeilen(t *testing.T) {
	l, k := ebenenLauf(t)
	n, err := l.e.s.D.Hinzufuegen(document.Nummer{Mitte: document.Pt(30, 20), Wert: 1, Radius: 5, Farbe: canvas.Schwaerzfarbe})
	if err != nil {
		t.Fatal(err)
	}
	l.setle()
	eb := &l.e.ebenen
	if len(eb.zeilen) != 4 || eb.zeilen[0].id != n {
		t.Fatalf("Zeilen %+v", eb.zeilen)
	}
	// Liste: Nummer, Kasten, Text, Bild. Nummer auf die Grenze zwischen Kasten und Text ziehen.
	a := mitte(eb.zeilen[0].rahmen)
	b := image.Pt(a.X, eb.zeilen[2].rahmen.Min.Y+2)
	l.ziehe(a, b)
	if got := []int{l.e.s.D.Stelle(n), l.e.s.D.Stelle(k)}; got[0] != 1 || got[1] != 2 {
		t.Errorf("Stellen Nummer/Kasten %v, Soll [1 2]", got)
	}
}

// F-586: Ausblenden meldet, dass der Inhalt in der Arbeitsdatei bleibt; in einem niedrigen Fenster bleiben Liste und
// Knöpfe in der Leiste (der Hinweis fällt weg).
func TestEbenenAusblendenMeldungUndNiedrigesFenster(t *testing.T) {
	l, k := ebenenLauf(t)
	l.e.s.WaehleEbene(k)
	l.taste("H", key.ModShortcut)
	if !l.ebene(k).Ausgeblendet || !strings.Contains(l.e.meldung.Text, "working file") {
		t.Errorf("Strg+H: ausgeblendet %v, Meldung %q", l.ebene(k).Ausgeblendet, l.e.meldung.Text)
	}
	l.e.s.WaehleEbene(l.id) // Text: höchster Inhalt über dem Abschnitt (Schrift, Farbe)
	l.fenst = image.Pt(1200, 560)
	l.setle()
	eb := &l.e.ebenen
	unten := 560 - fusszeileDp
	if !eb.gezeigt || eb.knopfRahmen[ebKnopfLoeschen].Max.Y > unten || eb.liste.Max.Y > unten {
		t.Errorf("Fenster 560: Liste %v, Knöpfe %v, Leiste endet %d", eb.liste, eb.knopfRahmen, unten)
	}
	// Noch niedriger, mit eingeklappter Farbe: Liste und Knöpfe bleiben in der Leiste.
	l.fenst = image.Pt(1200, 470)
	l.setle()
	if unten := 470 - fusszeileDp; !eb.gezeigt || eb.knopfRahmen[ebKnopfLoeschen].Max.Y > unten {
		t.Errorf("Fenster 470: Liste %v, Knöpfe %v, Leiste endet %d", eb.liste, eb.knopfRahmen, unten)
	}
}

// F-587: Leeren des Namensfelds einer Ebene ohne eigenen Namen ist kein Fehler; ein gesperrter Text lässt sich nicht
// per Enter bearbeiten.
func TestEbenenNameLeerUndGesperrterText(t *testing.T) {
	l, k := ebenenLauf(t)
	l.e.s.WaehleEbene(k)
	l.taste(key.NameF2, 0)
	for range "Rectangle" {
		l.taste(key.NameDeleteBackward, 0)
	}
	l.taste(key.NameReturn, 0)
	if eb := &l.e.ebenen; eb.name.fehler || eb.umbenennen != 0 {
		t.Errorf("leerer Name: Fehler %v, umbenennen %d", eb.name.fehler, eb.umbenennen)
	}
	l.e.s.WaehleEbene(l.id) // Text „Hallo“
	l.e.s.SchalteGesperrt(l.id)
	if l.e.s.KannTextBearbeiten() || l.e.s.TextBearbeiten() {
		t.Error("gesperrter Text lässt sich bearbeiten")
	}
}
