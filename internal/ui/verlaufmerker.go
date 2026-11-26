// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Verlauf je Aufnahme beim Wechsel im selben Fenster (ADR-0036 Zusatz Karte #356): Der Editor merkt sich das Dokument
// der verlassenen Aufnahme samt Rückgängig-Verlauf und dem Dateistand nach dem Speichern. Führt ein Wechsel zurück,
// lädt Anbindung.Fortsetzen die Datei und nimmt das gemerkte Dokument, wenn der Stand gleich geblieben ist.

import (
	"path/filepath"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/library"
)

// Grenzen der gemerkten Aufnahmen (ADR-0036 Zusatz Karte #356 Punkt 4).
const (
	merkerAnzahl       = 4
	merkerGrenze int64 = 512 << 20
)

// Fortsetzen lädt die Datei pfad wie Oeffnen und nimmt statt des gelesenen Dokuments d, wenn die Datei noch den Stand
// stand hat; sonst gilt das gelesene (Geoeffnet.Fortgesetzt sagt, welches). Läuft in einer Goroutine.
type Fortsetzen func(pfad string, d *document.Dokument, stand library.Stand) (Geoeffnet, error)

// gemerkteSitzung ist eine verlassene Aufnahme: absoluter Pfad, Dokument mit Verlauf, Dateistand beim Verlassen.
type gemerkteSitzung struct {
	pfad   string
	d      *document.Dokument
	stand  library.Stand
	bedarf int64
}

// verlaufMerker hält die gemerkten Aufnahmen, die zuletzt verlassene am Ende. Nur die Ereignisschleife greift zu.
type verlaufMerker struct {
	liste []gemerkteSitzung
}

// merke nimmt g auf (ersetzt einen Eintrag desselben Pfads) und wirft die am längsten verlassenen weg, bis Anzahl und
// Speicher in den Grenzen liegen. Ein Dokument allein über der Grenze wird nicht gemerkt.
func (m *verlaufMerker) merke(g gemerkteSitzung) {
	g.pfad = filepath.Clean(g.pfad)
	m.vergiss(g.pfad)
	if g.bedarf > merkerGrenze {
		return
	}
	m.liste = append(m.liste, g)
	summe := int64(0)
	for _, e := range m.liste {
		summe += e.bedarf
	}
	for len(m.liste) > merkerAnzahl || summe > merkerGrenze {
		summe -= m.liste[0].bedarf
		m.liste[0] = gemerkteSitzung{}
		m.liste = m.liste[1:]
	}
}

// nimm liefert den Eintrag für pfad und entfernt ihn (er wird beim nächsten Verlassen neu gemerkt).
func (m *verlaufMerker) nimm(pfad string) (gemerkteSitzung, bool) {
	pfad = filepath.Clean(pfad)
	for i, e := range m.liste {
		if e.pfad == pfad {
			m.liste = append(m.liste[:i:i], m.liste[i+1:]...)
			return e, true
		}
	}
	return gemerkteSitzung{}, false
}

// vergiss entfernt den Eintrag für pfad, falls es einen gibt.
func (m *verlaufMerker) vergiss(pfad string) {
	m.nimm(pfad)
}

// merkeVerlassene merkt die offene Aufnahme, bevor uebernimm sie ersetzt — nur wenn es sich lohnt und sicher ist
// (Punkt 3): Fortsetzen vorhanden, Pfad bekannt, keine Datei von außen, gespeichert, keine Schwärzung im Verlauf, und
// es gibt einen Verlauf. Der Stand ist der der Datei jetzt, direkt nach dem Speichern vor dem Wechsel.
func (e *editor) merkeVerlassene() {
	if e.bib == nil || e.bib.Fortsetzen == nil || e.pfad == nil || e.aussen || e.s.Geaendert() || e.s.Zieht() {
		return
	}
	d := e.s.D
	if d.VerlaufMitSchwaerzung() || !(d.KannRueckgaengig() || d.KannWiederholen()) {
		return
	}
	p := e.pfad()
	if p == "" {
		return
	}
	stand, err := library.StandVon(p)
	if err != nil {
		return
	}
	e.verlaeufe.merke(gemerkteSitzung{pfad: p, d: d, stand: stand, bedarf: d.Speicherbedarf()})
}
