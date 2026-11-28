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
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/fonts"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
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

// zwischenablageUebernommen setzt der Gio-Patch (app.ClipboardCancelled, F-328), wenn ein anderes Programm die
// Zwischenablage übernommen hat: Das Fenster hält die Kopie dann nicht mehr, die Warnung beim Schließen entfällt.
var zwischenablageUebernommen atomic.Bool

// exportOhneGrenze: Der Editor wurde mit --ignore-limit geöffnet; der Export prüft das Rechenbudget dann nicht
// (ADR-0033 Nachtrag 4, F-610). Gesetzt von cmd/schnapp vor Editor, gelesen von neuerEditor.
var exportOhneGrenze atomic.Bool

// SetzeExportOhneGrenze legt fest, ob der nächste Editor auch über dem Rechenbudget exportiert: nur für ein Dokument, das
// mit --ignore-limit geöffnet wurde (ADR-0033 Nachtrag 4, F-610). Ohne Aufruf, und mit false, lehnt der Export ab.
func SetzeExportOhneGrenze(an bool) { exportOhneGrenze.Store(an) }

// ExportOhneGrenze liefert die Einstellung von SetzeExportOhneGrenze (für Tests der Verdrahtung).
func ExportOhneGrenze() bool { return exportOhneGrenze.Load() }

// ErrOverlayOffen: Nach der Frist waren noch Overlay-Fenster offen; der Editor öffnet dann nicht (F-82).
var ErrOverlayOffen = errors.New("ui: Overlay-Fenster noch offen")

// ErrSpeichernHaengt: Das Speichern meldete eine Frist lang keinen Fortschritt, und der Nutzer hat das Fenster trotzdem
// geschlossen (zweite Anfrage oder weiteres Signal), oder das Zerstören hat nach zerstoerenFrist aufgegeben; die
// Änderungen sind womöglich nicht gesichert (F-283, F-298, ADR-0020 Nachträge 7 und 9).
var ErrSpeichernHaengt = errors.New("ui: Speichern antwortet nicht")

// Fristen der Arbeit im Hintergrund (ADR-0020 Nachtrag 7); Variablen, damit Tests sie verkürzen.
var (
	// speichernFrist: Meldet ein Speichern so lange keinen Fortschritt (internal/saveprogress), gilt es als hängend:
	// Die Fußzeile sagt es, und eine weitere Schließen-Anfrage oder ein weiteres Signal schließt ohne seine Antwort
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

// Speichern speichert das Dokument (Aufrufer: Bibliothek, ADR-0020). Der Fehlertext erscheint in der Fußzeile und
// muss schon übersetzt sein.
type Speichern func(d *document.Dokument) (Gespeichert, error)

// Gespeichert beschreibt ein gelungenes Speichern, das der Nutzer erfahren muss (F-116).
type Gespeichert struct {
	Datei   string // neuer Dateiname für die Titelleiste (nur Name, kein Ordner); leer = unverändert
	Hinweis string // übersetzter Text für die Fußzeile statt „Saved to the library“; leer = gewöhnlich
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
//
// rettung führt die Rettungskopie gegen Absturz und SIGKILL (ADR-0029); nil: keine.
func Editor(titel string, d *document.Dokument, speichern Speichern, rettung Rettung) error {
	return EditorMit(Geoeffnet{Titel: titel, Dokument: d, Speichern: speichern, Rettung: rettung, OhneGrenze: exportOhneGrenze.Load()}, nil)
}

// editorFuer baut den Editor für g ohne Fenster: Sitzung, Pfad, Anbindung und der Merker für eine Datei von außen
// (ADR-0016 Nachtrag 11; eigene Funktion, damit Tests die Verdrahtung ohne Fenster prüfen, F-832).
func editorFuer(g Geoeffnet, bib *Anbindung) *editor {
	e := neuerEditor(g.Titel, g.Dokument, g.Speichern, g.Rettung)
	e.pfad, e.stand, e.bib = g.Pfad, g.Stand, bib
	e.aussenSetzen(g.Aussen)
	return e
}

// EditorMit ist Editor mit der Anbindung an die Bibliothek (ADR-0036): Leiste der letzten Aufnahmen und Wechsel im
// selben Fenster. bib nil: ohne Leiste. g.OhneGrenze gilt für den Export (ADR-0033 Nachtrag 4).
func EditorMit(g Geoeffnet, bib *Anbindung) error {
	frist := time.Now().Add(2 * time.Second)
	for offeneOverlays.Load() > 0 {
		if time.Now().After(frist) {
			return ErrOverlayOffen
		}
		time.Sleep(20 * time.Millisecond)
	}
	app.ClipboardWriteError = func(err error) { // vor dem ersten Fenster (F-177)
		diagnoseEditor("Zwischenablage nicht geschrieben: %v", err)
		zwischenablageFehlt.Store(true)
	}
	exportOhneGrenze.Store(g.OhneGrenze)
	e := editorFuer(g, bib)
	// Wecker statt einzelnem Invalidate: Gio verwirft ein Invalidate nach einem schon angenommenen (F-231).
	e.wecker = canvas.NeuerWecker(e.w.Invalidate, weckTakt, weckGrenze)
	defer e.wecker.Ende()
	e.leisteStart()    // Index und Vorschauen im Hintergrund (ADR-0036 Punkt 7); braucht den Wecker
	wecker := e.wecker // der Rückruf liest nur diese Kopie, nie e.wecker (Datenwettlauf)
	zwischenablageRueckrufSetzen(wecker.Anfordern)
	e.ocrSucheStarten() // tesseract einmal im Hintergrund suchen (ADR-0035 Punkt 1)
	ctx, abbruch := context.WithCancel(context.Background())
	defer abbruch() // schließt einen offenen Exportdialog mit dem Fenster
	e.exportCtx = ctx
	e.weitereSignale = make(chan struct{}, 1)
	ende := e.signaleAbfangen()
	defer ende()
	return e.laufen()
}

// neuerEditor legt den Editor an, wendet eine Rettungskopie an und baut das erste Ansichtsbild; das Fenster öffnet
// erst laufen. Eigene Funktion, damit Tests den Aufbau ohne Fenster prüfen (F-385).
func neuerEditor(titel string, d *document.Dokument, speichern Speichern, rettung Rettung) *editor {
	th := material.NewTheme()
	// Nur die Go-Schriften, wie im Renderer: sonst zeigte der Bildschirm Zeichen aus Systemschriften, die in der
	// gespeicherten Datei als Kästchen erscheinen (Kritik A-3).
	th.Shaper = text.NewShaper(text.NoSystemFonts(), text.WithCollection(gofont.Collection()))
	e := &editor{w: new(app.Window), thema: th, s: canvas.NeueSitzung(d), titel: titel, speichern: speichern, rettung: rettung,
		ohneGrenze: exportOhneGrenze.Load(), fokusHolen: true}
	// Systemschriften im Hintergrund suchen, damit die Schriftliste sie kennt (ADR-0021 Nachtrag 10 Punkt 3).
	fonts.StarteSuche(e.w.Invalidate)
	e.stileLaden()
	e.farbfelderLaden()
	e.werkzeugStileLaden() // Stil je Werkzeug aus tools.ini (ADR-0041)
	e.rettungStart()       // vor dem ersten Bild: Wiederholte Schwärzungen ändern die Pixel
	e.bildNeu()
	return e
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
	schnell   schnellLeiste  // Abschnitt „Schnellstile“ der rechten Leiste (ADR-0037)
	meldung   canvas.Meldung // Zeile in der Fußzeile (Speichern, Hinweis, Fehler); Vorrang regelt canvas.StatusZeile (F-151)
	titelSet  string

	// fokusHolen: die Leinwand holt sich beim nächsten Bild den Tastaturfokus (Öffnen, Wechsel; F-907). fokusBleibt:
	// der laufende Wechsel kam per Taste an einer Vorschau der Leiste, der Fokus bleibt dort (F-912, F-918).
	fokusHolen, fokusBleibt bool

	wache *canvas.Signalwache // nimmt Signale an (eigene Goroutine), Ereignisschleife fragt Angekommen
	// wecker weckt die Ereignisschleife für Signal und Dateidialog, bis sie ein Ereignis bearbeitet (F-231).
	wecker *canvas.Wecker
	menue  menue // Menüzeile (ADR-0031, Regeln aus ADR-0026)
	// bilder zählt die gezeichneten Bilder (Diagnose mit SCHNAPP_DEBUG, Messung F-235).
	bilder int
	// zug sammelt die Bildzeiten des laufenden Zeigerzugs (nur mit SCHNAPP_DEBUG, F-250, internal/ui/bildzeit.go).
	zug zugZeiten
	// Export (ADR-0025): Der Dialog läuft in einer Goroutine, die Antwort kommt über exportAntwort in die Ereignisschleife.
	exportCtx     context.Context
	exportOffen   bool // ein Dialog läuft (nur einer zugleich); nur die Ereignisschleife liest und setzt es
	exportAntwort chan exportErgebnis
	// Dialog Öffnen (Karte #326): wie der Export, eigene Antwort; nur die Ereignisschleife liest und setzt es.
	oeffnenOffen   bool
	oeffnenAntwort chan exportErgebnis
	// Speichern unter (ADR-0046): Dialog und Schreiben in Goroutinen, Antworten in die Ereignisschleife.
	unter unterZustand
	// neuGestartet: Zeit des letzten Starts einer Neuen Aufnahme (Sperre gegen Mehrfachstart, Kritik K-2).
	neuGestartet time.Time

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
	einf            einfuegeZustand // Lesen der Zwischenablage (ADR-0045)
	kopierLaeuft    bool
	kopierStart     time.Time // Beginn des laufenden Kopierens (Frist beim Schließen, F-281)
	kopierStand     uint64    // Dokumentstand des Abzugs, den das laufende Kopieren ablegt (Meldung, F-285)
	// schliessenNachKopie: Eine Schließen-Anfrage kam während des Kopierens; das Fenster bleibt offen, bis die Antwort
	// da ist oder die Frist um ist, dann fragt es erneut (F-281).
	schliessenNachKopie bool
	// kopierFehlerHaelt: Das Kopieren, auf das ein Schließen wartete, ist gescheitert; das Fenster bleibt einmal offen (F-337).
	kopierFehlerHaelt bool
	// kopierFehlerText ist die Meldung des letzten Kopierfehlers; ein Halt wegen des Kopierfehlers zeigt sie erneut, auch
	// wenn die Leiste inzwischen etwas anderes sagte (F-358).
	kopierFehlerText string
	// meldungHaelt: Die Meldung in der Fußzeile sagt, was die nächste Schließen-Anfrage tut (Verwerfen-Warnung,
	// Hinweis, Fenster-Warnung). Späte Antworten (Kopieren, Export, Fehler der Zwischenablage; meldeNebenbei) verdrängen
	// sie nicht, solange sie frisch ist (F-356, V6-1); jede andere Meldung löst sie ab (melde). Der Kopierfehler-Halt braucht den Schutz nicht: Solange er steht, läuft kein
	// Kopieren, dessen Antwort ihn verdrängen könnte (ein neues Strg+C löst ihn ohnehin mit „Copying…“ ab).
	meldungHaelt haltArt
	// schliessenNachSpeichern: Eine Schließen-Anfrage wartet auf die Antwort des Speicherns (F-341); ein Kopierfehler, der
	// dazwischen eintrifft, wird behandelt wie einer, auf den ein Schließen wartet (kopierenAbholen).
	schliessenNachSpeichern bool
	// fensterHaeltKopie: Die Zwischenablage trägt das Bild nur über dieses Fenster (kein Halter möglich, Rückfall,
	// ADR-0015 Nachtrag 2); fensterWarnung: Das Schließen wurde deswegen schon einmal angehalten (F-295).
	fensterHaeltKopie, fensterWarnung bool
	// wechselBehaeltAnsicht: uebernimm lässt die volle Ansicht der Bibliothek offen (die offene Aufnahme wurde aus ihr
	// weggeworfen, Karte #347).
	wechselBehaeltAnsicht bool
	// verlaeufe: Dokumente verlassener Aufnahmen mit ihrem Verlauf (ADR-0036 Zusatz Karte #356).
	verlaeufe verlaufMerker
	// kopieIstText: Die Kopie, die das Fenster hält, ist Text (Text aus dem Bild, ADR-0024 Nachtrag 9); die Warnung sagt das.
	kopieIstText bool
	// ocr: Text aus dem Bild kopieren (ocr.go, ADR-0035).
	ocr ocrZustand

	// ohneGrenze: mit --ignore-limit geöffnet, der Export lehnt über dem Rechenbudget nicht ab (ADR-0033 Nachtrag 4).
	ohneGrenze bool
	// aussen: Die offene Datei liegt außerhalb der Bibliothek und wird bei der ersten Änderung gespeichert
	// (uebernahmePruefen, ADR-0016 Nachtrag 11); nach dem Anstoßen des Speicherns false.
	aussen bool

	rettung Rettung        // Rettungskopie (ADR-0029); nil: keine
	rett    rettungZustand // Bündelung der Rettungskopie
	// rettungEnden schließt, wenn alle im Hintergrund gestarteten Rettung.Ende eines Wechsels durch sind (F-637); nil:
	// keins.
	rettungEnden chan struct{}

	// Bibliothek in der Oberfläche (ADR-0036): bib verbindet mit Index, Vorschauen und Ladefunktion (nil: keine Leiste,
	// kein Wechsel); pfad liefert die Datei, die den Stand trägt; unten ist die Leiste der letzten Aufnahmen
	// (leiste_unten.go), wechsel ein laufender Wechsel der Aufnahme (wechsel.go).
	bib     *Anbindung
	pfad    func() string
	stand   func() library.Stand // Dateistand der eigenen Speicherfunktion (Geoeffnet.Stand, F-834)
	unten   untenLeiste
	wechsel wechselZustand
	// bibl ist die volle Ansicht der Bibliothek (bibliothek.go, ADR-0036 Punkte 3 und 4).
	bibl bibAnsicht

	speicherStart       time.Time     // Beginn des laufenden Speicherns (F-283)
	haengtGemeldet      bool          // die Fußzeile sagt schon, dass das Speichern nicht antwortet
	speichernAufgegeben bool          // geschlossen ohne Antwort des Speicherns; beim Zerstören nicht mehr warten
	weitereSignale      chan struct{} // jedes weitere Signal nach dem ersten (Wach-Goroutine), F-283

	// knoepfe: Werkzeugleiste links, Farbfeld und die anklickbaren Felder der Fußzeile (ADR-0031). Alle übrigen
	// Handlungen stehen in der Menüzeile.
	knoepfe struct {
		auswahl, pfeil, kasten, text, nummer, farbe  widget.Clickable
		sprechblase                                  widget.Clickable // Werkzeug Sprechblase (ADR-0021 Nachtrag 13)
		lupe                                         widget.Clickable // Werkzeug Lupe (ADR-0033)
		ellipse                                      widget.Clickable // Werkzeug Ellipse (ADR-0047)
		textmarker                                   widget.Clickable // Werkzeug Textmarker (ADR-0048)
		fuellung, tausch                             widget.Clickable // Farbpaar der Werkzeugleiste (ADR-0021 Nachtrag 11)
		schwaerzen, verpixeln, zuschneiden, stauchen widget.Clickable
		fussFarbe, fussZoom                          widget.Clickable
	}
	// seiteZu: Die rechte Leiste ist eingeklappt (Ansicht → Seitenleiste, Strg+Umschalt+B; ADR-0031 Punkt 3). Der
	// Nullwert zeigt sie, sofern das Fenster breit genug ist (seiteSichtbar).
	seiteZu bool
	// seite: Knöpfe der rechten Leiste (seitenleiste.go).
	seite seitenKnoepfe
	// ebenen ist der Abschnitt „Ebenen“ der rechten Leiste (ADR-0032, ebenen.go).
	ebenen ebenenLeiste
	// lupe: Abschnitt „Vergrößerung“ der rechten Leiste und gemerkte Bilder der Lupen (ADR-0033, lupe.go).
	lupe lupenLeiste
	// spitze: Schalter „Spitze“ der rechten Leiste (ADR-0021 Nachtrag 13, spitze.go).
	spitze spitzenLeiste
	// schattenL: Abschnitt „Shadow“ der rechten Leiste (ADR-0021 Nachtrag 26 Punkt 7, schattenleiste.go).
	schattenL schattenLeiste
	// kontext: Kontextmenü der Leinwand (ADR-0038, kontext.go); leinwandUrsprung ist die linke obere Ecke der Leinwand im
	// Fenster beim letzten Zeichnen (Rechtsklicks kommen in Leinwandkoordinaten).
	kontext          kontextMenue
	leinwandUrsprung image.Point
	leinwandRahmen   image.Rectangle // Fläche der Leinwand im Fenster (zweiter Rechtsklick, Kritik K-6 #313a)
	// dateimanager: „Im Dateimanager zeigen“ aus dem Kontextmenü einer Vorschau (ADR-0038 Punkt 3, kontext_vorschau.go).
	dateimanager dateimanagerZustand
	pfeilform    pfeilformLeiste
	strichart    strichartLeiste // Klappfeld „Line style“ (ADR-0021 Nachtrag 25, strichart.go)
	// schatten: gemerkte Schattenbilder je Objekt (ADR-0021 Nachtrag 26, schatten.go).
	schatten      map[int]schattenBild
	schattenBytes int
	schattenNeu   int      // Zahl der neu gerechneten Schattenbilder (für Tests)
	schattenRunde int      // zählt die gezeichneten Bilder (Verdrängen, V-3)
	satz          textsatz // Glyphenumrisse der Beschriftungen (textsatz.go)
	// nav: Zustand des Navigators auf der Leinwand (navigator.go, ADR-0031 Nachtrag 9); die Adresse ist das Zeigerziel.
	nav navigatorZustand
	// fussFeldRahmen: Lage der Felder Farbe und Zoom beim letzten Zeichnen (leer: nicht gezeigt), farbKnopfRahmen die des
	// Farbfelds der Werkzeugleiste; die Aufklapplisten hängen sich daran (listenAnker).
	fussFeldRahmen  map[canvas.FeldArt]image.Rectangle
	farbKnopfRahmen image.Rectangle
	// listeAmFarbKnopf: Die offene Farbliste kam vom Farbfeld der Werkzeugleiste und hängt dort, nicht an der Fußzeile.
	listeAmFarbKnopf bool
	// zeigerFenster ist der Zeigerort über der Leinwand in Fensterpixeln (nil: außerhalb), für die
	// Fußzeile (ADR-0031 Punkt 4).
	zeigerFenster *document.Punkt
	zeigerForm    canvas.ZeigerArt // Form beim letzten Rechnen (merkeZeiger, Zeichnen); einmal je Bewegung (Kritik K-5)
	// fussRahmen: Lage der Felder, die die Fußzeile beim letzten Zeichnen zeigte (von rechts nach links; Tests, F-514).
	fussRahmen []image.Rectangle
	// hilfeGroesse: Größe der Tastenübersicht beim letzten Zeichnen (Tests, F-509: sie passt auf die Leinwand).
	hilfeGroesse image.Point
	// hilfeOffen: die Übersicht der Tastenkürzel liegt über der Leinwand (Menü Hilfe, F1; hilfe.go).
	hilfeOffen bool
	// hilfeRoll: Rollen der Liste der Übersicht, wenn sie nicht in die Tafel passt (F-705, tafelrollen.go).
	hilfeRoll rollZustand
	// einst: Einstellungstafel (Menü Datei → „Settings…“, Strg+,; ADR-0034, einstellungen.go).
	einst tafel

	// Doppelklick erkennen (ADR-0021: Text bearbeiten): Zeit und Ort des letzten Drucks der linken Taste.
	bildStand    uint64      // canvas.Sitzung.BildAenderung beim letzten bildNeu (Schwärzen ändert die Pixel)
	bildOrt      image.Point // Lage des Leinwandbilds in Ansichtskoordinaten (Vollansicht beim Zuschneiden)
	kantenStand  kantenBild  // Stand der Nähte beim letzten bildNeu: liegen Fasersaum und Abdunklung im Bild (F-407)
	letzterDruck time.Duration
	letzterOrt   document.Punkt
	hatDruck     bool

	// rad merkt die Art der laufenden Rad-Geste (ADR-0040 Risiko 1): Nachlauf-Ereignisse des Touchpads tragen die Zeit
	// des letzten echten Ereignisses, aber die Umschalter zum Zeitpunkt des Ticks.
	rad radGeste

	// werkzeugStand ist tools.ini, wie sie nach dem Lesen aussähe (werkzeugstile.go, ADR-0041): beim Schließen wird nur
	// geschrieben, wenn sich etwas geändert hat.
	werkzeugStand string

	// objEbene: über bereichGrenze zeigt der Editor die Objekte aus der Objektebene des Renderers (objektebene.go,
	// ADR-0023 Nachtrag 20, F-787); bereiche zählt die abgespielten Bereiche des laufenden eigenen Zeichnens.
	objEbene objektEbene
	bereiche int

	// naht hält die Kanten der Nähte als aufgezeichnete Ops (Ansichtskoordinaten), bis sich Ansichtsgröße, Nähte,
	// Kantenform oder KantenFaktor ändern (F-183, ADR-0023 Nachtrag 6): Bei vielen Nähten in einem breiten Bild baute der
	// Editor sonst jedes Bild alle Pfade neu.
	naht struct {
		groesse image.Point
		naehte  []document.Naht
		form    document.Kante
		faktor  int
		// saumImBild: Fasersaum und Abdunklung kommen aus dem Bild der Leinwand, die Aufzeichnung hat nur Spalt und Schatten.
		saumImBild bool
		ops        op.Ops
		aufruf     op.CallOp
		gueltig    bool
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

// bildNeu baut das Bild der Leinwand: die Ansicht, beim Werkzeug Zuschneiden die Vollansicht (ADR-0023 Nachtrag 5),
// abgelegt bei bildOrt in Ansichtskoordinaten.
func (e *editor) bildNeu() {
	img, stand := leinwandBild(e.s)
	e.kantenStand = stand
	e.bildOp = paint.NewImageOp(img)
	e.bildOrt = e.s.Bildflaeche().Min
}

// leinwandBild baut die Pixel des Leinwandbilds und liefert den Stand, mit dem es gebaut wurde (ohne Gio, damit ein
// Test die Verdrahtung prüfen kann, F-407). Fasersaum und Abdunklung der Nähte mischt der Renderer ins Bild: Gio mischte
// damals halbdurchsichtige Flächen linear, der Renderer in sRGB; im Fasersaum zeigte der Editor so im Mittel 22 von 255
// zu hell (F-407; seit ADR-0020 Nachtrag 14 mischt Gio in sRGB, das Vorrastern bleibt). Beim Werkzeug Zuschneiden (Vollansicht) bleibt das Bild, wie es ist.
func leinwandBild(s *canvas.Sitzung) (*image.NRGBA, kantenBild) {
	img := s.Leinwandbild()
	stand := kantenBildStand(s)
	if stand.im {
		render.KantenOhneSchatten(img, s.D)
	}
	return img, stand
}

// bildPruefen baut das Leinwandbild neu, wenn sich Pixel (Schwärzen, ADR-0022) oder der Stand der Kanten geändert haben:
// Kantenform, Nähte oder Werkzeug Zuschneiden, dann mischt der Renderer den Fasersaum neu (F-407). Eigene Methode, damit
// ein Test sie ohne Fenster fährt.
func (e *editor) bildPruefen() {
	if n := e.s.BildAenderung(); n != e.bildStand {
		e.bildStand = n
		e.bildNeu() // geschwärzte Pixel sofort zeigen (ADR-0022)
	} else if !kantenBildStand(e.s).gleich(e.kantenStand) {
		e.bildNeu()
	}
}

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
		case app.ConfigEvent:
			e.fensterFokus(ev.Config.Focused)
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
		e.nachEreignis()
	}
}

// nachEreignis läuft nach jedem Ereignis der Schleife: Meldung, Antworten aus Hintergrund-Goroutinen, Signale,
// Rettungskopie. Eigene Funktion, damit Tests es ohne Fenster aufrufen (F-385).
func (e *editor) nachEreignis() {
	// Nach jedem Ereignis nachsehen, ob die Meldung noch gilt: Eine abgelöste kehrt nicht zurück (F-160), auch wenn
	// Ändern und Rückgängig in einem Bild liegen.
	e.meldung.Frisch(e.s.D.Stand(), e.s.Werkzeug)
	if e.zwischenablageFehltMelden() {
		e.w.Invalidate()
	}
	if e.zwischenablageUebernommenMelden() {
		e.w.Invalidate()
	}
	// Auch nach Ereignissen ohne Bild (Gio sendet bei inaktivem Fenster nur ein Weckereignis) die Dialogantwort
	// annehmen, sonst läge sie bis zum nächsten Bild (F-231); ein Folgebild zeigt sie. Ebenso die Antwort des
	// Speicherns (ADR-0020 Nachtrag 6).
	if e.oeffnenAbholen() {
		e.wecker.Anfordern()
	}
	if e.speichernUnterAbholen() {
		e.wecker.Anfordern()
	}
	if e.exportAbholen() {
		e.wecker.Anfordern()
	}
	if e.speichernAbholen() {
		e.wecker.Anfordern()
	}
	if e.dateimanagerAbholen() {
		e.wecker.Anfordern()
	}
	if e.wechselAbholen() || e.leisteIndexAbholen() || e.bibliothekAbholen() { // Wechsel, Index der Leiste und der Ansicht (ADR-0036)
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
	if e.uebernahmePruefen() {
		e.wecker.Anfordern()
	}
	e.rettungPruefen() // Rettungskopie gebündelt nach Änderungen (ADR-0029)
}

// zwischenablageFehltMelden ersetzt, wenn der Gio-Patch einen Schreibfehler der Zwischenablage gemeldet hat, die Meldung
// „Copied“ durch den Fehler und setzt die Marke zurück (F-177); liefert, ob es etwas zu zeigen gab.
func (e *editor) zwischenablageFehltMelden() bool {
	if !zwischenablageFehlt.Swap(false) {
		return false
	}
	text := i18n.T("Could not copy: no clipboard is available.")
	if e.meldungHaelt == haltFenster {
		// Die Fenster-Warnung stimmt nicht mehr: Gio bietet nichts an (Sitzungs-QA 22.11., Q-2, Gegenstück zu V6-3).
		e.melde(text, true)
	} else {
		e.meldeNebenbei(text, true) // statt „Copied“ (F-177); eine andere haltende Meldung bleibt (V6-1)
	}
	e.fensterHaeltKopie = false // Gio bietet nichts an: nichts, wovor zu warnen wäre (F-321)
	return true
}

// zwischenablageRueckrufSetzen setzt den Merker zurück und hängt den Rückruf des Gio-Patchs ein (F-328, QA-4); als
// eigene Funktion, damit ein Test die Verdrahtung trägt.
func zwischenablageRueckrufSetzen(anfordern func()) {
	zwischenablageUebernommen.Store(false)
	app.ClipboardCancelled = func() { // Gios Wayland-Schleife, nicht die Editor-Goroutine (F-328)
		zwischenablageUebernommen.Store(true)
		anfordern()
	}
}

// zwischenablageUebernommenMelden nimmt, wenn ein anderes Programm die Zwischenablage übernommen hat, den Merker
// fensterHaeltKopie zurück und löst eine gezeigte Fenster-Warnung ohne neuen Text auf (F-328); liefert, ob es etwas
// zu tun gab.
func (e *editor) zwischenablageUebernommenMelden() bool {
	if !zwischenablageUebernommen.Swap(false) {
		return false
	}
	// Für die Handprobe F-392: wann KWin cancelled schickt (nach fremdem Kopieren oder schon beim eigenen Strg+C).
	diagnoseEditor("Zwischenablage übernommen (cancelled), Fenster hielt Kopie: %v", e.fensterHaeltKopie)
	if !e.fensterHaeltKopie {
		return false
	}
	e.fensterHaeltKopie = false
	if e.meldungHaelt == haltFenster {
		e.melde("", false)
	}
	return true
}

// zerstoert räumt beim Zerstören des Fensters auf und liefert den Rückgabewert von Editor. Läuft noch ein Speichern im
// Hintergrund, wartet es auf seine Antwort (ADR-0020 Nachtrag 6), höchstens zerstoerenFrist lang (Nachtrag 7); ein
// weiterer Versuch nach einem Signal läuft ebenso. Ein laufendes Kopieren bekommt seine Frist, damit der Halter der
// Zwischenablage noch startet (F-281).
func (e *editor) zerstoert(fensterErr error) error {
	e.ocrAbbrechen()     // eine laufende Texterkennung endet mit dem Fenster (ADR-0035 Punkt 2)
	e.wechselAbbrechen() // ein laufender Wechsel gibt die Rettungskopie der geladenen Aufnahme frei (ADR-0036)
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
	e.einfuegenAbwarten(wechselFrist)        // ein laufendes „New from clipboard“ legt zu Ende ab (F-988)
	e.bibliothekArbeitAbwarten(wechselFrist) // Schlagworte/Papierkorb der Ansicht schreiben zu Ende (F-708)
	e.s.TextBestaetigen()                    // getippter Text geht nicht verloren (ADR-0021 Punkt 6)
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
	e.rettungEnde()
	e.rettungEndenAbwarten(wechselFrist) // Rettungskopien früherer Aufnahmen dieses Fensters (F-637)
	e.werkzeugStileSchreiben()           // Stil je Werkzeug für das nächste Fenster (ADR-0041 Punkt 4)
	// Zuletzt, nach Notspeichern und Aufräumen (F-1016, Verify V-2): Eine laufende Kopie (Speichern unter) schreibt zu
	// Ende, sonst blieben eine halbe temporäre Datei oder im Rückfall ein gekürztes Ziel liegen (ADR-0046 Punkt 6, F-1009);
	// ein weiteres Signal beendet das Warten.
	e.speichernUnterAbwarten(unterFrist)
	return errors.Join(fensterErr, e.schlussErr, err)
}

// fehlerAusgabe ist die Fehlerausgabe des Editors; Naht für Tests (F-337).
var fehlerAusgabe io.Writer = os.Stderr

// hinweisAufStderr: Das Fenster ist (gleich) zu, die Fußzeile weg — ein Hinweis (z. B. neue Aufnahme statt der
// alten Datei) geht auf die Fehlerausgabe, damit er nicht still verloren ist (F-116).
func hinweisAufStderr(g Gespeichert) {
	if g.Hinweis != "" {
		fmt.Fprintln(os.Stderr, g.Hinweis)
	}
}

// schliessen beantwortet eine Schließen-Anfrage; false hält das Fenster offen (Entscheidung in canvas.Sitzung). Ist
// etwas zu speichern, läuft es im Hintergrund (ADR-0020 Nachtrag 6): Das Fenster bleibt offen und zeichnet weiter, die
// Antwort entscheidet in speichernFertig wie bisher — zu; einmal offen mit Hinweis; offen mit Fehler, und die nächste
// Anfrage verwirft. Hängt ein Speichern über die Frist und sagt die Fußzeile das schon, schließt die Anfrage ohne
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
		// Nur erreichbar, wenn der Abzug nach dem Abbrechen eines Zugs scheitert (canvas.Sitzung.SchliessenAnfrage); kein
		// Test erreicht den Zweig, „haltend“ steht hier der Einheitlichkeit halber (Verify 6, Mutante M2/A2).
		e.meldeHaltend(haltVerwerfen, i18n.T("Could not save. Close again to discard the changes."), true)
	}
	if z == canvas.SchlussWarten {
		e.schliessenNachSpeichern = true
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
	text := i18n.T("The clipboard holds the image only while this window is open. Close again to discard it.")
	if e.kopieIstText {
		text = i18n.T("The clipboard holds the text only while this window is open. Close again to discard it.")
	}
	e.meldeHaltend(haltFenster, text, true)
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
	e.rettungVorSpeichern() // vor dem Speichern sichern: ein hängendes Speichern plus SIGKILL (V7-7)
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
			if r.err == nil {
				e.rettungGespeichert()
			}
			e.speichernMelden(r, ende, false, false) // das Fenster ist schon weg: kein Halt
			// Eine Schließen-Anfrage, deren Fehler erst jetzt ankommt, hat niemand gesehen: Die Fußzeile verschwindet
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

// speichernHaengtMelden sagt in der Fußzeile, dass das Speichern nicht antwortet und das Schließen jetzt ohne seine
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
		e.rettungGespeichert()
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
	if r.err == nil {
		e.rettungGespeichert() // gespeichert: die Rettungskopie ist überflüssig (ADR-0029 Punkt 5)
		e.aussen = false       // jedes gelungene Speichern hat die Datei von außen übernommen (K-4)
	}
	// Der Kopierfehler kam vor dem Speicher-Ende (F-341): Er hält das Fenster einmal offen wie bei F-337 — aber nur, wenn
	// dieses Speichern schließen würde, kein Signal vorgemerkt ist (ein Signal schließt in jedem Fall, ADR-0020 Nachtrag
	// 2, F-346) und kein neues Kopieren läuft (dessen Antwort entscheidet, F-348). Der Merker wird vorher gelesen und in
	// jedem Ausgang ohne Erneut verbraucht (F-347); nur eine wegen des Kopierens angehaltene Anfrage behält ihn für
	// nachKopieSchliessen. ende.Erneut == nil ist neben !SignalVorgemerkt() doppelt (Erneut gibt es nur nach einem Signal,
	// canvas.Sitzung.SpeichernEnde) und steht nur der Lesbarkeit halber da (F-370, Mutante Q6 äquivalent).
	halt := e.kopierFehlerHaelt && ende.Erneut == nil && ende.Schliessen && !e.s.SignalVorgemerkt() && !e.kopierLaeuft
	// Der Hinweis hält eine Schließen-Anfrage offen: die des Schließens (SchlussHinweis) oder eine wegen des Kopierens
	// angehaltene, die er gleich verwirft (F-357). Nur dann ist er haltend (V6-2).
	hinweisHaelt := ende.Schluss == canvas.SchlussHinweis || (r.err == nil && r.g.HaeltOffen() && e.schliessenNachKopie)
	if ende.Erneut == nil {
		e.schliessenNachSpeichern = false
		if ende.Schluss == canvas.SchlussFehler || ende.Schluss == canvas.SchlussHinweis || (r.err == nil && r.g.HaeltOffen()) {
			// Das Speicher-Ende hat die Schließen-Anfrage entschieden (offen mit Fehler oder Hinweis). Eine wegen des
			// Kopierens angehaltene Anfrage darf danach nicht von selbst neu gestellt werden: Sie gälte als Bestätigung
			// des Verwerfens bzw. schlösse vor dem Lesen des Hinweises (F-352). Dasselbe gilt für einen Hinweis aus einem
			// Strg+S (Ende.Schluss bleibt dann SchlussZu, F-357).
			e.schliessenNachKopie = false
		}
		if !e.schliessenNachKopie {
			e.kopierFehlerHaelt = false
		}
	}
	// Steht der Merker noch neben einer wegen des Kopierens angehaltenen Anfrage, hält gleich nachKopieSchliessen das
	// Fenster für den Kopierfehler offen; auch dann verdrängt „Saved“ ihn nicht (F-353). Läuft schon ein neues Kopieren,
	// entscheidet dessen Antwort (F-354), und das Speicher-Ende meldet sich wie sonst (F-359).
	e.speichernMelden(r, ende, halt || (e.kopierFehlerHaelt && e.schliessenNachKopie && !e.kopierLaeuft), hinweisHaelt)
	if r.err == nil {
		e.leisteNachSpeichern() // Vorschau und Index sind neu (ADR-0036 Punkt 7)
	}
	e.wechselNachSpeichern(r, ende) // ein wartender Wechsel der Aufnahme (ADR-0036 Punkt 5)
	switch {
	case ende.Erneut != nil:
		e.speichernStarten(ende.Erneut)
	case halt:
		// Die Meldung „Could not copy“ bleibt stehen (speichernMelden lässt sie), die nächste Anfrage schließt. Der Halt
		// beantwortet auch eine wegen des Kopierens angehaltene Anfrage (F-351).
		e.kopierFehlerHaelt, e.schliessenNachKopie = false, false
	case ende.Schliessen:
		// Eine Schließen-Anfrage mit Speichern vorher warnt wie eine ohne (F-321); ein Signal schließt in jedem Fall.
		// Läuft noch ein Kopieren, wartet das Schließen auf dessen Antwort wie im Zweig SchlussZu: Erst sie sagt, ob das
		// Fenster die Kopie hält (F-329); danach stellt nachKopieSchliessen die Anfrage erneut, und fensterWarnen greift.
		if !e.s.SignalVorgemerkt() && (e.kopierenHaeltOffen() || e.fensterWarnen()) {
			return
		}
		e.schliessenFrei()
	}
}

// speichernMelden zeigt das Ergebnis in Titel- und Fußzeile. Schließt das Fenster gleich oder bleibt es für einen
// Hinweis offen, geht der Hinweis zusätzlich auf die Fehlerausgabe (F-116, ADR-0020 Nachtrag 4). Hält ein Kopierfehler
// das Fenster offen (kopierHalt, F-341), verdrängt „Saved“ dessen Meldung nicht: Die Leiste zeigt den Kopierfehler
// erneut, auch wenn sie inzwischen etwas anderes sagte, etwa dass das Speichern nicht antwortet (F-358).
func (e *editor) speichernMelden(r speicherErgebnis, ende canvas.Ende, kopierHalt, hinweisHaelt bool) {
	switch {
	case r.err == nil:
		if !kopierHalt {
			e.gespeichertMelden(r.g, hinweisHaelt)
		} else {
			if r.g.Datei != "" {
				e.titel = r.g.Datei
			}
			e.melde(e.kopierFehlerText, true)
		}
		if ende.Schliessen || ende.Schluss == canvas.SchlussHinweis {
			hinweisAufStderr(r.g)
		}
	case ende.SignalFehler:
		e.schlussErr = r.err
	case ende.Schluss == canvas.SchlussFehler:
		e.meldeHaltend(haltVerwerfen, fmt.Sprintf(i18n.T("Could not save: %v. Close again to discard the changes."), ohnePaketpraefix(r.err.Error())), true)
	case ende.Erneut == nil:
		e.melde(fmt.Sprintf(i18n.T("Could not save: %v"), ohnePaketpraefix(r.err.Error())), true)
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
	// typ ist der Typ, unter dem das Fenster selbst anbietet (leer: wlclip.MIME); meldung ersetzt „Copied to clipboard“.
	// Beides setzen textKopieren (ocr.go) und objekteKopieren (einfuegen.go).
	typ, meldung string
	// objekte: kopierte Objekte, die das Fenster selbst anbietet (nur ihr Bild, ADR-0045 Punkt 2); Strg+V nimmt sie,
	// solange das Fenster die Zwischenablage hält.
	objekte *document.Dokument
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
			if err := halteZwischenablage(daten, wlclip.ArtBild); err != nil {
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
			e.einf.schneide = nil // nichts kopiert: nichts ausschneiden (F-989)
			e.kopierFehlerText = fmt.Sprintf(i18n.T("Could not copy: %v"), ohnePaketpraefix(r.err.Error()))
			if e.schliessenNachKopie || e.schliessenNachSpeichern || e.s.SignalVorgemerkt() {
				// Ein Schließen hat auf dieses Kopieren gewartet (F-337) oder steht nach dem Speichern an, das noch
				// läuft (F-341): Der Fehler geht zusätzlich auf die Fehlerausgabe, und das Fenster bleibt einmal offen,
				// damit die Meldung gelesen wird; die nächste Anfrage schließt. Ein Signal schließt in jedem Fall, ebenso
				// ein schon freigegebenes Schließen (erzwungen): Dann ist kein Halt mehr zu merken.
				fmt.Fprintf(fehlerAusgabe, i18n.T("Could not copy: %v")+"\n", ohnePaketpraefix(r.err.Error()))
				e.kopierFehlerHaelt = !e.s.SignalVorgemerkt() && !e.erzwungen
				e.melde(e.kopierFehlerText, true)
				return
			}
			// Sagt die Leiste, was die nächste Schließen-Anfrage tut, bleibt das stehen, und der Kopierfehler geht auf die
			// Fehlerausgabe (F-356).
			e.meldeNebenbei(e.kopierFehlerText, true)
			return
		}
		// Ausschneiden: erst jetzt löschen, nach gelungenem Kopieren und nur bei unverändertem Stand (F-989).
		if ids := e.einf.schneide; ids != nil {
			e.einf.schneide = nil
			if !e.s.Ausschneiden(ids, e.einf.schneideStand) {
				r.meldung = i18n.T("Copied to the clipboard; the objects were not removed. Press Ctrl+X again to remove them.")
			}
		}
		// Die jüngste Antwort entscheidet: Ein früherer Kopierfehler hält das Fenster nicht mehr (F-354).
		e.kopierFehlerHaelt = false
		// Ein Abbruch bis hierher betraf eine ältere Quelle; er darf die neue Kopie nicht abräumen (F-328, QA-5).
		zwischenablageUebernommen.Store(false)
		e.fensterHaeltKopie, e.fensterWarnung = r.daten != nil, false
		e.kopieIstText = r.typ != "" && r.objekte == nil // Text aus dem Bild: die Warnung beim Schließen sagt „Text“
		e.einf.kopie = r.objekte                         // nur gültig, solange das Fenster die Kopie hält
		if r.daten != nil {
			typ := wlclip.MIME
			if r.typ != "" {
				typ = r.typ
			}
			gtx.Execute(clipboard.WriteCmd{Type: typ, Data: io.NopCloser(bytes.NewReader(r.daten))})
		}
		// Die Zwischenablage trägt den Stand beim Tastendruck. Wurde seitdem bearbeitet, sagt die Meldung das, statt
		// den Eindruck zu wecken, das jetzt Sichtbare (etwa eine neue Schwärzung) sei kopiert (F-285).
		text := i18n.T("Copied to clipboard")
		if e.s.D.Stand() != e.kopierStand {
			text = i18n.T("Copied to clipboard as it was when copying began; later changes are not included.")
		}
		if r.meldung != "" { // Text aus dem Bild: „Copied N characters“ (ADR-0035)
			text = r.meldung
		}
		if e.meldungHaelt == haltFenster && e.haltendeMeldung() && r.daten == nil {
			// Die Fenster-Warnung stimmt nicht mehr: Ein Halter hat das Bild übernommen (V6-3).
			e.melde(text, false)
			return
		}
		// Sagt die Leiste, was die nächste Schließen-Anfrage tut, ist das wichtiger als die Bestätigung (F-356).
		e.meldeNebenbei(text, false)
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
	if e.schliessenNachSpeichern {
		// Eine zweite Anfrage hat ein Speichern gestartet: Dessen Ende entscheidet und verbraucht einen Kopierfehler-Halt
		// erst dann (speichernFertig, F-348).
		return false
	}
	if e.kopierFehlerHaelt { // das Kopieren ist gescheitert: einmal offen lassen, die nächste Anfrage schließt (F-337)
		e.kopierFehlerHaelt = false
		return false
	}
	return true
}

// kopierenAbwarten wartet beim Zerstören des Fensters auf ein laufendes Kopieren, höchstens bis kopierFrist ab Beginn
// (F-281): Der Halter der Zwischenablage startet in der Goroutine; endete der Prozess vorher, bliebe die Zwischenablage
// still leer. Was das Fenster selbst hätte anbieten müssen (kein Halter möglich), geht mit dem Fenster verloren; das
// steht wie eine abgelaufene Frist auf der Fehlerausgabe, denn die Fußzeile ist weg.
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
			fmt.Fprintln(fehlerAusgabe, i18n.T("Copying was not finished when the window closed; the clipboard may be empty."))
			return
		}
	}
	e.kopierLaeuft = false
	switch {
	case r.err != nil:
		fmt.Fprintf(fehlerAusgabe, i18n.T("Could not copy: %v")+"\n", ohnePaketpraefix(r.err.Error()))
	case r.daten != nil:
		fmt.Fprintln(fehlerAusgabe, i18n.T("Copying was not finished when the window closed; the clipboard may be empty."))
	}
}

// exportErgebnis ist die Antwort des Dateidialogs auf dem Weg in die Ereignisschleife.
type exportErgebnis struct {
	ergebnis portal.Ergebnis
	err      error
	// zugeordnet: beim Öffnen die Aufnahme, die die gewählte Datei öffnen würde (Anbindung.Zuordnen); leer: keine
	// Zuordnung.
	zugeordnet string
}

// exportStarten öffnet den Dateidialog des Desktop-Portals (ADR-0025). Er läuft in einer eigenen Goroutine, denn die
// Antwort kommt erst nach der Wahl des Nutzers; sie weckt das Fenster, geschrieben wird in der Ereignisschleife
// (exportAbholen), wie bei der Signalwache (F-77). Nur ein Dialog zugleich.
func (e *editor) exportStarten() {
	if e.exportOffen || e.unter.offen || e.s.Zieht() || e.wechsel.laedt { // während des Ladens einer anderen Aufnahme nicht (F-652)
		return
	}
	e.exportOffen = true
	if !e.haltendeMeldung() {
		// Sonst verdeckte eine frische Meldung den Hinweis zum offenen Dialog (A-12). Eine haltende Meldung (etwa die
		// Verwerfen-Warnung) bleibt: Bricht der Nutzer den Dialog ab, muss sie noch dastehen (Sitzungs-QA 22.11., Q-1).
		e.melde("", false)
	}
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
		r.ergebnis, r.err = exportDialog(ctx, anfrage)
		ch <- r
		if wecker != nil {
			wecker.Anfordern() // erst nach dem Ablegen der Antwort (F-231)
		}
	}()
}

// exportDialog zeigt den Speichern-Dialog des Exports; Naht für Tests (wie oeffnenDialog und unterDialog).
var exportDialog = func(ctx context.Context, a portal.Anfrage) (portal.Ergebnis, error) {
	d, err := portal.Neu()
	if err != nil {
		return portal.Ergebnis{}, err
	}
	defer d.Close()
	return d.Speichern(ctx, a)
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

// exportFertig wertet die Antwort aus: Abbruch tut nichts, Fehler und Erfolg erscheinen in der Fußzeile.
func (e *editor) exportFertig(r exportErgebnis) {
	switch {
	case errors.Is(r.err, portal.ErrAbgebrochen), errors.Is(r.err, context.Canceled):
		return
	case errors.Is(r.err, portal.ErrKeinPortal):
		diagnoseEditor("Export: %v", r.err)
		e.meldeNebenbei(i18n.T("No file dialog available (desktop portal missing)."), true)
		return
	case r.err != nil:
		e.meldeNebenbei(fmt.Sprintf(i18n.T("Could not export: %v"), exportFehlerText(r.err, "")), true)
		return
	}
	pfad, err := portal.URIZuPfad(r.ergebnis.URI)
	if err != nil {
		e.meldeNebenbei(i18n.T("Could not export: only local files can be written."), true)
		return
	}
	f, pfad, angehaengt := export.Waehle(pfad, export.AusIndex(r.ergebnis.Filter))
	// Der Dialog hat nur den Namen ohne Endung bestätigt: Ein vorhandenes Ziel nicht still ersetzen (A-1).
	if err := export.PruefeAngehaengt(pfad, angehaengt); err != nil {
		e.meldeNebenbei(exportFehlerText(err, filepath.Base(pfad)), true)
		return
	}
	// Wie beim Speichern: ein laufender Text gehört mit in die Datei, ein laufendes Ziehen verschiebt den Export (A-5).
	e.s.TextBestaetigen()
	if e.s.Zieht() {
		e.meldeNebenbei(i18n.T("Could not export: finish the current action first."), true)
		return
	}
	// Rechenbudget vor dem Kodieren (ADR-0033 Nachtrag 2 Punkt 3): Im Editor kann ein Dokument über das Budget wachsen,
	// das beim Öffnen geprüft wurde; der Export (auch der SVG-Export mit den Bildern der Lupen) liefe dann unbegrenzt.
	// Mit --ignore-limit geöffnet (ADR-0033 Nachtrag 4, F-610): kein Abbruch, wie Speichern und --export --ignore-limit;
	// die Leiste sagt nach dem Schreiben, dass es lange dauern kann (der Export läuft in der Ereignisschleife, vorher
	// käme kein Bild mehr dazwischen).
	ueberBudget := false
	if err := render.Pruefe(e.s.D); err != nil {
		diagnoseEditor("Export: %v", err)
		if !e.ohneGrenze {
			e.meldeNebenbei(i18n.T("Could not export: the image has too many or too large annotations. Remove some first."), true)
			return
		}
		ueberBudget = true
	}
	// Grenze der Lupenpixel im SVG (ADR-0033 Nachtrag 5, QS-2): eigene Meldung, auch sie vor dem Kodieren; mit
	// --ignore-limit wie das Budget.
	if f == export.SVG {
		if err := fileformat.PruefeSVGExport(e.s.D); err != nil {
			diagnoseEditor("Export: %v", err)
			if !e.ohneGrenze {
				e.meldeNebenbei(i18n.T("Could not export as SVG: the magnifiers would embed too many pixels. Export as PNG or JPG, or remove some magnifiers."), true)
				return
			}
			ueberBudget = true
		}
	}
	schreibeDok := export.Schreibe
	if e.ohneGrenze {
		schreibeDok = export.SchreibeOhneGrenze
	}
	schreibe := export.SchreibeAtomar
	if angehaengt {
		schreibe = export.SchreibeOhneErsetzen // Ziel könnte während des Renderns entstehen (Verify N-4)
	}
	err = schreibe(pfad, func(w io.Writer) error { return schreibeDok(e.s.D, f, w) })
	if err != nil {
		e.meldeNebenbei(fmt.Sprintf(i18n.T("Could not export: %v"), exportFehlerText(err, "")), true)
		return
	}
	if ueberBudget {
		e.meldeNebenbei(fmt.Sprintf(i18n.T("Exported to %s. The image is over the render limit (--ignore-limit); exporting can take long."), filepath.Base(pfad)), false)
		return
	}
	e.meldeNebenbei(fmt.Sprintf(i18n.T("Exported to %s"), filepath.Base(pfad)), false)
}

// ohneDataControl merkt sich, dass der Compositor kein data-control anbietet: Weitere Strg+C starten dann keinen
// vergeblichen Halter mehr (Kritik Fable, Befund 6).
var ohneDataControl atomic.Bool

// uebergib startet den Halter-Prozess (wlclip.Uebergib); Naht für Tests, die die Art (Bild oder Text) prüfen.
var uebergib = wlclip.Uebergib

// halteZwischenablage übergibt das Bild bzw. den Text (art) an einen Halter-Prozess (dasselbe Programm,
// wlclip.Uebergib). Mit WAYLAND_SOCKET (geerbte Verbindung, etwa in verschachtelten Sitzungen) fände der Halter nicht
// sicher denselben Compositor; dann bietet das Fenster selbst an (Befund 5).
func halteZwischenablage(daten []byte, art wlclip.Art) error {
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
	pid, manager, err := uebergib(programm, art, daten, wlclip.Frist)
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

// gespeichertMelden zeigt ein gelungenes Speichern in Titel- und Fußzeile (F-116). haelt: Der Hinweis hält eine
// Schließen-Anfrage offen (SchlussHinweis, F-357); dann verdrängt ihn keine späte Antwort (F-356). Ein Hinweis aus einem
// gewöhnlichen Strg+S sagt nichts über die nächste Anfrage und lässt einem späteren Kopierfehler den Vortritt (V6-2).
func (e *editor) gespeichertMelden(g Gespeichert, haelt bool) {
	if g.Datei != "" {
		e.titel = g.Datei
	}
	e.melde(i18n.T("Saved to the library"), false)
	if g.Hinweis != "" {
		if haelt {
			e.meldeHaltend(haltHinweis, g.Hinweis, g.Warnung)
		} else {
			e.melde(g.Hinweis, g.Warnung)
		}
	}
}

// melde zeigt text in der Fußzeile, bis das Dokument sich ändert oder (außer bei Fehlern) das Werkzeug wechselt
// (F-151). Ein gelungenes Speichern löst einen früheren Fehler ab.
func (e *editor) melde(text string, fehler bool) {
	e.meldung = canvas.NeueMeldung(text, fehler, e.s.D.Stand(), e.s.Werkzeug)
	e.meldungHaelt = keinHalt
}

// haltArt sagt, was eine haltende Meldung über die nächste Schließen-Anfrage aussagt (F-356).
type haltArt int

const (
	keinHalt      haltArt = iota
	haltVerwerfen         // Speichern gescheitert: die nächste Anfrage verwirft
	haltHinweis           // Hinweis beim Speichern: die nächste Anfrage schließt
	haltFenster           // nur das Fenster hält die Kopie: die nächste Anfrage verwirft sie (F-295)
)

// meldeHaltend zeigt eine Meldung, die sagt, was die nächste Schließen-Anfrage tut (F-356): Späte Antworten (Kopieren,
// Export, Zwischenablage) verdrängen sie nicht, solange sie gilt (meldeNebenbei).
func (e *editor) meldeHaltend(art haltArt, text string, fehler bool) {
	e.melde(text, fehler)
	e.meldungHaelt = art
}

// haltendeMeldung meldet, ob die Fußzeile gerade eine haltende Meldung zeigt, die noch gilt (gleicher Dokumentstand,
// bei einem Hinweis auch gleiches Werkzeug; canvas.Meldung.Frisch).
func (e *editor) haltendeMeldung() bool {
	return e.meldungHaelt != keinHalt && e.meldung.Frisch(e.s.D.Stand(), e.s.Werkzeug)
}

// meldeNebenbei zeigt die Meldung einer späten Antwort (Kopieren, Export, Zwischenablage), ohne eine haltende Meldung zu
// verdrängen (F-356, V6-1): Dann geht ein Fehler auf die Fehlerausgabe, eine Bestätigung nur ins Diagnose-Log.
func (e *editor) meldeNebenbei(text string, fehler bool) {
	if !e.haltendeMeldung() {
		e.melde(text, fehler)
		return
	}
	if fehler {
		fmt.Fprintln(fehlerAusgabe, text)
		return
	}
	diagnoseEditor("Meldung %q bleibt, nicht gezeigt: %q", e.meldung.Text, text)
}

// meldeAbgelehnt zeigt, warum ein Druck mit einem Zeichenwerkzeug nichts angelegt hat (F-154).
func (e *editor) meldeAbgelehnt() {
	switch e.s.Abgelehnt() {
	case canvas.AblehnungObjekte:
		e.melde(e.grenzText(canvas.AblehnungObjekte), true)
	case canvas.AblehnungNummer:
		e.melde(e.grenzText(canvas.AblehnungNummer), true)
	case canvas.AblehnungTextVerborgen:
		e.melde(i18n.T("This text lies in a compressed area. Restore cut-out areas (Image menu) to edit it."), false)
	case canvas.AblehnungSichtbar:
		e.melde(i18n.T("Not done: an object would leave the visible image."), false)
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
	_ = e.oeffnenAbholen()
	_ = e.speichernUnterAbholen()
	_ = e.speichernAbholen() // ebenso die Antwort des Speicherns und des Kopierens (ADR-0020 Nachtrag 6)
	_ = e.wechselAbholen()   // und des Wechsels der Aufnahme und des Index der Leiste (ADR-0036)
	_ = e.leisteIndexAbholen()
	_ = e.bibliothekAbholen()
	e.kopierenAbholen(gtx)
	e.einfuegenAbholen() // Lesen der Zwischenablage (ADR-0045), nach dem Kopieren
	e.ocrAbholen()       // Suche und Texterkennung (ADR-0035), nach dem Kopieren
	// Vor allen anderen: Klicks und Tasten der offenen Einstellungstafel (ADR-0034).
	e.einstellungenAuswerten(gtx)
	// Ebenso die volle Ansicht der Bibliothek: ihre Tasten vor feldTasten und tasten (ADR-0036).
	e.bibliothekAuswerten(gtx)
	e.feldTasten(gtx) // vor tasten: Enter, Esc und Rücktaste gehören dem fokussierten Eingabefeld (ADR-0031 Nachtrag 5)
	e.tasten(gtx)
	// Menü zuerst: Startet ein Eintrag das Speichern, wirken Klicks auf Leisten im selben Bild nicht mehr (F-280).
	// Bei offener Ansicht der Bibliothek sind Werkzeugleiste, rechte Leiste und Leiste unten nicht gezeichnet und
	// bekommen keine Klicks; die Felder der Fußzeile sperrt fussAuswerten selbst (ADR-0036 Punkt 4).
	e.menueAuswerten(gtx)
	e.knoepfeAuswerten(gtx)
	e.fussAuswerten(gtx)
	e.seiteAuswerten(gtx)
	e.leisteAuswerten(gtx)
	paint.Fill(gtx.Ops, farbeFenster)
	if e.bibl.offen {
		mx := gtx.Constraints.Max
		e.zeichneBibliothek(gtx, image.Rect(0, gtx.Dp(menuezeileDp), mx.X, max(gtx.Dp(menuezeileDp), mx.Y-gtx.Dp(fusszeileDp))))
		e.leisteVerborgen()
		e.zeichneMenuezeile(gtx)
		e.zeichneFusszeile(gtx)
	} else {
		flaeche := e.leinwandFlaeche(gtx) // zwischen Menüzeile, Werkzeugleiste, Fußzeile und rechter Leiste (ADR-0031)

		// Leinwand: eigener Ursprung, damit Zeigerpositionen schon Leinwandkoordinaten sind.
		e.leinwandUrsprung, e.leinwandRahmen = flaeche.Min, flaeche // Anker des Kontextmenüs (ADR-0038)
		t := op.Offset(flaeche.Min).Push(gtx.Ops)
		g := gtx
		g.Constraints = layout.Exact(flaeche.Size())
		e.leinwand(g)
		t.Pop()

		e.zeichneMenuezeile(gtx)
		e.zeichneWerkzeugleiste(gtx)
		e.zeichneSeitenleiste(gtx)
		if e.leisteSichtbar(gtx) { // unter der Leinwand, so breit wie sie (ADR-0036 Punkt 1)
			e.zeichneLeisteUnten(gtx, image.Rect(flaeche.Min.X, flaeche.Max.Y, flaeche.Max.X, flaeche.Max.Y+gtx.Dp(leisteUntenDp)))
		} else {
			e.leisteVerborgen()
		}
		e.zeichneFusszeile(gtx)
		e.zeichneSchriftliste(gtx) // über Leisten und Fußzeile, unter Hilfe und Menü
	}
	e.zeichneHilfe(gtx)
	e.zeichneEinstellungen(gtx)
	e.zeichneMenue(gtx)     // über allem, unter dem Titel des offenen Menüs
	e.zeichneFeldWache(gtx) // durchlässig über allem: Klick woanders übernimmt eine Feldeingabe (A-2)
}

// --- Leinwand ---

func (e *editor) leinwand(gtx layout.Context) {
	groesse := gtx.Constraints.Max
	e.s.SetzeFenster(groesse)
	e.zeiger(gtx)
	e.ocrBereichPruefen() // Bereich gewählt (Loslassen hier, Enter in tasten): Erkennung starten (ADR-0035)
	e.bildPruefen()

	c := clip.Rect(image.Rectangle{Max: groesse}).Push(gtx.Ops)
	paint.Fill(gtx.Ops, farbeGrund)
	a := e.s.Abb
	af := op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(float32(a.Zoom), float32(a.Zoom))).
		Offset(f32.Pt(float32(a.Versatz.X), float32(a.Versatz.Y)))).Push(gtx.Ops)
	b := e.bildOp
	if a.Zoom >= 2 {
		b.Filter = paint.FilterNearest // einzelne Pixel sichtbar
	}
	// Das Bild liegt bei bildOrt (beim Zuschneiden die Vollansicht, ADR-0023 Nachtrag 5) und ist auf sich selbst
	// begrenzt.
	bo := op.Offset(e.bildOrt).Push(gtx.Ops)
	bi := clip.Rect(image.Rectangle{Max: b.Size()}).Push(gtx.Ops)
	b.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	bi.Pop()
	bo.Pop()
	// Objekte im Ausschnitt der Ansicht: Was außerhalb liegt, fehlt auch in der gespeicherten Datei (Kritik A-7).
	bc := clip.Rect(image.Rectangle{Max: e.s.D.Groesse()}).Push(gtx.Ops)
	e.zeichneObjekte(gtx)
	e.zeichneNaehte(gtx)
	bc.Pop()
	af.Pop()
	e.zeichneAuswahl(gtx)
	e.zeichneRahmen(gtx)
	e.zeichneTextBereich(gtx)
	e.zeichneHilfslinien(gtx)
	e.zeichneEinfuegemarke(gtx)

	e.zeigerForm = e.s.Zeiger(e.zeigerFenster) // ein Zug oder die Sperre ändern die Form auch ohne Bewegung
	systemZeiger(e.zeigerForm).Add(gtx.Ops)    // je Lage: Griffe, Objekte, Werkzeug (Karte #323)
	event.Op(gtx.Ops, e)
	c.Pop()
	// Außerhalb des Ausschnitts der Leinwand angemeldet: Gio gibt ein Zeigerereignis auch den Zielen weiter, die im selben
	// offenen Ausschnitt davor angemeldet sind; so erreichen Druck, Ziehen und Rad die Werkzeuge nicht.
	e.zeichneNavigator(gtx)
}

// zeichneRahmen zeigt den gerade gezogenen Rahmen in Fensterpixeln: beim Schwärzen und beim Stauchen-Band (ADR-0022,
// ADR-0023) halbdurchsichtig dunkel, beim Zuschneiden ist das Bild außerhalb des Rahmens abgedunkelt; immer mit hellem
// Rand, damit er auf hellen wie dunklen Bildern sichtbar ist.
//
// Beim Werkzeug Zuschneiden liegt der Rahmen immer da (ADR-0023 Nachtrag 5): auf dem Zuschnitt oder dem gezogenen
// Rahmen, mit acht Griffen; abgedunkelt ist die Vollansicht außerhalb davon.
func (e *editor) zeichneRahmen(gtx layout.Context) {
	a, b, ok := e.s.Rahmen()
	zuschneiden := e.s.Werkzeug == canvas.WerkzeugZuschneiden
	if zuschneiden {
		a, b, ok = e.s.ZuschnittRahmen()
	}
	if !ok {
		return
	}
	rund := func(p document.Punkt) image.Point { return image.Pt(int(math.Round(p.X)), int(math.Round(p.Y))) }
	r := image.Rectangle{Min: rund(a), Max: rund(b)}.Canon()
	dunkel := farbeZuschnittDunkel
	if zuschneiden {
		f := e.s.Bildflaeche()
		bild := image.Rectangle{Min: rund(e.s.Abb.ZuFenster(document.Pt(float64(f.Min.X), float64(f.Min.Y)))),
			Max: rund(e.s.Abb.ZuFenster(document.Pt(float64(f.Max.X), float64(f.Max.Y))))}
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
	if zuschneiden {
		h := gtx.Dp(5)
		mx, my := (r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2
		for _, m := range []image.Point{r.Min, {mx, r.Min.Y}, {r.Max.X, r.Min.Y}, {r.Max.X, my}, r.Max, {mx, r.Max.Y},
			{r.Min.X, r.Max.Y}, {r.Min.X, my}} {
			q := image.Rect(m.X-h, m.Y-h, m.X+h, m.Y+h)
			paint.FillShape(gtx.Ops, farbeText, clip.Rect(q).Op())
			rahmen(gtx.Ops, q, gtx.Dp(1), farbeAuswahl)
		}
	}
}

// zeichneEinfuegemarke zeigt während der Texteingabe die Einfügemarke am Ende der letzten Zeile (orange wie der
// Tastaturfokus, Corporate Design §6), in Fensterpixeln. Lage und Breite der Zeile kommen aus dem Dokument
// (document.Text.Zeilen), wie in zeichneObjekte.
func (e *editor) zeichneEinfuegemarke(gtx layout.Context) {
	t, ok := e.s.TextObjekt()
	if !ok {
		return
	}
	zeilen := t.Zeilen()
	i := len(zeilen) - 1
	z := zeilen[i] // Anfang nach der Ausrichtung, Vorschub aus demselben Satz wie Renderer und Editor (ADR-0021 Nachtrag 10)
	// Von der Oberlänge (0,75 Schriftgrad über der Grundlinie) bis unter die Unterlänge (0,25 darunter).
	ort := e.s.D.ZuAnsicht(document.Pt(z.X, t.Ort.Y)).Plus(document.Pt(z.Vorschub, t.Schriftgrad*(0.25+document.Zeilenabstand*float64(i))))
	oben := e.s.Abb.ZuFenster(ort)
	unten := e.s.Abb.ZuFenster(ort.Plus(document.Pt(0, t.Schriftgrad)))
	x := int(math.Round(oben.X))
	r := image.Rect(x, int(math.Round(oben.Y)), x+gtx.Dp(2), int(math.Round(unten.Y)))
	paint.FillShape(gtx.Ops, farbeAkzent, clip.Rect(r).Op())
}

func (e *editor) zeiger(gtx layout.Context) {
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target:  e,
			Kinds:   pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel | pointer.Scroll | pointer.Move | pointer.Leave,
			ScrollX: pointer.ScrollRange{Min: -1e6, Max: 1e6},
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
		if ok && (pe.Kind == pointer.Move || pe.Kind == pointer.Leave) {
			// Nur der Zeigerort der Fußzeile; ein Folgebild nur, wenn sich das angezeigte Pixel ändert.
			if e.merkeZeiger(pe) {
				gtx.Execute(op.InvalidateCmd{})
			}
			continue
		}
		if ok {
			e.merkeZeiger(pe)
		}
		if !(probeOhneFolgebildBeiZug && ok && pe.Kind == pointer.Drag) {
			gtx.Execute(op.InvalidateCmd{})
		}
		if !ok || e.s.SpeichernLaeuft() { // während des Speicherns ruht das Bearbeiten (ADR-0020 Nachtrag 6)
			continue
		}
		p := document.Pt(float64(pe.Position.X), float64(pe.Position.Y))
		e.s.Umschalt = pe.Modifiers.Contain(key.ModShift) // Pfeil 45°, Quadrat (ADR-0021)
		e.s.SetzeAlt(pe.Modifiers.Contain(key.ModAlt))    // Alt schaltet das Einrasten ab (ADR-0021 Nachtrag 12)
		switch pe.Kind {
		case pointer.Press:
			if e.menue.offen {
				// Hält die Leinwand den Zeiger noch vom Rechtsklick, käme ein Linksdruck hier an und zöge unter dem
				// offenen Kontextmenü (Kritik K-4 #313a); wie bei Tasten (F-257) wirkt hinter einer offenen Liste nichts.
				continue
			}
			if os.Getenv("SCHNAPP_DEBUG") != "" {
				e.zug.beginne() // Bildzeiten dieses Zugs sammeln (F-250)
			}
			switch {
			case pe.Buttons.Contain(pointer.ButtonTertiary):
				e.s.Druecken(p, canvas.TasteMitte)
			case pe.Buttons == pointer.ButtonSecondary: // Kontextmenü (ADR-0038 Punkt 2)
				e.kontextRechtsklick(image.Pt(int(math.Round(float64(pe.Position.X))), int(math.Round(float64(pe.Position.Y)))))
			case pe.Buttons.Contain(pointer.ButtonPrimary) && e.s.Leertaste():
				// Leertaste gehalten: jeder Druck verschiebt die Ansicht, keiner zählt für einen Doppelklick (F-906).
				e.hatDruck = false
				e.s.Druecken(p, canvas.TastePrimaer)
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
			// ADR-0040: Strg+Rad zoomt zum Zeiger, das Rad allein rollt die Ansicht.
			zoom, nachlauf := e.rad.art(pe.Time, pe.Modifiers.Contain(key.ModShortcut))
			switch {
			case zoom && nachlauf:
				// Der Nachlauf des Touchpads zoomt nicht weiter (ADR-0040 Nachtrag 1 Punkt 8, F-923): Zoomen endet,
				// wenn die Finger ruhen.
			case zoom:
				e.s.Scrollen(p, radSchritt(pe.Scroll.Y))
			default:
				// Gio meldet Scroll in logischen Pixeln, Positionen in Fensterpixeln: mit der Skala malnehmen, damit
				// der Inhalt dem Finger folgt (ADR-0040 Risiko 2). PxPerDp enthält auch die Textskalierung des
				// Desktops (Gio: ppdp × Skala); weicht sie von 1 ab, rollt die Ansicht um diesen Faktor weiter als der
				// Finger (F-925, bewusste Grenze bis der Gio-Patch die Fensterskala allein liefert).
				k := float64(gtx.Metric.PxPerDp)
				e.s.Rollen(float64(pe.Scroll.X)*k, float64(pe.Scroll.Y)*k)
			}
		}
	}
}

// radGeste hält die Art einer Rad-Geste fest. Ein Ereignis, das höchstens radGesteZeit nach dem vorigen kommt (oder
// dieselbe Zeit trägt wie der Nachlauf von Gio), gehört zur selben Geste und behält ihre Art; erst eine neue Geste
// fragt Strg neu ab. So wird aus einem Nachlauf des Touchpads kein Zoomen, wenn Strg mittendrin gedrückt wird.
type radGeste struct {
	zeit time.Duration
	zoom bool
	gilt bool
}

const radGesteZeit = 150 * time.Millisecond

// art entscheidet für ein Rad-Ereignis zur Zeit t mit gedrücktem Strg (strg), ob es zoomt, und ob es Nachlauf ist
// (dieselbe Zeit wie das vorige Ereignis der Geste: Gio gibt Nachlauf-Ticks die Zeit des letzten echten Ereignisses).
// Das Fenster gleitet: Jedes Ereignis der Geste verlängert sie um radGesteZeit.
func (r *radGeste) art(t time.Duration, strg bool) (zoom, nachlauf bool) {
	nachlauf = r.gilt && t == r.zeit
	if !r.gilt || t < r.zeit || t-r.zeit > radGesteZeit {
		r.zoom = strg
	}
	r.zeit, r.gilt = t, true
	return r.zoom, nachlauf
}

// radSchritt rechnet das Mausrad in Zoomschritte zu 10 % um. Gio liefert Pixel; ein Rastschritt des Mausrads kommt
// als 150 px (3 Schritte, Obergrenze). Eine untere Grenze gibt es nicht (F-923): Das Touchpad meldet viele kleine
// Strecken, und der Zoom folgt dann der Strecke des Fingers statt je Ereignis mindestens 2,4 %.
func radSchritt(scrollY float32) float64 {
	dy := float64(scrollY) / 40
	return math.Copysign(math.Min(3, math.Abs(dy)), dy)
}

// zeigerAnsicht ist der Zeigerort in Ansichtskoordinaten für die Fußzeile (nil: Zeiger nicht im Fenster). Er wird aus
// der Fensterposition bei jedem Bild neu gerechnet, damit Zoom und Verschieben per Taste ihn nicht veralten lassen
// (F-513).
func (e *editor) zeigerAnsicht() *document.Punkt {
	if e.zeigerFenster == nil {
		return nil
	}
	p := e.s.Abb.ZuAnsicht(*e.zeigerFenster)
	return &p
}

// systemZeiger bildet die Zeigerform der Sitzung auf die Zeiger des Systems ab (Karte #323).
func systemZeiger(z canvas.ZeigerArt) pointer.Cursor {
	switch z {
	case canvas.ZeigerFadenkreuz:
		return pointer.CursorCrosshair
	case canvas.ZeigerText:
		return pointer.CursorText
	case canvas.ZeigerVerschieben:
		return pointer.CursorAllScroll
	case canvas.ZeigerGreifen:
		return pointer.CursorGrab
	case canvas.ZeigerGreifend:
		return pointer.CursorGrabbing
	case canvas.ZeigerNWSO:
		return pointer.CursorNorthWestSouthEastResize
	case canvas.ZeigerNOSW:
		return pointer.CursorNorthEastSouthWestResize
	case canvas.ZeigerNS:
		return pointer.CursorNorthSouthResize
	case canvas.ZeigerOW:
		return pointer.CursorEastWestResize
	}
	return pointer.CursorDefault
}

// merkeZeiger hält die Fensterposition des Zeigers für die Fußzeile fest und meldet, ob sich das angezeigte Pixel
// oder die Form des Zeigers geändert hat (dann braucht es ein Folgebild).
func (e *editor) merkeZeiger(pe pointer.Event) bool {
	zeigt := func() (image.Point, bool) {
		if a := e.zeigerAnsicht(); a != nil {
			return e.s.ZeigerOrt(*a)
		}
		return image.Point{}, false
	}
	altOrt, altDa := zeigt()
	altForm := e.zeigerForm
	if pe.Kind == pointer.Leave {
		e.zeigerFenster = nil
	} else {
		p := document.Pt(float64(pe.Position.X), float64(pe.Position.Y))
		e.zeigerFenster = &p
	}
	ort, da := zeigt()
	e.zeigerForm = e.s.Zeiger(e.zeigerFenster)
	return altDa != da || altOrt != ort || altForm != e.zeigerForm
}

// zeichneObjekte zeichnet in Ansichtskoordinaten (die Affine-Abbildung liegt schon auf dem Stapel). Schrift in
// Ansichtspixeln: eigene Metrik mit 1 px je sp.
func (e *editor) zeichneObjekte(gtx layout.Context) {
	d := e.s.D
	an := func(p document.Punkt) f32.Point {
		q := d.ZuAnsicht(p)
		return f32.Pt(float32(q.X), float32(q.Y))
	}
	sicht := image.Rectangle{Max: d.Groesse()}
	defer e.lupenAufraeumen()
	defer e.schattenAufraeumen()
	// Über vielen Nähten zeigt der Editor ab bereichGrenze abgespielten Bereichen die Objektebene des Renderers statt
	// eigener Pfade (ADR-0023 Nachtrag 20, F-787): Gio rastert jedes Bild alle Bereiche neu, an der Grenze von
	// render.Pruefe rund 500 000 (120 gefüllte runde Kästen über 4096 Kacheln: 1,7 s je Bild mit llvmpipe, Dateien über
	// dem Budget öffnet --ignore-limit beliebig teuer). Die Ebene gilt, bis sich Dokument, Pixel oder Objekte ändern;
	// dann zählt der Editor neu.
	if e.objEbene.passt(e) {
		e.objEbene.zeige(gtx)
		return
	}
	m := op.Record(gtx.Ops)
	e.bereiche = 0
	for _, o := range d.Objekte() {
		if !d.ObjektSichtbar(o) || o.Ebenenwerte().Ausgeblendet {
			continue // ganz im Verborgenen oder ausgeblendet: wie Renderer und SVG-Export (ADR-0023 Punkt 7, ADR-0032)
		}
		e.zeichneObjekt(gtx, o, an, sicht, false)
	}
	aufruf := m.Stop()
	e.objEbene.bereiche = e.bereiche
	if e.bereiche > bereichGrenze && e.objEbene.baue(e) {
		e.objEbene.zeige(gtx)
		return
	}
	if e.objEbene.an || e.objEbene.objekte != nil {
		e.objEbene.frei()
	}
	aufruf.Add(gtx.Ops)
}

// zeichneObjekt zeichnet o; an bildet Originalpunkte in die Zeichenfläche ab, sicht begrenzt die Textzeilen (in
// derselben Fläche). Die Deckkraft der Ebene wirkt als Gios Deckkraft-Ebene auf das Objekt als Ganzes (ADR-0032 Punkt
// 10). Leinwand und Vorschau der Ebenenliste zeichnen über diese eine Funktion; vorschau sagt, dass an das Original
// ohne Ansicht abbildet (das Bild einer Lupe liegt dann nicht in Ansichtskoordinaten, ADR-0033).
func (e *editor) zeichneObjekt(gtx layout.Context, o document.Objekt, an func(document.Punkt) f32.Point, sicht image.Rectangle,
	vorschau bool) {
	if p := o.Ebenenwerte().Deckkraft(); o.Ebenenwerte().Multipliziert() {
		// Textmarker: Gios Ebene, multiplizierend eingemischt (Gio-Patch, ADR-0048 Punkt 3)
		defer paint.PushMultiply(gtx.Ops, float32(p)/100).Pop()
	} else if p < 100 {
		defer paint.PushOpacity(gtx.Ops, float32(p)/100).Pop()
	}
	if !vorschau { // Schatten unter dem Objekt, in der Deckkraft der Ebene (ADR-0021 Nachtrag 26 Punkt 4)
		e.zeichneSchatten(gtx, o)
	}
	switch v := o.(type) {
	case document.Pfeil:
		// Abgeschnitten wie Renderer und SVG-Export (F-748, ADR-0023 Nachtrag 17): einmal im Original aufgezeichnet und in
		// jedem Bereich abgespielt, den eine Strecke samt Kopf berührt (document.StreckenStuecke). Dieselben
		// überlappungsfreien Teile wie der Renderer (render.ZerlegePfeil, ADR-0021 Nachtrag 19): Gios Strich hätte nur
		// runde Enden, der Schaft endet aber unter dem Kopf.
		pz := render.ZerlegePfeil(v)
		var stuecke []document.Stueck
		if !vorschau && len(pz.Schaft) > 0 {
			for _, st := range e.s.D.StreckenStuecke(pz.Schaft, false, pz.Suche) {
				stuecke = append(stuecke, st.Stueck)
			}
		}
		e.jeStueckIn(gtx, stuecke, an, vorschau, func(abb func(document.Punkt) f32.Point) {
			fuelleTeile(gtx, pz.Teile, v.Strich.Farbe, abb)
		})
		if len(pz.Einzeln) > 0 {
			// Köpfe ohne Schaft in einer Füllung: Überlappen sie (kurzer Pfeil mit zwei Spitzen), deckt eine
			// halbdurchsichtige Farbe trotzdem einmal (Sitzungs-Kritik 26.11. K-6).
			h := document.Rechteck{Min: pz.Einzeln[0][0], Max: pz.Einzeln[0][0]}
			for _, k := range pz.Einzeln {
				for _, q := range k {
					h = h.Vereint(document.Rechteck{Min: q, Max: q})
				}
			}
			e.jeStueck(gtx, h, 1, an, vorschau, func(abb func(document.Punkt) f32.Point) {
				fuelleTeile(gtx, pz.Einzeln, v.Strich.Farbe, abb)
			})
		}
	case document.Kasten:
		if v.Ellipse() {
			e.zeichneEllipse(gtx, v, an, vorschau)
			break
		}
		rechteck := func(abb func(document.Punkt) f32.Point) (image.Rectangle, clip.RRect) {
			min, max := abb(v.Bereich.Min), abb(v.Bereich.Max)
			r := image.Rect(int(math.Round(float64(min.X))), int(math.Round(float64(min.Y))),
				int(math.Round(float64(max.X))), int(math.Round(float64(max.Y))))
			return r, clip.UniformRRect(r, kastenRundung(v, r))
		}
		fuellung := func(abb func(document.Punkt) f32.Point) {
			if v.Fuellung.A > 0 {
				_, rr := rechteck(abb)
				paint.FillShape(gtx.Ops, v.Fuellung, rr.Op(gtx.Ops))
			}
		}
		strich := func(abb func(document.Punkt) f32.Point) {
			r, rr := rechteck(abb)
			if v.Strich.Breite > 0 && v.Strich.Farbe.A > 0 {
				if eckig := kastenStrich(v, r); eckig != nil && v.Strich.Art != document.StrichVoll {
					// Gemustert: dieselben Rechtecke bzw. Scheiben wie der Renderer (ADR-0021 Nachtrag 25).
					q := document.R(document.Pt(float64(r.Min.X), float64(r.Min.Y)), document.Pt(float64(r.Max.X), float64(r.Max.Y)))
					fuelleTeile(gtx, render.RahmenTeile(q, v.Strich), v.Strich.Farbe, punktF32)
				} else if eckig != nil {
					// Ohne Rundung spitze Außenecken (F-498); Gios Strich kennt nur runde Ecken.
					var pfad clip.Path
					pfad.Begin(gtx.Ops)
					for _, q := range eckig {
						pfad.MoveTo(f32.Pt(float32(q.Min.X), float32(q.Min.Y)))
						pfad.LineTo(f32.Pt(float32(q.Max.X), float32(q.Min.Y)))
						pfad.LineTo(f32.Pt(float32(q.Max.X), float32(q.Max.Y)))
						pfad.LineTo(f32.Pt(float32(q.Min.X), float32(q.Max.Y)))
						pfad.Close()
					}
					paint.FillShape(gtx.Ops, v.Strich.Farbe, clip.Outline{Path: pfad.End()}.Op())
				} else {
					paint.FillShape(gtx.Ops, v.Strich.Farbe, clip.Stroke{Path: rr.Path(gtx.Ops), Width: float32(v.Strich.Breite)}.Op())
				}
			}
		}
		if b := document.R(v.Bereich.Min, v.Bereich.Max); document.WirksamerEckradius(v.Eckradius, b.Breite(), b.Hoehe()) > 0 {
			// Gerundet: je Bereich abgeschnitten wie Renderer und SVG-Export statt über die Ecken neu gerundet (F-749,
			// ADR-0023 Nachtrag 17). Die Füllung in jedem Bereich des Rechtecks, der Strich wie im Renderer nur in den
			// Bereichen, die sein Umriss berührt (document.StreckenStuecke von document.RundesRechteck; F-765, ADR-0023
			// Nachtrag 18): Über 4096 Kacheln spielte der Editor sonst jeden Strich in allen Bereichen ab. Die Teile des
			// Strichs sind dieselben wie im Renderer (document.NeuerKastenStrich, ADR-0023 Nachtrag 21), auch in der
			// Vorschau der Ebenenliste (F-789: Gios Strich zeigte dort bei Strich > 2·Radius ein Loch).
			if v.Fuellung.A > 0 { // ohne Füllung kein leeres Abspielen in jedem Bereich (F-787)
				e.jeStueck(gtx, b, v.Strich.Breite/2+1, an, vorschau, fuellung)
			}
			if v.Strich.Breite > 0 && v.Strich.Farbe.A > 0 {
				if v.Strich.Art != document.StrichVoll {
					// Gemustert (ADR-0021 Nachtrag 25): dieselben Teile wie der Renderer, in den Bereichen, die die
					// Striche berühren — wie der Schaft eines Pfeils.
					sz := render.ZerlegeRundenRahmen(b, v.Eckradius, v.Strich)
					var stuecke []document.Stueck
					if !vorschau {
						for _, st := range e.s.D.StreckenStuecke(sz.Zug, false, sz.Suche) {
							stuecke = append(stuecke, st.Stueck)
						}
					}
					e.jeStueckIn(gtx, stuecke, an, vorschau, func(abb func(document.Punkt) f32.Point) {
						fuelleTeile(gtx, sz.Teile, v.Strich.Farbe, abb)
					})
					break
				}
				ks := document.NeuerKastenStrich(b, v.Eckradius, v.Strich.Breite/2)
				if vorschau {
					kastenStrichVorschau(gtx, ks, v.Strich.Farbe, an)
				} else {
					e.strichJeStueck(gtx, ks, v.Strich)
				}
			}
		} else {
			fuellung(an)
			strich(an)
		}
	case document.Nummer:
		// Je sichtbarem Bereich und auf ihn beschnitten wie Renderer und SVG-Export (F-745, ADR-0023 Nachtrag 15).
		e.jeStueck(gtx, document.Reichweite(v), 0, an, vorschau, func(abb func(document.Punkt) f32.Point) {
			m := abb(v.Mitte)
			rad := float32(v.Radius)
			kr := image.Rect(int(math.Round(float64(m.X-rad))), int(math.Round(float64(m.Y-rad))),
				int(math.Round(float64(m.X+rad))), int(math.Round(float64(m.Y+rad))))
			paint.FillShape(gtx.Ops, v.Farbe, clip.Ellipse(kr).Op(gtx.Ops))
			// Grundlinie wie im Renderer: eine halbe Versalhöhe unter der Mitte (document.Nummer.Grundlinie).
			e.textBei(gtx, fmt.Sprint(v.Wert), float32(v.Radius*1.2), f32.Pt(m.X, m.Y+nummerGrundlinieAnsicht(v)), true,
				color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		})
	case document.Lupe:
		e.zeichneLupe(gtx, v, an, vorschau)
	case document.Text:
		// Wie der Renderer (F-735): Liegt nur die Spitze in der Ansicht, nur das Dreieck, keine Zeile. Die Vorschau der
		// Ebenenliste zeigt das Objekt ganz.
		nurSpitze := !vorschau && e.s.D.NurSpitzeSichtbar(v)
		if sa, sp, sb, ok := v.Spitzendreieck(); ok {
			// Sprechblase (ADR-0021 Nachtrag 13 Punkte 2 und 3): Schild samt Dreieck als ein Vieleck in der Füllfarbe, ohne
			// Füllung nur die beiden Schenkel in der Schriftfarbe — dieselben Funktionen des Dokuments wie der Renderer.
			umriss := document.Sprechblase(v)
			if nurSpitze {
				umriss = []document.Punkt{sa, sp, sb}
			}
			r := document.Rechteck{Min: umriss[0], Max: umriss[0]}
			for _, p := range umriss[1:] {
				r = r.Vereint(document.Rechteck{Min: p, Max: p})
			}
			// Je sichtbarem Bereich und auf ihn beschnitten wie Renderer, SVG-Export und Treffer (F-695).
			e.jeStueck(gtx, r, v.SchenkelBreite()/2+1, an, vorschau, func(abb func(document.Punkt) f32.Point) {
				if v.Hintergrund.A > 0 {
					var pfad clip.Path
					pfad.Begin(gtx.Ops)
					for i, p := range umriss {
						if i == 0 {
							pfad.MoveTo(abb(p))
						} else {
							pfad.LineTo(abb(p))
						}
					}
					pfad.Close()
					paint.FillShape(gtx.Ops, v.Hintergrund, clip.Outline{Path: pfad.End()}.Op())
				} else {
					var pfad clip.Path
					pfad.Begin(gtx.Ops)
					pfad.MoveTo(abb(sa))
					pfad.LineTo(abb(sp))
					pfad.LineTo(abb(sb))
					paint.FillShape(gtx.Ops, v.Farbe, clip.Stroke{Path: pfad.End(), Width: float32(v.SchenkelBreite())}.Op())
				}
			})
		} else if v.Hintergrund.A > 0 && !nurSpitze {
			g := v.Grenzen()
			a, b := an(g.Min), an(g.Max)
			paint.FillShape(gtx.Ops, v.Hintergrund, clip.Rect(image.Rect(int(a.X), int(a.Y), int(b.X), int(b.Y))).Op())
		}
		if nurSpitze {
			return
		}
		// Zeilen und Striche wie im Renderer: Schrift, Anfang nach der Ausrichtung und Grundlinie aus dem Dokument,
		// Glyphen aus den Umrissen derselben Datei (ADR-0021 Nachtrag 10 Punkt 5).
		sch := v.Schriftdatei()
		// Während der Eingabe steht der bearbeitete Text ungeschnitten ab dem abgebildeten Anfang jeder Zeile, wie die
		// Einfügemarke: Liegt sein Ende im Streifen, sähe man sonst nicht, was man tippt (F-752, ADR-0023 Nachtrag 17).
		// Nach der Eingabe ist er wieder beschnitten; Renderer, Export und Datei ändern sich nicht.
		eingabe, inEingabe := e.s.TextObjekt()
		if vorschau || (inEingabe && eingabe.ID == v.ID) {
			for _, zeile := range v.Zeilen() {
				p := an(document.Pt(zeile.X, zeile.Grundlinie))
				e.satz.zeichneZeile(gtx, sch, zeile.Text, float64(p.X), float64(p.Y), v.Schriftgrad, v.Farbe, sicht)
			}
		} else {
			// Glyphen je sichtbarem Bereich wie der Renderer (F-739, ADR-0023 Nachtrag 13): um den Versatz des Bereichs
			// verschoben und immer auf ihn beschnitten, auch bei nur einem (Kritik A-1, [S]). Zeilenweise über die Abschnitte
			// (document.Abschnitte, F-747, F-750, ADR-0023 Nachtrag 16): Jede Zeile sucht die senkrechten Abschnitte ihrer
			// Höhe per Binärsuche, wird einmal im Original als Pfad aufgezeichnet und je Bereich nur verschoben abgespielt.
			xs, ys := e.s.D.TextAbschnitte(v)
			spalten := document.Beschneide(xs, sicht.Min.X, sicht.Max.X)
			reihen := document.Beschneide(ys, sicht.Min.Y, sicht.Max.Y)
			if len(spalten) > 0 && len(reihen) > 0 {
				oben, unten, seitlich := sch.Huelle()
				grad := v.Schriftgrad
				for _, zeile := range v.Zeilen() {
					rs := document.AbschnitteIn(reihen, zeile.Grundlinie-grad*oben, zeile.Grundlinie+grad*unten)
					ss := document.AbschnitteIn(spalten, zeile.X-seitlich*grad, math.Inf(1))
					if len(rs) == 0 || len(ss) == 0 {
						continue
					}
					// Sicht im Original: alle Spalten ab der ersten berührten, die Reihen der Zeile.
					fenster := image.Rect(ss[0].A, rs[0].A, ss[len(ss)-1].B, rs[len(rs)-1].B)
					laeufe, n := e.satz.zeileInLaeufen(gtx, sch, zeile.Text, zeile.X, zeile.Grundlinie, grad, v.Farbe, fenster)
					if n == 0 {
						continue
					}
					// Jeder Lauf nur in den Spalten, die seine Tinte berührt (F-765, ADR-0023 Nachtrag 18): rechts endet die
					// Zeile mit ihrer letzten Glyphe, und eine lange Zeile spielt je Spalte nur ihre Glyphen in der Nähe.
					for _, l := range laeufe {
						for _, sp := range document.AbschnitteIn(ss, l.von, l.bis) {
							e.bereiche += len(rs)
							for _, ze := range rs {
								versatz := image.Pt(sp.Versatz, ze.Versatz)
								schnitt := clip.Rect(image.Rect(sp.A, ze.A, sp.B, ze.B).Add(versatz)).Push(gtx.Ops)
								t := op.Offset(versatz).Push(gtx.Ops)
								l.aufruf.Add(gtx.Ops)
								t.Pop()
								schnitt.Pop()
							}
						}
					}
				}
			}
		}
		for _, st := range v.Striche() { // mit gebrochenen Ecken wie der Renderer (Sitzungs-QA B-9)
			a, b := an(st.Min), an(st.Max)
			var sp clip.Path
			sp.Begin(gtx.Ops)
			sp.MoveTo(a)
			sp.LineTo(f32.Pt(b.X, a.Y))
			sp.LineTo(b)
			sp.LineTo(f32.Pt(a.X, b.Y))
			sp.Close()
			paint.FillShape(gtx.Ops, v.Farbe, clip.Outline{Path: sp.End()}.Op())
		}
	}
}

// jeStueck zeichnet f je sichtbarem Bereich der Ansicht, den r (Original, um rand vergrößert) berührt, auf diesen Bereich
// beschnitten und mit der Abbildung Original → Ansicht in diesem Bereich (document.Stuecke): Schräge Kanten über einer
// Naht stehen so wie im Renderer, im SVG-Export und im Treffer (F-695, ADR-0023 Nachtrag 14). Die Vorschau der
// Ebenenliste bildet das Original ohne Ansicht ab (an) und zeichnet einmal ohne Schnitt.
func (e *editor) jeStueck(gtx layout.Context, r document.Rechteck, rand float64, an func(document.Punkt) f32.Point,
	vorschau bool, f func(abb func(document.Punkt) f32.Point)) {
	if vorschau {
		f(an)
		return
	}
	r = document.Rechteck{Min: r.Min.Minus(document.Pt(rand, rand)), Max: r.Max.Plus(document.Pt(rand, rand))}
	e.jeStueckIn(gtx, e.s.D.Stuecke(r), an, false, f)
}

// jeStueckIn zeichnet f einmal im Original auf und spielt es in jedem der Bereiche stuecke ab, auf ihn beschnitten und um
// seinen Versatz (ganze Pixel) verschoben: Über vielen Nähten baute der Editor sonst je Bereich Pfad und Schrift neu
// (F-750, ADR-0023 Nachtrag 16). Die Vorschau der Ebenenliste zeichnet einmal mit an, ohne Schnitt.
func (e *editor) jeStueckIn(gtx layout.Context, stuecke []document.Stueck, an func(document.Punkt) f32.Point,
	vorschau bool, f func(abb func(document.Punkt) f32.Point)) {
	if vorschau {
		f(an)
		return
	}
	if len(stuecke) == 0 {
		return
	}
	m := op.Record(gtx.Ops)
	f(func(p document.Punkt) f32.Point { return f32.Pt(float32(p.X), float32(p.Y)) })
	aufruf := m.Stop()
	e.bereiche += len(stuecke)
	for _, st := range stuecke {
		z := st.Ziel()
		schnitt := clip.Rect(image.Rect(int(math.Round(z.Min.X)), int(math.Round(z.Min.Y)), int(math.Round(z.Max.X)),
			int(math.Round(z.Max.Y)))).Push(gtx.Ops)
		t := op.Offset(image.Pt(int(math.Round(st.Versatz.X)), int(math.Round(st.Versatz.Y)))).Push(gtx.Ops)
		aufruf.Add(gtx.Ops)
		t.Pop()
		schnitt.Pop()
	}
}

// punktF32 bildet einen Punkt ohne Umrechnung ins Fenster ab (Teile, die schon in Fensterkoordinaten stehen).
func punktF32(p document.Punkt) f32.Point { return f32.Pt(float32(p.X), float32(p.Y)) }

// fuelleTeile füllt die Vielecke teile (Original, über abb ins Fenster) als einen Pfad: Überlappungsfreie Teile mit
// gemeinsamen Kanten decken dann lückenlos (wie kastenStrichVorschau). Jedes Vieleck läuft dabei gleich herum wie im
// Renderer (render.raster): Gegenläufige Nachbarn höben sich an ihrer gemeinsamen Kante auf, eine helle Fuge — so stand
// sie am Ansatz der Pfeilspitze (Sechseck des Schafts gegen den Kopf) und in den Fächern gebogener Striche
// (ADR-0021 Nachtrag 25 Punkt 11).
func fuelleTeile(gtx layout.Context, teile [][]document.Punkt, farbe color.NRGBA, abb func(document.Punkt) f32.Point) {
	var pfad clip.Path
	pfad.Begin(gtx.Ops)
	for _, t := range teile {
		if len(t) < 3 {
			continue
		}
		if document.VieleckFlaeche(t) < 0 {
			pfad.MoveTo(abb(t[len(t)-1]))
			for i := len(t) - 2; i >= 0; i-- {
				pfad.LineTo(abb(t[i]))
			}
		} else {
			pfad.MoveTo(abb(t[0]))
			for _, q := range t[1:] {
				pfad.LineTo(abb(q))
			}
		}
		pfad.Close()
	}
	paint.FillShape(gtx.Ops, farbe, clip.Outline{Path: pfad.End()}.Op())
}

// kastenStrichVorschau füllt die Teile des Strichs eines runden Kastens (document.NeuerKastenStrich) als einen Pfad, mit
// an abgebildet: die Vorschau der Ebenenliste, ohne Nähte. Dieselben Teile wie Leinwand und Renderer (F-789).
func kastenStrichVorschau(gtx layout.Context, ks document.KastenStrich, farbe color.NRGBA, an func(document.Punkt) f32.Point) {
	if len(ks.Teile) == 0 {
		return
	}
	var pfad clip.Path
	pfad.Begin(gtx.Ops)
	for _, teil := range ks.Teile {
		pfad.MoveTo(an(teil[0]))
		for _, p := range teil[1:] {
			pfad.LineTo(an(p))
		}
		pfad.Close()
	}
	paint.FillShape(gtx.Ops, farbe, clip.Outline{Path: pfad.End()}.Op())
}

// strichJeStueck zeichnet den Strich eines runden Kastens aus seinen Teilen ks (Original; document.NeuerKastenStrich:
// konvex, überlappungsfrei, gleich herum, dieselben wie im Renderer, ADR-0023 Nachtrag 21) je sichtbarem Bereich, den
// sein Umriss berührt (document.StreckenStuecke mit ks.Rand), und dort nur die Teile der Strecken in der Nähe — in
// denselben Bereichen wie der Renderer (F-765, ADR-0023 Nachträge 18 und 19). Jeder Teil ist selbst auf den Bereich
// beschnitten und um dessen Versatz verschoben; weil die Teile nicht überlappen, addiert Gio keine Kantenglättung
// doppelt, und weil ihre inneren Ecken nie gegeneinander laufen, zeigt ein kleiner runder Kasten kein Loch (F-783,
// F-792). Ein Bereich, den die Teile ganz decken, ist ein Rechteck; benachbarte volle Bereiche einer Zeile werden ein
// Rechteck (F-782: Gio rastert sonst je Bereich alle Kanten der Teile). Alle zusammen sind ein Pfad mit einer Füllung.
// Eine Füllung mit Schnitt je Bereich kostete Gio über 4096 Kacheln rund 40 µs je Bereich mit llvmpipe (200 Kästen:
// 3,3 s je Bild).
func (e *editor) strichJeStueck(gtx layout.Context, ks document.KastenStrich, s document.Strich) {
	if len(ks.Teile) == 0 {
		return
	}
	var pfad clip.Path
	pfad.Begin(gtx.Ops)
	vieleck := func(ecken []document.Punkt, v document.Punkt) {
		for k, p := range ecken {
			p = p.Plus(v)
			if k == 0 {
				pfad.MoveTo(f32.Pt(float32(p.X), float32(p.Y)))
			} else {
				pfad.LineTo(f32.Pt(float32(p.X), float32(p.Y)))
			}
		}
		pfad.Close()
	}
	// Lauf voller Bereiche in einer Zeile (Ansicht, ganze Pixel), gleich herum wie die Vierecke.
	var lauf document.Rechteck
	laufOffen := false
	laufEnde := func() {
		if laufOffen {
			r := []document.Punkt{lauf.Min, document.Pt(lauf.Max.X, lauf.Min.Y), lauf.Max,
				document.Pt(lauf.Min.X, lauf.Max.Y)}
			if ks.Umlauf < 0 {
				slices.Reverse(r)
			}
			vieleck(r, document.Punkt{})
			laufOffen = false
		}
	}
	groesst := 0
	for _, t := range ks.Teile {
		groesst = max(groesst, len(t))
	}
	// Ein konvexes Vieleck mit m Ecken hat nach den vier Schnitten höchstens m+4.
	arbeit, puffer := make([]document.Punkt, 0, groesst+4), make([]document.Punkt, 0, groesst+4)
	gesehen, marke := make([]int, len(ks.Teile)), 0
	var teile []document.Punkt // beschnittene Vielecke eines Bereichs hintereinander
	var laengen []int
	for _, st := range e.s.D.StreckenStuecke(ks.Umriss, true, ks.Rand) {
		z := st.Ziel()
		v := document.Pt(math.Round(st.Versatz.X), math.Round(st.Versatz.Y))
		q := document.Rechteck{Min: document.Pt(math.Round(z.Min.X), math.Round(z.Min.Y)).Minus(v),
			Max: document.Pt(math.Round(z.Max.X), math.Round(z.Max.Y)).Minus(v)}
		teile, laengen = teile[:0], laengen[:0]
		flaeche := 0.0
		marke++
		for _, i := range st.Strecken {
			t := ks.TeilVon[i]
			if gesehen[t] == marke {
				continue // Fächer mehrerer Strecken: einmal je Bereich
			}
			gesehen[t] = marke
			g := ks.Grenzen[t]
			if g.Max.X <= q.Min.X || g.Min.X >= q.Max.X || g.Max.Y <= q.Min.Y || g.Min.Y >= q.Max.Y {
				continue
			}
			ecken := schneideVieleck(append(arbeit[:0], ks.Teile[t]...), q, puffer[:0])
			if len(ecken) < 3 {
				continue
			}
			flaeche += math.Abs(document.VieleckFlaeche(ecken))
			teile = append(teile, ecken...)
			laengen = append(laengen, len(ecken))
		}
		if len(laengen) == 0 {
			continue
		}
		if flaeche >= q.Breite()*q.Hoehe()-1e-6 {
			// Ganz gedeckt (die Vierecke überlappen nicht): ein Rechteck, an einen Lauf derselben Zeile angehängt.
			r := document.Rechteck{Min: q.Min.Plus(v), Max: q.Max.Plus(v)}
			if laufOffen && r.Min.Y == lauf.Min.Y && r.Max.Y == lauf.Max.Y && r.Min.X == lauf.Max.X {
				lauf.Max.X = r.Max.X
				continue
			}
			laufEnde()
			lauf, laufOffen = r, true
			e.bereiche++ // ein Rechteck je Lauf voller Bereiche
			continue
		}
		laufEnde()
		e.bereiche++
		k := 0
		for _, l := range laengen {
			vieleck(teile[k:k+l], v)
			k += l
		}
	}
	laufEnde()
	paint.FillShape(gtx.Ops, s.Farbe, clip.Outline{Path: pfad.End()}.Op())
}

// schneideVieleck schneidet das konvexe Vieleck p auf das Rechteck r zu (Sutherland-Hodgman, Umlaufsinn bleibt) und
// liefert das Ergebnis in p (Platz für m+4 Ecken bei m Ecken vorher; ein Viereck hat nach vier Schnitten höchstens 8).
// puffer ist Arbeitsplatz derselben Größe.
func schneideVieleck(p []document.Punkt, r document.Rechteck, puffer []document.Punkt) []document.Punkt {
	schnitt := func(ein []document.Punkt, aus []document.Punkt, innen func(document.Punkt) bool,
		kreuz func(a, b document.Punkt) document.Punkt) []document.Punkt {
		aus = aus[:0]
		for k := range ein {
			a, b := ein[k], ein[(k+1)%len(ein)]
			ia, ib := innen(a), innen(b)
			if ia {
				aus = append(aus, a)
			}
			if ia != ib {
				aus = append(aus, kreuz(a, b))
			}
		}
		return aus
	}
	senkrecht := func(x float64) func(a, b document.Punkt) document.Punkt {
		return func(a, b document.Punkt) document.Punkt {
			return document.Pt(x, a.Y+(b.Y-a.Y)*(x-a.X)/(b.X-a.X))
		}
	}
	waagerecht := func(y float64) func(a, b document.Punkt) document.Punkt {
		return func(a, b document.Punkt) document.Punkt {
			return document.Pt(a.X+(b.X-a.X)*(y-a.Y)/(b.Y-a.Y), y)
		}
	}
	p2 := schnitt(p, puffer, func(q document.Punkt) bool { return q.X >= r.Min.X }, senkrecht(r.Min.X))
	p = schnitt(p2, p, func(q document.Punkt) bool { return q.X <= r.Max.X }, senkrecht(r.Max.X))
	p2 = schnitt(p, p2, func(q document.Punkt) bool { return q.Y >= r.Min.Y }, waagerecht(r.Min.Y))
	return schnitt(p2, p, func(q document.Punkt) bool { return q.Y <= r.Max.Y }, waagerecht(r.Max.Y))
}

// kastenRundung ist der Eckradius in Bildpunkten, mit dem der Editor den Kasten auf dem ganzzahligen Rechteck r
// zeichnet: derselbe wirksame Radius wie in Renderer und SVG-Export (document.WirksamerEckradius), auf ganze Punkte
// gerundet, weil Gios clip.UniformRRect ganzzahlig rechnet. Eigene Funktion, damit ein Test ihn ohne Gio-Zeichnen mit
// Renderer und SVG vergleicht (F-519).
func kastenRundung(v document.Kasten, r image.Rectangle) int {
	return int(math.Round(document.WirksamerEckradius(v.Eckradius, float64(r.Dx()), float64(r.Dy()))))
}

// kastenStrich liefert die Rechtecke des Strichs, wenn der Kasten auf dem ganzzahligen Rechteck r ohne Rundung
// gezeichnet wird (WirksamerEckradius = 0, F-498), sonst nil: dann zeichnet der Editor mit Gios rundem Strich. Eigene
// Funktion, damit ein Test die Wahl ohne Gio-Zeichnen prüft (Kritik A-2, Mutanten Q15/E11).
func kastenStrich(v document.Kasten, r image.Rectangle) []document.Rechteck {
	if document.WirksamerEckradius(v.Eckradius, float64(r.Dx()), float64(r.Dy())) != 0 {
		return nil
	}
	return kastenRahmen(r, v.Strich.Breite)
}

// kastenRahmen sind die Rechtecke des eckigen Kasten-Strichs auf dem ganzzahligen Rechteck r (document.EckigerRahmen,
// F-498); eigene Funktion, damit ein Test sie ohne Gio-Zeichnen prüfen kann.
func kastenRahmen(r image.Rectangle, strichBreite float64) []document.Rechteck {
	return document.EckigerRahmen(document.R(document.Pt(float64(r.Min.X), float64(r.Min.Y)),
		document.Pt(float64(r.Max.X), float64(r.Max.Y))), strichBreite)
}

// textGrundlinieAnsicht ist die Grundlinie von Zeile i relativ zu Ort.Y (wie document.Text.Grundlinie, F-342).
func textGrundlinieAnsicht(grad float64, i int) float32 {
	return float32(grad * (1 + document.Zeilenabstand*float64(i)))
}

// nummerGrundlinieAnsicht ist der Abstand der Ziffern-Grundlinie von der Mitte (document.Nummer.Grundlinie, F-342).
func nummerGrundlinieAnsicht(n document.Nummer) float32 { return float32(n.Grundlinie() - n.Mitte.Y) }

// zeichneNaehte legt die Kanten der Stauch-Nähte über Bild und Objekte, in Ansichtskoordinaten und mit denselben
// Vielecken und Schichten wie Renderer und SVG-Export (ADR-0023 Punkt 6, Nachträge 6 und 7; den durchsichtigen Spalt
// zeigt ein Schachbrett, schachbrett.go). Die Pfade werden nur neu gebaut,
// wenn sich Ansichtsgröße, Nähte, Kantenform oder KantenFaktor ändern; sonst spielt der Editor die Aufzeichnung ab
// (F-183, e.naht).
func (e *editor) zeichneNaehte(gtx layout.Context) {
	g, naehte, form, faktor := e.s.D.Groesse(), e.s.D.Naehte(), e.s.D.Kante(), e.s.D.KantenFaktor()
	c := &e.naht
	saumImBild := e.kantenStand.im // Fasersaum und Abdunklung liegen im Bild der Leinwand (bildNeu, F-407)
	if !c.gueltig || c.groesse != g || !slices.Equal(c.naehte, naehte) || c.form != form || c.faktor != faktor ||
		c.saumImBild != saumImBild {
		c.ops.Reset()
		aufnahme := op.Record(&c.ops)
		schichten := document.KantenSchichten()
		// Mittellinie je Achse einmal, die Schichten verschieben sie nur quer (F-418); ein Puffer für alle Bänder.
		linien := nahtLinien{g: g, form: form, faktor: faktor}
		var band []document.Punkt
		linie := linien.fuer
		pfad := func(band []document.Punkt) clip.PathSpec {
			var p clip.Path
			p.Begin(&c.ops)
			for i, q := range band {
				pt := f32.Pt(float32(q.X), float32(q.Y))
				if i == 0 {
					p.MoveTo(pt)
				} else {
					p.LineTo(pt)
				}
			}
			p.Close()
			return p.End()
		}
		// Erst die Spalte aller Nähte als Schachbrett (durchsichtig, ADR-0023 Nachtrag 7), dann die Schichten — dieselbe
		// Reihenfolge wie Renderer und SVG-Export.
		spaltVon, spaltBis := document.Spaltband()
		var bretter schachbretter
		spalte := func() {
			for _, n := range naehte {
				l, laenge := linie(n)
				band = l.Band(band, n, spaltVon, spaltBis)
				stapel := clip.Outline{Path: pfad(band)}.Op().Push(&c.ops)
				bretter.male(&c.ops, n, laenge, faktor)
				stapel.Pop()
			}
		}
		durchgang := func(auswahl func(document.Schicht) bool) {
			for _, n := range naehte {
				l, _ := linie(n)
				for _, sch := range schichten {
					if auswahl(sch) {
						band = l.Band(band, n, sch.Von, sch.Bis)
						paint.FillShape(&c.ops, sch.Farbe, clip.Outline{Path: pfad(band)}.Op())
					}
				}
			}
		}
		// Die Reihenfolge (auch die zweite Leerung bei sich berührenden Nähten, F-438, F-499) liegt in fuehreNahtAblaufAus.
		nahtAblauf(naehte, faktor, saumImBild, spalte, durchgang)
		c.aufruf = aufnahme.Stop()
		c.groesse, c.naehte, c.form, c.faktor, c.saumImBild, c.gueltig = g, naehte, form, faktor, saumImBild, true
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

// textBei setzt Text in Ansichtspixeln auf die Grundlinie p.Y (wie der Renderer, F-108): mit dem Anfang bei p.X oder,
// mit mitte, waagrecht mittig um p.X.
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
	t := op.Affine(f32.Affine2D{}.Offset(textEcke(p, dims, mitte))).Push(gtx.Ops)
	call.Add(gtx.Ops)
	t.Pop()
}

// textEcke ist die linke obere Ecke des Textfelds, damit seine Grundlinie auf p.Y liegt (mit mitte: seine Mitte auf p.X).
func textEcke(p f32.Point, dims layout.Dimensions, mitte bool) f32.Point {
	// dims.Baseline ist der Abstand von der UNTERkante des Felds zur Grundlinie (Gio); von oben sind es Size.Y − Baseline.
	ecke := p.Sub(f32.Pt(0, float32(dims.Size.Y-dims.Baseline)))
	if mitte {
		ecke.X -= float32(dims.Size.X) / 2
	}
	return ecke
}

// zeichneAuswahl zeichnet Rahmen und Griffe des ausgewählten Objekts in Fensterpixeln (unabhängig vom Zoom), dazu den
// Rahmen jedes weiteren gewählten Objekts ohne Griffe und einen aufgezogenen Auswahlrahmen (ADR-0039 Punkt 5).
func (e *editor) zeichneAuswahl(gtx layout.Context) {
	fe := func(p document.Punkt) image.Point {
		q := e.s.Abb.ZuFenster(e.s.D.ZuAnsicht(p))
		return image.Pt(int(math.Round(q.X)), int(math.Round(q.Y)))
	}
	for _, id := range e.s.Gewaehlt() {
		if id == e.s.Auswahl {
			continue
		}
		if w, ok := e.s.D.Objekt(id); ok {
			for _, g := range auswahlRechtecke(w) {
				r := image.Rectangle{Min: fe(g.Min), Max: fe(g.Max)}.Canon()
				auswahlLinie(gtx, r.Inset(-gtx.Dp(2)))
			}
		}
	}
	if r, ok := e.s.AuswahlRahmen(); ok {
		q := image.Rect(int(math.Round(r.Min.X)), int(math.Round(r.Min.Y)), int(math.Round(r.Max.X)),
			int(math.Round(r.Max.Y))).Canon()
		paint.FillShape(gtx.Ops, farbeAuswahlFlaeche, clip.Rect(q).Op())
		auswahlLinie(gtx, q)
	}
	o, ok := e.s.D.Objekt(e.s.Auswahl)
	if !ok || o.Ebenenwerte().Ausgeblendet { // Ausgeblendetes zeigt keinen Rahmen (ADR-0032 Punkt 4)
		return
	}
	for _, g := range auswahlRechtecke(o) {
		r := image.Rectangle{Min: fe(g.Min), Max: fe(g.Max)}.Canon()
		auswahlLinie(gtx, r.Inset(-gtx.Dp(2)))
	}
	h := gtx.Dp(5)
	if o.Ebenenwerte().Gesperrt {
		return // gesperrt: Rahmen ohne Griffe (Punkt 5)
	}
	for _, gr := range canvas.Griffe(o) {
		m := fe(gr.Ort)
		q := image.Rect(m.X-h, m.Y-h, m.X+h, m.Y+h)
		paint.FillShape(gtx.Ops, farbeText, clip.Rect(q).Op())
		rahmen(gtx.Ops, q, gtx.Dp(1), farbeAuswahl)
	}
}

// auswahlLinie zeichnet einen Auswahlrahmen: die grüne Linie mit einer dunklen Kontur außen, damit er auch auf grünem
// oder hellem Bildgrund zu sehen ist (Sichtprobe #334).
func auswahlLinie(gtx layout.Context, r image.Rectangle) {
	rahmen(gtx.Ops, r.Inset(-gtx.Dp(1)), gtx.Dp(1), farbeAuswahlKontur)
	rahmen(gtx.Ops, r, gtx.Dp(1), farbeAuswahl)
}

// --- Tasten ---

// alleTastenFilter sind die Filter der Tastenschleife: die Tasten der Tabelle (tastenFilter), der Fokus der Leinwand und
// die Navigation im Menü (menueNavigationsFilter, ADR-0026 Nachtrag 4). Mehr meldet der Editor nicht an; der Test
// TestTastenfilterNurTabelleUndMenueNavigation hält das fest.
func (e *editor) alleTastenFilter() []event.Filter {
	var filter []event.Filter
	textLaeuft := e.s.TextLaeuft()
	for _, f := range tastenFilter(textLaeuft) {
		if f.Name == key.NameTab {
			// Tab fängt nur ab, solange die Leinwand den Fokus hat; in einem Eingabefeld der Leiste bleibt es die
			// Fokusfahrt (Karte #220, F-574): Sonst käme man aus Hex und Deckkraft während einer Texteingabe nicht heraus.
			f.Focus = e
		}
		if f.Name == key.NameSpace {
			if textLaeuft {
				continue // in der Texteingabe ist die Leertaste ein Zeichen (EditEvent)
			}
			f.Focus = e // nur mit Fokus auf der Leinwand; Knöpfe behalten ihre Leertaste
		}
		filter = append(filter, f)
	}
	filter = append(filter, key.FocusFilter{Target: e})
	if e.hilfeOffen { // Bild auf/ab rollen die offene Übersicht (F-705); Pfeile, Pos1 und Ende stehen im Menüfilter
		filter = append(filter, key.Filter{Name: key.NamePageUp}, key.Filter{Name: key.NamePageDown})
	}
	for _, f := range menueNavigationsFilter(e.leertastenZiele()...) {
		filter = append(filter, f)
	}
	for _, f := range e.unten.markierTasten() { // Markierung an einer Vorschau der Leiste (Karte #361)
		filter = append(filter, f)
	}
	return filter
}

// fensterFokus folgt dem Fokus des Fensters: Verliert es ihn, käme das Loslassen der Leertaste nicht mehr an, sie gilt
// als losgelassen (ADR-0039 Nachtrag 1).
func (e *editor) fensterFokus(an bool) {
	if !an {
		e.s.SetzeLeertaste(false)
	}
}

// leertasteGilt meldet, ob die gehaltene Leertaste die Ansicht verschieben darf: Fokus auf der Leinwand, keine
// Texteingabe, kein offenes Menü, keine Tafel und keine volle Ansicht der Bibliothek darüber (ADR-0039 Nachtrag 1).
func (e *editor) leertasteGilt(gtx layout.Context) bool {
	return gtx.Focused(e) && !e.s.TextLaeuft() && !e.menue.offen && !e.hilfeOffen && !e.einst.offen && !e.bibl.offen
}

// leertastenZiele sind die Knöpfe, an denen die Leertaste wie Enter ein Menü oder eine Liste mit Fokus auf dem ersten
// Eintrag öffnet: die Menütitel und die Felder mit Liste (Fußfelder Farbe und Zoom, Farbfeld links; Sitzungs-QA A-6).
func (e *editor) leertastenZiele() []event.Tag {
	var z []event.Tag
	for i := range e.menue.titel {
		z = append(z, &e.menue.titel[i])
	}
	z = append(z, e.schnell.ziele()...)
	z = append(z, e.unten.ziele()...)
	z = append(z, &e.pfeilform.feld[0], &e.pfeilform.feld[1], &e.strichart.feld)
	return append(z, &e.knoepfe.fussFarbe, &e.knoepfe.fussZoom, &e.knoepfe.farbe, &e.knoepfe.fuellung)
}

func (e *editor) tasten(gtx layout.Context) {
	e.menue.gewechselt = false
	if e.fokusHolen && !e.menue.offen && !e.hilfeOffen && !e.einst.offen && !e.bibl.offen {
		// Ohne Fokus auf der Leinwand kämen weder Leertaste+Ziehen noch Kürzel an, bis man ins Bild klickt (F-907).
		e.fokusHolen = false
		gtx.Execute(key.FocusCmd{Tag: e})
	}
	if e.s.Leertaste() && !e.leertasteGilt(gtx) {
		e.s.SetzeLeertaste(false) // Fokus weg, Texteingabe, Menü oder Tafel: das Loslassen käme nicht an
	}
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
		if k, ok := ev.(key.Event); ok && k.Name == key.NameAlt {
			// Alt gedrückt oder losgelassen, ohne dass sich der Zeiger bewegt (ADR-0021 Nachtrag 12 Punkt 4): Ein
			// laufender Zug rastet sofort nicht mehr bzw. wieder ein. Ohne Zug merkt die Sitzung nur den Zustand; das
			// nächste Zeigerereignis setzt ihn ohnehin neu. Während des Speicherns ruht der Zug (Sitzung.Ziehen).
			e.s.AltTaste(k.State == key.Press)
			gtx.Execute(op.InvalidateCmd{})
			continue
		}
		if k, ok := ev.(key.Event); ok && k.Name == key.NameSpace && gtx.Focused(e) && !e.s.TextLaeuft() {
			// Leertaste an der Leinwand (ADR-0039 Nachtrag 1): solange gehalten, verschiebt Ziehen die Ansicht. Andere
			// Ziele der Leertaste (Menütitel, Felder mit Liste) haben eigene Filter mit eigenem Fokus.
			// Loslassen gilt immer, Drücken nur ohne Strg, Alt und Super (die sind Kürzel, nicht Verschieben; F-905).
			e.s.SetzeLeertaste(k.State == key.Press && e.leertasteGilt(gtx) &&
				!k.Modifiers.Contain(key.ModCtrl) && !k.Modifiers.Contain(key.ModAlt) && !k.Modifiers.Contain(key.ModSuper))
			gtx.Execute(op.InvalidateCmd{})
			continue
		}
		if e.hilfeOffen && !e.menue.offen {
			// Übersicht der Tastenkürzel offen (F1): Esc und F1 schließen sie, auch während des Speicherns; alle anderen
			// Tasten und getippter Text wirken nicht, damit nichts unbemerkt hinter der Tafel die Leinwand ändert (wie
			// beim offenen Menü, F-257; Sitzungs-QA B-2).
			if k, ok := ev.(key.Event); ok && k.State == key.Press {
				r := &e.hilfeRoll
				if k.Name == key.NameEscape || k.Name == key.NameF1 {
					e.hilfeOffen = false
				} else if n, rolle := rollTaste(k.Name, r.rollen, r.rollMax, gtx.Dp(hilfeZeileDp), r.sicht.Dy()); rolle &&
					!k.Modifiers.Contain(key.ModShortcut) && !k.Modifiers.Contain(key.ModAlt) {
					r.rollen = n // F-705: Pfeile, Bild auf/ab, Pos1, Ende rollen die Liste
					gtx.Execute(op.InvalidateCmd{})
				}
			}
			continue
		}
		if e.einst.offen && !e.menue.offen {
			continue // Einstellungstafel: ihre Tasten holt einstellungenAuswerten vorher ab, die übrigen wirken nicht (ADR-0034)
		}
		if e.bibl.offen && !e.menue.offen && !e.s.SpeichernLaeuft() {
			// Volle Ansicht der Bibliothek (ADR-0036 Punkt 4): Ihre Tasten holt bibliothekAuswerten vorher ab. Von den
			// übrigen wirken nur F10 und die Kürzel der Menüeinträge, die in der Ansicht bedienbar sind; Werkzeugtasten,
			// Leinwand-Kürzel und getippter Text gehen nicht an das Bild darunter.
			if k, ok := ev.(key.Event); ok && k.State == key.Press {
				t, ok := findeTaste(k.Name, k.Modifiers, false)
				switch {
				case k.Name == key.NameF10 && k.Modifiers.Contain(key.ModShift):
					// Kontextmenü der Vorschau mit Fokus (ADR-0038 Punkt 6); ohne Vorschau mit Fokus nichts, nie das Menü
					// Datei (Kritik K-1 #313a).
					e.vorschauKontextPerTaste(gtx)
				case k.Name == key.NameF10:
					e.menueOeffne(menueDatei, true)
				case ok && !t.hatWerkzeug && bibliothekErlaubt(t.akt):
					e.fuehreAus(gtx, t.akt, false)
				}
			}
			continue
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
				} else if k.Name == key.NameEscape && k.Modifiers == 0 && e.leisteMarkierungEsc() {
					// Esc hebt die Markierung der Leiste auch während des Speicherns auf, das einem Papierkorb-Wechsel
					// vorausgeht: Sonst nähme der Wechsel die übrigen markierten trotz Esc mit (Verify N-1 zu F-887).
				} else if t, ok := findeTaste(k.Name, k.Modifiers, e.s.TextLaeuft()); ok && !t.hatWerkzeug &&
					((t.akt == aktKopieren && !e.bibl.offen) || t.akt == aktSchliessen) { // Strg+C nicht an der Ansicht vorbei (F-678 Punkt 3)
					e.fuehreAus(gtx, t.akt, false)
				}
			}
			continue
		}
		if e.seite.sw.offen && !e.menue.offen {
			// Offene Schriftliste (ADR-0031 Nachtrag 6): Pfeile, Pos1, Ende, Enter und Esc bedienen sie, getippte
			// Buchstaben springen zur Familie; alles andere wirkt nicht, damit kein Buchstabe als Werkzeugkürzel hinter
			// der Liste das Werkzeug wechselt.
			switch ev := ev.(type) {
			case key.EditEvent:
				e.schriftListeTippe(ev.Text)
			case key.Event:
				if ev.State == key.Press {
					e.schriftListeTaste(gtx, ev.Name)
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
		if k.Name == key.NameSpace && k.Modifiers == 0 { // nur an den Menütiteln und den Feldern mit Liste angemeldet; wirkt wie Enter
			if e.menue.offen && !e.menueKnopf(e.fokussierterKnopf(gtx)) && e.menueTitelFokus(gtx) < 0 {
				continue // hinter einer offenen Liste wirkt die Leertaste nicht (F-873, F-257)
			}
			if i := e.menueTitelFokus(gtx); i >= 0 {
				e.menueOeffne(i, true)
			} else if c := e.fokussierterKnopf(gtx); c == &e.knoepfe.fussFarbe || c == &e.knoepfe.fussZoom ||
				c == &e.knoepfe.farbe || c == &e.knoepfe.fuellung {
				e.knopfPerTaste(c)
			} else if c != nil && (e.schnell.stelle(c) >= 0 || e.unten.stelle(c) >= 0) { // Schnellstile, Leiste unten
				e.knopfPerTaste(c)
			} else if c != nil && c == &e.unten.bibKnopf { // Knopf „Library“ der Leiste (ADR-0036 Punkt 1)
				e.knopfPerTaste(c)
			}
			continue
		}
		if k.Name == key.NameF10 && k.Modifiers.Contain(key.ModShift) && !e.menue.offen && !e.bibl.offen {
			// Kontextmenü der Vorschau mit Fokus in der Leiste, sonst der Auswahl (ADR-0038 Punkt 6)
			if !e.vorschauKontextPerTaste(gtx) {
				e.kontextPerTaste()
			}
			continue
		}
		if k.Name == key.NameF10 { // öffnet das erste Menü bzw. schließt das offene (ADR-0031 Punkt 1)
			if e.menue.offen {
				e.menueSchliesse(gtx, false)
			} else {
				e.menueOeffne(menueDatei, true)
			}
			continue
		}
		if e.menue.offen {
			// Offenes Menü (ADR-0026 Punkt 1, Nachträge 2 und 3): Esc schließt es, Enter löst einen Knopf oder Eintrag
			// mit Tastaturfokus aus, Pfeile hoch und runter, Pos1 und Ende wählen den Eintrag; die Leertaste am
			// Menütitel ist oben behandelt, Tab fährt den Fokus wie sonst (Gio). Alle anderen Tasten wirken nicht,
			// damit Entf oder ein Buchstabe nicht unbemerkt hinter der Liste die Leinwand ändert (F-257).
			switch k.Name {
			case key.NameEscape:
				// Fokus zurück an den Menütitel nur, wenn per Tastatur geöffnet wurde und kein Text läuft; sonst an die
				// Leinwand, sonst gingen getippte Zeichen verloren (F-255 V-1).
				e.menueSchliesse(gtx, e.menue.perTasteOffen && !e.s.TextLaeuft())
			case key.NameUpArrow, key.NameDownArrow, key.NameHome, key.NameEnd:
				e.menueTaste(gtx, k.Name)
			case key.NameLeftArrow:
				e.menueWechsel(-1)
			case key.NameRightArrow:
				e.menueWechsel(1)
			case key.NameReturn, key.NameEnter:
				// Nach einem Menüwechsel im selben Bild liegt der Fokus noch im alten Menü (F-512).
				// Nur Einträge, Menütitel und das Feld der Liste: Ein Knopf hinter der Liste (etwa die Vorschau, auf die der
				// Rechtsklick den Fokus gab) wirkt nicht (F-873, F-257).
				if c := e.fokussierterKnopf(gtx); c != nil && !e.menue.gewechselt && e.menueKnopf(c) {
					e.knopfPerTaste(c)
				}
			}
			continue
		}
		if !e.s.TextLaeuft() && e.schnellTaste(gtx, k) {
			continue // Pfeile und Entf an einer Kachel der Schnellstile (ADR-0037 Punkt 7)
		}
		if !e.s.TextLaeuft() && e.scheibeTaste(gtx, k) {
			continue // Pfeile an der Richtungsscheibe des Schattens (ADR-0031 Nachtrag 13)
		}
		if !e.s.TextLaeuft() && e.farbfeldTaste(gtx, k) {
			continue // Entf an einem Farbfeld (Karte #364)
		}
		if !e.s.TextLaeuft() && e.leisteTaste(gtx, k) {
			continue // Pfeile, Pos1 und Ende an einer Vorschau der Leiste (ADR-0036 Punkt 1)
		}
		t, ok := findeTaste(k.Name, k.Modifiers, e.s.TextLaeuft())
		if e.fokusFeld(gtx) != nil && ((ok && (t.hatWerkzeug || zwischenablageAktion(t.akt))) || k.Name == key.NameDeleteForward) {
			// Im Eingabefeld der Leiste (Hex, Deckkraft) sind Buchstaben und Ziffern Eingabe (kommt als EditEvent), keine
			// Kürzel; Entf wirkt dort nicht (ADR-0031 Nachtrag 5). Enter, Esc und die Rücktaste holt feldTasten vorher ab.
			// Strg+C/X/V gelten dort dem Feld, nie den Objekten der Leinwand (F-986).
			continue
		}
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
			if e.einfuegenEsc() { // zuerst ein laufendes Lesen der Zwischenablage (F-987)
				continue
			}
			if e.ocrEsc() { // zuerst eine laufende Texterkennung (ADR-0035 Punkt 5)
				continue
			}
			if e.leisteMarkierungEsc() { // dann die Markierung der Leiste, auch ohne Fokus an einer Vorschau (F-890)
				continue
			}
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
	alle := []*widget.Clickable{&k.farbe, &k.fuellung, &k.tausch, &k.fussFarbe, &k.fussZoom}
	alle = append(alle, e.seite.alle()...)
	alle = append(alle, e.ebenen.alleKnoepfe()...)
	alle = append(alle, e.lupe.alle()...)
	alle = append(alle, &e.spitze.knopf)
	alle = append(alle, e.schattenL.alle()...)
	alle = append(alle, e.strichart.alle()...)
	alle = append(alle, e.pfeilform.alle()...)
	alle = append(alle, e.schnell.alle()...)
	alle = append(alle, e.unten.alle()...) // Vorschauen der Leiste (ADR-0036)
	for i := range e.menue.titel {
		alle = append(alle, &e.menue.titel[i])
	}
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
	for _, w := range e.werkzeugKnoepfe() {
		if w.c.Clicked(gtx) && an() {
			e.s.SetzeWerkzeug(w.w)
		}
	}
	// Farbpaar (ADR-0021 Nachtrag 11 Punkt 3): Ein Klick auf ein Feld macht es aktiv und zeigt die Farbe wie bisher;
	// das graue Feld Füllung (Pfeil, Nummer) tut nichts.
	if k.farbe.Clicked(gtx) && an() {
		e.s.SetzeFeld(canvas.FeldVordergrund)
		e.farbKnopfGeklickt(gtx)
	}
	if k.fuellung.Clicked(gtx) && an() && e.s.SetzeFeld(canvas.FeldFuellung) {
		e.farbKnopfGeklickt(gtx)
	}
	if k.tausch.Clicked(gtx) && an() {
		e.s.TauscheFarben()
	}
}

// farbKnopfGeklickt: Das Farbfeld der Werkzeugleiste zeigt die Farbe in der rechten Leiste (ADR-0031 Punkt 2). Kann die
// Leiste die Farbe zeigen, klappt es Leiste und Abschnitt „Farbe“ auf, gibt der Farbe den Vorrang vor den Ebenen und
// rollt den Farbwähler ins Bild — auch wenn die Leiste schon offen ist (ADR-0031 Nachtrag 12, F-817: vorher öffnete
// der Klick dann nur die kleine Farbliste, und ein eingeklappter Abschnitt blieb zu). Nur wenn die Leiste die Farbe
// nicht zeigen kann (Fenster zu schmal, Werkzeug ohne Farbe), öffnet es die Farbliste am Farbfeld.
func (e *editor) farbKnopfGeklickt(gtx layout.Context) {
	if seitePasst(gtx) && e.s.Seitenleiste().Farbe {
		e.seiteZu = false
		e.seite.farbeZu = false
		e.ebenen.vorrang = false
		e.seite.farbeZeigen = true
		gtx.Execute(op.InvalidateCmd{})
		return
	}
	e.listeAmFarbKnopf = true
	e.menueOeffne(listeFarbe, false)
}

type werkzeugKnopf struct {
	c    *widget.Clickable
	w    canvas.Werkzeug
	text string
}

// zeichneEllipse zeichnet einen Kasten mit Form Ellipse (ADR-0047 Punkt 2) aus denselben Teilen wie der Renderer: die
// Füllung im abgetasteten Umriss in jedem Bereich der Ellipse, der Strich (render.ZerlegeEllipse) nur in den Bereichen,
// die der Umriss bzw. seine Stücke berühren — wie der runde Kasten (ADR-0023 Nachträge 17 bis 21).
func (e *editor) zeichneEllipse(gtx layout.Context, v document.Kasten, an func(document.Punkt) f32.Point, vorschau bool) {
	b := v.Bereich
	o := e.s.D.OriginalBild().Rect
	bild := document.R(document.Pt(float64(o.Min.X), float64(o.Min.Y)), document.Pt(float64(o.Max.X), float64(o.Max.Y)))
	if !vorschau && !document.EllipseSichtbar(v, bild) {
		return // berührt das Bild nicht: nicht abtasten (F-1023, wie der Renderer)
	}
	if v.Fuellung.A > 0 {
		umriss := [][]document.Punkt{document.EllipsenUmriss(b)}
		e.jeStueck(gtx, b, 1, an, vorschau, func(abb func(document.Punkt) f32.Point) {
			fuelleTeile(gtx, umriss, v.Fuellung, abb)
		})
	}
	if v.Strich.Breite > 0 && v.Strich.Farbe.A > 0 {
		sz := render.ZerlegeEllipse(b, v.Strich)
		var stuecke []document.Stueck
		if !vorschau {
			for _, st := range e.s.D.StreckenStuecke(sz.Zug, sz.Zu, sz.Suche) {
				stuecke = append(stuecke, st.Stueck)
			}
		}
		e.jeStueckIn(gtx, stuecke, an, vorschau, func(abb func(document.Punkt) f32.Point) {
			fuelleTeile(gtx, sz.Teile, v.Strich.Farbe, abb)
		})
	}
}

// werkzeugKnoepfe sind die Knöpfe der Werkzeuggruppe in Reihenfolge der Leiste (Kürzel V, A, H, R, E, T, K, N, M, B, P,
// C, X; ADR-0021, ADR-0022, ADR-0023, ADR-0033, ADR-0047, ADR-0048).
func (e *editor) werkzeugKnoepfe() []werkzeugKnopf {
	k := &e.knoepfe
	return []werkzeugKnopf{
		{&k.auswahl, canvas.WerkzeugAuswahl, i18n.T("Select")},
		{&k.pfeil, canvas.WerkzeugPfeil, i18n.T("Arrow")},
		{&k.textmarker, canvas.WerkzeugTextmarker, i18n.T("Highlighter")},
		{&k.kasten, canvas.WerkzeugKasten, i18n.T("Rectangle")},
		{&k.ellipse, canvas.WerkzeugEllipse, i18n.T("Ellipse")},
		{&k.text, canvas.WerkzeugText, i18n.T("Text")},
		{&k.sprechblase, canvas.WerkzeugSprechblase, i18n.T("Speech bubble")},
		{&k.nummer, canvas.WerkzeugNummer, i18n.T("Step")},
		{&k.lupe, canvas.WerkzeugLupe, i18n.T("Magnifier")},
		{&k.schwaerzen, canvas.WerkzeugSchwaerzen, i18n.T("Redact")},
		{&k.verpixeln, canvas.WerkzeugVerpixeln, i18n.T("Pixelate")},
		{&k.zuschneiden, canvas.WerkzeugZuschneiden, i18n.T("Crop")},
		{&k.stauchen, canvas.WerkzeugStauchen, i18n.T("Cut out")},
	}
}

// hinweis ist die Bedienung des aktiven Werkzeugs bzw. der Texteingabe für die Fußzeile (ADR-0021).
func (e *editor) hinweis() string {
	if e.bibl.offen { // volle Ansicht der Bibliothek (ADR-0036)
		return e.bibl.hinweis
	}
	if e.exportOffen {
		return i18n.T("File dialog is open.") // der Dialog kann hinter dem Fenster liegen (A-12)
	}
	if t := e.ocrHinweis(); t != "" { // Bereichsmodus und Erkennung (ADR-0035)
		return t
	}
	if e.schnell.hinweis != "" { // Name der Kachel mit Fokus oder Zeiger (ADR-0037 Punkt 2)
		return e.schnell.hinweis
	}
	if e.seite.ff.hinweis != "" { // Farbfeld oder „+“ mit Fokus oder Zeiger (Karte #364)
		return e.seite.ff.hinweis
	}
	if e.pfeilform.hinweis != "" { // Feld der Spitze am Anfang oder Ende (ADR-0021 Nachtrag 20)
		return e.pfeilform.hinweis
	}
	if e.strichart.hinweis != "" { // Feld der Strichart (ADR-0021 Nachtrag 25)
		return e.strichart.hinweis
	}
	if e.unten.hinweis != "" { // Vorschau der Leiste mit Fokus oder Zeiger (ADR-0036)
		return e.unten.hinweis
	}
	if e.s.PipetteLaeuft() {
		return i18n.T("Click in the image to pick its color. Esc cancels.")
	}
	if e.s.TextLaeuft() {
		return i18n.T("Type the text. Enter: done, Shift+Enter: new line, Esc: discard.")
	}
	if n := e.s.AnzahlGewaehlt(); n > 1 { // Mehrfachauswahl (ADR-0039 Punkt 5)
		return fmt.Sprintf(i18n.N("%d object selected. Drag one to move all; Layer menu: align, distribute.",
			"%d objects selected. Drag one to move all; Layer menu: align, distribute.", n), n)
	}
	switch e.s.Werkzeug {
	case canvas.WerkzeugAuswahl:
		return i18n.T("Click to select, Shift+click to add. Drag in the image to select several. Space+drag moves the view.")
	case canvas.WerkzeugPfeil:
		return i18n.T("Drag to draw an arrow. Shift: 45° steps. Drag the middle handle to bend it.")
	case canvas.WerkzeugKasten:
		return i18n.T("Drag to draw a rectangle. Shift: square.")
	case canvas.WerkzeugEllipse:
		return i18n.T("Drag to draw an ellipse. Shift: circle.")
	case canvas.WerkzeugTextmarker:
		return i18n.T("Drag along the text to highlight it. Shift: horizontal.")
	case canvas.WerkzeugText:
		return i18n.T("Click to place text, or click a text to edit it.")
	case canvas.WerkzeugSprechblase:
		return i18n.T("Drag from the spot to point at; let go where the text goes. Or click to place a bubble, or click a text to edit it.")
	case canvas.WerkzeugNummer:
		if a := e.s.Voll(); a != canvas.AblehnungKeine {
			return e.grenzText(a) // keine Nummer versprechen, die nicht kommt (F-154)
		}
		return fmt.Sprintf(i18n.T("Click to place step number %d."), e.s.D.NaechsteNummer())
	case canvas.WerkzeugLupe:
		return i18n.T("Drag over the part to magnify. The enlarged copy appears next to it; drag either frame to move it.")
	case canvas.WerkzeugSchwaerzen:
		return i18n.T("Drag over the area to black out. The pixels are overwritten for good once saved.")
	case canvas.WerkzeugVerpixeln:
		return i18n.T("Drag over the area to pixelate. The pixels are overwritten for good once saved.")
	case canvas.WerkzeugZuschneiden:
		// Zuschneiden ist kein Schwärzen: Das Weggeschnittene bleibt im Original der Arbeitsdatei (ADR-0023 Punkt 8).
		return i18n.T("Drag the edges or corners inwards, or drag a new frame. Double-click: whole image again. Cropped parts stay in the working file; redact what must go.")
	case canvas.WerkzeugStauchen:
		// Wie beim Zuschneiden bleibt das Herausgenommene im Original der Arbeitsdatei (ADR-0023 Punkt 8, F-185).
		return i18n.T("Drag across the part to cut out: vertically for rows, horizontally for columns. Double-click a seam to restore it. Cut-out parts stay in the working file; redact what must go.")
	}
	return ""
}
