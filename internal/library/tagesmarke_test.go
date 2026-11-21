// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTagesmarkePfad(t *testing.T) {
	umgebung := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	for _, tc := range []struct {
		env  map[string]string
		soll string
	}{
		{map[string]string{"XDG_STATE_HOME": "/zustand", "HOME": "/heim"}, "/zustand/schnapp/last-cleanup"},
		{map[string]string{"XDG_STATE_HOME": "relativ", "HOME": "/heim"}, "/heim/.local/state/schnapp/last-cleanup"},
		{map[string]string{"HOME": "relativ"}, ""},
		{map[string]string{}, ""},
	} {
		if got := TagesmarkePfad(umgebung(tc.env)); got != tc.soll {
			t.Errorf("%v: %q, Soll %q", tc.env, got, tc.soll)
		}
	}
}

// Höchstens einmal am Kalendertag (Ortszeit), am nächsten Tag wieder; Rechte 0700/0600.
func TestBeanspruchenHeute(t *testing.T) {
	marke := filepath.Join(t.TempDir(), "state", "schnapp", "last-cleanup")
	ort := time.FixedZone("Test", 3600)
	morgens := time.Date(2026, 11, 19, 0, 30, 0, 0, ort)
	abends := time.Date(2026, 11, 19, 23, 59, 0, 0, ort)
	naechster := time.Date(2026, 11, 20, 0, 0, 1, 0, ort)

	for i, tc := range []struct {
		jetzt time.Time
		soll  bool
	}{{morgens, true}, {abends, false}, {naechster, true}, {naechster, false}} {
		got, err := BeanspruchenHeute(marke, tc.jetzt)
		if err != nil || got != tc.soll {
			t.Fatalf("Schritt %d (%v): %v, %v; Soll %v", i+1, tc.jetzt, got, err, tc.soll)
		}
	}
	daten, err := os.ReadFile(marke)
	if err != nil || string(daten) != "2026-11-20\n" {
		t.Fatalf("Marke %q, %v", daten, err)
	}
	st, err := os.Stat(marke)
	if err != nil || st.Mode().Perm() != 0o600 {
		t.Fatalf("Rechte Datei %v, %v", st.Mode(), err)
	}
	st, err = os.Stat(filepath.Dir(marke))
	if err != nil || st.Mode().Perm() != 0o700 {
		t.Fatalf("Rechte Ordner %v, %v", st.Mode(), err)
	}
}

// Fremder Inhalt gilt als „heute noch nicht“; ohne Pfad kommt ErrKeinZustandsordner.
func TestBeanspruchenHeuteSonderfaelle(t *testing.T) {
	marke := filepath.Join(t.TempDir(), "last-cleanup")
	if err := os.WriteFile(marke, []byte("Unsinn"), 0o600); err != nil {
		t.Fatal(err)
	}
	jetzt := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	if got, err := BeanspruchenHeute(marke, jetzt); !got || err != nil {
		t.Errorf("fremder Inhalt: %v, %v", got, err)
	}
	if got, err := BeanspruchenHeute("", jetzt); got || !errors.Is(err, ErrKeinZustandsordner) {
		t.Errorf("ohne Pfad: %v, %v", got, err)
	}
}
