// SPDX-License-Identifier: GPL-3.0-or-later

package settings

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestSetzeZeilen prüft die Regeln aus ADR-0034 Punkt 5 am Text, ohne Datei.
func TestSetzeZeilen(t *testing.T) {
	fall := []struct {
		name, ein string
		a         []Aenderung
		aus       string
	}{
		{"ersetzt nur die Zeile, Kommentare bleiben",
			"# Kopf\n[capture]\n  screen = pick   \n; Notiz\nseam = wave\n",
			[]Aenderung{{"capture", "screen", "all"}},
			"# Kopf\n[capture]\nscreen = all\n; Notiz\nseam = wave\n"},
		{"letzte Zeile des Schlüssels im letzten Abschnitt",
			"[capture]\nscreen = pick\n[library]\ncleanup = true\n[capture]\nscreen = pointer\nscreen = kaputt\n",
			[]Aenderung{{"capture", "screen", "all"}},
			"[capture]\nscreen = pick\n[library]\ncleanup = true\n[capture]\nscreen = pointer\nscreen = all\n"},
		{"fehlender Schlüssel hinter die letzte Schlüsselzeile",
			"[capture]\nscreen = pick\n\n# Ende\n[library]\ncleanup = true\n",
			[]Aenderung{{"capture", "seam", "zigzag"}},
			"[capture]\nscreen = pick\nseam = zigzag\n\n# Ende\n[library]\ncleanup = true\n"},
		{"fehlender Abschnitt am Ende, ohne Umbruch am Dateiende",
			"[capture]\nscreen = pick",
			[]Aenderung{{"agents", "capture", "yes"}, {"agents", "edit", "no"}},
			"[capture]\nscreen = pick\n\n[agents]\ncapture = yes\nedit = no\n"},
		{"CRLF bleibt CRLF",
			"[library]\r\ncleanup = true\r\n",
			[]Aenderung{{"library", "cleanup", "false"}, {"library", "cleanup-days", "30"}},
			"[library]\r\ncleanup = false\r\ncleanup-days = 30\r\n"},
		{"unbekannter Abschnitt und kaputter Kopf werden nicht beschrieben",
			"[fremd]\nscreen = x\n[capture\nscreen = y\n",
			[]Aenderung{{"capture", "screen", "all"}},
			"[fremd]\nscreen = x\n[capture\nscreen = y\n\n[capture]\nscreen = all\n"},
		{"Zeile vor jedem Abschnitt zählt nicht",
			"screen = pick\n[capture]\n",
			[]Aenderung{{"capture", "screen", "all"}},
			"screen = pick\n[capture]\nscreen = all\n"},
		{"BOM bleibt",
			"\xef\xbb\xbf[general]\nlanguage = de\n",
			[]Aenderung{{"general", "language", "auto"}},
			"\xef\xbb\xbf[general]\nlanguage = auto\n"},
		{"Abschnittskopf mit Leerraum",
			"[ agents ]\n",
			[]Aenderung{{"agents", "library", "yes"}},
			"[ agents ]\nlibrary = yes\n"},
		{"leere Datei",
			"",
			[]Aenderung{{"capture", "screen", "all"}},
			"[capture]\nscreen = all\n"},
	}
	for _, f := range fall {
		if got := setzeZeilen(f.ein, f.a); got != f.aus {
			t.Errorf("%s:\n got %q\nwant %q", f.name, got, f.aus)
		}
		// Was geschrieben ist, liest der Leser auch so.
		e, _ := Parse("", setzeZeilen(f.ein, f.a))
		for _, a := range f.a {
			for _, w := range Werte(e) {
				if w.Abschnitt == a.Abschnitt && w.Schluessel == a.Schluessel && w.Wert != a.Wert {
					t.Errorf("%s: Leser sieht %s.%s = %q, geschrieben %q", f.name, a.Abschnitt, a.Schluessel, w.Wert, a.Wert)
				}
			}
		}
	}
}

// TestWerteDeckenParse: Jeder Schlüssel, den der Leser kennt, steht in Werte (und damit in der Tafel,
// ADR-0034 Punkt 2), und Werte ist für Voreinstellungen und Abweichungen rundlauffest.
func TestWerteDeckenParse(t *testing.T) {
	quelle, err := os.ReadFile("einstellungen.go")
	if err != nil {
		t.Fatal(err)
	}
	bekannt := map[string]bool{}
	for _, w := range Werte(Voreinstellungen()) {
		bekannt[w.Abschnitt+"."+w.Schluessel] = true
	}
	// Die case-Zeilen des Lesers nennen jeden Schlüssel als "abschnitt.schlüssel".
	gefunden := 0
	for _, z := range strings.Split(string(quelle), "\n") {
		z = strings.TrimSpace(z)
		if !strings.HasPrefix(z, "case \"") {
			continue
		}
		for _, teil := range strings.Split(strings.TrimSuffix(strings.TrimPrefix(z, "case "), ":"), ",") {
			k := strings.Trim(strings.TrimSpace(teil), "\"")
			if !strings.Contains(k, ".") {
				continue
			}
			gefunden++
			if !bekannt[k] {
				t.Errorf("Schlüssel %s fehlt in Werte (und in der Tafel)", k)
			}
		}
	}
	if gefunden != len(bekannt) {
		t.Errorf("Leser nennt %d Schlüssel, Werte %d", gefunden, len(bekannt))
	}

	anders := Voreinstellungen()
	anders.Sprache = "pt_BR"
	anders.Bildschirm = WahlAlle
	anders.Editor = false
	anders.Kante = 2
	anders.Bibliothek.An, anders.Bibliothek.Tage = false, 7
	anders.Leiste = false
	anders.Agenten = Agenten{Aufnehmen: true, Bibliothek: true, Bestaetigen: false, Bild: false, Bearbeiten: true}
	for _, e := range []Einstellungen{Voreinstellungen(), anders} {
		var b strings.Builder
		for _, w := range Werte(e) {
			b.WriteString("[" + w.Abschnitt + "]\n" + w.Schluessel + " = " + w.Wert + "\n")
		}
		zurueck, warn := Parse("", b.String())
		if len(warn) > 0 || !reflect.DeepEqual(zurueck, e) {
			t.Errorf("Rundlauf: %+v, Warnungen %v, erwartet %+v", zurueck, warn, e)
		}
	}
	if u := Unterschiede(Voreinstellungen(), Voreinstellungen()); len(u) != 0 {
		t.Errorf("Unterschiede ohne Änderung: %v", u)
	}
	if u := Unterschiede(Voreinstellungen(), anders); len(u) != len(Werte(anders)) {
		t.Errorf("Unterschiede: %d statt %d", len(u), len(Werte(anders)))
	}
}

// TestSchreibeDatei prüft Anlegen, Rechte, Verweise und die Fälle, in denen nichts geschrieben wird.
func TestSchreibeDatei(t *testing.T) {
	ordner := t.TempDir()
	pfad := filepath.Join(ordner, "schnapp", "settings.ini")

	// Neu anlegen: Ordner 0700, Datei 0600, Kopfzeile.
	if err := SchreibeDatei(pfad, []Aenderung{{"capture", "screen", "all"}}); err != nil {
		t.Fatal(err)
	}
	daten, _ := os.ReadFile(pfad)
	if want := kopfNeueDatei + "\n\n[capture]\nscreen = all\n"; string(daten) != want {
		t.Errorf("neue Datei %q, erwartet %q", daten, want)
	}
	if info, _ := os.Stat(pfad); info.Mode().Perm() != 0o600 {
		t.Errorf("Rechte %v", info.Mode().Perm())
	}
	if info, _ := os.Stat(filepath.Dir(pfad)); info.Mode().Perm() != 0o700 {
		t.Errorf("Ordnerrechte %v", info.Mode().Perm())
	}

	// Vorhandene Rechte bleiben.
	if err := os.Chmod(pfad, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := SchreibeDatei(pfad, []Aenderung{{"capture", "screen", "pick"}}); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Stat(pfad); info.Mode().Perm() != 0o640 {
		t.Errorf("Rechte nach dem Schreiben %v", info.Mode().Perm())
	}

	// Keine temporären Reste.
	eintraege, _ := os.ReadDir(filepath.Dir(pfad))
	if len(eintraege) != 1 {
		t.Errorf("Reste im Ordner: %v", eintraege)
	}

	// Verweis: das Ziel wird geschrieben, der Verweis bleibt.
	zielOrdner := filepath.Join(ordner, "dotfiles")
	if err := os.Mkdir(zielOrdner, 0o700); err != nil {
		t.Fatal(err)
	}
	ziel := filepath.Join(zielOrdner, "schnapp.ini")
	if err := os.WriteFile(ziel, []byte("[capture]\nscreen = pick\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	verweis := filepath.Join(ordner, "verweis.ini")
	if err := os.Symlink(ziel, verweis); err != nil {
		t.Fatal(err)
	}
	if err := SchreibeDatei(verweis, []Aenderung{{"capture", "screen", "pointer"}}); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Lstat(verweis); info.Mode()&os.ModeSymlink == 0 {
		t.Error("Verweis ist ersetzt")
	}
	if d, _ := os.ReadFile(ziel); string(d) != "[capture]\nscreen = pointer\n" {
		t.Errorf("Ziel %q", d)
	}
	leer := filepath.Join(ordner, "leer.ini")
	if err := os.Symlink(filepath.Join(ordner, "gibtsnicht"), leer); err != nil {
		t.Fatal(err)
	}
	if err := SchreibeDatei(leer, []Aenderung{{"capture", "screen", "all"}}); !errors.Is(err, ErrVerweisLeer) {
		t.Errorf("Verweis ins Leere: %v", err)
	}

	// Nicht schreiben: zu groß, kein UTF-8, Ordner statt Datei, ungültige Werte.
	gross := filepath.Join(ordner, "gross.ini")
	inhalt := strings.Repeat("# x\n", maxDateigroesse/4+1)
	if err := os.WriteFile(gross, []byte(inhalt), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := SchreibeDatei(gross, []Aenderung{{"capture", "screen", "all"}}); !errors.Is(err, ErrZuGross) {
		t.Errorf("zu groß: %v", err)
	}
	if d, _ := os.ReadFile(gross); string(d) != inhalt {
		t.Error("zu große Datei verändert")
	}
	// F-629 a: Eine Datei knapp unter der Grenze wüchse durch die neue Zeile darüber; danach verwürfe der Leser sie
	// ganz, und jede Korrektur scheiterte. Der Schreiber lehnt ab und lässt die Datei stehen.
	knapp := filepath.Join(ordner, "knapp.ini")
	inhalt = "[agents]\ncapture = yes\n" + strings.Repeat("# x\n", (maxDateigroesse-len("[agents]\ncapture = yes\n"))/4)
	if len(inhalt) > maxDateigroesse || len(inhalt) < maxDateigroesse-3 {
		t.Fatalf("Prüfdatei hat %d Bytes", len(inhalt))
	}
	if err := os.WriteFile(knapp, []byte(inhalt), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := SchreibeDatei(knapp, []Aenderung{{"capture", "screen", "all"}}); !errors.Is(err, ErrZuGross) {
		t.Errorf("über die Grenze gewachsen: %v", err)
	}
	if d, _ := os.ReadFile(knapp); string(d) != inhalt {
		t.Error("über die Grenze gewachsene Datei geschrieben")
	}
	if e, warn := LadeDatei(knapp); !e.Agenten.Aufnehmen || len(warn) > 0 {
		t.Errorf("Datei nach dem Versuch unbrauchbar: %+v %v", e.Agenten, warn)
	}
	// Ein Ersetzen, das nicht wächst, gelingt weiter.
	if err := SchreibeDatei(knapp, []Aenderung{{"agents", "capture", "no"}}); err != nil {
		t.Errorf("Ersetzen ohne Wachsen: %v", err)
	}
	kaputt := filepath.Join(ordner, "kaputt.ini")
	if err := os.WriteFile(kaputt, []byte("[capture]\n\xff\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := SchreibeDatei(kaputt, []Aenderung{{"capture", "screen", "all"}}); !errors.Is(err, ErrKeinUTF8) {
		t.Errorf("kein UTF-8: %v", err)
	}
	if err := SchreibeDatei(zielOrdner, []Aenderung{{"capture", "screen", "all"}}); !errors.Is(err, ErrKeineDatei) {
		t.Errorf("Ordner: %v", err)
	}
	for _, a := range []Aenderung{
		{"capture", "screen", "alle"},
		{"capture", "screen", "all\n[agents]\ncapture = yes"},
		{"agents", "nix", "yes"},
		{"fremd", "screen", "all"},
		{"capture]\n[agents", "capture", "yes"},
		{"library", "cleanup-days", "0"},
	} {
		if err := SchreibeDatei(pfad, []Aenderung{{"capture", "seam", "wave"}, a}); !errors.Is(err, ErrUngueltig) {
			t.Errorf("%+v: %v", a, err)
		}
	}
	if d, _ := os.ReadFile(pfad); strings.Contains(string(d), "wave") {
		t.Error("bei ungültiger Änderung ist trotzdem etwas geschrieben")
	}
	// Fehler tragen das Paketpräfix (ADR-0019 Nachtrag 2).
	if err := SchreibeDatei(gross, []Aenderung{{"capture", "screen", "all"}}); err == nil || !strings.HasPrefix(err.Error(), "settings: ") {
		t.Errorf("Präfix: %v", err)
	}
	// Keine Änderung: nichts anfassen, auch keine Datei anlegen.
	nichts := filepath.Join(ordner, "nichts", "settings.ini")
	if err := SchreibeDatei(nichts, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(nichts); !os.IsNotExist(err) {
		t.Errorf("ohne Änderung angelegt: %v", err)
	}
}

// TestSchreibeMitXDG nutzt den Ort aus XDG_CONFIG_HOME.
func TestSchreibeMitXDG(t *testing.T) {
	ordner := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", ordner)
	if err := Schreibe([]Aenderung{{"agents", "capture", "yes"}}); err != nil {
		t.Fatal(err)
	}
	e, warn := Lade()
	if len(warn) > 0 || !e.Agenten.Aufnehmen {
		t.Errorf("gelesen %+v, %v", e.Agenten, warn)
	}
}

// ADR-0034 Nachtrag 1 (F-625): Der Ort der Datei des MCP-Servers kommt aus dem übergebenen Heim, nie aus
// XDG_CONFIG_HOME oder HOME; ohne bekanntes Heim gibt es keinen Ort.
func TestAgentenPfad(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	heim := t.TempDir()
	p, err := AgentenPfadIn(heim + "/")
	if err != nil || p != filepath.Join(heim, ".config", "schnapp", "settings.ini") {
		t.Errorf("AgentenPfadIn: %q %v", p, err)
	}
	for _, h := range []string{"", "relativ/heim"} {
		if _, err := AgentenPfadIn(h); !errors.Is(err, ErrKeinHeim) || !strings.HasPrefix(err.Error(), "settings: ") {
			t.Errorf("%q: %v", h, err)
		}
	}
	alt := BenutzerHeim
	t.Cleanup(func() { BenutzerHeim = alt })
	BenutzerHeim = func() (string, error) { return heim, nil }
	if q, err := AgentenPfad(); err != nil || q != p {
		t.Errorf("AgentenPfad: %q %v", q, err)
	}
	BenutzerHeim = func() (string, error) { return "", errors.New("unbekannt") }
	if _, err := AgentenPfad(); !errors.Is(err, ErrKeinHeim) {
		t.Errorf("ohne Heim: %v", err)
	}
}
