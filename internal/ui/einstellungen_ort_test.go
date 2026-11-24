// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"errors"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gioui.org/io/key"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// schreibeIni legt die Datei pfad samt Ordner mit inhalt an.
func schreibeIni(t *testing.T, pfad, inhalt string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(pfad), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pfad, []byte(inhalt), 0o600); err != nil {
		t.Fatal(err)
	}
}

func leseIni(t *testing.T, pfad string) string {
	t.Helper()
	b, err := os.ReadFile(pfad)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// ADR-0034 Nachtrag 1 (F-625): Weicht XDG_CONFIG_HOME vom Ort ab, den `schnapp mcp` liest (<Heim der
// Benutzerdatenbank>/.config/schnapp/settings.ini), liest und schreibt der Reiter Agents die Datei des Servers, die
// übrigen Reiter die aus XDG_CONFIG_HOME; der Reiter Agents sagt es mit beiden Orten.
func TestTafelAgentenOrt(t *testing.T) {
	heim := t.TempDir()
	setzeHeim(t, heim)
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	server, err := settings.AgentenPfad()
	if err != nil || server != filepath.Join(heim, ".config", "schnapp", "settings.ini") {
		t.Fatalf("AgentenPfad: %q %v", server, err)
	}
	eigene := filepath.Join(xdg, "schnapp", "settings.ini")
	schreibeIni(t, server, "[agents]\ncapture = yes\nlibrary = yes\n")
	schreibeIni(t, eigene, "[agents]\ncapture = no\n[capture]\nseam = wave\n")

	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.fokusiere(l.e)
	l.taste(",", key.ModShortcut)
	tf := &l.e.einst
	if !tf.wert.Agenten.Aufnehmen || !tf.wert.Agenten.Bibliothek || tf.wert.Kante != document.KanteWelle {
		t.Fatalf("geladen: Agenten %+v, Kante %v", tf.wert.Agenten, tf.wert.Kante)
	}
	if !tf.rahmen.hinweis.Empty() {
		t.Error("Hinweis zum Ort außerhalb des Reiters Agents")
	}
	l.klick(mitte(tf.rahmen.reiter[reiterAgenten]))
	h := tf.agentenHinweis()
	if !strings.Contains(h, server) || !strings.Contains(h, eigene) {
		t.Errorf("Hinweis nennt nicht beide Orte: %q", h)
	}
	if tf.rahmen.hinweis.Empty() {
		t.Error("Hinweis zum Ort nicht gezeichnet")
	}
	ag := zeilenDes(reiterAgenten)
	if tf.rahmen.hinweis.Max.Y > tf.rahmen.feld[ag[0]].Min.Y {
		t.Errorf("Hinweis überdeckt die Felder: %v %v", tf.rahmen.hinweis, tf.rahmen.feld[ag[0]])
	}
	l.klick(mitte(tf.rahmen.feld[ag[0]])) // Capture screen: aus
	tf.waehle(3, 2)                       // Kante: Zacken (Reiter Aufnahme, XDG-Datei)
	l.klick(mitte(tf.rahmen.ok))
	if tf.offen || l.e.meldung.Text != "Settings saved" {
		t.Fatalf("Save: offen %v, Meldung %q, Fehler %q", tf.offen, l.e.meldung.Text, tf.fehler)
	}
	// Die Ladefunktion des Servers (cmd/schnapp mitHeim: settings.LadeDatei(settings.AgentenPfadIn(heim))) sieht den Wert.
	if e, warn := settings.LadeDatei(server); e.Agenten.Aufnehmen || !e.Agenten.Bibliothek || len(warn) > 0 {
		t.Errorf("Server sieht: %+v %v", e.Agenten, warn)
	}
	if s := leseIni(t, server); strings.Contains(s, "seam") {
		t.Errorf("Datei des Servers trägt die Kante:\n%s", s)
	}
	if s := leseIni(t, eigene); s != "[agents]\ncapture = no\n[capture]\nseam = zigzag\n" {
		t.Errorf("XDG-Datei:\n%s", s)
	}

	// Wieder geöffnet: Der Reiter zeigt, was für den Server gilt.
	l.taste(",", key.ModShortcut)
	if tf.wert.Agenten.Aufnehmen || !tf.wert.Agenten.Bibliothek || tf.wert.Kante != document.KanteZacken {
		t.Errorf("neu geöffnet: %+v %v", tf.wert.Agenten, tf.wert.Kante)
	}
	l.taste(key.NameEscape, 0)
}

// Scheitert die Datei des Servers, schreibt die Tafel auch die andere nicht, bleibt offen und zeigt den Fehler.
func TestTafelAgentenOrtFehler(t *testing.T) {
	heim := t.TempDir()
	setzeHeim(t, heim)
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	server, _ := settings.AgentenPfad()
	if err := os.MkdirAll(server, 0o700); err != nil { // ein Ordner statt der Datei
		t.Fatal(err)
	}
	eigene := filepath.Join(xdg, "schnapp", "settings.ini")
	tf := &tafel{}
	tf.oeffne(settings.Voreinstellungen(), nil, eigene, nil)
	tf.agentenOrt(server, settings.LadeDatei)
	if tf.agentenDatei != server {
		t.Fatalf("agentenDatei %q", tf.agentenDatei)
	}
	tf.schalte(zeilenDes(reiterAgenten)[0])
	tf.schalte(2) // open-editor
	if geschrieben, ok := tf.speichere(settings.SchreibeDatei); ok || geschrieben || !strings.Contains(tf.fehler, "not a regular file") {
		t.Errorf("speichern: %v %v %q", geschrieben, ok, tf.fehler)
	}
	if _, err := os.Stat(eigene); !os.IsNotExist(err) {
		t.Errorf("XDG-Datei trotz Fehler geschrieben: %v", err)
	}
}

// Zeigt XDG_CONFIG_HOME über einen Verweis auf <heim>/.config, ist es dieselbe Datei: kein Hinweis, eine Datei.
// Kennt die Benutzerdatenbank kein Heim, schreibt der Reiter Agents nach XDG und sagt, dass Agenten die
// Voreinstellungen behalten.
func TestTafelAgentenOrtGleichOderUnbekannt(t *testing.T) {
	heim := t.TempDir()
	setzeHeim(t, heim)
	if err := os.MkdirAll(filepath.Join(heim, ".config", "schnapp"), 0o700); err != nil {
		t.Fatal(err)
	}
	verweis := filepath.Join(t.TempDir(), "konfig")
	if err := os.Symlink(filepath.Join(heim, ".config"), verweis); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", verweis)
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.fokusiere(l.e)
	l.taste(",", key.ModShortcut)
	tf := &l.e.einst
	if tf.agentenDatei != "" || tf.agentenHinweis() != "" {
		t.Errorf("gleiche Datei über Verweis: %q %q", tf.agentenDatei, tf.agentenHinweis())
	}
	l.taste(key.NameEscape, 0)

	settings.BenutzerHeim = func() (string, error) { return "", errors.New("unbekannt") }
	l.taste(",", key.ModShortcut)
	if tf.agentenDatei != "" || !tf.agentenOhneOrt || !strings.Contains(tf.agentenHinweis(), "defaults") {
		t.Errorf("ohne Heim: %q %v %q", tf.agentenDatei, tf.agentenOhneOrt, tf.agentenHinweis())
	}
	tf.schalte(zeilenDes(reiterAgenten)[0])
	if _, ok := tf.speichere(settings.SchreibeDatei); !ok {
		t.Fatalf("ohne Heim: %q", tf.fehler)
	}
	if s := leseIni(t, filepath.Join(verweis, "schnapp", "settings.ini")); !strings.Contains(s, "capture = yes") {
		t.Errorf("XDG-Datei:\n%s", s)
	}
}

// F-626: Ein ungültiges Zahlfeld speichert nicht — leer getippt (90, zweimal Rücktaste) oder zu groß (40000): Enter
// lässt die Datei, wie sie ist, die Tafel bleibt offen und der Fuß nennt den Fehler. Verlässt man das Feld, gilt der
// Wert vor dem Tippen, kein Zwischenstand.
func TestTafelZahlUngueltigSpeichertNicht(t *testing.T) {
	alt := "[library]\ncleanup = true\ncleanup-days = 90\n"
	l, p := tafelLauf(t, alt)
	tf := &l.e.einst
	zumFeld := func() {
		t.Helper()
		l.taste(",", key.ModShortcut)
		tf.waehleReiter(reiterBibliothek)
		l.setle()
		l.taste(key.NameTab, 0) // vom Schalter bar über cleanup zur Zahl (ADR-0036 Punkt 2)
		l.taste(key.NameTab, 0)
		if tf.ziel() != zahlZeile() {
			t.Fatalf("Ziel %d", tf.ziel())
		}
	}
	zumFeld()
	l.taste(key.NameDeleteBackward, 0)
	l.taste(key.NameDeleteBackward, 0)
	if !tf.zahlFehler || tf.zahlAnzeige() != "" || tf.wert.Bibliothek.Tage != 90 {
		t.Errorf("leer: Fehler %v, Anzeige %q, Tage %d", tf.zahlFehler, tf.zahlAnzeige(), tf.wert.Bibliothek.Tage)
	}
	l.taste(key.NameReturn, 0)
	if !tf.offen || !strings.Contains(tf.gezeigteFehler, "must be a number from 1 to") {
		t.Errorf("Enter bei leerem Feld: offen %v, Fuß %q", tf.offen, tf.gezeigteFehler)
	}
	if l.e.meldung.Text == "Settings saved" || leseIni(t, p) != alt {
		t.Errorf("leer gespeichert: %q\n%s", l.e.meldung.Text, leseIni(t, p))
	}
	l.klick(mitte(tf.rahmen.ok))
	if !tf.offen || leseIni(t, p) != alt {
		t.Error("Klick auf Save bei leerem Feld speichert")
	}
	// Feld verlassen, dann speichern: 90 bleibt, nicht 9.
	l.taste(key.NameTab, 0)
	l.taste(key.NameTab, 0)
	l.taste(key.NameReturn, 0)
	if tf.offen || leseIni(t, p) != alt {
		t.Errorf("nach dem Verlassen: offen %v, Datei\n%s", tf.offen, leseIni(t, p))
	}

	zumFeld()
	l.tippe("40000")
	if !tf.zahlFehler || tf.wert.Bibliothek.Tage != 90 {
		t.Errorf("40000: Fehler %v, Tage %d", tf.zahlFehler, tf.wert.Bibliothek.Tage)
	}
	l.taste(key.NameReturn, 0)
	if !tf.offen || leseIni(t, p) != alt {
		t.Errorf("40000 gespeichert: offen %v\n%s", tf.offen, leseIni(t, p))
	}
	l.taste(key.NameEscape, 0)
}
