// SPDX-License-Identifier: GPL-3.0-or-later

package export

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// F-340: Ein Close-Fehler der temporären Datei in schreibeAtomar ist ein Fehler (anders als im Rückfall, wo der Inhalt
// nach dem Sync steht): das Ziel bleibt unberührt, die temporäre Datei wird entfernt. Der Fehler entsteht über die Naht
// oeffneTemp im eigenen Code (Hülle um die echte Datei).

type tempSchliessFehler struct {
	tempSchreiber
	fehler error
}

func (s tempSchliessFehler) Close() error {
	s.tempSchreiber.Close()
	return s.fehler
}

func biegeTempUm(t *testing.T, huelle func(tempSchreiber) tempSchreiber) {
	t.Helper()
	orig := oeffneTemp
	t.Cleanup(func() { oeffneTemp = orig })
	oeffneTemp = func(ordner string) (tempSchreiber, error) {
		f, err := orig(ordner)
		if err != nil {
			return nil, err
		}
		return huelle(f), nil
	}
}

func TestSchreibeAtomarCloseFehlerWirdGemeldet(t *testing.T) {
	for _, ohneErsetzen := range []bool{false, true} {
		kaputt := errors.New("Close")
		biegeTempUm(t, func(f tempSchreiber) tempSchreiber { return tempSchliessFehler{f, kaputt} })

		// Vorhandenes Ziel bleibt unverändert (nur beim Ersetzen), neues Ziel wird nicht angelegt.
		ordner := t.TempDir()
		neu := filepath.Join(ordner, "neu.png")
		err := schreibeAtomar(neu, ohneErsetzen, 0, schreibeNeu)
		if err == nil {
			t.Fatalf("ohneErsetzen=%v: kein Fehler trotz Close-Fehler", ohneErsetzen)
		}
		if !errors.Is(err, kaputt) {
			t.Errorf("Fehler %v trägt die Ursache nicht", err)
		}
		if !strings.HasPrefix(err.Error(), "export:") {
			t.Errorf("Fehlertext %q beginnt nicht mit „export:“", err)
		}
		if _, err := os.Lstat(neu); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("Ziel wurde angelegt: %v", err)
		}
		if es, _ := os.ReadDir(ordner); len(es) != 0 {
			t.Errorf("Rest im Ordner: %v", es)
		}
	}

	alt := []byte("alter Inhalt")
	ziel := zielMit(t, alt)
	kaputt := errors.New("Close")
	biegeTempUm(t, func(f tempSchreiber) tempSchreiber { return tempSchliessFehler{f, kaputt} })
	if err := schreibeAtomar(ziel, false, 0, schreibeNeu); !errors.Is(err, kaputt) {
		t.Fatalf("Fehler %v, erwartet die Ursache", err)
	}
	if b, _ := os.ReadFile(ziel); string(b) != string(alt) {
		t.Errorf("Ziel %q, erwartet der alte Inhalt", b)
	}
	if es, _ := os.ReadDir(filepath.Dir(ziel)); len(es) != 1 {
		t.Errorf("Rest neben dem Ziel: %v", es)
	}
}
