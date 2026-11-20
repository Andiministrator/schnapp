// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

// Meldung ist eine Zeile der Statusleiste mit dem Zeitpunkt, an dem sie entstand (F-151): Dokumentstand und
// Werkzeug. Daran erkennt StatusZeile, ob sie noch frisch ist.
type Meldung struct {
	Text   string
	Fehler bool

	stand    uint64
	werkzeug Werkzeug
	da       bool
}

// NeueMeldung hält text fest, wie sie bei diesem Dokumentstand und Werkzeug entsteht (nach dem Speichern, einem
// Hinweis oder Fehler).
func NeueMeldung(text string, fehler bool, stand uint64, w Werkzeug) Meldung {
	return Meldung{Text: text, Fehler: fehler, stand: stand, werkzeug: w, da: text != ""}
}

// Frisch meldet, ob die Meldung noch gilt: Eine gewöhnliche Meldung endet mit der nächsten Änderung am Dokument (anderer
// Verlaufsstand) oder dem Wechsel des Werkzeugs; ein Fehler bleibt bis zur nächsten Änderung stehen (oder bis eine
// neue Meldung, etwa ein gelungenes Speichern, ihn ablöst) — er wird nicht schon durch einen Werkzeugwechsel
// überdeckt.
func (m Meldung) Frisch(stand uint64, w Werkzeug) bool {
	if !m.da || m.stand != stand {
		return false
	}
	return m.Fehler || m.werkzeug == w
}

// StatusZeile entscheidet, was die linke Seite der Statusleiste zeigt (F-151): eine frische Meldung hat Vorrang vor
// dem Hinweis des Werkzeugs; danach zeigt die Leiste wieder den Hinweis (leer beim Werkzeug Auswählen).
func StatusZeile(m Meldung, stand uint64, w Werkzeug, hinweis string) (text string, fehler, meldung bool) {
	if m.Frisch(stand, w) {
		return m.Text, m.Fehler, true
	}
	return hinweis, false, false
}
