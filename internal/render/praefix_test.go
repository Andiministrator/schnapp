// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"

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

// Die Schrift lässt sich nicht kaputt machen (eingebettete Go-Schriften); der Fehlerweg wird nachgestellt: Das Anlegen
// des Face liefert einen rohen Fehler ohne Präfix, danach wird es zurückgesetzt. Bild und PNG müssen ihn hüllen.
func TestFehlertextPraefixSchriftfehler(t *testing.T) {
	d := weissesDokument(t, 100, 40)
	hinzu(t, d, document.Text{Ort: document.Pt(5, 5), Inhalt: "Hallo", Schriftgrad: 20, Farbe: weiss})
	if _, err := Bild(d); err != nil { // lädt die Schriften echt
		t.Fatal(err)
	}
	alt := neueFace
	roh := errors.New("Schrift kaputt")
	neueFace = func(*sfnt.Font, *opentype.FaceOptions) (font.Face, error) { return nil, roh }
	defer func() { neueFace = alt }()

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
