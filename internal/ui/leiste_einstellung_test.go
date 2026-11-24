// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"errors"
	"image"
	"os"
	"testing"

	"gioui.org/io/key"

	"github.com/Andiministrator/schnapp/internal/settings"
)

// F-655 (KB-3): Die Leiste ist aus (Probe-Lauf, gescheitertes Merken, ein zweites Fenster), settings.ini sagt yes. Ein
// Speichern der Tafel ohne Änderung an „bar“ lässt sie aus; ändert die Tafel „bar“, folgt sie.
func TestTafelSchaltetLeisteNurBeiAenderung(t *testing.T) {
	eigeneEinstellungen(t, "")
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.e.unten.an = false
	l.taste(",", key.ModShortcut)
	if !l.e.einst.offen {
		t.Fatal("Tafel nicht offen")
	}
	l.e.einst.wert.Sprache = "de" // eine andere Änderung, damit wirklich geschrieben wird
	l.e.einstellungenSpeichern(l.gtx())
	if l.e.einst.offen {
		t.Fatalf("Tafel nicht gespeichert: %q", l.e.einst.fehler)
	}
	if l.e.unten.an {
		t.Error("Tafel-Speichern ohne Änderung an bar schaltet die Leiste ein")
	}
	// Mit Änderung an bar folgt die Leiste (aus der Datei: yes → no, dann no → yes).
	l.e.unten.an = true
	l.taste(",", key.ModShortcut)
	l.e.einst.wert.Leiste = false
	l.e.einstellungenSpeichern(l.gtx())
	if l.e.unten.an {
		t.Error("Leiste folgt der Tafel nicht (aus)")
	}
	l.taste(",", key.ModShortcut)
	l.e.einst.wert.Leiste = true
	l.e.einstellungenSpeichern(l.gtx())
	if !l.e.unten.an {
		t.Error("Leiste folgt der Tafel nicht (ein)")
	}
}

// F-658 (KB-9): merkeEinstellung schreibt nur library.bar; jeder andere Schlüssel, vor allem eine Einwilligung der
// Agenten, wird abgelehnt und nichts geschrieben.
func TestMerkeEinstellungNurLeiste(t *testing.T) {
	p := eigeneEinstellungen(t, "")
	for _, a := range []settings.Aenderung{
		{Abschnitt: "agents", Schluessel: "capture", Wert: "yes"},
		{Abschnitt: "library", Schluessel: "agents", Wert: "yes"},
		{Abschnitt: "general", Schluessel: "bar", Wert: "no"},
	} {
		if err := merkeEinstellung(a); !errors.Is(err, errNichtMerkbar) {
			t.Errorf("%s.%s: %v", a.Abschnitt, a.Schluessel, err)
		}
	}
	if _, err := os.Stat(p); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("settings.ini geschrieben: %v", err)
	}
	if err := merkeEinstellung(settings.Aenderung{Abschnitt: "library", Schluessel: "bar", Wert: "no"}); err != nil {
		t.Fatal(err)
	}
	if w, _ := settings.LadeDatei(p); w.Leiste {
		t.Error("library.bar nicht gemerkt")
	}
}
