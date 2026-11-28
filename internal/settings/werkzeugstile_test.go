// SPDX-License-Identifier: GPL-3.0-or-later

package settings

import (
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0041 Punkt 3: Schreiben und Lesen geben dieselben Werte; der Schatten trägt seine Werte auch ausgeschaltet.
func TestWerkzeugStileHinUndZurueck(t *testing.T) {
	rein := []WerkzeugStil{
		{Stil: Stil{Art: ArtPfeil, Farbe: color.NRGBA{R: 1, G: 2, B: 3, A: 255}, Breite: 12,
			Schatten: document.Schatten{An: false, Winkel: 150, Laenge: 7, Weich: 3, Farbe: color.NRGBA{A: 200}}},
			Strichart: document.StrichGestrichelt, KopfEnde: document.KopfPunkt, KopfAnfang: document.KopfRaute},
		{Stil: Stil{Art: ArtKasten, Farbe: color.NRGBA{R: 9, A: 255}, Fuellung: color.NRGBA{G: 9, A: 100}, Breite: 2,
			Schatten: document.Schatten{An: true, Winkel: 90, Laenge: 4, Weich: 0, Farbe: color.NRGBA{A: 255}}},
			Eckradius: 5, Strichart: document.StrichGepunktet},
		{Stil: Stil{Art: ArtText, Farbe: color.NRGBA{A: 255}, Fuellung: color.NRGBA{R: 255, A: 255}, Schriftgrad: 30,
			Schrift: "Go Mono", Fett: true, Durchgestr: true, Ausrichtung: document.AusrichtungRechts}},
		{Stil: Stil{Art: ArtNummer, Farbe: color.NRGBA{B: 200, A: 255}, Radius: 20}},
		{Stil: Stil{Art: ArtLupe, Farbe: color.NRGBA{R: 7, A: 255}, Breite: 3}, Strichart: document.StrichGestrichelt},
		{Stil: Stil{Art: ArtEllipse, Farbe: color.NRGBA{G: 7, A: 255}, Fuellung: color.NRGBA{B: 9, A: 50}, Breite: 5},
			Strichart: document.StrichGepunktet}, // ADR-0047 Punkt 4
		{Stil: Stil{Art: ArtTextmarker, Farbe: color.NRGBA{R: 255, G: 230, B: 40, A: 255}, Breite: 24},
			Strichart: document.StrichGestrichelt, Freihand: true}, // ADR-0048 Punkt 5, ADR-0049 Punkt 5
		{Stil: Stil{Art: ArtFreihand, Farbe: color.NRGBA{B: 90, A: 255}, Breite: 4},
			Strichart: document.StrichGepunktet}, // ADR-0049 Punkt 4
	}
	pfad := filepath.Join(t.TempDir(), "konfig", "tools.ini")
	if err := SchreibeWerkzeugStileDatei(pfad, rein); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(pfad); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("Datei %v, Rechte %v", err, info.Mode().Perm())
	}
	raus, warn := LadeWerkzeugStileDatei(pfad)
	if len(warn) != 0 || len(raus) != len(rein) {
		t.Fatalf("%d Stile, Warnungen %q", len(raus), warn)
	}
	for i := range rein {
		r := raus[i]
		r.Gesetzt = nil
		if r.Art != rein[i].Art || r.Farbe != rein[i].Farbe || r.Breite != rein[i].Breite || r.Schatten != rein[i].Schatten ||
			r.Strichart != rein[i].Strichart || r.KopfEnde != rein[i].KopfEnde ||
			(r.Art == ArtPfeil && r.KopfAnfang != rein[i].KopfAnfang) ||
			r.Eckradius != rein[i].Eckradius || r.Schriftgrad != rein[i].Schriftgrad || r.Schrift != rein[i].Schrift ||
			r.Fett != rein[i].Fett || r.Durchgestr != rein[i].Durchgestr || r.Ausrichtung != rein[i].Ausrichtung ||
			r.Radius != rein[i].Radius || r.Fuellung != rein[i].Fuellung || r.Freihand != rein[i].Freihand {
			t.Errorf("[%s] gelesen %+v, geschrieben %+v", rein[i].Art, r, rein[i])
		}
		if r.Art != ArtPfeil && r.KopfAnfang != document.KopfKeine { // ohne Spitzen: keine am Anfang
			t.Errorf("[%s] Spitze am Anfang %v", r.Art, r.KopfAnfang)
		}
		if len(raus[i].Gesetzt) != len(werkzeugSchluessel(rein[i].Art)) {
			t.Errorf("[%s] gesetzt %v", rein[i].Art, raus[i].Gesetzt)
		}
	}
}

// ADR-0041 Punkt 3: Ein ungültiger oder unbekannter Schlüssel fällt allein weg, der Rest gilt; ein zweiter Abschnitt
// desselben Werkzeugs und ein unbekannter Abschnitt fallen mit Warnung weg; shadow_on gilt in jeder Reihenfolge.
func TestWerkzeugStileNachsichtig(t *testing.T) {
	inhalt := strings.Join([]string{
		"[arrow]", "color = #112233", "width = viel", "farbe = rot", "line_style = dashed", "line_style = dotted",
		"shadow = 135 6 8 #000000FF", "shadow_on = yes",
		"[arrow]", "color = #FFFFFF",
		"[callout]", "color = #FFFFFF",
		"[box]", "corner_radius = -1", "end_head = dot",
	}, "\n")
	st, warn := ParseWerkzeugStile("tools.ini", inhalt)
	if len(st) != 2 {
		t.Fatalf("%d Abschnitte, Soll 2", len(st))
	}
	p := st[0]
	if p.Farbe != (color.NRGBA{0x11, 0x22, 0x33, 0xFF}) || p.Strichart != document.StrichGestrichelt || !p.Schatten.An ||
		p.Schatten.Laenge != 6 || p.Gesetzt["width"] || !p.Gesetzt["color"] || !p.Gesetzt["shadow_on"] {
		t.Errorf("Pfeil %+v", p)
	}
	if st[1].Gesetzt["corner_radius"] || st[1].Gesetzt["end_head"] || len(st[1].Gesetzt) != 0 {
		t.Errorf("Kasten gesetzt %v (end_head gehört nicht zum Kasten)", st[1].Gesetzt)
	}
	// width, farbe, zweite line_style, zweites [arrow], [callout], corner_radius, end_head: sieben Warnungen.
	if len(warn) != 7 {
		t.Errorf("%d Warnungen, Soll 7: %q", len(warn), warn)
	}
}

// Schreiben auf ein Verzeichnis statt einer Datei scheitert mit ErrWerkzeugeKeineDatei; Lesen eines Verzeichnisses
// warnt und liefert nichts.
func TestWerkzeugStileKeineDatei(t *testing.T) {
	pfad := filepath.Join(t.TempDir(), "tools.ini")
	if err := os.Mkdir(pfad, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := SchreibeWerkzeugStileDatei(pfad, nil); err == nil || !strings.Contains(err.Error(), ErrWerkzeugeKeineDatei.Error()) {
		t.Errorf("Schreiben auf ein Verzeichnis: %v", err)
	}
	if st, warn := LadeWerkzeugStileDatei(pfad); st != nil || len(warn) != 1 {
		t.Errorf("Lesen eines Verzeichnisses: %v, %q", st, warn)
	}
	if st, warn := LadeWerkzeugStileDatei(filepath.Join(t.TempDir(), "fehlt.ini")); st != nil || warn != nil {
		t.Errorf("fehlende Datei: %v, %q", st, warn)
	}
}
