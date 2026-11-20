// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"errors"

	"github.com/Andiministrator/schnapp/document"
)

// Speichern im Hintergrund (ADR-0020 Nachtrag 6, F-143): Die Oberfläche fragt hier an und bekommt einen Abzug des
// Dokuments, den sie in einer Goroutine speichert; die Antwort meldet sie mit SpeichernEnde zurück, in ihrer
// Ereignisschleife. Bis dahin ruht das Bearbeiten (SpeichernLaeuft), damit der gespeicherte Stand der aktuelle bleibt
// und das Leeren des Verlaufs nach einer Schwärzung (ADR-0022 Punkt 3) für genau diesen Stand gilt. Die Sperre liegt
// nicht nur in der Oberfläche: Jeder Einstieg der Sitzung, der das Dokument ändert, fragt gesperrt und tut während des
// Speicherns nichts (F-280). Schließen-Anfragen und Signale während des Speicherns werden vorgemerkt und nach der
// Antwort entschieden.

// Anlass sagt, wozu gespeichert wird.
type Anlass int

const (
	AnlassJetzt      Anlass = iota // Strg+S oder Knopf Speichern
	AnlassSchliessen               // Schließen-Anfrage des Fensters
	AnlassSignal                   // SIGTERM, SIGINT, SIGHUP (ADR-0020 Nachtrag 2)
)

// Fehler von SpeichernJetzt.
var (
	// ErrNichtsZuSpeichern: Der Stand ist schon gespeichert (F-109); es wird nichts geschrieben.
	ErrNichtsZuSpeichern = errors.New("canvas: nichts zu speichern")
	// ErrSpeichernLaeuft: Ein Speichern läuft schon; es läuft höchstens eins zugleich.
	ErrSpeichernLaeuft = errors.New("canvas: Speichern läuft schon")
	// ErrZugLaeuft: Ein Ziehen läuft; Strg+S speichert erst danach.
	ErrZugLaeuft = errors.New("canvas: Ziehen läuft")
)

// SchlussWarten: Ein Speichern läuft (gerade begonnen oder schon vorher); die Schließen-Anfrage wird nach seiner
// Antwort entschieden (Ergebnis von SpeichernEnde).
const SchlussWarten Schluss = SchlussHinweis + 1

type hintergrund struct {
	laeuft     bool
	anlass     Anlass
	stand      uint64 // Stand des Abzugs, der gerade gespeichert wird
	schwaerzt  bool   // der Verlauf des Abzugs enthielt eine Schwärzung (ADR-0022 Punkt 3)
	schliessen bool   // eine Schließen-Anfrage wartet auf die Antwort
	signal     bool   // ein Signal kam; nach der Antwort wird geschlossen
}

// SpeichernLaeuft meldet, ob ein Speichern im Hintergrund läuft. Solange ruht das Bearbeiten.
func (s *Sitzung) SpeichernLaeuft() bool { return s.hg.laeuft }

// gesperrt meldet, ob Änderungen am Dokument gerade ruhen (F-280): Während des Speicherns nimmt die Sitzung keine an,
// gleich welcher Weg der Oberfläche sie auslöst (Zeiger, Knopf, Menü, Taste, mehrere Klicks in einem Bild). Jeder
// ändernde Einstieg fragt das zuerst, ebenso Werkzeug- und Farbwahl (beide bestätigen eine Eingabe oder färben um).
// Die Ansicht (Zoom, Einpassen) ändert das Dokument nicht; sie sperrt nur die Oberfläche.
func (s *Sitzung) gesperrt() bool { return s.hg.laeuft }

// SignalVorgemerkt meldet, ob ein Signal angenommen wurde: Das Fenster schließt, sobald das Speichern zurück ist.
func (s *Sitzung) SignalVorgemerkt() bool { return s.hg.signal }

// beginne legt den Abzug an und merkt das Speichern als laufend.
func (s *Sitzung) beginne(a Anlass) (*document.Dokument, error) {
	abzug, err := s.D.Abzug()
	if err != nil {
		return nil, err
	}
	s.hg.laeuft, s.hg.anlass, s.hg.stand, s.hg.schwaerzt = true, a, s.D.Stand(), s.D.VerlaufMitSchwaerzung()
	return abzug, nil
}

// SpeichernJetzt beginnt ein Speichern auf Strg+S: Ein getippter Text wird bestätigt (ADR-0021 Punkt 6). Ohne
// Änderung kommt ErrNichtsZuSpeichern, während eines Ziehens ErrZugLaeuft, während eines anderen Speicherns
// ErrSpeichernLaeuft — dann gibt es keinen Abzug.
func (s *Sitzung) SpeichernJetzt() (*document.Dokument, error) {
	if s.hg.laeuft {
		return nil, ErrSpeichernLaeuft
	}
	s.TextBestaetigen()
	if s.Zieht() {
		return nil, ErrZugLaeuft
	}
	if !s.Geaendert() {
		return nil, ErrNichtsZuSpeichern
	}
	return s.beginne(AnlassJetzt)
}

// SchliessenAnfrage beantwortet eine Schließen-Anfrage des Fensters in zwei Schritten (F-107, F-143): SchlussZu
// heißt schließen (nichts zu speichern oder Verwerfen bestätigt); SchlussWarten heißt offen halten und auf die Antwort
// des Speicherns warten — mit Abzug, wenn das Speichern jetzt beginnt, ohne, wenn schon eins läuft. Ein laufender Zug
// wird abgebrochen, ein getippter Text bestätigt.
func (s *Sitzung) SchliessenAnfrage() (Schluss, *document.Dokument) {
	if s.hg.laeuft {
		s.hg.schliessen = true
		return SchlussWarten, nil
	}
	s.TextBestaetigen()
	if s.zug != keinZug {
		s.Abbrechen()
	}
	if !s.Ungesichert() {
		return SchlussZu, nil
	}
	abzug, err := s.beginne(AnlassSchliessen)
	if err != nil { // nach dem Abbrechen läuft kein Zug mehr; nur zur Sicherheit: dann wie gescheitert
		s.verwerfen, s.verwerfenBei = true, s.D.Stand()
		return SchlussFehler, nil
	}
	return SchlussWarten, abzug
}

// SignalAnfrage nimmt ein Signal an (ADR-0020 Nachtrag 2): gespeichert wird jeder geänderte Stand, auch nach einem
// gescheiterten Speichern beim Schließen, danach wird in jedem Fall geschlossen. zu heißt: jetzt schließen (nichts zu
// speichern); sonst wartet das Fenster auf die Antwort — mit Abzug, wenn das Speichern jetzt beginnt.
func (s *Sitzung) SignalAnfrage() (zu bool, abzug *document.Dokument) {
	s.hg.signal = true
	if s.hg.laeuft {
		return false, nil
	}
	s.TextBestaetigen()
	if s.zug != keinZug {
		s.Abbrechen()
	}
	if !s.Geaendert() {
		return true, nil
	}
	abzug, err := s.beginne(AnlassSignal)
	if err != nil {
		return true, nil
	}
	return false, abzug
}

// Ende ist der Ausgang eines Speicherns im Hintergrund für die Oberfläche.
type Ende struct {
	// Schliessen: Das Fenster soll jetzt schließen (Schließen-Anfrage oder Signal).
	Schliessen bool
	// Schluss entscheidet eine wartende Schließen-Anfrage (SchlussZu, SchlussHinweis, SchlussFehler); ohne Anfrage
	// SchlussZu.
	Schluss Schluss
	// Erneut ist ein Abzug für ein weiteres Speichern: nach einem Signal, wenn das laufende Speichern (Strg+S oder
	// Schließen) gescheitert ist. Die Oberfläche speichert ihn wie den ersten und meldet wieder SpeichernEnde.
	Erneut *document.Dokument
	// SignalFehler: Das Speichern nach einem Signal ist gescheitert; der Stand gilt als verworfen, geschlossen wird
	// trotzdem.
	SignalFehler bool
}

// SpeichernEnde nimmt die Antwort des Speicherns an (warnung: Hinweis muss gelesen werden; err: gescheitert). Gelungen
// gilt der Stand des Abzugs als gespeichert, und ein Verlauf mit Schwärzung wird geleert (ADR-0022 Punkt 3). Danach
// werden vorgemerkte Schließen-Anfragen und Signale entschieden.
func (s *Sitzung) SpeichernEnde(warnung bool, err error) Ende {
	if !s.hg.laeuft {
		return Ende{Schluss: SchlussZu}
	}
	anlass := s.hg.anlass
	s.hg.laeuft = false
	if err == nil {
		s.gespeichert, s.verwerfen = s.hg.stand, false
		// ADR-0022 Punkt 3: Die Datei trägt jetzt die Schwärzung; kein Rückgängig darf die alten Pixel zurückholen. Geleert
		// wird ohne Bedingung an den Stand (F-280): Hätte sich das Dokument trotz der Sperre geändert, holte ein
		// Rückgängig über die neue Änderung hinweg sonst das Geschwärzte zurück. Den Verlauf zu verlieren ist der
		// kleinere Schaden. Während des Speicherns läuft kein Zug (gesperrt), VergissVerlauf gelingt.
		if s.hg.schwaerzt || s.D.VerlaufMitSchwaerzung() {
			_ = s.D.VergissVerlauf()
		}
	}
	if s.hg.signal {
		if anlass == AnlassSignal || err == nil || !s.Geaendert() {
			if err != nil {
				s.verwerfen, s.verwerfenBei = true, s.D.Stand()
			}
			return Ende{Schliessen: true, Schluss: SchlussZu, SignalFehler: err != nil}
		}
		// Ein Strg+S oder Schließen ist gescheitert, das Signal versucht es noch einmal (Ende.Erneut: ein neues Speichern).
		abzug, e := s.beginne(AnlassSignal)
		if e != nil {
			return Ende{Schliessen: true, Schluss: SchlussZu, SignalFehler: true}
		}
		return Ende{Erneut: abzug}
	}
	if !(anlass == AnlassSchliessen || s.hg.schliessen) {
		return Ende{Schluss: SchlussZu}
	}
	s.hg.schliessen = false
	switch {
	case err != nil:
		s.verwerfen, s.verwerfenBei = true, s.D.Stand()
		return Ende{Schluss: SchlussFehler}
	case warnung:
		return Ende{Schluss: SchlussHinweis}
	}
	return Ende{Schliessen: true, Schluss: SchlussZu}
}
