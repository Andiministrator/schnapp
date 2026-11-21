// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"math"
	"slices"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Fensterwerte der Bedienung (Pixel des Fensterpuffers, unabhängig vom Zoom).
const (
	// Toleranz für Treffer auf Objekte und Griffe.
	Toleranz = 6.0
	// Rand beim Einpassen.
	Rand = 24.0
	// Ab dieser Strecke ist ein Druck ein Ziehen (darunter ein Klick).
	Ziehschwelle = 3.0
)

// Taste unterscheidet die Maustaste beim Drücken.
type Taste int

const (
	TastePrimaer Taste = iota // links: auswählen, verschieben, Griffe; auf leerer Fläche verschieben der Ansicht
	TasteMitte                // Mitte: Ansicht verschieben
)

type zugArt int

const (
	keinZug zugArt = iota
	zugObjekt
	zugGriff
	zugAnsicht
	zugNeu  // neues Objekt mit einem Zeichenwerkzeug (ADR-0021)
	zugText // Texteingabe: läuft über Druck und Loslassen hinaus bis Bestätigen oder Abbrechen
)

// Sitzung ist der Zustand des Editors ohne Gio (ADR-0020 Punkt 2): Dokument, Abbildung, Auswahl und laufender Zug.
// Alle Punkte kommen in Fensterkoordinaten. Änderungen am Dokument laufen über dessen Züge und Befehle, damit
// Rückgängig sie als je eine Änderung kennt.
type Sitzung struct {
	D   *document.Dokument
	Abb Abbildung
	// Auswahl ist die Kennung des ausgewählten Objekts, 0 = keins.
	Auswahl int
	// Werkzeug ist das aktive Werkzeug; Wechsel über SetzeWerkzeug (bestätigt eine Texteingabe).
	Werkzeug Werkzeug
	// Stil gilt für neue Objekte; Farbe über SetzeFarbe (färbt auch die Auswahl um).
	Stil Stil
	// Umschalt: Die Umschalttaste ist gedrückt (die Oberfläche setzt es je Zeigerereignis): Pfeil rastet auf 45°,
	// Kasten wird ein Quadrat.
	Umschalt bool

	gespeichert uint64 // Verlaufsstand beim Öffnen oder letzten Speichern (F-109)
	// verwerfen: Speichern beim Schließen ist bei diesem Stand gescheitert; eine weitere Schließen-Anfrage ohne neue
	// Änderung verwirft ihn (F-107).
	verwerfen    bool
	verwerfenBei uint64

	fenster    image.Point
	eingepasst bool // Abbildung folgt der Fenstergröße, bis der Nutzer zoomt oder verschiebt
	zug        zugArt
	griff      GriffArt
	start      document.Punkt  // Fensterpunkt beim Drücken
	startObj   document.Objekt // Objekt beim Drücken (ZieheGriff/Verschiebe wirken immer darauf)
	startAbb   Abbildung
	bewegt     bool
	neuArt     Werkzeug // Werkzeug beim Beginn eines zugNeu
	neuID      int      // Kennung des neuen Objekts im Zug, 0 = noch nicht angelegt (unter der Ziehschwelle)
	textID     int      // Kennung des Texts in der Eingabe
	textNeu    bool     // die Eingabe gilt einem neuen Text (Abbrechen entfernt ihn)
	pan        bool     // mittlere Taste verschiebt die Ansicht, während die Texteingabe weiterläuft (F-154)
	abgelehnt  Ablehnung
	// Schwärzen (ADR-0022): Rahmen in Ansichtspunkten, solange gezogen wird.
	rahmen           bool
	rahmenA, rahmenB document.Punkt
	ersatzSchwarz    bool   // letzter Verpixeln-Rahmen war zu schmal und wurde geschwärzt
	rahmenZuKlein    bool   // letzter Schwärzungsrahmen lag unter der Mindestgröße und tat nichts (F-227)
	bildAenderung    uint64 // zählt Änderungen am Ansichtsbild: Schwärzungen, Zuschnitt, Streifen
	ansichtMeldung   AnsichtMeldung
	// hg ist das Speichern im Hintergrund (hintergrund.go, ADR-0020 Nachtrag 6).
	hg hintergrund
}

// NeueSitzung beginnt mit eingepasster Ansicht, ohne Auswahl, mit dem Werkzeug Auswählen und dem Standardstil.
func NeueSitzung(d *document.Dokument) *Sitzung {
	return &Sitzung{D: d, Abb: Abbildung{Zoom: 1}, eingepasst: true, gespeichert: d.Stand(), Stil: StandardStil()}
}

// Geaendert meldet, ob der Verlaufsstand vom gespeicherten abweicht (F-109): Rückgängig bis zum gespeicherten Stand
// ist wieder „ungeändert“; ein laufender Zug zählt erst, wenn er losgelassen ist.
func (s *Sitzung) Geaendert() bool { return s.D.Stand() != s.gespeichert }

// SetzeFenster meldet die Fenstergröße. Solange der Nutzer nicht selbst gezoomt oder verschoben hat, wird neu
// eingepasst (der erste Frame kommt unter Wayland oft in falscher Größe).
func (s *Sitzung) SetzeFenster(g image.Point) {
	s.fenster = g
	if s.eingepasst && !s.Zieht() {
		s.Abb = Einpassen(s.D.Groesse(), g, Rand)
	}
}

// Einpassen zeigt die ganze Ansicht im Fenster (Strg+0).
func (s *Sitzung) Einpassen() {
	if s.Zieht() {
		return // während eines Zugs bliebe der Startpunkt in der alten Abbildung (Kritik A-6)
	}
	s.eingepasst = true
	s.Abb = Einpassen(s.D.Groesse(), s.fenster, Rand)
}

// EinsZuEins zeigt die Ansicht 1:1 um die Fenstermitte (Strg+1).
func (s *Sitzung) EinsZuEins() {
	if s.Zieht() {
		return
	}
	s.eingepasst = false
	s.Abb = s.Abb.Eins(document.Pt(float64(s.fenster.X)/2, float64(s.fenster.Y)/2))
}

// Scrollen zoomt zum Zeiger: dy < 0 vergrößert (Rad nach oben). Ein Rastschritt (dy = ±1) ändert um 10 %.
func (s *Sitzung) Scrollen(p document.Punkt, dy float64) {
	if dy == 0 || math.IsNaN(dy) || math.IsInf(dy, 0) || s.Zieht() {
		return
	}
	s.eingepasst = false
	s.Abb = s.Abb.ZoomUm(p, math.Pow(1.1, -math.Max(-10, math.Min(10, dy))))
}

// original rechnet einen Fensterpunkt ins Original um.
func (s *Sitzung) original(p document.Punkt) document.Punkt {
	return s.D.ZuOriginal(s.Abb.ZuAnsicht(p))
}

// toleranz ist die Fenster-Toleranz in Pixeln des Originals.
func (s *Sitzung) toleranz() float64 { return Toleranz / s.Abb.Zoom }

// Druecken beginnt einen Zug: an einem Griff des ausgewählten Objekts, mit einem Zeichenwerkzeug ein neues Objekt
// (ADR-0021), sonst an einem Objekt (wählt es aus) oder auf leerer Fläche bzw. mit der mittleren Taste an der Ansicht.
// Eine laufende Texteingabe wird vorher bestätigt. Während eines Zugs ist ein weiterer Druck wirkungslos, ebenso
// während des Speicherns (F-280).
func (s *Sitzung) Druecken(p document.Punkt, t Taste) {
	if s.gesperrt() {
		return
	}
	s.abgelehnt, s.ersatzSchwarz, s.rahmenZuKlein, s.ansichtMeldung = AblehnungKeine, false, false, AnsichtKeine
	if s.pan {
		return
	}
	if t == TasteMitte && s.zug == zugText {
		// Die mittlere Taste verschiebt nur die Ansicht, die Texteingabe läuft weiter (ADR-0021 Punkt 1, F-154).
		s.pan, s.start, s.bewegt, s.startAbb = true, p, false, s.Abb
		return
	}
	s.TextBestaetigen()
	if s.zug != keinZug || s.D.ZugLaeuft() {
		return
	}
	s.start, s.bewegt, s.startAbb = p, false, s.Abb
	if t == TasteMitte {
		s.zug = zugAnsicht
		return
	}
	o := s.sichtbareObjekte() // Verborgenes ist nicht zu greifen (ADR-0023 Punkt 7)
	q := s.original(p)
	if sel, ok := s.D.Objekt(s.Auswahl); ok && !ohneGriffe(s.Werkzeug) {
		if art, ok := GriffTreffer(sel, q, s.toleranz()); ok && s.D.BeginneZug(sel.Kennung()) == nil {
			s.zug, s.griff, s.startObj = zugGriff, art, sel
			return
		}
	}
	if s.Werkzeug != WerkzeugAuswahl {
		// Ein neues Objekt beginnt nur im sichtbaren Bild (F-150); daneben passiert nichts. Griffe des ausgewählten
		// Objekts sind oben schon vorgegangen und bleiben auch im Rand greifbar.
		if s.imBild(p) {
			s.beginneNeu(q)
		}
		return
	}
	if id, ok := Treffer(o, q, s.toleranz()); ok {
		s.Auswahl = id
		obj, _ := s.D.Objekt(id)
		if s.D.BeginneZug(id) == nil {
			s.zug, s.startObj = zugObjekt, obj
		}
		return
	}
	s.Auswahl = 0
	s.zug = zugAnsicht
}

// Ziehen führt den Zug zum Fensterpunkt p.
func (s *Sitzung) Ziehen(p document.Punkt) {
	if s.pan {
		if s.bewegt || p.Minus(s.start).Laenge() >= Ziehschwelle {
			s.bewegt, s.eingepasst = true, false
			s.Abb = s.startAbb.Verschoben(p.Minus(s.start))
		}
		return
	}
	if s.zug == keinZug || s.zug == zugText || s.gesperrt() {
		return
	}
	if !s.bewegt && p.Minus(s.start).Laenge() < Ziehschwelle {
		return
	}
	s.bewegt = true
	switch s.zug {
	case zugAnsicht:
		s.eingepasst = false
		s.Abb = s.startAbb.Verschoben(p.Minus(s.start))
	case zugObjekt:
		d := s.original(p).Minus(s.original(s.start))
		_ = s.D.ZugSetze(Verschiebe(s.startObj, s.begrenzeVerschieben(s.startObj, d)))
	case zugGriff:
		_ = s.D.ZugSetze(s.begrenzeGriff(s.startObj, s.griff, s.original(p)))
	case zugNeu:
		s.zieheNeu(p)
	}
}

// Loslassen beendet den Zug; ein Objektzug landet als eine Änderung im Verlauf (ohne Bewegung als keine), ein neues
// Objekt als eine Änderung, wenn es groß genug ist. Eine Texteingabe läuft weiter.
func (s *Sitzung) Loslassen(p document.Punkt) {
	if s.pan {
		s.Ziehen(p)
		s.pan = false
		return
	}
	if s.zug == zugText || s.gesperrt() {
		return
	}
	s.Ziehen(p)
	vor := s.D.Stand()
	switch s.zug {
	case zugObjekt, zugGriff:
		if s.bewegt {
			_ = s.D.BeendeZug() // ein unveränderter Zug hinterlässt keinen neuen Stand
		} else {
			_ = s.D.BrecheZugAb()
		}
	case zugNeu:
		s.beendeNeu()
	}
	s.merkeAenderung(vor)
	s.zug = keinZug
}

// merkeAenderung hebt die Verwerfen-Marke auf, wenn sich der Stand seit vor geändert hat: Jede neue Bearbeitung nach
// einem gescheiterten Speichern verlangt einen neuen Speicherversuch, auch wenn Rückgängig danach denselben Stand
// wieder herstellt (F-154).
func (s *Sitzung) merkeAenderung(vor uint64) {
	if s.D.Stand() != vor {
		s.verwerfen = false
	}
}

// imBild meldet, ob der Fensterpunkt p im sichtbaren Bild liegt (Ansicht nach Zuschnitt und Stauchen, Rand
// eingeschlossen).
func (s *Sitzung) imBild(p document.Punkt) bool {
	a, g := s.Abb.ZuAnsicht(p), s.D.Groesse()
	return a.X >= 0 && a.Y >= 0 && a.X <= float64(g.X) && a.Y <= float64(g.Y)
}

// klemmeAnsicht begrenzt einen Ansichtspunkt auf das sichtbare Bild.
func (s *Sitzung) klemmeAnsicht(a document.Punkt) document.Punkt {
	g := s.D.Groesse()
	return document.Pt(math.Max(0, math.Min(float64(g.X), a.X)), math.Max(0, math.Min(float64(g.Y), a.Y)))
}

// Abbrechen stellt den Stand vor dem Zug wieder her (Esc während des Ziehens, Zeiger-Cancel). Eine Texteingabe bleibt
// (sie verwirft nur TextAbbrechen).
func (s *Sitzung) Abbrechen() {
	if s.pan {
		s.Abb, s.pan = s.startAbb, false
		return
	}
	switch s.zug {
	case zugText:
		return
	case zugObjekt, zugGriff:
		_ = s.D.BrecheZugAb()
	case zugNeu:
		if s.neuID != 0 {
			_ = s.D.BrecheZugAb()
		}
		s.rahmen = false
	case zugAnsicht:
		s.Abb = s.startAbb
	}
	s.zug = keinZug
}

// Zieht meldet einen laufenden Zug mit der Maus (nicht die Texteingabe, dafür TextLaeuft).
func (s *Sitzung) Zieht() bool { return s.pan || (s.zug != keinZug && s.zug != zugText) }

// Rueckgaengig nimmt die letzte Änderung zurück (nicht während eines Zugs, F-18). Verschwindet dabei das
// ausgewählte Objekt, endet die Auswahl.
func (s *Sitzung) Rueckgaengig() bool { return s.verlauf(s.D.Rueckgaengig) }

// Wiederholen stellt die zuletzt zurückgenommene Änderung wieder her.
func (s *Sitzung) Wiederholen() bool { return s.verlauf(s.D.Wiederholen) }

func (s *Sitzung) verlauf(f func() error) bool {
	if s.zug != keinZug || s.gesperrt() {
		return false
	}
	z, g, st := s.D.Zuschnitt(), s.D.Groesse(), s.D.Streifen()
	verborgen := s.VerborgeneObjekte()
	if f() != nil {
		return false
	}
	if _, ok := s.D.Objekt(s.Auswahl); !ok {
		s.Auswahl = 0
	}
	if s.D.Zuschnitt() != z || s.D.Groesse() != g || !slices.Equal(s.D.Streifen(), st) {
		// Zuschnitt oder Streifen zurückgenommen bzw. wiederholt: neu einpassen (ADR-0023 Punkt 5), wie nach dem
		// Zuschneiden melden, wenn mehr verborgen ist (F-184).
		s.ansichtGeaendert(verborgen)
	}
	return true
}

// Loeschen entfernt das ausgewählte Objekt.
func (s *Sitzung) Loeschen() bool {
	if s.zug != keinZug || s.Auswahl == 0 || s.gesperrt() {
		return false
	}
	vor := s.D.Stand()
	if s.D.Entfernen(s.Auswahl) != nil {
		return false
	}
	s.merkeAenderung(vor)
	s.Auswahl = 0
	return true
}

// Abwaehlen hebt die Auswahl auf (Esc ohne Zug).
func (s *Sitzung) Abwaehlen() { s.Auswahl = 0 }

// Gespeichert meldet, dass der aktuelle Stand gespeichert ist. Hält der Verlauf eine Schwärzung, wird er geleert:
// Nach dem Speichern holt kein Rückgängig ungeschwärzte Pixel zurück, die ein weiteres Speichern auf die Platte
// brächte (ADR-0022 Punkt 3).
func (s *Sitzung) Gespeichert() {
	s.gespeichert, s.verwerfen = s.D.Stand(), false
	if s.D.VerlaufMitSchwaerzung() {
		_ = s.D.VergissVerlauf() // während eines Zugs nicht möglich; gespeichert wird ohne Zug
	}
}

// Ungesichert meldet, ob beim Schließen noch gespeichert werden muss: geändert und nicht schon nach einem gescheiterten
// Speichern zum Verwerfen freigegeben (SchliessenAnfrage, SpeichernEnde).
func (s *Sitzung) Ungesichert() bool {
	return s.Geaendert() && !(s.verwerfen && s.verwerfenBei == s.D.Stand())
}

// Schluss ist der Ausgang einer Schließen-Anfrage.
type Schluss int

const (
	// SchlussZu: Das Fenster darf schließen (nichts zu speichern, gespeichert oder Verwerfen bestätigt).
	SchlussZu Schluss = iota
	// SchlussFehler: Speichern ist gescheitert, das Fenster bleibt offen und zeigt den Fehler. Eine weitere Anfrage
	// ohne neue Änderung schließt und verwirft den Stand.
	SchlussFehler
	// SchlussHinweis: gespeichert, aber mit einer Warnung (z. B. auf eine neue Aufnahme ausgewichen); das Fenster bleibt
	// offen, damit sie gelesen wird. Die nächste Anfrage schließt.
	SchlussHinweis
)
