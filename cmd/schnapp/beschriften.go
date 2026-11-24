// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/Andiministrator/schnapp/internal/annotate"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/render"
	"github.com/Andiministrator/schnapp/internal/saveprogress"
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
// [capture] seam für ein einfaches Bild. Signale fängt es wie der Editor ab (Nachtrag 11, Kritik #202 B-2): Das erste
// bricht vor dem Speichern ab (nichts gespeichert, Exit 1); ein Speichern, das schon läuft, wird fertig — samt neuer
// Vorschau und Entfernen einer ausgewichenen Fassung —, das Ergebnis steht auf stdout, Exit 1.
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
	lage := new(beschriftenLage)
	defer lage.ende()
	var b beschriftung
	unterbrochen := mitBeschriftenWache(lage, fehler, func(ctx context.Context) {
		b, err = beschrifteDatei(ctx, a.Datei, es, orte, kante, a.OhneGrenze, false, lage)
	})
	var f *annotate.Fehler
	switch {
	case unterbrochen && err != nil && lage.phase.Load() == 0:
		fmt.Fprintln(fehler, i18n.T("Interrupted by a signal; nothing was saved."))
		return exitFehler
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
	code = exitOK
	if unterbrochen {
		// Wie --output nach dem Schreiben (ADR-0030 Nachtrag 1): Die Datei trägt den Stand, der Aufruf endet mit Exit 1.
		fmt.Fprintf(fehler, i18n.T("Interrupted by a signal after %s was written completely.")+"\n", b.erg.Path)
		code = exitFehler
	}
	if a.JSON {
		if !utf8.ValidString(b.erg.Path) { // wie gibPfadAus (KA-8): gespeichert ist trotzdem
			fmt.Fprintln(fehler, i18n.T("The path is not valid UTF-8 and cannot be written as JSON."))
			return code
		}
		if c := gibJSONAus(aus, fehler, b.erg); c != exitOK {
			return c
		}
		return code
	}
	gibPfadAus(aus, fehler, false, b.erg.Path, image.Pt(b.erg.Width, b.erg.Height))
	return code
}

// errBeschriftenLesen kennzeichnet einen Fehler beim Lesen der Datei (die Ursache ist mit eingepackt).
var errBeschriftenLesen = errors.New("lesen")

// beschriftung ist das Ergebnis von beschrifteDatei: die Zeile für --json bzw. das MCP-Werkzeug und der übersetzte
// Hinweis des Speicherns (leer: keiner).
type beschriftung struct {
	erg     beschriftet
	hinweis string
}

// beschriftenLage ist der Stand eines Beschriftens für die Signale (Kritik #202 B-2): Phase 0 vor dem Speichern, 1
// speichert, 2 gespeichert (der Aufrufer meldet das Ergebnis noch). Ab Phase 1 steht die Lage in laufendeBeschriftungen,
// bis der Aufrufer ende ruft.
type beschriftenLage struct {
	phase atomic.Int32
	seit  atomic.Int64 // Wechsel in die jetzige Phase, Nanosekunden ab saveprogress.Basis
}

// beschriftenFrist: So lange ohne Fortschritt (internal/saveprogress) hält ein laufendes Speichern weitere Signale noch
// zurück, wie speichernFrist im Editor (ADR-0020 Nachtrag 7). Variable, damit Tests sie verkürzen.
var beschriftenFrist = 15 * time.Second

// laufendeBeschriftungen hält die Lagen der gerade speichernden Beschriftungen (*beschriftenLage → struct{}).
var laufendeBeschriftungen sync.Map

func (l *beschriftenLage) setze(p int32) {
	if l == nil {
		return
	}
	l.seit.Store(int64(saveprogress.Jetzt().Sub(saveprogress.Basis())))
	l.phase.Store(p)
	laufendeBeschriftungen.Store(l, struct{}{})
}

// ende nimmt die Lage aus laufendeBeschriftungen: Weitere Signale hält sie nicht mehr zurück.
func (l *beschriftenLage) ende() { laufendeBeschriftungen.Delete(l) }

// haelt meldet, ob die Lage ein weiteres Signal zurückhält: Sie speichert oder hat gespeichert, und der letzte
// Fortschritt (oder der Phasenwechsel) liegt weniger als beschriftenFrist zurück.
func (l *beschriftenLage) haelt() bool {
	if l.phase.Load() == 0 {
		return false
	}
	letzt := saveprogress.Basis().Add(time.Duration(l.seit.Load()))
	if z := saveprogress.Zuletzt(); z.After(letzt) {
		letzt = z
	}
	return saveprogress.Jetzt().Sub(letzt) < beschriftenFrist
}

// signalHalten meldet, ob irgendein laufendes Beschriften ein weiteres Signal zurückhält (CLI-Wache und MCP-Server).
func signalHalten() bool {
	halten := false
	laufendeBeschriftungen.Range(func(k, _ any) bool {
		if k.(*beschriftenLage).haelt() {
			halten = true
			return false
		}
		return true
	})
	return halten
}

// mitBeschriftenWache fängt SIGTERM, SIGINT und SIGHUP, solange f läuft (geerbtes Ignorieren bleibt, F-443). Das erste
// Signal bricht den ctx von f ab: Vor dem Speichern speichert f dann nicht; ein laufendes Speichern läuft zu Ende.
// Weitere Signale hält die Wache zurück, solange lage speichert und Fortschritt meldet (wie der Editor, ADR-0020
// Nachträge 2 und 7); sonst — vor dem Speichern oder bei einem hängenden Speichern — beendet ein weiteres Signal den
// Prozess sofort mit Exit 1. Zurück kommt, ob ein Signal kam.
func mitBeschriftenWache(lage *beschriftenLage, fehler io.Writer, f func(ctx context.Context)) bool {
	ctx, abbrechen := context.WithCancel(context.Background())
	defer abbrechen()
	var fangen []os.Signal
	for _, s := range signaleDerWache {
		if !signal.Ignored(s) {
			fangen = append(fangen, s)
		}
	}
	kanal := make(chan os.Signal, 4)
	var kamen atomic.Int32
	fertig := make(chan struct{})
	ende := make(chan struct{})
	go func() {
		defer close(ende)
		for {
			select {
			case <-fertig:
				return
			case <-kanal:
				if kamen.Add(1) == 1 {
					abbrechen()
					continue
				}
				if lage.haelt() {
					continue // abgefangen: das Speichern läuft und kommt voran
				}
				var text string
				switch lage.phase.Load() {
				case 0:
					text = i18n.T("Interrupted again; nothing was saved.")
				case 1:
					text = i18n.T("Interrupted again while saving made no progress; the result may be incomplete.")
				default:
					text = i18n.T("Interrupted again; the file was already saved.")
				}
				meldeKurz(fehler, text) // ein blockiertes stderr hält das Beenden nicht auf (QS-1)
				beendeProzess(exitFehler)
				return
			}
		}
	}()
	if len(fangen) > 0 { // Notify ohne Signale fängt alle
		meldeAn(kanal, fangen...)
	}
	f(ctx)
	if len(fangen) > 0 {
		meldeAb(kanal)
	}
	close(fertig)
	<-ende
	for { // ein Signal, das knapp vor dem Abmelden eintraf, zählt noch
		select {
		case <-kanal:
			kamen.Add(1)
		default:
			return kamen.Load() > 0
		}
	}
}

// beschrifteDatei liest datei, legt die Einträge darauf und speichert wie der Editor in die Bibliothek orte. Alles oder
// nichts: Scheitert ein Eintrag (*annotate.Fehler), läge das Ergebnis über dem Rechenbudget (render.ErrZuAufwendig,
// außer ohneGrenze) oder ist ctx vor dem Speichern abgebrochen, wird nichts gespeichert. Ein laufendes Speichern bricht
// ctx nicht mehr ab. Ein Lesefehler trägt errBeschriftenLesen. lage (nil: keine) bekommt die Phasen für die Signale.
func beschrifteDatei(ctx context.Context, datei string, es []annotate.Eintrag, orte library.Orte, kante document.Kante,
	ohneGrenze, agent bool, lage *beschriftenLage) (beschriftung, error) {
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
	// ablegenIn ohne die Budget-Warnung des Editors („vor dem Schließen“), die Kommandozeile hat ihre eigene (B-7).
	var ziel string
	speichern := ablegenIn(library.Neu(orte), abs, g, stand, func(p string) { ziel = p }, agent)
	erg, err := annotate.Wende(ctx, d, es)
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
	lage.setze(1)
	gesp, err := speichern(d)
	if err != nil {
		return beschriftung{}, err
	}
	lage.setze(2)
	if ohneGrenze {
		if render.Pruefe(d) != nil {
			text := i18n.T("Saved, but the file has too many or too large annotations to be opened again without --ignore-limit.")
			if gesp.Hinweis != "" {
				text = gesp.Hinweis + " " + text
			}
			gesp.Hinweis, gesp.Warnung = text, true
		}
	}
	gr := d.Groesse()
	return beschriftung{erg: beschriftet{Path: ziel, Width: gr.X, Height: gr.Y, Annotations: erg.Beschriftungen,
		Redactions: erg.Schwaerzungen, Warning: gesp.Warnung}, hinweis: gesp.Hinweis}, nil
}

// liesListe liest die Liste aus der Datei spez oder, bei „-“, von stdin. Ungültig: exitAufruf mit Meldung; nicht
// lesbar (fehlt, Ordner, E/A-Fehler): exitFehler (Kritik #202 B-7).
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
	var l *annotate.LeseFehler
	if errors.As(err, &l) {
		fmt.Fprintf(fehler, i18n.T("Cannot read the annotation list %s: %s")+"\n", spez, nutzerFehler(l.Err))
		return nil, exitFehler
	}
	if err != nil {
		fmt.Fprintf(fehler, i18n.T("Invalid annotation list %s: %s")+"\n", spez, err)
		return nil, exitAufruf
	}
	return es, exitOK
}
