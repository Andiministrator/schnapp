// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"testing"
)

// leistenKnoepfe sind 14 Knöpfe wie im Editor: neun Werkzeuge, dann Farbe | Rückgängig, Wiederholen | Einpassen, 1:1.
// Breiten in Pixeln bei 1 dp = 1 px.
func leistenKnoepfe() []LeistenKnopf {
	var k []LeistenKnopf
	for i := 0; i < 9; i++ {
		k = append(k, LeistenKnopf{Breite: 80})
	}
	return append(k,
		LeistenKnopf{Breite: 90, Gruppe: true},
		LeistenKnopf{Breite: 70, Gruppe: true},
		LeistenKnopf{Breite: 70},
		LeistenKnopf{Breite: 60, Gruppe: true},
		LeistenKnopf{Breite: 50},
	)
}

func leistenMass(breite int) LeistenMass {
	return LeistenMass{RandX: 12, RandY: 10, Abstand: 8, KnopfHoehe: 36, Speichern: 90, Breite: breite}
}

// Breites Fenster: eine Zeile, Positionen wie der frühere Einzeilen-Code (x += Breite + Abstand, Gruppe +Abstand), Höhe 56.
func TestLeisteBreiteEineZeile(t *testing.T) {
	k := leistenKnoepfe()
	pos, h, z := LeisteAnordnen(k, leistenMass(2000))
	if z != 1 || h != 56 {
		t.Fatalf("Zeilen %d Höhe %d, soll 1 und 56", z, h)
	}
	x := 12
	for i, kn := range k {
		if kn.Gruppe {
			x += 8
		}
		if pos[i] != image.Pt(x, 10) {
			t.Fatalf("Knopf %d bei %v, soll %v", i, pos[i], image.Pt(x, 10))
		}
		x += kn.Breite + 8
	}
}

// Schmales Fenster (480): mehrere Zeilen; nichts überlappt, nichts ragt über den rechten Rand oder unter „Speichern“.
func TestLeisteSchmalUmbruch(t *testing.T) {
	k := leistenKnoepfe()
	m := leistenMass(480)
	pos, h, z := LeisteAnordnen(k, m)
	if z < 2 {
		t.Fatalf("480 px: %d Zeile(n), soll Umbruch", z)
	}
	// Sollwerte: Zeile 1 bis x=480-12-90-8=370: 12,100,188,276 (Knopf 4 endet 356+... 276+80=356 ≤ 370), Knopf 4 bei 364 passt nicht.
	if pos[3] != image.Pt(276, 10) || pos[4] != image.Pt(12, 54) {
		t.Fatalf("Umbruchstelle: Knopf 3 %v, Knopf 4 %v", pos[3], pos[4])
	}
	if want := 2*10 + z*36 + (z-1)*8; h != want {
		t.Fatalf("Höhe %d, soll %d", h, want)
	}
	speichern := image.Rect(m.Breite-m.RandX-m.Speichern, m.RandY, m.Breite-m.RandX, m.RandY+m.KnopfHoehe)
	var rs []image.Rectangle
	for i, kn := range k {
		r := image.Rectangle{Min: pos[i], Max: pos[i].Add(image.Pt(kn.Breite, m.KnopfHoehe))}
		if r.Max.X > m.Breite-m.RandX {
			t.Errorf("Knopf %d ragt über den rechten Rand: %v", i, r)
		}
		if r.Overlaps(speichern) {
			t.Errorf("Knopf %d überlappt Speichern: %v", i, r)
		}
		for j, o := range rs {
			if r.Overlaps(o) {
				t.Errorf("Knopf %d überlappt Knopf %d", i, j)
			}
		}
		rs = append(rs, r)
	}
}

// Ein Knopf, der breiter ist als jede Zeile, steht allein am Zeilenanfang (kein Endlosumbruch, nie geteilt).
func TestLeisteBreiterKnopf(t *testing.T) {
	k := []LeistenKnopf{{Breite: 50}, {Breite: 900}, {Breite: 50}}
	pos, _, z := LeisteAnordnen(k, leistenMass(300))
	if z != 3 || pos[1] != image.Pt(12, 54) || pos[2] != image.Pt(12, 98) {
		t.Fatalf("Zeilen %d, Positionen %v", z, pos)
	}
}

// Eine Gruppe am Zeilenanfang bekommt keinen Zusatzabstand.
func TestLeisteGruppeAmZeilenanfang(t *testing.T) {
	k := []LeistenKnopf{{Breite: 300}, {Breite: 60, Gruppe: true}}
	pos, _, z := LeisteAnordnen(k, leistenMass(480))
	if z != 2 || pos[1] != image.Pt(12, 54) {
		t.Fatalf("Zeilen %d, Positionen %v", z, pos)
	}
}

// Ohne Knöpfe bleibt eine Zeile.
func TestLeisteLeer(t *testing.T) {
	_, h, z := LeisteAnordnen(nil, leistenMass(480))
	if z != 1 || h != 56 {
		t.Fatalf("Zeilen %d Höhe %d", z, h)
	}
}
