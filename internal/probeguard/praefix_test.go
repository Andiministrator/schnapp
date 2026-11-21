// SPDX-License-Identifier: GPL-3.0-or-later

package probeguard

import (
	"strings"
	"testing"
)

// Jeder Ablehnungsgrund trägt das Präfix genau einmal (ADR-0019 Nachtrag 2, F-339).
func TestFehlertextPraefix(t *testing.T) {
	umgebungen := []map[string]string{
		{},
		{"PROBE_INNEN": "1"},
		{"PROBE_INNEN": "1", "DBUS_SESSION_BUS_ADDRESS": "unix:path=/x"},
		{"PROBE_INNEN": "1", "DBUS_SESSION_BUS_ADDRESS": "unix:path=/x", "WAYLAND_DISPLAY": "schnapp-probe"},
	}
	for i, u := range umgebungen {
		err := NurVirtuell(func(k string) string { return u[k] })
		if err == nil {
			t.Fatalf("Fall %d: kein Fehler", i)
		}
		if s := err.Error(); !strings.HasPrefix(s, "probeguard: ") || strings.Count(s, "probeguard:") != 1 {
			t.Errorf("Fall %d: Fehlertext %q ohne genau ein Präfix", i, s)
		}
	}
}
