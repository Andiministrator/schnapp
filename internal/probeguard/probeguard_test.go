// SPDX-License-Identifier: GPL-3.0-or-later

package probeguard

import "testing"

// TestNurVirtuell: Nur die vollständige Probe-Umgebung von scripts/probe.sh wird angenommen; jede einzelne fehlende
// oder falsche Angabe lehnt ab (F-229).
func TestNurVirtuell(t *testing.T) {
	voll := map[string]string{"PROBE_INNEN": "1", "PROBE_AUSSENBUS": "unix:path=/run/user/1000/bus",
		"DBUS_SESSION_BUS_ADDRESS": "unix:path=/a/privat", "WAYLAND_DISPLAY": "schnapp-probe", "XDG_RUNTIME_DIR": "/a/tmp/probe/run"}
	hole := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	if err := NurVirtuell(hole(voll)); err != nil {
		t.Fatalf("vollständige Probe-Umgebung abgelehnt: %v", err)
	}
	for name, aendern := range map[string]func(map[string]string){
		"ohne Marke":       func(m map[string]string) { delete(m, "PROBE_INNEN") },
		"Marke 0":          func(m map[string]string) { m["PROBE_INNEN"] = "0" },
		"ohne Busadresse":  func(m map[string]string) { m["DBUS_SESSION_BUS_ADDRESS"] = "" },
		"äußerer Bus":      func(m map[string]string) { m["DBUS_SESSION_BUS_ADDRESS"] = m["PROBE_AUSSENBUS"] },
		"echter Wayland":   func(m map[string]string) { m["WAYLAND_DISPLAY"] = "wayland-0" },
		"echter Laufzeit":  func(m map[string]string) { m["XDG_RUNTIME_DIR"] = "/run/user/1000" },
		"Laufzeit ähnlich": func(m map[string]string) { m["XDG_RUNTIME_DIR"] = "/a/tmp/probe/runx" },
	} {
		m := map[string]string{}
		for k, v := range voll {
			m[k] = v
		}
		aendern(m)
		if err := NurVirtuell(hole(m)); err == nil {
			t.Errorf("%s: angenommen", name)
		}
	}
}
