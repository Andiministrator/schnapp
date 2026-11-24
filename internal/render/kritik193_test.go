// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image"
	"image/color"
	"math/rand"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Kritik Karte #193 — Erkundung.

// K193-Spalt: Der Spalt einer Stauch-Naht radiert behaltene Pixel der Ansicht (Alpha 0). Zeigt eine Lupe, deren Quelle
// über der Naht liegt, diese Pixel trotzdem?
func TestK193LupeZeigtPixelUnterSpalt(t *testing.T) {
	d := musterDokument(t, 200, 120)
	if err := d.SetzeStreifen([]document.Streifen{{Von: 80, Bis: 120}}); err != nil {
		t.Fatal(err)
	}
	// Naht in der Ansicht bei x = 80. Ohne Lupe rendern und die radierten Spalten suchen.
	ohne, err := Bild(d)
	if err != nil {
		t.Fatal(err)
	}
	// Quelle: Original 120..140 x 90..110 (Ansicht 80..100, rechts an der Naht), Ziel 2x oben links (weit weg).
	hinzu(t, d, lupeOhneStrich(document.R(document.Pt(120, 90), document.Pt(140, 110)), document.R(document.Pt(4, 4), document.Pt(44, 44))))
	mit, err := Bild(d)
	if err != nil {
		t.Fatal(err)
	}
	orig := d.OriginalBild()
	radiert, leck := 0, 0
	for y := 90; y < 110; y++ {
		for vx := 80; vx < 100; vx++ {
			if ohne.NRGBAAt(vx, y).A != 0 || mit.NRGBAAt(vx, y).A != 0 {
				continue
			}
			radiert++
			ox := vx + 40
			zx, zy := 4+(ox-120)*2, 4+(y-90)*2
			if mit.NRGBAAt(zx, zy) == orig.NRGBAAt(ox, y) {
				leck++
			}
		}
	}
	t.Logf("im Quellbereich radiert (Spalt, Alpha 0): %d Pixel, davon im Ziel der Lupe im Klartext: %d", radiert, leck)
	if leck > 0 {
		t.Errorf("%d Quellpixel sind in der Ansicht radiert (Spalt), stehen aber im Ziel der Lupe", leck)
	}
}

// K193-Budget: Viele Lupen mit winzigem Ziel und der ganzen Ansicht als Quelle kosten im Budget fast nichts.
func TestK193BudgetWinzigesZiel(t *testing.T) {
	b := image.NewNRGBA(image.Rect(0, 0, 1920, 1080))
	rnd := rand.New(rand.NewSource(1))
	rnd.Read(b.Pix)
	for i := 3; i < len(b.Pix); i += 4 {
		b.Pix[i] = 255
	}
	d, err := document.Neu(b)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 1000; i++ {
		hinzu(t, d, document.Lupe{Quelle: document.R(document.Pt(0, 0), document.Pt(1920, 1080)),
			Ziel: document.R(document.Pt(float64(i), 0), document.Pt(float64(i)+1, 1)), Strich: document.Strich{Farbe: color.NRGBA{A: 255}}})
	}
	a := Aufwand(d)
	t.Logf("Aufwand %v von Budget %v; Pruefe: %v", a, grenzen.gesamt, Pruefe(d))
	// ADR-0033 Nachtrag 2 Punkt 3: je Lupe zählt die aufs Original begrenzte Quellfläche (der SVG-Export bettet sie ein).
	if soll := 1000 * fuellGewicht * 1920 * 1080; a < soll {
		t.Errorf("Aufwand %v, Soll mindestens %v (Quellfläche je Lupe)", a, soll)
	}
}

// K193-1 am Rand: Quelle größer als das Bild zählt nur die Fläche im Original; knapp unter dem Budget passt, eine Lupe
// mehr nicht.
func TestK193BudgetQuelleGrenze(t *testing.T) {
	const w, h = 2000, 1000
	d := weissesDokument(t, w, h)
	lupe := document.Lupe{Quelle: document.R(document.Pt(-300, -200), document.Pt(w+500, h+700)),
		Ziel: document.R(document.Pt(0, 0), document.Pt(1, 1)), Strich: document.Strich{Farbe: color.NRGBA{A: 255}}}
	hinzu(t, d, lupe)
	je := fuellGewicht*w*h + fuellGewicht // Quelle im Original und ein Zielpixel
	if a := Aufwand(d); a != je {
		t.Fatalf("eine Lupe: Aufwand %v, Soll %v", a, je)
	}
	n := int(grenzen.gesamt / je) // so viele passen
	for i := 1; i < n; i++ {
		hinzu(t, d, lupe)
	}
	if err := Pruefe(d); err != nil {
		t.Fatalf("%d Lupen (Aufwand %v) abgelehnt: %v", n, Aufwand(d), err)
	}
	hinzu(t, d, lupe)
	if err := Pruefe(d); err == nil {
		t.Errorf("%d Lupen (Aufwand %v) über dem Budget angenommen", n+1, Aufwand(d))
	}
}

// K193-Rueck: Schwärzen, Rückgängig, Wiederholen — LupenInhalt folgt jedes Mal dem Original.
func TestK193RueckgaengigSchwaerzen(t *testing.T) {
	d := musterDokument(t, 40, 30)
	l := lupeOhneStrich(document.R(document.Pt(2, 2), document.Pt(6, 6)), document.R(document.Pt(20, 10), document.Pt(28, 18)))
	id, err := d.Hinzufuegen(l)
	if err != nil {
		t.Fatal(err)
	}
	l.ID = id
	klar := LupenInhalt(d, l).NRGBAAt(21, 11)
	if err := d.Schwaerzen(image.Rect(2, 2, 6, 6), document.Schwaerzung{Farbe: color.NRGBA{A: 255}}); err != nil {
		t.Fatal(err)
	}
	if got := LupenInhalt(d, l).NRGBAAt(21, 11); got != (color.NRGBA{A: 255}) {
		t.Errorf("nach Schwärzen %v", got)
	}
	if err := d.Rueckgaengig(); err != nil {
		t.Fatal(err)
	}
	if got := LupenInhalt(d, l).NRGBAAt(21, 11); got != klar {
		t.Errorf("nach Rückgängig %v, Soll %v", got, klar)
	}
	if err := d.Wiederholen(); err != nil {
		t.Fatal(err)
	}
	if got := LupenInhalt(d, l).NRGBAAt(21, 11); got != (color.NRGBA{A: 255}) {
		t.Errorf("nach Wiederholen %v", got)
	}
}

// K193: Dauer von LupenInhalt für ein großes Ziel (der Editor rechnet es in jedem Bild eines Zugs neu, weil der
// Merker die Lupe selbst im Schlüssel trägt).
func TestK193DauerGrossesZiel(t *testing.T) {
	d := musterDokument(t, 3840, 2160)
	l := lupeOhneStrich(document.R(document.Pt(0, 0), document.Pt(1200, 700)), document.R(document.Pt(100, 50), document.Pt(3700, 2150)))
	t0 := time.Now()
	b := LupenInhalt(d, l)
	t.Logf("LupenInhalt %v: %v", b.Rect, time.Since(t0))
	t0 = time.Now()
	hinzu(t, d, l)
	if _, err := Bild(d); err != nil {
		t.Fatal(err)
	}
	t.Logf("Bild mit dieser Lupe: %v", time.Since(t0))
}

// K193: Gruppendeckkraft — die Lupe mit 99 % deckt dieselben Pixel wie mit 100 % (Rahmen des Zwischenbilds), auch mit
// dickem Strich und an einer Naht.
func TestK193DeckkraftUmriss(t *testing.T) {
	for _, naht := range []bool{false, true} {
		spur := func(p int) map[image.Point]bool {
			d := weissesDokument(t, 200, 100)
			if naht {
				if err := d.SetzeStreifen([]document.Streifen{{Von: 60, Bis: 100}}); err != nil {
					t.Fatal(err)
				}
			}
			grund, err := Bild(d)
			if err != nil {
				t.Fatal(err)
			}
			l := document.Lupe{Quelle: document.R(document.Pt(10, 10), document.Pt(60, 40)), Ziel: document.R(document.Pt(100, 20), document.Pt(190, 74)),
				Strich: document.Strich{Farbe: color.NRGBA{R: 255, A: 255}, Breite: 12}}
			l.Ebene = l.Ebene.MitDeckkraft(p)
			hinzu(t, d, l)
			b, err := Bild(d)
			if err != nil {
				t.Fatal(err)
			}
			m := map[image.Point]bool{}
			for y := 0; y < b.Rect.Dy(); y++ {
				for x := 0; x < b.Rect.Dx(); x++ {
					if b.NRGBAAt(x, y) != grund.NRGBAAt(x, y) {
						m[image.Pt(x, y)] = true
					}
				}
			}
			return m
		}
		voll, halb := spur(100), spur(99)
		fehlt := 0
		for p := range voll {
			if !halb[p] {
				fehlt++
			}
		}
		t.Logf("Naht %v: %d Pixel bei 100 %%, %d bei 99 %%, bei 99 %% fehlen %d", naht, len(voll), len(halb), fehlt)
		if fehlt > 0 {
			t.Errorf("Naht %v: %d Pixel fehlen mit Gruppendeckkraft", naht, fehlt)
		}
	}
}

// K193-5 (Mutant K19): Liegt die rechte Kante des Ziels knapp vor einem Streifen, ragt der starre Rahmen (halbe
// Strichbreite) in der Ansicht über die abgebildeten Grenzen hinaus, weiter als der Spalt der Naht reicht. Das
// Zwischenbild der Gruppendeckkraft muss ihn mit dem Rand ebenenRahmen ganz fassen: Bei 99 % deckt die Lupe dieselben
// Pixel wie bei 100 %.
func TestK193DeckkraftRandVorStreifen(t *testing.T) {
	spur := func(p int) map[image.Point]bool {
		d := weissesDokument(t, 300, 120)
		if err := d.SetzeStreifen([]document.Streifen{{Von: 161, Bis: 200}}); err != nil {
			t.Fatal(err)
		}
		grund, err := Bild(d)
		if err != nil {
			t.Fatal(err)
		}
		// Ziel bis x = 160, Strich 40: Grenzen bis 180 (im Streifen, abgebildet auf die Naht bei 161), der Rahmen reicht in
		// der Ansicht bis 180; der Spalt endet spätestens bei 161 + 8 + 6.
		l := document.Lupe{Quelle: document.R(document.Pt(10, 40), document.Pt(40, 70)), Ziel: document.R(document.Pt(100, 30), document.Pt(160, 90)),
			Strich: document.Strich{Farbe: color.NRGBA{R: 255, A: 255}, Breite: 40}}
		l.Ebene = l.Ebene.MitDeckkraft(p)
		hinzu(t, d, l)
		b, err := Bild(d)
		if err != nil {
			t.Fatal(err)
		}
		m := map[image.Point]bool{}
		for y := 0; y < b.Rect.Dy(); y++ {
			for x := 0; x < b.Rect.Dx(); x++ {
				if b.NRGBAAt(x, y) != grund.NRGBAAt(x, y) {
					m[image.Pt(x, y)] = true
				}
			}
		}
		return m
	}
	voll, halb := spur(100), spur(99)
	fehlt, rechts := 0, 0
	for p := range voll {
		// Nur rechts vom Ziel zählen: An den geglätteten Rändern der Linie fällt ein Pixel mit winziger Deckung bei 99 %
		// auf den Grund zurück (Rundung, kein Fehler des Zwischenbilds).
		if !halb[p] && p.X >= 150 {
			fehlt++
		}
		if p.X >= 176 {
			rechts++
		}
	}
	if rechts == 0 {
		t.Fatal("Vorbedingung: der Rahmen reicht nicht hinter den Spalt")
	}
	if fehlt > 0 {
		t.Errorf("%d Pixel fehlen mit Gruppendeckkraft (Rand des Zwischenbilds zu knapp)", fehlt)
	}
}
