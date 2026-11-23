// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"strings"
	"testing"
)

// F-486: B war fremd, A schließt, B speichert unter demselben Namen: Das ist die Übernahme der Sperre, kein Speichern
// als andere Datei.
func TestRettungLageFremdDannDieselbeDatei(t *testing.T) {
	_, abs1, _ := lageUmgebung(t)
	a := rettungFuer(abs1, false)
	b := rettungFuer(abs1, false)
	if text, _, _ := b.Lage(); !strings.Contains(text, "open in another Schnapp window") {
		t.Fatalf("B nicht fremd: %q", text)
	}
	a.Ende() // das andere Fenster lässt los
	b.nachSpeichern()(abs1)
	b.s.Ende()
	text, fehler, nr := b.Lage()
	if strings.Contains(text, "different file") || !strings.Contains(text, "has let go") || fehler || nr == 0 {
		t.Errorf("Lage nach Speichern derselben Datei: %q (Fehler %v, Nr %d)", text, fehler, nr)
	}
}
