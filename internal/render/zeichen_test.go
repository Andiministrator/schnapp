// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"errors"
	"image"
	"image/color"
	"strings"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

// verborgeneTexte legt n Texte à zeichen Runen weit außerhalb der Ansicht an (präparierte Datei aus F-224).
func verborgeneTexte(t *testing.T, n, zeichen int) *document.Dokument {
	t.Helper()
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 100, 100)))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < n; i++ {
		// Jeder Inhalt anders, sonst vermisst der Zwischenspeicher nur einen.
		inhalt := string(rune('A'+i%26)) + string(rune('A'+i/26%26)) + strings.Repeat("W", zeichen-2)
		if _, err := d.Hinzufuegen(document.Text{Ort: document.Pt(900000, 900000+float64(i)), Schriftgrad: 20,
			Inhalt: inhalt, Farbe: color.NRGBA{A: 255}}); err != nil {
			t.Fatal(err)
		}
	}
	return d
}

// F-224: Verborgene Texte rastert niemand, vermessen werden sie trotzdem. Pruefe lehnt eine Datei mit mehr Zeichen als
// ZeichenBudget ab, bevor es vermisst — schnell, auch wenn der Aufwand sonst 0 wäre.
func TestPruefeZaehltVerborgeneTexte(t *testing.T) {
	d := verborgeneTexte(t, 400, document.MaxTextZeichen)
	start := time.Now()
	err := Pruefe(d)
	dauer := time.Since(start)
	if !errors.Is(err, ErrZuAufwendig) || !strings.Contains(err.Error(), "Zeichen") {
		t.Fatalf("Pruefe: %v (Soll: abgelehnt wegen Zeichen)", err)
	}
	if zeichenZeit > 0 && dauer > zeichenZeit {
		t.Errorf("Ablehnen dauerte %v (Soll < %v: zählen, nicht vermessen)", dauer, zeichenZeit)
	}
}

// Genau am ZeichenBudget wird nicht abgelehnt; ein Zeichen mehr schon.
func TestZeichenBudgetGrenze(t *testing.T) {
	alt := grenzen
	defer func() { grenzen = alt }()
	grenzen.zeichen = 30
	d := verborgeneTexte(t, 3, 10)
	if err := Pruefe(d); err != nil {
		t.Errorf("genau am Budget: %v", err)
	}
	if _, err := d.Hinzufuegen(document.Text{Ort: document.Pt(5, 5), Schriftgrad: 20, Inhalt: "ä",
		Farbe: color.NRGBA{A: 255}}); err != nil {
		t.Fatal(err)
	}
	if err := Pruefe(d); !errors.Is(err, ErrZuAufwendig) {
		t.Errorf("ein Zeichen (Rune, nicht Byte) darüber: %v", err)
	}
}

// Unter dem Budget hält der Zwischenspeicher der Tinte jeden weiteren Durchlauf kurz (F-224): Der zweite Trockenlauf
// über knapp eine Million verborgene Zeichen vermisst nichts mehr.
func TestVerborgeneTexteZweiterDurchlaufKurz(t *testing.T) {
	d := verborgeneTexte(t, ZeichenBudget/document.MaxTextZeichen, document.MaxTextZeichen)
	start := time.Now()
	if err := Pruefe(d); err != nil {
		t.Fatal(err)
	}
	t.Logf("erster Durchlauf (vermessen): %v", time.Since(start))
	start = time.Now()
	if err := Pruefe(d); err != nil {
		t.Fatal(err)
	}
	if dauer := time.Since(start); zeichenZeit > 0 && dauer > zeichenZeit {
		t.Errorf("zweiter Durchlauf %v (Soll < %v)", dauer, zeichenZeit)
	}
}
