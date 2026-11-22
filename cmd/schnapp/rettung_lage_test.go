// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-454: Lage meldet nichts, solange nichts geschah; fremd, wenn ein anderes Fenster die Sperre hält; und die
// Übernahme, wenn die Sperre später frei wird.
func TestRettungLageFremdUndUebernahme(t *testing.T) {
	staat := t.TempDir()
	alt := rettungUmgebung
	rettungUmgebung = func(k string) string {
		if k == "XDG_STATE_HOME" {
			return staat
		}
		return os.Getenv(k)
	}
	t.Cleanup(func() { rettungUmgebung = alt })
	abs := filepath.Join(t.TempDir(), "datei.png")
	if err := os.WriteFile(abs, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	a := rettungFuer(abs, false)
	if a == nil {
		t.Fatal("keine Rettung")
	}
	if text, _, nr := a.Lage(); text != "" || nr != 0 {
		t.Errorf("erstes Fenster: %q (Nr %d), erwartet nichts", text, nr)
	}
	b := rettungFuer(abs, false)
	if b == nil {
		t.Fatal("keine zweite Rettung")
	}
	defer b.Ende()
	text, fehler, nr := b.Lage()
	if !strings.Contains(text, "open in another Schnapp window") || fehler || nr == 0 {
		t.Errorf("zweites Fenster: %q (Fehler %v, Nr %d)", text, fehler, nr)
	}
	a.Ende()
	d, err := document.Neu(testbild(20, 10, 3))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(1, 1), document.Pt(5, 5))}); err != nil {
		t.Fatal(err)
	}
	b.Sichere(d)
	b.s.Ende() // wartet auf die Kette: die Übernahme der Sperre ist durch
	text2, fehler2, nr2 := b.Lage()
	if !strings.Contains(text2, "protected by a rescue copy again") || fehler2 || nr2 <= nr {
		t.Errorf("nach der Übernahme: %q (Fehler %v, Nr %d, vorher %d)", text2, fehler2, nr2, nr)
	}
}
