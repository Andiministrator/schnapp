// SPDX-License-Identifier: GPL-3.0-or-later

package main

// Tests aus Verify 7 (Fable, V7-8, F-385): die Verdrahtung der Rettungskopie im Programm (--ignore-limit, neue
// Ausgangsdatei nach dem Speichern, Rettung auch nach einer Aufnahme, Aufräumen beim Start). Die Mutationsprobe
// scripts/mutanten/rescue.json prüft, dass sie es tragen.

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/render"
	"github.com/Andiministrator/schnapp/internal/rescue"
	"github.com/Andiministrator/schnapp/internal/ui"
)

// gestartet hält, was der (nicht gestartete) Editor bekommt.
type gestartet struct {
	speichern ui.Speichern
	rettung   ui.Rettung
	dok       *document.Dokument
	geoeffnet ui.Geoeffnet  // alles, was der Editor bekam (ADR-0036)
	bib       *ui.Anbindung // Anbindung an die Bibliothek für Leiste und Wechsel
}

// rettungsLauf richtet eine eigene Umgebung ein (HOME, Zustandsordner) und ersetzt das Fenster durch eine Attrappe.
// Zurück kommen der Ordner der Rettungskopien, eine Datei von außen (40 x 30) und die Attrappe.
func rettungsLauf(t *testing.T) (ort, aussen string, g *gestartet) {
	t.Helper()
	i18n.Setze(nil)
	heim := t.TempDir()
	t.Setenv("HOME", heim)
	for _, name := range []string{"XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME"} {
		t.Setenv(name, "")
	}
	staat := t.TempDir()
	alt := rettungUmgebung
	rettungUmgebung = func(k string) string {
		if k == "XDG_STATE_HOME" {
			return staat
		}
		return os.Getenv(k)
	}
	t.Cleanup(func() { rettungUmgebung = alt })
	d0, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 40, 30)))
	if err != nil {
		t.Fatal(err)
	}
	aussen = schreibeDatei(t, t.TempDir(), "bild.png", d0)
	g = &gestartet{}
	altStart := editorStarten
	editorStarten = func(o ui.Geoeffnet, bib *ui.Anbindung) error {
		g.speichern, g.rettung, g.dok = o.Speichern, o.Rettung, o.Dokument
		g.geoeffnet, g.bib = o, bib
		return nil
	}
	t.Cleanup(func() { editorStarten = altStart })
	// Die zuletzt gestartete Rettung beenden, bevor die Testordner verschwinden: Ihre Aufträge laufen in Goroutinen
	// (F-417), ihre Sperre hält eine Datei offen (F-413).
	t.Cleanup(func() {
		if g.rettung != nil {
			g.rettung.Ende()
		}
	})
	return filepath.Join(staat, "schnapp", "rescue"), aussen, g
}

// C3, C4: Nach einem gelungenen Speichern ist die gespeicherte Datei (hier eine neue Aufnahme der Bibliothek) die neue
// Ausgangsdatei: Die Kopie der alten verschwindet (sobald die Kette der Aufträge abgearbeitet ist, F-417), die nächste
// Kopie liegt unter dem Namen der neuen.
func TestRettungNeueBasisNachSpeichern(t *testing.T) {
	ort, aussen, g := rettungsLauf(t)
	if err := oeffneImEditor(aussen, false, document.KanteAbriss); err != nil {
		t.Fatal(err)
	}
	mitKasten(t, g.dok)
	g.rettung.Sichere(g.dok)
	g.rettung.Ende()
	alt := filepath.Join(ort, rescue.Name(aussen))
	if _, err := os.Stat(alt); err != nil {
		t.Fatalf("Vorbedingung: Kopie fehlt: %v", err)
	}
	if _, err := g.speichern(g.dok); err != nil {
		t.Fatal(err)
	}
	g.rettung.Ende() // NeueBasis löscht in der Kette (F-417)
	if _, err := os.Stat(alt); !os.IsNotExist(err) {
		t.Errorf("Kopie der alten Ausgangsdatei nach dem Speichern noch da: %v", err)
	}
	mitKasten(t, g.dok)
	g.rettung.Sichere(g.dok)
	g.rettung.Ende()
	eintraege, _ := os.ReadDir(ort)
	if len(eintraege) != 1 || eintraege[0].Name() == rescue.Name(aussen) {
		t.Errorf("nach dem Speichern: %v, erwartet eine Kopie unter dem Namen der neuen Aufnahme", eintraege)
	}
}

// C9: Lässt sich die gespeicherte Datei nicht lesen, ist sie trotzdem die neue Ausgangsdatei (mit einer Kennung, die
// nie passt); die Kopie bekommt ihren Namen, nicht den eines leeren Pfads.
func TestRettungNachSpeichernUnlesbar(t *testing.T) {
	_, aussen, _ := rettungsLauf(t)
	r := rettungFuer(aussen, false)
	if r == nil {
		t.Fatal("keine Rettung")
	}
	weg := filepath.Join(t.TempDir(), "fehlt.png")
	r.nachSpeichern()(weg)
	r.Ende() // NeueBasis arbeitet in der Kette (F-417): vor dem Aufräumen des Testordners abwarten
	if b := r.s.Basis(); b.Pfad != weg || b.SHA256 != "" {
		t.Errorf("Basis %+v, erwartet Pfad %s ohne Prüfsumme", b, weg)
	}
}

// C1, C2, C5: Eine Rettungskopie über dem Rechenbudget wird ohne --ignore-limit nicht angewendet (render.Pruefe), mit
// dem Schalter schon.
func TestRettungIgnoreLimit(t *testing.T) {
	ort, aussen, g := rettungsLauf(t)
	k, err := rescue.KennungVon(aussen)
	if err != nil {
		t.Fatal(err)
	}
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 40, 30)))
	if err != nil {
		t.Fatal(err)
	}
	budget := render.GlyphBudget
	n := int(budget/(4096*4096)) + 1 // wie ueberBudget
	for i := 0; i < n; i++ {
		if _, err := d.Hinzufuegen(document.Text{Ort: wOrt, Inhalt: "W", Schriftgrad: 4096,
			Farbe: color.NRGBA{R: 255, A: 255}}); err != nil {
			t.Fatal(err)
		}
	}
	kp, err := rescue.Erzeuge(d, k, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, fall := range []struct {
		ohneGrenze bool
		objekte    int
	}{{false, 0}, {true, n}} {
		if err := rescue.Schreibe(ort, kp); err != nil {
			t.Fatal(err)
		}
		if err := oeffneImEditor(aussen, fall.ohneGrenze, document.KanteAbriss); err != nil {
			t.Fatal(err)
		}
		text, fehler := g.rettung.Wiederherstellen(g.dok)
		if got := len(g.dok.Objekte()); got != fall.objekte {
			t.Errorf("--ignore-limit %v: %d Objekte, erwartet %d (Meldung %q, Fehler %v)", fall.ohneGrenze, got,
				fall.objekte, text, fehler)
		}
		g.rettung.Ende() // Fenster zu: Sperre frei für den nächsten Durchgang (F-413)
	}
}

// F-413: Ein zweites Fenster auf derselben Datei sagt in der Statusleiste, dass es keine Rettungskopie führt, und lässt
// die des ersten liegen.
func TestRettungZweitesFenster(t *testing.T) {
	ort, aussen, g := rettungsLauf(t)
	if err := oeffneImEditor(aussen, false, document.KanteAbriss); err != nil {
		t.Fatal(err)
	}
	erstes := g.rettung
	mitKasten(t, g.dok)
	erstes.Sichere(g.dok)
	if err := oeffneImEditor(aussen, false, document.KanteAbriss); err != nil {
		t.Fatal(err)
	}
	text, fehler := g.rettung.Wiederherstellen(g.dok)
	if !strings.Contains(text, "another Schnapp window") || fehler || len(g.dok.Objekte()) != 0 {
		t.Errorf("zweites Fenster: %q (Fehler %v), %d Objekte", text, fehler, len(g.dok.Objekte()))
	}
	g.rettung.Verwirf()
	g.rettung.Ende()
	erstes.Ende()
	if _, err := os.Stat(filepath.Join(ort, rescue.Name(aussen))); err != nil {
		t.Errorf("Kopie des ersten Fensters weg: %v", err)
	}
}

// C6: Auch die eben abgelegte Aufnahme (bearbeiteNeu) bekommt eine Rettungskopie.
func TestBearbeiteNeuMitRettung(t *testing.T) {
	_, _, g := rettungsLauf(t)
	orte, err := library.OrteAus(os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	b := library.Neu(orte)
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 8, 8)))
	if err != nil {
		t.Fatal(err)
	}
	e, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{})
	if err != nil {
		t.Fatal(err)
	}
	abs := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
	if err := bearbeiteNeu(abgelegt{pfad: abs, d: d, stand: standVon(t, abs)}); err != nil {
		t.Fatal(err)
	}
	if g.rettung == nil {
		t.Error("nach einer Aufnahme keine Rettung")
	}
}

// C7: Beim Start räumt Schnapp Rettungskopien über der Haltezeit auf.
func TestRettungRaeumtBeimStartAuf(t *testing.T) {
	ort, aussen, _ := rettungsLauf(t)
	if err := os.MkdirAll(ort, 0o700); err != nil {
		t.Fatal(err)
	}
	alt := filepath.Join(ort, rescue.Name(filepath.Join(t.TempDir(), "weg.png")))
	if err := os.WriteFile(alt, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	z := time.Now().Add(-rescue.Haltezeit - time.Hour)
	if err := os.Chtimes(alt, z, z); err != nil {
		t.Fatal(err)
	}
	if err := oeffneImEditor(aussen, false, document.KanteAbriss); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(alt); !os.IsNotExist(err) {
		t.Errorf("alte Kopie nicht aufgeräumt: %v", err)
	}
}
