// SPDX-License-Identifier: GPL-3.0-or-later

package export

import (
	"errors"
	"os"
	"strings"
	"testing"
)

// F-331: Ein Close-Fehler nach erfolgreichem Schreiben im Rückfall wird gemeldet, nicht verschluckt. Der Fehler
// entsteht über die Naht oeffneDirekt im eigenen Code (Hülle um die echte Datei).

// schliessFehlerDatei schließt die echte Datei und meldet dennoch einen Fehler (etwa NFS: Schreibfehler erst beim Close).
type schliessFehlerDatei struct {
	zielDatei
	fehler error
}

func (s schliessFehlerDatei) Close() error {
	s.zielDatei.Close()
	return s.fehler
}

func TestRueckfallCloseFehlerWirdGemeldet(t *testing.T) {
	ziel := zielMit(t, []byte("alt und länger als der neue Inhalt"))
	kaputt := errors.New("Close")
	biegeUm(t, 1, func(f zielDatei) zielDatei { return schliessFehlerDatei{f, kaputt} })
	err := schreibeDirekt(ziel, schreibeNeu, os.ErrPermission)
	if !errors.Is(err, kaputt) {
		t.Fatalf("Fehler %v, erwartet der Close-Fehler", err)
	}
	if !strings.HasPrefix(err.Error(), "export: schließen") {
		t.Errorf("Fehlertext %q beginnt nicht mit „export: schließen“", err)
	}
	// Der Inhalt steht trotzdem (Sync ging voraus); der Fehler meldet nur das Schließen.
	if b, _ := os.ReadFile(ziel); string(b) != "neuer Inhalt, kürzer" {
		t.Errorf("Ziel %q", b)
	}
}
