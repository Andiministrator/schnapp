// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"errors"
	"image/color"
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

// F-439 (ADR-0023 Nachtrag 8, entschieden: so lassen): Die gewählte Form gehört zum Dokument, nicht zu den Nähten. Sind
// alle Nähte zurückgeholt, bleibt sie; die nächste Naht erscheint in dieser Form, nicht in der aus `[capture] seam`.
// Rückgängig und Wiederholen der Form bleiben davon unberührt.
func TestKanteBleibtOhneNaehte(t *testing.T) {
	d := leeresDokument(t, 200, 100)
	if err := d.SetzeKante(KanteWelle); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]Streifen{{Zeilen: true, Von: 30, Bis: 40}}); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen(nil); err != nil {
		t.Fatal(err)
	}
	if len(d.Naehte()) != 0 || d.Kante() != KanteWelle {
		t.Errorf("ohne Naht: Form %v, Nähte %v; Soll Welle bleibt", d.Kante(), d.Naehte())
	}
	if err := d.SetzeStreifen([]Streifen{{Von: 50, Bis: 60}}); err != nil {
		t.Fatal(err)
	}
	if d.Kante() != KanteWelle {
		t.Errorf("neue Naht in Form %v, Soll Welle", d.Kante())
	}
}

// abrissSoll rechnet die Formel aus ADR-0023 Nachtrag 6 (Faktor 0,9 nach Nachtrag 7) unabhängig von Kante.mitte nach.
func abrissSoll(t float64) float64 {
	return 0.9 * (2.6*math.Sin(2*math.Pi*t/71+0.3) + 1.6*math.Sin(2*math.Pi*t/29+1.9) +
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

// ADR-0023 Nachtrag 7: Die Mittellinie lenkt höchstens 6 aus (Abriss 0,9 · 6,15 = 5,54), Spalt ±8 und Rand 2 — die
// halbe Bandhöhe ist 16, und keine Schicht ragt darüber hinaus. Die Schichttabelle steht hier als Literal (F-421): Wer
// eine Schicht entfernt, ihre Deckkraft oder Lage ändert, fällt hier auf, nicht erst im Bild.
func TestKantenSchichtenImBand(t *testing.T) {
	if ZackenBandHalb != 16 || KantenSpaltHalb != 8 || abrissFaktor != 0.9 {
		t.Fatalf("ZackenBandHalb %v, Spalt %v, Abriss %v; Soll 16, 8, 0,9", float64(ZackenBandHalb),
			float64(KantenSpaltHalb), float64(abrissFaktor))
	}
	if von, bis := Spaltband(); von != -8 || bis != 8 {
		t.Errorf("Spaltband %v..%v, Soll −8..8", von, bis)
	}
	for _, k := range []Kante{KanteAbriss, KanteWelle, KanteZacken} {
		groesste := 0.0
		for tb := 0.0; tb < 5000; tb += 0.25 {
			groesste = math.Max(groesste, math.Abs(k.mitte(tb)))
		}
		if groesste > KantenAuslenkung || groesste < 3.9 {
			t.Errorf("%v: größte Auslenkung %v, Soll höchstens %v", k, groesste, KantenAuslenkung)
		}
		if k == KanteAbriss && groesste < 5 {
			t.Errorf("Abriss: größte Auslenkung %v, mit Faktor 0,9 Soll über 5", groesste)
		}
	}
	weiss := func(a uint8) color.NRGBA { return color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: a} }
	schwarz := func(a uint8) color.NRGBA { return color.NRGBA{A: a} }
	schatten := func(a uint8) color.NRGBA { return color.NRGBA{R: 0x10, G: 0x12, B: 0x14, A: a} }
	soll := []Schicht{
		{-9.6, -8, weiss(0x30)}, {-8.8, -8, weiss(0x76)}, // Fasersaum oben bzw. links
		{8, 9.6, weiss(0x30)}, {8, 8.8, weiss(0x76)}, // Fasersaum unten bzw. rechts
		{-10, -8, schwarz(0x08)}, {-9, -8, schwarz(0x13)}, // Abdunklung oben bzw. links
		{8, 10, schwarz(0x05)}, {8, 9, schwarz(0x0D)}, // Abdunklung unten bzw. rechts
		{-8, -1, schatten(0x08)}, {-8, -4, schatten(0x10)}, {-8, -6, schatten(0x18)}, {-8, -7, schatten(0x20)}, // Schatten
	}
	s := KantenSchichten()
	if !reflect.DeepEqual(s, soll) {
		t.Fatalf("Schichten\n%v\nSoll\n%v", s, soll)
	}
	weiteste := 0.0
	for i, sch := range s {
		if !(sch.Von < sch.Bis) || sch.Farbe.A == 0 || sch.Farbe.A == 255 {
			t.Errorf("Schicht %d: %+v leer oder deckend", i, sch)
		}
		weiteste = math.Max(weiteste, math.Max(math.Abs(sch.Von), math.Abs(sch.Bis)))
	}
	if weiteste+KantenAuslenkung != ZackenBandHalb {
		t.Errorf("weiteste Schicht %v + Auslenkung %v ≠ %v", weiteste, KantenAuslenkung, float64(ZackenBandHalb))
	}
	s[0].Von = 99
	if KantenSchichten()[0].Von != -9.6 {
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
