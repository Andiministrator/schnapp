// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"testing"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// schattenLauf legt einen Kasten an, wählt ihn und zeichnet die Leiste.
func schattenLauf(t *testing.T) (*menueLauf, int) {
	t.Helper()
	l := neuerMenueLauf(t, image.Pt(1200, 1000))
	id, err := l.e.s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(20, 20), document.Pt(120, 80)),
		Strich: document.Strich{Farbe: color.NRGBA{R: 200, A: 255}, Breite: 4}})
	if err != nil {
		t.Fatal(err)
	}
	l.e.s.Auswahl = id
	l.setle()
	return l, id
}

func schattenVon(t *testing.T, l *menueLauf, id int) document.Schatten {
	t.Helper()
	o, ok := l.e.s.D.Objekt(id)
	if !ok {
		t.Fatal("Objekt fehlt")
	}
	return o.Ebenenwerte().Schatten
}

// ADR-0021 Nachtrag 26 Punkt 7: Der Schalter schaltet den Schatten des gewählten Kastens an (ein Rückgängig-Schritt),
// erst dann stehen Richtung, Regler und Farbe da; − / + drehen in 15°-Schritten.
func TestSchattenLeisteSchalterUndRichtung(t *testing.T) {
	l, id := schattenLauf(t)
	k := &l.e.schattenL
	if k.schalterRahmen.Empty() {
		t.Fatal("Schalter nicht gezeichnet")
	}
	if !k.farbRahmen.Empty() || !k.laenge.Empty() {
		t.Error("Einstellungen stehen bei ausgeschaltetem Schatten da")
	}
	l.klick(mitte(k.schalterRahmen))
	l.setle()
	if sch := schattenVon(t, l, id); !sch.An || sch.Winkel != 135 {
		t.Fatalf("nach dem Einschalten %+v", sch)
	}
	if k.farbRahmen.Empty() || k.laenge.Empty() || k.weich.Empty() || k.richtText != "135°" {
		t.Fatalf("Einstellungen fehlen: Richtung %q", k.richtText)
	}
	l.klick(mitte(k.richtRahmen[1]))
	l.setle()
	if sch := schattenVon(t, l, id); sch.Winkel != 150 || k.richtText != "150°" {
		t.Errorf("nach +: %v, Anzeige %q", sch.Winkel, k.richtText)
	}
	l.klick(mitte(k.richtRahmen[0]))
	l.klick(mitte(k.richtRahmen[0]))
	l.setle()
	if sch := schattenVon(t, l, id); sch.Winkel != 120 {
		t.Errorf("nach zweimal −: %v", sch.Winkel)
	}
	for range 3 {
		l.e.s.Rueckgaengig()
	}
	if sch := schattenVon(t, l, id); !sch.An || sch.Winkel != 135 {
		t.Errorf("drei Rückgängig: %+v", sch)
	}
	l.e.s.Rueckgaengig()
	if schattenVon(t, l, id).An {
		t.Error("das Einschalten ist kein eigener Rückgängig-Schritt")
	}
	// Ausschalten: am Objekt der Nullwert, die Einstellungen verschwinden.
	l.setle()
	l.klick(mitte(k.schalterRahmen))
	l.setle()
	l.klick(mitte(k.schalterRahmen))
	l.setle()
	if sch := schattenVon(t, l, id); sch != (document.Schatten{}) || !k.farbRahmen.Empty() {
		t.Errorf("nach dem Ausschalten %+v", sch)
	}
}

// Ziehen an „Distance“ und „Softness“ ist je ein Rückgängig-Schritt; über das Ende hinaus 40 bzw. 20 px (Faktor 1).
func TestSchattenLeisteRegler(t *testing.T) {
	l, id := schattenLauf(t)
	k := &l.e.schattenL
	l.klick(mitte(k.schalterRahmen))
	l.setle()
	vor := schattenVon(t, l, id)
	l.ziehe(lageIn(k.laenge, 0.3, 0.5), lageIn(k.laenge, 1.3, 0.5))
	if sch := schattenVon(t, l, id); sch.Laenge != canvas.SchattenLaengeRegler || sch.Weich != vor.Weich {
		t.Fatalf("nach dem Ziehen an Distance: %+v", sch)
	}
	if k.zieht != schattenReglerKeiner || l.e.s.BreiteZugLaeuft() {
		t.Error("Zug hängt nach dem Loslassen")
	}
	l.setle()
	l.ziehe(lageIn(k.weich, 0.8, 0.5), lageIn(k.weich, 0.5, 0.5))
	if sch := schattenVon(t, l, id); sch.Weich != canvas.SchattenWeichRegler/2 || sch.Laenge != canvas.SchattenLaengeRegler {
		t.Fatalf("nach dem Ziehen an Softness: %+v", sch)
	}
	l.e.s.Rueckgaengig()
	if sch := schattenVon(t, l, id); sch.Weich != vor.Weich || sch.Laenge != canvas.SchattenLaengeRegler {
		t.Errorf("ein Rückgängig nach Softness: %+v", sch)
	}
	l.e.s.Rueckgaengig()
	if sch := schattenVon(t, l, id); sch.Laenge != vor.Laenge {
		t.Errorf("ein Rückgängig nach Distance: %+v", sch)
	}
}

// Das Farbfeld macht den Schatten zum aktiven Feld des Farbwählers, klappt den Abschnitt „Farbe“ auf und benennt das
// Feld; die Farbwahl färbt dann den Schatten, nicht den Strich.
func TestSchattenLeisteFarbe(t *testing.T) {
	l, id := schattenLauf(t)
	k := &l.e.schattenL
	l.klick(mitte(k.schalterRahmen))
	l.setle()
	l.e.seite.farbeZu = true
	l.setle()
	l.klick(mitte(k.farbRahmen))
	l.setle()
	if l.e.s.AktivesFeld() != canvas.FeldSchatten || l.e.seite.farbeZu || !l.e.seite.farbeGezeigt {
		t.Fatalf("Feld %v, Farbe zu %v", l.e.s.AktivesFeld(), l.e.seite.farbeZu)
	}
	if n := farbfeldName(l.e.s.Fuellbar(), l.e.s.AktivesFeld(), color.NRGBA{}); n != "Shadow" {
		t.Errorf("Name des aktiven Felds %q", n)
	}
	gruen := color.NRGBA{G: 180, A: 90}
	l.e.s.SetzeFarbe(gruen)
	o, _ := l.e.s.D.Objekt(id)
	if sch := schattenVon(t, l, id); sch.Farbe != gruen || o.(document.Kasten).Strich.Farbe == gruen {
		t.Errorf("Schattenfarbe %v, Strich %v", sch.Farbe, o.(document.Kasten).Strich.Farbe)
	}
}

// Ohne Farbe (Auswählen ohne Objekt) steht kein Abschnitt da, und seine Knöpfe fallen aus der Fokusfahrt.
func TestSchattenLeisteNurMitFarbe(t *testing.T) {
	l, _ := schattenLauf(t)
	l.e.s.Auswahl = 0
	l.e.s.SetzeWerkzeug(canvas.WerkzeugAuswahl)
	l.setle()
	if !l.e.schattenL.schalterRahmen.Empty() || len(l.e.schattenL.alle()) != 0 {
		t.Error("Abschnitt Shadow ohne Farbe gezeigt")
	}
}
