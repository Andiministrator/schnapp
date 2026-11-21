// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"log"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/io/clipboard"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/export"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/portal"
	"github.com/Andiministrator/schnapp/internal/render"
	"github.com/Andiministrator/schnapp/internal/saveprogress"
	"github.com/Andiministrator/schnapp/internal/wlclip"
)

// Weitere Farben laut brand/corporate-design.md.
var (
	farbeFenster    = color.NRGBA{R: 0x1E, G: 0x21, B: 0x24, A: 0xFF} // grau-900
	farbeNebentext  = color.NRGBA{R: 0x92, G: 0x98, B: 0x9F, A: 0xFF} // grau-400
	farbeGruenText  = color.NRGBA{R: 0x86, G: 0xC2, B: 0xA6, A: 0xFF} // gruen-300
	farbeLeisteVoll = color.NRGBA{R: 0x2A, G: 0x2E, B: 0x33, A: 0xFF} // grau-800, deckend
)

// offeneOverlays zählt Overlay-Fenster, die noch nicht zerstört sind (F-82: der Editor öffnet erst danach).
var offeneOverlays atomic.Int32

// zwischenablageFehlt setzt der Gio-Patch (app.ClipboardWriteError, F-177), wenn ein Kopieren mangels Sitz oder
// Datengerät des Compositors still nichts tat. clipboard.WriteCmd hat upstream keinen Rückgabewert, und Gio führt
// den Befehl erst beim Abschluss des Bilds aus; die Ereignisschleife des Editors ersetzt danach die Meldung
// „Copied to clipboard“ durch einen Fehler.
var zwischenablageFehlt atomic.Bool

// ErrOverlayOffen: Nach der Frist waren noch Overlay-Fenster offen; der Editor öffnet dann nicht (F-82).
var ErrOverlayOffen = errors.New("ui: Overlay-Fenster noch offen")

// ErrSpeichernHaengt: Das Speichern meldete eine Frist lang keinen Fortschritt, und der Nutzer hat das Fenster trotzdem
// geschlossen (zweite Anfrage oder weiteres Signal), oder das Zerstören hat nach zerstoerenFrist aufgegeben; die
// Änderungen sind womöglich nicht gesichert (F-283, F-298, ADR-0020 Nachträge 7 und 9).
var ErrSpeichernHaengt = errors.New("ui: Speichern antwortet nicht")

// Fristen der Arbeit im Hintergrund (ADR-0020 Nachtrag 7); Variablen, damit Tests sie verkürzen.
var (
	// speichernFrist: Meldet ein Speichern so lange keinen Fortschritt (internal/saveprogress), gilt es als hängend:
	// Die Statusleiste sagt es, und eine weitere Schließen-Anfrage oder ein weiteres Signal schließt ohne seine Antwort
	// (F-283). Ein Speichern mit Fortschritt wird nie aufgegeben (F-298, ADR-0020 Nachtrag 9).
	speichernFrist = 15 * time.Second
	// zerstoerenFrist: So lange wartet der Editor beim Zerstören des Fensters ohne Anfrage (Wayland weg) höchstens auf
	// ein laufendes Speichern — länger als ein Dokument am Budget braucht (rund 45 s, Nachtrag 3). Äußerste Grenze, gilt
	// auch bei Fortschritt (Andi, 21.11.2026, Nachtrag 9).
	zerstoerenFrist = 2 * time.Minute
	// kopierFrist: So lange warten Schließen und Zerstören höchstens auf ein laufendes Kopieren (F-281).
	kopierFrist = 10 * time.Second
)

// fristTimer startet den Timer, mit dem speichernAbwarten die äußerste Grenze zerstoerenFrist überwacht: Kanal der
// Frist und Abbruchfunktion. Variable, damit Tests die Grenze ohne echtes Warten auslösen und die übergebene Restzeit
// prüfen können (F-317); sie trägt keine Entscheidung, nur das Warten.
var fristTimer = func(rest time.Duration) (<-chan time.Time, func()) {
	t := time.NewTimer(rest)
	return t.C, func() { t.Stop() }
}

// Speichern speichert das Dokument (Aufrufer: Bibliothek, ADR-0020). Der Fehlertext erscheint in der Statusleiste und
// muss schon übersetzt sein.
type Speichern func(d *document.Dokument) (Gespeichert, error)

// Gespeichert beschreibt ein gelungenes Speichern, das der Nutzer erfahren muss (F-116).
type Gespeichert struct {
	Datei   string // neuer Dateiname für die Titelleiste (nur Name, kein Ordner); leer = unverändert
	Hinweis string // übersetzter Text für die Statusleiste statt „Saved to the library“; leer = gewöhnlich
	Warnung bool   // Hinweis als Warnung zeigen (z. B. Datei nicht aktualisiert, alter Stand liegt noch)
}

// HaeltOffen meldet, ob das Fenster beim Schließen einmal offen bleiben soll, damit der Hinweis gelesen wird: bei jeder
// Warnung und bei jedem Hinweis, etwa dem Ablageort einer neuen Aufnahme (F-158, ADR-0020 Nachtrag 4). Die nächste
// Schließen-Anfrage schließt (zweistufig: canvas.Sitzung.SchliessenAnfrage, dann SpeichernEnde).
func (g Gespeichert) HaeltOffen() bool { return g.Warnung || g.Hinweis != "" }

// Editor öffnet das Dokument in einem Fenster und kehrt zurück, wenn es geschlossen wird. Ungespeicherte Änderungen
// werden beim Schließen gespeichert (die Bibliothek speichert immer, Kernprinzip 4); scheitert das, bleibt das Fenster
// mit der Meldung offen, und erst eine zweite Schließen-Anfrage verwirft (F-107, ADR-0020 Nachtrag 2). Ein Signal
// (SIGTERM, SIGINT, SIGHUP) speichert und schließt; weitere Signale bis zur Rückkehr werden abgefangen (F-152).
// Rückgabewert ist ein Fehler des Fensters oder des letzten Speicherns ohne Rückfrage. titel ist der Dateiname für die Titelleiste. Muss neben app.Main laufen (Starte).
func Editor(titel string, d *document.Dokument, speichern Speichern) error {
	frist := time.Now().Add(2 * time.Second)
	for offeneOverlays.Load() > 0 {
		if time.Now().After(frist) {
			return ErrOverlayOffen
		}
		time.Sleep(20 * time.Millisecond)
	}
	th := material.NewTheme()
	// Nur die Go-Schriften, wie im Renderer: sonst zeigte der Bildschirm Zeichen aus Systemschriften, die in der
	// gespeicherten Datei als Kästchen erscheinen (Kritik A-3).
	th.Shaper = text.NewShaper(text.NoSystemFonts(), text.WithCollection(gofont.Collection()))
	e := &editor{w: new(app.Window), thema: th, s: canvas.NeueSitzung(d), titel: titel, speichern: speichern}
	app.ClipboardWriteError = func(err error) { // vor dem ersten Fenster (F-177)
		diagnoseEditor("Zwischenablage nicht geschrieben: %v", err)
		zwischenablageFehlt.Store(true)
	}
	e.bildNeu()
	// Wecker statt einzelnem Invalidate: Gio verwirft ein Invalidate nach einem schon angenommenen (F-231).
	e.wecker = canvas.NeuerWecker(e.w.Invalidate, weckTakt, weckGrenze)
	defer e.wecker.Ende()
	ctx, abbruch := context.WithCancel(context.Background())
	defer abbruch() // schließt einen offenen Exportdialog mit dem Fenster
	e.exportCtx = ctx
	e.weitereSignale = make(chan struct{}, 1)
	ende := e.signaleAbfangen()
	defer ende()
	return e.laufen()
}

// signaleAbfangen leitet SIGTERM, SIGINT und SIGHUP (Abmelden, Strg+C im Terminal, kill) in ein geordnetes Schließen
// mit Speichern um (F-107): Die Wache (canvas.Signalwache) setzt nur eine Marke und weckt das Fenster (Invalidate ist
// dafür gedacht); gespeichert und geschlossen wird in der eigenen Ereignisschleife (F-77). Weitere Signale fängt sie ab,
// bis der Editor zurückkehrt — ein zweites Signal bricht das Speichern nicht ab (F-152, ADR-0020 Nachtrag 2). Die
// Rückgabe beendet das Abfangen; danach gilt wieder die Voreinstellung.
func (e *editor) signaleAbfangen() func() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	e.wache = canvas.NeueSignalwache(sig, func() { signal.Stop(sig) }, func(s os.Signal, erstes bool) {
		if !erstes {
			// Hängt ein Speichern länger als die Frist, schließt ein weiteres Signal ohne seine Antwort (F-283); sonst
			// läuft es weiter (F-152). Entschieden wird in der Ereignisschleife.
			diagnoseEditor("weiteres Signal %v: abgefangen", s)
			select {
			case e.weitereSignale <- struct{}{}:
			default:
			}
			e.wecker.Anfordern()
			return
		}
		diagnoseEditor("Signal %v: speichern und schließen", s)
		e.wecker.Anfordern() // die Marke ist gesetzt; wiederholt, bis die Schleife sie sieht (F-231)
	})
	return e.wache.Ende
}

type editor struct {
	w         *app.Window
	thema     *material.Theme
	s         *canvas.Sitzung
	titel     string
	speichern Speichern
	bildOp    paint.ImageOp
	meldung   canvas.Meldung // Zeile in der Statusleiste (Speichern, Hinweis, Fehler); Vorrang regelt canvas.StatusZeile (F-151)
	titelSet  string

	wache *canvas.Signalwache // nimmt Signale an (eigene Goroutine), Ereignisschleife fragt Angekommen
	// wecker weckt die Ereignisschleife für Signal und Dateidialog, bis sie ein Ereignis bearbeitet (F-231).
	wecker *canvas.Wecker
	menue  menue // Menü der Leiste (ADR-0026)
	// bilder zählt die gezeichneten Bilder (Diagnose mit SCHNAPP_DEBUG, Messung F-235).
	bilder int
	// zug sammelt die Bildzeiten des laufenden Zeigerzugs (nur mit SCHNAPP_DEBUG, F-250, internal/ui/bildzeit.go).
	zug zugZeiten
	// Export (ADR-0025): Der Dialog läuft in einer Goroutine, die Antwort kommt über exportAntwort in die Ereignisschleife.
	exportCtx     context.Context
	exportOffen   bool // ein Dialog läuft (nur einer zugleich); nur die Ereignisschleife liest und setzt es
	exportAntwort chan exportErgebnis

	// perform setzt eine Fensteraktion ab; nil heißt e.w.Perform. Naht für Tests ohne Fenster (F-299).
	perform func(system.Action)

	erzwungen bool // Schließen freigegeben (Signal, Speichern zurück): kein ClosingEvent mehr abbrechen
	// schlussErr geht in die Rückgabe von Editor: der Fehler des Speicherns nach einem Signal, ErrSpeichernHaengt nach
	// dem Aufgeben (F-283, wird zurückgenommen, wenn das Speichern doch noch gelingt, F-298) und der Fehler einer
	// Schließen-Anfrage, der erst beim Zerstören ankommt (F-300).
	schlussErr error

	// Speichern und Kopieren im Hintergrund (ADR-0020 Nachtrag 6): Die Goroutine legt die Antwort in den Kanal und
	// weckt die Schleife; angenommen wird nur in der Ereignisschleife. Höchstens je eins zugleich (canvas.Sitzung
	// führt das Speichern, kopierLaeuft das Kopieren).
	speicherAntwort chan speicherErgebnis
	kopierAntwort   chan kopierErgebnis
	kopierLaeuft    bool
	kopierStart     time.Time // Beginn des laufenden Kopierens (Frist beim Schließen, F-281)
	kopierStand     uint64    // Dokumentstand des Abzugs, den das laufende Kopieren ablegt (Meldung, F-285)
	// schliessenNachKopie: Eine Schließen-Anfrage kam während des Kopierens; das Fenster bleibt offen, bis die Antwort
	// da ist oder die Frist um ist, dann fragt es erneut (F-281).
	schliessenNachKopie bool
	// fensterHaeltKopie: Die Zwischenablage trägt das Bild nur über dieses Fenster (kein Halter möglich, Rückfall,
	// ADR-0015 Nachtrag 2); fensterWarnung: Das Schließen wurde deswegen schon einmal angehalten (F-295).
	fensterHaeltKopie, fensterWarnung bool

	speicherStart       time.Time     // Beginn des laufenden Speicherns (F-283)
	haengtGemeldet      bool          // die Statusleiste sagt schon, dass das Speichern nicht antwortet
	speichernAufgegeben bool          // geschlossen ohne Antwort des Speicherns; beim Zerstören nicht mehr warten
	weitereSignale      chan struct{} // jedes weitere Signal nach dem ersten (Wach-Goroutine), F-283

	knoepfe struct {
		auswahl, pfeil, kasten, text, nummer, farbe  widget.Clickable
		schwaerzen, verpixeln, zuschneiden, stauchen widget.Clickable
		rueck, wieder, einpassen, eins, speichern    widget.Clickable
		export, ganz                                 widget.Clickable
	}

	// Doppelklick erkennen (ADR-0021: Text bearbeiten): Zeit und Ort des letzten Drucks der linken Taste.
	bildStand    uint64 // canvas.Sitzung.BildAenderung beim letzten bildNeu (Schwärzen ändert die Pixel)
	letzterDruck time.Duration
	letzterOrt   document.Punkt
	hatDruck     bool

	// naht hält die Zackenkanten als aufgezeichnete Ops (Ansichtskoordinaten), bis sich Ansichtsgröße oder Nähte
	// ändern (F-183): Bei vielen Nähten in einem breiten Bild baute der Editor sonst jedes Bild alle Pfade neu.
	naht struct {
		groesse image.Point
		naehte  []document.Naht
		ops     op.Ops
		aufruf  op.CallOp
		gueltig bool
	}
}

// Doppelklick: zweiter Druck innerhalb dieser Zeit und dieses Abstands (Fensterpixel).
const (
	doppelklickZeit    = 400 * time.Millisecond
	doppelklickAbstand = 6.0
)

// Takt und Obergrenze des Weckers (F-231): Er weckt das Fenster im Takt, bis es ein Ereignis bearbeitet; spätestens
// nach der Obergrenze gibt er auf.
const (
	weckTakt   = 50 * time.Millisecond
	weckGrenze = time.Minute
)

// diagnoseEditor schreibt nur mit gesetztem SCHNAPP_DEBUG (echte Läufe).
func diagnoseEditor(format string, args ...any) {
	if os.Getenv("SCHNAPP_DEBUG") != "" {
		log.Printf("[editor] "+format, args...)
	}
}

func (e *editor) bildNeu() { e.bildOp = paint.NewImageOp(e.s.D.Ansichtsbild()) }

// quittiere meldet dem Wecker, dass die Schleife ein Ereignis bearbeitet, das ein Bild trägt. Ab einem FrameEvent sieht
// die Schleife alles, was vor dem letzten Anfordern abgelegt wurde (Marke, Dialogantwort), und zeigt es im selben Bild;
// der Wecker darf ruhen, eine spätere Anforderung bleibt offen (F-231). Ein Ereignis ohne Bild (Fokuswechsel als
// ConfigEvent) quittiert nicht: Es nähme die Antwort an, aber nichts zeigte sie, und „Dateidialog geöffnet“ bliebe bis
// zum nächsten Ereignis stehen (F-239). Der Wecker weckt dann weiter, bis ein Bild kommt (oder die Obergrenze greift).
func quittiere(w *canvas.Wecker, ereignis event.Event) {
	if _, ok := ereignis.(app.FrameEvent); ok {
		w.Quittieren()
	}
}

func (e *editor) laufen() error {
	e.w.Option(app.Title(e.fensterTitel()), app.Size(unit.Dp(1200), unit.Dp(800)), app.MinSize(unit.Dp(480), unit.Dp(320)))
	if probeAutoWahl {
		// Probe-Build: Vollbild, damit Fensterkoordinaten gleich Bildschirmkoordinaten sind (Fernbedienung, Karte #95).
		e.w.Option(app.Fullscreen.Option())
	}
	e.titelSet = e.fensterTitel()
	var ops op.Ops
	var letzteGroesse image.Point
	for {
		ereignis := e.w.Event()
		quittiere(e.wecker, ereignis)
		switch ev := ereignis.(type) {
		case app.DestroyEvent:
			diagnoseEditor("Fenster zu: Fehler %v, ungesichert %v, Zug %v", ev.Err, e.s.Ungesichert(), e.s.Zieht())
			// Zählung für Messungen wie F-235 (knowledge/messungen/f-235.md)
			diagnoseEditor("Bilder gezeichnet: %d", e.bilder)
			return e.zerstoert(ev.Err)
		case *app.ClosingEvent:
			// Gio fragt vor dem Zerstören (Fensterknopf, Strg+W, Compositor); Abort hält das Fenster offen (F-107).
			if !e.erzwungen && !e.schliessen() {
				ev.Abort()
			}
		case app.FrameEvent:
			e.bilder++
			var bildStart time.Time
			messeZug := e.zug.aktiv // nur mit SCHNAPP_DEBUG während eines Zugs; sonst keine Uhr (F-250)
			if messeZug {
				bildStart = time.Now()
			}
			if ev.Size != letzteGroesse { // probe.sh wartet auf diese Zeile, bis das Fenster steht
				letzteGroesse = ev.Size
				diagnoseEditor("Fenstergröße %dx%d", ev.Size.X, ev.Size.Y)
			}
			gtx := app.NewContext(&ops, ev)
			e.frame(gtx)
			ev.Frame(gtx.Ops)
			if messeZug {
				e.zug.bild(time.Since(bildStart))
			}
			if t := e.fensterTitel(); t != e.titelSet {
				e.titelSet = t
				e.w.Option(app.Title(t))
			}
		}
		// Nach jedem Ereignis nachsehen, ob die Meldung noch gilt: Eine abgelöste kehrt nicht zurück (F-160), auch wenn
		// Ändern und Rückgängig in einem Bild liegen.
		e.meldung.Frisch(e.s.D.Stand(), e.s.Werkzeug)
		if zwischenablageFehlt.Swap(false) {
			e.melde(i18n.T("Could not copy: no clipboard is available."), true) // statt „Copied“ (F-177)
			e.fensterHaeltKopie = false                                         // Gio bietet nichts an: nichts, wovor zu warnen wäre (F-321)
			e.w.Invalidate()
		}
		// Auch nach Ereignissen ohne Bild (Gio sendet bei inaktivem Fenster nur ein Weckereignis) die Dialogantwort
		// annehmen, sonst läge sie bis zum nächsten Bild (F-231); ein Folgebild zeigt sie. Ebenso die Antwort des
		// Speicherns (ADR-0020 Nachtrag 6).
		if e.exportAbholen() {
			e.wecker.Anfordern()
		}
		if e.speichernAbholen() {
			e.wecker.Anfordern()
		}
		if e.wache.Angekommen() && !e.erzwungen && !e.s.SignalVorgemerkt() {
			e.nachSignal()
		}
		if e.weiteresSignalSchliesst() {
			e.schliessenFrei()
		}
		if e.speichernHaengtMelden() {
			e.w.Invalidate()
		}
		if e.nachKopieSchliessen() {
			e.w.Perform(system.ActionClose) // erneut anfragen; schliessen entscheidet wie sonst (F-281)
		}
	}
}

// zerstoert räumt beim Zerstören des Fensters auf und liefert den Rückgabewert von Editor. Läuft noch ein Speichern im
// Hintergrund, wartet es auf seine Antwort (ADR-0020 Nachtrag 6), höchstens zerstoerenFrist lang (Nachtrag 7); ein
// weiterer Versuch nach einem Signal läuft ebenso. Ein laufendes Kopieren bekommt seine Frist, damit der Halter der
// Zwischenablage noch startet (F-281).
func (e *editor) zerstoert(fensterErr error) error {
	e.speichernAbwarten()
	if e.speichernAufgegeben {
		select { // kam die Antwort inzwischen doch, zählt sie (F-298); gewartet wird nicht
		case r := <-e.speicherAntwort:
			e.spaeteAntwort(r)
			e.speichernAbwarten() // ein Neuversuch nach dem Signal (spaeteAntwort) wird abgewartet
		default:
		}
	}
	e.kopierenAbwarten()
	e.s.TextBestaetigen() // getippter Text geht nicht verloren (ADR-0021 Punkt 6)
	if e.s.Zieht() {
		e.s.Abbrechen()
	}
	// Ein geordnetes Schließen hat schon gespeichert (ClosingEvent). Endet das Fenster ohne Anfrage (Wayland weg),
	// liegt das Dokument noch im Speicher: speichern (Kritik A-4). Nach einem gescheiterten Speichern beim Schließen
	// hat der Nutzer das Verwerfen mit der zweiten Anfrage bestätigt. Hängt ein Speichern (aufgegeben), gibt es keinen
	// zweiten Versuch daneben: Die Speicherfunktion führt Zustand und darf nicht zweimal zugleich laufen.
	var err error
	if e.s.Ungesichert() && !e.speichernAufgegeben {
		var g Gespeichert
		g, err = e.speichern(e.s.D)
		diagnoseEditor("Speichern beim Zerstören: %v", err)
		if err == nil {
			e.s.Gespeichert()
			hinweisAufStderr(g)
		}
	}
	return errors.Join(fensterErr, e.schlussErr, err)
}

// hinweisAufStderr: Das Fenster ist (gleich) zu, die Statusleiste weg — ein Hinweis (z. B. neue Aufnahme statt der
// alten Datei) geht auf die Fehlerausgabe, damit er nicht still verloren ist (F-116).
func hinweisAufStderr(g Gespeichert) {
	if g.Hinweis != "" {
		fmt.Fprintln(os.Stderr, g.Hinweis)
	}
}

// schliessen beantwortet eine Schließen-Anfrage; false hält das Fenster offen (Entscheidung in canvas.Sitzung). Ist
// etwas zu speichern, läuft es im Hintergrund (ADR-0020 Nachtrag 6): Das Fenster bleibt offen und zeichnet weiter, die
// Antwort entscheidet in speichernFertig wie bisher — zu; einmal offen mit Hinweis; offen mit Fehler, und die nächste
// Anfrage verwirft. Hängt ein Speichern über die Frist und sagt die Statusleiste das schon, schließt die Anfrage ohne
// seine Antwort (F-283, ADR-0020 Nachtrag 7). Wäre zu schließen, läuft aber noch ein Kopieren, bleibt das Fenster bis
// zu dessen Antwort offen (höchstens kopierFrist), sonst startete der Halter der Zwischenablage nie (F-281).
func (e *editor) schliessen() bool {
	if e.haengtGemeldet && e.speichernHaengt() {
		e.speichernAufgeben()
		return true
	}
	z, abzug := e.s.SchliessenAnfrage()
	diagnoseEditor("Schließen-Anfrage: %v, Speichern beginnt %v", z, abzug != nil)
	switch z {
	case canvas.SchlussZu:
		if e.kopierenHaeltOffen() {
			return false
		}
		if e.fensterWarnen() {
			return false
		}
		return true
	case canvas.SchlussFehler:
		e.melde(i18n.T("Could not save. Close again to discard the changes."), true)
	}
	if abzug != nil {
		e.speichernStarten(abzug)
	}
	if e.w != nil { // ohne Fenster nur im Test
		e.w.Invalidate() // Meldung zeigen
	}
	return false
}

// fensterWarnen hält das Schließen einmal an, wenn nur dieses Fenster das kopierte Bild trägt (kein Halter, Rückfall,
// ADR-0015 Nachtrag 2): Ohne Halter verschwindet das Bild mit dem Fenster aus der Zwischenablage; das einmal sagen,
// statt es nur auf der Fehlerausgabe zu verlieren (F-295). Das nächste Schließen schließt. Gilt für jeden Weg, der
// ein Schließen ausführt, auch den, auf dem vorher gespeichert wurde (F-321). true: Das Fenster bleibt offen.
func (e *editor) fensterWarnen() bool {
	if !e.fensterHaeltKopie || e.fensterWarnung {
		return false
	}
	e.fensterWarnung = true
	e.melde(i18n.T("The clipboard holds the image only while this window is open. Close again to discard it."), true)
	if e.w != nil {
		e.w.Invalidate()
	}
	return true
}

// nachSignal nimmt ein Signal an: Es speichert im Hintergrund und schließt das Fenster danach in jedem Fall, ohne es
// offen zu halten — auch nach einem vorher gescheiterten Speichern (canvas.Sitzung.SignalAnfrage). Weitere Signale
// fängt die Wache ab (F-152).
func (e *editor) nachSignal() {
	zu, abzug := e.s.SignalAnfrage()
	diagnoseEditor("Signal: zu %v, Speichern beginnt %v", zu, abzug != nil)
	if abzug != nil {
		e.speichernStarten(abzug)
		return
	}
	if zu {
		e.schliessenFrei()
	}
	// sonst läuft schon ein Speichern; speichernFertig schließt danach
}

// schliessenFrei gibt das Fenster zum Schließen frei und schließt es in der eigenen Schleife (F-77).
func (e *editor) schliessenFrei() {
	e.erzwungen = true
	if e.perform != nil { // Naht für Tests ohne Fenster
		e.perform(system.ActionClose)
		return
	}
	e.w.Perform(system.ActionClose)
}

// speicherErgebnis ist die Antwort eines Speicherns im Hintergrund.
type speicherErgebnis struct {
	g   Gespeichert
	err error
}

// speichernStarten speichert den Abzug in einer Goroutine (ADR-0020 Nachtrag 6). Die Speicherfunktion führt Zustand;
// canvas.Sitzung lässt höchstens ein Speichern zugleich zu. Die Antwort holt die Ereignisschleife ab.
func (e *editor) speichernStarten(abzug *document.Dokument) {
	e.melde(i18n.T("Saving…"), false)
	if e.speicherAntwort == nil {
		e.speicherAntwort = make(chan speicherErgebnis, 1)
	}
	speichern, ch, wecker := e.speichern, e.speicherAntwort, e.wecker
	start := saveprogress.Jetzt()
	e.speicherStart, e.haengtGemeldet = start, false
	fertig := make(chan struct{})
	if wecker != nil {
		// Zur Frist und danach regelmäßig nachsehen, ob es hängt (speichernHaengtMelden): Mit Fortschritt verschiebt sich
		// der Zeitpunkt, an dem es als hängend gilt (F-298).
		frist := speichernFrist
		var pruefe func()
		pruefe = func() {
			select {
			case <-fertig:
				return
			default:
			}
			wecker.Anfordern()
			time.AfterFunc(max(frist/4, time.Millisecond), pruefe)
		}
		time.AfterFunc(frist, pruefe)
	}
	go func() {
		g, err := speichern(abzug)
		diagnoseEditor("Speichern im Hintergrund: %v, %v", saveprogress.Jetzt().Sub(start).Round(time.Millisecond), err)
		close(fertig)
		ch <- speicherErgebnis{g, err}
		if wecker != nil {
			wecker.Anfordern() // erst nach dem Ablegen der Antwort (F-231)
		}
	}()
}

// speichernAbholen nimmt die Antwort des Speicherns an, falls eine da ist (Ereignisschleife).
func (e *editor) speichernAbholen() bool {
	select {
	case r := <-e.speicherAntwort:
		if e.speichernAufgegeben {
			e.spaeteAntwort(r) // das Fenster schließt schon (F-298)
			return true
		}
		e.speichernFertig(r)
		return true
	default:
		return false
	}
}

// speichernAbwarten wartet beim Zerstören des Fensters auf ein laufendes Speichern; ein weiterer Versuch nach einem
// Signal läuft ebenso im Hintergrund und wird abgewartet. Gewartet wird höchstens zerstoerenFrist ab Beginn des
// jeweiligen Speicherns, auch bei Fortschritt; ein weiteres Signal beendet das Warten nur, wenn das Speichern seit
// speichernFrist keinen Fortschritt gemeldet hat (ADR-0020 Nachträge 7 und 9).
func (e *editor) speichernAbwarten() {
	for e.s.SpeichernLaeuft() && !e.speichernAufgegeben {
		fristC, fristStop := fristTimer(max(0, e.speicherStart.Add(zerstoerenFrist).Sub(saveprogress.Jetzt())))
		select {
		case r := <-e.speicherAntwort:
			fristStop()
			ende := e.s.SpeichernEnde(r.g.HaeltOffen(), r.err)
			e.speichernMelden(r, ende)
			// Eine Schließen-Anfrage, deren Fehler erst jetzt ankommt, hat niemand gesehen: Die Statusleiste verschwindet
			// mit dem Fenster, und der Stand gilt schon als verworfen. Also in die Rückgabe, nicht still verlieren (F-300).
			// Anders in speichernFertig: Dort zeigt ein Bild den Fehler, und die nächste Anfrage verwirft.
			if ende.Schluss == canvas.SchlussFehler && e.schlussErr == nil {
				e.schlussErr = r.err
			}
			if ende.Erneut != nil {
				e.speichernStarten(ende.Erneut)
			}
		case <-e.weitereSignale:
			fristStop()
			if e.speichernHaengt() {
				e.speichernAufgeben()
			}
		case <-fristC:
			e.speichernAufgeben()
		}
	}
}

// speichernHaengt meldet, ob das laufende Speichern seit speichernFrist keinen Fortschritt gemeldet hat (F-298, ADR-0020
// Nachtrag 9): gerechnet ab dem letzten Fortschritt aus internal/saveprogress, frühestens ab dem Beginn dieses
// Speicherns.
func (e *editor) speichernHaengt() bool {
	if !e.s.SpeichernLaeuft() {
		return false
	}
	letzter := e.speicherStart
	if t := saveprogress.Zuletzt(); t.After(letzter) {
		letzter = t
	}
	return saveprogress.Jetzt().Sub(letzter) >= speichernFrist
}

// speichernHaengtMelden sagt in der Statusleiste, dass das Speichern nicht antwortet und das Schließen jetzt ohne seine
// Antwort geht (F-283); einmal je Stillstand. Kommt danach wieder Fortschritt, nimmt es die Meldung zurück, und die
// nächste Anfrage hält das Fenster wieder offen (F-298). Liefert, ob sich die Meldung geändert hat.
func (e *editor) speichernHaengtMelden() bool {
	if e.haengtGemeldet && e.s.SpeichernLaeuft() && !e.speichernHaengt() {
		e.haengtGemeldet = false
		e.melde(i18n.T("Saving…"), false)
		return true
	}
	if e.haengtGemeldet || !e.speichernHaengt() {
		return false
	}
	e.haengtGemeldet = true
	e.melde(i18n.T("Saving does not respond. Closing the window now quits without waiting; changes may be lost."), true)
	return true
}

// speichernAufgeben schließt ohne die Antwort des Speicherns (F-283): Die Goroutine läuft weiter, bis der Prozess
// endet; Editor liefert ErrSpeichernHaengt, damit Fehlerausgabe und Exit-Code es sagen — es sei denn, die Antwort kommt
// vor der Rückkehr doch noch (spaeteAntwort, F-298).
func (e *editor) speichernAufgeben() {
	diagnoseEditor("Speichern antwortet seit %v nicht: aufgegeben", saveprogress.Jetzt().Sub(e.speicherStart).Round(time.Millisecond))
	e.speichernAufgegeben, e.erzwungen = true, true
	if e.schlussErr == nil {
		e.schlussErr = ErrSpeichernHaengt
	}
}

// spaeteAntwort nimmt die Antwort eines schon aufgegebenen Speicherns an (F-298): Gelang es doch, gilt der Stand als
// gesichert und ErrSpeichernHaengt steht nicht mehr in der Rückgabe; scheiterte es, steht dort sein Fehler. Scheiterte
// ein Strg+S oder Schließen, während ein Signal vorgemerkt war, gilt wie sonst der Neuversuch nach dem Signal
// (ADR-0020 Nachtrag 2): Er startet, das Aufgeben ist aufgehoben, und zerstoert wartet ihn mit den üblichen Fristen
// ab (Verify F-298 B-3). Ein Fehler, der schon vor dem Aufgeben in schlussErr stand, bleibt stehen.
func (e *editor) spaeteAntwort(r speicherErgebnis) {
	ende := e.s.SpeichernEnde(r.g.HaeltOffen(), r.err)
	diagnoseEditor("Speichern nach dem Aufgeben zurück: %v, Ende %+v", r.err, ende)
	if r.err == nil {
		hinweisAufStderr(r.g)
	}
	haengt := errors.Is(e.schlussErr, ErrSpeichernHaengt)
	if ende.Erneut != nil {
		if haengt {
			e.schlussErr = nil
		}
		e.speichernAufgegeben = false
		e.speichernStarten(ende.Erneut)
		return
	}
	if haengt {
		e.schlussErr = r.err
	}
}

// weiteresSignalSchliesst nimmt ein weiteres Signal an (nach dem ersten): Hängt das Speichern über die Frist, gibt der
// Editor es auf und das Fenster schließt (F-283); sonst bleibt es beim Abfangen (F-152).
func (e *editor) weiteresSignalSchliesst() bool {
	select {
	case <-e.weitereSignale:
	default:
		return false
	}
	if !e.speichernHaengt() || e.erzwungen {
		return false
	}
	e.speichernAufgeben()
	return true
}

// speichernFertig wertet die Antwort aus (canvas.Sitzung.SpeichernEnde): Meldung, weiterer Versuch nach einem Signal,
// Schließen.
func (e *editor) speichernFertig(r speicherErgebnis) {
	ende := e.s.SpeichernEnde(r.g.HaeltOffen(), r.err)
	diagnoseEditor("Speichern zurück: %v, Ende %+v", r.err, ende)
	e.speichernMelden(r, ende)
	switch {
	case ende.Erneut != nil:
		e.speichernStarten(ende.Erneut)
	case ende.Schliessen:
		// Eine Schließen-Anfrage mit Speichern vorher warnt wie eine ohne (F-321); ein Signal schließt in jedem Fall.
		if !e.s.SignalVorgemerkt() && e.fensterWarnen() {
			return
		}
		e.schliessenFrei()
	}
}

// speichernMelden zeigt das Ergebnis in Titel- und Statusleiste. Schließt das Fenster gleich oder bleibt es für einen
// Hinweis offen, geht der Hinweis zusätzlich auf die Fehlerausgabe (F-116, ADR-0020 Nachtrag 4).
func (e *editor) speichernMelden(r speicherErgebnis, ende canvas.Ende) {
	switch {
	case r.err == nil:
		e.gespeichertMelden(r.g)
		if ende.Schliessen || ende.Schluss == canvas.SchlussHinweis {
			hinweisAufStderr(r.g)
		}
	case ende.SignalFehler:
		e.schlussErr = r.err
	case ende.Schluss == canvas.SchlussFehler:
		e.melde(fmt.Sprintf(i18n.T("Could not save: %v. Close again to discard the changes."), r.err), true)
	case ende.Erneut == nil:
		e.melde(fmt.Sprintf(i18n.T("Could not save: %v"), r.err), true)
	}
}

func (e *editor) fensterTitel() string {
	t := e.titel + " — Schnapp"
	if e.s.Geaendert() {
		t = "• " + t
	}
	return t
}

// speichernJetzt speichert auf Strg+S im Hintergrund (ADR-0020 Nachtrag 6); ein getippter Text kommt mit (ADR-0021
// Punkt 6), während eines Ziehens oder eines anderen Speicherns geschieht nichts.
func (e *editor) speichernJetzt() {
	abzug, err := e.s.SpeichernJetzt()
	switch {
	case errors.Is(err, canvas.ErrNichtsZuSpeichern):
		// Nichts zu schreiben (F-109): sonst würde die Datei neu geschrieben und als bearbeitet markiert.
		e.melde(i18n.T("No changes to save."), false)
	case err != nil:
		diagnoseEditor("Strg+S: %v", err)
	default:
		e.speichernStarten(abzug)
	}
}

// kopierErgebnis ist die Antwort eines Kopierens im Hintergrund: daten nur, wenn das Fenster selbst anbieten muss.
type kopierErgebnis struct {
	daten []byte
	err   error
}

// kopieren legt das fertig gerenderte Bild als PNG (ohne Bearbeitungsdaten) in die Zwischenablage (Karte #39). Rendern,
// Kodieren und die Übergabe an den Halter laufen auf einem Abzug in einer Goroutine (F-178, ADR-0020 Nachtrag 6). Das
// Bild übernimmt ein abgelöster Halter-Prozess über das data-control-Protokoll, damit es das Schließen des Editors
// überdauert (ADR-0024, E-19). Bietet der Compositor das nicht an oder meldet sich der Halter nicht, bietet das Fenster
// selbst an wie bisher (ADR-0015 Nachtrag 2), in der Ereignisschleife (kopierenAbholen) — dann gehört der Inhalt dem
// Fenster. Kann auch Gio das Bild nicht anbieten, ersetzt die Ereignisschleife die Meldung (zwischenablageFehlt, F-177).
// Kopieren geht auch während eines Speicherns (F-285): Es ändert nichts am Dokument, und während des Speicherns läuft
// kein Zug. Schließen und Zerstören warten auf ein laufendes Kopieren (F-281).
func (e *editor) kopieren(gtx layout.Context) {
	if e.kopierLaeuft {
		e.melde(i18n.T("Copying is already in progress."), false)
		return
	}
	abzug, err := e.s.D.Abzug()
	if err != nil {
		e.melde(i18n.T("Could not copy: finish the current action first."), true)
		return
	}
	e.kopierLaeuft, e.kopierStart, e.kopierStand = true, time.Now(), e.s.D.Stand()
	e.melde(i18n.T("Copying…"), false) // ein 4K-Bild braucht spürbar; ohne Hinweis wirkte Strg+C wie ohne Wirkung (F-285)
	if e.kopierAntwort == nil {
		e.kopierAntwort = make(chan kopierErgebnis, 1)
	}
	ch, wecker := e.kopierAntwort, e.wecker
	go func() {
		var r kopierErgebnis
		daten, err := render.PNG(abzug)
		switch {
		case err != nil:
			r.err = err
		default:
			if err := halteZwischenablage(daten); err != nil {
				diagnoseEditor("Halter der Zwischenablage nicht gestartet, Fenster bietet selbst an: %v", err)
				r.daten = daten
			}
		}
		ch <- r
		if wecker != nil {
			wecker.Anfordern()
		}
	}()
}

// kopierenAbholen nimmt die Antwort des Kopierens an (zu Beginn eines Bilds, denn der Rückfall braucht gtx).
func (e *editor) kopierenAbholen(gtx layout.Context) {
	select {
	case r := <-e.kopierAntwort:
		e.kopierLaeuft = false
		if r.err != nil {
			e.melde(fmt.Sprintf(i18n.T("Could not copy: %v"), r.err), true)
			return
		}
		e.fensterHaeltKopie, e.fensterWarnung = r.daten != nil, false
		if r.daten != nil {
			gtx.Execute(clipboard.WriteCmd{Type: wlclip.MIME, Data: io.NopCloser(bytes.NewReader(r.daten))})
		}
		// Die Zwischenablage trägt den Stand beim Tastendruck. Wurde seitdem bearbeitet, sagt die Meldung das, statt
		// den Eindruck zu wecken, das jetzt Sichtbare (etwa eine neue Schwärzung) sei kopiert (F-285).
		if e.s.D.Stand() == e.kopierStand {
			e.melde(i18n.T("Copied to clipboard"), false)
		} else {
			e.melde(i18n.T("Copied to clipboard as it was when copying began; later changes are not included."), false)
		}
	default:
	}
}

// kopierenHaeltOffen entscheidet bei einer Schließen-Anfrage, die schließen dürfte, ob ein laufendes Kopieren das
// Fenster offen hält (F-281): ja bis zur Antwort, höchstens bis kopierFrist ab Beginn. Danach fragt nachKopieSchliessen
// erneut; ist die Frist schon um, schließt die Anfrage, und zerstoert meldet das unfertige Kopieren.
func (e *editor) kopierenHaeltOffen() bool {
	if !e.kopierLaeuft {
		return false
	}
	rest := time.Until(e.kopierStart.Add(kopierFrist))
	if rest <= 0 {
		return false
	}
	if !e.schliessenNachKopie && e.wecker != nil {
		time.AfterFunc(rest, e.wecker.Anfordern) // zur Frist nachsehen, falls die Antwort ausbleibt
	}
	e.schliessenNachKopie = true
	diagnoseEditor("Schließen wartet auf das Kopieren (höchstens %v)", rest.Round(time.Millisecond))
	return true
}

// nachKopieSchliessen meldet, ob eine wegen des Kopierens angehaltene Schließen-Anfrage jetzt erneut zu stellen ist:
// Die Antwort des Kopierens ist angenommen oder die Frist um.
func (e *editor) nachKopieSchliessen() bool {
	if !e.schliessenNachKopie || (e.kopierLaeuft && time.Now().Before(e.kopierStart.Add(kopierFrist))) {
		return false
	}
	e.schliessenNachKopie = false
	return true
}

// kopierenAbwarten wartet beim Zerstören des Fensters auf ein laufendes Kopieren, höchstens bis kopierFrist ab Beginn
// (F-281): Der Halter der Zwischenablage startet in der Goroutine; endete der Prozess vorher, bliebe die Zwischenablage
// still leer. Was das Fenster selbst hätte anbieten müssen (kein Halter möglich), geht mit dem Fenster verloren; das
// steht wie eine abgelaufene Frist auf der Fehlerausgabe, denn die Statusleiste ist weg.
func (e *editor) kopierenAbwarten() {
	if !e.kopierLaeuft {
		return
	}
	frist := time.NewTimer(max(0, time.Until(e.kopierStart.Add(kopierFrist))))
	defer frist.Stop()
	var r kopierErgebnis
	select {
	case r = <-e.kopierAntwort:
	case <-frist.C:
		select { // Antwort und Frist zugleich: die Antwort gilt
		case r = <-e.kopierAntwort:
		default:
			fmt.Fprintln(os.Stderr, i18n.T("Copying was not finished when the window closed; the clipboard may be empty."))
			return
		}
	}
	e.kopierLaeuft = false
	switch {
	case r.err != nil:
		fmt.Fprintf(os.Stderr, i18n.T("Could not copy: %v")+"\n", r.err)
	case r.daten != nil:
		fmt.Fprintln(os.Stderr, i18n.T("Copying was not finished when the window closed; the clipboard may be empty."))
	}
}

// exportErgebnis ist die Antwort des Dateidialogs auf dem Weg in die Ereignisschleife.
type exportErgebnis struct {
	ergebnis portal.Ergebnis
	err      error
}

// exportStarten öffnet den Dateidialog des Desktop-Portals (ADR-0025). Er läuft in einer eigenen Goroutine, denn die
// Antwort kommt erst nach der Wahl des Nutzers; sie weckt das Fenster, geschrieben wird in der Ereignisschleife
// (exportAbholen), wie bei der Signalwache (F-77). Nur ein Dialog zugleich.
func (e *editor) exportStarten() {
	if e.exportOffen || e.s.Zieht() {
		return
	}
	e.exportOffen = true
	e.meldung = canvas.Meldung{} // sonst verdeckte eine frische Meldung den Hinweis zum offenen Dialog (A-12)
	if e.exportAntwort == nil {
		e.exportAntwort = make(chan exportErgebnis, 1)
	}
	name := strings.TrimSuffix(e.titel, filepath.Ext(e.titel)) + export.PNG.Endung()
	anfrage := portal.Anfrage{
		Titel:     i18n.T("Export"),
		Dateiname: name,
		Filter: []portal.Filter{
			dialogFilter(i18n.T("PNG image"), export.PNG),
			dialogFilter(i18n.T("JPEG image"), export.JPG),
			dialogFilter(i18n.T("SVG image"), export.SVG),
		},
		Aktuell: 0,
	}
	ctx, ch, wecker := e.exportCtx, e.exportAntwort, e.wecker
	go func() {
		var r exportErgebnis
		d, err := portal.Neu()
		if err == nil {
			r.ergebnis, r.err = d.Speichern(ctx, anfrage)
			d.Close()
		} else {
			r.err = err
		}
		ch <- r
		wecker.Anfordern() // erst nach dem Ablegen der Antwort (F-231)
	}()
}

// dialogFilter beschreibt den Filter eines Formats (Glob und MIME-Typ).
func dialogFilter(name string, f export.Format) portal.Filter {
	return portal.Filter{Name: name, Globs: []string{f.Glob()}, MIMEs: []string{f.MIME()}}
}

// exportAbholen nimmt die Antwort des Dialogs an (Ereignisschleife) und schreibt die Datei.
// Rückgabe: ob eine Antwort angenommen wurde.
func (e *editor) exportAbholen() bool {
	select {
	case r := <-e.exportAntwort: // ein nil-Kanal blockiert für immer: dann gilt default
		e.exportOffen = false
		e.exportFertig(r)
		return true
	default:
		return false
	}
}

// exportFertig wertet die Antwort aus: Abbruch tut nichts, Fehler und Erfolg erscheinen in der Statusleiste.
func (e *editor) exportFertig(r exportErgebnis) {
	switch {
	case errors.Is(r.err, portal.ErrAbgebrochen), errors.Is(r.err, context.Canceled):
		return
	case errors.Is(r.err, portal.ErrKeinPortal):
		diagnoseEditor("Export: %v", r.err)
		e.melde(i18n.T("No file dialog available (desktop portal missing)."), true)
		return
	case r.err != nil:
		e.melde(fmt.Sprintf(i18n.T("Could not export: %v"), exportFehlerText(r.err, "")), true)
		return
	}
	pfad, err := portal.URIZuPfad(r.ergebnis.URI)
	if err != nil {
		e.melde(i18n.T("Could not export: only local files can be written."), true)
		return
	}
	f, pfad, angehaengt := export.Waehle(pfad, export.AusIndex(r.ergebnis.Filter))
	// Der Dialog hat nur den Namen ohne Endung bestätigt: Ein vorhandenes Ziel nicht still ersetzen (A-1).
	if err := export.PruefeAngehaengt(pfad, angehaengt); err != nil {
		e.melde(exportFehlerText(err, filepath.Base(pfad)), true)
		return
	}
	// Wie beim Speichern: ein laufender Text gehört mit in die Datei, ein laufendes Ziehen verschiebt den Export (A-5).
	e.s.TextBestaetigen()
	if e.s.Zieht() {
		e.melde(i18n.T("Could not export: finish the current action first."), true)
		return
	}
	schreibe := export.SchreibeAtomar
	if angehaengt {
		schreibe = export.SchreibeOhneErsetzen // Ziel könnte während des Renderns entstehen (Verify N-4)
	}
	err = schreibe(pfad, func(w io.Writer) error { return export.Schreibe(e.s.D, f, w) })
	if err != nil {
		e.melde(fmt.Sprintf(i18n.T("Could not export: %v"), exportFehlerText(err, "")), true)
		return
	}
	e.melde(fmt.Sprintf(i18n.T("Exported to %s"), filepath.Base(pfad)), false)
}

// ohneDataControl merkt sich, dass der Compositor kein data-control anbietet: Weitere Strg+C starten dann keinen
// vergeblichen Halter mehr (Kritik Fable, Befund 6).
var ohneDataControl atomic.Bool

// halteZwischenablage übergibt das Bild an einen Halter-Prozess (dasselbe Programm, wlclip.Uebergib). Mit
// WAYLAND_SOCKET (geerbte Verbindung, etwa in verschachtelten Sitzungen) fände der Halter nicht sicher denselben
// Compositor; dann bietet das Fenster selbst an (Befund 5).
func halteZwischenablage(daten []byte) error {
	if err := wlclip.HalterMoeglich(os.Getenv); err != nil {
		return err
	}
	if ohneDataControl.Load() {
		return wlclip.ErrKeinDataControl
	}
	programm, err := os.Executable()
	if err != nil {
		return err
	}
	pid, manager, err := wlclip.Uebergib(programm, daten, wlclip.Frist)
	if err != nil {
		if errors.Is(err, wlclip.ErrKeinDataControl) {
			ohneDataControl.Store(true)
		}
		return err
	}
	diagnoseEditor("Zwischenablage an Halter übergeben: PID %d, %s, %d Byte, endet bei neuem Inhalt oder nach %v",
		pid, manager, len(daten), wlclip.Haltezeit)
	return nil
}

// gespeichertMelden zeigt ein gelungenes Speichern in Titel- und Statusleiste (F-116).
func (e *editor) gespeichertMelden(g Gespeichert) {
	if g.Datei != "" {
		e.titel = g.Datei
	}
	e.melde(i18n.T("Saved to the library"), false)
	if g.Hinweis != "" {
		e.melde(g.Hinweis, g.Warnung)
	}
}

// melde zeigt text in der Statusleiste, bis das Dokument sich ändert oder (außer bei Fehlern) das Werkzeug wechselt
// (F-151). Ein gelungenes Speichern löst einen früheren Fehler ab.
func (e *editor) melde(text string, fehler bool) {
	e.meldung = canvas.NeueMeldung(text, fehler, e.s.D.Stand(), e.s.Werkzeug)
}

// meldeAbgelehnt zeigt, warum ein Druck mit einem Zeichenwerkzeug nichts angelegt hat (F-154).
func (e *editor) meldeAbgelehnt() {
	switch e.s.Abgelehnt() {
	case canvas.AblehnungObjekte:
		e.melde(e.grenzText(canvas.AblehnungObjekte), true)
	case canvas.AblehnungNummer:
		e.melde(e.grenzText(canvas.AblehnungNummer), true)
	}
}

// grenzText ist die Meldung zu einer Grenze des Dokuments.
func (e *editor) grenzText(a canvas.Ablehnung) string {
	switch a {
	case canvas.AblehnungObjekte:
		return fmt.Sprintf(i18n.T("The limit of %d annotations is reached. Delete some to add more."), document.MaxObjekte)
	case canvas.AblehnungNummer:
		return fmt.Sprintf(i18n.T("Step numbers end at %d; no further number can be placed."), document.MaxNummer)
	}
	return ""
}

func (e *editor) frame(gtx layout.Context) {
	// Die Antwort des Dateidialogs wird zu Beginn des Bildes abgeholt, das die Goroutine per Invalidate angestoßen hat,
	// damit dieses Bild sie schon zeigt. Nach dem Bild wäre es zu spät: Gio verwirft ein zweites Invalidate, bis die
	// Ereignisschlange leer ist (app.Window.mayInvalidate) — „Dateidialog geöffnet“ und der graue Knopf blieben bis
	// zum nächsten Ereignis stehen (F-216). Kam die Antwort erst nach diesem Abholen, weckt der Wecker weiter, bis ein
	// Ereignis sie annimmt (F-231).
	_ = e.exportAbholen()
	_ = e.speichernAbholen() // ebenso die Antwort des Speicherns und des Kopierens (ADR-0020 Nachtrag 6)
	e.kopierenAbholen(gtx)
	e.tasten(gtx)
	e.knoepfeAuswerten(gtx)
	e.menueAuswerten(gtx)
	paint.Fill(gtx.Ops, farbeFenster)
	knoepfe := e.leistenKnoepfe()
	pos, leiste := e.ordneLeiste(gtx, knoepfe) // Höhe folgt dem Umbruch (F-148), nie eine feste Größe
	status := gtx.Dp(28)
	flaeche := image.Rect(0, leiste, gtx.Constraints.Max.X, gtx.Constraints.Max.Y-status)
	if flaeche.Dy() < 1 {
		flaeche.Max.Y = flaeche.Min.Y + 1
	}

	// Leinwand: eigener Ursprung, damit Zeigerpositionen schon Leinwandkoordinaten sind.
	t := op.Offset(flaeche.Min).Push(gtx.Ops)
	g := gtx
	g.Constraints = layout.Exact(flaeche.Size())
	e.leinwand(g)
	t.Pop()

	menueRechts := e.zeichneLeiste(gtx, knoepfe, pos, leiste)
	e.zeichneStatus(gtx, status)
	e.zeichneWerkzeugHinweis(gtx, knoepfe, pos)
	e.zeichneMenue(gtx, menueRechts, gtx.Dp(10)+gtx.Dp(36)+gtx.Dp(4)) // über allem, unter dem Menüknopf
}

// --- Leinwand ---

func (e *editor) leinwand(gtx layout.Context) {
	groesse := gtx.Constraints.Max
	e.s.SetzeFenster(groesse)
	e.zeiger(gtx)
	if n := e.s.BildAenderung(); n != e.bildStand {
		e.bildStand = n
		e.bildNeu() // geschwärzte Pixel sofort zeigen (ADR-0022)
	}

	c := clip.Rect(image.Rectangle{Max: groesse}).Push(gtx.Ops)
	paint.Fill(gtx.Ops, farbeGrund)
	a := e.s.Abb
	af := op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(float32(a.Zoom), float32(a.Zoom))).
		Offset(f32.Pt(float32(a.Versatz.X), float32(a.Versatz.Y)))).Push(gtx.Ops)
	b := e.bildOp
	if a.Zoom >= 2 {
		b.Filter = paint.FilterNearest // einzelne Pixel sichtbar
	}
	// Objekte im selben Ausschnitt wie das Bild: Was außerhalb liegt, fehlt auch in der gespeicherten Datei (Kritik A-7).
	bc := clip.Rect(image.Rectangle{Max: e.s.D.Groesse()}).Push(gtx.Ops)
	b.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	e.zeichneObjekte(gtx)
	e.zeichneNaehte(gtx)
	bc.Pop()
	af.Pop()
	e.zeichneAuswahl(gtx)
	e.zeichneRahmen(gtx)
	e.zeichneEinfuegemarke(gtx)

	switch e.s.Werkzeug {
	case canvas.WerkzeugAuswahl:
		pointer.CursorDefault.Add(gtx.Ops)
	case canvas.WerkzeugText:
		pointer.CursorText.Add(gtx.Ops)
	default:
		pointer.CursorCrosshair.Add(gtx.Ops)
	}
	event.Op(gtx.Ops, e)
	c.Pop()
}

// zeichneRahmen zeigt den gerade gezogenen Rahmen in Fensterpixeln: beim Schwärzen und beim Stauchen-Band (ADR-0022,
// ADR-0023) halbdurchsichtig dunkel, beim Zuschneiden ist das Bild außerhalb des Rahmens abgedunkelt; immer mit hellem
// Rand, damit er auf hellen wie dunklen Bildern sichtbar ist.
func (e *editor) zeichneRahmen(gtx layout.Context) {
	a, b, ok := e.s.Rahmen()
	if !ok {
		return
	}
	rund := func(p document.Punkt) image.Point { return image.Pt(int(math.Round(p.X)), int(math.Round(p.Y))) }
	r := image.Rectangle{Min: rund(a), Max: rund(b)}.Canon()
	dunkel := color.NRGBA{A: 0x99}
	if e.s.Werkzeug == canvas.WerkzeugZuschneiden {
		g := e.s.D.Groesse()
		bild := image.Rectangle{Min: rund(e.s.Abb.ZuFenster(document.Pt(0, 0))),
			Max: rund(e.s.Abb.ZuFenster(document.Pt(float64(g.X), float64(g.Y))))}
		for _, k := range []image.Rectangle{
			{bild.Min, image.Pt(bild.Max.X, r.Min.Y)}, {image.Pt(bild.Min.X, r.Max.Y), bild.Max},
			{image.Pt(bild.Min.X, r.Min.Y), image.Pt(r.Min.X, r.Max.Y)}, {image.Pt(r.Max.X, r.Min.Y), image.Pt(bild.Max.X, r.Max.Y)},
		} {
			if k = k.Canon(); !k.Empty() {
				paint.FillShape(gtx.Ops, dunkel, clip.Rect(k).Op())
			}
		}
	} else {
		paint.FillShape(gtx.Ops, dunkel, clip.Rect(r).Op())
	}
	w := gtx.Dp(1)
	for _, k := range []image.Rectangle{
		{r.Min, image.Pt(r.Max.X, r.Min.Y+w)}, {image.Pt(r.Min.X, r.Max.Y-w), r.Max},
		{r.Min, image.Pt(r.Min.X+w, r.Max.Y)}, {image.Pt(r.Max.X-w, r.Min.Y), r.Max},
	} {
		paint.FillShape(gtx.Ops, color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xCC}, clip.Rect(k).Op())
	}
}

// zeichneEinfuegemarke zeigt während der Texteingabe die Einfügemarke am Ende der letzten Zeile (orange wie der
// Tastaturfokus, Corporate Design §6), in Fensterpixeln. Die Zeilenbreite misst derselbe Textsatz wie zeichneObjekte.
func (e *editor) zeichneEinfuegemarke(gtx layout.Context) {
	t, ok := e.s.TextObjekt()
	if !ok {
		return
	}
	zeilen := strings.Split(t.Inhalt, "\n")
	i := len(zeilen) - 1
	breite := e.textBreite(gtx, document.Zeichenbar(zeilen[i]), float32(t.Schriftgrad))
	ort := e.s.D.ZuAnsicht(t.Ort).Plus(document.Pt(float64(breite), t.Schriftgrad*1.3*float64(i)))
	oben := e.s.Abb.ZuFenster(ort)
	unten := e.s.Abb.ZuFenster(ort.Plus(document.Pt(0, t.Schriftgrad*1.25)))
	x := int(math.Round(oben.X))
	r := image.Rect(x, int(math.Round(oben.Y)), x+gtx.Dp(2), int(math.Round(unten.Y)))
	paint.FillShape(gtx.Ops, farbeAkzent, clip.Rect(r).Op())
}

func (e *editor) zeiger(gtx layout.Context) {
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target:  e,
			Kinds:   pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel | pointer.Scroll,
			ScrollY: pointer.ScrollRange{Min: -1e6, Max: 1e6},
		})
		if !ok {
			return
		}
		// Leiste und Statuszeile sind in diesem Bild schon gezeichnet; ein Folgebild zeigt, was der Zeiger an ihnen
		// ändert (etwa „Rückgängig“ nach einem Klick, der einen Zug abbricht — Echtlauf #92).
		// Gemessen (F-235, 4K-Aufnahme auf 4K-Probebildschirm, 1209 Zeigerereignisse): mit und ohne Folgebild bei Drag
		// gleich viele Bilder (788 gegen 782) und gleiche Bildzeit; Gio fasst die Anforderungen zusammen. Bleibt so.
		pe, ok := ev.(pointer.Event)
		if !(probeOhneFolgebildBeiZug && ok && pe.Kind == pointer.Drag) {
			gtx.Execute(op.InvalidateCmd{})
		}
		if !ok || e.s.SpeichernLaeuft() { // während des Speicherns ruht das Bearbeiten (ADR-0020 Nachtrag 6)
			continue
		}
		p := document.Pt(float64(pe.Position.X), float64(pe.Position.Y))
		e.s.Umschalt = pe.Modifiers.Contain(key.ModShift) // Pfeil 45°, Quadrat (ADR-0021)
		switch pe.Kind {
		case pointer.Press:
			if os.Getenv("SCHNAPP_DEBUG") != "" {
				e.zug.beginne() // Bildzeiten dieses Zugs sammeln (F-250)
			}
			switch {
			case pe.Buttons.Contain(pointer.ButtonTertiary):
				e.s.Druecken(p, canvas.TasteMitte)
			case pe.Buttons.Contain(pointer.ButtonPrimary):
				if e.hatDruck && pe.Time-e.letzterDruck <= doppelklickZeit &&
					p.Minus(e.letzterOrt).Laenge() <= doppelklickAbstand {
					e.hatDruck = false // ein dritter Klick zählt wieder einfach
					e.s.Doppelklick(p)
				} else {
					e.hatDruck, e.letzterDruck, e.letzterOrt = true, pe.Time, p
					e.s.Druecken(p, canvas.TastePrimaer)
				}
			}
			e.meldeAbgelehnt() // Grenze erreicht: sichtbar sagen, warum nichts entstand (F-154)
			e.meldeAnsicht()   // Doppelklick mit Zuschneiden/Stauchen
			// Tastaturfokus auf die Leinwand: Tastenkürzel und getippter Text (key.EditEvent) kommen hier an.
			gtx.Execute(key.FocusCmd{Tag: e})
		case pointer.Drag:
			e.s.Ziehen(p)
		case pointer.Release:
			e.s.Loslassen(p)
			if e.s.NimmVerpixelnErsetzt() {
				e.melde(i18n.T("Too narrow to pixelate; the area was blacked out instead."), false)
			}
			if e.s.NimmRahmenZuKlein() {
				e.melde(i18n.T("The area is too small to redact at this zoom; zoom in or draw a larger area."), false)
			}
			e.meldeAnsicht()
			diagnoseEditor("Loslassen: Auswahl %d, geändert %v, Bilder bisher %d", e.s.Auswahl, e.s.Geaendert(), e.bilder)
			if n, mittel, laengste, ok := e.zug.beende(); ok {
				diagnoseEditor("Zug: %d Bilder, Bildzeit mittel %.2f ms, längste %.2f ms", n, ms(mittel), ms(laengste))
			}
		case pointer.Cancel:
			e.s.Abbrechen()
		case pointer.Scroll:
			// Gio liefert Pixel; ein Rastschritt des Mausrads ergibt je nach System 10–120. Auf Schritte von etwa
			// 10 % umrechnen und begrenzen, damit ein Touchpad nicht springt.
			dy := float64(pe.Scroll.Y) / 40
			if dy != 0 {
				dy = math.Copysign(math.Min(3, math.Max(0.25, math.Abs(dy))), dy)
			}
			e.s.Scrollen(p, dy)
		}
	}
}

// zeichneObjekte zeichnet in Ansichtskoordinaten (die Affine-Abbildung liegt schon auf dem Stapel). Schrift in
// Ansichtspixeln: eigene Metrik mit 1 px je sp.
func (e *editor) zeichneObjekte(gtx layout.Context) {
	d := e.s.D
	an := func(p document.Punkt) f32.Point {
		q := d.ZuAnsicht(p)
		return f32.Pt(float32(q.X), float32(q.Y))
	}
	for _, o := range d.Objekte() {
		if !d.ObjektSichtbar(o) {
			continue // ganz im Verborgenen: wie Renderer und SVG-Export (ADR-0023 Punkt 7)
		}
		switch v := o.(type) {
		case document.Pfeil:
			var pfad clip.Path
			pfad.Begin(gtx.Ops)
			pfad.MoveTo(an(v.A))
			const n = 48
			for i := 1; i <= n; i++ {
				pfad.LineTo(an(v.Punkt(float64(i) / n)))
			}
			paint.FillShape(gtx.Ops, v.Strich.Farbe, clip.Stroke{Path: pfad.End(), Width: float32(v.Strich.Breite)}.Op())
			r := v.RichtungAmEnde()
			gl := math.Max(8, 4*v.Strich.Breite)
			nrm := document.Pt(-r.Y, r.X)
			fuss := v.B.Minus(r.Mal(gl))
			var spitze clip.Path
			spitze.Begin(gtx.Ops)
			spitze.MoveTo(an(v.B))
			spitze.LineTo(an(fuss.Plus(nrm.Mal(gl / 2))))
			spitze.LineTo(an(fuss.Minus(nrm.Mal(gl / 2))))
			spitze.Close()
			paint.FillShape(gtx.Ops, v.Strich.Farbe, clip.Outline{Path: spitze.End()}.Op())
		case document.Kasten:
			min, max := an(v.Bereich.Min), an(v.Bereich.Max)
			r := image.Rect(int(math.Round(float64(min.X))), int(math.Round(float64(min.Y))),
				int(math.Round(float64(max.X))), int(math.Round(float64(max.Y))))
			rr := clip.UniformRRect(r, int(math.Round(v.Eckradius)))
			if v.Fuellung.A > 0 {
				paint.FillShape(gtx.Ops, v.Fuellung, rr.Op(gtx.Ops))
			}
			if v.Strich.Breite > 0 && v.Strich.Farbe.A > 0 {
				paint.FillShape(gtx.Ops, v.Strich.Farbe, clip.Stroke{Path: rr.Path(gtx.Ops), Width: float32(v.Strich.Breite)}.Op())
			}
		case document.Nummer:
			m := an(v.Mitte)
			rad := float32(v.Radius)
			kr := image.Rect(int(math.Round(float64(m.X-rad))), int(math.Round(float64(m.Y-rad))),
				int(math.Round(float64(m.X+rad))), int(math.Round(float64(m.Y+rad))))
			paint.FillShape(gtx.Ops, v.Farbe, clip.Ellipse(kr).Op(gtx.Ops))
			e.textBei(gtx, fmt.Sprint(v.Wert), float32(v.Radius*1.2), m, true, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		case document.Text:
			if v.Hintergrund.A > 0 {
				g := v.Grenzen()
				a, b := an(g.Min), an(g.Max)
				paint.FillShape(gtx.Ops, v.Hintergrund, clip.Rect(image.Rect(int(a.X), int(a.Y), int(b.X), int(b.Y))).Op())
			}
			ort := an(v.Ort)
			for i, zeile := range strings.Split(v.Inhalt, "\n") {
				zeile = document.Zeichenbar(zeile) // wie im Renderer
				p := ort.Add(f32.Pt(0, float32(v.Schriftgrad*1.3*float64(i))))
				e.textBei(gtx, zeile, float32(v.Schriftgrad), p, false, v.Farbe)
			}
		}
	}
}

// zeichneNaehte legt die Zackenkanten der Stauch-Nähte über Bild und Objekte, in Ansichtskoordinaten und mit denselben
// Vielecken wie Renderer und SVG-Export (ADR-0023 Punkt 6). Die Pfade werden nur neu gebaut, wenn sich Ansichtsgröße
// oder Nähte ändern; sonst spielt der Editor die Aufzeichnung ab (F-183, e.naht).
func (e *editor) zeichneNaehte(gtx layout.Context) {
	g, naehte := e.s.D.Groesse(), e.s.D.Naehte()
	c := &e.naht
	if !c.gueltig || c.groesse != g || !slices.Equal(c.naehte, naehte) {
		c.ops.Reset()
		aufnahme := op.Record(&c.ops)
		for _, n := range naehte {
			laenge := g.X
			if !n.Zeilen {
				laenge = g.Y
			}
			for _, band := range []struct {
				halb  float64
				farbe color.NRGBA
			}{{document.ZackenRandHalb, document.ZackenRand}, {document.ZackenLinieHalb, document.ZackenLinie}} {
				var p clip.Path
				p.Begin(&c.ops)
				for i, q := range document.Zackenband(n, laenge, e.s.D.ZackenFaktor(), band.halb) {
					pt := f32.Pt(float32(q.X), float32(q.Y))
					if i == 0 {
						p.MoveTo(pt)
					} else {
						p.LineTo(pt)
					}
				}
				p.Close()
				paint.FillShape(&c.ops, band.farbe, clip.Outline{Path: p.End()}.Op())
			}
		}
		c.aufruf = aufnahme.Stop()
		c.groesse, c.naehte, c.gueltig = g, naehte, true
	}
	c.aufruf.Add(gtx.Ops)
}

// meldeAnsicht zeigt einmal, was das letzte Zuschneiden oder Stauchen zu sagen hat (ADR-0023 Punkte 3 und 7).
func (e *editor) meldeAnsicht() {
	switch e.s.NimmAnsichtMeldung() {
	case canvas.AnsichtVerborgen:
		n := e.s.VerborgeneObjekte()
		e.melde(fmt.Sprintf(i18n.N("%d annotation is now hidden outside the visible area; it returns when the crop or cut is removed.",
			"%d annotations are now hidden outside the visible area; they return when the crop or cut is removed.", n), n), false)
	case canvas.AnsichtLeer:
		e.melde(i18n.T("Nothing of the image would remain; cut out a smaller part."), true)
	case canvas.AnsichtZuviele:
		e.melde(i18n.T("Too many cuts; undo or remove one first."), true)
	case canvas.AnsichtRand:
		e.melde(i18n.T("The part reached the edge and was cropped off; “Whole image” brings it back."), false)
	}
}

// textBreite misst eine Zeile in Ansichtspixeln mit demselben Textsatz wie textBei (normaler Schnitt).
func (e *editor) textBreite(gtx layout.Context, s string, groesse float32) float32 {
	if s == "" {
		return 0
	}
	g := gtx
	g.Metric = unit.Metric{PxPerDp: 1, PxPerSp: 1}
	g.Constraints = layout.Constraints{Max: image.Pt(1<<20, 1<<20)}
	lbl := material.Label(e.thema, unit.Sp(groesse), s)
	lbl.MaxLines = 1
	m := op.Record(gtx.Ops)
	dims := lbl.Layout(g)
	m.Stop() // nur messen, nicht zeichnen
	return float32(dims.Size.X)
}

// textBei setzt Text in Ansichtspixeln: mittig um p (mitte) oder mit der linken oberen Ecke bei p.
func (e *editor) textBei(gtx layout.Context, s string, groesse float32, p f32.Point, mitte bool, c color.NRGBA) {
	g := gtx
	g.Metric = unit.Metric{PxPerDp: 1, PxPerSp: 1}
	g.Constraints = layout.Constraints{Max: image.Pt(1<<20, 1<<20)}
	lbl := material.Label(e.thema, unit.Sp(groesse), s)
	lbl.Color = c
	lbl.MaxLines = 1
	if mitte {
		lbl.Font.Weight = 700 // fett wie im Renderer
	}
	m := op.Record(gtx.Ops)
	dims := lbl.Layout(g)
	call := m.Stop()
	ecke := p
	if mitte {
		ecke = p.Sub(f32.Pt(float32(dims.Size.X)/2, float32(dims.Size.Y)/2))
	}
	t := op.Affine(f32.Affine2D{}.Offset(ecke)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	t.Pop()
}

// zeichneAuswahl zeichnet Rahmen und Griffe des ausgewählten Objekts in Fensterpixeln (unabhängig vom Zoom).
func (e *editor) zeichneAuswahl(gtx layout.Context) {
	o, ok := e.s.D.Objekt(e.s.Auswahl)
	if !ok {
		return
	}
	fe := func(p document.Punkt) image.Point {
		q := e.s.Abb.ZuFenster(e.s.D.ZuAnsicht(p))
		return image.Pt(int(math.Round(q.X)), int(math.Round(q.Y)))
	}
	g := o.Grenzen()
	r := image.Rectangle{Min: fe(g.Min), Max: fe(g.Max)}.Canon()
	rahmen(gtx.Ops, r.Inset(-gtx.Dp(2)), gtx.Dp(1), farbeAuswahl)
	h := gtx.Dp(5)
	for _, gr := range canvas.Griffe(o) {
		m := fe(gr.Ort)
		q := image.Rect(m.X-h, m.Y-h, m.X+h, m.Y+h)
		paint.FillShape(gtx.Ops, farbeText, clip.Rect(q).Op())
		rahmen(gtx.Ops, q, gtx.Dp(1), farbeAuswahl)
	}
}

// --- Tasten ---

// alleTastenFilter sind die Filter der Tastenschleife: die Tasten der Tabelle (tastenFilter), der Fokus der Leinwand und
// die Navigation im Menü (menueNavigationsFilter, ADR-0026 Nachtrag 4). Mehr meldet der Editor nicht an; der Test
// TestTastenfilterNurTabelleUndMenueNavigation hält das fest.
func (e *editor) alleTastenFilter() []event.Filter {
	var filter []event.Filter
	for _, f := range tastenFilter(e.s.TextLaeuft()) {
		filter = append(filter, f)
	}
	filter = append(filter, key.FocusFilter{Target: e})
	for _, f := range menueNavigationsFilter(&e.menue.knopf) {
		filter = append(filter, f)
	}
	return filter
}

func (e *editor) tasten(gtx layout.Context) {
	for {
		// Die Tasten stehen in tastenTabelle (ADR-0026): Jede hat einen Knopf, einen Menüeintrag oder eine begründete
		// Ausnahme. Während der Texteingabe fängt der Editor Tab ab (ohne Wirkung, ADR-0021 Nachtrag 1): Sonst wanderte
		// der Fokus auf einen Knopf, und weitere Zeichen gingen still verloren (F-153). Sonst bleibt Tab die Fokusfahrt.
		// Getippte Zeichen kommen als key.EditEvent an die fokussierte Leinwand.
		filter := e.alleTastenFilter()
		ev, ok := gtx.Event(filter...)
		if !ok {
			return
		}
		if e.s.SpeichernLaeuft() {
			// Während des Speicherns ruht das Bearbeiten (ADR-0020 Nachtrag 6): abholen, damit Filter und Fokus
			// bestehen bleiben, und verwerfen. Nur Strg+C wirkt: Kopieren ändert nichts am Dokument, und während des
			// Speicherns läuft kein Zug, der Abzug gelingt (F-285). Ebenso Strg+W und Enter auf dem Menüeintrag
			// „Schließen“: Sie sind eine Schließen-Anfrage wie der Fensterknopf, die SchliessenAnfrage vormerkt (F-299).
			if k, ok := ev.(key.Event); ok && k.State == key.Press {
				if e.menue.offen {
					if k.Name == key.NameReturn || k.Name == key.NameEnter {
						// Äußerste von drei Sperren (danach an() in menueAuswerten/knoepfeAuswerten und
						// canvas.Sitzung.gesperrt): bewusst doppelt, einzeln nicht prüfbar; die Wirkung prüft
						// TestEnterWaehrendSpeichernAendertNichts über alle drei (F-307).
						if c := e.fokussierterKnopf(gtx); c != nil && c == e.schliessenEintrag() {
							e.knopfPerTaste(c)
						}
					}
				} else if t, ok := findeTaste(k.Name, k.Modifiers, e.s.TextLaeuft()); ok && !t.hatWerkzeug &&
					(t.akt == aktKopieren || t.akt == aktSchliessen) {
					e.fuehreAus(gtx, t.akt, false)
				}
			}
			continue
		}
		if ed, ok := ev.(key.EditEvent); ok {
			// Bei offenem Menü wirkt getippter Text nicht (ADR-0026 Nachtrag 1, F-261); die Tasten des Menüs regelt
			// der Zweig unten.
			if e.s.TextLaeuft() && !e.menue.offen {
				e.s.Tippe(ed.Text)
			}
			continue
		}
		k, ok := ev.(key.Event)
		if !ok || k.State != key.Press {
			continue
		}
		if k.Name == key.NameSpace { // nur am Menüknopf angemeldet; wirkt wie Enter
			if gtx.Focused(&e.menue.knopf) {
				e.menueUmschalten(true)
			}
			continue
		}
		if e.menue.offen {
			// Offenes Menü (ADR-0026 Punkt 1, Nachträge 2 und 3): Esc schließt es, Enter löst einen Knopf oder Eintrag
			// mit Tastaturfokus aus, Pfeile hoch und runter, Pos1 und Ende wählen den Eintrag; die Leertaste am
			// Menüknopf ist oben behandelt, Tab fährt den Fokus wie sonst (Gio). Alle anderen Tasten wirken nicht,
			// damit Entf oder ein Buchstabe nicht unbemerkt hinter der Liste die Leinwand ändert (F-257).
			switch k.Name {
			case key.NameEscape:
				// Fokus zurück an den Menüknopf nur, wenn per Tastatur geöffnet wurde und kein Text läuft; sonst an die
				// Leinwand, sonst gingen getippte Zeichen verloren (F-255 V-1).
				e.menueSchliesse(gtx, e.menue.perTasteOffen && !e.s.TextLaeuft())
			case key.NameUpArrow, key.NameDownArrow, key.NameHome, key.NameEnd:
				e.menueTaste(gtx, k.Name)
			case key.NameReturn, key.NameEnter:
				if c := e.fokussierterKnopf(gtx); c != nil {
					e.knopfPerTaste(c)
				}
			}
			continue
		}
		t, ok := findeTaste(k.Name, k.Modifiers, e.s.TextLaeuft())
		if e.s.TextLaeuft() {
			// Während der Texteingabe sind Buchstaben Text (kommt als EditEvent), keine Kürzel: Werkzeugtasten aus der
			// Tabelle, Entf und Tab wirken nicht (Tab: F-153), die Rücktaste löscht im Text. Die Werkzeugtasten kommen aus der Tabelle,
			// nicht aus einer eigenen Liste (F-256).
			switch {
			case k.Name == key.NameDeleteBackward:
				e.s.TextZurueck()
				continue
			case ok && t.hatWerkzeug, k.Name == key.NameDeleteForward, k.Name == key.NameTab:
				continue
			}
		}
		switch {
		case !ok:
		case t.hatWerkzeug:
			e.s.SetzeWerkzeug(t.werkzeug)
		case t.akt == aktEsc:
			e.s.Esc() // in Stufen: Text verwerfen, Ziehen abbrechen, abwählen, zurück zu Auswählen
		case t.akt == aktEnter:
			// Ohne Texteingabe löst Enter einen per Tastatur fokussierten Knopf aus, statt von der Leinwand
			// verbraucht zu werden (F-153).
			if !e.s.TextLaeuft() {
				if c := e.fokussierterKnopf(gtx); c != nil {
					e.knopfPerTaste(c)
					continue
				}
			}
			e.s.Enter(k.Modifiers.Contain(key.ModShift))
		default:
			// Per Taste wirken Kopieren und Export in der Texteingabe nicht (wie Strg+C bisher).
			e.fuehreAus(gtx, t.akt, false)
		}
	}
}

// fokussierterKnopf liefert den Knopf der Leiste, der gerade den Tastaturfokus hat (nil: keiner).
func (e *editor) fokussierterKnopf(gtx layout.Context) *widget.Clickable {
	k := &e.knoepfe
	alle := []*widget.Clickable{&k.rueck, &k.wieder, &k.ganz, &k.einpassen, &k.eins, &k.export, &k.speichern, &k.farbe,
		&e.menue.knopf}
	if e.menue.offen {
		for i := range e.menueEintraege() {
			alle = append(alle, &e.menue.eintraege[i])
		}
	}
	for _, w := range e.werkzeugKnoepfe() {
		alle = append(alle, w.c)
	}
	for _, c := range alle {
		if gtx.Focused(c) {
			return c
		}
	}
	return nil
}

func (e *editor) rueckgaengig() {
	if e.s.Rueckgaengig() {
		e.bildNeu() // Schwärzen und Zuschnitt ändern das Bild selbst
	}
}

func (e *editor) wiederholen() {
	if e.s.Wiederholen() {
		e.bildNeu()
	}
}

// --- Leiste ---

func (e *editor) knoepfeAuswerten(gtx layout.Context) {
	k := &e.knoepfe
	// Während des Speicherns ruht das Bearbeiten (ADR-0020 Nachtrag 6): Clicked holt die Klicks ab (zuerst
	// ausgewertet), an verwirft sie. an fragt je Klick neu: Liegen mehrere Klicks in einem Bild und startet einer das
	// Speichern, wirken die folgenden nicht mehr (F-280; canvas sperrt zusätzlich).
	an := func() bool { return !e.s.SpeichernLaeuft() }
	if k.rueck.Clicked(gtx) && an() {
		e.rueckgaengig()
	}
	if k.wieder.Clicked(gtx) && an() {
		e.wiederholen()
	}
	if k.einpassen.Clicked(gtx) && an() {
		e.s.Einpassen()
	}
	if k.ganz.Clicked(gtx) && an() {
		e.fuehreAus(gtx, aktGanzesBild, true)
	}
	if k.eins.Clicked(gtx) && an() {
		e.s.EinsZuEins()
	}
	if k.export.Clicked(gtx) && an() {
		e.fuehreAus(gtx, aktExport, true) // wie Speichern: ein laufender Text kommt mit
	}
	if k.speichern.Clicked(gtx) && an() {
		e.speichernJetzt()
	}
	for _, w := range e.werkzeugKnoepfe() {
		if w.c.Clicked(gtx) && an() {
			e.s.SetzeWerkzeug(w.w)
		}
	}
	if k.farbe.Clicked(gtx) && an() {
		e.s.NaechsteFarbe()
	}
}

type werkzeugKnopf struct {
	c    *widget.Clickable
	w    canvas.Werkzeug
	text string
}

// werkzeugKnoepfe sind die Knöpfe der Werkzeuggruppe in Reihenfolge der Leiste (Kürzel V, A, R, T, N, B, P, C, X;
// ADR-0021, ADR-0022, ADR-0023).
func (e *editor) werkzeugKnoepfe() []werkzeugKnopf {
	k := &e.knoepfe
	return []werkzeugKnopf{
		{&k.auswahl, canvas.WerkzeugAuswahl, i18n.T("Select")},
		{&k.pfeil, canvas.WerkzeugPfeil, i18n.T("Arrow")},
		{&k.kasten, canvas.WerkzeugKasten, i18n.T("Rectangle")},
		{&k.text, canvas.WerkzeugText, i18n.T("Text")},
		{&k.nummer, canvas.WerkzeugNummer, i18n.T("Step")},
		{&k.schwaerzen, canvas.WerkzeugSchwaerzen, i18n.T("Redact")},
		{&k.verpixeln, canvas.WerkzeugVerpixeln, i18n.T("Pixelate")},
		{&k.zuschneiden, canvas.WerkzeugZuschneiden, i18n.T("Crop")},
		{&k.stauchen, canvas.WerkzeugStauchen, i18n.T("Cut out")},
	}
}

type knopfArt int

const (
	knopfNormal knopfArt = iota
	knopfAktiv           // aktives Werkzeug: grüner Rand, grüner Text
	knopfHaupt           // Hauptaktion: grüne Fläche
)

// leistenKnopf ist ein Eintrag der Werkzeugleiste links von „Speichern“.
type leistenKnopf struct {
	c      *widget.Clickable
	text   string
	art    knopfArt
	an     bool
	muster color.NRGBA // Farbfeld vor dem Text (A = 0: keins)
	gruppe bool        // beginnt eine neue Gruppe (zusätzlicher Abstand)
}

// leistenKnoepfe liefert die Knöpfe der Leiste in Reihenfolge.
func (e *editor) leistenKnoepfe() []leistenKnopf {
	k := &e.knoepfe
	var links []leistenKnopf
	for _, w := range e.werkzeugKnoepfe() {
		art := knopfNormal
		if e.s.Werkzeug == w.w {
			art = knopfAktiv
		}
		links = append(links, leistenKnopf{c: w.c, text: w.text, art: art, an: true})
	}
	// Gruppen: Werkzeuge | Farbe | Verlauf | Ansicht. Während der Texteingabe sperrt der Zug den Verlauf (F-18).
	return append(links,
		leistenKnopf{c: &k.farbe, text: i18n.T("Color"), an: true, muster: e.s.Stil.Farbe, gruppe: true},
		leistenKnopf{c: &k.rueck, text: i18n.T("Undo"), an: e.s.D.KannRueckgaengig(), gruppe: true},
		leistenKnopf{c: &k.wieder, text: i18n.T("Redo"), an: e.s.D.KannWiederholen()},
		leistenKnopf{c: &k.ganz, text: i18n.T("Whole image"), an: e.s.KannGanzesBild(), gruppe: true},
		leistenKnopf{c: &k.einpassen, text: i18n.T("Fit"), an: true, gruppe: true},
		leistenKnopf{c: &k.eins, text: "1:1", an: true},
		leistenKnopf{c: &k.export, text: i18n.T("Export…"), an: !e.exportOffen, gruppe: true},
	)
}

// messeKnopf liefert die Breite eines Knopfs, ohne etwas zu zeichnen. Der Knopf wird in einen Makro aufgenommen, der
// nie abgespielt wird, und zwar als nicht bedienbarer (an = false): So legt die Messung weder Klickflächen an noch
// verbraucht sie Ereignisse des Clickable, die das echte Zeichnen danach noch braucht. Die Breite hängt nur von Text
// und Farbfeld ab, nicht von an oder art. Zwei Durchgänge mit echtem Zeichnen hätten die Ereignisse doppelt gelesen.
func (e *editor) messeKnopf(gtx layout.Context, c *widget.Clickable, text string, muster color.NRGBA) int {
	m := op.Record(gtx.Ops)
	w := e.knopf(gtx, c, text, knopfNormal, false, image.Point{}, muster)
	m.Stop()
	return w
}

// ordneLeiste misst die Knöpfe und bricht sie um, wenn sie nicht neben „Speichern“ passen (F-148). Liefert die Ecken
// und die Höhe der Leiste in Pixeln; die Leinwand beginnt darunter.
func (e *editor) ordneLeiste(gtx layout.Context, links []leistenKnopf) ([]image.Point, int) {
	kn := make([]canvas.LeistenKnopf, len(links))
	for i, l := range links {
		kn[i] = canvas.LeistenKnopf{Breite: e.messeKnopf(gtx, l.c, l.text, l.muster), Gruppe: l.gruppe}
	}
	m := canvas.LeistenMass{RandX: gtx.Dp(12), RandY: gtx.Dp(10), Abstand: gtx.Dp(8), KnopfHoehe: gtx.Dp(36),
		Speichern: e.rechterBlock(gtx), Breite: gtx.Constraints.Max.X}
	pos, hoehe, _ := canvas.LeisteAnordnen(kn, m)
	return pos, hoehe
}

// rechterBlock ist die Breite rechts in der ersten Zeile: „Menü“, Abstand, „Speichern“ (ADR-0026 Punkt 1).
func (e *editor) rechterBlock(gtx layout.Context) int {
	return e.messeKnopf(gtx, &e.menue.knopf, i18n.T("Menu"), color.NRGBA{}) + gtx.Dp(8) +
		e.messeKnopf(gtx, &e.knoepfe.speichern, i18n.T("Save"), color.NRGBA{})
}

// zeichneLeiste zeichnet die Leiste und liefert die rechte Kante des Menüknopfs (dort schließt die Liste an).
func (e *editor) zeichneLeiste(gtx layout.Context, links []leistenKnopf, pos []image.Point, hoehe int) int {
	breite := gtx.Constraints.Max.X
	paint.FillShape(gtx.Ops, farbeLeisteVoll, clip.Rect(image.Rect(0, 0, breite, hoehe)).Op())
	paint.FillShape(gtx.Ops, farbeLinie, clip.Rect(image.Rect(0, hoehe-gtx.Dp(1), breite, hoehe)).Op())
	for i, kn := range links {
		e.knopf(gtx, kn.c, kn.text, kn.art, kn.an, pos[i], kn.muster)
	}
	// Speichern rechts in der ersten Zeile.
	m := op.Record(gtx.Ops)
	w := e.knopf(gtx, &e.knoepfe.speichern, i18n.T("Save"), knopfHaupt, true, image.Point{}, color.NRGBA{})
	call := m.Stop()
	t := op.Offset(image.Pt(breite-gtx.Dp(12)-w, gtx.Dp(10))).Push(gtx.Ops)
	call.Add(gtx.Ops)
	t.Pop()
	// „Menü“ links neben „Speichern“; offen wie ein aktives Werkzeug hervorgehoben.
	art := knopfNormal
	if e.menue.offen {
		art = knopfAktiv
	}
	rechts := breite - gtx.Dp(12) - w - gtx.Dp(8)
	mb := e.messeKnopf(gtx, &e.menue.knopf, i18n.T("Menu"), color.NRGBA{})
	e.knopf(gtx, &e.menue.knopf, i18n.T("Menu"), art, true, image.Pt(rechts-mb, gtx.Dp(10)), color.NRGBA{})
	return rechts
}

// knopf zeichnet eine Schaltfläche (Radius 6, Höhe 36 dp) bei ecke und liefert ihre Breite. Ein muster mit A > 0
// erscheint als rundes Farbfeld vor dem Text (Knopf „Farbe“; der Text nennt die Handlung, die Farbe ist nicht das
// einzige Merkmal, Corporate Design §3).
func (e *editor) knopf(gtx layout.Context, c *widget.Clickable, s string, art knopfArt, an bool, ecke image.Point,
	muster color.NRGBA) int {
	lbl := material.Label(e.thema, unit.Sp(14), s)
	lbl.Color = farbeText
	switch {
	case !an:
		lbl.Color = farbeNebentext
	case art == knopfAktiv:
		lbl.Color = farbeGruenText
	case art == knopfHaupt:
		lbl.Color = farbeGrund
	}
	g := gtx
	g.Constraints.Min = image.Point{}
	m := op.Record(gtx.Ops)
	dims := lbl.Layout(g)
	textCall := m.Stop()
	innen := image.Pt(gtx.Dp(14), (gtx.Dp(36)-dims.Size.Y)/2)
	groesse := image.Pt(dims.Size.X+2*innen.X, gtx.Dp(36))
	feld := 0 // Breite des Farbfelds samt Abstand
	if muster.A > 0 {
		feld = gtx.Dp(16 + 8)
		groesse.X += feld
	}

	t := op.Offset(ecke).Push(gtx.Ops)
	r := image.Rectangle{Max: groesse}
	rr := clip.UniformRRect(r, gtx.Dp(6))
	switch {
	case art == knopfHaupt && an:
		paint.FillShape(gtx.Ops, farbeAuswahl, rr.Op(gtx.Ops))
	default:
		paint.FillShape(gtx.Ops, farbeGrund, rr.Op(gtx.Ops))
		rand := farbeLinie
		if art == knopfAktiv {
			rand = farbeAuswahl
		}
		paint.FillShape(gtx.Ops, rand, clip.Stroke{Path: rr.Path(gtx.Ops), Width: float32(gtx.Dp(1))}.Op())
	}
	if gtx.Focused(c) { // 2 px Ring mit 2 px Abstand zur Randlinie (corporate-design.md §6; gemessen, ADR-0026 Nachtrag 3)
		paint.FillShape(gtx.Ops, farbeAkzent, clip.Stroke{Path: clip.UniformRRect(r.Inset(-gtx.Dp(4)), gtx.Dp(8)).Path(gtx.Ops),
			Width: float32(gtx.Dp(2))}.Op())
	}
	if feld > 0 {
		d := gtx.Dp(16)
		oy := (groesse.Y - d) / 2
		kr := image.Rect(innen.X, oy, innen.X+d, oy+d)
		paint.FillShape(gtx.Ops, muster, clip.Ellipse(kr).Op(gtx.Ops))
		paint.FillShape(gtx.Ops, farbeText, clip.Stroke{Path: clip.Ellipse(kr).Path(gtx.Ops), Width: float32(gtx.Dp(1))}.Op())
	}
	tt := op.Offset(innen.Add(image.Pt(feld, 0))).Push(gtx.Ops)
	textCall.Add(gtx.Ops)
	tt.Pop()
	if an {
		g.Constraints = layout.Exact(groesse)
		c.Layout(g, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: groesse} })
		pointer.CursorPointer.Add(gtx.Ops)
	}
	t.Pop()
	return groesse.X
}

func (e *editor) zeichneStatus(gtx layout.Context, hoehe int) {
	mx := gtx.Constraints.Max
	paint.FillShape(gtx.Ops, farbeLeisteVoll, clip.Rect(image.Rect(0, mx.Y-hoehe, mx.X, mx.Y)).Op())
	g := e.s.D.Groesse()
	rechts := fmt.Sprintf("%d × %d · %d %%", g.X, g.Y, int(math.Round(e.s.Abb.Zoom*100)))
	rb := e.statusText(gtx, rechts, farbeNebentext, mx.Y-hoehe, false, mx.X)
	// Eine frische Meldung geht vor dem Bedienhinweis des Werkzeugs, danach zeigt die Leiste wieder den Hinweis (F-151).
	links, fehler, meldung := canvas.StatusZeile(&e.meldung, e.s.D.Stand(), e.s.Werkzeug, e.hinweis(), e.s.TextLaeuft())
	c := farbeNebentext
	switch {
	case fehler:
		c = color.NRGBA{R: 0xE6, G: 0x78, B: 0x73, A: 0xFF} // rot-500
	case meldung:
		c = farbeGruenText
	}
	if links != "" {
		// Links bleibt Platz bis vor die Größenangabe; ein längerer Hinweis wird im schmalen Fenster gekürzt, statt
		// über die Größenangabe zu laufen (Echtlauf #92).
		e.statusText(gtx, links, c, mx.Y-hoehe, true, mx.X-rb-gtx.Dp(36))
	}
}

// hinweis ist die Bedienung des aktiven Werkzeugs bzw. der Texteingabe für die Statusleiste (ADR-0021).
func (e *editor) hinweis() string {
	if e.exportOffen {
		return i18n.T("File dialog is open.") // der Dialog kann hinter dem Fenster liegen (A-12)
	}
	if e.s.TextLaeuft() {
		return i18n.T("Type the text. Enter: done, Shift+Enter: new line, Esc: discard.")
	}
	switch e.s.Werkzeug {
	case canvas.WerkzeugPfeil:
		return i18n.T("Drag to draw an arrow. Shift: 45° steps. Drag the middle handle to bend it.")
	case canvas.WerkzeugKasten:
		return i18n.T("Drag to draw a rectangle. Shift: square.")
	case canvas.WerkzeugText:
		return i18n.T("Click to place text, or click a text to edit it.")
	case canvas.WerkzeugNummer:
		if a := e.s.Voll(); a != canvas.AblehnungKeine {
			return e.grenzText(a) // keine Nummer versprechen, die nicht kommt (F-154)
		}
		return fmt.Sprintf(i18n.T("Click to place step number %d."), e.s.D.NaechsteNummer())
	case canvas.WerkzeugSchwaerzen:
		return i18n.T("Drag over the area to black out. The pixels are overwritten for good once saved.")
	case canvas.WerkzeugVerpixeln:
		return i18n.T("Drag over the area to pixelate. The pixels are overwritten for good once saved.")
	case canvas.WerkzeugZuschneiden:
		// Zuschneiden ist kein Schwärzen: Das Weggeschnittene bleibt im Original der Arbeitsdatei (ADR-0023 Punkt 8).
		return i18n.T("Drag a frame to crop. Double-click: whole image again. Cropped parts stay in the working file; redact what must go.")
	case canvas.WerkzeugStauchen:
		// Wie beim Zuschneiden bleibt das Herausgenommene im Original der Arbeitsdatei (ADR-0023 Punkt 8, F-185).
		return i18n.T("Drag across the part to cut out: vertically for rows, horizontally for columns. Double-click a seam to restore it. Cut-out parts stay in the working file; redact what must go.")
	}
	return ""
}

// statusText schreibt s einzeilig in die Statusleiste (links oder rechtsbündig), höchstens breite Pixel breit, und
// liefert die gezeichnete Breite.
func (e *editor) statusText(gtx layout.Context, s string, c color.NRGBA, y int, links bool, breite int) int {
	lbl := material.Label(e.thema, unit.Sp(12), s)
	lbl.Color = c
	lbl.MaxLines = 1 // kürzt mit „…“, wenn breite nicht reicht
	g := gtx
	g.Constraints.Min = image.Point{}
	g.Constraints.Max.X = max(0, breite)
	m := op.Record(gtx.Ops)
	dims := lbl.Layout(g)
	call := m.Stop()
	x := gtx.Dp(12)
	if !links {
		x = gtx.Constraints.Max.X - gtx.Dp(12) - dims.Size.X
	}
	t := op.Offset(image.Pt(x, y+(gtx.Dp(28)-dims.Size.Y)/2)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	t.Pop()
	return dims.Size.X
}
