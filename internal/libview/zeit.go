// SPDX-License-Identifier: GPL-3.0-or-later

// Package libview ist die Logik der Bibliothek in der Oberfläche ohne Gio (ADR-0036 Punkt 9): die Leiste der letzten
// Aufnahmen (Liste, Fokus, Rollen), die Zeitangabe unter einer Vorschau und der Speicher der Vorschaubilder, die im
// Hintergrund aus dem Vorschau-Cache der Bibliothek geladen werden. Gezeichnet wird in internal/ui (leiste_unten.go).
package libview

import (
	"time"

	"github.com/Andiministrator/schnapp/internal/i18n"
)

// Zeitangabe ist der Text unter einer Vorschau (ADR-0036 Punkt 1): heute die Uhrzeit, gestern „yesterday“, sonst
// Wochentag und Datum, in einem anderen Jahr mit Jahr. Gerechnet wird in der Zone von jetzt; die Muster kommen über den
// Kontext „date part“ aus dem Katalog (Teilmuster ohne Jahr oder ohne Datum; „date format“ verlangt laut KA-7 ein
// minutengenau zurücklesbares Muster, ADR-0014 Nachtrag 2), die Wochentage über den Kontext „weekday“.
func Zeitangabe(t, jetzt time.Time) string {
	t = t.In(jetzt.Location())
	j, m, d := t.Date()
	hj, hm, hd := jetzt.Date()
	gj, gm, gd := jetzt.AddDate(0, 0, -1).Date()
	switch {
	case j == hj && m == hm && d == hd:
		return t.Format(i18n.TK("date part", "15:04"))
	case j == gj && m == gm && d == gd:
		return i18n.T("yesterday")
	case j == hj:
		return wochentag(t.Weekday()) + " " + t.Format(i18n.TK("date part", "Jan 2"))
	}
	return wochentag(t.Weekday()) + " " + t.Format(i18n.TK("date part", "Jan 2, 2006"))
}

// wochentag ist der kurze Name des Wochentags in der Sprache der Oberfläche (Go kennt nur englische Namen).
func wochentag(w time.Weekday) string {
	switch w {
	case time.Monday:
		return i18n.TK("weekday", "Mon")
	case time.Tuesday:
		return i18n.TK("weekday", "Tue")
	case time.Wednesday:
		return i18n.TK("weekday", "Wed")
	case time.Thursday:
		return i18n.TK("weekday", "Thu")
	case time.Friday:
		return i18n.TK("weekday", "Fri")
	case time.Saturday:
		return i18n.TK("weekday", "Sat")
	}
	return i18n.TK("weekday", "Sun")
}
