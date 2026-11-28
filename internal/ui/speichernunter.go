// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Datei → „Save As…“ (Strg+Umschalt+S, ADR-0046 Punkte 1 bis 6): Der Speichern-Dialog des Desktop-Portals läuft in einer
// Goroutine wie der Export; danach schreibt eine zweite Goroutine einen Abzug als Arbeitsdatei an den gewählten Ort.
// Das Fenster bleibt bei der Aufnahme in der Bibliothek: Titel, Pfad, Rettungskopie und ungesicherter Stand ändern sich
// nicht, die Kopie ist eine bewusste Ausgabe wie ein Export.

import (
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/export"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/portal"
	"github.com/Andiministrator/schnapp/internal/render"
)

// unterZustand ist ein laufendes Speichern unter; nur die Ereignisschleife liest und setzt es.
type unterZustand struct {
	offen   bool                // der Dialog läuft
	dialog  chan exportErgebnis // Antwort des Dialogs
	laeuft  bool                // das Schreiben läuft
	antwort chan unterErgebnis  // Antwort des Schreibens
}

// unterErgebnis ist das Ende eines Schreibens.
type unterErgebnis struct {
	name        string // Dateiname des Ziels (ohne Ordner, für die Fußzeile)
	err         error
	ueberBudget bool // geschrieben, aber über dem Rechenbudget (wie beim Speichern, ADR-0046 Punkt 6)
	ganz        bool // das Dokument hat Zuschnitt oder Streifen: die Kopie trägt mehr als das sichtbare Bild (Punkt 5)
}

// unterDialog zeigt den Speichern-Dialog; Naht für Tests.
var unterDialog = func(ctx context.Context, a portal.Anfrage) (portal.Ergebnis, error) {
	d, err := portal.Neu()
	if err != nil {
		return portal.Ergebnis{}, err
	}
	defer d.Close()
	return d.Speichern(ctx, a)
}

// unterSchreiben schreibt die Arbeitsdatei mit dem Inhalt schreibe nach pfad; ohneErsetzen: ein vorhandenes Ziel
// bleibt (angehängte Endung). Naht für Tests.
var unterSchreiben = func(pfad string, ohneErsetzen bool, schreibe func(io.Writer) error) error {
	if ohneErsetzen {
		return export.SchreibeOhneErsetzen(pfad, schreibe)
	}
	return export.SchreibeAtomar(pfad, schreibe)
}

// arbeitsdateiEndung ist die Endung einer Arbeitsdatei (ADR-0008: PNG mit eigenen Blöcken).
const arbeitsdateiEndung = ".png"

// speichernUnterStarten öffnet den Dialog; nur einer zugleich, nicht neben einem anderen Dialog, einem laufenden
// Schreiben, einem Zug oder dem Laden einer anderen Aufnahme (ADR-0046 Punkt 6).
func (e *editor) speichernUnterStarten() {
	if e.unter.offen || e.unter.laeuft || e.exportOffen || e.oeffnenOffen || e.s.Zieht() || e.wechsel.laedt {
		return
	}
	e.unter.offen = true
	if e.unter.dialog == nil {
		e.unter.dialog = make(chan exportErgebnis, 1)
	}
	anfrage := portal.Anfrage{
		Titel:     i18n.T("Save As"),
		Dateiname: strings.TrimSuffix(e.titel, filepath.Ext(e.titel)) + arbeitsdateiEndung,
		Filter: []portal.Filter{
			{Name: i18n.T("Schnapp image"), Globs: []string{"*" + arbeitsdateiEndung}, MIMEs: []string{"image/png"}},
		},
	}
	ctx, ch, wecker := e.exportCtx, e.unter.dialog, e.wecker
	if ctx == nil {
		ctx = context.Background()
	}
	go func() {
		var r exportErgebnis
		r.ergebnis, r.err = unterDialog(ctx, anfrage)
		ch <- r
		if wecker != nil {
			wecker.Anfordern() // erst nach dem Ablegen der Antwort (F-231)
		}
	}()
}

// speichernUnterAbholen nimmt die Antworten von Dialog und Schreiben in der Ereignisschleife an; Rückgabe: ob eine kam.
func (e *editor) speichernUnterAbholen() bool {
	select {
	case r := <-e.unter.dialog: // nil-Kanal: default
		e.unter.offen = false
		e.speichernUnterGewaehlt(r)
		return true
	case r := <-e.unter.antwort:
		e.unter.laeuft = false
		e.speichernUnterFertig(r)
		return true
	default:
		return false
	}
}

// speichernUnterGewaehlt wertet die Wahl aus und beginnt das Schreiben: Abbruch tut nichts, Fehler stehen in der
// Fußzeile.
func (e *editor) speichernUnterGewaehlt(r exportErgebnis) {
	switch {
	case errors.Is(r.err, portal.ErrAbgebrochen), errors.Is(r.err, context.Canceled):
		return
	case errors.Is(r.err, portal.ErrKeinPortal):
		diagnoseEditor("Speichern unter: %v", r.err)
		e.meldeNebenbei(i18n.T("No file dialog available (desktop portal missing)."), true)
		return
	case r.err != nil:
		diagnoseEditor("Speichern unter: %v", r.err)
		e.meldeNebenbei(i18n.T("Could not show the file dialog."), true)
		return
	}
	pfad, err := portal.URIZuPfad(r.ergebnis.URI)
	if err != nil {
		e.meldeNebenbei(i18n.T("Could not save: only local files can be written."), true)
		return
	}
	angehaengt := false
	if !strings.EqualFold(filepath.Ext(pfad), arbeitsdateiEndung) {
		pfad, angehaengt = pfad+arbeitsdateiEndung, true
	}
	// Der Dialog hat nur den Namen ohne Endung bestätigt: ein vorhandenes Ziel nicht still ersetzen (wie der Export, A-1).
	if err := export.PruefeAngehaengt(pfad, angehaengt); err != nil {
		e.meldeNebenbei(exportFehlerText(err, filepath.Base(pfad)), true)
		return
	}
	quelle := ""
	if e.pfad != nil {
		quelle = e.pfad()
	}
	if gleicheDatei(quelle, pfad) || selbeDatei(quelle, pfad) {
		// An der Speicherfunktion vorbei geschrieben, sähe das nächste Speichern eine Änderung von außen (Punkt 4).
		e.meldeNebenbei(i18n.T("This is the open capture itself. Save it with Ctrl+S."), true)
		return
	}
	// Wie beim Export: ein laufender Text gehört mit in die Datei, ein laufendes Ziehen verschiebt das Schreiben.
	e.s.TextBestaetigen()
	abzug, err := e.s.D.Abzug()
	if err != nil || e.s.Zieht() {
		e.meldeNebenbei(i18n.T("Could not save: finish the current action first."), true)
		return
	}
	e.unter.laeuft = true
	if e.unter.antwort == nil {
		e.unter.antwort = make(chan unterErgebnis, 1)
	}
	e.melde(i18n.T("Saving a copy…"), false)
	ch, wecker := e.unter.antwort, e.wecker
	go func() {
		ch <- schreibeKopie(abzug, quelle, pfad, angehaengt)
		if wecker != nil {
			wecker.Anfordern()
		}
	}()
}

// selbeDatei meldet, ob a und b dieselbe vorhandene Datei sind, auch über einen anderen Namen (harter Verweis);
// gleicheDatei deckt gleichen Pfad und Ordnerverweise ab.
func selbeDatei(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	sa, err := os.Stat(a)
	if err != nil {
		return false
	}
	sb, err := os.Stat(b)
	return err == nil && os.SameFile(sa, sb)
}

// schreibeKopie rendert den Abzug d und schreibt ihn als Arbeitsdatei nach pfad; Metadaten aus quelle (ADR-0046
// Punkt 2). Läuft in einer Goroutine.
func schreibeKopie(d *document.Dokument, quelle, pfad string, ohneErsetzen bool) unterErgebnis {
	r := unterErgebnis{name: filepath.Base(pfad), ganz: zeigtNichtAlles(d)}
	r.ueberBudget = render.Pruefe(d) != nil // geschrieben wird trotzdem, wie beim Speichern (Punkt 6)
	m := kopieMetadaten(quelle)
	var sichtbar image.Image
	sichtbar, r.err = render.Bild(d)
	if r.err != nil {
		return r
	}
	r.err = unterSchreiben(pfad, ohneErsetzen, func(w io.Writer) error {
		return fileformat.SchreibeArbeitsdatei(w, d, sichtbar, m)
	})
	return r
}

// kopieMetadaten liest Autor, Aufnahmezeit und Schlagworte der Datei quelle; ist sie keine lesbare Arbeitsdatei, ohne
// Autor und Schlagworte mit der Zeit des Schreibens. Immer „bearbeitet“, nie ein Pfad (Kernprinzip 3).
func kopieMetadaten(quelle string) fileformat.Metadaten {
	var m fileformat.Metadaten
	if quelle != "" {
		if f, err := os.Open(quelle); err == nil {
			if g, err := fileformat.LeseMetadaten(f); err == nil && !g.Fremd {
				m = g.Meta
			}
			f.Close()
		}
	}
	m.Bearbeitet = true
	if m.Erstellt.IsZero() {
		m.Erstellt = time.Now().UTC().Truncate(time.Second)
	}
	return m
}

// zeigtNichtAlles meldet, ob das sichtbare Bild weniger zeigt als das Original (Zuschnitt oder Streifen).
func zeigtNichtAlles(d *document.Dokument) bool {
	return len(d.Streifen()) > 0 || d.Zuschnitt() != d.OriginalBild().Rect
}

// speichernUnterFertig zeigt das Ende des Schreibens in der Fußzeile.
func (e *editor) speichernUnterFertig(r unterErgebnis) {
	if r.err != nil {
		e.meldeNebenbei(fmt.Sprintf(i18n.T("Could not save a copy: %v"), exportFehlerText(r.err, "")), true)
		return
	}
	text := fmt.Sprintf(i18n.T("Saved a copy as %s. This window keeps editing the capture in the library."), r.name)
	if r.ganz {
		text += " " + i18n.T("The copy contains the whole original; to share only the visible image, use Export.")
	}
	if r.ueberBudget {
		text += " " + i18n.T("It has too many or too large annotations to be opened again.")
	}
	e.meldeNebenbei(text, r.ueberBudget)
}

// speichernUnterAbwarten läuft beim Zerstören: Ein laufendes Schreiben bekommt höchstens frist (ADR-0046 Punkt 6),
// danach steht ein Hinweis auf der Fehlerausgabe; die Fußzeile gibt es dann nicht mehr.
func (e *editor) speichernUnterAbwarten(frist time.Duration) {
	if !e.unter.laeuft {
		return
	}
	t := time.NewTimer(frist)
	defer t.Stop()
	select {
	case r := <-e.unter.antwort:
		e.unter.laeuft = false
		if r.err != nil {
			fmt.Fprintf(fehlerAusgabe, i18n.T("Could not save a copy: %v")+"\n", exportFehlerText(r.err, ""))
		}
	case <-t.C:
		fmt.Fprintln(fehlerAusgabe, i18n.T("Saving a copy was not finished when the window closed; the copy may be missing."))
	}
}
