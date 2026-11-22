// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"errors"
	"math"
	"reflect"
	"strconv"
	"testing"
)

// ADR-0023 Nachtrag 6: Namen der Formen in Arbeitsdatei und Einstellung, Reihenfolge des Menüs.
func TestKanteNamen(t *testing.T) {
	for _, c := range []struct {
		k    Kante
		name string
	}{{KanteAbriss, "tear"}, {KanteWelle, "wave"}, {KanteZacken, "zigzag"}} {
		if c.k.Name() != c.name || c.k.String() != c.name {
			t.Errorf("%d: Name %q, Soll %q", c.k, c.k.Name(), c.name)
		}
		if k, ok := KanteAus(c.name); !ok || k != c.k {
			t.Errorf("KanteAus(%q) = %v %v", c.name, k, ok)
		}
	}
	for _, name := range []string{"", "Tear", "zickzack", " wave"} {
		if k, ok := KanteAus(name); ok || k != KanteZacken {
			t.Errorf("KanteAus(%q) = %v %v, Soll zigzag und false", name, k, ok)
		}
	}
	if Kante(7).Name() != "" {
		t.Error("ungültige Form hat einen Namen")
	}
	if KanteAbriss.Naechste() != KanteWelle || KanteWelle.Naechste() != KanteZacken || KanteZacken.Naechste() != KanteAbriss {
		t.Error("Naechste läuft nicht reihum tear → wave → zigzag → tear")
	}
}

// abrissSoll rechnet die Formel aus ADR-0023 Nachtrag 6 unabhängig von Kante.mitte nach.
func abrissSoll(t float64) float64 {
	return 0.8 * (2.6*math.Sin(2*math.Pi*t/71+0.3) + 1.6*math.Sin(2*math.Pi*t/29+1.9) +
		1.0*math.Sin(2*math.Pi*t/13+4.1) + 0.6*math.Sin(2*math.Pi*t/7.3+2.2) + 0.35*math.Sin(2*math.Pi*t/3.1+5.7))
}

// Die Mittellinien der drei Formen nach dem ADR, abgetastet je 1 px (Abriss), 2 px (Welle), 8 px (Zacken-Ecken), auf
// Hundertstel gerundet; bei Faktor 2 doppelt so weit und doppelt so hoch.
func TestKantenbandFormen(t *testing.T) {
	n := Naht{Zeilen: true, Lage: 100}
	for _, c := range []struct {
		k       Kante
		schritt float64
		soll    func(float64) float64
	}{
		{KanteAbriss, 1, abrissSoll},
		{KanteWelle, 2, func(t float64) float64 { return 4 * math.Sin(2*math.Pi*t/32) }},
		{KanteZacken, 8, func(t float64) float64 { return 4 * float64(int(t/8)%2*2-1) }},
	} {
		for _, f := range []int{1, 2} {
			band := Kantenband(c.k, n, 300, f, -1.5, 3)
			anz := len(band)/2 - 1
			if want := int(math.Ceil(300 / (c.schritt * float64(f)))); anz != want {
				t.Fatalf("%v Faktor %d: %d Schritte, Soll %d", c.k, f, anz, want)
			}
			for i := 0; i <= anz; i++ {
				tb := float64(i) * c.schritt
				m := math.Round(c.soll(tb)*100) / 100 * float64(f)
				oben, unten := band[i], band[len(band)-1-i]
				if oben.X != tb*float64(f) || unten.X != oben.X {
					t.Fatalf("%v Faktor %d Punkt %d: x %v/%v, Soll %v", c.k, f, i, oben.X, unten.X, tb*float64(f))
				}
				if math.Abs(oben.Y-(100+m-1.5*float64(f))) > 1e-9 || math.Abs(unten.Y-(100+m+3*float64(f))) > 1e-9 {
					t.Fatalf("%v Faktor %d Punkt %d: y %v/%v, Mitte %v", c.k, f, i, oben.Y, unten.Y, 100+m)
				}
			}
		}
	}
	// Fest, ohne Zufall: zweimal dasselbe.
	if !reflect.DeepEqual(Kantenband(KanteAbriss, n, 500, 1, -5, 5), Kantenband(KanteAbriss, n, 500, 1, -5, 5)) {
		t.Error("Abriss nicht fest")
	}
}

// Kurze Zahlen: Jede Querlage ist eine ganze Zahl von Hundertsteln und als solche geschrieben (SVG-Export).
func TestKantenbandHundertstel(t *testing.T) {
	for _, k := range []Kante{KanteAbriss, KanteWelle, KanteZacken} {
		for _, p := range Kantenband(k, Naht{Zeilen: true, Lage: 1234}, 400, 3, -6, 5) {
			if math.Round(p.Y*100)/100 != p.Y {
				t.Fatalf("%v: Querlage %v keine ganzen Hundertstel", k, p.Y)
			}
			if s := len(strconv.FormatFloat(p.Y, 'g', -1, 64)); s > 8 {
				t.Fatalf("%v: Querlage %v wird als %d Zeichen geschrieben", k, p.Y, s)
			}
		}
	}
}

// ADR-0023 Nachtrag 6: Die Mittellinie lenkt höchstens 5 aus (Abriss 4,92), Spalt ±5 und Kante 1 — die halbe Bandhöhe
// ist 11, und keine Schicht ragt darüber hinaus.
func TestKantenSchichtenImBand(t *testing.T) {
	if ZackenBandHalb != 11 {
		t.Fatalf("ZackenBandHalb %v, Soll 11", float64(ZackenBandHalb))
	}
	for _, k := range []Kante{KanteAbriss, KanteWelle, KanteZacken} {
		groesste := 0.0
		for tb := 0.0; tb < 5000; tb += 0.25 {
			groesste = math.Max(groesste, math.Abs(k.mitte(tb)))
		}
		if groesste > KantenAuslenkung || groesste < 3.9 {
			t.Errorf("%v: größte Auslenkung %v, Soll höchstens %v", k, groesste, KantenAuslenkung)
		}
	}
	s := KantenSchichten()
	weiteste := 0.0
	var oben, unten float64 // Summe der Deckkraft der Schatten je Seite
	for i, sch := range s {
		if !(sch.Von < sch.Bis) || sch.Farbe.A == 0 {
			t.Errorf("Schicht %d: %+v leer", i, sch)
		}
		weiteste = math.Max(weiteste, math.Max(math.Abs(sch.Von), math.Abs(sch.Bis)))
		if sch.Farbe.A < 255 && sch.Von >= -KantenSpaltHalb && sch.Bis <= KantenSpaltHalb {
			if sch.Von == -KantenSpaltHalb {
				oben += float64(sch.Farbe.A)
			} else {
				unten += float64(sch.Farbe.A)
			}
		}
	}
	if weiteste+KantenAuslenkung != ZackenBandHalb {
		t.Errorf("weiteste Schicht %v + Auslenkung %v ≠ %v", weiteste, KantenAuslenkung, float64(ZackenBandHalb))
	}
	if s[0] != (Schicht{-5, 5, KantenSpalt}) || KantenSpalt.A != 255 {
		t.Errorf("erste Schicht %+v, Soll der deckende Spalt ±5", s[0])
	}
	if oben <= unten || unten == 0 {
		t.Errorf("Schatten oben %v, unten %v: oben soll kräftiger sein", oben, unten)
	}
	s[0].Von = 99
	if KantenSchichten()[0].Von != -5 {
		t.Error("KantenSchichten liefert keine Kopie")
	}
}

// Die Form wechselt als ein Rückgängig-Schritt; dieselbe Form ist keiner, eine ungültige ein Fehler; Abzug nimmt sie mit.
func TestSetzeKante(t *testing.T) {
	d := leeresDokument(t, 50, 40)
	if d.Kante() != KanteAbriss {
		t.Fatalf("neues Dokument: %v, Soll tear", d.Kante())
	}
	vorher := d.Stand()
	if err := d.SetzeKante(KanteAbriss); err != nil || d.KannRueckgaengig() {
		t.Errorf("dieselbe Form: %v, Schritt %v", err, d.KannRueckgaengig())
	}
	if err := d.SetzeKante(KanteWelle); err != nil || d.Kante() != KanteWelle || d.Stand() == vorher {
		t.Fatalf("SetzeKante: %v %v", err, d.Kante())
	}
	a, err := d.Abzug()
	if err != nil || a.Kante() != KanteWelle {
		t.Errorf("Abzug: %v %v", a.Kante(), err)
	}
	if err := d.Rueckgaengig(); err != nil || d.Kante() != KanteAbriss || d.Stand() != vorher {
		t.Errorf("Rückgängig: %v %v", err, d.Kante())
	}
	if err := d.Wiederholen(); err != nil || d.Kante() != KanteWelle {
		t.Errorf("Wiederholen: %v %v", err, d.Kante())
	}
	if err := d.SetzeKante(Kante(3)); err == nil || d.Kante() != KanteWelle {
		t.Errorf("ungültige Form angenommen: %v", d.Kante())
	}
	if _, err := d.BeginneNeuenZug(Text{Ort: Pt(2, 20), Inhalt: "z", Schriftgrad: 10}); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeKante(KanteZacken); !errors.Is(err, ErrZugLaeuft) {
		t.Errorf("während eines Zugs: %v", err)
	}
}

// VorgabeKante setzt ohne Schritt und ohne neuen Stand, nur solange nichts geändert ist.
func TestVorgabeKante(t *testing.T) {
	d := leeresDokument(t, 50, 40)
	vorher := d.Stand()
	if err := d.VorgabeKante(KanteZacken); err != nil || d.Kante() != KanteZacken || d.Stand() != vorher || d.KannRueckgaengig() {
		t.Fatalf("Vorgabe: %v %v", err, d.Kante())
	}
	if err := d.VorgabeKante(Kante(9)); err == nil {
		t.Error("ungültige Form angenommen")
	}
	if err := d.SetzeKante(KanteWelle); err != nil {
		t.Fatal(err)
	}
	if err := d.VorgabeKante(KanteAbriss); !errors.Is(err, ErrVerlaufNichtLeer) || d.Kante() != KanteWelle {
		t.Errorf("nach einer Änderung: %v %v", err, d.Kante())
	}
	if err := d.Rueckgaengig(); err != nil {
		t.Fatal(err)
	}
	if err := d.VorgabeKante(KanteAbriss); !errors.Is(err, ErrVerlaufNichtLeer) {
		t.Errorf("mit zurückgenommenem Schritt: %v", err)
	}
	if err := d.VergissVerlauf(); err != nil {
		t.Fatal(err)
	}
	if err := d.VorgabeKante(KanteAbriss); err != nil || d.Kante() != KanteAbriss {
		t.Errorf("nach VergissVerlauf: %v %v", err, d.Kante())
	}
}
