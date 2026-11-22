// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// Karte #23: Das Go-Werkzeug erzeugt die eingecheckten SVGs byte-gleich (wie früher paket.py und cd.py). Ohne die
// Schriftdatei (etwa in der CI) wird der Test übersprungen, bei jedem anderen Fehler nicht (Sitzungs-QA, QB-4).
func TestSVGWieEingecheckt(t *testing.T) {
	b, err := neuerBauer()
	if errors.Is(err, os.ErrNotExist) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	logo := func(art string, mit, grund bool) func() (string, error) {
		return func() (string, error) { return b.logo(art, mit, grund) }
	}
	fest := func(s string) func() (string, error) { return func() (string, error) { return s, nil } }
	faelle := map[string]func() (string, error){
		"logo/svg/schnapp-logo-dunkel.svg":                            logo("dunkel", true, false),
		"logo/svg/schnapp-logo-hell-mit-grund.svg":                    logo("hell", true, true),
		"logo/svg/schnapp-logo-ohne-unterzeile-dunkel.svg":            logo("dunkel", false, false),
		"logo/svg/schnapp-logo-einfarbig-weiss.svg":                   logo("weiss", true, false),
		"logo/svg/schnapp-logo-ohne-unterzeile-einfarbig-schwarz.svg": logo("schwarz", false, false),
		"logo/svg/schnapp-symbol-hell.svg":                            fest(symbol("hell", true)),
		"logo/svg/schnapp-s-dunkel.svg":                               fest(symbol("dunkel", false)),
		"logo/favicon/favicon.svg":                                    fest(symbol("dunkel", true)),
		"logo/sozial/schnapp-sozial-1280x640.svg":                     b.sozial,
		"cd/farben.svg":     b.farbfelder,
		"cd/schutzraum.svg": b.schutzraum,
	}
	for name, f := range faelle {
		soll, err := os.ReadFile(filepath.Join(b.marke, name))
		if err != nil {
			t.Fatal(err)
		}
		ist, err := f()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if ist != string(soll) {
			t.Errorf("%s: weicht ab (%d gegen %d Bytes)", name, len(ist), len(soll))
		}
	}
}

func TestPyZahl(t *testing.T) {
	for f, soll := range map[float64]string{0: "0.0", 241: "241.0", -41: "-41.0", 382.0390625: "382.0390625",
		0.30000000000000004: "0.30000000000000004", 1e-5: "1e-05", 1.5e16: "1.5e+16", 123456789012345.6: "123456789012345.6"} {
		if ist := pyZahl(f); ist != soll {
			t.Errorf("pyZahl(%v) = %q, soll %q", f, ist, soll)
		}
	}
}
