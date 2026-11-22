// SPDX-License-Identifier: GPL-3.0-or-later

package main

// F-478: Unbekannte JSON-Schlüssel und ungültige timeout-Werte scheitern beim Laden (auch mit -trocken); ein Mutant, der
// einen Test am Test-Timeout hängen lässt, hat einen eigenen Status und zählt als rot.

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func trockenMit(t *testing.T, mutanten string) (int, string) {
	t.Helper()
	q := kritikWurzel(t, "package p\n\nimport \"testing\"\n\nfunc TestF(t *testing.T) {\n\tif F() != 1 {\n\t\tt.Fatal(\"F\")\n\t}\n}\n", mutanten)
	var aus bytes.Buffer
	code := lauf([]string{"-wurzel", q, "-mutanten", filepath.Join(q, "mutanten"), "-trocken"}, &aus)
	return code, aus.String()
}

func TestUnbekannterSchluesselScheitertTrocken(t *testing.T) {
	code, aus := trockenMit(t, `[{"name":"x","datei":"p/p.go","paket":"./p","alt":"return 1","neu":"return 2","timout":"60s"}]`)
	if code == 0 || !strings.Contains(aus, "timout") {
		t.Errorf("Exit %d, erwartet Fehler mit dem Schlüssel „timout“:\n%s", code, aus)
	}
}

func TestUngueltigerTimeoutScheitertTrocken(t *testing.T) {
	for _, tm := range []string{"abc", "60", "-5s", "0s"} {
		code, aus := trockenMit(t, `[{"name":"x","datei":"p/p.go","paket":"./p","alt":"return 1","neu":"return 2","timeout":"`+tm+`"}]`)
		if code == 0 || !strings.Contains(aus, "timeout") {
			t.Errorf("timeout %q: Exit %d, erwartet Fehler:\n%s", tm, code, aus)
		}
	}
	if code, aus := trockenMit(t, `[{"name":"x","datei":"p/p.go","paket":"./p","alt":"return 1","neu":"return 2","timeout":"45s"}]`); code != 0 {
		t.Errorf("gültiger timeout: Exit %d:\n%s", code, aus)
	}
}

func TestHaengendeTestsLesenRunningTests(t *testing.T) {
	text := "panic: test timed out after 1s\n\trunning tests:\n\t\tTestA (1s)\n\t\tTestB (0s)\n\ngoroutine 1 [running]:\n"
	if got := haengendeTests(text); !reflect.DeepEqual(got, []string{"TestA", "TestB"}) {
		t.Errorf("Tests %v", got)
	}
	// Timeout ohne laufenden Test (Hänger in init oder TestMain): kein Beleg, dass ein Test den Mutanten bemerkt hat.
	if got := haengendeTests("panic: test timed out after 1s\n\ngoroutine 1 [running]:\n"); len(got) != 0 {
		t.Errorf("ohne laufenden Test: %v", got)
	}
	if got := haengendeTests("--- FAIL: TestX (0.00s)\nFAIL\n"); len(got) != 0 {
		t.Errorf("ohne Timeout: %v", got)
	}
}

// Ein Mutant, der den Test hängen lässt, endet am Test-Timeout: Status HÄNGT, Exit 0, der Test ist genannt.
func TestHaengerIstEigenerStatusUndRot(t *testing.T) {
	q := kritikWurzel(t,
		"package p\n\nimport (\n\t\"testing\"\n\t\"time\"\n)\n\nfunc TestF(t *testing.T) {\n\tfor F() != 1 {\n\t\ttime.Sleep(time.Hour)\n\t}\n}\n",
		`[{"name":"haengt","datei":"p/p.go","paket":"./p","alt":"return 1","neu":"return 2","timeout":"2s"}]`)
	var aus bytes.Buffer
	code := lauf([]string{"-wurzel", q, "-mutanten", filepath.Join(q, "mutanten")}, &aus)
	s := aus.String()
	if code != 0 || !strings.Contains(s, "haengt: HÄNGT") || !strings.Contains(s, "TestF") || !strings.Contains(s, "1 HÄNGT") {
		t.Errorf("Exit %d, erwartet HÄNGT mit TestF und Exit 0:\n%s", code, s)
	}
}
