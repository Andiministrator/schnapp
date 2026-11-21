// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"errors"
	"image"
	"testing"
	"time"

	"gioui.org/io/key"
	"gioui.org/io/system"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
)

// zaehleSchliessen hängt die Naht e.perform ein und zählt die Schließen-Anfragen (ohne Fenster, F-299).
func zaehleSchliessen(l *menueLauf) *int {
	n := new(int)
	l.e.perform = func(a system.Action) {
		if a == system.ActionClose {
			*n++
		}
	}
	return n
}

// F-299: Strg+W ist eine Schließen-Anfrage wie der Fensterknopf und geht auch während des Speicherns weiter
// (ADR-0020 Nachtrag 6); SchliessenAnfrage merkt sie vor.
func TestStrgWWaehrendSpeichern(t *testing.T) {
	i18n.Setze(nil)

	// Kontrolle: ohne Speichern erreicht Strg+W die Handlung.
	k := neuerMenueLauf(t, image.Pt(1200, 800))
	kn := zaehleSchliessen(k)
	k.fokusiere(k.e)
	k.taste("W", key.ModShortcut)
	if *kn != 1 {
		t.Fatalf("Kontrolle: Strg+W ohne Speichern löst %d Schließen-Anfragen aus, erwartet 1", *kn)
	}

	l := neuerMenueLauf(t, image.Pt(1200, 800))
	n := zaehleSchliessen(l)
	v := neuerSperrSpeicher()
	defer close(v.frei)
	l.e.speichern = v.speichern
	l.fokusiere(l.e)
	l.taste("S", key.ModShortcut)
	if !l.e.s.SpeichernLaeuft() {
		t.Fatal("Speichern läuft nicht")
	}
	stand := l.e.s.D.Stand()
	l.taste("W", key.ModShortcut)
	if *n != 1 {
		t.Errorf("Strg+W während des Speicherns: %d Schließen-Anfragen, erwartet 1 (Meldung %q)", *n, l.e.meldung.Text)
	}
	// Die Anfrage selbst bleibt vorgemerkt: Das Fenster schließt nicht sofort, solange das Speichern läuft.
	if l.e.schliessen() {
		t.Error("Schließen-Anfrage während des Speicherns schließt sofort")
	}
	// Keine andere Handlung rutscht durch: Strg+Z ändert während des Speicherns nichts.
	l.taste("Z", key.ModShortcut)
	if l.e.s.D.Stand() != stand {
		t.Errorf("Strg+Z wirkt während des Speicherns: Stand %d, vorher %d", l.e.s.D.Stand(), stand)
	}
	if *n != 1 {
		t.Errorf("Strg+Z löste eine Schließen-Anfrage aus: %d", *n)
	}
}

// F-299: Enter auf dem fokussierten Menüeintrag „Schließen“ wirkt während des Speicherns, Enter auf einem anderen
// Eintrag nicht.
func TestMenueEnterSchliessenWaehrendSpeichern(t *testing.T) {
	i18n.Setze(nil)
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	n := zaehleSchliessen(l)
	v := neuerSperrSpeicher()
	defer close(v.frei)
	l.e.speichern = v.speichern
	l.oeffneMitTaste()
	z, abzug := l.e.s.SchliessenAnfrage()
	if z != canvas.SchlussWarten || abzug == nil {
		t.Fatal("kein Speichern")
	}
	l.e.speichernStarten(abzug)
	l.fokusiere(&l.e.menue.eintraege[l.index(aktRueckgaengig)])
	l.taste(key.NameReturn, 0)
	if *n != 0 {
		t.Fatalf("Enter auf „Rückgängig“ löst %d Schließen-Anfragen aus", *n)
	}
	l.fokusiere(&l.e.menue.eintraege[l.index(aktSchliessen)])
	l.taste(key.NameReturn, 0)
	if *n != 1 {
		t.Errorf("Enter auf „Schließen“ während des Speicherns: %d Schließen-Anfragen, erwartet 1", *n)
	}
}

// F-307: Enter auf einem anderen Menüeintrag oder einem fokussierten Leistenknopf bei offenem Menü ändert während des
// Speicherns nichts: Stand und Inhalt des Dokuments bleiben, es gibt keine Schließen-Anfrage. Gemessen wird die
// Wirkung, nicht die Zahl der Anfragen. Zwei Sperren tragen das gemeinsam: der Wächter in `tasten`
// (`c == schliessenEintrag()`) und `an()` in `menueAuswerten`/`knoepfeAuswerten`; die Tests belegen das Ergebnis,
// nicht, welche der beiden es hält (der Wächter allein ist von außen nicht prüfbar, siehe Bericht zu F-307).
func TestEnterWaehrendSpeichernAendertNichts(t *testing.T) {
	i18n.Setze(nil)
	faelle := map[string]func(l *menueLauf) any{
		"Menüeintrag Rückgängig": func(l *menueLauf) any { return &l.e.menue.eintraege[l.index(aktRueckgaengig)] },
		"Menüeintrag Wiederholen": func(l *menueLauf) any {
			return &l.e.menue.eintraege[l.index(aktWiederholen)]
		},
		"Leistenknopf Rückgängig bei offenem Menü": func(l *menueLauf) any { return &l.e.knoepfe.rueck },
		"Leistenknopf Speichern bei offenem Menü":  func(l *menueLauf) any { return &l.e.knoepfe.speichern },
	}
	for name, ziel := range faelle {
		l := neuerMenueLauf(t, image.Pt(1200, 800))
		n := zaehleSchliessen(l)
		v := neuerSperrSpeicher()
		l.e.speichern = v.speichern
		l.oeffneMitTaste()
		z, abzug := l.e.s.SchliessenAnfrage()
		if z != canvas.SchlussWarten || abzug == nil {
			t.Fatalf("%s: kein Speichern", name)
		}
		l.e.speichernStarten(abzug)
		if !l.e.s.SpeichernLaeuft() {
			t.Fatalf("%s: Speichern läuft nicht", name)
		}
		stand, inhalt := l.e.s.D.Stand(), l.inhalt()
		l.fokusiere(ziel(l))
		l.taste(key.NameReturn, 0)
		if got := l.e.s.D.Stand(); got != stand {
			t.Errorf("%s: Enter änderte den Stand von %d auf %d", name, stand, got)
		}
		if got := l.inhalt(); got != inhalt {
			t.Errorf("%s: Enter änderte den Inhalt von %q auf %q", name, inhalt, got)
		}
		if *n != 0 {
			t.Errorf("%s: %d Schließen-Anfragen", name, *n)
		}
		close(v.frei)
	}
}

// F-300: Scheitert das Speichern einer Schließen-Anfrage, und das Fenster wird zerstört, bevor ein Bild den Fehler
// gezeigt hat, geht der Fehler in die Rückgabe von zerstoert, statt still verworfen zu werden.
func TestSchliessenFehlerBeimZerstoeren(t *testing.T) {
	e := testEditor(t)
	if _, err := e.s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(1, 1), document.Pt(5, 5))}); err != nil {
		t.Fatal(err)
	}
	v := neuerSperrSpeicher()
	v.fehler = []error{errors.New("platte voll")}
	e.speichern = v.speichern
	if e.schliessen() {
		t.Fatal("Schließen-Anfrage mit Änderung schließt sofort")
	}
	go func() { time.Sleep(20 * time.Millisecond); close(v.frei) }()
	err := e.zerstoert(nil)
	if err == nil {
		t.Fatalf("Speichern beim Schließen gescheitert, Fenster zerstört: Rückgabe nil (Aufrufe %d)", v.aufrufe.Load())
	}
	if !errors.Is(err, v.fehler[0]) {
		t.Errorf("Rückgabe trägt den Speicherfehler nicht: %v", err)
	}
	if n := v.aufrufe.Load(); n != 1 {
		t.Errorf("Speicherfunktion %d-mal aufgerufen, erwartet 1", n)
	}
}

// F-300, Gegenprobe: Hat ein Bild den Fehler schon gezeigt (Antwort über speichernFertig), und der Nutzer schließt
// erneut, gilt das als Verwerfen — Rückgabe nil, kein zweiter Speicherversuch.
func TestSchliessenFehlerGezeigtDannVerwerfen(t *testing.T) {
	e := testEditor(t)
	if _, err := e.s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(1, 1), document.Pt(5, 5))}); err != nil {
		t.Fatal(err)
	}
	v := neuerSperrSpeicher()
	close(v.frei)
	v.fehler = []error{errors.New("platte voll")}
	e.speichern = v.speichern
	if e.schliessen() {
		t.Fatal("Schließen-Anfrage mit Änderung schließt sofort")
	}
	ende := time.Now().Add(5 * time.Second)
	for !e.speichernAbholen() {
		if time.Now().After(ende) {
			t.Fatal("keine Antwort des Speicherns")
		}
		time.Sleep(time.Millisecond)
	}
	if e.schliessen() != true {
		t.Fatal("zweite Schließen-Anfrage nach angezeigtem Fehler verwirft nicht")
	}
	if err := e.zerstoert(nil); err != nil {
		t.Errorf("Verwerfen nach angezeigtem Fehler liefert einen Fehler: %v", err)
	}
	if n := v.aufrufe.Load(); n != 1 {
		t.Errorf("Speicherfunktion %d-mal aufgerufen, erwartet 1", n)
	}
}
