// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"time"

	"gioui.org/io/key"
)

// freigabe ist die Logik des Bestätigungsdialogs ohne Fenster (ADR-0030 Nachtrag 4 Punkt 4, F-464): Erlauben geht nur
// über einen Mausklick, dessen Drücken UND Loslassen nach der Sperre liegen. Für die Tastatur gibt es keinen Weg zu
// „Erlauben“; Esc lehnt ab. Die Zeiten sind die der Bilder, in denen die Ereignisse ankommen.
type freigabe struct {
	sperreBis    time.Time // Ende der Sperre; null, solange kein Bild gezeigt wurde
	druckGueltig bool      // das laufende Drücken auf „Erlauben“ lag nach der Sperre
	fokussiert   bool      // das Fenster hat den Fokus
	neuSperren   bool      // das nächste Bild setzt die Sperre neu (Fokus erhalten)
}

// bild meldet ein Bild zur Zeit jetzt; das erste setzt die Sperre, ebenso das erste nach einem Fokus-Erhalt.
func (f *freigabe) bild(jetzt time.Time) {
	if f.sperreBis.IsZero() || f.neuSperren {
		f.sperreBis = jetzt.Add(bestaetigenSperre)
		f.neuSperren = false
	}
}

// fokus meldet den Fokus des Fensters (F-481). Bekommt das Fenster ihn, beginnt die Sperre mit dem nächsten Bild neu:
// Erst dann ist es sicher zu sehen, und ein Fenster, das sich nach vorn drängt, öffnet kein Klickfenster. Verliert es
// ihn, verfällt ein laufendes Drücken.
func (f *freigabe) fokus(hat bool) {
	switch {
	case hat && !f.fokussiert:
		f.neuSperren = true
		f.druckGueltig = false
	case !hat:
		f.druckGueltig = false
	}
	f.fokussiert = hat
}

// gesperrt: vor dem ersten Bild oder innerhalb der Sperre.
func (f *freigabe) gesperrt(jetzt time.Time) bool {
	return f.sperreBis.IsZero() || jetzt.Before(f.sperreBis)
}

// druck: Der Zeiger drückt auf „Erlauben“. Ein Drücken in der Sperre macht den ganzen Klick ungültig, auch wenn das
// Loslassen nach der Sperre kommt.
func (f *freigabe) druck(jetzt time.Time) {
	f.druckGueltig = !f.gesperrt(jetzt)
}

// abbruch: Der Klick wurde abgebrochen (Zeiger verlassen, Fenster verloren).
func (f *freigabe) abbruch() { f.druckGueltig = false }

// klick: Losgelassen über „Erlauben“. true heißt erlaubt.
func (f *freigabe) klick(jetzt time.Time) bool {
	ok := f.druckGueltig && !f.gesperrt(jetzt)
	f.druckGueltig = false
	return ok
}

// taste wertet eine Taste aus. Keine Taste erlaubt je etwas; Esc lehnt ab.
func (f *freigabe) taste(name key.Name) (ablehnen bool) {
	return name == key.NameEscape
}
