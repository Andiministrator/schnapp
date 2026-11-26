// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Übernahme einer Datei von außen bei der ersten Änderung (ADR-0016 Nachtrag 11, Karte #348, F-825): Eine Datei
// außerhalb der Bibliothek öffnet sich, ohne kopiert zu werden. Sobald das Dokument zum ersten Mal geändert ist und keine
// Eingabe mehr läuft, speichert der Editor wie auf Strg+S; dieses Speichern legt sie als neue Aufnahme ab (cmd/schnapp
// ablegenIn), sie erscheint vorn in der Leiste, und die Fußzeile nennt die neue Aufnahme. Die Datei von außen bleibt
// unberührt.

import "github.com/Andiministrator/schnapp/internal/i18n"

// aussenSetzen merkt beim Öffnen und nach jedem Wechsel, ob die offene Datei von außen kommt, und sagt es in der
// Fußzeile: Ohne Änderung kommt sie nicht in die Bibliothek.
func (e *editor) aussenSetzen(aussen bool) {
	e.aussen = aussen
	if aussen {
		e.melde(i18n.T("This file is not in the library. It is added as a new capture as soon as you change it; the file itself stays unchanged."), false)
	}
}

// uebernahmePruefen stößt das Speichern einer Datei von außen nach ihrer ersten Änderung an (Ereignisschleife), sobald
// nichts mehr läuft, was ein Speichern unterbräche oder vorzeitig abschlösse: kein Zug (auch nicht im Farbwähler oder
// beim Bereich für den Text aus dem Bild), keine Texteingabe (SpeichernJetzt bestätigte sie), kein anderes Speichern,
// kein Wechsel, kein Schließen, kein Signal. Angestoßen wird einmal; scheitert es, bleibt es beim gewöhnlichen
// Speichern (Strg+S, Schließen). Liefert, ob es angestoßen hat.
func (e *editor) uebernahmePruefen() bool {
	if !e.aussen || !e.s.Geaendert() {
		return false
	}
	if e.s.Zieht() || e.s.TextLaeuft() || e.s.TextBereichLaeuft() || e.s.SpeichernLaeuft() || e.s.Ruht() ||
		e.s.SignalVorgemerkt() || e.speichernAufgegeben || e.erzwungen || e.wechsel.ziel != "" {
		return false
	}
	e.aussen = false
	e.speichernJetzt()
	return true
}
