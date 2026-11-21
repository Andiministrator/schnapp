// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

// Meldung ist eine Zeile der Statusleiste mit dem Zeitpunkt, an dem sie entstand (F-151): Dokumentstand und
// Werkzeug. Daran erkennt StatusZeile, ob sie noch frisch ist. Eine Meldung, die einmal abgelöst war, kehrt nicht
// zurück (F-160): Ändern und Rückgängig oder ein Werkzeugwechsel hin und zurück führen auf den alten Stand, die
// Meldung beschriebe dann aber einen Zustand, der inzwischen vorbei war.
type Meldung struct {
	Text   string
	Fehler bool

	stand     uint64
	werkzeug  Werkzeug
	da        bool
	abgeloest bool
}

// NeueMeldung hält text fest, wie sie bei diesem Dokumentstand und Werkzeug entsteht (nach dem Speichern, einem
// Hinweis oder Fehler).
func NeueMeldung(text string, fehler bool, stand uint64, w Werkzeug) Meldung {
	return Meldung{Text: text, Fehler: fehler, stand: stand, werkzeug: w, da: text != ""}
}

// Frisch meldet, ob die Meldung noch gilt: Eine gewöhnliche Meldung endet mit der nächsten Änderung am Dokument (anderer
// Verlaufsstand) oder dem Wechsel des Werkzeugs; ein Fehler bleibt bis zur nächsten Änderung stehen (oder bis eine
// neue Meldung, etwa ein gelungenes Speichern, ihn ablöst) — er wird nicht schon durch einen Werkzeugwechsel
// überdeckt. Endet die Meldung, merkt sie sich das: Sie kehrt nicht zurück, auch wenn Stand und Werkzeug später wieder
// stimmen (F-160). Die Oberfläche fragt deshalb bei jedem Bild.
func (m *Meldung) Frisch(stand uint64, w Werkzeug) bool {
	if !m.da || m.abgeloest {
		return false
	}
	if m.stand != stand || (!m.Fehler && m.werkzeug != w) {
		m.abgeloest = true
		return false
	}
	return true
}

// StatusZeile entscheidet, was die linke Seite der Statusleiste zeigt (F-151): eine frische Meldung hat Vorrang vor
// dem Hinweis des Werkzeugs; danach zeigt die Leiste wieder den Hinweis (leer beim Werkzeug Auswählen). Während einer
// Texteingabe (eingabe) geht der Hinweis vor einer gewöhnlichen Meldung wie „Gespeichert“ — sie gälte nicht mehr für
// das, was gerade getippt wird, und löst sich damit ab; ein Fehler bleibt sichtbar (F-160).
func StatusZeile(m *Meldung, stand uint64, w Werkzeug, hinweis string, eingabe bool) (text string, fehler, meldung bool) {
	if m.Frisch(stand, w) {
		if eingabe && !m.Fehler {
			m.abgeloest = true
			return hinweis, false, false
		}
		return m.Text, m.Fehler, true
	}
	return hinweis, false, false
}
