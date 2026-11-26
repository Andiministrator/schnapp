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
	// Eine Zeile Text (14 dp ab 4 dp unter der Vorschau) passt in die Leiste, eine zweite nicht mehr (Karte #353).
	if unten := erste.Max.Y + 4 + 14; unten > k.rahmen.Max.Y {
		t.Errorf("Zeile endet bei %d, Leiste bei %d", unten, k.rahmen.Max.Y)
	}
	if unten := erste.Max.Y + 18 + 14; unten <= k.rahmen.Max.Y {
		t.Errorf("Leiste %d dp hoch — Platz für eine zweite Zeile, Soll niedriger", k.rahmen.Dy())
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
	if got := leisteAufloesungKurz(e); got != "1920×1080" {
		t.Errorf("kurze Auflösung %q", got)
	}
	if got := leisteAufloesungKurz(library.Eintrag{}); got != "" {
		t.Errorf("kurz ohne Maße %q", got)
	}
}

// Karte #353: Zeit und Auflösung stehen in einer Zeile; passt die lange Form nicht, die kurze, sonst nur die Zeit.
func TestLeisteZeileWahl(t *testing.T) {
	for _, f := range []struct {
		name                             string
		breite, zeit, voll, kurz, luecke int
		soll                             int
	}{
		{"alles passt", 120, 30, 60, 50, 6, zeileVoll},
		{"genau passend", 96, 30, 60, 50, 6, zeileVoll},
		{"ein Pixel zu breit", 95, 30, 60, 50, 6, zeileKurz},
		{"kurz genau passend", 86, 30, 60, 50, 6, zeileKurz},
		{"auch kurz zu breit", 85, 30, 60, 50, 6, zeileNurZeit},
		{"ohne Auflösung", 120, 30, 0, 0, 6, zeileNurZeit},
	} {
		if got := leisteZeileWahl(f.breite, f.zeit, f.voll, f.kurz, f.luecke); got != f.soll {
			t.Errorf("%s: %d, Soll %d", f.name, got, f.soll)
		}
	}
}

// Karte #353: Unter jeder Vorschau steht die Auflösung rechtsbündig in derselben Zeile wie die Zeit — bei heutiger
// Uhrzeit und 1920 × 1080 passt die lange Form in die Kachel.
func TestLeisteZeileHeutePasstVoll(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 1)
	gtx := l.gtx()
	zeit := l.e.leisteTextBreite(gtx, "23:59")
	voll := l.e.leisteTextBreite(gtx, "1920 × 1080")
	kurz := l.e.leisteTextBreite(gtx, "1920×1080")
	if zeit <= 0 || voll <= kurz || kurz <= 0 {
		t.Fatalf("Messung unplausibel: Zeit %d, voll %d, kurz %d", zeit, voll, kurz)
	}
	if w := leisteZeileWahl(gtx.Dp(leisteKachelB), zeit, voll, kurz, gtx.Dp(6)); w != zeileVoll {
		t.Errorf("heute mit 1920 × 1080: Wahl %d, Soll voll (Zeit %d, voll %d, Kachel %d)", w, zeit, voll, gtx.Dp(leisteKachelB))
	}
	// Gezeichnet: heute die lange Form; in einer schmalen Kachel die kurze; in einer winzigen keine.
	jetzt := time.Date(2026, 11, 26, 12, 0, 0, 0, time.Local)
	ein := library.Eintrag{Pfad: "2026-11/a.png", Erstellt: jetzt.Add(-time.Hour), Breite: 1920, Hoehe: 1080}
	kr := image.Rect(0, 0, gtx.Dp(leisteKachelB), gtx.Dp(leisteKachelH))
	if got := l.e.leisteZeile(gtx, ein, jetzt, kr, false); got != "1920 × 1080" {
		t.Errorf("Kachel %d px: Auflösung %q, Soll lang", kr.Dx(), got)
	}
	zeitB := l.e.leisteTextBreite(gtx, "11:00")
	schmal := image.Rect(0, 0, zeitB+gtx.Dp(6)+kurz, kr.Dy())
	if got := l.e.leisteZeile(gtx, ein, jetzt, schmal, false); got != "1920×1080" {
		t.Errorf("Kachel %d px: Auflösung %q, Soll kurz", schmal.Dx(), got)
	}
	winzig := image.Rect(0, 0, zeitB+gtx.Dp(6), kr.Dy())
	if got := l.e.leisteZeile(gtx, ein, jetzt, winzig, false); got != "" {
		t.Errorf("Kachel %d px: Auflösung %q, Soll keine", winzig.Dx(), got)
	}
}
