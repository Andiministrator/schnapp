// SPDX-License-Identifier: GPL-3.0-or-later

package libview

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
)

// Zeitraum ist die Auswahl im Kopf der vollen Ansicht (ADR-0036 Punkt 3): heute, die letzten 7 oder 30 Tage (heute
// eingeschlossen), das laufende Jahr oder alles.
type Zeitraum int

const (
	ZeitraumAlle Zeitraum = iota // Nullwert: alles
	ZeitraumHeute
	ZeitraumWoche
	ZeitraumMonat
	ZeitraumJahr
)

// Zeitraeume sind die Zeiträume in der Reihenfolge der Auswahlliste.
var Zeitraeume = []Zeitraum{ZeitraumHeute, ZeitraumWoche, ZeitraumMonat, ZeitraumJahr, ZeitraumAlle}

// Von ist der Beginn des Zeitraums in der Zone von jetzt (Mitternacht); der Nullwert bei „alles“.
func (z Zeitraum) Von(jetzt time.Time) time.Time {
	j, m, d := jetzt.Date()
	heute := time.Date(j, m, d, 0, 0, 0, 0, jetzt.Location())
	switch z {
	case ZeitraumHeute:
		return heute
	case ZeitraumWoche:
		return heute.AddDate(0, 0, -6)
	case ZeitraumMonat:
		return heute.AddDate(0, 0, -29)
	case ZeitraumJahr:
		return time.Date(j, time.January, 1, 0, 0, 0, 0, jetzt.Location())
	}
	return time.Time{}
}

// Text ist der Name des Zeitraums in der Sprache der Oberfläche.
func (z Zeitraum) Text() string {
	switch z {
	case ZeitraumHeute:
		return i18n.T("Today")
	case ZeitraumWoche:
		return i18n.T("Last 7 days")
	case ZeitraumMonat:
		return i18n.T("Last 30 days")
	case ZeitraumJahr:
		return i18n.T("This year")
	}
	return i18n.T("All dates")
}

// Anfrage baut die Suche der vollen Ansicht: Teil eines Schlagworts aus dem Suchfeld (leer: alle; F-671), Zeitraum, Agenten-Aufnahmen nur
// mit dem Schalter (ADR-0036 Punkt 3).
func Anfrage(suche string, z Zeitraum, agenten bool, jetzt time.Time) library.Anfrage {
	return library.Anfrage{Teilwort: strings.TrimSpace(suche), Von: z.Von(jetzt), OhneAgenten: !agenten}
}

// ZerlegeSchlagworte zerlegt die Eingabe unter einer Kachel (F2) in Schlagworte: Komma trennt, Leerraum an den Enden
// fällt weg, leere und (ohne Rücksicht auf Groß- und Kleinschreibung) doppelte fallen heraus (ADR-0036 Punkt 4).
func ZerlegeSchlagworte(s string) []string {
	var aus []string
	for _, w := range strings.Split(s, ",") {
		w = strings.TrimSpace(w)
		if w == "" {
			continue
		}
		doppelt := false
		for _, a := range aus {
			if strings.EqualFold(a, w) {
				doppelt = true
				break
			}
		}
		if !doppelt {
			aus = append(aus, w)
		}
	}
	return aus
}

// SchlagwortText ist der Text, mit dem das Eingabefeld beginnt: die Schlagworte mit Komma und Leerzeichen.
func SchlagwortText(worte []string) string { return strings.Join(worte, ", ") }

// MaxEingabe ist die größte Zahl von Zeichen im Suchfeld und im Schlagwortfeld.
const MaxEingabe = 200

// Tippe hängt getippten Text an die Eingabe s an: ohne Steuerzeichen (ein Zeilenumbruch übernimmt nicht), höchstens
// MaxEingabe Zeichen.
func Tippe(s, text string) string {
	n := len([]rune(s))
	var b strings.Builder
	b.WriteString(s)
	for _, r := range text {
		if n >= MaxEingabe {
			break
		}
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
		n++
	}
	return b.String()
}

// Zurueck nimmt das letzte Zeichen der Eingabe weg (Rücktaste).
func Zurueck(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	return string(r[:len(r)-1])
}

// KachelZeit ist Datum und Uhrzeit unter einer Kachel: im laufenden Jahr ohne Jahr. Die Muster kommen wie bei der
// Leiste über den Kontext „date part“.
func KachelZeit(t, jetzt time.Time) string {
	t = t.In(jetzt.Location())
	uhr := t.Format(i18n.TK("date part", "15:04"))
	if t.Year() == jetzt.Year() {
		return t.Format(i18n.TK("date part", "Jan 2")) + " " + uhr
	}
	return t.Format(i18n.TK("date part", "Jan 2, 2006")) + " " + uhr
}

// KachelAngaben ist die zweite Zeile unter einer Kachel: Größe in Pixeln und die Schlagworte.
func KachelAngaben(e library.Eintrag) string {
	s := fmt.Sprintf(i18n.TK("status bar", "%d × %d"), e.Breite, e.Hoehe)
	if len(e.Schlagworte) > 0 {
		s += " · " + strings.Join(e.Schlagworte, ", ")
	}
	return s
}

// MonatTitel ist die Überschrift eines Monats im Raster: Name und Jahr, dazu die Zahl der Aufnahmen.
func MonatTitel(g Gruppe) string {
	return monatName(g.Monat) + " " + strconv.Itoa(g.Jahr) + "  ·  " +
		fmt.Sprintf(i18n.N("%d capture", "%d captures", g.Anzahl()), g.Anzahl())
}

// monatName ist der Name des Monats in der Sprache der Oberfläche (Go kennt nur englische Namen).
func monatName(m time.Month) string {
	switch m {
	case time.January:
		return i18n.TK("month", "January")
	case time.February:
		return i18n.TK("month", "February")
	case time.March:
		return i18n.TK("month", "March")
	case time.April:
		return i18n.TK("month", "April")
	case time.May:
		return i18n.TK("month", "May")
	case time.June:
		return i18n.TK("month", "June")
	case time.July:
		return i18n.TK("month", "July")
	case time.August:
		return i18n.TK("month", "August")
	case time.September:
		return i18n.TK("month", "September")
	case time.October:
		return i18n.TK("month", "October")
	case time.November:
		return i18n.TK("month", "November")
	}
	return i18n.TK("month", "December")
}
