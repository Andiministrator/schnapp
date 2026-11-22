// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// ADR-0023 Nachtrag 6: [capture] seam gelangt über plane in den Auftrag, die Voreinstellung ist tear.
func TestPlaneKante(t *testing.T) {
	e := settings.Voreinstellungen()
	if p := plane(Aufruf{Modus: ModusBereich}, e); p.Kante != document.KanteAbriss {
		t.Errorf("Voreinstellung: %v", p.Kante)
	}
	e.Kante = document.KanteWelle
	for _, m := range []Modus{ModusBereich, ModusOeffnen, ModusFenster} {
		if p := plane(Aufruf{Modus: m}, e); p.Kante != document.KanteWelle {
			t.Errorf("Modus %v: %v, Soll wave", m, p.Kante)
		}
	}
}

// Eine Aufnahme legt ihr Dokument mit der eingestellten Form ab, ohne Rückgängig-Schritt (keine Änderung des Nutzers).
func TestAufnahmeMitKante(t *testing.T) {
	i18n.Setze(nil)
	for _, k := range []document.Kante{document.KanteAbriss, document.KanteWelle, document.KanteZacken} {
		l := neuerLauf(t)
		if code, _, fehler := l.fuehre(Auftrag{Modus: ModusFenster, Kante: k}); code != exitOK || fehler != "" {
			t.Fatalf("%v: Code %d, %q", k, code, fehler)
		}
		if len(l.abgelegt) != 1 || l.abgelegt[0].Kante() != k || l.abgelegt[0].KannRueckgaengig() {
			t.Errorf("%v: abgelegt %v", k, l.abgelegt)
		}
	}
}

// Öffnen: Ein einfaches PNG bekommt die eingestellte Form, eine Arbeitsdatei behält ihre eigene; beides ohne Schritt.
func TestOeffnenMitKante(t *testing.T) {
	_, _, g := rettungsLauf(t)
	dir := t.TempDir()
	einfach := filepath.Join(dir, "einfach.png")
	f, err := os.Create(einfach)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewNRGBA(image.Rect(0, 0, 30, 20))); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := oeffneImEditor(einfach, false, document.KanteWelle); err != nil {
		t.Fatal(err)
	}
	if g.dok.Kante() != document.KanteWelle || g.dok.KannRueckgaengig() {
		t.Errorf("einfaches PNG: %v, Rückgängig %v, Soll wave ohne Schritt", g.dok.Kante(), g.dok.KannRueckgaengig())
	}

	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 30, 20)))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.VorgabeKante(document.KanteZacken); err != nil {
		t.Fatal(err)
	}
	arbeit := schreibeDatei(t, dir, "arbeit.png", d)
	if err := oeffneImEditor(arbeit, false, document.KanteWelle); err != nil {
		t.Fatal(err)
	}
	if g.dok.Kante() != document.KanteZacken || g.dok.KannRueckgaengig() {
		t.Errorf("Arbeitsdatei: %v, Soll ihre eigene Form zigzag", g.dok.Kante())
	}
}
