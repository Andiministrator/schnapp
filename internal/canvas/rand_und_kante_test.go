// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// keineAenderung prüft, dass ein Druck im Rand nichts bewirkt: kein Objekt, kein Verlaufsschritt, nicht geändert,
// nächste Nummer unverändert, nichts im Zug (F-150).
func keineAenderung(t *testing.T, s *Sitzung, was string, vorObjekte, vorSchritte int) {
	t.Helper()
	if n := len(s.D.Objekte()); n != vorObjekte {
		t.Errorf("%s: %d Objekte, vorher %d", was, n, vorObjekte)
	}
	if n := schritte(t, s); n != vorSchritte {
		t.Errorf("%s: %d Verlaufsschritte, vorher %d", was, n, vorSchritte)
	}
	if s.Geaendert() || s.Zieht() || s.TextLaeuft() || s.D.ZugLaeuft() {
		t.Errorf("%s: geändert %v, zieht %v, Text %v, Zug %v", was, s.Geaendert(), s.Zieht(), s.TextLaeuft(), s.D.ZugLaeuft())
	}
	if s.D.NaechsteNummer() != 1 {
		t.Errorf("%s: nächste Nummer %d (Soll 1)", was, s.D.NaechsteNummer())
	}
}

// F-150: Kein Zeichenwerkzeug legt etwas an, wenn der Druck im grauen Rand liegt (Bild 200×100 bei Versatz 24).
func TestRandKlickLegtNichtsAn(t *testing.T) {
	for _, w := range []struct {
		name string
		w    Werkzeug
	}{{"Nummer", WerkzeugNummer}, {"Text", WerkzeugText}, {"Pfeil", WerkzeugPfeil}, {"Kasten", WerkzeugKasten}} {
		for _, p := range []document.Punkt{
			document.Pt(5, 5), document.Pt(240, 140), document.Pt(100, 10), document.Pt(10, 80), document.Pt(23, 24),
			document.Pt(100, 125), document.Pt(225, 70),
		} {
			s, _, _ := sitzung(t)
			s.SetzeWerkzeug(w.w)
			vor, schr := len(s.D.Objekte()), schritte(t, s)
			s.Druecken(p, TastePrimaer)
			s.Ziehen(p.Plus(document.Pt(30, 20)))
			s.Loslassen(p.Plus(document.Pt(30, 20)))
			s.Tippe("geheim") // ein Textfeld im Rand dürfte keinen Text aufnehmen
			s.Enter(false)
			keineAenderung(t, s, w.name, vor, schr)
		}
	}
}

// Die Bildkante selbst gehört zum Bild.
func TestKanteGehoertZumBild(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugNummer)
	s.Druecken(f(0, 0), TastePrimaer)
	s.Loslassen(f(0, 0))
	if _, ok := letztes(t, s).(document.Nummer); !ok || s.Auswahl == 0 {
		t.Errorf("Nummer auf der Bildkante nicht angelegt: %T", letztes(t, s))
	}
}

// F-150: Zieht man über den Rand hinaus, bleibt der Endpunkt im Bild (Pfeil-Spitze, Kasten-Ecke, Nummer).
func TestZiehenUeberRandBegrenzt(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugPfeil)
	zieh(s, f(150, 80), document.Pt(300, 200))
	if pf := letztes(t, s).(document.Pfeil); !nah(pf.A, document.Pt(150, 80)) || !nah(pf.B, document.Pt(200, 100)) {
		t.Errorf("Pfeil %v → %v, Soll (150,80) → (200,100)", pf.A, pf.B)
	}
	s.SetzeWerkzeug(WerkzeugKasten)
	s.Abwaehlen() // sonst träfe der Druck den Griff des eben gezogenen Pfeils
	zieh(s, f(150, 80), document.Pt(-40, -50))
	if k := letztes(t, s).(document.Kasten); !nah(k.Bereich.Min, document.Pt(0, 0)) || !nah(k.Bereich.Max, document.Pt(150, 80)) {
		t.Errorf("Kasten %v, Soll (0,0)–(150,80)", k.Bereich)
	}
	// Mit Umschalt bleiben Pfeil und Quadrat im Bild.
	s.Umschalt = true
	s.SetzeWerkzeug(WerkzeugKasten)
	s.Abwaehlen()
	zieh(s, f(120, 60), document.Pt(500, 160))
	k := letztes(t, s).(document.Kasten)
	if k.Bereich.Max.X > 200+1e-9 || k.Bereich.Max.Y > 100+1e-9 {
		t.Errorf("Quadrat verlässt das Bild: %v", k.Bereich)
	}
	s.SetzeWerkzeug(WerkzeugPfeil)
	s.Abwaehlen()
	zieh(s, f(100, 50), document.Pt(500, 130))
	if pf := letztes(t, s).(document.Pfeil); pf.B.X > 200+1e-9 || pf.B.Y > 100+1e-9 || pf.B.Y < -1e-9 {
		t.Errorf("gerasteter Pfeil verlässt das Bild: B %v", pf.B)
	}
	s.Umschalt = false
	s.SetzeWerkzeug(WerkzeugNummer)
	s.Abwaehlen()
	zieh(s, f(100, 50), document.Pt(-90, 300))
	if n := letztes(t, s).(document.Nummer); !nah(n.Mitte, document.Pt(0, 100)) {
		t.Errorf("Nummer %v, Soll (0,100)", n.Mitte)
	}
}

// F-150: Auch mit Zuschnitt (Bild 150×80 sichtbar, Zoom 2, Versatz 10) zählt nur die sichtbare Ansicht.
func TestRandImZuschnitt(t *testing.T) {
	s, _, _ := sitzung(t)
	if err := s.D.SetzeZuschnitt(image.Rect(50, 20, 200, 100)); err != nil {
		t.Fatal(err)
	}
	s.Abb = Abbildung{Zoom: 2, Versatz: document.Pt(10, 10)}
	s.SetzeWerkzeug(WerkzeugNummer)
	vor, stand := len(s.D.Objekte()), s.D.Stand()
	for _, p := range []document.Punkt{document.Pt(5, 50), document.Pt(100, 5), document.Pt(311, 50), document.Pt(100, 171)} {
		s.Druecken(p, TastePrimaer)
		s.Loslassen(p)
	}
	if n := len(s.D.Objekte()); n != vor || s.D.Stand() != stand {
		t.Errorf("%d Objekte (vorher %d), Stand geändert %v", n, vor, s.D.Stand() != stand)
	}
	s.Druecken(document.Pt(40, 40), TastePrimaer)
	s.Loslassen(document.Pt(40, 40))
	if n := letztes(t, s).(document.Nummer); !nah(n.Mitte, document.Pt(65, 35)) {
		t.Errorf("Nummer im Zuschnitt %v, Soll (65,35)", n.Mitte)
	}
	// Zieht man darüber hinaus, endet die Spitze an der Kante des Zuschnitts.
	s.SetzeWerkzeug(WerkzeugPfeil)
	s.Abwaehlen()
	zieh(s, document.Pt(40, 40), document.Pt(900, 900))
	if pf := letztes(t, s).(document.Pfeil); !nah(pf.B, document.Pt(200, 100)) {
		t.Errorf("Spitze %v, Soll (200,100)", pf.B)
	}
}

// F-150: Die Griffe eines ausgewählten Objekts bleiben im Rand greifbar, auch mit Zeichenwerkzeug.
func TestGriffImRandBleibtGreifbar(t *testing.T) {
	s, _, _ := sitzung(t)
	id, err := s.D.Hinzufuegen(document.GeraderPfeil(document.Pt(100, 30), document.Pt(200, 30), rot))
	if err != nil {
		t.Fatal(err)
	}
	// Erst das Werkzeug, dann die Auswahl: ein Werkzeugwechsel hebt sie auf (ADR-0039 Nachtrag 3); gewählt mit einem
	// Zeichenwerkzeug ist etwa das gerade gezeichnete Objekt.
	s.SetzeWerkzeug(WerkzeugNummer)
	s.Auswahl = id
	vor := len(s.D.Objekte())
	s.Druecken(f(204, 30), TastePrimaer) // 4 px hinter dem Bildrand, am Endgriff
	if !s.Zieht() {
		t.Fatal("Griff im Rand nicht gegriffen")
	}
	s.Ziehen(f(190, 40))
	s.Loslassen(f(190, 40))
	pf := letztes(t, s).(document.Pfeil)
	if len(s.D.Objekte()) != vor || pf.ID != id || !nah(pf.B, document.Pt(190, 40)) {
		t.Errorf("%d Objekte (vorher %d), Pfeil %d B %v", len(s.D.Objekte()), vor, pf.ID, pf.B)
	}
}

// F-154 (1): Nummer über document.MaxNummer scheitert nicht still, sondern meldet den Grund; die Statusleiste kann
// vorher fragen (Voll).
func TestNummerUeberMaxMeldet(t *testing.T) {
	s, _, _ := sitzung(t)
	if _, err := s.D.Hinzufuegen(document.Nummer{Mitte: document.Pt(20, 80), Wert: document.MaxNummer, Radius: 14}); err != nil {
		t.Fatal(err)
	}
	s.SetzeWerkzeug(WerkzeugNummer)
	s.Abwaehlen()
	if s.Voll() != AblehnungNummer {
		t.Errorf("Voll %v, Soll AblehnungNummer", s.Voll())
	}
	vor := len(s.D.Objekte())
	s.Druecken(f(150, 80), TastePrimaer)
	s.Loslassen(f(150, 80))
	if len(s.D.Objekte()) != vor || s.Abgelehnt() != AblehnungNummer || s.Zieht() {
		t.Errorf("%d Objekte (vorher %d), abgelehnt %v", len(s.D.Objekte()), vor, s.Abgelehnt())
	}
	// Der nächste Druck setzt die Meldung zurück; ein anderes Werkzeug geht weiter.
	s.SetzeWerkzeug(WerkzeugKasten)
	if s.Voll() != AblehnungKeine {
		t.Errorf("Kasten: Voll %v", s.Voll())
	}
	zieh(s, f(60, 60), f(90, 90))
	if s.Abgelehnt() != AblehnungKeine || len(s.D.Objekte()) != vor+1 {
		t.Errorf("Kasten: abgelehnt %v, %d Objekte", s.Abgelehnt(), len(s.D.Objekte()))
	}
}

// F-154 (1): Beim Höchststand an Objekten meldet jedes Zeichenwerkzeug den Grund; ein Klick auf vorhandenen Text
// bearbeitet ihn trotzdem.
func TestObjekteUeberMaxMeldet(t *testing.T) {
	s, _, _ := sitzung(t)
	if _, err := s.D.Hinzufuegen(document.Text{Ort: document.Pt(100, 20), Inhalt: "x", Schriftgrad: 20, Farbe: Weiss}); err != nil {
		t.Fatal(err)
	}
	for len(s.D.Objekte()) < document.MaxObjekte {
		if _, err := s.D.Hinzufuegen(document.Nummer{Mitte: document.Pt(5, 5), Wert: 1, Radius: 2}); err != nil {
			t.Fatal(err)
		}
	}
	for _, w := range []Werkzeug{WerkzeugPfeil, WerkzeugKasten, WerkzeugText, WerkzeugNummer} {
		s.SetzeWerkzeug(w)
		s.Abwaehlen()
		s.Druecken(f(150, 80), TastePrimaer)
		s.Ziehen(f(190, 95))
		s.Loslassen(f(190, 95))
		if len(s.D.Objekte()) != document.MaxObjekte || s.Abgelehnt() != AblehnungObjekte || s.Zieht() || s.TextLaeuft() {
			t.Errorf("Werkzeug %v: %d Objekte, abgelehnt %v", w, len(s.D.Objekte()), s.Abgelehnt())
		}
	}
	s.SetzeWerkzeug(WerkzeugText)
	s.Druecken(f(105, 25), TastePrimaer)
	if !s.TextLaeuft() || s.Abgelehnt() != AblehnungKeine {
		t.Errorf("Klick auf vorhandenen Text bei vollem Dokument: Text läuft %v, abgelehnt %v", s.TextLaeuft(), s.Abgelehnt())
	}
}

// F-154 (2): Die mittlere Taste verschiebt nur die Ansicht, die Texteingabe läuft weiter (ADR-0021 Punkt 1).
func TestMitteWaehrendTexteingabe(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugText)
	s.Druecken(f(60, 70), TastePrimaer)
	s.Loslassen(f(60, 70))
	s.Tippe("ab")
	vor := s.Abb
	s.Druecken(f(10, 10), TasteMitte)
	if !s.TextLaeuft() || !s.Zieht() {
		t.Fatalf("Text läuft %v, zieht %v", s.TextLaeuft(), s.Zieht())
	}
	s.Ziehen(f(40, 30))
	s.Loslassen(f(40, 30))
	if !s.TextLaeuft() || s.Zieht() {
		t.Errorf("mittlere Taste beendet die Texteingabe: Text läuft %v, zieht %v", s.TextLaeuft(), s.Zieht())
	}
	if !nah(s.Abb.Versatz, vor.Versatz.Plus(document.Pt(30, 20))) || s.Abb.Zoom != vor.Zoom {
		t.Errorf("Ansicht %+v, Soll Versatz %v", s.Abb, vor.Versatz.Plus(document.Pt(30, 20)))
	}
	s.Tippe("c")
	s.Enter(false)
	if tx, ok := letztes(t, s).(document.Text); !ok || tx.Inhalt != "abc" {
		t.Errorf("Text %+v, Soll \"abc\"", letztes(t, s))
	}
	// Abbruch mit Cancel stellt die Ansicht wieder her, der Text bleibt.
	s.Druecken(f(60, 70), TastePrimaer)
	s.Tippe("z")
	davor := s.Abb
	s.Druecken(f(0, 0), TasteMitte)
	s.Ziehen(f(20, 20))
	s.Abbrechen()
	if s.Abb != davor || !s.TextLaeuft() || s.Zieht() {
		t.Errorf("nach Abbrechen: Ansicht %+v (Soll %+v), Text läuft %v, zieht %v", s.Abb, davor, s.TextLaeuft(), s.Zieht())
	}
}
