// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Karte #311 (ADR-0036 Nachtrag 1): Der Knopf „Library“ ist ein schmaler Symbolknopf, so hoch wie eine Vorschau, und
// unter jeder Vorschau stehen Zeit und Auflösung.

import (
	"image"
	"testing"
	"time"

	"gioui.org/io/pointer"

	"github.com/Andiministrator/schnapp/internal/library"
)

func TestLeisteKnopfSchmalVorschauenRuecken(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	k := &l.e.unten
	if len(k.kacheln) == 0 {
		t.Fatal("keine Vorschau gezeichnet")
	}
	erste := k.kacheln[0]
	knopfRechts := k.rahmen.Min.X + leisteKnopfRand + leisteKnopfB
	if erste.Min.X != knopfRechts+leisteRandDp {
		t.Errorf("erste Vorschau beginnt bei %d, Soll %d (Knopf %d dp breit)", erste.Min.X, knopfRechts+leisteRandDp, leisteKnopfB)
	}
	if leisteKnopfB > 48 {
		t.Errorf("Knopf %d dp breit — nicht mehr schmal", leisteKnopfB)
	}
	// Zwei Zeilen Text (je 14 dp ab 4 dp unter der Vorschau) passen in die Leiste.
	if unten := erste.Max.Y + 18 + 14; unten > k.rahmen.Max.Y {
		t.Errorf("zweite Zeile endet bei %d, Leiste bei %d", unten, k.rahmen.Max.Y)
	}
	// Der Knopf öffnet die Bibliothek und meldet sich in der Statuszeile.
	l.maus(pointer.Move, image.Pt(k.rahmen.Min.X+leisteKnopfRand+leisteKnopfB/2, erste.Min.Y+erste.Dy()/2), 0)
	l.setle()
	if k.hinweis == "" {
		t.Error("kein Hinweis über dem Knopf")
	}
}

func TestLeisteAufloesung(t *testing.T) {
	e := library.Eintrag{Pfad: "2026-11/a.png", Erstellt: time.Now(), Breite: 1920, Hoehe: 1080}
	if got := leisteAufloesung(e); got != "1920 × 1080" {
		t.Errorf("Auflösung %q", got)
	}
	if got := leisteAufloesung(library.Eintrag{}); got != "" {
		t.Errorf("ohne Maße %q", got)
	}
}
