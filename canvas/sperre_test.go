// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"github.com/Andiministrator/schnapp/document"
)

// sperrDok: Bild 40 × 40, hellgrau, mit einer Schwärzung oben links im Verlauf.
func sperrDok(t *testing.T) (*Sitzung, *document.Dokument) {
	t.Helper()
	b := image.NewNRGBA(image.Rect(0, 0, 40, 40))
	for i := range b.Pix {
		b.Pix[i] = 200
	}
	d, err := document.Neu(b)
	if err != nil {
		t.Fatal(err)
	}
	s := NeueSitzung(d)
	s.SetzeFenster(image.Pt(100, 100))
	if err := d.Schwaerzen(image.Rect(0, 0, 10, 10), document.Schwaerzung{Farbe: color.NRGBA{A: 255}}); err != nil {
		t.Fatal(err)
	}
	return s, d
}

func geschwaerzt(d *document.Dokument) bool {
	return d.OriginalBild().NRGBAAt(3, 3) == color.NRGBA{A: 255}
}

// F-280: Während des Speicherns nimmt die Sitzung keine Änderung am Dokument an, gleich über welchen Einstieg — die
// Sperre liegt nicht nur in der Oberfläche. Jeder ändernde Einstieg wird mit Zuständen aufgerufen, in denen er ohne
// Sperre etwas täte (Auswahl, Text, Zuschnitt, Naht, Wiederholen-Schritt, alle Werkzeuge).
func TestSperreWaehrendSpeichern(t *testing.T) {
	s := verlaufSitzung(t)
	d := s.D
	k, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(10, 10), document.Pt(50, 50)),
		Strich: rot})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(document.Text{Ort: document.Pt(120, 20), Inhalt: "Hallo", Schriftgrad: 12,
		Farbe: color.NRGBA{A: 255}}); err != nil {
		t.Fatal(err)
	}
	if err := d.Schwaerzen(image.Rect(60, 60, 80, 80), document.Schwaerzung{Farbe: color.NRGBA{A: 255}}); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeAnsicht(image.Rect(2, 2, 198, 98), []document.Streifen{{Von: 90, Bis: 100}}); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(150, 60), document.Pt(170, 80))}); err != nil {
		t.Fatal(err)
	}
	if err := d.Rueckgaengig(); err != nil { // ein Wiederholen-Schritt liegt bereit
		t.Fatal(err)
	}
	s.Einpassen()
	s.Auswahl = k
	if !s.KannLoeschen() || !s.KannGanzesBild() || !s.KannZurueckholen() || !d.KannWiederholen() {
		t.Fatal("Vorbedingung: Löschen, Ganzes Bild, Zurückholen und Wiederholen müssten gehen")
	}

	if _, err := s.SpeichernJetzt(); err != nil {
		t.Fatal(err)
	}
	stand, objekte, stil, werkzeug := d.Stand(), len(d.Objekte()), s.Stil, s.Werkzeug
	anfang := werkzeug
	bild := s.BildAenderung()
	pruefe := func(was string) {
		t.Helper()
		if d.Stand() != stand || len(d.Objekte()) != objekte || s.Stil != stil || s.Werkzeug != werkzeug ||
			s.BildAenderung() != bild || s.Zieht() || s.TextLaeuft() || d.ZugLaeuft() || !d.KannWiederholen() {
			t.Errorf("%s wirkt während des Speicherns: Stand %d/%d, Objekte %d/%d, Werkzeug %v/%v, zieht %v, Text %v, Zug %v",
				was, d.Stand(), stand, len(d.Objekte()), objekte, s.Werkzeug, werkzeug, s.Zieht(), s.TextLaeuft(), d.ZugLaeuft())
		}
	}
	if s.KannLoeschen() || s.KannGanzesBild() || s.KannZurueckholen() || s.KannTextBearbeiten() {
		t.Error("Menüeinträge während des Speicherns bedienbar")
	}

	text := fv(s, 125, 22)
	for _, w := range []Werkzeug{WerkzeugAuswahl, WerkzeugPfeil, WerkzeugKasten, WerkzeugText, WerkzeugNummer,
		WerkzeugSchwaerzen, WerkzeugVerpixeln, WerkzeugZuschneiden, WerkzeugStauchen} {
		s.Werkzeug, werkzeug = w, w // SetzeWerkzeug ist gesperrt; das Feld direkt setzen, um jeden Zug zu versuchen
		for _, p := range []document.Punkt{fv(s, 30, 30), fv(s, 5, 5), text} {
			s.Druecken(p, TastePrimaer)
			s.Ziehen(p.Plus(document.Pt(30, 20)))
			s.Loslassen(p.Plus(document.Pt(30, 20)))
			s.Druecken(p, TasteMitte)
			s.Loslassen(p)
			s.Doppelklick(p)
			s.Loslassen(p)
		}
		s.Doppelklick(fv(s, 88, 50)) // an der Naht
		pruefe(fmt.Sprintf("Zug mit Werkzeug %d", w))
	}
	s.Werkzeug, werkzeug = anfang, anfang
	s.Auswahl = k
	for name, f := range map[string]func(){
		"Rückgängig":        func() { s.Rueckgaengig() },
		"Wiederholen":       func() { s.Wiederholen() },
		"Löschen":           func() { s.Loeschen() },
		"Farbe":             func() { s.SetzeFarbe(color.NRGBA{B: 255, A: 255}) },
		"Nächste Farbe":     func() { s.NaechsteFarbe() },
		"Werkzeug":          func() { s.SetzeWerkzeug(WerkzeugKasten) },
		"Ganzes Bild":       func() { s.HebeZuschnittAuf() },
		"Zurückholen":       func() { s.HoleGestauchtesZurueck() },
		"Naht":              func() { s.EntferneNaht(fv(s, 88, 50)) },
		"Text bearbeiten":   func() { s.TextBearbeiten() },
		"Tippen":            func() { s.Tippe("x") },
		"Zeilenumbruch":     func() { s.TextZeilenumbruch() },
		"Rücktaste":         func() { s.TextZurueck() },
		"Text bestätigen":   func() { s.TextBestaetigen() },
		"Text verwerfen":    func() { s.TextAbbrechen() },
		"Enter":             func() { s.Enter(false) },
		"Umschalt+Enter":    func() { s.Enter(true) },
		"Esc":               func() { s.Esc() },
		"Abbrechen":         func() { s.Abbrechen() },
		"Doppelklick Text":  func() { s.Doppelklick(text) },
		"Druck auf Auswahl": func() { s.Druecken(fv(s, 10, 10), TastePrimaer); s.Ziehen(fv(s, 40, 40)); s.Loslassen(fv(s, 40, 40)) },
	} {
		f()
		pruefe(name)
		s.Auswahl = k
	}
	if s.Auswahl != k {
		t.Error("Auswahl verloren")
	}

	// Danach geht alles wieder: Die Schwärzung im gespeicherten Verlauf leert ihn, Löschen wirkt.
	s.SpeichernEnde(false, nil)
	if d.KannRueckgaengig() || d.KannWiederholen() || s.Geaendert() {
		t.Error("nach dem Speichern: Verlauf nicht leer oder geändert")
	}
	if !s.Loeschen() || len(d.Objekte()) != objekte-1 {
		t.Error("Löschen nach dem Speichern wirkt nicht")
	}
}

// F-280, ADR-0022 Punkt 3: Ändert sich das Dokument während des Speicherns doch (hier am Dokument vorbei, wie es ein
// Loch in der Sperre täte), leert ein gelungenes Speichern mit Schwärzung den Verlauf trotzdem: Rückgängig holt das
// Geschwärzte nicht zurück. Der neue Stand gilt als ungespeichert.
func TestSpeichernEndeLeertAuchBeiAnderemStand(t *testing.T) {
	s, d := sperrDok(t)
	a, err := s.SpeichernJetzt()
	if err != nil || !geschwaerzt(a) {
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(20, 20), document.Pt(30, 30)),
		Strich: document.Strich{Breite: 1}}); err != nil {
		t.Fatal(err)
	}
	s.SpeichernEnde(false, nil)
	s.Rueckgaengig()
	s.Rueckgaengig()
	if !geschwaerzt(d) || d.KannRueckgaengig() {
		t.Error("Rückgängig holt nach dem Speichern das Geschwärzte zurück")
	}
	if !s.Geaendert() {
		t.Error("die Änderung während des Speicherns gilt als gespeichert")
	}
}

// ADR-0022 Punkt 3 über alle Wege des Speicherns im Hintergrund: Jeder Weg, der nach einem gelungenen Speichern das
// Fenster offen lässt, hat den Verlauf geleert; der Abzug ist geschwärzt und ohne Verlauf.
func TestKeinRueckgaengigNachSpeichernMitSchwaerzung(t *testing.T) {
	wege := map[string]func(s *Sitzung) *document.Dokument{
		"Strg+S": func(s *Sitzung) *document.Dokument {
			a, err := s.SpeichernJetzt()
			if err != nil {
				t.Fatal(err)
			}
			s.SpeichernEnde(false, nil)
			return a
		},
		"Strg+S mit Hinweis": func(s *Sitzung) *document.Dokument {
			a, _ := s.SpeichernJetzt()
			s.SpeichernEnde(true, nil)
			return a
		},
		"Schließen mit Hinweis": func(s *Sitzung) *document.Dokument {
			_, a := s.SchliessenAnfrage()
			if e := s.SpeichernEnde(true, nil); e.Schliessen {
				t.Fatal("Hinweis hält nicht offen")
			}
			return a
		},
		"Schließen während Strg+S, Hinweis": func(s *Sitzung) *document.Dokument {
			a, _ := s.SpeichernJetzt()
			s.SchliessenAnfrage()
			s.SpeichernEnde(true, nil)
			return a
		},
		"Strg+S gescheitert, dann gelungen": func(s *Sitzung) *document.Dokument {
			s.SpeichernJetzt()
			s.SpeichernEnde(false, errPlatte)
			if !s.D.KannRueckgaengig() {
				t.Fatal("gescheitert: Verlauf weg")
			}
			a, _ := s.SpeichernJetzt()
			s.SpeichernEnde(false, nil)
			return a
		},
		"Schließen gescheitert, dann Strg+S": func(s *Sitzung) *document.Dokument {
			s.SchliessenAnfrage()
			s.SpeichernEnde(false, errPlatte)
			a, err := s.SpeichernJetzt()
			if err != nil {
				t.Fatal(err)
			}
			s.SpeichernEnde(false, nil)
			return a
		},
		"Signal nach gescheitertem Strg+S": func(s *Sitzung) *document.Dokument {
			s.SpeichernJetzt()
			s.SignalAnfrage()
			e := s.SpeichernEnde(false, errPlatte)
			if e.Erneut == nil {
				t.Fatal("kein weiterer Versuch")
			}
			s.SpeichernEnde(false, nil)
			return e.Erneut
		},
		"Schließen mit Abzug aus der Speicherfunktion": func(s *Sitzung) *document.Dokument {
			var a *document.Dokument
			schliesse(s, func() (bool, error) { a, _ = s.D.Abzug(); return true, nil })
			return a
		},
	}
	for name, weg := range wege {
		s, d := sperrDok(t)
		a := weg(s)
		if a == nil {
			t.Errorf("%s: kein Abzug", name)
			continue
		}
		if !geschwaerzt(a) || !a.Geschwaerzt() || a.SchwaerzStand() != d.SchwaerzStand() || a.KannRueckgaengig() {
			t.Errorf("%s: Abzug nicht geschwärzt oder mit Verlauf", name)
		}
		if a.Rueckgaengig() == nil || !geschwaerzt(a) {
			t.Errorf("%s: Rückgängig am Abzug wirkt", name)
		}
		if s.Rueckgaengig() || s.Wiederholen() || d.KannRueckgaengig() || d.KannWiederholen() || d.VerlaufMitSchwaerzung() || !geschwaerzt(d) {
			t.Errorf("%s: Rückgängig nach dem Speichern möglich oder Schwärzung weg", name)
		}
		if s.Geaendert() || s.SpeichernLaeuft() {
			t.Errorf("%s: geändert %v, läuft %v", name, s.Geaendert(), s.SpeichernLaeuft())
		}
	}
}

// Schwärzung, Rückgängig, Wiederholen, Speichern: Der Verlauf ist danach leer, die Schwärzung bleibt.
func TestSpeichernNachWiederholterSchwaerzung(t *testing.T) {
	s, d := sperrDok(t)
	if !s.Rueckgaengig() || !s.Wiederholen() {
		t.Fatal("Vorbedingung")
	}
	if _, err := s.SpeichernJetzt(); err != nil {
		t.Fatal(err)
	}
	s.SpeichernEnde(false, nil)
	if d.KannRueckgaengig() || !geschwaerzt(d) {
		t.Error("Rückgängig nach Speichern")
	}
}

// Ein Zug mit der mittleren Taste (Ansicht verschieben) über eine Schließen-Anfrage hinweg: Die Anfrage bricht ihn ab,
// das Loslassen danach hinterlässt nichts, und der nächste Zug legt wieder an.
func TestAnsichtszugUeberSchliessen(t *testing.T) {
	s, _, _ := sitzung(t)
	verschiebe(t, s, 10)
	s.Druecken(f(5, 5), TasteMitte)
	s.Ziehen(f(30, 30))
	if z, a := s.SchliessenAnfrage(); z != SchlussWarten || a == nil || s.Zieht() {
		t.Fatalf("Schließen im Zug: %v, Abzug %v, zieht %v", z, a != nil, s.Zieht())
	}
	s.Loslassen(f(30, 30)) // fällt in die Sperre
	s.SpeichernEnde(true, nil)
	if s.Zieht() {
		t.Fatal("nach dem Speichern hängt ein Zug")
	}
	n := len(s.D.Objekte())
	s.SetzeWerkzeug(WerkzeugKasten)
	s.Druecken(f(60, 60), TastePrimaer)
	s.Ziehen(f(90, 90))
	s.Loslassen(f(90, 90))
	if len(s.D.Objekte()) != n+1 {
		t.Errorf("Zug nach dem Speichern legt nichts an: %d Objekte, Soll %d", len(s.D.Objekte()), n+1)
	}
}
