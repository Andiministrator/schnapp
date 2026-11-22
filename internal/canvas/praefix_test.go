// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"strings"
	"testing"
)

// F-379 (ADR-0019 Nachtrag 2): Die exportierten Fehlerwege des Pakets sind die Sentinel-Fehler von SpeichernJetzt
// (ihr Text beginnt mit "canvas: ", genau einmal); SpeichernEnde nimmt einen Fehler nur entgegen, gibt aber keinen
// zurück, und die übrigen exportierten Funktionen haben keine Fehlerrückgabe.
func TestFehlertextPraefixSentinels(t *testing.T) {
	for name, e := range map[string]error{
		"ErrNichtsZuSpeichern": ErrNichtsZuSpeichern, "ErrSpeichernLaeuft": ErrSpeichernLaeuft, "ErrZugLaeuft": ErrZugLaeuft,
	} {
		if s := e.Error(); !strings.HasPrefix(s, "canvas: ") || strings.Count(s, "canvas:") != 1 {
			t.Errorf("%s: Fehlertext %q ohne genau ein Präfix \"canvas: \"", name, s)
		}
	}
}
