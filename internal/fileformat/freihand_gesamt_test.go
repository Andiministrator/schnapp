// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"image/color"
	"runtime"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// freihandRiese baut ein SVG mit n Freihand-Strichen zu je MaxFreihandPunkte Punkten "0,0" (F-1049): stark
// komprimierbar, aber knapp unter maxSVG entpackt.
func freihandRiese(t *testing.T, n int) (svg, png []byte) {
	t.Helper()
	d := beispielDok(t, 120, 90)
	fuege(t, d, document.NeuerFreihand([]document.Punkt{document.Pt(5, 40), document.Pt(100, 50)},
		document.Strich{Farbe: color.NRGBA{R: 200, A: 255}, Breite: 3}))
	var buf bytes.Buffer
	if err := SchreibeArbeitsdatei(&buf, d, d.Ansichtsbild(), Metadaten{}); err != nil {
		t.Fatal(err)
	}
	s := string(entpackeSVG(t, buf.Bytes()))
	von := strings.Index(s, `<g s:typ="freihand"`)
	bis := strings.Index(s[von:], `</g>`) + von + len(`</g>`)
	if von < 0 || bis < von {
		t.Fatal("Freihand-Element nicht gefunden")
	}
	el := s[von:bis]
	punkte := strings.TrimSuffix(strings.Repeat("0,0 ", document.MaxFreihandPunkte), " ")
	el = strings.Replace(el, `s:punkte="5,40 100,50"`, `s:punkte="`+punkte+`"`, 1)
	// Darstellung (Pfad) des Elements klein halten, sie zählt nicht zu den Punkten.
	fremd := s[:von] + strings.Repeat(el, n) + s[bis:]
	return []byte(fremd), pngMitSVG(t, buf.Bytes(), fremd)
}

func angefordertMiB(f func()) float64 {
	var m0, m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m0)
	f()
	runtime.ReadMemStats(&m1)
	return float64(m1.TotalAlloc-m0.TotalAlloc) / (1 << 20)
}

// F-1049: Gesamtgrenze der Freihand-Punkte je Datei. Alle Lesewege (Arbeitsdatei, Metadaten, Angaben der
// Rettungskopie, Objekte der Zwischenablage) gehen durch leseSVG und halten dieselbe Grenze.
func TestFreihandGesamtgrenze(t *testing.T) {
	svg, png := freihandRiese(t, 4000)
	t.Logf("SVG %d MiB, %d Punkte", len(svg)>>20, 4000*document.MaxFreihandPunkte)
	const soll = 100.0
	wege := map[string]func() error{
		"LeseArbeitsdatei": func() error { _, err := LeseArbeitsdatei(bytes.NewReader(png)); return err },
		"LeseMetadaten":    func() error { _, err := LeseMetadaten(bytes.NewReader(png)); return err },
		"LeseAngaben":      func() error { _, err := LeseAngaben(bytes.NewReader(svg)); return err },
		"LeseObjekte":      func() error { _, err := LeseObjekte(bytes.NewReader(svg)); return err },
	}
	for name, f := range wege {
		var err error
		mib := angefordertMiB(func() { err = f() })
		t.Logf("%s: %.0f MiB, Fehler: %v", name, mib, err)
		if err == nil {
			t.Errorf("%s: Datei mit %d Freihand-Punkten wurde angenommen", name, 4000*document.MaxFreihandPunkte)
		}
		if mib > soll {
			t.Errorf("%s forderte %.0f MiB an (Soll unter %.0f)", name, mib, soll)
		}
	}
}

// Genau MaxFreihandGesamt Punkte gehen durch, ein Strich mehr nicht; die Grenze gilt für die Summe, nicht je Strich.
func TestFreihandGesamtgrenzeRand(t *testing.T) {
	je := document.MaxFreihandPunkte
	voll := document.MaxFreihandGesamt / je
	svg, _ := freihandRiese(t, voll)
	// Mit Länge ungleich 0, sonst lehnt document den Strich ab: erste Koordinate je Strich ändern ist nicht nötig,
	// wir prüfen hier nur den Fehlergrund.
	_, err := LeseAngaben(bytes.NewReader(svg))
	if err != nil && strings.Contains(err.Error(), "Freihand-Punkte in der Datei") {
		t.Errorf("genau an der Grenze abgelehnt: %v", err)
	}
	svg, _ = freihandRiese(t, voll+1)
	_, err = LeseAngaben(bytes.NewReader(svg))
	if err == nil || !strings.Contains(err.Error(), "Freihand-Punkte in der Datei") {
		t.Errorf("ein Strich über der Grenze: Fehler %v", err)
	}
}

// Der Schreiber erzeugt keine Datei, die der Leser ablehnt (F-1049), und document.PruefeInhalt urteilt gleich.
func TestFreihandGesamtgrenzeSchreiber(t *testing.T) {
	d := beispielDok(t, 120, 90)
	pts := make([]document.Punkt, document.MaxFreihandPunkte)
	for i := range pts {
		pts[i] = document.Pt(float64(i%100), float64(i%80))
	}
	st := document.Strich{Farbe: color.NRGBA{R: 200, A: 255}, Breite: 3}
	var os []document.Objekt
	for range document.MaxFreihandGesamt/document.MaxFreihandPunkte + 1 {
		os = append(os, document.NeuerFreihand(pts, st))
	}
	if _, err := document.PruefeInhalt(d.Groesse(), d.Zuschnitt(), nil, os); err == nil {
		t.Error("PruefeInhalt nimmt zu viele Freihand-Punkte an")
	}
	fuege(t, d, os...)
	var buf bytes.Buffer
	if err := SchreibeArbeitsdatei(&buf, d, d.Ansichtsbild(), Metadaten{}); err == nil {
		t.Error("SchreibeArbeitsdatei schreibt zu viele Freihand-Punkte")
	}
}
