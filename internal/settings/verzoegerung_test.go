// SPDX-License-Identifier: GPL-3.0-or-later

package settings

import "testing"

// ADR-0042: [capture] delay nimmt ganze Sekunden 0 bis MaxVerzoegerung; ein ungültiger Wert warnt und lässt 0.
func TestParseVerzoegerung(t *testing.T) {
	for _, f := range []struct {
		wert  string
		soll  int
		warnt bool
	}{
		{"0", 0, false}, {"5", 5, false}, {"60", 60, false},
		{"61", 0, true}, {"-1", 0, true}, {"3.5", 0, true}, {"bald", 0, true},
	} {
		e, warn := Parse("settings.ini", "[capture]\ndelay = "+f.wert+"\n")
		if e.Verzoegerung != f.soll || (len(warn) > 0) != f.warnt {
			t.Errorf("delay = %s: %d, Warnungen %q", f.wert, e.Verzoegerung, warn)
		}
	}
	if Voreinstellungen().Verzoegerung != 0 {
		t.Error("Voreinstellung nicht 0")
	}
}

// ADR-0043: [capture] cursor yes/no (auch true/false), Voreinstellung no.
func TestParseMauszeiger(t *testing.T) {
	for wert, soll := range map[string]bool{"yes": true, "true": true, "no": false, "false": false} {
		if e, warn := Parse("settings.ini", "[capture]\ncursor = "+wert+"\n"); e.Mauszeiger != soll || len(warn) > 0 {
			t.Errorf("cursor = %s: %v, %q", wert, e.Mauszeiger, warn)
		}
	}
	if e, warn := Parse("settings.ini", "[capture]\ncursor = vielleicht\n"); e.Mauszeiger || len(warn) != 1 {
		t.Errorf("ungültig: %v, %q", e.Mauszeiger, warn)
	}
	if Voreinstellungen().Mauszeiger {
		t.Error("Voreinstellung an")
	}
}
