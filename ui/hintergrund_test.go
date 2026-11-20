// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"testing"
	"time"

	"gioui.org/io/key"
	"gioui.org/layout"

	"github.com/Andiministrator/schnapp/document"
	"github.com/Andiministrator/schnapp/i18n"
)

// warteSpeichern wartet auf die Antwort eines Speicherns im Hintergrund und wertet sie aus wie die Ereignisschleife.
func warteSpeichern(t *testing.T, e *editor) {
	t.Helper()
	select {
	case r := <-e.speicherAntwort:
		e.speichernFertig(r)
	case <-time.After(10 * time.Second):
		t.Fatal("keine Antwort des Speicherns")
	}
}

// warteKopieren wartet auf die Antwort eines Kopierens im Hintergrund und wertet sie aus wie ein Bild.
func warteKopieren(t *testing.T, e *editor, gtx layout.Context) {
	t.Helper()
	select {
	case r := <-e.kopierAntwort:
		e.kopierAntwort <- r // kopierenAbholen nimmt sie wie im Bild
		e.kopierenAbholen(gtx)
	case <-time.After(10 * time.Second):
		t.Fatal("keine Antwort des Kopierens")
	}
}

// F-143, ADR-0020 Nachtrag 6: Strg+S speichert einen Abzug im Hintergrund. Solange es läuft, zeichnet das Fenster
// weiter, aber Tasten, Klicks und getippter Text wirken nicht; danach gilt der Stand als gespeichert, und die
// Leinwand hat den Tastaturfokus noch (Entf wirkt wieder).
func TestSpeichernImHintergrundSperrtBearbeiten(t *testing.T) {
	i18n.Setze(nil)
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	erhalten := make(chan *document.Dokument, 1)
	frei := make(chan struct{})
	l.e.speichern = func(d *document.Dokument) (Gespeichert, error) {
		erhalten <- d
		<-frei
		return Gespeichert{}, nil
	}
	l.fokusiere(l.e)
	if !l.e.s.Geaendert() {
		t.Fatal("Vorbedingung: Dokument nicht geändert")
	}
	stand := l.e.s.D.Stand()
	l.taste("S", key.ModShortcut)
	var abzug *document.Dokument
	select {
	case abzug = <-erhalten:
	case <-time.After(10 * time.Second):
		t.Fatal("Speichern nicht begonnen")
	}
	if abzug == l.e.s.D || abzug.Stand() != stand || !l.e.s.SpeichernLaeuft() {
		t.Fatalf("Abzug %p (Dokument %p), Stand %d/%d, läuft %v", abzug, l.e.s.D, abzug.Stand(), stand, l.e.s.SpeichernLaeuft())
	}
	if l.e.meldung.Text != i18n.T("Saving…") {
		t.Errorf("Meldung während des Speicherns %q", l.e.meldung.Text)
	}

	// Während des Speicherns: Entf, Strg+Z, ein Werkzeugbuchstabe, getippter Text und ein zweites Strg+S wirken nicht.
	werkzeug := l.e.s.Werkzeug
	l.taste(key.NameDeleteForward, 0)
	l.taste("Z", key.ModShortcut)
	l.taste("T", 0)
	l.tippe("xyz")
	l.taste("S", key.ModShortcut)
	if l.inhalt() != "Hallo" || l.e.s.D.Stand() != stand || l.e.s.Werkzeug != werkzeug {
		t.Errorf("Bearbeitung während des Speicherns: Inhalt %q, Stand %d (Soll %d), Werkzeug %v", l.inhalt(),
			l.e.s.D.Stand(), stand, l.e.s.Werkzeug)
	}
	select {
	case <-erhalten:
		t.Error("zweites Speichern während des ersten")
	default:
	}

	close(frei)
	// Die Antwort kommt über den Kanal; ein Bild holt sie ab.
	ende := time.Now().Add(10 * time.Second)
	for l.e.s.SpeichernLaeuft() && time.Now().Before(ende) {
		l.bild()
		time.Sleep(time.Millisecond)
	}
	if l.e.s.SpeichernLaeuft() || l.e.s.Geaendert() || l.e.meldung.Text != i18n.T("Saved to the library") {
		t.Fatalf("nach der Antwort: läuft %v, geändert %v, Meldung %q", l.e.s.SpeichernLaeuft(), l.e.s.Geaendert(),
			l.e.meldung.Text)
	}
	if !l.fokus(l.e) {
		t.Error("Leinwand hat den Fokus verloren")
	}
	l.taste(key.NameDeleteForward, 0)
	if l.inhalt() != "<weg>" {
		t.Errorf("Entf nach dem Speichern: %q", l.inhalt())
	}
}

// Das Speichern bekommt einen Abzug: Was nach dem Start am Dokument geschieht, erreicht die gespeicherte Fassung nicht.
func TestSpeichernImHintergrundMitAbzug(t *testing.T) {
	e := testEditor(t)
	if _, err := e.s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(1, 1), document.Pt(5, 5))}); err != nil {
		t.Fatal(err)
	}
	erhalten := make(chan int, 1)
	frei := make(chan struct{})
	e.speichern = func(d *document.Dokument) (Gespeichert, error) {
		<-frei
		erhalten <- len(d.Objekte())
		return Gespeichert{}, nil
	}
	e.speichernJetzt()
	if _, err := e.s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(6, 6), document.Pt(9, 9))}); err != nil {
		t.Fatal(err)
	}
	close(frei)
	if n := <-erhalten; n != 1 {
		t.Errorf("gespeicherte Fassung hat %d Objekte (Soll 1)", n)
	}
	warteSpeichern(t, e)
	if !e.s.Geaendert() {
		t.Error("späterer Kasten gilt als gespeichert")
	}
}

// F-178: Kopieren läuft im Hintergrund; ein zweites Strg+C, solange das erste läuft, wartet nicht und sagt es.
func TestKopierenImHintergrund(t *testing.T) {
	t.Setenv("WAYLAND_SOCKET", "1") // der Halter startet keinen Prozess; das Fenster böte selbst an
	e := testEditor(t)
	gtx := layout.Context{}
	e.kopieren(gtx)
	if !e.kopierLaeuft {
		t.Fatal("Kopieren läuft nicht")
	}
	e.kopieren(gtx)
	if e.meldung.Text != i18n.T("Copying is already in progress.") {
		t.Errorf("zweites Kopieren: %q", e.meldung.Text)
	}
	warteKopieren(t, e, gtx)
	if e.kopierLaeuft || e.meldung.Text != i18n.T("Copied to clipboard") {
		t.Errorf("nach der Antwort: läuft %v, Meldung %q", e.kopierLaeuft, e.meldung.Text)
	}
}
