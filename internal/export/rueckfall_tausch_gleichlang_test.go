// SPDX-License-Identifier: GPL-3.0-or-later

package export

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// F-331: Die Prüfung os.SameFile in oeffneDirekt trägt allein, wenn die untergeschobene Datei gleich lang ist; die
// Längenprüfung in leseAlt fängt nur andere Längen (TestRueckfallTauschVorOeffnen). Die fremde Datei bleibt unverändert.
func TestRueckfallTauschGleichLangeDatei(t *testing.T) {
	ziel := zielMit(t, []byte("alt: zwölf!!"))
	fremd := filepath.Join(filepath.Dir(ziel), "fremd.png")
	inhalt := []byte("fremd: zwölf") // gleiche Länge in Bytes wie der alte Inhalt
	if len(inhalt) != len("alt: zwölf!!") {
		t.Fatal("Vorbedingung: gleiche Länge")
	}
	if err := os.WriteFile(fremd, inhalt, 0o600); err != nil {
		t.Fatal(err)
	}
	vorOeffnen(t, func() {
		if err := os.Rename(fremd, ziel); err != nil {
			t.Fatal(err)
		}
	})
	if err := schreibeDirekt(ziel, schreibeNeu, os.ErrPermission); !errors.Is(err, os.ErrPermission) {
		t.Errorf("Fehler %v, erwartet die Ursache", err)
	}
	if got, err := os.ReadFile(ziel); err != nil || !bytes.Equal(got, inhalt) {
		t.Errorf("Ersatzdatei verändert: %q, %v", got, err)
	}
}
