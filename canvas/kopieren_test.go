// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import "testing"

// Karte #39: Strg+C kopiert das Bild nur außerhalb der Texteingabe.
func TestKopierenErlaubt(t *testing.T) {
	s := &Sitzung{}
	if !s.KopierenErlaubt() {
		t.Error("ohne Texteingabe muss Kopieren erlaubt sein")
	}
	s.zug = zugText
	if s.KopierenErlaubt() {
		t.Error("während der Texteingabe darf Strg+C nicht kopieren")
	}
}
