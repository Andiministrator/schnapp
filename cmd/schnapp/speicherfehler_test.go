package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/i18n"
)

// F-129 (1): Ein Join aus Fenster- und Speicherfehler nennt beide Teile, den übersetzten Text nur einmal und nie die
// Ursache mit ihrem Pfad.
func TestNutzerFehlerJoinFensterUndSpeichern(t *testing.T) {
	i18n.Setze(nil)
	ursache := fmt.Errorf("open /geheim/ordner/x.png: %w", fs.ErrPermission)
	fenster := errors.New("Fenster: Verbindung verloren")
	cases := map[string]error{
		"Fenster zuerst": errors.Join(fenster, speicherFehler(ursache)),
		"Dreier-Join":    errors.Join(fenster, nil, speicherFehler(ursache)),
		"verschachtelt":  errors.Join(errors.Join(fenster), speicherFehler(ursache)),
	}
	for name, err := range cases {
		got := nutzerFehler(err)
		if !strings.Contains(got, "Fenster: Verbindung verloren") {
			t.Errorf("%s: Fensterfehler fehlt in %q", name, got)
		}
		if strings.Count(got, "No permission to write the file.") != 1 {
			t.Errorf("%s: übersetzter Text nicht genau einmal in %q", name, got)
		}
		if strings.Contains(got, "/geheim/") || strings.Contains(got, "x.png") {
			t.Errorf("%s: Pfad der Ursache in %q", name, got)
		}
	}
	if got, soll := nutzerFehler(cases["Fenster zuerst"]), "Fenster: Verbindung verloren; No permission to write the file."; got != soll {
		t.Errorf("Join: %q, Soll %q", got, soll)
	}
}

// Ein einzelner Speicherfehler bleibt wie bisher nur der übersetzte Text.
func TestNutzerFehlerEinzelnerSpeicherfehlerUnveraendert(t *testing.T) {
	i18n.Setze(nil)
	err := speicherFehler(fmt.Errorf("open /geheim/ordner/x.png: %w", fs.ErrPermission))
	if got, soll := nutzerFehler(err), "No permission to write the file."; got != soll {
		t.Errorf("%q, Soll %q", got, soll)
	}
	if got := nutzerFehler(fmt.Errorf("umhüllt: %w", err)); got != "No permission to write the file." {
		t.Errorf("umhüllt: %q", got)
	}
}

// F-129 (2): Mit SCHNAPP_DEBUG steht die Ursache auf stderr, ohne nicht; Fehler ohne Speicherfehler schreiben nichts.
func TestDiagnoseUrsache(t *testing.T) {
	err := errors.Join(errors.New("Fenster"), speicherFehler(errors.New("rename /geheim/ordner/x.png: kaputt")))

	t.Setenv("SCHNAPP_DEBUG", "1")
	var an bytes.Buffer
	diagnoseUrsache(&an, err)
	if soll := "[schnapp] Ursache des Speicherfehlers: rename /geheim/ordner/x.png: kaputt\n"; an.String() != soll {
		t.Errorf("Debug an: %q, Soll %q", an.String(), soll)
	}
	an.Reset()
	diagnoseUrsache(&an, errors.New("fremd"))
	if an.Len() != 0 {
		t.Errorf("ohne Speicherfehler: %q", an.String())
	}

	t.Setenv("SCHNAPP_DEBUG", "")
	var aus bytes.Buffer
	diagnoseUrsache(&aus, err)
	if aus.Len() != 0 {
		t.Errorf("Debug aus: %q", aus.String())
	}
}
