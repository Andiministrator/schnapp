// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"

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

// ADR-0021 Nachtrag 26 Punkt 7, ADR-0031 Nachtrag 13: Das Kästchen schaltet den Schatten des gewählten Kastens an (ein
// Rückgängig-Schritt), erst dann stehen Richtung, Regler und Farbe da. Ein Klick auf die Scheibe setzt die Richtung zum
// Zeiger hin (15°-Raster), Pfeile an der Scheibe mit Fokus drehen um 15°.
func TestSchattenLeisteSchalterUndRichtung(t *testing.T) {
	l, id := schattenLauf(t)
	k := &l.e.schattenL
	if k.schalterRahmen.Empty() {
		t.Fatal("Kästchen nicht gezeichnet")
	}
	if !k.farbRahmen.Empty() || !k.laenge.Empty() || !k.scheibeRahmen.Empty() {
		t.Error("Einstellungen stehen bei ausgeschaltetem Schatten da")
	}
	l.klick(mitte(k.schalterRahmen))
	l.setle()
	if sch := schattenVon(t, l, id); !sch.An || sch.Winkel != 135 {
		t.Fatalf("nach dem Einschalten %+v", sch)
	}
	if k.farbRahmen.Empty() || k.laenge.Empty() || k.weich.Empty() || k.scheibeRahmen.Empty() || k.richtText != "135°" {
		t.Fatalf("Einstellungen fehlen: Richtung %q", k.richtText)
	}
	m, d := mitte(k.scheibeRahmen), k.scheibeRahmen.Dx()/3
	l.klick(m.Add(image.Pt(d, 0))) // rechts: der Schatten fällt nach rechts, Licht von links (180°)
	l.setle()
	if sch := schattenVon(t, l, id); sch.Winkel != 180 || k.richtText != "180°" {
		t.Errorf("nach Klick rechts: %v, Anzeige %q", sch.Winkel, k.richtText)
	}
	l.klick(m.Add(image.Pt(d/5, d))) // unten, ein wenig rechts: 101,3° rastet auf 105° (90° ist genau unten)
	l.setle()
	if sch := schattenVon(t, l, id); sch.Winkel != 105 {
		t.Errorf("nach Klick unten: %v, Soll 105", sch.Winkel)
	}
	l.fokusiere(&k.scheibe)
	l.taste(key.NameRightArrow, 0)
	l.setle()
	if sch := schattenVon(t, l, id); sch.Winkel != 120 {
		t.Errorf("nach Pfeil rechts: %v, Soll 120", sch.Winkel)
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
	if l.e.seite.paarRahmen[paarSchatten].Empty() {
		t.Error("Reiter Schatten fehlt")
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

// Kritik K-4/K-6 #363b: − / + neben den Reglern sind Knöpfe (auch per Tab erreichbar) und ändern um einen Pixel;
// 359,6° zeigt die Leiste als „0°“, nicht als „360°“.
func TestSchattenLeisteStufenUndAnzeige(t *testing.T) {
	l, id := schattenLauf(t)
	k := &l.e.schattenL
	l.klick(mitte(k.schalterRahmen))
	l.setle()
	if len(k.alle()) != 7 {
		t.Fatalf("%d Knöpfe im Abschnitt, Soll 7", len(k.alle()))
	}
	l.klick(mitte(k.stufeRahmen[0][1]))
	l.klick(mitte(k.stufeRahmen[1][0]))
	l.setle()
	if sch := schattenVon(t, l, id); sch.Laenge != 7 || sch.Weich != 7 {
		t.Errorf("nach + Abstand und − Weichheit: %+v", sch)
	}
	sch := schattenVon(t, l, id)
	sch.Winkel = 359.6
	l.e.s.SetzeSchatten(sch)
	l.setle()
	if k.richtText != "0°" {
		t.Errorf("Anzeige %q, Soll 0°", k.richtText)
	}
}

// Verify V-2 #363b: Enter am fokussierten + neben „Distance“ macht genau einen Schritt.
func TestSchattenStufePerTastatur(t *testing.T) {
	l, id := schattenLauf(t)
	k := &l.e.schattenL
	l.klick(mitte(k.schalterRahmen))
	l.setle()
	l.fokusiere(&k.stufe[0][1])
	if !l.fokus(&k.stufe[0][1]) {
		t.Fatal("+ neben Distance nimmt keinen Fokus")
	}
	l.taste(key.NameReturn, 0)
	if sch := schattenVon(t, l, id); sch.Laenge != 7 {
		t.Errorf("Enter am +: Länge %v, Soll 7 (genau ein Schritt)", sch.Laenge)
	}
}

// Verify V-3 #363b: Ein eigener Stil mit Schatten sagt es im Hinweis der Statuszeile.
func TestSchnellNameMitSchatten(t *testing.T) {
	st := canvas.Schnellstil{Art: canvas.SchnellKasten, Breite: 4, Schatten: document.StandardSchatten(1)}
	if !strings.Contains(schnellName(st), "drop shadow") {
		t.Errorf("Hinweis %q ohne Schatten", schnellName(st))
	}
	st.Schatten = document.Schatten{}
	if strings.Contains(schnellName(st), "drop shadow") {
		t.Errorf("Hinweis %q nennt einen Schatten, den es nicht gibt", schnellName(st))
	}
}

// ADR-0031 Nachtrag 13 Punkt 3: Ziehen an der Scheibe ist ein Rückgängig-Schritt; mit Umschalt rastet die Richtung nicht
// ein (ganze Grad).
func TestSchattenScheibeZiehenUndUmschalt(t *testing.T) {
	l, id := schattenLauf(t)
	k := &l.e.schattenL
	l.klick(mitte(k.schalterRahmen))
	l.setle()
	m, d := mitte(k.scheibeRahmen), float32(k.scheibeRahmen.Dx()/3)
	mf := f32.Pt(float32(m.X), float32(m.Y))
	senden := func(kind pointer.Kind, p f32.Point, b pointer.Buttons, mod key.Modifiers) {
		l.r.Queue(pointer.Event{Kind: kind, Source: pointer.Mouse, Buttons: b, Position: p, Modifiers: mod})
	}
	senden(pointer.Move, mf.Add(f32.Pt(d, 0)), 0, 0)
	senden(pointer.Press, mf.Add(f32.Pt(d, 0)), pointer.ButtonPrimary, 0)
	senden(pointer.Move, mf.Add(f32.Pt(d, d*0.2)), pointer.ButtonPrimary, key.ModShift) // 168,7° frei: 169° (Router: Move statt Drag)
	senden(pointer.Release, mf.Add(f32.Pt(d, d*0.2)), 0, key.ModShift)
	l.setle()
	if sch := schattenVon(t, l, id); sch.Winkel != 169 {
		t.Errorf("mit Umschalt gezogen: %v, Soll 169", sch.Winkel)
	}
	if k.zieht != schattenReglerKeiner || l.e.s.BreiteZugLaeuft() {
		t.Error("Zug hängt nach dem Loslassen")
	}
	l.e.s.Rueckgaengig()
	if sch := schattenVon(t, l, id); sch.Winkel != 135 {
		t.Errorf("ein Rückgängig nach dem Ziehen: %v, Soll 135", sch.Winkel)
	}
}
