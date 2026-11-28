// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Datei → Öffnen… (Strg+O, ADR-0026 Nachtrag 11, Karte #326): Der Dateidialog des Desktop-Portals (OpenFile) läuft in
// einer Goroutine wie der Export; die gewählte Datei öffnet sich über wechselAnfrage im selben Fenster, wie eine
// Aufnahme aus der Leiste — erst wird gespeichert, dann geladen. Ein einfaches Bild wird beim Speichern zu einer neuen
// Aufnahme in der Bibliothek (ADR-0016), eine Arbeitsdatei bleibt, wo sie liegt.

import (
	"context"
	"errors"

	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/portal"
)

// oeffnenDialog zeigt den Dialog; Naht für Tests.
var oeffnenDialog = func(ctx context.Context, a portal.Anfrage) (portal.Ergebnis, error) {
	d, err := portal.Neu()
	if err != nil {
		return portal.Ergebnis{}, err
	}
	defer d.Close()
	return d.Oeffnen(ctx, a)
}

// oeffnenGrau ist die Erklärung am grauen Eintrag Öffnen: ohne Anbindung an die Bibliothek kann das Fenster keine andere
// Datei laden.
func (e *editor) oeffnenGrau() string {
	if e.bib == nil || e.bib.Oeffnen == nil {
		return e.bibliothekGrau()
	}
	return ""
}

// oeffnenStarten öffnet den Dialog; nur einer zugleich, nicht während eines Zugs, Exports oder Wechsels (das Speichern
// sperrt erst die Wahl, über wechselAnfrage).
func (e *editor) oeffnenStarten() {
	if e.oeffnenGrau() != "" || e.oeffnenOffen || e.exportOffen || e.unter.offen || e.s.Zieht() || e.wechsel.laedt {
		return
	}
	e.oeffnenOffen = true
	if e.oeffnenAntwort == nil {
		e.oeffnenAntwort = make(chan exportErgebnis, 1)
	}
	anfrage := portal.Anfrage{
		Titel: i18n.T("Open"),
		Filter: []portal.Filter{
			{Name: i18n.T("Images"), Globs: []string{"*.png", "*.jpg", "*.jpeg"}, MIMEs: []string{"image/png", "image/jpeg"}},
		},
	}
	ctx, ch, wecker, zuordnen := e.exportCtx, e.oeffnenAntwort, e.wecker, e.bib.Zuordnen
	if ctx == nil {
		ctx = context.Background()
	}
	go func() {
		var r exportErgebnis
		r.ergebnis, r.err = oeffnenDialog(ctx, anfrage)
		if r.err == nil && zuordnen != nil {
			// Hier und nicht in der Ereignisschleife: Die Zuordnung liest die Datei (Summe).
			if pfad, err := portal.URIZuPfad(r.ergebnis.URI); err == nil {
				r.zugeordnet = zuordnen(pfad)
			}
		}
		ch <- r
		if wecker != nil {
			wecker.Anfordern() // erst nach dem Ablegen der Antwort (F-231)
		}
	}()
}

// oeffnenAbholen nimmt die Antwort des Dialogs in der Ereignisschleife an; Rückgabe: ob eine kam.
func (e *editor) oeffnenAbholen() bool {
	select {
	case r := <-e.oeffnenAntwort: // nil-Kanal: default
		e.oeffnenOffen = false
		e.oeffnenFertig(r)
		return true
	default:
		return false
	}
}

// oeffnenFertig wertet die Wahl aus: Abbruch tut nichts, Fehler stehen in der Fußzeile, sonst wechselt das Fenster.
func (e *editor) oeffnenFertig(r exportErgebnis) {
	switch {
	case errors.Is(r.err, portal.ErrAbgebrochen), errors.Is(r.err, context.Canceled):
		return
	case errors.Is(r.err, portal.ErrKeinPortal):
		diagnoseEditor("Öffnen: %v", r.err)
		e.meldeNebenbei(i18n.T("No file dialog available (desktop portal missing)."), true)
		return
	case r.err != nil:
		diagnoseEditor("Öffnen: %v", r.err)
		e.meldeNebenbei(i18n.T("Could not show the file dialog."), true)
		return
	}
	pfad, err := portal.URIZuPfad(r.ergebnis.URI)
	if err != nil {
		e.meldeNebenbei(i18n.T("Only local files can be opened."), true)
		return
	}
	if r.zugeordnet != "" && e.istOffen(r.zugeordnet) {
		// Die gewählte Datei ist die offene Aufnahme (früher übernommen, F-828): nicht speichern und neu laden, der
		// Rückgängig-Verlauf bliebe sonst nicht.
		e.meldeNebenbei(i18n.T("This file is already open."), false)
		return
	}
	if e.wechselAnfrage(pfad) {
		e.wechsel.ausDialog = true // nach wechselAnfrage: die setzt es zurück
	}
}
