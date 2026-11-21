// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"strings"
	"testing"
	"unsafe"
)

// F-241: Die Speichergrenze des Tinten-Zwischenspeichers zählt Eintrag und Zeilen, nicht nur den Inhalt.

// Die Schätzung je Zeile ist die echte Größe einer zeilenTinte.
func TestTintenZeileIstGroesseDerZeile(t *testing.T) {
	if g := int(unsafe.Sizeof(zeilenTinte{})); g != tintenZeile {
		t.Errorf("zeilenTinte belegt %d Byte, tintenZeile sagt %d", g, tintenZeile)
	}
}

// mitLeeremSpeicher leert den Zwischenspeicher, setzt die Grenze und stellt beides danach wieder her.
func mitLeeremSpeicher(t *testing.T, grenze int) *schriftMass {
	t.Helper()
	schriftMu.Lock()
	t.Cleanup(schriftMu.Unlock)
	alt := tintenSpeicherMax
	clear(tintenSpeicher)
	tintenSpeicherBytes, tintenSpeicherMax = 0, grenze
	t.Cleanup(func() {
		clear(tintenSpeicher)
		tintenSpeicherBytes, tintenSpeicherMax = 0, alt
	})
	s := schriftFuer(false)
	if s == nil {
		t.Skip("Schrift nicht ladbar")
	}
	return s
}

// Ein zeilenreicher Text mit wenig Inhalt übersteigt die Grenze und leert den Speicher; danach zählt nur er.
func TestTintenSpeicherLeertNachZeilen(t *testing.T) {
	s := mitLeeremSpeicher(t, 4096)
	textZeilen(s, "a", 12)
	klein := tintenSchluessel{"a", 12}
	if _, ok := tintenSpeicher[klein]; !ok {
		t.Fatal("Vorbedingung: kleiner Eintrag gespeichert")
	}
	viele := strings.Repeat("a\n", 99) + "a" // 199 Byte Inhalt, 100 Zeilen mit Tinte
	z := textZeilen(s, viele, 12)
	if len(z) != 100 {
		t.Fatalf("%d Zeilen, Soll 100", len(z))
	}
	if _, ok := tintenSpeicher[klein]; ok {
		t.Error("Speicher nicht geleert, obwohl 100 Zeilen die Grenze übersteigen")
	}
	if len(tintenSpeicher) != 1 || tintenSpeicherBytes != tintenKosten(viele, cap(z)) {
		t.Errorf("%d Einträge, %d Byte (Soll 1 und %d)", len(tintenSpeicher), tintenSpeicherBytes, tintenKosten(viele, cap(z)))
	}
}

// Auch ein leerer Inhalt belegt einen Eintrag und wird gezählt.
func TestTintenSpeicherZaehltLeerenInhalt(t *testing.T) {
	s := mitLeeremSpeicher(t, 1<<20)
	textZeilen(s, "", 12)
	if tintenSpeicherBytes < tintenEintrag {
		t.Errorf("leerer Inhalt zählt %d Byte, Soll mindestens %d", tintenSpeicherBytes, tintenEintrag)
	}
	// Ein zweiter Abruf kommt aus dem Speicher und zählt nicht erneut.
	vor := tintenSpeicherBytes
	textZeilen(s, "", 12)
	if tintenSpeicherBytes != vor {
		t.Errorf("Treffer zählt erneut: %d → %d", vor, tintenSpeicherBytes)
	}
}
