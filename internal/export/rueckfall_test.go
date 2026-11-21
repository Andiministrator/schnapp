// SPDX-License-Identifier: GPL-3.0-or-later

package export

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// schreibgeschuetzterOrdner liefert einen Ordner, in dem man keine neue Datei anlegen, eine vorhandene aber
// beschreiben darf (das Verhalten des Dokumenten-Portals unter Flatpak, F-196).
func schreibgeschuetzterOrdner(t *testing.T) string {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("als root wirkt die Rechteprüfung nicht")
	}
	ordner := t.TempDir()
	if err := os.Chmod(ordner, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(ordner, 0o700) })
	return ordner
}

// TestSchreibeAtomarRueckfallDirekt: Darf neben dem Ziel keine temporäre Datei entstehen, wird das vorhandene Ziel
// direkt beschrieben (F-196).
func TestSchreibeAtomarRueckfallDirekt(t *testing.T) {
	eltern := t.TempDir()
	ziel := filepath.Join(eltern, "bild.png")
	if err := os.WriteFile(ziel, []byte("alter, längerer Inhalt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(eltern, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(eltern, 0o700) })
	if os.Geteuid() == 0 {
		t.Skip("als root wirkt die Rechteprüfung nicht")
	}
	var meldung string
	Diagnose = func(f string, a ...any) { meldung = f }
	t.Cleanup(func() { Diagnose = func(string, ...any) {} })
	if err := SchreibeAtomar(ziel, func(w io.Writer) error { _, err := w.Write([]byte("neu")); return err }); err != nil {
		t.Fatalf("Fehler %v", err)
	}
	if b, _ := os.ReadFile(ziel); string(b) != "neu" {
		t.Errorf("Ziel %q, erwartet „neu“", b)
	}
	if meldung == "" {
		t.Error("kein Hinweis im Debug-Log")
	}
	if n := eintraege(t, eltern); n != 1 {
		t.Errorf("%d Einträge im Ordner, soll 1", n)
	}
}

// TestSchreibeAtomarRueckfallRenderfehler: Ein Fehler beim Erzeugen lässt das Ziel im Rückfall unberührt.
func TestSchreibeAtomarRueckfallRenderfehler(t *testing.T) {
	ordner := schreibgeschuetzterOrdner(t)
	// Das Ziel muss vor dem Sperren des Ordners existieren: über einen zweiten Ordner geht das nicht, also schreibgeschützt
	// nachträglich anlegen.
	os.Chmod(ordner, 0o700)
	ziel := filepath.Join(ordner, "bild.png")
	if err := os.WriteFile(ziel, []byte("alt"), 0o600); err != nil {
		t.Fatal(err)
	}
	os.Chmod(ordner, 0o500)
	kaputt := errors.New("Renderfehler")
	if err := SchreibeAtomar(ziel, func(w io.Writer) error { w.Write([]byte("halb")); return kaputt }); !errors.Is(err, kaputt) {
		t.Fatalf("Fehler %v", err)
	}
	if b, _ := os.ReadFile(ziel); string(b) != "alt" {
		t.Errorf("Ziel verändert: %q", b)
	}
}

// TestSchreibeAtomarRueckfallNurMitVorhandenemZiel: Ohne vorhandenes Ziel und bei SchreibeOhneErsetzen bleibt es beim
// Fehler der temporären Datei (kein stilles Anlegen, kein Ersetzen).
func TestSchreibeAtomarRueckfallNurMitVorhandenemZiel(t *testing.T) {
	ordner := schreibgeschuetzterOrdner(t)
	neu := filepath.Join(ordner, "neu.png")
	if err := SchreibeAtomar(neu, func(w io.Writer) error { return nil }); !errors.Is(err, os.ErrPermission) {
		t.Errorf("Fehler %v, erwartet ErrPermission", err)
	}
	if err := SchreibeOhneErsetzen(neu, func(w io.Writer) error { return nil }); !errors.Is(err, os.ErrPermission) {
		t.Errorf("Fehler %v, erwartet ErrPermission", err)
	}
	if _, err := os.Lstat(neu); err == nil {
		t.Error("Ziel entstanden")
	}
}
