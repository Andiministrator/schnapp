// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"unicode/utf8"

	"github.com/Andiministrator/schnapp/internal/annotate"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/render"
)

// eingabe ist stdin für --annotate=- (Tests ersetzen es).
var eingabe io.Reader = os.Stdin

// beschriftet ist das Ergebnis von --annotate mit --json (ADR-0030 Nachtrag 10 Punkt 4).
type beschriftet struct {
	Path        string `json:"path"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Annotations int    `json:"annotations"`
	Redactions  int    `json:"redactions"`
	Warning     bool   `json:"warning"`
}

// beschrifte legt die Liste a.Spez auf a.Datei und speichert wie der Editor (ADR-0030 Nachtrag 10). kante ist
// [capture] seam für ein einfaches Bild.
func beschrifte(a Aufruf, aus, fehler io.Writer, kante document.Kante) int {
	es, code := liesListe(a.Spez, fehler)
	if code != exitOK {
		return code
	}
	orte, err := library.OrteAus(os.Getenv)
	if err != nil {
		fmt.Fprintf(fehler, i18n.T("Cannot annotate %s: %s")+"\n", a.Datei, nutzerFehler(err))
		return exitFehler
	}
	b, err := beschrifteDatei(context.Background(), a.Datei, es, orte, kante, a.OhneGrenze)
	var f *annotate.Fehler
	switch {
	case errors.As(err, &f):
		fmt.Fprintf(fehler, i18n.T("Invalid annotation list %s: %s")+"\n", a.Spez, err)
		return exitAufruf
	case errors.Is(err, errBeschriftenLesen):
		fmt.Fprintf(fehler, i18n.T("Cannot open %s: %s")+"\n", a.Datei, nutzerFehler(err))
		return exitFehler
	case errors.Is(err, render.ErrZuAufwendig):
		fmt.Fprintf(fehler, i18n.T("Cannot annotate %s: the result would have too many or too large annotations to be opened again.")+"\n", a.Datei)
		return exitFehler
	case err != nil:
		fmt.Fprintf(fehler, i18n.T("Cannot annotate %s: %s")+"\n", a.Datei, nutzerFehler(err))
		return exitFehler
	}
	if b.hinweis != "" {
		fmt.Fprintln(fehler, b.hinweis)
	}
	if a.JSON {
		if !utf8.ValidString(b.erg.Path) { // wie gibPfadAus (KA-8): gespeichert ist trotzdem
			fmt.Fprintln(fehler, i18n.T("The path is not valid UTF-8 and cannot be written as JSON."))
			return exitOK
		}
		return gibJSONAus(aus, fehler, b.erg)
	}
	gibPfadAus(aus, fehler, false, b.erg.Path, image.Pt(b.erg.Width, b.erg.Height))
	return exitOK
}

// errBeschriftenLesen kennzeichnet einen Fehler beim Lesen der Datei (die Ursache ist mit eingepackt).
var errBeschriftenLesen = errors.New("lesen")

// beschriftung ist das Ergebnis von beschrifteDatei: die Zeile für --json bzw. das MCP-Werkzeug und der übersetzte
// Hinweis des Speicherns (leer: keiner).
type beschriftung struct {
	erg     beschriftet
	hinweis string
}

// beschrifteDatei liest datei, legt die Einträge darauf und speichert wie der Editor in die Bibliothek orte. Alles oder
// nichts: Scheitert ein Eintrag (*annotate.Fehler), läge das Ergebnis über dem Rechenbudget (render.ErrZuAufwendig,
// außer ohneGrenze) oder ist ctx abgebrochen, wird nichts gespeichert. Ein Lesefehler trägt errBeschriftenLesen.
func beschrifteDatei(ctx context.Context, datei string, es []annotate.Eintrag, orte library.Orte, kante document.Kante,
	ohneGrenze bool) (beschriftung, error) {
	abs, g, stand, err := ladeFuerEditor(datei, ohneGrenze)
	if err != nil {
		return beschriftung{}, errors.Join(errBeschriftenLesen, err)
	}
	d := g.Dokument
	if g.Fremd {
		if err := d.VorgabeKante(kante); err != nil {
			return beschriftung{}, errors.Join(errBeschriftenLesen, err)
		}
	}
	// Die Speicherfunktion entsteht wie im Editor vor der ersten Änderung: Sie merkt sich den Schwärzungsstand der
	// gelesenen Datei und warnt so, wenn eine Fassung mit dem Geschwärzten im Klartext stehen bleibt (F-169, F-221).
	var ziel string
	speichern := speichernMitNachher(library.Neu(orte), abs, g, stand, func(p string) { ziel = p })
	erg, err := annotate.Wende(d, es)
	if err != nil {
		return beschriftung{}, err
	}
	if !ohneGrenze {
		// Wie beim Öffnen (ADR-0020 Nachtrag 3): eine Datei, die der Editor danach ablehnte, wird nicht geschrieben.
		if err := render.Pruefe(d); err != nil {
			return beschriftung{}, err
		}
	}
	if err := ctx.Err(); err != nil {
		return beschriftung{}, err
	}
	gesp, err := speichern(d)
	if err != nil {
		return beschriftung{}, err
	}
	gr := d.Groesse()
	return beschriftung{erg: beschriftet{Path: ziel, Width: gr.X, Height: gr.Y, Annotations: erg.Beschriftungen,
		Redactions: erg.Schwaerzungen, Warning: gesp.Warnung}, hinweis: gesp.Hinweis}, nil
}

// liesListe liest die Liste aus der Datei spez oder, bei „-“, von stdin. Ungültig: exitAufruf mit Meldung.
func liesListe(spez string, fehler io.Writer) ([]annotate.Eintrag, int) {
	r := eingabe
	if spez != "-" {
		f, err := os.Open(spez)
		if err != nil {
			fmt.Fprintf(fehler, i18n.T("Cannot read the annotation list %s: %s")+"\n", spez, nutzerFehler(err))
			return nil, exitFehler
		}
		defer f.Close()
		r = f
	}
	es, err := annotate.Lies(r)
	if err != nil {
		fmt.Fprintf(fehler, i18n.T("Invalid annotation list %s: %s")+"\n", spez, err)
		return nil, exitAufruf
	}
	return es, exitOK
}
