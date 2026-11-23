// SPDX-License-Identifier: GPL-3.0-or-later

package settings

import (
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/i18n"
)

// TestAgentenVoreinstellung: Aufnehmen und Bibliothek sind aus, Dialog und Bild an (ADR-0030 Nachtrag 2).
func TestAgentenVoreinstellung(t *testing.T) {
	a := Voreinstellungen().Agenten
	if a.Aufnehmen || a.Bibliothek || a.Bearbeiten || !a.Bestaetigen || !a.Bild {
		t.Errorf("Voreinstellung [agents]: %+v", a)
	}
}

func TestAgentenWerte(t *testing.T) {
	i18n.Setze(nil)
	e, warn := Parse("s.ini", "[agents]\ncapture = yes\nlibrary = true\nconfirm = no\nimage = false\nedit = yes\n")
	if len(warn) != 0 {
		t.Fatalf("Warnungen: %q", warn)
	}
	if want := (Agenten{Aufnehmen: true, Bibliothek: true, Bearbeiten: true}); e.Agenten != want {
		t.Errorf("[agents]: %+v, erwartet %+v", e.Agenten, want)
	}
}

// TestAgentenUngueltigBleibtAus: Ein Tippfehler darf keine Einwilligung erzeugen und keinen Dialog abschalten.
func TestAgentenUngueltigBleibtAus(t *testing.T) {
	i18n.Setze(nil)
	e, warn := Parse("s.ini", "[agents]\ncapture = ja\nlibrary = 1\nconfirm = nein\nimage = ?\nedit = j\nnetwork = yes\n")
	if e.Agenten != Voreinstellungen().Agenten {
		t.Errorf("ungültige Werte änderten [agents]: %+v", e.Agenten)
	}
	if len(warn) != 6 || !strings.Contains(warn[5], "network") {
		t.Errorf("erwartet 6 Warnungen (5 Werte, 1 Schlüssel): %q", warn)
	}
}

// TestAgentenUngueltigNachGueltig: Ein späterer ungültiger Wert nimmt eine Einwilligung nicht zurück, ein späterer
// gültiger schon (gleiche Regel wie überall, F-93).
func TestAgentenSpaeterWert(t *testing.T) {
	i18n.Setze(nil)
	e, _ := Parse("s.ini", "[agents]\ncapture = yes\ncapture = no\n")
	if e.Agenten.Aufnehmen {
		t.Error("der letzte gültige Wert zählt nicht")
	}
}
