// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"strings"
	"testing"
)

const lauf = `{"Action":"run","Package":"x/a","Test":"TestGut"}
{"Action":"output","Package":"x/a","Test":"TestGut","Output":"=== RUN   TestGut\n"}
{"Action":"pass","Package":"x/a","Test":"TestGut"}
{"Action":"run","Package":"x/a","Test":"TestEcht"}
{"Action":"output","Package":"x/a","Test":"TestEcht","Output":"=== RUN   TestEcht\n"}
{"Action":"output","Package":"x/a","Test":"TestEcht","Output":"    a_test.go:9: tesseract nicht installiert\n"}
{"Action":"output","Package":"x/a","Test":"TestEcht","Output":"--- SKIP: TestEcht (0.00s)\n"}
{"Action":"skip","Package":"x/a","Test":"TestEcht"}
{"Action":"run","Package":"x/a","Test":"TestMessung/teil"}
{"Action":"skip","Package":"x/a","Test":"TestMessung/teil"}
{"Action":"pass","Package":"x/a"}
{"Action":"skip","Package":"x/leer"}
`

func lies(t *testing.T, erlaubt string, ereignisse string) (ergebnis, string, bool) {
	t.Helper()
	m, err := leseErlaubt(strings.NewReader(erlaubt))
	if err != nil {
		t.Fatal(err)
	}
	erg, err := werte(strings.NewReader(ereignisse), m)
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	ok := bericht(&b, erg)
	return erg, b.String(), ok
}

func TestUnerlaubterSprungScheitert(t *testing.T) {
	erg, text, ok := lies(t, "x/a TestMessung # nur von Hand\n", lauf)
	if ok {
		t.Errorf("unerlaubter Sprung gilt als bestanden:\n%s", text)
	}
	if erg.bestanden != 1 || len(erg.sprünge) != 2 {
		t.Errorf("Zählung: %+v", erg)
	}
	if !strings.Contains(text, "SKIP UNERLAUBT x/a TestEcht: a_test.go:9: tesseract nicht installiert") {
		t.Errorf("Grund fehlt:\n%s", text)
	}
	if !strings.Contains(text, "SKIP erlaubt   x/a TestMessung/teil") {
		t.Errorf("erlaubter Untertest nicht als erlaubt gezeigt:\n%s", text)
	}
}

func TestAlleSpruengeErlaubt(t *testing.T) {
	_, text, ok := lies(t, "x/a TestEcht\nx/a TestMessung\n\n# Kommentar\nx/b TestNie # kommt nicht vor\n", lauf)
	if !ok {
		t.Errorf("erlaubte Sprünge gelten als Fehler:\n%s", text)
	}
	if !strings.Contains(text, "Hinweis: erlaubter Sprung kam nicht vor: x/b TestNie") {
		t.Errorf("ungenutzter Eintrag nicht gemeldet:\n%s", text)
	}
}

func TestGescheiterterTestZeigtAusgabe(t *testing.T) {
	ereignisse := `{"Action":"run","Package":"x/a","Test":"TestRot"}
{"Action":"output","Package":"x/a","Test":"TestRot","Output":"    a_test.go:3: falsch\n"}
{"Action":"fail","Package":"x/a","Test":"TestRot"}
{"Action":"output","Package":"x/a","Output":"FAIL\n"}
{"Action":"fail","Package":"x/a"}
`
	erg, text, ok := lies(t, "", ereignisse)
	if ok || erg.gescheitert != 1 || !strings.Contains(text, "a_test.go:3: falsch") {
		t.Errorf("gescheiterter Test: ok=%v %+v\n%s", ok, erg, text)
	}
}

func TestBaufehlerOhneTestScheitert(t *testing.T) {
	ereignisse := `{"ImportPath":"x/a [x/a.test]","Action":"build-output","Output":"a.go:1: syntax error\n"}
{"ImportPath":"x/a [x/a.test]","Action":"build-fail"}
{"Action":"start","Package":"x/a"}
{"Action":"output","Package":"x/a","Output":"FAIL\tx/a [build failed]\n"}
{"Action":"fail","Package":"x/a"}
`
	_, text, ok := lies(t, "", ereignisse)
	if ok || !strings.Contains(text, "syntax error") {
		t.Errorf("Baufehler: ok=%v\n%s", ok, text)
	}
}

func TestOhneEreignisseScheitert(t *testing.T) {
	if _, _, ok := lies(t, "", ""); ok {
		t.Error("leere Eingabe gilt als bestanden")
	}
}

func TestListeMitFalscherZeile(t *testing.T) {
	if _, err := leseErlaubt(strings.NewReader("x/a\n")); err == nil {
		t.Error("Zeile ohne Testname angenommen")
	}
}

// Ein abgeschnittener Lauf (Paket gestartet, kein Ende) scheitert, auch ohne gescheiterten Test (Kritik C-1).
func TestAbgeschnittenerLaufScheitert(t *testing.T) {
	ereignisse := `{"Action":"start","Package":"x/a"}
{"Action":"run","Package":"x/a","Test":"TestGut"}
{"Action":"output","Package":"x/a","Test":"TestGut","Output":"=== RUN   TestGut\n"}
`
	_, text, ok := lies(t, "", ereignisse)
	if ok || !strings.Contains(text, "Paket x/a ohne Ende") {
		t.Errorf("abgeschnittener Lauf: ok=%v\n%s", ok, text)
	}
	vollstaendig := ereignisse + `{"Action":"pass","Package":"x/a","Test":"TestGut"}
{"Action":"pass","Package":"x/a"}
`
	if _, text, ok := lies(t, "", vollstaendig); !ok {
		t.Errorf("vollständiger Lauf gilt als Fehler:\n%s", text)
	}
}
