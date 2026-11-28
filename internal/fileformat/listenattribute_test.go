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

// F-1035 (Kritik #333 K-1): s:punkte aus lauter Leerzeichen darf nicht je Leerzeichen ein Feld anlegen.
// strings.Split vor der Zählung forderte für 60 Mio. Leerzeichen über 1 GiB an; jetzt wird erst gezählt.
// Gleiches Muster: s:zuschnitt, s:stil, s:schatten.
func TestListenAttributeOhneRiesenFeld(t *testing.T) {
	d := beispielDok(t, 120, 90)
	fuege(t, d, document.NeuerFreihand([]document.Punkt{document.Pt(5, 40), document.Pt(100, 50)},
		document.Strich{Farbe: color.NRGBA{R: 200, A: 255}, Breite: 3}))
	var buf bytes.Buffer
	if err := SchreibeArbeitsdatei(&buf, d, d.Ansichtsbild(), Metadaten{}); err != nil {
		t.Fatal(err)
	}
	svg := string(entpackeSVG(t, buf.Bytes()))
	leer := strings.Repeat(" ", 8<<20)
	fremd := strings.Replace(svg, `s:punkte="5,40 95,10"`, `s:punkte="`+leer+`"`, 1)
	if fremd == svg {
		t.Fatal("Ersetzung ohne Wirkung")
	}
	datei := pngMitSVG(t, buf.Bytes(), fremd)
	var m0, m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m0)
	_, err := LeseArbeitsdatei(bytes.NewReader(datei))
	runtime.ReadMemStats(&m1)
	if err == nil {
		t.Error("Punkte aus Leerzeichen müssen die Datei beschädigen")
	}
	// Vorher: 16 Byte je Leerzeichen = 128 MiB zusätzlich zum Text; Sollwert deutlich darunter.
	if mib := float64(m1.TotalAlloc-m0.TotalAlloc) / (1 << 20); mib > 64 {
		t.Errorf("Lesen forderte %.0f MiB an (Soll unter 64)", mib)
	}
}

// Zuschnitt, Stil und Schatten mit überlangem Wert: kein Absturz, Zuschnitt beschädigt, Rest wird ignoriert.
func TestListenAttributeUeberlang(t *testing.T) {
	if s := leseSchatten(strings.Repeat(" ", 1<<20)); s.An {
		t.Error("überlanger Schatten gilt als Schatten")
	}
	var tx document.Text
	leseTextStil(&tx, map[string]string{"stil": "bold" + strings.Repeat(" ", 1<<20) + "italic"})
	if !tx.Fett || tx.Kursiv {
		t.Errorf("überlanger Stil: Fett %v Kursiv %v, erwartet true/false", tx.Fett, tx.Kursiv)
	}
}
