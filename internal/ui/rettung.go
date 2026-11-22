// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Rettung sichert ungesicherte Änderungen ohne Rendern gegen Absturz, SIGKILL und Stromausfall (ADR-0029). Der Editor
// ruft sie nur aus der Ereignisschleife; nil heißt: keine Rettungskopie.
type Rettung interface {
	// Wiederherstellen wendet beim Öffnen eine passende Rettungskopie an (nach dem Anlegen der Sitzung, damit das
	// Dokument als geändert gilt und Rückgängig auf den gespeicherten Stand führt). Zurück kommt die Meldung für die
	// Statusleiste (leer: nichts zu sagen).
	Wiederherstellen(d *document.Dokument) (meldung string, fehler bool)
	// Sichere bildet die Rettungskopie (Angaben, keine Pixel) und schreibt sie im Hintergrund.
	Sichere(d *document.Dokument)
	// Verwirf löscht die Rettungskopie: Der Stand ist gespeichert.
	Verwirf()
	// Ende wartet auf begonnene Schreibaufträge.
	Ende()
}

// Bündelung (ADR-0029 Punkt 4): frühestens rettungRuhe nach der letzten Änderung, höchstens rettungTakt nach der
// ersten ungesicherten.
const (
	rettungRuhe = 2 * time.Second
	rettungTakt = 10 * time.Second
)

// rettungUhr ist die Uhr der Bündelung; Naht für Tests.
var rettungUhr = time.Now

// rettungZustand hält, was die Bündelung braucht.
type rettungZustand struct {
	gesichert  uint64    // Dokumentstand der zuletzt geschriebenen (oder angewendeten) Kopie, bzw. der gespeicherte
	beobachtet uint64    // zuletzt gesehener Stand
	letzte     time.Time // Zeit der letzten Änderung
	erste      time.Time // erste ungesicherte Änderung seit der letzten Kopie; Null: keine
	geweckt    time.Time // Zeitpunkt, zu dem schon ein Wecken bestellt ist
}

// rettungStart wendet beim Öffnen eine Rettungskopie an und merkt den Stand.
func (e *editor) rettungStart() {
	if e.rettung == nil {
		return
	}
	if text, fehler := e.rettung.Wiederherstellen(e.s.D); text != "" {
		e.melde(text, fehler)
	}
	st := e.s.D.Stand()
	e.rett = rettungZustand{gesichert: st, beobachtet: st}
}

// rettungPruefen läuft nach jedem Ereignis: Ist der Stand seit der letzten Kopie geändert und die Bündelung um,
// schreibt es eine neue Kopie; sonst bestellt es ein Wecken zum fälligen Zeitpunkt. Während eines Speicherns und nach
// der Freigabe zum Schließen ruht es. Ein laufender Zug (Ziehen, Texteingabe) hält es nicht auf, sonst bliebe eine
// fertige Änderung so lange ungesichert, wie getippt wird (Verify 7, V7-2). Die Kopie trägt aber nur den Stand Stand()
// ohne das Objekt im Zug (document.ObjekteOhneZug): Inhalt der Kopie und gesichert passen so immer zusammen; ein mit
// Esc verworfener Text kommt nach einem Absturz nicht wieder und liegt nie auf der Platte, ein übernommener ändert den
// Stand und wird nach der Bündelung gesichert (F-409). Steht das Dokument wieder auf dem gespeicherten Stand, verwirft
// es die Kopie.
func (e *editor) rettungPruefen() {
	if e.rettung == nil || e.s.SpeichernLaeuft() || e.erzwungen {
		return
	}
	jetzt := rettungUhr()
	st := e.s.D.Stand()
	if st != e.rett.beobachtet {
		e.rett.beobachtet, e.rett.letzte = st, jetzt
		if e.rett.erste.IsZero() {
			e.rett.erste = jetzt
		}
	}
	if st == e.rett.gesichert {
		e.rett.erste = time.Time{}
		return
	}
	if !e.s.Geaendert() {
		e.rettung.Verwirf()
		e.rett.gesichert, e.rett.erste = st, time.Time{}
		return
	}
	faellig := e.rett.letzte.Add(rettungRuhe)
	if t := e.rett.erste.Add(rettungTakt); t.Before(faellig) {
		faellig = t
	}
	if jetzt.Before(faellig) {
		if !e.rett.geweckt.Equal(faellig) && e.wecker != nil {
			e.rett.geweckt = faellig
			time.AfterFunc(faellig.Sub(jetzt), e.wecker.Anfordern)
		}
		return
	}
	e.rettung.Sichere(e.s.D)
	e.rett.gesichert, e.rett.erste = st, time.Time{}
}

// rettungVorSpeichern schreibt beim Beginn eines Speicherns eine noch nicht gesicherte Änderung sofort, ohne Bündelung:
// Hängt das Speichern und folgt ein SIGKILL, wäre sie sonst verloren (Verify 7, V7-7).
func (e *editor) rettungVorSpeichern() {
	if e.rettung == nil || e.erzwungen {
		return
	}
	if st := e.s.D.Stand(); st != e.rett.gesichert && e.s.Geaendert() {
		e.rettung.Sichere(e.s.D)
		e.rett.gesichert, e.rett.erste = st, time.Time{}
	}
}

// rettungGespeichert verwirft die Rettungskopie nach einem gelungenen Speichern (ADR-0029 Punkt 5).
func (e *editor) rettungGespeichert() {
	if e.rettung == nil {
		return
	}
	e.rettung.Verwirf()
	st := e.s.D.Stand()
	e.rett.gesichert, e.rett.beobachtet, e.rett.erste = st, st, time.Time{}
}

// rettungEnde räumt beim Zerstören auf: Ohne Änderung gegenüber dem Gespeicherten ist die Kopie überflüssig. Nach
// einem gescheiterten Speichern (auch wenn das Verwerfen bestätigt ist) bleibt sie (ADR-0029 Punkt 5).
func (e *editor) rettungEnde() {
	if e.rettung == nil {
		return
	}
	if !e.s.Geaendert() {
		e.rettung.Verwirf()
	}
	e.rettung.Ende()
}
