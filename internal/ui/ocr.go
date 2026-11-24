// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Text aus dem Bild kopieren (ADR-0035, Karte #195). Beim Öffnen des Editors sucht eine Goroutine einmal tesseract
// (ocr.Suche); davon hängt ab, ob der Menüeintrag Bearbeiten → „Copy text from image“ (Strg+Umschalt+T) bedienbar ist.
// Der Eintrag schaltet den einmaligen Bereichsmodus der Sitzung ein (canvas/textbereich.go); den gewählten Bereich
// holt ocrBereichPruefen ab und startet die Erkennung auf einem Abzug in einer Goroutine. Das Ergebnis kommt in der
// Ereignisschleife an (ocrAbholen) und geht als Text in die Zwischenablage, auf demselben Weg wie das Bild bei Strg+C
// (kopierAntwort: Halter im Textmodus, sonst bietet das Fenster an; ADR-0024 Nachtrag 9).

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"time"
	"unicode/utf8"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/ocr"
	"github.com/Andiministrator/schnapp/internal/render"
	"github.com/Andiministrator/schnapp/internal/wlclip"
)

// textErkenner ist, was die Oberfläche vom gefundenen tesseract braucht (*ocr.Erkenner). Eigene Schnittstelle, damit
// Tests eine Attrappe einsetzen, ohne ein Programm zu starten.
type textErkenner interface {
	Sprachen(oberflaeche string) string
	Erkenne(ctx context.Context, b image.Image, sprachen string) (string, error)
}

// sucheErkenner sucht tesseract (ocr.Suche); Naht für Tests.
var sucheErkenner = func(ctx context.Context) (textErkenner, error) {
	e, err := ocr.Suche(ctx)
	if err != nil {
		return nil, err // kein *ocr.Erkenner(nil) in der Schnittstelle
	}
	return e, nil
}

// ocrSuchErgebnis ist die Antwort der Suche, ocrErgebnis die einer Erkennung.
type ocrSuchErgebnis struct {
	erkenner textErkenner
	err      error
}

type ocrErgebnis struct {
	text string
	err  error
}

// ocrZustand führt Suche und Erkennung; nur die Ereignisschleife liest und schreibt ihn.
type ocrZustand struct {
	suche    chan ocrSuchErgebnis // Antwort der Suche; nil: nicht gestartet
	gesucht  bool                 // die Antwort der Suche ist angenommen
	erkenner textErkenner         // nil: kein tesseract (sucheErr sagt, warum)
	sucheErr error
	laeuft   bool               // eine Erkennung läuft
	abbruch  context.CancelFunc // beendet die laufende Erkennung (Esc, Schließen)
	antwort  chan ocrErgebnis   // je Erkennung neu: Eine abgebrochene liefert in einen verlassenen Kanal
}

// ocrKontext ist der Kontext des Editors (endet mit dem Fenster, Editor); ohne Fenster (Tests) der Hintergrund.
func (e *editor) ocrKontext() context.Context {
	if e.exportCtx != nil {
		return e.exportCtx
	}
	return context.Background()
}

// ocrSucheStarten sucht tesseract einmal im Hintergrund (ADR-0035 Punkt 1).
func (e *editor) ocrSucheStarten() {
	if e.ocr.suche != nil {
		return
	}
	ch := make(chan ocrSuchErgebnis, 1)
	e.ocr.suche = ch
	ctx, wecker := e.ocrKontext(), e.wecker
	go func() {
		er, err := sucheErkenner(ctx)
		ch <- ocrSuchErgebnis{er, err}
		if wecker != nil {
			wecker.Anfordern()
		}
	}()
}

// ocrGrau ist die Erklärung, warum „Copy text from image“ gerade nicht geht (leer: es geht). Sie steht in der
// Statuszeile, solange der graue Menüeintrag überfahren oder per Tastatur angesteuert ist, und nach dem Kürzel.
func (e *editor) ocrGrau() string {
	switch {
	case !e.ocr.gesucht:
		return i18n.T("Looking for tesseract…")
	case e.ocr.erkenner == nil && errors.Is(e.ocr.sucheErr, ocr.ErrKeineSprache):
		return i18n.T("Install language data for tesseract to recognize text.")
	case e.ocr.erkenner == nil && (e.ocr.sucheErr == nil || errors.Is(e.ocr.sucheErr, ocr.ErrKeinProgramm)):
		return i18n.T("Install tesseract to recognize text.")
	case e.ocr.erkenner == nil:
		return fmt.Sprintf(i18n.T("Text recognition is not available: %v"), ohnePaketpraefix(e.ocr.sucheErr.Error()))
	case e.ocr.laeuft:
		return i18n.T("Text recognition is already running.")
	}
	return ""
}

// ocrBeginnen ist die Handlung des Menüeintrags und des Kürzels: Bereichsmodus ein, oder die Erklärung, warum nicht.
func (e *editor) ocrBeginnen() {
	if t := e.ocrGrau(); t != "" {
		e.melde(t, false)
		return
	}
	if e.s.BeginneTextBereich() && !e.haltendeMeldung() {
		// Eine frühere Meldung („Copied …“, „No text found“) verdeckte sonst die Anleitung des Modus; eine haltende
		// (was das nächste Schließen tut) bleibt stehen.
		e.melde("", false)
	}
}

// ocrHinweis ist der Bedienhinweis der Statuszeile während des Bereichsmodus und der Erkennung (leer: keiner).
func (e *editor) ocrHinweis() string {
	switch {
	case e.s.TextBereichLaeuft():
		return i18n.T("Drag over the text to copy; Enter takes the whole image; Esc cancels.")
	case e.ocr.laeuft:
		return i18n.T("Recognizing text…")
	}
	return ""
}

// ocrBereichPruefen startet die Erkennung, sobald der Bereichsmodus einen Bereich gewählt hat (Loslassen oder Enter).
func (e *editor) ocrBereichPruefen() {
	if r, ok := e.s.NimmTextBereich(); ok {
		e.ocrStarten(r)
	}
}

// ocrStarten erkennt den Text im Bereich (Pixel des sichtbaren Bilds) in einer Goroutine auf einem Abzug (ADR-0035
// Punkte 4 und 5); das Bearbeiten geht weiter.
func (e *editor) ocrStarten(bereich image.Rectangle) {
	if e.ocr.erkenner == nil || e.ocr.laeuft {
		return
	}
	abzug, err := e.s.D.Abzug()
	if err != nil {
		e.melde(i18n.T("Could not recognize text: finish the current action first."), true)
		return
	}
	sprachen := e.ocr.erkenner.Sprachen(i18n.Sprache())
	ctx, abbruch := context.WithCancel(e.ocrKontext())
	ch := make(chan ocrErgebnis, 1)
	e.ocr.laeuft, e.ocr.abbruch, e.ocr.antwort = true, abbruch, ch
	er, wecker := e.ocr.erkenner, e.wecker
	diagnoseEditor("Texterkennung: Bereich %v, Sprachen %s", bereich, sprachen)
	e.melde(i18n.T("Recognizing text…"), false)
	go func() {
		var r ocrErgebnis
		b, err := ocrBild(abzug, bereich)
		if err == nil {
			r.text, r.err = er.Erkenne(ctx, b, sprachen)
		} else {
			r.err = err
		}
		ch <- r
		if wecker != nil {
			wecker.Anfordern()
		}
	}()
}

// ocrBild ist das sichtbare Bild wie beim Export (render.Bild: Ansicht mit Zuschnitt, Stauchung, Schwärzungen und
// Beschriftungen; nie das Original, Kernprinzip 2), ausgeschnitten auf bereich und auf Weiß gelegt: Der Spalt einer
// Stauch-Naht ist durchsichtig, tesseract sähe ihn sonst je nach Leser schwarz.
func ocrBild(d *document.Dokument, bereich image.Rectangle) (image.Image, error) {
	b, err := render.Bild(d)
	if err != nil {
		return nil, err
	}
	bereich = bereich.Intersect(b.Bounds())
	weiss := image.NewNRGBA(image.Rectangle{Max: bereich.Size()})
	draw.Draw(weiss, weiss.Bounds(), image.White, image.Point{}, draw.Src)
	draw.Draw(weiss, weiss.Bounds(), b, bereich.Min, draw.Over)
	return weiss, nil
}

// ocrAbholen nimmt die Antworten von Suche und Erkennung an (zu Beginn eines Bilds, nach kopierenAbholen). Läuft noch
// ein Kopieren, bleibt die Antwort der Erkennung liegen, bis es fertig ist: Beide gehen über kopierAntwort.
func (e *editor) ocrAbholen() {
	if e.ocr.suche != nil && !e.ocr.gesucht {
		select {
		case r := <-e.ocr.suche:
			e.ocr.gesucht, e.ocr.erkenner, e.ocr.sucheErr = true, r.erkenner, r.err
			diagnoseEditor("Texterkennung: tesseract gefunden %v (%v)", r.erkenner != nil, r.err)
		default:
		}
	}
	if !e.ocr.laeuft || e.kopierLaeuft {
		return
	}
	select {
	case r := <-e.ocr.antwort:
		e.ocrAbbrechen() // gibt den Kontext frei; die Erkennung ist schon zu Ende
		e.ocrFertig(r)
	default:
	}
}

// ocrFertig zeigt das Ergebnis einer Erkennung: Text in die Zwischenablage, sonst „No text found“ (die Zwischenablage
// bleibt dann, wie sie ist) oder der Fehler (ADR-0035 Punkt 5).
func (e *editor) ocrFertig(r ocrErgebnis) {
	switch {
	case errors.Is(r.err, context.Canceled):
		// abgebrochen (Esc, Schließen): gemeldet hat das der Abbruch selbst
	case errors.Is(r.err, ocr.ErrZuGross):
		e.meldeNebenbei(i18n.T("The area is too large for text recognition."), true)
	case errors.Is(r.err, ocr.ErrFrist):
		e.meldeNebenbei(i18n.T("Text recognition took too long."), true)
	case errors.Is(r.err, ocr.ErrLeer), r.err == nil && r.text == "":
		e.meldeNebenbei(i18n.T("No text found"), false)
	case r.err != nil:
		e.meldeNebenbei(fmt.Sprintf(i18n.T("Could not recognize text: %v"), ohnePaketpraefix(r.err.Error())), true)
	default:
		e.textKopieren(r.text)
	}
}

// ocrEsc bricht mit Esc eine laufende Erkennung ab, wenn Esc sonst nichts zu tun hätte, was Vorrang hat (Bereichsmodus,
// Zug, Texteingabe); true: Esc ist verbraucht.
func (e *editor) ocrEsc() bool {
	if !e.ocr.laeuft || e.s.TextBereichLaeuft() || e.s.Zieht() || e.s.TextLaeuft() {
		return false
	}
	e.ocrAbbrechen()
	e.melde(i18n.T("Text recognition cancelled."), false)
	return true
}

// ocrAbbrechen beendet eine laufende Erkennung (der Kontext beendet das eigene tesseract) und vergisst ihre Antwort.
func (e *editor) ocrAbbrechen() {
	if e.ocr.abbruch != nil {
		e.ocr.abbruch()
	}
	e.ocr.laeuft, e.ocr.abbruch, e.ocr.antwort = false, nil, nil
}

// textMime ist der Typ, unter dem Gio Text anbietet (die üblichen Text-Typen, ADR-0015).
const textMime = "application/text"

// textKopieren legt den erkannten Text in die Zwischenablage wie kopieren das Bild (ADR-0024 Nachtrag 9): Ein Halter im
// Textmodus übernimmt ihn, damit er das Schließen überdauert; meldet sich keiner, bietet das Fenster selbst an
// (kopierenAbholen). Schließen und Zerstören warten darauf wie auf ein Kopieren (kopierLaeuft).
func (e *editor) textKopieren(text string) {
	n := utf8.RuneCountInString(text)
	meldung := fmt.Sprintf(i18n.N("Copied %d character", "Copied %d characters", n), n)
	e.kopierLaeuft, e.kopierStart, e.kopierStand = true, time.Now(), e.s.D.Stand()
	if e.kopierAntwort == nil {
		e.kopierAntwort = make(chan kopierErgebnis, 1)
	}
	ch, wecker := e.kopierAntwort, e.wecker
	go func() {
		r := kopierErgebnis{typ: textMime, meldung: meldung}
		if err := halteZwischenablage([]byte(text), wlclip.ArtText); err != nil {
			diagnoseEditor("Halter für Text nicht gestartet, Fenster bietet selbst an: %v", err)
			r.daten = []byte(text)
		}
		ch <- r
		if wecker != nil {
			wecker.Anfordern()
		}
	}()
}

// zeichneTextBereich zeigt den gerade gezogenen Bereich des Bereichsmodus in Fensterpixeln: hell getönt mit hellem und
// dunklem Rand, damit der Text darunter lesbar bleibt (anders als der dunkle Rahmen beim Schwärzen).
func (e *editor) zeichneTextBereich(gtx layout.Context) {
	a, b, ok := e.s.TextBereichRahmen()
	if !ok {
		return
	}
	rund := func(p document.Punkt) image.Point { return image.Pt(int(math.Round(p.X)), int(math.Round(p.Y))) }
	r := image.Rectangle{Min: rund(a), Max: rund(b)}.Canon()
	paint.FillShape(gtx.Ops, color.NRGBA{R: 0xF5, G: 0xA6, B: 0x5B, A: 0x33}, clip.Rect(r).Op()) // orange-500, leicht
	w := max(1, gtx.Dp(1))
	rahmen(gtx.Ops, r, w, color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xDD})
	rahmen(gtx.Ops, r.Inset(w), w, color.NRGBA{A: 0x99})
}
