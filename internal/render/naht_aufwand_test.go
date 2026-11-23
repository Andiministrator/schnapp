// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"errors"
	"image"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

// nahtDokument ist ein leeres Bild b×h in der Form k mit zn Zeilen-Nähten in gleichen Abständen (je ein Streifen von
// einer Zeile).
func nahtDokument(t testing.TB, b, h int, k document.Kante, zn int) *document.Dokument {
	t.Helper()
	d, err := document.NeuAus(image.NewNRGBA(image.Rect(0, 0, b, h)))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.VorgabeKante(k); err != nil {
		t.Fatal(err)
	}
	var st []document.Streifen
	for i := 0; i < zn; i++ {
		v := (i + 1) * h / (zn + 1)
		st = append(st, document.Streifen{Zeilen: true, Von: v, Bis: v + 1})
	}
	if err := d.SetzeStreifen(st); err != nil {
		t.Fatal(err)
	}
	return d
}

// F-418: Der Trockenlauf von Pruefe hört auf, sobald eine Grenze überschritten ist — bei den Nähten wie bei den
// Objekten. Geprüft am gezählten Aufwand, nicht an der Uhr: Er liegt knapp über der Grenze und weit unter dem vollen.
// Aufwand und Bild zählen weiter alles.
func TestPruefeBrichtUeberDerGrenzeAb(t *testing.T) {
	alt := grenzen
	defer func() { grenzen = alt }()

	d := nahtDokument(t, 2000, 400, document.KanteAbriss, 40)
	voll := trockenlauf(d, false)
	if Aufwand(d) != voll.summe() {
		t.Fatalf("Aufwand %v, voller Trockenlauf %v", Aufwand(d), voll.summe())
	}
	grenzen.kanten = voll.kanten / 20
	teil := trockenlauf(d, true)
	if teil.kanten <= grenzen.kanten {
		t.Errorf("Nähte: Abbruch vor der Grenze (%v ≤ %v)", teil.kanten, grenzen.kanten)
	}
	if teil.kanten > voll.kanten/10 {
		t.Errorf("Nähte: kein Abbruch, gezählt %v von %v", teil.kanten, voll.kanten)
	}
	if err := Pruefe(d); !errors.Is(err, ErrZuAufwendig) {
		t.Errorf("Pruefe: %v, Soll ErrZuAufwendig", err)
	}
	// Eine Grenze, die erst in den Schichten erreicht wird (die Spalte kosten zusammen rund 14 %): auch dort bricht der
	// Trockenlauf ab (F-418, F-438: die Nähte hier liegen eng, die Spalte werden zweimal geleert).
	grenzen.kanten = voll.kanten / 3
	if teil := trockenlauf(d, true); teil.kanten <= grenzen.kanten || teil.kanten > voll.kanten/2 {
		t.Errorf("Nähte, späte Grenze: gezählt %v von %v, Grenze %v", teil.kanten, voll.kanten, grenzen.kanten)
	}
	grenzen.kanten = voll.kanten / 20
	if a := trockenlauf(d, false); a != voll {
		t.Errorf("ohne Abbruch %+v, vorher %+v", a, voll)
	}
	if _, a, err := bildMitAufwand(d); err != nil || a.summe() != voll.summe() {
		t.Errorf("Bild zählt %v (%v), Soll %v", a.summe(), err, voll.summe())
	}

	grenzen = alt
	e := weissesDokument(t, 200, 200)
	for i := 0; i < 100; i++ {
		hinzu(t, e, document.Kasten{Bereich: document.R(document.Pt(0, 0), document.Pt(200, 200)), Fuellung: blau})
	}
	voll = trockenlauf(e, false)
	grenzen.gesamt = voll.summe() / 20
	teil = trockenlauf(e, true)
	if teil.summe() <= grenzen.gesamt || teil.summe() > voll.summe()/10 {
		t.Errorf("Objekte: gezählt %v, Grenze %v, voll %v", teil.summe(), grenzen.gesamt, voll.summe())
	}
	if err := Pruefe(e); !errors.Is(err, ErrZuAufwendig) {
		t.Errorf("Pruefe Objekte: %v, Soll ErrZuAufwendig", err)
	}

	// Dritte Grenze (F-434 a): die der Glyphen. Gesamt und Kanten bleiben auf ihren Vorgaben, so kann nur der
	// Glyphen-Zweig von abgebrochen() das Zählen beenden. Jeder der 100 Texte kostet 1 % des vollen Aufwands.
	grenzen = alt
	g := weissesDokument(t, 200, 200)
	for i := 0; i < 100; i++ {
		hinzu(t, g, document.Text{Ort: document.Pt(10, float64(i)), Inhalt: "Hallo", Schriftgrad: 20, Farbe: rot})
	}
	voll = trockenlauf(g, false)
	if voll.glyph == 0 {
		t.Fatal("Texte kosten keine Glyphen")
	}
	grenzen.glyph = voll.glyph / 20
	teil = trockenlauf(g, true)
	if teil.glyph <= grenzen.glyph || teil.glyph > voll.glyph/10 {
		t.Errorf("Glyphen: gezählt %v, Grenze %v, voll %v", teil.glyph, grenzen.glyph, voll.glyph)
	}
	if err := Pruefe(g); !errors.Is(err, ErrZuAufwendig) {
		t.Errorf("Pruefe Glyphen: %v, Soll ErrZuAufwendig", err)
	}
}

// F-418: 1023 Abriss-Nähte auf 32768×2100 (präparierte Datei) lehnt Pruefe ab, ohne lange zu blockieren — vorher rund
// 52 s, nachher rund 2 s (ein Kern). Die Grenze ist großzügig; unter -race und mit -short entfällt der Test.
func TestPruefeVieleAbrissNaehte(t *testing.T) {
	if testing.Short() || pruefeNahtZeit == 0 {
		t.Skip("lang (275 MB Bild); nicht unter -short oder -race")
	}
	d := nahtDokument(t, 32768, 2100, document.KanteAbriss, 1023)
	start := time.Now()
	err := Pruefe(d)
	dauer := time.Since(start)
	if !errors.Is(err, ErrZuAufwendig) {
		t.Fatalf("Pruefe: %v, Soll ErrZuAufwendig", err)
	}
	if dauer > pruefeNahtZeit {
		t.Errorf("Pruefe brauchte %v, Grenze %v", dauer, pruefeNahtZeit)
	}
	t.Logf("Pruefe 1023 Abriss-Nähte: %v", dauer)
}
