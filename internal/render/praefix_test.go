// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"errors"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// einmalPraefix prüft die Fehlertext-Regel (ADR-0019 Nachtrag 2, F-379): Der Text beginnt mit "render: " und das
// Präfix kommt genau einmal vor (keine doppelte Hülle).
func einmalPraefix(t *testing.T, name string, err error) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: kein Fehler", name)
		return
	}
	s := err.Error()
	if !strings.HasPrefix(s, "render: ") || strings.Count(s, "render:") != 1 {
		t.Errorf("%s: Fehlertext %q ohne genau ein Präfix \"render: \"", name, s)
	}
}

// Die Schrift lässt sich nicht kaputt machen (eingebettete Go-Schriften); der Fehlerweg wird nachgestellt: Der
// Einmal-Lauf ist erledigt, die gemerkte Ursache ein roher Fehler ohne Präfix. Bild und PNG müssen ihn hüllen.
func TestFehlertextPraefixSchriftfehler(t *testing.T) {
	schriftEinmal.Do(func() {})
	alt := schriftFehler
	roh := errors.New("Schrift kaputt")
	schriftFehler = roh
	defer func() { schriftFehler = alt }()

	d := weissesDokument(t, 100, 40)
	hinzu(t, d, document.Text{Ort: document.Pt(5, 5), Inhalt: "Hallo", Schriftgrad: 20, Farbe: weiss})

	_, err := Bild(d)
	einmalPraefix(t, "Bild", err)
	if !errors.Is(err, roh) {
		t.Errorf("Ursache von Bild verloren: %v", err)
	}
	_, err = PNG(d)
	einmalPraefix(t, "PNG", err)
	if !errors.Is(err, roh) {
		t.Errorf("Ursache von PNG verloren: %v", err)
	}
}

func TestFehlertextPraefixPruefe(t *testing.T) {
	einmalPraefix(t, "ErrZuAufwendig", ErrZuAufwendig)
	alt := grenzen
	defer func() { grenzen = alt }()
	grenzen.zeichen = 3
	d := weissesDokument(t, 100, 40)
	hinzu(t, d, document.Text{Ort: document.Pt(5, 5), Inhalt: "Hallo", Schriftgrad: 20, Farbe: weiss})
	err := Pruefe(d)
	einmalPraefix(t, "Pruefe Zeichen", err)
	if !errors.Is(err, ErrZuAufwendig) {
		t.Errorf("ErrZuAufwendig verloren: %v", err)
	}
}
