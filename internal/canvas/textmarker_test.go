// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"math"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// ADR-0048: Das Werkzeug Textmarker legt einen multiplizierenden Pfeil ohne Köpfe an (Umschalt: waagerecht); Leiste,
// Schnellstile und Stil je Werkzeug kennen ihn als eigenes Werkzeug; die Köpfe der Pfeile lassen ihn in Ruhe.

func TestTextmarkerAnlegen(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugTextmarker)
	zieh(s, f(60, 60), f(150, 80))
	m, ok := letztes(t, s).(document.Pfeil)
	if !ok || !m.Multipliziert() || m.Ende != document.KopfKeine || m.KopfAnfang() != document.KopfKeine {
		t.Fatalf("Textmarker %+v", letztes(t, s))
	}
	if m.Strich.Breite != TextmarkerBreite || m.Strich.Farbe != TextmarkerGelb || m.Strich.Art != document.StrichVoll {
		t.Errorf("Stil %+v", m.Strich)
	}
	if m.Deckkraft() != 100 || m.Schatten.An {
		t.Errorf("Ebene %+v", m.Ebene)
	}
	// Umschalt: waagerecht, auch bei steiler Bewegung.
	s.Umschalt = true
	zieh(s, f(20, 20), f(120, 90))
	if m = letztes(t, s).(document.Pfeil); m.A.Y != m.B.Y || m.A.X >= m.B.X || math.Abs(m.B.X-120) > 1e-9 {
		t.Errorf("Umschalt: %v → %v", m.A, m.B)
	}
	// Der Pfeil bleibt ein Pfeil.
	s.Umschalt = false
	s.SetzeWerkzeug(WerkzeugPfeil)
	zieh(s, f(10, 90), f(60, 95))
	if p := letztes(t, s).(document.Pfeil); p.Multipliziert() || p.Ende != document.KopfDreieck {
		t.Errorf("Pfeil %+v", p)
	}
}

// Auf großen Bildern wächst die Breite mit dem Faktor wie jede Voreinstellung (ADR-0021 Nachtrag 7, ADR-0048 Punkt 5).
func TestTextmarkerBreiteMitFaktor(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 2160, 2160))) // kurze Seite 2160: Faktor 2
	if err != nil {
		t.Fatal(err)
	}
	s := NeueSitzung(d)
	if s.faktor() != 2 {
		t.Fatalf("Faktor %v", s.faktor())
	}
	s.SetzeWerkzeug(WerkzeugTextmarker)
	if s.Stil.Breite != 2*TextmarkerBreite {
		t.Errorf("Breite bei Faktor 2: %v", s.Stil.Breite)
	}
}

func TestTextmarkerInLeisteUndSchnellstilen(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugTextmarker)
	if i := s.Seitenleiste(); i.Pfeilform || !i.Strich || !i.Farbe {
		t.Errorf("Leiste beim Werkzeug: %+v", i)
	}
	zieh(s, f(60, 60), f(150, 80))
	id := letztes(t, s).Kennung()
	s.SetzeWerkzeug(WerkzeugAuswahl)
	s.waehleNur(id)
	if w, ok := objektWerkzeug(letztes(t, s)); !ok || w != WerkzeugTextmarker {
		t.Errorf("objektWerkzeug %v", w)
	}
	if i := s.Seitenleiste(); i.Pfeilform || !i.Strich {
		t.Errorf("Leiste beim gewählten Textmarker: %+v", i)
	}
	if !strichWerkzeug(WerkzeugTextmarker) || fuellWerkzeug(WerkzeugTextmarker) || !farbWerkzeug(WerkzeugTextmarker) {
		t.Error("Textmarker: Strich und Farbe, keine Füllung")
	}
	if a, ok := werkzeugArt(WerkzeugTextmarker); !ok || a != SchnellTextmarker {
		t.Errorf("Art %v", a)
	}
	if w, ok := schnellWerkzeug(SchnellTextmarker); !ok || w != WerkzeugTextmarker {
		t.Errorf("Werkzeug der Art %v", w)
	}
	ein := EingebauteSchnellstile(SchnellTextmarker)
	if len(ein) != 3 || ein[0].Farbe != TextmarkerGelb || ein[1].Farbe != TextmarkerGruen || ein[2].Farbe != TextmarkerRosa {
		t.Fatalf("eingebaute Stile %+v", ein)
	}
	for _, st := range ein {
		if b := st.bereinigt(); !b.gueltig() || b.Breite != TextmarkerBreite || st.Name() == "" {
			t.Errorf("Stil %+v", st)
		}
	}
	if st, ok := objektStandOhneSchatten(letztes(t, s)); !ok || st.Art != SchnellTextmarker {
		t.Errorf("Stand: %+v", st)
	}
	if werkzeugAbschnitt(WerkzeugTextmarker) != settings.ArtTextmarker {
		t.Error("Abschnitt in tools.ini")
	}
	// Ein Schnellstil ändert den gewählten Textmarker, die Mischung bleibt.
	if !s.WendeSchnellstilAn(ein[2]) {
		t.Fatal("Schnellstil Rosa ohne Wirkung")
	}
	if m := letztes(t, s).(document.Pfeil); m.Strich.Farbe != TextmarkerRosa || !m.Multipliziert() {
		t.Errorf("nach dem Schnellstil: %+v", m)
	}
	// Eigener Stil wie in styles.ini und zurück.
	eigen, ok := SchnellstilAus(ein[1].Einstellung())
	if !ok || eigen.Art != SchnellTextmarker || eigen.Breite != TextmarkerBreite {
		t.Errorf("Rundlauf über styles.ini: %+v", eigen)
	}
}

// Der Stil des Textmarkers ist ein eigener: Eine andere Breite ändert den Pfeil nicht (ADR-0041).
func TestTextmarkerEigenerStil(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugTextmarker)
	if s.Stil.Breite != TextmarkerBreite || s.Stil.Farbe != TextmarkerGelb {
		t.Fatalf("Voreinstellung %+v", s.Stil)
	}
	s.SetzeBreite(30)
	s.SetzeWerkzeug(WerkzeugPfeil)
	if s.Stil.Breite == 30 || s.Stil.Farbe == TextmarkerGelb {
		t.Errorf("Stil des Textmarkers am Pfeil: %+v", s.Stil)
	}
	s.SetzeWerkzeug(WerkzeugTextmarker)
	if s.Stil.Breite != 30 {
		t.Errorf("Breite des Textmarkers %v", s.Stil.Breite)
	}
}

// ADR-0048 Punkt 2: Die Köpfe der Pfeile übergehen Textmarker, allein und in einer Mehrfachauswahl.
func TestTextmarkerOhneKoepfe(t *testing.T) {
	s, _, p := sitzung(t)
	s.SetzeWerkzeug(WerkzeugTextmarker)
	zieh(s, f(20, 80), f(90, 80))
	m := letztes(t, s).Kennung()
	s.SetzeWerkzeug(WerkzeugAuswahl)
	s.waehleNur(m)
	if s.SetzeKopf(false, document.KopfPunkt) {
		t.Error("Kopf am gewählten Textmarker gesetzt")
	}
	if s.stilVon(WerkzeugTextmarker).KopfEnde == document.KopfPunkt || s.stilVon(WerkzeugPfeil).KopfEnde == document.KopfPunkt {
		t.Error("Kopf am gewählten Textmarker landet im Stil eines Werkzeugs")
	}
	s.waehleNur(p)
	s.SchalteGewaehlt(m)
	if s.AnzahlGewaehlt() != 2 {
		t.Fatalf("gewählt %d", s.AnzahlGewaehlt())
	}
	if !s.SetzeKopf(false, document.KopfPunkt) {
		t.Fatal("Kopf am Pfeil der Mehrfachauswahl nicht gesetzt")
	}
	if o, _ := s.D.Objekt(p); o.(document.Pfeil).Ende != document.KopfPunkt {
		t.Errorf("Pfeil %+v", o)
	}
	if o, _ := s.D.Objekt(m); o.(document.Pfeil).Ende != document.KopfKeine {
		t.Errorf("Textmarker bekam einen Kopf: %+v", o)
	}
	if _, ok := eigenschaftWert(mustObjekt(t, s, m), EigKopfEnde); ok {
		t.Error("Textmarker trägt die Eigenschaft Kopf")
	}
}

func mustObjekt(t *testing.T, s *Sitzung, id int) document.Objekt {
	t.Helper()
	o, ok := s.D.Objekt(id)
	if !ok {
		t.Fatalf("Objekt %d fehlt", id)
	}
	return o
}
