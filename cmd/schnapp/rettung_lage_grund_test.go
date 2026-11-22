// SPDX-License-Identifier: GPL-3.0-or-later

package main

// Tests zu F-472 (Text der Rettungs-Lage je Auslöser) und F-473 (gestört meldet einen Fehler). Aus den Proben des
// Kritikers B (Sprint 12), mit dem Zustandsordner und den Dateien je Test.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/rescue"
)

// lageUmgebung stellt einen eigenen Zustandsordner ein und legt zwei Ausgangsdateien an.
func lageUmgebung(t *testing.T) (staat, abs1, abs2 string) {
	t.Helper()
	staat = t.TempDir()
	alt := rettungUmgebung
	rettungUmgebung = func(k string) string {
		if k == "XDG_STATE_HOME" {
			return staat
		}
		return os.Getenv(k)
	}
	t.Cleanup(func() { rettungUmgebung = alt })
	ordner := t.TempDir()
	abs1, abs2 = filepath.Join(ordner, "eins.png"), filepath.Join(ordner, "zwei.png")
	for _, p := range []string{abs1, abs2} {
		if err := os.WriteFile(p, []byte(p), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return staat, abs1, abs2
}

func sperrnameVon(pfad string) string {
	return strings.TrimSuffix(rescue.Name(pfad), ".json") + ".lock"
}

// F-472: Wer fremd war und als andere Datei speichert, hat nichts von einem anderen Fenster übernommen: Der Text sagt es
// ehrlich, und das andere Fenster hält seine Datei weiter.
func TestRettungLageFremdDannAndereDatei(t *testing.T) {
	_, abs1, abs2 := lageUmgebung(t)
	a := rettungFuer(abs1, false)
	defer a.Ende()
	b := rettungFuer(abs1, false)
	if text, _, _ := b.Lage(); !strings.Contains(text, "open in another Schnapp window") {
		t.Fatalf("B nicht fremd: %q", text)
	}
	b.nachSpeichern()(abs2)
	b.s.Ende()
	text, fehler, nr := b.Lage()
	if strings.Contains(text, "let go") || !strings.Contains(text, "Saved as a different file") || fehler || nr == 0 {
		t.Errorf("Lage nach Basiswechsel: %q (Fehler %v, Nr %d)", text, fehler, nr)
	}
}

// F-472 und F-473: Eine gestörte Sitzung meldet einen Fehler; nach dem Speichern als andere Datei ist sie geschützt,
// ohne dass ein Fenster losgelassen hätte.
func TestRettungLageGestoertDannAndereDatei(t *testing.T) {
	staat, abs1, abs2 := lageUmgebung(t)
	ort := filepath.Join(staat, "schnapp", "rescue")
	if err := os.MkdirAll(filepath.Join(ort, sperrnameVon(abs1)), 0o700); err != nil {
		t.Fatal(err)
	}
	c := rettungFuer(abs1, false)
	text, fehler, _ := c.Lage()
	if !strings.Contains(text, "unavailable") || !fehler {
		t.Fatalf("gestört: %q (Fehler %v), erwartet Text und Fehlerkennzeichen", text, fehler)
	}
	c.nachSpeichern()(abs2)
	c.s.Ende()
	text, fehler, _ = c.Lage()
	if strings.Contains(text, "let go") || !strings.Contains(text, "Saved as a different file") || fehler {
		t.Errorf("Lage nach Basiswechsel: %q (Fehler %v)", text, fehler)
	}
}

// F-472: Gelingt die Übernahme der Sperre, die hinterlassene Kopie ließ sich aber nicht beiseitelegen, ist die Sitzung
// stumm: Die Meldung sagt „nicht geschützt“ als Fehler, nie „wieder geschützt“. Ein Speichern als andere Datei hebt das auf.
func TestRettungLageUebernahmeMitAus(t *testing.T) {
	staat, abs1, abs2 := lageUmgebung(t)
	ort := filepath.Join(staat, "schnapp", "rescue")
	d, err := document.Neu(testbild(20, 10, 3))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(1, 1), document.Pt(5, 5))}); err != nil {
		t.Fatal(err)
	}
	a := rettungFuer(abs1, false)
	b := rettungFuer(abs1, false)
	defer b.Ende()
	a.Sichere(d)
	a.Ende() // die Kopie von A bleibt liegen, die Sperre ist frei
	kopie := filepath.Join(ort, rescue.Name(abs1))
	if _, err := os.Lstat(kopie); err != nil {
		t.Fatalf("Kopie von A fehlt: %v", err)
	}
	// Beiseitelegen scheitert (Ordner ohne Schreibrecht), die Sperre lässt sich noch nehmen: Die Sperrdatei steht schon da.
	if err := os.WriteFile(filepath.Join(ort, sperrnameVon(abs1)), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(ort, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(ort, 0o700) })
	if _, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(2, 2), document.Pt(6, 6))}); err != nil {
		t.Fatal(err)
	}
	b.Sichere(d)
	b.s.Ende()
	text, fehler, nr := b.Lage()
	if strings.Contains(text, "protected by a rescue copy again") {
		t.Errorf("Lage %q: Die Sitzung ist stumm, nicht geschützt", text)
	}
	if !strings.Contains(text, "could not be set aside") || !fehler || nr == 0 {
		t.Errorf("Lage %q (Fehler %v, Nr %d), erwartet „nicht beiseite zu legen“ als Fehler", text, fehler, nr)
	}
	if err := os.Chmod(ort, 0o700); err != nil {
		t.Fatal(err)
	}
	b.nachSpeichern()(abs2)
	b.s.Ende()
	text, fehler, _ = b.Lage()
	if !strings.Contains(text, "Saved as a different file") || fehler {
		t.Errorf("Lage nach Basiswechsel: %q (Fehler %v)", text, fehler)
	}
}
