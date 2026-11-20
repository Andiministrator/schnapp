// SPDX-License-Identifier: GPL-3.0-or-later

//go:build probe

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestNurVirtuell: ohne den virtuellen KWin von scripts/probe.sh (Socket „schnapp-probe“, Laufzeitordner tmp/probe/run)
// meldet nurVirtuell einen Fehler.
func TestNurVirtuell(t *testing.T) {
	for _, f := range []struct {
		name, display, run string
		fehler             bool
	}{
		{"virtuell", "schnapp-probe", "/home/x/schnapp/tmp/probe/run", false},
		{"echte Sitzung", "wayland-0", "/run/user/1000", true},
		{"nur Display passt", "schnapp-probe", "/run/user/1000", true},
		{"nur Ordner passt", "wayland-0", "/home/x/schnapp/tmp/probe/run", true},
		{"Display mit Zusatz", "schnapp-probe2", "/a/tmp/probe/run", true},
		{"Display absolut", "/run/user/1000/schnapp-probe", "/a/tmp/probe/run", true},
		{"alles leer", "", "", true},
		{"Ordner ähnlich", "schnapp-probe", "/a/tmp/probe/runx", true},
	} {
		env := map[string]string{"WAYLAND_DISPLAY": f.display, "XDG_RUNTIME_DIR": f.run, "PROBE_INNEN": "1",
			"DBUS_SESSION_BUS_ADDRESS": "unix:path=/a/privat", "PROBE_AUSSENBUS": "unix:path=/run/user/1000/bus"}
		if err := nurVirtuell(func(k string) string { return env[k] }); (err != nil) != f.fehler {
			t.Errorf("%s: Fehler %v, erwartet Fehler=%v", f.name, err, f.fehler)
		}
	}
	// D-Bus-Weg (Verify N-5): ohne Marke, ohne Busadresse oder mit dem äußeren Bus wird abgelehnt.
	basis := map[string]string{"WAYLAND_DISPLAY": "schnapp-probe", "XDG_RUNTIME_DIR": "/a/tmp/probe/run", "PROBE_INNEN": "1",
		"DBUS_SESSION_BUS_ADDRESS": "unix:path=/a/privat", "PROBE_AUSSENBUS": "unix:path=/run/user/1000/bus"}
	for name, aendern := range map[string]func(map[string]string){
		"ohne PROBE_INNEN": func(m map[string]string) { delete(m, "PROBE_INNEN") },
		"ohne Busadresse":  func(m map[string]string) { m["DBUS_SESSION_BUS_ADDRESS"] = "" },
		"äußerer Bus":      func(m map[string]string) { m["DBUS_SESSION_BUS_ADDRESS"] = m["PROBE_AUSSENBUS"] },
	} {
		env := map[string]string{}
		for k, v := range basis {
			env[k] = v
		}
		aendern(env)
		if err := nurVirtuell(func(k string) string { return env[k] }); err == nil {
			t.Errorf("%s: angenommen", name)
		}
	}
}

// TestLaufNurVirtuell: In einer Umgebung ohne virtuellen KWin endet lauf mit Exit 1 und ohne Verbindungsversuch (ein
// Versuch würde „Verbindung:“ melden, nicht „Abbruch:“).
func TestLaufNurVirtuell(t *testing.T) {
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	buch := filepath.Join(t.TempDir(), "drehbuch.txt")
	if err := os.WriteFile(buch, []byte("warte 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var aus, fehler bytes.Buffer
	if code := lauf([]string{buch}, &aus, &fehler); code != 1 {
		t.Errorf("Exit %d, erwartet 1", code)
	}
	if !bytes.HasPrefix(fehler.Bytes(), []byte("Abbruch:")) {
		t.Errorf("Meldung %q, erwartet „Abbruch: …“", fehler.String())
	}
}
