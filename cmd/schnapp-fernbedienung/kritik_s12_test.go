// SPDX-License-Identifier: GPL-3.0-or-later

//go:build probe

package main

// Regressionstests zu F-614 (Sammelkritik Sprint 12, KS-5, KS-8, KS-11; Karten #226, #237): Befehle, die selbst drücken
// und lösen, verweigern sich bei gehaltener Taste, statt die Buchführung zu verfälschen; abspielen löst am Ende alles.

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// KS-8: klick/ziehe mit einer Maustaste, die druecke hält, senden Drücken und Lösen derselben Taste. Der gehaltene Zug
// endet im Compositor still, der Abspieler führt die Taste weiter als gehalten und löst sie später ein zweites Mal.
func TestKritikKlickWaehrendGehaltenerMaustaste(t *testing.T) {
	ev, a, err := spiele(t, "druecke 10 10\nklick 20 20\n")
	t.Logf("Ereignisse %q, Fehler %v, gehalten %v", ev, err, a.knoepfe)
	if err == nil {
		t.Errorf("klick bei gehaltener linker Taste angenommen; gesendet %q, der Abspieler führt %v weiter als gehalten",
			strings.Join(ev, ","), a.knoepfe)
	}
}

func TestKritikZieheWaehrendGehaltenerMaustaste(t *testing.T) {
	ev, a, err := spiele(t, "druecke 10 10\nziehe 1 1 5 5 1\nloslassen\n")
	t.Logf("Ereignisse %q, Fehler %v, gehalten %v", ev, err, a.knoepfe)
	if err == nil {
		t.Errorf("ziehe bei gehaltener linker Taste angenommen; gesendet %q (zweimal Drücken, zweimal Lösen)", strings.Join(ev, ","))
	}
}

// KS-8: taste KOMBI mit einem Modifikator, den taste-halten hält, löst ihn am Ende der Kombination; der Abspieler führt
// ihn weiter als gehalten.
func TestKritikTasteLoestGehaltenenModifikator(t *testing.T) {
	ev, a, err := spiele(t, "taste-halten Alt\ntaste Alt+B\n")
	t.Logf("Ereignisse %q, Fehler %v, gehalten %v", ev, err, a.tasten)
	if err == nil && len(a.tasten) == 1 && ev[len(ev)-1] == "taste 56 0" {
		t.Errorf("taste Alt+B löste das gehaltene Alt (%q), der Abspieler führt es weiter als gehalten (%v)",
			strings.Join(ev, ","), a.tasten)
	}
}

// fehlEingabe scheitert beim n-ten Tastaturereignis.
type fehlEingabe struct {
	aufzeichner
	n int
}

func (f *fehlEingabe) Tastatur(code, zustand uint32) error {
	f.n--
	if f.n == 0 {
		return errors.New("Verbindung weg")
	}
	return f.aufzeichner.Tastatur(code, zustand)
}

// Prüfung: Scheitert taste-halten mitten in der Kombination, löst alleLoslassen, was schon gedrückt ist.
func TestKritikFehlerMittenInTasteHalten(t *testing.T) {
	befehle, err := ParseDrehbuch("taste-halten Strg+Umschalt+B\n")
	if err != nil {
		t.Fatal(err)
	}
	r := &fehlEingabe{n: 3}
	a := &abspieler{e: r}
	if err := a.fuehreAus(befehle[0]); err == nil {
		t.Fatal("Kontrolle: kein Fehler")
	}
	r.ereignisse = nil
	if err := a.alleLoslassen(); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(r.ereignisse, ","); got != "taste 42 0,taste 29 0" {
		t.Errorf("nach dem Fehler gelöst: %q, Soll Umschalt und Strg", got)
	}
}

// Prüfung: Scheitert das Lösen in taste-loslassen, bleibt die Taste als gehalten geführt und wird am Ende erneut gelöst.
func TestKritikFehlerBeimLoesen(t *testing.T) {
	befehle, err := ParseDrehbuch("taste-halten Alt\ntaste-loslassen Alt\n")
	if err != nil {
		t.Fatal(err)
	}
	r := &fehlEingabe{n: 2}
	a := &abspieler{e: r}
	if err := a.fuehreAus(befehle[0]); err != nil {
		t.Fatal(err)
	}
	if err := a.fuehreAus(befehle[1]); err == nil {
		t.Fatal("Kontrolle: kein Fehler")
	}
	r.ereignisse = nil
	if err := a.alleLoslassen(); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(r.ereignisse, ","); got != "taste 56 0" {
		t.Errorf("am Ende gelöst: %q, Soll Alt", got)
	}
}

// KS-5: klick, ziehe und taste lassen die Buchführung heil, wenn sie ablehnen (nichts gesendet, was die Taste löst).
func TestAblehnenSendetNichts(t *testing.T) {
	for _, f := range []struct{ buch, soll string }{
		{"druecke 10 10\nklick 20 20\n", "zeiger 10 10,maus 0x110 1"},
		{"druecke 10 10\nziehe 1 1 5 5 1\n", "zeiger 10 10,maus 0x110 1"},
		{"taste-halten Alt\ntaste Alt+B\n", "taste 56 1"},
		{"taste-halten B\ntaste B\n", "taste 48 1"},
	} {
		ev, _, err := spiele(t, f.buch)
		if err == nil || strings.Join(ev, ",") != f.soll {
			t.Errorf("%q: Ereignisse %q, Fehler %v, erwartet Ablehnung nach %q", f.buch, strings.Join(ev, ","), err, f.soll)
		}
	}
}

// Andere Maustasten und andere Tasten bleiben benutzbar, solange etwas gehalten wird.
func TestAndereTasteBleibtBenutzbar(t *testing.T) {
	ev, a, err := spiele(t, "druecke 10 10\nrechtsklick 20 20\ntaste-halten Alt\ntaste B\n")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a.knoepfe, []uint32{btnLinks}) || !reflect.DeepEqual(a.tasten, []uint32{56}) {
		t.Errorf("Buchführung: Maus %v, Tasten %v; Ereignisse %q", a.knoepfe, a.tasten, ev)
	}
}

// KS-11 / KM25: taste-loslassen nimmt die Taste aus der Buchführung.
func TestTasteLoslassenFuehrtBuch(t *testing.T) {
	_, a, err := spiele(t, "taste-halten Strg+Alt\ntaste-loslassen Strg+Alt\n")
	if err != nil || len(a.tasten) != 0 {
		t.Errorf("gehalten %v, Fehler %v", a.tasten, err)
	}
	// Wieder halten und lösen klappt, nur wenn nichts hängen blieb.
	if _, _, err := spiele(t, "taste-halten Alt\ntaste-loslassen Alt\ntaste-halten Alt\ntaste-loslassen Alt\n"); err != nil {
		t.Errorf("erneutes Halten: %v", err)
	}
}

// KS-11 / KM27: loslassen löst alle gehaltenen Maustasten.
func TestLoslassenLoestAlleMaustasten(t *testing.T) {
	ev, a, err := spiele(t, "druecke 1 1\ndruecke-rechts\nloslassen\n")
	if err != nil || len(a.knoepfe) != 0 {
		t.Fatalf("gehalten %v, Fehler %v", a.knoepfe, err)
	}
	if got, want := strings.Join(ev, ","), "zeiger 1 1,maus 0x110 1,maus 0x111 1,maus 0x111 0,maus 0x110 0"; got != want {
		t.Errorf("Ereignisse %q, erwartet %q", got, want)
	}
	// Ein zweites loslassen ist ein Fehler: nichts mehr gehalten.
	if _, _, err := spiele(t, "druecke\nloslassen\nloslassen\n"); err == nil {
		t.Error("zweites loslassen angenommen")
	}
}

// KS-11 / KM26: klick merkt die Zeigerlage; bewege danach geht von dort in Schritten.
func TestKlickMerktZeigerlage(t *testing.T) {
	ev, _, err := spiele(t, "klick 20 20\nbewege 40 20 2\n")
	if err != nil {
		t.Fatal(err)
	}
	if n := len(ev); n < 2 || ev[n-2] != "zeiger 30 20" || ev[n-1] != "zeiger 40 20" {
		t.Errorf("Ereignisse %q, erwartet Schritte von 20 20 nach 40 20", ev)
	}
}

func spieleAb(t *testing.T, text string) (int, []string, string) {
	t.Helper()
	befehle, err := ParseDrehbuch(text)
	if err != nil {
		t.Fatal(err)
	}
	r := &aufzeichner{}
	var aus, fehler bytes.Buffer
	code := abspielen(&abspieler{e: r}, befehle, 0, nil, &aus, &fehler)
	return code, r.ereignisse, fehler.String()
}

// KS-11 / KM24: Am Ende des Drehbuchs wird alles Gehaltene gelöst, mit Exit 0 bei Erfolg.
func TestAbspielenLoestAmEnde(t *testing.T) {
	code, ev, fe := spieleAb(t, "druecke 1 2\ntaste-halten Alt\n")
	if code != 0 || fe != "" {
		t.Errorf("Exit %d, Fehlerausgabe %q", code, fe)
	}
	if got, want := strings.Join(ev, ","), "zeiger 1 2,maus 0x110 1,taste 56 1,maus 0x110 0,taste 56 0"; got != want {
		t.Errorf("Ereignisse %q, erwartet %q", got, want)
	}
}

// Auch nach einem Fehler wird gelöst, und der Exit ist nicht 0; das gilt für einen Verstoß gegen das Halten.
func TestAbspielenLoestNachFehler(t *testing.T) {
	for _, text := range []string{"taste-halten Alt\nloslassen\n", "druecke 1 2\nklick 3 4\n", "taste-halten Alt\ntaste Alt+B\n"} {
		code, ev, fe := spieleAb(t, text)
		if code != 1 || fe == "" {
			t.Errorf("%q: Exit %d, Fehlerausgabe %q, erwartet Exit 1", text, code, fe)
		}
		if len(ev) < 2 || !strings.HasSuffix(ev[len(ev)-1], " 0") {
			t.Errorf("%q: am Ende nicht gelöst: %q", text, ev)
		}
	}
}

// Scheitert der Abgleich mit dem Compositor, endet das Abspielen mit Exit 1 und löst trotzdem.
func TestAbspielenAbgleichFehler(t *testing.T) {
	befehle, _ := ParseDrehbuch("druecke 1 2\nwarte 0\n")
	r := &aufzeichner{}
	var aus, fehler bytes.Buffer
	code := abspielen(&abspieler{e: r}, befehle, 0, func() error { return errors.New("weg") }, &aus, &fehler)
	if code != 1 || !strings.Contains(fehler.String(), "Compositor: weg") {
		t.Errorf("Exit %d, Fehlerausgabe %q", code, fehler.String())
	}
	if last := r.ereignisse[len(r.ereignisse)-1]; last != "maus 0x110 0" {
		t.Errorf("nicht gelöst: %q", r.ereignisse)
	}
}
