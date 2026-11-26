// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"math"
	"strings"
	"testing"

	"gioui.org/io/key"
	"gioui.org/io/pointer"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// kontextLauf legt einen Kasten an (nicht gewählt) und liefert seine Mitte im Fenster.
func kontextLauf(t *testing.T) (*menueLauf, int, image.Point) {
	t.Helper()
	l := neuerMenueLauf(t, image.Pt(1200, 900))
	id, err := l.e.s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(4, 4), document.Pt(20, 14)),
		Strich: document.Strich{Farbe: color.NRGBA{R: 200, A: 255}, Breite: 2}})
	if err != nil {
		t.Fatal(err)
	}
	l.e.s.Auswahl = 0
	l.setle()
	return l, id, l.fensterVon(document.Pt(4, 9)) // auf dem linken Strich (das Testbild ist 40 × 30)
}

// fensterVon ist der Fensterpunkt eines Punkts des Originals.
func (l *menueLauf) fensterVon(p document.Punkt) image.Point {
	q := l.e.s.Abb.ZuFenster(l.e.s.D.ZuAnsicht(p))
	return image.Pt(int(math.Round(q.X)), int(math.Round(q.Y))).Add(l.e.leinwandUrsprung)
}

// eintragMit liefert die Stelle des Eintrags mit der Handlung a in der offenen Liste (-1: fehlt).
func (l *menueLauf) eintragMit(a aktion) int {
	for i, en := range l.e.menueEintraege() {
		if en.akt == a {
			return i
		}
	}
	return -1
}

// ADR-0038 Punkt 2: Ein Rechtsklick auf ein Objekt wählt es — auch mit einem Zeichenwerkzeug — und öffnet am Punkt die
// Liste für das Objekt; ein Eintrag wirkt wie im Menü der Zeile (Doppeln). Das Werkzeug bleibt.
func TestKontextmenueAmObjekt(t *testing.T) {
	l, id, p := kontextLauf(t)
	l.e.s.SetzeWerkzeug(canvas.WerkzeugPfeil)
	l.rechtsklick(p)
	m := &l.e.menue
	if !m.offen || m.welches != listeKontext || l.e.s.Auswahl != id || !l.e.kontext.objekt {
		t.Fatalf("offen %v, Liste %d, Auswahl %d (Soll %d)", m.offen, m.welches, l.e.s.Auswahl, id)
	}
	if l.e.s.Werkzeug != canvas.WerkzeugPfeil {
		t.Errorf("Werkzeug %v, Soll Pfeil", l.e.s.Werkzeug)
	}
	if r := m.rahmen[0]; r.Min.X != p.X || r.Min.Y < p.Y {
		t.Errorf("erste Zeile bei %v, Klick bei %v", r, p)
	}
	i := l.eintragMit(aktDoppeln)
	if i < 0 || l.eintragMit(aktKopieren) >= 0 {
		t.Fatalf("Einträge für ein Objekt falsch: Doppeln %d, Kopieren %d", i, l.eintragMit(aktKopieren))
	}
	n := len(l.e.s.D.Objekte())
	l.klick(mitte(m.rahmen[i]))
	if len(l.e.s.D.Objekte()) != n+1 || m.offen {
		t.Errorf("nach „Duplicate“: %d Objekte (Soll %d), Liste offen %v", len(l.e.s.D.Objekte()), n+1, m.offen)
	}
}

// Ohne Objekt unter dem Zeiger hebt der Rechtsklick die Auswahl auf und zeigt die Einträge für das Bild; Esc schließt
// die Liste und gibt den Fokus an die Leinwand.
func TestKontextmenueAufLeererFlaeche(t *testing.T) {
	l, id, _ := kontextLauf(t)
	l.e.s.Auswahl = id
	l.rechtsklick(l.fensterVon(document.Pt(36, 1))) // neben dem Kasten und über dem Text „Hallo“ des Testlaufs
	if !l.e.menue.offen || l.e.s.Auswahl != 0 || l.e.kontext.objekt {
		t.Fatalf("offen %v, Auswahl %d, objekt %v", l.e.menue.offen, l.e.s.Auswahl, l.e.kontext.objekt)
	}
	if l.eintragMit(aktKopieren) < 0 || l.eintragMit(aktDoppeln) >= 0 {
		t.Error("Einträge für das Bild falsch")
	}
	l.taste(key.NameEscape, 0)
	if l.e.menue.offen || !l.fokus(l.e) {
		t.Errorf("nach Esc: offen %v, Fokus an der Leinwand %v", l.e.menue.offen, l.fokus(l.e))
	}
}

// ADR-0038 Punkt 6: Umschalt+F10 öffnet die Liste für die Auswahl an ihrer Mitte, mit Fokus auf dem ersten Eintrag;
// F10 allein öffnet weiter das Menü Datei.
func TestKontextmenuePerTaste(t *testing.T) {
	l, id, _ := kontextLauf(t)
	l.e.s.Auswahl = id
	l.setle()
	l.taste(key.NameF10, key.ModShift)
	m := &l.e.menue
	if !m.offen || m.welches != listeKontext || !l.e.kontext.objekt {
		t.Fatalf("offen %v, Liste %d", m.offen, m.welches)
	}
	if soll := l.fensterVon(document.Pt(12, 9)); l.e.kontext.anker.Min != soll {
		t.Errorf("Anker %v, Soll Mitte des Kastens %v", l.e.kontext.anker.Min, soll)
	}
	l.setle()
	if !l.fokus(&m.eintraege[menueErster(l.e.menueEintraege())]) {
		t.Error("Fokus nicht auf dem ersten Eintrag")
	}
	l.taste(key.NameEscape, 0)
	l.taste(key.NameF10, 0)
	if !m.offen || m.welches != menueDatei {
		t.Errorf("F10 allein: offen %v, Menü %d", m.offen, m.welches)
	}
}

// „Als Schnellstil sichern“ ist grau mit Erklärung, wenn der Stil schon eine Kachel ist (Kasten im eingebauten Rot).
func TestKontextmenueSchnellstilGrau(t *testing.T) {
	l, _, p := kontextLauf(t)
	l.rechtsklick(p)
	i := l.eintragMit(aktSchnellstilSichern)
	if i < 0 {
		t.Fatal("Eintrag fehlt")
	}
	en := l.e.menueEintraege()[i]
	if en.an == (en.erklaerung != "") {
		t.Errorf("an %v, Erklärung %q", en.an, en.erklaerung)
	}
}

// Kritik K-2 #313a: Die Übersicht F1 nennt Umschalt+F10.
func TestKontextmenueInDerUebersicht(t *testing.T) {
	e := testEditor(t)
	for _, z := range e.hilfeZeilen() {
		if z.text == "Context menu" && strings.HasSuffix(z.taste, "+F10") {
			return
		}
	}
	t.Error("Übersicht ohne Zeile „Context menu“ mit Umschalt+F10")
}

// Kritik K-4 #313a: Ein Linksdruck bei gehaltener rechter Taste zieht nicht unter dem offenen Kontextmenü.
func TestKontextmenueKeinZugDahinter(t *testing.T) {
	l, id, p := kontextLauf(t)
	l.maus(pointer.Move, p, 0)
	l.setle()
	l.maus(pointer.Press, p, pointer.ButtonSecondary)
	l.setle()
	l.maus(pointer.Press, p, pointer.ButtonSecondary|pointer.ButtonPrimary)
	l.setle()
	if !l.e.menue.offen || l.e.s.Zieht() {
		t.Fatalf("offen %v, Zug %v", l.e.menue.offen, l.e.s.Zieht())
	}
	vorher, _ := l.e.s.D.Objekt(id)
	l.maus(pointer.Move, p.Add(image.Pt(30, 0)), pointer.ButtonSecondary|pointer.ButtonPrimary)
	l.maus(pointer.Release, p.Add(image.Pt(30, 0)), 0)
	l.setle()
	if o, _ := l.e.s.D.Objekt(id); !document.Gleich(o, vorher) {
		t.Error("das Objekt hat sich hinter dem Menü bewegt")
	}
}

// Kritik K-6 #313a: Ein zweiter Rechtsklick bei offenem Kontextmenü öffnet es am neuen Punkt für das neue Ziel.
func TestKontextmenueZweiterRechtsklick(t *testing.T) {
	l, id, p := kontextLauf(t)
	l.rechtsklick(l.fensterVon(document.Pt(36, 1)))
	if !l.e.menue.offen || l.e.kontext.objekt {
		t.Fatal("Kontrolle: Menü für das Bild nicht offen")
	}
	l.rechtsklick(p)
	if !l.e.menue.offen || l.e.s.Auswahl != id || !l.e.kontext.objekt || l.e.kontext.anker.Min != p {
		t.Errorf("offen %v, Auswahl %d (Soll %d), objekt %v, Anker %v (Soll %v)", l.e.menue.offen, l.e.s.Auswahl, id,
			l.e.kontext.objekt, l.e.kontext.anker.Min, p)
	}
}
