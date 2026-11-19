// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"errors"
	"image"
	"io"
	"io/fs"
	"os"
	"testing"

	"github.com/Andiministrator/schnapp/fileformat"
)

// ersetzeVorschauNaht tauscht kodiereVorschau und entferneDatei für die Dauer des Tests (nil = unverändert).
func ersetzeVorschauNaht(t *testing.T, kodiere func(io.Writer, image.Image) error, entferne func(string) error) {
	t.Helper()
	altK, altE := kodiereVorschau, entferneDatei
	if kodiere != nil {
		kodiereVorschau = kodiere
	}
	if entferne != nil {
		entferneDatei = entferne
	}
	t.Cleanup(func() { kodiereVorschau, entferneDatei = altK, altE })
}

var errKodieren = errors.New("Probe: Vorschau nicht kodierbar")

// Scheitert das Neuschreiben der Vorschau beim Speichern, verschwindet die alte: Sie zeigte sonst den vorigen
// Stand (nach dem Schwärzen den ungeschwärzten, F-115). Das Speichern selbst gilt (ErrVorschau, kein ErrVorschauAlt).
func TestSpeichernEntferntVeralteteVorschau(t *testing.T) {
	b, _ := neueBibliothek(t)
	d := neuesDokument(t, 40, 30)
	e, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{})
	if err != nil {
		t.Fatal(err)
	}
	pfad := b.VorschauPfad(e.Pfad)
	if _, err := os.Stat(pfad); err != nil {
		t.Fatalf("Vorschau nach Ablegen fehlt: %v", err)
	}
	ersetzeVorschauNaht(t, func(io.Writer, image.Image) error { return errKodieren }, nil)
	_, _, err = b.Speichern(e.Pfad, standIn(t, b, e.Pfad), d, d.Ansichtsbild())
	if !errors.Is(err, ErrVorschau) || errors.Is(err, ErrVorschauAlt) {
		t.Fatalf("Fehler %v (Soll: ErrVorschau ohne ErrVorschauAlt)", err)
	}
	if _, err := os.Stat(pfad); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("veraltete Vorschau liegt noch: %v", err)
	}
}

// Lässt sich auch die alte Vorschau nicht entfernen, meldet Speichern ErrVorschauAlt.
func TestSpeichernMeldetBleibendeVorschau(t *testing.T) {
	b, _ := neueBibliothek(t)
	d := neuesDokument(t, 40, 30)
	e, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{})
	if err != nil {
		t.Fatal(err)
	}
	ersetzeVorschauNaht(t, func(io.Writer, image.Image) error { return errKodieren },
		func(string) error { return fs.ErrPermission })
	_, _, err = b.Speichern(e.Pfad, standIn(t, b, e.Pfad), d, d.Ansichtsbild())
	if !errors.Is(err, ErrVorschauAlt) || !errors.Is(err, ErrVorschau) {
		t.Fatalf("Fehler %v (Soll: ErrVorschau und ErrVorschauAlt)", err)
	}
	if _, err := os.Stat(b.VorschauPfad(e.Pfad)); err != nil {
		t.Errorf("Probe: alte Vorschau sollte noch liegen: %v", err)
	}
}
