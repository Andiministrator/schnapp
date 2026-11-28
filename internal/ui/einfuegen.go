// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"slices"
	"time"

	"gioui.org/layout"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/render"
	"github.com/Andiministrator/schnapp/internal/wlclip"
)

// Objekte kopieren, ausschneiden und einfügen, neue Aufnahme aus der Zwischenablage (ADR-0045). Kopieren geht über
// denselben Weg wie das Bild (kopierAntwort, Halter der Art ArtObjekte); Lesen über wlclip.Lies in einer Goroutine.

// maxObjekteDaten begrenzt, was Strg+V als Objekte liest (wie das SVG eines snAP-Blocks).
const maxObjekteDaten = 64 << 20

// liesZwischenablage liest die Zwischenablage (wlclip.Lies); Naht für Tests.
var liesZwischenablage = wlclip.Lies

// bildTypen sind die Bildtypen, die „New from clipboard“ liest, in dieser Reihenfolge (ADR-0045 Punkt 4).
var bildTypen = []string{wlclip.MIME, "image/jpeg"}

// einfuegeZustand ist ein laufendes Lesen der Zwischenablage (Strg+V oder „New from clipboard“).
type einfuegeZustand struct {
	laeuft  bool
	antwort chan einfuegeErgebnis
	// kopie: Objekte der letzten eigenen Kopie, die das Fenster selbst anbietet (kein Halter, ADR-0045 Punkt 2). Gilt
	// nur, solange fensterHaeltKopie gilt.
	kopie *document.Dokument
}

// einfuegeErgebnis ist die Antwort des Lesens.
type einfuegeErgebnis struct {
	neu     bool               // „New from clipboard“: pfad der neuen Aufnahme
	pfad    string             // neu: abgelegte Aufnahme
	q       *document.Dokument // Strg+V: gelesene Objekte
	typen   []string           // angebotene Typen (für die Meldung, wenn nichts passte)
	err     error
	gelesen bool // es gab etwas zu lesen (sonst bestimmen typen die Meldung)
}

// kopierenOderObjekte ist Strg+C: mit Auswahl die Objekte, sonst das Bild (ADR-0045 Punkt 1).
func (e *editor) kopierenOderObjekte(gtx layout.Context) {
	if e.s.KannObjekteKopieren() {
		e.objekteKopieren(false)
		return
	}
	e.kopieren(gtx)
}

// objekteKopieren kopiert die Auswahl (ADR-0045 Punkt 2): das SVG ohne Pixel sofort, das PNG der Objekte in einer
// Goroutine auf einem Abzug wie das Bild; ausschneiden löscht die Auswahl danach in einem Schritt.
func (e *editor) objekteKopieren(ausschneiden bool) {
	if e.kopierLaeuft {
		e.melde(i18n.T("Copying is already in progress."), false)
		return
	}
	if ausschneiden && !e.s.KannAusschneiden() {
		return
	}
	q, err := e.s.KopierObjekte()
	if err != nil {
		if !errors.Is(err, canvas.ErrNichtsGewaehlt) {
			e.melde(i18n.T("Could not copy: finish the current action first."), true)
		}
		return
	}
	ids := e.s.Gewaehlt()
	abzug, err := e.s.D.Abzug()
	if err != nil {
		e.melde(i18n.T("Could not copy: finish the current action first."), true)
		return
	}
	svg := fileformat.AngabenSVG(q)
	n := len(q.Objekte())
	if ausschneiden {
		e.s.Ausschneiden()
	}
	e.kopierLaeuft, e.kopierStart, e.kopierStand = true, time.Now(), e.s.D.Stand()
	e.melde(i18n.T("Copying…"), false)
	if e.kopierAntwort == nil {
		e.kopierAntwort = make(chan kopierErgebnis, 1)
	}
	ch, wecker := e.kopierAntwort, e.wecker
	meldung := fmt.Sprintf(i18n.N("Copied %d object to the clipboard", "Copied %d objects to the clipboard", n), n)
	if ausschneiden {
		meldung = fmt.Sprintf(i18n.N("Cut %d object to the clipboard", "Cut %d objects to the clipboard", n), n)
	}
	go func() {
		r := kopierErgebnis{meldung: meldung}
		png, err := render.ObjektePNG(abzug, ids)
		switch {
		case err != nil:
			r.err = err
		default:
			if err := halteZwischenablage(wlclip.Packe(svg, png), wlclip.ArtObjekte); err != nil {
				diagnoseEditor("Halter der Zwischenablage nicht gestartet, Fenster bietet das Bild der Objekte an: %v", err)
				r.daten, r.typ, r.objekte = png, wlclip.MIME, q
			}
		}
		ch <- r
		if wecker != nil {
			wecker.Anfordern()
		}
	}()
}

// einfuegenStarten ist Strg+V (ADR-0045 Punkt 3): liest die Zwischenablage in einer Goroutine; einfuegenAbholen
// fügt ein. Ohne Lesemöglichkeit nimmt es die Objekte, die das Fenster selbst anbietet.
func (e *editor) einfuegenStarten() {
	switch {
	case e.einf.laeuft:
		e.meldeNebenbei(i18n.T("Reading the clipboard…"), false)
		return
	case !e.s.KannEinfuegen() || e.s.TextLaeuft():
		e.melde(i18n.T("Could not paste: finish the current action first."), false)
		return
	}
	if !e.lesenMoeglich() {
		if e.fensterHaeltKopie && e.einf.kopie != nil {
			e.einfuegenAnwenden(e.einf.kopie)
			return
		}
		e.melde(i18n.T("This desktop does not let Schnapp read the clipboard."), false)
		return
	}
	e.einfuegenLesen(false, func() einfuegeErgebnis {
		g, err := liesZwischenablage(func(typen []string) string {
			if slices.Contains(typen, wlclip.MIMEObjekte) {
				return wlclip.MIMEObjekte
			}
			return ""
		}, maxObjekteDaten)
		r := einfuegeErgebnis{typen: g.Typen, err: err, gelesen: g.Typ != ""}
		if err == nil && r.gelesen {
			r.q, r.err = fileformat.LeseObjekte(bytes.NewReader(g.Daten))
		}
		return r
	})
}

// neuAusZwischenablage ist Datei → „New from clipboard“ (ADR-0045 Punkt 4): Bild lesen, als Aufnahme ablegen, dann
// dorthin wechseln (einfuegenAbholen).
func (e *editor) neuAusZwischenablage() {
	switch grund := e.neuGrau(); {
	case grund != "":
		e.melde(grund, false)
		return
	case e.einf.laeuft:
		e.meldeNebenbei(i18n.T("Reading the clipboard…"), false)
		return
	}
	ablegen := e.bib.Ablegen
	e.einfuegenLesen(true, func() einfuegeErgebnis {
		g, err := liesZwischenablage(func(typen []string) string {
			for _, t := range bildTypen {
				if slices.Contains(typen, t) {
					return t
				}
			}
			return ""
		}, fileformat.MaxDatei)
		r := einfuegeErgebnis{neu: true, typen: g.Typen, err: err, gelesen: g.Typ != ""}
		if err != nil || !r.gelesen {
			return r
		}
		gel, err := fileformat.OeffneBild(g.Daten)
		if err != nil {
			r.err = err
			return r
		}
		r.pfad, r.err = ablegen(gel.Dokument)
		return r
	})
}

// neuGrau ist der Grund, warum „New from clipboard“ nicht geht (leer: geht).
func (e *editor) neuGrau() string {
	switch {
	case e.bib == nil || e.bib.Ablegen == nil || e.bib.Oeffnen == nil:
		return i18n.T("Not available here: no library is connected.")
	case !e.lesenMoeglich():
		return i18n.T("This desktop does not let Schnapp read the clipboard.")
	}
	return ""
}

// lesenMoeglich meldet, ob wlclip.Lies gehen kann (kein WAYLAND_SOCKET, data-control nicht schon als fehlend erkannt).
func (e *editor) lesenMoeglich() bool {
	return wlclip.HalterMoeglich(os.Getenv) == nil && !ohneDataControl.Load()
}

// einfuegenLesen startet lies in einer Goroutine.
func (e *editor) einfuegenLesen(neu bool, lies func() einfuegeErgebnis) {
	e.einf.laeuft = true
	ch := make(chan einfuegeErgebnis, 1)
	e.einf.antwort = ch
	wecker := e.wecker
	e.meldeNebenbei(i18n.T("Reading the clipboard…"), false)
	go func() {
		ch <- lies()
		if wecker != nil {
			wecker.Anfordern()
		}
	}()
}

// einfuegenAbholen nimmt die Antwort des Lesens an (Ereignisschleife).
func (e *editor) einfuegenAbholen() {
	if !e.einf.laeuft {
		return
	}
	var r einfuegeErgebnis
	select {
	case r = <-e.einf.antwort:
	default:
		return
	}
	e.einf.laeuft, e.einf.antwort = false, nil
	if errors.Is(r.err, wlclip.ErrKeinDataControl) {
		ohneDataControl.Store(true)
	}
	switch {
	case errors.Is(r.err, wlclip.ErrLeer):
		e.melde(i18n.T("The clipboard is empty."), false)
	case r.err != nil:
		diagnoseEditor("Zwischenablage: %v", r.err)
		e.melde(einfuegeFehler(r.err, r.neu), true)
	case !r.gelesen:
		e.melde(nichtsGelesen(r.typen, r.neu), false)
	case r.neu:
		if e.wechselAnfrage(r.pfad) {
			e.wechsel.ausZwischenablage = true // nach wechselAnfrage: die setzt den Zustand neu
			return                             // die Fußzeile sagt „Opening…“ bzw. „Saving…“
		}
		e.melde(i18n.T("The image is saved in the library as a new capture."), false)
	default:
		e.einfuegenAnwenden(r.q)
	}
}

// einfuegenAnwenden fügt q ein (canvas.Einfuegen mit dem Budget des Öffnens) und meldet das Ergebnis.
func (e *editor) einfuegenAnwenden(q *document.Dokument) {
	ids, err := e.s.Einfuegen(q, render.Pruefe)
	switch {
	case errors.Is(err, render.ErrZuAufwendig):
		e.melde(i18n.T("Could not paste: the objects would take too long to draw."), true)
	case errors.Is(err, canvas.ErrNichtJetzt):
		e.melde(i18n.T("Could not paste: finish the current action first."), false)
	case errors.Is(err, canvas.ErrNichtsEinzufuegen):
		e.melde(i18n.T("The clipboard holds nothing to paste."), false)
	case err != nil:
		diagnoseEditor("Einfügen: %v", err)
		e.melde(fmt.Sprintf(i18n.T("Could not paste: %v"), ohnePaketpraefix(err.Error())), true)
	default:
		e.melde(fmt.Sprintf(i18n.N("Pasted %d object", "Pasted %d objects", len(ids)), len(ids)), false)
	}
}

// nichtsGelesen ist die Meldung, wenn kein passender Typ da war.
func nichtsGelesen(typen []string, neu bool) string {
	bild := slices.ContainsFunc(typen, func(t string) bool { return slices.Contains(bildTypen, t) })
	switch {
	case neu:
		return i18n.T("The clipboard holds no image.")
	case bild:
		return i18n.T("The clipboard holds an image. File → New from clipboard (Ctrl+Shift+V) opens it as a new capture.")
	}
	return i18n.T("The clipboard holds nothing to paste.")
}

// einfuegeFehler übersetzt einen Fehler des Lesens.
func einfuegeFehler(err error, neu bool) string {
	switch {
	case errors.Is(err, wlclip.ErrKeinDataControl), errors.Is(err, wlclip.ErrWaylandSocket):
		return i18n.T("This desktop does not let Schnapp read the clipboard.")
	case errors.Is(err, wlclip.ErrZuGross), errors.Is(err, fileformat.ErrZuGross), errors.Is(err, document.ErrBildGroesse):
		return i18n.T("The content of the clipboard is too large.")
	case errors.Is(err, fileformat.ErrBeschaedigt), errors.Is(err, fileformat.ErrKeinPNG):
		if neu {
			return i18n.T("The image in the clipboard could not be read.")
		}
		return i18n.T("The objects in the clipboard could not be read.")
	}
	if neu {
		return fmt.Sprintf(i18n.T("Could not create a capture from the clipboard: %v"), fehlerGrund(err))
	}
	return fmt.Sprintf(i18n.T("Could not read the clipboard: %v"), ohnePaketpraefix(err.Error()))
}
