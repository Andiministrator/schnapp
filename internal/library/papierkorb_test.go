// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
)

// ADR-0036 Punkt 4: Entf in der vollen Ansicht legt eine Aufnahme in den Papierkorb, auch mit Schlagworten, bearbeitet
// oder jung (keine Schutzregeln des Aufräumens); sie fällt aus dem Index, die Vorschau verschwindet, .trashinfo nennt
// den alten Ort.
func TestInPapierkorb(t *testing.T) {
	a := neuerBestand(t)
	if err := os.MkdirAll(filepath.Dir(a.b.VorschauPfad(a.wort.Pfad)), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(a.b.VorschauPfad(a.wort.Pfad), []byte("png"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, e := range []Eintrag{a.wort, a.jung, a.bearb} {
		v, err := a.b.InPapierkorb(e.Pfad, a.jetzt)
		if err != nil {
			t.Fatalf("%s: %v", e.Pfad, err)
		}
		if v.Pfad != e.Pfad || a.dateiDa(t, e) {
			t.Errorf("%s: %+v, Datei noch da: %v", e.Pfad, v, a.dateiDa(t, e))
		}
		if _, err := os.Stat(v.Papierkorb); err != nil {
			t.Errorf("nicht im Papierkorb: %v", err)
		}
		info, err := os.ReadFile(filepath.Join(a.b.Orte.Papierkorb, "info", filepath.Base(v.Papierkorb)+".trashinfo"))
		if err != nil || !strings.Contains(string(info), "Path="+filepath.ToSlash(filepath.Join(a.b.Orte.Ablage, e.Pfad))) {
			t.Errorf("trashinfo %q, %v", info, err)
		}
	}
	if _, err := os.Stat(a.b.VorschauPfad(a.wort.Pfad)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Vorschau noch da: %v", err)
	}
	if got := a.indexPfade(t); !reflect.DeepEqual(got, []string{a.alt.Pfad}) {
		t.Errorf("Index %v", got)
	}
}

// Nur gültige Pfade in der Ablage und nur reguläre Dateien; ein zweiter Versuch findet nichts mehr.
func TestInPapierkorbAbgelehnt(t *testing.T) {
	a := neuerBestand(t)
	for _, rel := range []string{"../x.png", "/abs.png", "", "2026-05/x.txt"} {
		if _, err := a.b.InPapierkorb(rel, a.jetzt); !errors.Is(err, ErrPfad) {
			t.Errorf("%q: %v", rel, err)
		}
	}
	verweis := "2026-05/verweis.png"
	if err := os.Symlink(filepath.Join(a.b.Orte.Ablage, filepath.FromSlash(a.alt.Pfad)),
		filepath.Join(a.b.Orte.Ablage, filepath.FromSlash(verweis))); err != nil {
		t.Fatal(err)
	}
	if _, err := a.b.InPapierkorb(verweis, a.jetzt); !errors.Is(err, ErrPfad) {
		t.Errorf("Verweis: %v", err)
	}
	if !a.dateiDa(t, a.alt) {
		t.Error("Ziel des Verweises bewegt")
	}
	if _, err := a.b.InPapierkorb(a.alt.Pfad, a.jetzt); err != nil {
		t.Fatal(err)
	}
	if _, err := a.b.InPapierkorb(a.alt.Pfad, a.jetzt); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("zweimal: %v", err)
	}
	if !strings.HasPrefix(func() string { _, err := a.b.InPapierkorb("../x.png", a.jetzt); return err.Error() }(), "library: ") {
		t.Error("Fehler ohne Paketpräfix")
	}
}

// Anderes Dateisystem: die Aufnahme bleibt liegen und im Index, die Reservierung im Papierkorb wird zurückgenommen.
func TestInPapierkorbAnderesDateisystem(t *testing.T) {
	a := neuerBestand(t)
	alt := umbenenne
	umbenenne = func(string, string) error { return &os.LinkError{Op: "rename", Err: syscall.EXDEV} }
	defer func() { umbenenne = alt }()
	if _, err := a.b.InPapierkorb(a.wort.Pfad, a.jetzt); !errors.Is(err, ErrAnderesDateisystem) {
		t.Fatalf("%v", err)
	}
	if !a.dateiDa(t, a.wort) || len(a.indexPfade(t)) != 4 {
		t.Error("Aufnahme oder Index verändert")
	}
	infos, _ := os.ReadDir(filepath.Join(a.b.Orte.Papierkorb, "info"))
	if len(infos) != 0 {
		t.Errorf("Reservierung liegt noch: %v", infos)
	}
}
