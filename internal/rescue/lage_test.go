// SPDX-License-Identifier: GPL-3.0-or-later

package rescue

// Tests zu Lage (F-454): Die Oberfläche sieht am Zähler, dass die Sitzung fremd wurde oder die Sperre später übernahm.

import (
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Eine Sitzung ohne fremdes Fenster hat Zähler 0.
func TestLageUnbeteiligt(t *testing.T) {
	_, k := ausgang(t, "datei")
	s := NeueSicherung(t.TempDir(), k)
	defer s.Ende()
	if l := s.Lage(); l.Fremd || l.Gestoert || l.Nr != 0 {
		t.Errorf("Lage %+v, erwartet unberührt mit Nr 0", l)
	}
}

// Fremd beim Anlegen, frei nach der Übernahme der Sperre (F-436): Der Zähler wächst bei jedem Wechsel.
func TestLageFremdUndUebernahme(t *testing.T) {
	_, k := ausgang(t, "datei")
	ort := t.TempDir()
	a := NeueSicherung(ort, k)
	b := NeueSicherung(ort, k)
	l := b.Lage()
	if !l.Fremd || l.Nr == 0 {
		t.Fatalf("zweites Fenster: Lage %+v, erwartet fremd mit Nr > 0", l)
	}
	d, _ := document.Neu(bild(64, 64))
	bearbeite(t, d)
	b.Sichere(d) // A hält die Sperre noch: bleibt fremd, Zähler gleich
	warteKette(t, b)
	if l2 := b.Lage(); !l2.Fremd || l2.Nr != l.Nr {
		t.Errorf("Lage %+v, erwartet unverändert %+v", l2, l)
	}
	a.Ende()
	mitText(t, d)
	b.Sichere(d)
	warteKette(t, b)
	defer b.Ende()
	if l3 := b.Lage(); l3.Fremd || l3.Nr <= l.Nr {
		t.Errorf("Lage %+v nach Übernahme, erwartet frei mit Nr > %d", l3, l.Nr)
	}
}

// Wird die Sitzung bei NeueBasis fremd, ändert sich die Lage (F-454).
func TestLageNeueBasisFremd(t *testing.T) {
	_, k := ausgang(t, "datei")
	_, k2 := ausgang(t, "neu")
	ort := t.TempDir()
	a := NeueSicherung(ort, k2)
	defer a.Ende()
	b := NeueSicherung(ort, k)
	defer b.Ende()
	if l := b.Lage(); l.Fremd || l.Nr != 0 {
		t.Fatalf("Lage vorher %+v", l)
	}
	b.NeueBasis(k2)
	warteKette(t, b)
	if l := b.Lage(); !l.Fremd || l.Nr == 0 {
		t.Errorf("Lage %+v, erwartet fremd mit Nr > 0", l)
	}
}
