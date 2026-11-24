// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// eigeneEinstellungen legt einen eigenen Ordner für settings.ini an (XDG_CONFIG_HOME) und liefert den Pfad der Datei;
// inhalt "" legt keine Datei an. Das Heim der Benutzerdatenbank (settings.BenutzerHeim) zeigt so, dass der MCP-Server
// dieselbe Datei liest (<heim>/.config = XDG_CONFIG_HOME).
func eigeneEinstellungen(t *testing.T, inhalt string) string {
	t.Helper()
	heim := t.TempDir()
	setzeHeim(t, heim)
	d := filepath.Join(heim, ".config")
	t.Setenv("XDG_CONFIG_HOME", d)
	p := filepath.Join(d, "schnapp", "settings.ini")
	if inhalt != "" {
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(inhalt), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

// setzeHeim setzt das Heim der Benutzerdatenbank für den Test.
func setzeHeim(t *testing.T, heim string) {
	t.Helper()
	alt := settings.BenutzerHeim
	t.Cleanup(func() { settings.BenutzerHeim = alt })
	settings.BenutzerHeim = func() (string, error) { return heim, nil }
}

// offeneTafel liefert eine geöffnete Tafel mit den Werten aus inhalt (ohne Editor).
func offeneTafel(t *testing.T, inhalt string) *tafel {
	t.Helper()
	p := eigeneEinstellungen(t, inhalt)
	w, warn := settings.LadeDatei(p)
	tf := &tafel{}
	tf.oeffne(w, warn, p, []string{"de", "en"})
	return tf
}

// ADR-0034 Punkt 2: Jeder Schlüssel des Lesers hat genau eine Zeile, in der Reihenfolge von settings.Werte, mit
// Beschriftung; jeder Reiter hat Zeilen; jede Auswahl kennt ihren aktuellen Wert.
func TestTafelDecktAlleSchluessel(t *testing.T) {
	testEditor(t) // Ausgangssprache
	w := settings.Werte(settings.Voreinstellungen())
	if len(w) != len(tafelZeilen) {
		t.Fatalf("Tafel hat %d Zeilen, der Leser %d Schlüssel", len(tafelZeilen), len(w))
	}
	tf := &tafel{}
	tf.oeffne(settings.Voreinstellungen(), nil, "", tafelSprachen())
	for i, a := range w {
		z := tafelZeilen[i]
		if z.abschnitt != a.Abschnitt || z.schluessel != a.Schluessel {
			t.Errorf("Zeile %d: %s.%s, Leser %s.%s", i, z.abschnitt, z.schluessel, a.Abschnitt, a.Schluessel)
		}
		if zeileTitel(i) == "" {
			t.Errorf("%s.%s ohne Beschriftung", z.abschnitt, z.schluessel)
		}
		if z.art == artAuswahl && tf.gewaehlt(i) < 0 {
			t.Errorf("%s.%s: Voreinstellung %q fehlt in der Auswahl", z.abschnitt, z.schluessel, a.Wert)
		}
		if z.art == artSchalter && a.Wert != "true" && a.Wert != "false" && a.Wert != "yes" && a.Wert != "no" {
			t.Errorf("%s.%s ist Schalter, Wert %q", z.abschnitt, z.schluessel, a.Wert)
		}
	}
	for r := range reiterAnzahl {
		if len(zeilenDes(r)) == 0 {
			t.Errorf("Reiter %d ohne Zeilen", r)
		}
	}
	// Jede Option jeder Auswahl ist ein Wert, den der Leser annimmt (sonst scheiterte Save am Schreiber).
	for i, z := range tafelZeilen {
		for j, o := range tf.optionen(i) {
			kopie := *tf
			kopie.waehle(i, j)
			if got := werteText(kopie.wert, i); got != o.wert {
				t.Errorf("%s.%s: Option %q setzt %q", z.abschnitt, z.schluessel, o.wert, got)
			}
			if _, warn := settings.Parse("", "["+z.abschnitt+"]\n"+z.schluessel+" = "+o.wert+"\n"); len(warn) > 0 {
				t.Errorf("%s.%s = %s: %v", z.abschnitt, z.schluessel, o.wert, warn)
			}
		}
	}
	// Die eingebauten Sprachen: Ausgangssprache und Deutsch.
	if s := tafelSprachen(); !slices.Contains(s, "en") || !slices.Contains(s, "de") {
		t.Errorf("Sprachen %v", s)
	}
}

// ADR-0034 Punkt 4: Tab und Umschalt+Tab laufen durch die Felder des Reiters, dann Abbrechen und Speichern, an den Enden
// um; Strg+Bild↑/↓ wechselt den Reiter (auch um), der Fokus steht dann auf dem ersten Feld.
func TestTafelFokusUndReiter(t *testing.T) {
	testEditor(t)
	tf := offeneTafel(t, "")
	if tf.reiter != reiterAllgemein || tf.ziel() != 0 {
		t.Fatalf("Start: Reiter %d, Ziel %d", tf.reiter, tf.ziel())
	}
	var folge []int
	for range 4 {
		tf.taste(key.NameTab, 0)
		folge = append(folge, tf.ziel())
	}
	if !slices.Equal(folge, []int{zielAbbrechen, zielSpeichern, 0, zielAbbrechen}) {
		t.Errorf("Tab-Folge %v", folge)
	}
	tf.taste(key.NameTab, key.ModShift)
	if tf.ziel() != 0 {
		t.Errorf("Umschalt+Tab: Ziel %d", tf.ziel())
	}
	tf.taste(key.NamePageDown, key.ModShortcut)
	if tf.reiter != reiterAufnahme || tf.ziel() != 1 {
		t.Errorf("Strg+Bild↓: Reiter %d, Ziel %d", tf.reiter, tf.ziel())
	}
	tf.taste(key.NamePageUp, key.ModShortcut)
	tf.taste(key.NamePageUp, key.ModShortcut)
	if tf.reiter != reiterAgenten || tf.ziel() != 7 {
		t.Errorf("Strg+Bild↑ läuft nicht um: Reiter %d, Ziel %d", tf.reiter, tf.ziel())
	}
	tf.taste(key.NamePageDown, 0) // ohne Strg: nichts
	if tf.reiter != reiterAgenten {
		t.Error("Bild↓ ohne Strg wechselt den Reiter")
	}
	ag := zeilenDes(reiterAgenten)
	if !slices.Equal(tf.ziele(), append(slices.Clone(ag), zielAbbrechen, zielSpeichern)) {
		t.Errorf("Ziele Agenten %v", tf.ziele())
	}
	tf.fokusAuf(ag[2])
	if tf.ziel() != ag[2] {
		t.Errorf("fokusAuf: %d", tf.ziel())
	}
	tf.fokusAuf(0) // Sprache steht nicht im Reiter Agenten
	if tf.ziel() != ag[2] {
		t.Error("fokusAuf auf ein Feld eines anderen Reiters")
	}
}

// ADR-0034 Punkte 3 und 4: Leertaste schaltet bzw. öffnet die Liste bzw. löst den Knopf aus, Enter speichert (auch aus
// einem Feld, außer auf Abbrechen), Esc verwirft.
func TestTafelLeertasteEnterEsc(t *testing.T) {
	testEditor(t)
	tf := offeneTafel(t, "")
	tf.waehleReiter(reiterAufnahme)
	if f := tf.taste(key.NameSpace, 0); f != folgeListe || tf.listeZeile != 1 {
		t.Errorf("Leertaste auf Auswahl: Folge %d, Zeile %d", f, tf.listeZeile)
	}
	tf.taste(key.NameTab, 0)
	vorher := tf.wert.Editor
	if f := tf.taste(key.NameSpace, 0); f != folgeNichts || tf.wert.Editor == vorher {
		t.Errorf("Leertaste auf Schalter: Folge %d, Editor %v", f, tf.wert.Editor)
	}
	if f := tf.taste(key.NameReturn, 0); f != folgeSpeichern {
		t.Errorf("Enter im Feld: %d", f)
	}
	if f := tf.taste(key.NameEnter, 0); f != folgeSpeichern {
		t.Errorf("Enter (Ziffernblock) im Feld: %d", f)
	}
	if f := tf.taste(key.NameEscape, 0); f != folgeSchliessen {
		t.Errorf("Esc: %d", f)
	}
	tf.fokusAuf(zielAbbrechen)
	if f := tf.taste(key.NameReturn, 0); f != folgeSchliessen {
		t.Errorf("Enter auf Abbrechen: %d", f)
	}
	if f := tf.taste(key.NameSpace, 0); f != folgeSchliessen {
		t.Errorf("Leertaste auf Abbrechen: %d", f)
	}
	tf.fokusAuf(zielSpeichern)
	if f := tf.taste(key.NameSpace, 0); f != folgeSpeichern {
		t.Errorf("Leertaste auf Speichern: %d", f)
	}
	if f := tf.taste("V", 0); f != folgeNichts {
		t.Errorf("Buchstabe: %d", f)
	}
}

// ADR-0034 Punkte 2 und 4: Zahl mit − und + bzw. Pfeilen, Grenzen 1 … MaxTage, Ziffern tippbar (erste ersetzt),
// Rücktaste; eine ungültige Eingabe lässt den Wert stehen und zeigt den Fehler.
func TestTafelZahl(t *testing.T) {
	testEditor(t)
	tf := offeneTafel(t, "[library]\ncleanup-days = 2\n")
	tf.waehleReiter(reiterBibliothek)
	tf.taste(key.NameTab, 0) // bar, dann cleanup (ADR-0036 Punkt 2), dann die Zahl
	tf.taste(key.NameTab, 0)
	if tf.ziel() != zahlZeile() {
		t.Fatalf("Ziel %d", tf.ziel())
	}
	tf.taste(key.NameDownArrow, 0)
	tf.taste("-", 0)
	if tf.wert.Bibliothek.Tage != 1 {
		t.Errorf("unter 1: %d", tf.wert.Bibliothek.Tage)
	}
	tf.taste(key.NameUpArrow, 0)
	tf.taste("+", key.ModShift)
	if tf.wert.Bibliothek.Tage != 3 {
		t.Errorf("Pfeil/+: %d", tf.wert.Bibliothek.Tage)
	}
	tf.tippe("4")
	tf.tippe("5x")
	if tf.wert.Bibliothek.Tage != 45 || tf.zahlAnzeige() != "45" || tf.zahlFehler {
		t.Errorf("getippt: %d %q %v", tf.wert.Bibliothek.Tage, tf.zahlAnzeige(), tf.zahlFehler)
	}
	tf.taste(key.NameDeleteBackward, 0)
	if tf.wert.Bibliothek.Tage != 4 {
		t.Errorf("Rücktaste: %d", tf.wert.Bibliothek.Tage)
	}
	// Leer oder ungültig: Es gilt wieder der Wert vor dem Tippen (3), nicht der Zwischenstand 4 (F-626).
	tf.taste(key.NameDeleteBackward, 0)
	if !tf.zahlFehler || tf.wert.Bibliothek.Tage != 3 || tf.zahlAnzeige() != "" {
		t.Errorf("leer: Fehler %v, Tage %d, Anzeige %q", tf.zahlFehler, tf.wert.Bibliothek.Tage, tf.zahlAnzeige())
	}
	tf.tippe("0")
	if !tf.zahlFehler || tf.wert.Bibliothek.Tage != 3 {
		t.Errorf("0: Fehler %v, Tage %d", tf.zahlFehler, tf.wert.Bibliothek.Tage)
	}
	tf.taste(key.NameTab, 0) // Fokus weg: die Anzeige zeigt wieder den Wert
	tf.taste(key.NameTab, key.ModShift)
	if tf.zahlFehler || tf.zahlAnzeige() != "3" {
		t.Errorf("nach Fokuswechsel: %v %q", tf.zahlFehler, tf.zahlAnzeige())
	}
	tf.tippe("99999")
	if !tf.zahlFehler || tf.wert.Bibliothek.Tage != 3 {
		t.Errorf("über MaxTage: %v %d", tf.zahlFehler, tf.wert.Bibliothek.Tage)
	}
	tf.tippe("1") // höchstens fünf Ziffern
	if tf.zahlAnzeige() != "99999" {
		t.Errorf("sechste Ziffer: %q", tf.zahlAnzeige())
	}
	tf.wert.Bibliothek.Tage = library.MaxTage
	tf.zahlSchritt(1)
	if tf.wert.Bibliothek.Tage != library.MaxTage {
		t.Errorf("über MaxTage geschritten: %d", tf.wert.Bibliothek.Tage)
	}
	tf.fokusAuf(zielSpeichern)
	tf.tippe("7") // ohne Zahlfeld: nichts
	if tf.wert.Bibliothek.Tage != library.MaxTage {
		t.Error("Ziffer außerhalb des Zahlfelds wirkt")
	}
}

// Schalter, Auswahl und Warnungen der Agenten (ADR-0034 Punkte 2 und 6).
func TestTafelSchalterAuswahlWarnung(t *testing.T) {
	testEditor(t)
	tf := offeneTafel(t, "[general]\nlanguage = pt_BR\n")
	if tf.optionText(0) != "pt_BR" || !slices.Contains(tf.sprachen, "pt_BR") {
		t.Errorf("von Hand gesetzte Sprache: %q %v", tf.optionText(0), tf.sprachen)
	}
	tf.waehle(0, 0)
	if tf.wert.Sprache != "" || tf.optionText(0) != "Like the system" {
		t.Errorf("auto: %q %q", tf.wert.Sprache, tf.optionText(0))
	}
	tf.waehle(0, 99) // außerhalb: nichts
	tf.waehle(3, 1)
	if tf.wert.Kante != document.KanteWelle {
		t.Errorf("Kante %v", tf.wert.Kante)
	}
	tf.waehle(1, 2)
	if tf.wert.Bildschirm != settings.WahlAlle {
		t.Errorf("Bildschirm %v", tf.wert.Bildschirm)
	}
	for _, z := range zeilenDes(reiterAgenten) {
		vorher := tf.schalter(z)
		tf.schalte(z)
		if tf.schalter(z) == vorher {
			t.Errorf("%s schaltet nicht", tafelZeilen[z].schluessel)
		}
		mitWarnung := tafelZeilen[z].schluessel == "capture" || tafelZeilen[z].schluessel == "library" ||
			tafelZeilen[z].schluessel == "edit"
		for range 2 { // beide Stellungen: an mit, aus ohne Warnung
			if tf.warnungSichtbar(z) != (mitWarnung && tf.schalter(z)) {
				t.Errorf("%s: Warnung %v bei %v", tafelZeilen[z].schluessel, tf.warnungSichtbar(z), tf.schalter(z))
			}
			tf.schalte(z)
		}
	}
	for _, z := range []int{2, 4} {
		vorher := tf.schalter(z)
		tf.schalte(z)
		if tf.schalter(z) == vorher || tf.warnungSichtbar(z) {
			t.Errorf("Zeile %d", z)
		}
	}
}

// ADR-0034 Punkt 7: Der Fuß nennt die Zahl ungültiger Zeilen, „mehr als“ über der Grenze des Lesers, und eine Warnung
// zur ganzen Datei wörtlich.
func TestTafelWarnungen(t *testing.T) {
	testEditor(t)
	tf := offeneTafel(t, "# Kopf\n[capture]\nscreen = gross\nseam = wave\n[agents]\nimage = vielleicht\n")
	if tf.warnZeilen != 2 || tf.warnMehr || tf.warnDatei != "" {
		t.Errorf("zwei Zeilen: %d %v %q", tf.warnZeilen, tf.warnMehr, tf.warnDatei)
	}
	if w := tf.fussWarnung(); w != "2 lines in settings.ini are invalid and stay as they are." {
		t.Errorf("Fuß: %q", w)
	}
	if tf.wert.Kante != document.KanteWelle || tf.wert.Bildschirm != settings.WahlKlick || !tf.wert.Agenten.Bild {
		t.Error("ungültige Zeilen: wirksame Werte falsch")
	}
	var viel strings.Builder
	viel.WriteString("[capture]\n")
	for range 25 {
		viel.WriteString("screen = gross\n")
	}
	tf = offeneTafel(t, viel.String())
	if !tf.warnMehr || tf.warnZeilen != 20 || !strings.HasPrefix(tf.fussWarnung(), "More than 20") {
		t.Errorf("viele: %d %v %q", tf.warnZeilen, tf.warnMehr, tf.fussWarnung())
	}
	tf = offeneTafel(t, "[capture]\nscreen = gross\n")
	if tf.fussWarnung() != "1 line in settings.ini is invalid and stays as it is." {
		t.Errorf("eine: %q", tf.fussWarnung())
	}
	p := eigeneEinstellungen(t, "")
	if err := os.MkdirAll(p, 0o700); err != nil { // settings.ini als Ordner
		t.Fatal(err)
	}
	w, warn := settings.LadeDatei(p)
	tf = &tafel{}
	tf.oeffne(w, warn, p, nil)
	if soll := "settings.ini: the settings file is not a regular file; using the defaults."; len(warn) != 1 ||
		tf.warnZeilen != 0 || tf.fussWarnung() != soll || !strings.HasSuffix(warn[0], soll[len("settings.ini"):]) {
		t.Errorf("Datei: %q %d (Leser %q)", tf.warnDatei, tf.warnZeilen, warn)
	}
	// Eine Warnung ohne Pfad (Ort unbekannt) steht wörtlich da.
	if _, _, d := warnungenZaehlen("/x/settings.ini", []string{"Cannot locate"}); d != "Cannot locate" {
		t.Errorf("ohne Pfad: %q", d)
	}
	tf = offeneTafel(t, "[capture]\nseam = wave\n")
	if tf.fussWarnung() != "" {
		t.Errorf("ohne Warnung: %q", tf.fussWarnung())
	}
}

// ADR-0034 Punkte 3 und 5: Save schreibt nur geänderte Schlüssel, Handänderungen bleiben; ohne Änderung entsteht keine
// Datei; ein Fehler bleibt in der Tafel stehen.
func TestTafelSpeichern(t *testing.T) {
	testEditor(t)
	alt := "# von Hand\n[capture]\nscreen = gross\nseam = wave\n"
	tf := offeneTafel(t, alt)
	p, _ := settings.Pfad()
	geschrieben, ok := tf.speichere(settings.SchreibeDatei)
	if !ok || geschrieben {
		t.Fatalf("ohne Änderung: geschrieben %v, ok %v", geschrieben, ok)
	}
	if b, _ := os.ReadFile(p); string(b) != alt {
		t.Fatalf("ohne Änderung verändert: %q", b)
	}
	tf.schalte(2) // open-editor aus
	tf.waehle(3, 2)
	geschrieben, ok = tf.speichere(settings.SchreibeDatei)
	if !ok || !geschrieben || tf.fehler != "" {
		t.Fatalf("speichern: %v %v %q", geschrieben, ok, tf.fehler)
	}
	b, _ := os.ReadFile(p)
	if soll := "# von Hand\n[capture]\nscreen = gross\nseam = zigzag\nopen-editor = false\n"; string(b) != soll {
		t.Errorf("Datei:\n%s\nSoll:\n%s", b, soll)
	}

	eigeneEinstellungen(t, "")
	tf = offeneTafel(t, "")
	p, _ = settings.Pfad()
	if err := os.MkdirAll(p, 0o700); err != nil {
		t.Fatal(err)
	}
	tf.schalte(7) // agents.capture
	geschrieben, ok = tf.speichere(settings.SchreibeDatei)
	if ok || geschrieben || !strings.HasSuffix(tf.fehler, ": the settings file is not a regular file") || strings.Contains(tf.fehler, ": settings: ") {
		t.Errorf("Fehler: %v %v %q", geschrieben, ok, tf.fehler)
	}
}

// --- über den Editor (Gio-Router) ---

// tafelLauf öffnet einen Editor mit eigenem XDG_CONFIG_HOME und inhalt als settings.ini.
func tafelLauf(t *testing.T, inhalt string) (*menueLauf, string) {
	t.Helper()
	p := eigeneEinstellungen(t, inhalt)
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.fokusiere(l.e)
	return l, p
}

// Strg+, und Menü Datei → „Settings…“ öffnen die Tafel mit frisch gelesenen Werten; Esc und Kreuz schließen ohne zu
// schreiben, der Fokus geht an die Leinwand.
func TestTafelOeffnenUndSchliessen(t *testing.T) {
	l, p := tafelLauf(t, "[capture]\nseam = wave\n")
	l.taste(",", key.ModShortcut)
	if !l.e.einst.offen || l.e.einst.wert.Kante != document.KanteWelle {
		t.Fatalf("Strg+,: offen %v, Kante %v", l.e.einst.offen, l.e.einst.wert.Kante)
	}
	if !l.fokus(&l.e.einst) {
		t.Error("die Tafel hat den Tastaturfokus nicht")
	}
	if l.e.einst.rahmen.tafel.Empty() {
		t.Fatal("Tafel nicht gezeichnet")
	}
	l.e.einst.schalte(2)
	l.taste(key.NameEscape, 0)
	if l.e.einst.offen || !l.fokus(l.e) {
		t.Errorf("Esc: offen %v, Fokus Leinwand %v", l.e.einst.offen, l.fokus(l.e))
	}
	if b, _ := os.ReadFile(p); string(b) != "[capture]\nseam = wave\n" {
		t.Errorf("Esc hat geschrieben: %q", b)
	}
	// frisch gelesen: Änderung von Hand zwischen zwei Öffnungen
	if err := os.WriteFile(p, []byte("[capture]\nseam = zigzag\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	l.zuMenue(aktEinstellungen)
	l.titel().Click()
	l.setle()
	l.e.menue.eintraege[l.index(aktEinstellungen)].Click()
	l.setle()
	if !l.e.einst.offen || l.e.einst.wert.Kante != document.KanteZacken || l.e.menue.offen {
		t.Fatalf("Menü: offen %v, Kante %v, Menü offen %v", l.e.einst.offen, l.e.einst.wert.Kante, l.e.menue.offen)
	}
	l.klick(mitte(l.e.einst.rahmen.kreuz))
	if l.e.einst.offen {
		t.Error("Kreuz schließt nicht")
	}
	l.taste(",", key.ModShortcut)
	l.klick(mitte(l.e.einst.rahmen.abbrechen))
	if l.e.einst.offen {
		t.Error("Abbrechen schließt nicht")
	}
	// Die Hilfe (F1) nennt das Kürzel.
	gefunden := false
	for _, z := range l.e.hilfeZeilen() {
		gefunden = gefunden || (z.text == "Settings…" && z.taste == "Ctrl+,")
	}
	if !gefunden {
		t.Error("Hilfe nennt Strg+, nicht")
	}
}

// ADR-0034 Punkt 4: Solange die Tafel offen ist, greift keine Taste des Editors, kein Klick und kein Mausrad darunter.
func TestTafelSperrtEditor(t *testing.T) {
	l, _ := tafelLauf(t, "")
	l.taste(",", key.ModShortcut)
	wz, stand, zoom := l.e.s.Werkzeug, l.e.s.D.Stand(), l.e.s.Abb.Zoom
	for _, k := range []struct {
		n key.Name
		m key.Modifiers
	}{{"R", 0}, {key.NameDeleteForward, 0}, {"S", key.ModShortcut}, {"Z", key.ModShortcut}, {key.NameF1, 0},
		{key.NameF10, 0}, {"B", key.ModShortcut | key.ModShift}, {"D", key.ModShortcut}, {key.NamePageDown, key.ModShortcut},
		{"W", key.ModShortcut}} {
		l.taste(k.n, k.m)
	}
	l.tippe("abc")
	if !l.e.einst.offen || l.e.hilfeOffen || l.e.menue.offen || l.e.seiteZu {
		t.Errorf("Tasten wirken hinter der Tafel: offen %v, Hilfe %v, Menü %v, Leiste zu %v", l.e.einst.offen,
			l.e.hilfeOffen, l.e.menue.offen, l.e.seiteZu)
	}
	if l.e.s.Werkzeug != wz || l.e.s.D.Stand() != stand || len(l.e.s.D.Objekte()) != 1 {
		t.Error("Werkzeug oder Dokument hinter der Tafel geändert")
	}
	if l.e.einst.reiter != reiterAufnahme {
		t.Errorf("Strg+Bild↓ wechselt den Reiter nicht: %d", l.e.einst.reiter)
	}
	// Klick und Rad außerhalb der Tafel: nichts, die Tafel bleibt offen.
	draussen := image.Pt(l.e.einst.rahmen.tafel.Min.X-20, l.e.einst.rahmen.tafel.Min.Y+50)
	l.klick(draussen)                      // Leinwand: Klick daneben wählte sonst den Text ab
	l.klick(image.Pt(1150, 300))           // rechte Leiste
	l.klick(mitte(l.e.einst.rahmen.tafel)) // leere Fläche in der Tafel
	l.r.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Position: f32.Pt(float32(draussen.X),
		float32(draussen.Y)), Scroll: f32.Pt(0, -60)})
	l.setle()
	if !l.e.einst.offen || l.e.s.D.Stand() != stand || l.e.s.Abb.Zoom != zoom || l.e.s.Auswahl != l.id {
		t.Errorf("Klick oder Rad neben die Tafel wirkt: offen %v, Zoom %v→%v, Auswahl %d", l.e.einst.offen, zoom,
			l.e.s.Abb.Zoom, l.e.s.Auswahl)
	}
	// Tab bleibt in der Tafel: kein Knopf des Editors bekommt den Fokus.
	for range 6 {
		l.taste(key.NameTab, 0)
		if c := l.e.fokussierterKnopf(l.gtx()); c != nil || !l.fokus(&l.e.einst) {
			t.Fatalf("Tab verlässt die Tafel (Knopf %v)", c != nil)
		}
	}
}

// Klicks: Reiter, Schalter, − und +, Speichern schreibt die Datei, schließt und meldet es.
func TestTafelKlicksUndSpeichern(t *testing.T) {
	l, p := tafelLauf(t, "; Kommentar von Hand\n[agents]\nconfirm = yes\n")
	l.taste(",", key.ModShortcut)
	tf := &l.e.einst
	l.klick(mitte(tf.rahmen.reiter[reiterAgenten]))
	if tf.reiter != reiterAgenten {
		t.Fatalf("Reiter %d", tf.reiter)
	}
	ag := zeilenDes(reiterAgenten)
	l.klick(mitte(tf.rahmen.feld[ag[0]]))
	if !tf.wert.Agenten.Aufnehmen || tf.ziel() != ag[0] {
		t.Fatalf("Klick auf Schalter: %v, Ziel %d", tf.wert.Agenten.Aufnehmen, tf.ziel())
	}
	if tf.rahmen.warnung[ag[0]].Empty() || tf.rahmen.warnung[ag[0]].Min.Y < tf.rahmen.feld[ag[0]].Max.Y {
		t.Error("Warnung unter „Capture screen“ fehlt")
	}
	l.klick(mitte(tf.rahmen.reiter[reiterBibliothek]))
	l.klick(mitte(tf.rahmen.plus))
	l.klick(mitte(tf.rahmen.plus))
	l.klick(mitte(tf.rahmen.minus))
	if tf.wert.Bibliothek.Tage != 91 || tf.ziel() != zahlZeile() {
		t.Errorf("−/+: %d, Ziel %d", tf.wert.Bibliothek.Tage, tf.ziel())
	}
	l.tippe("12")
	if tf.wert.Bibliothek.Tage != 12 {
		t.Errorf("getippt über den Router: %d", tf.wert.Bibliothek.Tage)
	}
	l.klick(mitte(tf.rahmen.ok))
	if tf.offen || l.e.meldung.Text != "Settings saved" || !l.fokus(l.e) {
		t.Errorf("Save: offen %v, Meldung %q", tf.offen, l.e.meldung.Text)
	}
	b, _ := os.ReadFile(p)
	if soll := "; Kommentar von Hand\n[agents]\nconfirm = yes\ncapture = yes\n\n[library]\ncleanup-days = 12\n"; string(b) != soll {
		t.Errorf("Datei:\n%s\nSoll:\n%s", b, soll)
	}
	// Enter speichert auch aus einem Feld.
	l.taste(",", key.ModShortcut)
	l.taste(key.NameSpace, 0) // Sprache: Liste öffnen
	if !l.e.menue.offen || l.e.menue.welches != listeEinstellung {
		t.Fatal("Leertaste öffnet die Auswahl nicht")
	}
	l.taste(key.NameEscape, 0)
	if l.e.menue.offen || !tf.offen || !l.fokus(&l.e.einst) {
		t.Fatalf("Esc in der Liste: Menü %v, Tafel %v", l.e.menue.offen, tf.offen)
	}
	l.taste(key.NamePageDown, key.ModShortcut)
	l.taste(key.NameTab, 0)
	l.taste(key.NameSpace, 0) // Editor öffnen: aus
	l.taste(key.NameReturn, 0)
	if tf.offen {
		t.Fatal("Enter speichert nicht")
	}
	if b, _ := os.ReadFile(p); !strings.Contains(string(b), "open-editor = false") {
		t.Errorf("Enter hat nicht geschrieben:\n%s", b)
	}
}

// Auswahl über die Liste des Menüs: per Tastatur (Leertaste, Pfeil, Enter) und per Maus (Klick auf Feld und Eintrag).
func TestTafelAuswahlliste(t *testing.T) {
	l, _ := tafelLauf(t, "")
	l.taste(",", key.ModShortcut)
	tf := &l.e.einst
	tf.waehleReiter(reiterAufnahme)
	l.setle()
	l.taste(key.NameSpace, 0)
	if !l.e.menue.offen || l.e.menue.welches != listeEinstellung || len(l.e.menueEintraege()) != 3 {
		t.Fatalf("Liste: offen %v, Einträge %d", l.e.menue.offen, len(l.e.menueEintraege()))
	}
	if !l.e.menueEintraege()[0].aktiv {
		t.Error("gewählter Eintrag nicht markiert")
	}
	if a := l.e.listenAnker(listeEinstellung); a != tf.rahmen.feld[1] {
		t.Errorf("Liste hängt nicht am Feld: %v %v", a, tf.rahmen.feld[1])
	}
	l.taste(key.NameDownArrow, 0)
	// Enter in der Liste: Schon im selben Bild geht der Fokus an die Tafel (menueSchliesse), nicht erst über das
	// Zurückholen im nächsten Bild (dazwischen gehörten Enter und Leertaste dem Feldknopf).
	l.r.Queue(key.Event{Name: key.NameReturn, State: key.Press})
	l.bild()
	if !l.fokus(&l.e.einst) {
		t.Errorf("nach Enter in der Liste liegt der Fokus nicht sofort bei der Tafel (Menü offen %v, Wert %v)", l.e.menue.offen, tf.wert.Bildschirm)
	}
	l.setle()
	if l.e.menue.offen || tf.wert.Bildschirm != settings.WahlZeiger || !tf.offen {
		t.Fatalf("Pfeil+Enter: Menü %v, Wert %v, Tafel %v", l.e.menue.offen, tf.wert.Bildschirm, tf.offen)
	}
	if !l.fokus(&l.e.einst) {
		t.Error("nach der Wahl hat die Tafel den Fokus nicht")
	}
	l.klick(mitte(tf.rahmen.feld[3])) // Kante
	if !l.e.menue.offen || tf.listeZeile != 3 || tf.ziel() != 3 {
		t.Fatalf("Klick auf Auswahl: Menü %v, Zeile %d", l.e.menue.offen, tf.listeZeile)
	}
	l.klick(mitte(l.e.menue.rahmen[2]))
	if l.e.menue.offen || tf.wert.Kante != document.KanteZacken {
		t.Errorf("Klick auf Eintrag: Menü %v, Kante %v", l.e.menue.offen, tf.wert.Kante)
	}
	l.klick(mitte(tf.rahmen.feld[3]))
	l.taste(key.NameEscape, 0)
	l.taste(key.NameEscape, 0)
	if tf.offen || l.e.menue.offen {
		t.Error("Esc, Esc: Liste, dann Tafel zu")
	}
}

// ADR-0034 Punkt 5: Kann nicht geschrieben werden, bleibt die Tafel offen und der Fuß zeigt den Fehler rot.
func TestTafelFehlerImFuss(t *testing.T) {
	l, p := tafelLauf(t, "")
	if err := os.MkdirAll(p, 0o700); err != nil {
		t.Fatal(err)
	}
	l.taste(",", key.ModShortcut)
	tf := &l.e.einst
	if tf.fussWarnung() == "" {
		t.Error("Warnung zur Datei fehlt im Fuß")
	}
	l.taste(key.NamePageDown, key.ModShortcut)
	l.taste(key.NameTab, 0)
	l.taste(key.NameSpace, 0)
	l.taste(key.NameReturn, 0)
	if !tf.offen || !strings.Contains(tf.gezeigteFehler, "not a regular file") {
		t.Errorf("Fehler: offen %v, Fuß %q", tf.offen, tf.gezeigteFehler)
	}
	if l.e.meldung.Text == "Settings saved" {
		t.Error("„Settings saved“ trotz Fehler")
	}
}

// Die Tafel zeichnet (Gio headless): Grund abgedunkelt, Schalter an grün mit Fokusring, aktiver Reiter grün.
func TestTafelZeichnet(t *testing.T) {
	l, _ := tafelLauf(t, "")
	l.taste(",", key.ModShortcut)
	tf := &l.e.einst
	tf.waehleReiter(reiterAgenten)
	tf.schalte(zeilenDes(reiterAgenten)[0])
	l.setle()
	b := headlessBild(t, l.fenst, func(gtx layout.Context) { l.e.frame(gtx) })
	z := zeilenDes(reiterAgenten)[0]
	feld := tf.rahmen.feld[z]
	px := func(p image.Point) color.RGBA { return b.RGBAAt(p.X, p.Y) }
	gleich := func(a color.RGBA, c color.NRGBA) bool { return a.R == c.R && a.G == c.G && a.B == c.B }
	if k := image.Pt(feld.Min.X+4, feld.Min.Y+feld.Dy()/2-6); !gleich(px(k), farbeAuswahl) {
		t.Errorf("Schalter an nicht grün: %v", px(k))
	}
	if k := image.Pt(feld.Min.X-3, feld.Min.Y+feld.Dy()/2); !gleich(px(k), farbeAkzent) {
		t.Errorf("kein Fokusring: %v", px(k))
	}
	if r := tf.rahmen.reiter[reiterAgenten]; !gleich(px(r.Min.Add(image.Pt(3, 3))), farbeAuswahl) {
		t.Errorf("aktiver Reiter nicht grün: %v", px(r.Min.Add(image.Pt(3, 3))))
	}
	if w := tf.rahmen.warnung[z]; !gleich(px(w.Min.Add(image.Pt(1, 2))), farbeAkzent) ||
		!gleich(px(image.Pt(w.Max.X-3, w.Min.Y+3)), farbeWarnGrund) {
		t.Errorf("Warnkasten: %v %v", px(w.Min.Add(image.Pt(1, 2))), px(image.Pt(w.Max.X-3, w.Min.Y+3)))
	}
	// außerhalb abgedunkelt: das Fenster (grau-900) ist dunkler als ohne Tafel
	if a := px(image.Pt(5, 790)); a.R >= farbeLeisteVoll.R {
		t.Errorf("nicht abgedunkelt: %v", a)
	}
}
