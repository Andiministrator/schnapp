// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Farbwähler im Abschnitt „Farbe“ der rechten Leiste (ADR-0031 Nachtrag 5, Bedeutung ADR-0021 Nachtrag 9): Farbfläche,
// Raster der zuletzt benutzten Farben (darunter die eigenen Farbfelder, farbfelder.go, ADR-0021 Nachtrag 24; die Pipette steht in der Zeile des Farbpaars), Farbton- und
// Deckkraft-Regler, Eingaben für Hex und Deckkraft. Was eine Lage oder Eingabe bewirkt, entscheidet canvas.Sitzung; hier stehen Lage, Bilder und Ereignisse.

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"slices"
	"strconv"
	"unicode"
	"unicode/utf8"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/icons"
)

// farbPlaetze ist die Zahl der Plätze im Raster rechts der Farbfläche: die zuletzt benutzten Farben (Karte #364).
const farbPlaetze = 8

// farbRegler ist das Bedienelement, in dem gerade gezogen wird.
type farbRegler int

const (
	reglerKeiner farbRegler = iota
	reglerFlaeche
	reglerTon
	reglerDeck
)

// eingabe ist ein einzeiliges Eingabefeld der Leiste (Hex, Deckkraft); sein Zeiger ist zugleich das Fokus-Ziel.
type eingabe struct {
	text    string // getippter Text (gilt nur mit Fokus und !neu)
	neu     bool   // der nächste getippte Text ersetzt die Anzeige
	fehler  bool   // die letzte Übernahme war ungültig: roter Rand
	rahmen  image.Rectangle
	gezeigt string // angezeigter Text beim letzten Zeichnen (Tests)
}

// farbwaehler hält Lage, Zustand und Bilder des Farbwählers.
type farbwaehler struct {
	flaeche, ton, deck image.Rectangle // Lage beim letzten Zeichnen; die Zeiger sind die Ziele der Zeigerereignisse
	zieht              farbRegler
	hex, deckkraft     eingabe
	hatteFokus         *eingabe // Feld mit Fokus im letzten Bild (feldTasten: Fokus nach dem Verschwinden des Felds)
	wache              int      // Ziel der durchlässigen Fläche über dem Fenster (zeichneFeldWache)
	pipette            widget.Clickable
	pipRahmen          image.Rectangle

	flaechenBild paint.ImageOp
	flaechenTon  float64
	flaecheDa    bool
	tonBild      paint.ImageOp
	tonDa        bool
	deckBild     paint.ImageOp
	deckFarbe    color.NRGBA
	deckDa       bool
	schach       map[image.Point]paint.ImageOp
}

// eingaben sind die Eingabefelder in Tab-Reihenfolge.
func (fw *farbwaehler) eingaben() []*eingabe { return []*eingabe{&fw.hex, &fw.deckkraft} }

// eingaben sind alle Eingabefelder der Leiste in Tab-Reihenfolge: Hex und Deckkraft der Farbe, Deckkraft der Ebene und,
// während einer Umbenennung, der Name (ADR-0032 Punkt 6).
func (e *editor) eingaben() []*eingabe {
	f := append(e.seite.fw.eingaben(), &e.ebenen.deck)
	if e.ebenen.umbenennen != 0 {
		f = append(f, &e.ebenen.name)
	}
	return f
}

// fokusFeld liefert das Eingabefeld mit Tastaturfokus (nil: keins).
func (e *editor) fokusFeld(gtx layout.Context) *eingabe {
	for _, f := range e.eingaben() {
		if gtx.Focused(f) {
			return f
		}
	}
	return nil
}

// feldWert ist der Text, den ein Feld ohne Eingabe zeigt.
func (e *editor) feldWert(f *eingabe) string {
	if t, ok := e.ebenenFeldWert(f); ok {
		return t
	}
	c := e.s.AktuelleFarbe()
	if f == &e.seite.fw.hex {
		return canvas.HexText(c)
	}
	return fmt.Sprintf(i18n.TK("status bar", "%d%%"), canvas.Deckkraft(c))
}

// feldZeichen meldet, ob r in das Feld f getippt werden darf; feldLaenge ist die Höchstlänge.
func (e *editor) feldZeichen(f *eingabe, r rune) bool {
	if f == &e.ebenen.name {
		return !unicode.IsControl(r)
	}
	if f == &e.seite.fw.hex {
		return r == '#' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
	}
	return (r >= '0' && r <= '9') || r == '%'
}

func (e *editor) feldLaenge(f *eingabe) int {
	if f == &e.ebenen.name {
		return document.MaxEbenenName
	}
	if f == &e.seite.fw.hex {
		return 9
	}
	return 4
}

// feldUebernehmen übernimmt die Eingabe; ungültig bleibt sie stehen und das Feld zeigt den Fehler.
func (e *editor) feldUebernehmen(f *eingabe) {
	if f.neu || !e.seiteAn() {
		return
	}
	ok := false
	switch f {
	case &e.ebenen.name:
		ok = e.s.Umbenennen(e.ebenen.umbenennen, f.text) || e.ebenenNameUnveraendert(f.text)
	case &e.ebenen.deck:
		ok = e.s.SetzeEbenenDeckkraftText(f.text)
	case &e.seite.fw.hex:
		ok = e.s.SetzeHex(f.text)
	default:
		ok = e.s.SetzeDeckkraftText(f.text)
	}
	f.fehler = !ok
	if ok {
		f.neu, f.text = true, ""
	}
}

// seiteAn meldet, ob Bedienung in der Leiste wirkt: nicht während des Speicherns und nicht unter der Hilfetafel.
func (e *editor) seiteAn() bool { return !e.s.SpeichernLaeuft() && !e.hilfeOffen }

// feldTasten verarbeitet Fokus, Tasten und getippten Text der Eingabefelder. Es läuft vor tasten: Enter, Esc und die
// Rücktaste gehören dem Feld, solange es den Fokus hat (Gio gibt eine Taste dem ersten passenden Filter des Bildes).
func (e *editor) feldTasten(gtx layout.Context) {
	fw := &e.seite.fw
	// Verschwindet ein Feld mit Fokus (Leiste klappt bei schmalem Fenster ein, Abschnitt zu), setzt Gio den Fokus ohne
	// Ereignis auf nil, und Tasten gingen ins Leere: Er geht an die Leinwand (Karte #220, F-574). Gemerkt wird das Feld
	// mit Fokus vom letzten Bild; hat es keine Lage mehr (die Leiste setzt sie je Bild zurück), war es das Verschwinden.
	// Das Namensfeld hat seine eigene Regel (nameHatteFokus).
	if f := e.fokusFeld(gtx); f != nil {
		fw.hatteFokus = f
	} else if h := fw.hatteFokus; h != nil {
		fw.hatteFokus = nil
		if h.rahmen.Empty() && h != &e.ebenen.name {
			h.neu, h.text, h.fehler = true, "", false
			gtx.Execute(key.FocusCmd{Tag: e})
		}
	}
	// Ein Klick woanders übernimmt eine schwebende Eingabe, bevor der Klick selbst wirkt (ADR-0031 Nachtrag 5,
	// Sitzungs-QA A-2): Die Wache liegt durchlässig über dem ganzen Fenster (zeichneFeldWache), dieser Aufruf läuft vor
	// Leinwand und Knöpfen. Der Fokus geht an die Leinwand; ein Klick in ein anderes Feld fokussiert danach dieses.
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &fw.wache, Kinds: pointer.Press})
		if !ok {
			break
		}
		pe, ok := ev.(pointer.Event)
		f := e.fokusFeld(gtx)
		if !ok || f == nil || pe.Position.Round().Add(image.Pt(0, e.seite.roll.rollen)).In(f.rahmen) {
			continue
		}
		// Ein Klick auf Farbfeld, Pipette, Fläche oder Regler setzt selbst eine Farbe: Die Eingabe wird verworfen statt
		// übernommen — übernommen rückte sie vorn in die zuletzt benutzten Farben und verschöbe das Feld unter dem Klick
		// (Verify V-1).
		if !e.inFarbwahl(pe.Position.Round()) {
			e.feldUebernehmen(f)
		}
		f.neu, f.text, f.fehler = true, "", false
		e.umbenennenEnde(f)
		gtx.Execute(key.FocusCmd{Tag: e})
	}
	// Das Namensfeld einer neuen Umbenennung bekommt den Fokus, sobald es gezeichnet war: Erst dann kennt Gio das Ziel,
	// und die Filter unten melden es im selben Bild an (ADR-0032 Punkt 6).
	if k := &e.ebenen; k.umbenennenFokus && !k.name.rahmen.Empty() {
		k.umbenennenFokus = false
		defer gtx.Execute(key.FocusCmd{Tag: &k.name})
	}
	// Bei offenem Menü oder offener Hilfetafel meldet das Feld keine Tasten an: Esc und Enter gehören dann dem Menü
	// bzw. der Tafel (A-3, A-7). Umschalt ist erlaubt, damit Umschalt+Enter nicht als Kürzel durchrutscht (A-5).
	tasten := !e.menue.offen && !e.hilfeOffen
	for _, f := range e.eingaben() {
		filter := []event.Filter{key.FocusFilter{Target: f}, pointer.Filter{Target: f, Kinds: pointer.Press}}
		if tasten {
			for _, n := range []key.Name{key.NameReturn, key.NameEnter, key.NameEscape, key.NameDeleteBackward} {
				filter = append(filter, key.Filter{Focus: f, Name: n, Optional: key.ModShift})
			}
			if e.s.TextLaeuft() {
				// Während einer Texteingabe gibt Tab aus dem Feld den Fokus an die Leinwand zurück, statt zum nächsten
				// Knopf zu fahren: Dort gingen weitere getippte Zeichen verloren (Karte #233, F-608). Ohne Texteingabe
				// bleibt Tab die Fokusfahrt.
				filter = append(filter, key.Filter{Focus: f, Name: key.NameTab, Optional: key.ModShift})
			}
		}
		for {
			ev, ok := gtx.Event(filter...)
			if !ok {
				break
			}
			switch ev := ev.(type) {
			case key.FocusEvent:
				switch {
				case ev.Focus:
					if f == &e.ebenen.name {
						e.ebenen.nameHatteFokus = true
					}
					if f.neu { // nach einer Unterbrechung (Fenster ohne Fokus) bleibt eine angefangene Eingabe stehen
						f.text, f.fehler = "", false
					}
				case gtx.Focused(f):
					// Das Fenster hat den Fokus verloren, das Feld behält ihn: nichts übernehmen, die Eingabe bleibt
					// stehen (A-2 d) — eine halb getippte Farbe wäre sonst schon gesetzt.
				case f == &e.ebenen.name && !e.ebenen.nameHatteFokus:
					// Das Namensfeld ist eben erst angemeldet und hatte den Fokus noch nicht: Gio meldet „kein Fokus“,
					// bevor der FocusCmd aus zeichneEbenen wirkt; das beendet die Umbenennung nicht.
				default:
					// Tab oder ein Fokuswechsel: übernehmen; ungültig bleibt es beim Alten (ADR-0031 Nachtrag 5).
					e.feldUebernehmen(f)
					f.neu, f.text, f.fehler = true, "", false
					e.umbenennenEnde(f)
				}
			case key.EditEvent:
				if !e.seiteAn() || e.menue.offen {
					continue
				}
				if f.neu {
					f.neu, f.text = false, ""
				}
				for _, r := range ev.Text {
					if e.feldZeichen(f, r) && utf8.RuneCountInString(f.text) < e.feldLaenge(f) {
						f.text += string(r)
					}
				}
				f.fehler = false
			case key.Event:
				if ev.State != key.Press || !e.seiteAn() {
					continue
				}
				switch ev.Name {
				case key.NameReturn, key.NameEnter:
					e.feldUebernehmen(f)
					if f == &e.ebenen.name && f.neu { // übernommen oder nichts getippt: Umbenennen endet
						e.umbenennenEnde(f)
						gtx.Execute(key.FocusCmd{Tag: e})
					}
				case key.NameEscape:
					f.neu, f.text, f.fehler = true, "", false
					e.umbenennenEnde(f)
					gtx.Execute(key.FocusCmd{Tag: e})
				case key.NameTab:
					// Der Fokuswechsel übernimmt die Eingabe (FocusEvent oben, Zweig „Tab oder ein Fokuswechsel“).
					gtx.Execute(key.FocusCmd{Tag: e})
				case key.NameDeleteBackward:
					if f.neu { // vom reinen Wert ohne Einheit aus (A-8: „50 %“ ließ sonst ein Leerzeichen stehen)
						f.neu, f.text = false, e.feldRohwert(f)
					}
					if _, n := utf8.DecodeLastRuneInString(f.text); n > 0 {
						f.text = f.text[:len(f.text)-n]
					}
					f.fehler = false
				}
			case pointer.Event:
				gtx.Execute(key.FocusCmd{Tag: f})
			}
		}
	}
}

// inFarbwahl meldet, ob p (Fensterpixel) auf einem Farbfeld, der Pipette, der Farbfläche oder einem Regler liegt (Lage
// beim letzten Zeichnen).
func (e *editor) inFarbwahl(p image.Point) bool {
	fw := &e.seite.fw
	p = p.Add(image.Pt(0, e.seite.roll.rollen)) // die Lagen sind Inhaltskoordinaten der rollenden Leiste
	ziele := append(e.seite.farbRahmen[:], fw.pipRahmen, fw.flaeche, fw.ton, fw.deck, e.seite.ff.plusRahmen)
	for _, r := range append(ziele, e.seite.ff.rahmen[:]...) { // seit Karte #364 auch die eigenen Farbfelder

		if p.In(r) {
			return true
		}
	}
	return false
}

// feldRohwert ist der Wert eines Felds ohne Einheit, Ausgangspunkt der Rücktaste.
func (e *editor) feldRohwert(f *eingabe) string {
	if f == &e.ebenen.name {
		return e.ebenenNameText()
	}
	if f == &e.ebenen.deck {
		eb, _ := e.s.GewaehlteEbene()
		return strconv.Itoa(eb.Deckkraft())
	}
	c := e.s.AktuelleFarbe()
	if f == &e.seite.fw.hex {
		return canvas.HexText(c)
	}
	return strconv.Itoa(canvas.Deckkraft(c))
}

// zeichneFeldWache meldet über dem ganzen Fenster eine durchlässige Fläche an, die jeden Druck sieht (feldTasten); nur
// solange ein Eingabefeld den Fokus hat.
func (e *editor) zeichneFeldWache(gtx layout.Context) {
	if e.fokusFeld(gtx) == nil {
		return
	}
	c := clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops)
	p := pointer.PassOp{}.Push(gtx.Ops)
	event.Op(gtx.Ops, &e.seite.fw.wache)
	p.Pop()
	c.Pop()
}

// farbeAuswerten verarbeitet Ziehen in Farbfläche und Reglern sowie die Pipette.
func (e *editor) farbeAuswerten(gtx layout.Context) {
	fw := &e.seite.fw
	if fw.pipette.Clicked(gtx) && e.seiteAn() {
		e.s.SchaltePipette()
	}
	for _, r := range []struct {
		tag *image.Rectangle
		art farbRegler
	}{{&fw.flaeche, reglerFlaeche}, {&fw.ton, reglerTon}, {&fw.deck, reglerDeck}} {
		for {
			ev, ok := gtx.Event(pointer.Filter{Target: r.tag,
				Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel})
			if !ok {
				break
			}
			pe, ok := ev.(pointer.Event)
			if !ok {
				continue
			}
			switch pe.Kind {
			case pointer.Press:
				if pe.Buttons != pointer.ButtonPrimary || fw.zieht != reglerKeiner || !e.seiteAn() {
					continue
				}
				if e.s.BeginneFarbZug() {
					fw.zieht = r.art
					e.farbeZiehen(r.art, *r.tag, pe.Position)
				}
			case pointer.Drag, pointer.Release:
				if fw.zieht != r.art {
					continue
				}
				e.farbeZiehen(r.art, *r.tag, pe.Position)
				if pe.Kind == pointer.Release {
					e.s.BeendeFarbZug()
					fw.zieht = reglerKeiner
				}
			case pointer.Cancel:
				if fw.zieht == r.art {
					e.s.BrecheFarbZugAb()
					fw.zieht = reglerKeiner
				}
			}
		}
	}
}

// farbeZiehen setzt die Farbe aus der Lage p (relativ zum Element r) im Regler art.
func (e *editor) farbeZiehen(art farbRegler, r image.Rectangle, p f32.Point) {
	if r.Dx() <= 0 || r.Dy() <= 0 {
		return
	}
	fx := math.Max(0, math.Min(1, float64(p.X)/float64(r.Dx())))
	fy := math.Max(0, math.Min(1, float64(p.Y)/float64(r.Dy())))
	h, a := e.s.FarbeHSV(), e.s.AktuelleFarbe().A
	switch art {
	case reglerFlaeche:
		h.S, h.V = fx, 1-fy
	case reglerTon:
		h.H = math.Min(fx*360, 359.99) // ganz rechts bleibt Rot am Ende, statt auf Rot am Anfang zu springen
		// Füllung „keine“ ({0,0,0,0} ist Schwarz): der Ton allein machte sie deckend schwarz; mit voller Sättigung und
		// Helligkeit zeigt der Regler den gewählten Ton (Kritik B-1).
		if a == 0 {
			h.S, h.V = 1, 1
		}
	case reglerDeck:
		a = uint8(math.Round(fx * 255))
	}
	if art != reglerDeck && a == 0 {
		a = 255 // eine ganz durchsichtige Farbe (Füllung „keine“) wird mit der Wahl eines Tons sichtbar (#219)
	}
	e.s.FarbZugHSV(h, a)
}

// --- Zeichnen ---

// bildIn zeichnet das Bild b gestreckt in r.
func bildIn(gtx layout.Context, b paint.ImageOp, r image.Rectangle, filter paint.ImageFilter) {
	g := b.Size()
	if g.X == 0 || g.Y == 0 || r.Empty() {
		return
	}
	b.Filter = filter
	c := clip.Rect(r).Push(gtx.Ops)
	t := op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(float32(r.Dx())/float32(g.X), float32(r.Dy())/float32(g.Y))).
		Offset(f32.Pt(float32(r.Min.X), float32(r.Min.Y)))).Push(gtx.Ops)
	b.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	t.Pop()
	c.Pop()
}

// schachIn zeichnet ein kleines Schachbrett (Felder zelle Pixel) in r, darunter liegt, was durchsichtig ist.
func (fw *farbwaehler) schachIn(gtx layout.Context, r image.Rectangle, zelle int) {
	if r.Empty() || zelle <= 0 {
		return
	}
	n := image.Pt((r.Dx()+zelle-1)/zelle, (r.Dy()+zelle-1)/zelle)
	if fw.schach == nil {
		fw.schach = map[image.Point]paint.ImageOp{}
	}
	b, ok := fw.schach[n]
	if !ok {
		m := image.NewNRGBA(image.Rectangle{Max: n})
		for y := 0; y < n.Y; y++ {
			for x := 0; x < n.X; x++ {
				c := color.NRGBA{R: 0x9A, G: 0x9E, B: 0xA3, A: 0xFF}
				if (x+y)%2 == 0 {
					c = color.NRGBA{R: 0xD4, G: 0xD6, B: 0xD9, A: 0xFF}
				}
				m.SetNRGBA(x, y, c)
			}
		}
		b = paint.NewImageOp(m)
		fw.schach[n] = b
	}
	bildIn(gtx, b, image.Rectangle{Min: r.Min, Max: r.Min.Add(n.Mul(zelle))}.Intersect(r), paint.FilterNearest)
}

// flaechenBildFuer liefert die Farbfläche zum Farbton h (Sättigung nach rechts, Hellwert nach oben).
func (fw *farbwaehler) flaechenBildFuer(h float64) paint.ImageOp {
	if !fw.flaecheDa || fw.flaechenTon != h {
		const n = 64
		m := image.NewNRGBA(image.Rect(0, 0, n, n))
		for y := 0; y < n; y++ {
			for x := 0; x < n; x++ {
				m.SetNRGBA(x, y, canvas.HSV{H: h, S: float64(x) / (n - 1), V: 1 - float64(y)/(n-1)}.Farbe(255))
			}
		}
		fw.flaechenBild, fw.flaechenTon, fw.flaecheDa = paint.NewImageOp(m), h, true
	}
	return fw.flaechenBild
}

func (fw *farbwaehler) tonBildHolen() paint.ImageOp {
	if !fw.tonDa {
		m := image.NewNRGBA(image.Rect(0, 0, 360, 1))
		for x := 0; x < 360; x++ {
			m.SetNRGBA(x, 0, canvas.HSV{H: float64(x), S: 1, V: 1}.Farbe(255))
		}
		fw.tonBild, fw.tonDa = paint.NewImageOp(m), true
	}
	return fw.tonBild
}

func (fw *farbwaehler) deckBildFuer(c color.NRGBA) paint.ImageOp {
	c.A = 0
	if !fw.deckDa || fw.deckFarbe != c {
		const n = 64
		m := image.NewNRGBA(image.Rect(0, 0, n, 1))
		for x := 0; x < n; x++ {
			d := c
			d.A = uint8(math.Round(float64(x) * 255 / (n - 1)))
			m.SetNRGBA(x, 0, d)
		}
		fw.deckBild, fw.deckFarbe, fw.deckDa = paint.NewImageOp(m), c, true
	}
	return fw.deckBild
}

// marke zeichnet eine Lagemarke: hell mit dunklem Rand, auf jeder Farbe zu sehen.
func marke(gtx layout.Context, r image.Rectangle) {
	paint.FillShape(gtx.Ops, farbeGrund, clip.Stroke{Path: clip.Rect(r.Inset(-gtx.Dp(1))).Path(),
		Width: float32(gtx.Dp(1))}.Op())
	paint.FillShape(gtx.Ops, color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}, clip.Stroke{Path: clip.Rect(r).Path(),
		Width: float32(gtx.Dp(2))}.Op())
}

// reglerAnmelden meldet r als Ziel der Zeigerereignisse tag an (Ursprung bei r.Min, Zeiger Fadenkreuz).
func reglerAnmelden(gtx layout.Context, tag event.Tag, r image.Rectangle) {
	t := op.Offset(r.Min).Push(gtx.Ops)
	c := clip.Rect(image.Rectangle{Max: r.Size()}).Push(gtx.Ops)
	event.Op(gtx.Ops, tag)
	pointer.CursorCrosshair.Add(gtx.Ops)
	c.Pop()
	t.Pop()
}

// farbFeld zeichnet ein Farbfeld (Schachbrett unter halbdurchsichtigen Farben, heller Rand, grün wenn gewählt).
func (e *editor) farbFeld(gtx layout.Context, c color.NRGBA, r image.Rectangle, gewaehlt, ueber bool) {
	if gewaehlt { // grüner Rahmen außen (Grün heißt aktiv, Orange bleibt dem Fokus)
		paint.FillShape(gtx.Ops, farbeAuswahl, clip.Rect(r.Inset(-gtx.Dp(2))).Op())
	} else if ueber {
		paint.FillShape(gtx.Ops, farbeNebentext, clip.Rect(r.Inset(-gtx.Dp(1))).Op())
	}
	if c.A < 255 {
		e.seite.fw.schachIn(gtx, r, gtx.Dp(6))
	}
	paint.FillShape(gtx.Ops, c, clip.Rect(r).Op())
	// Heller Rand, damit auch Dunkelgrau auf der dunklen Leiste zu sehen ist (Sitzungs-QA A-9).
	paint.FillShape(gtx.Ops, farbeNebentext, clip.Stroke{Path: clip.Rect(r).Path(), Width: float32(gtx.Dp(1))}.Op())
}

// zweiteFarbe legt bei gemischten Farben im Feld f (mehrere gewählte, ADR-0039 Nachtrag 4 Punkt 4) die nächste andere
// Farbe als Dreieck unten rechts über das Farbmuster r; oben links bleibt die gezeigte.
func (e *editor) zweiteFarbe(gtx layout.Context, f canvas.Farbfeld, r image.Rectangle) {
	c, ok := e.s.ZweiteFarbe(f)
	if !ok {
		return
	}
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(float32(r.Max.X), float32(r.Min.Y)))
	p.LineTo(f32.Pt(float32(r.Max.X), float32(r.Max.Y)))
	p.LineTo(f32.Pt(float32(r.Min.X), float32(r.Max.Y)))
	p.Close()
	dreieck := clip.Outline{Path: p.End()}.Op()
	if c.A < 255 {
		st := dreieck.Push(gtx.Ops)
		e.seite.fw.schachIn(gtx, r, gtx.Dp(6))
		st.Pop()
	}
	paint.FillShape(gtx.Ops, c, dreieck)
	if c.A == 0 {
		zeichneKeine(gtx, r)
	}
	paint.FillShape(gtx.Ops, farbeNebentext, clip.Stroke{Path: clip.Rect(r).Path(), Width: float32(gtx.Dp(1))}.Op())
}

// zeichneFarbwaehler zeichnet den Farbwähler ab y in der Breite breite bei x und liefert das neue y.
func (e *editor) zeichneFarbwaehler(gtx layout.Context, x, y, breite, spalte int) int {
	k := &e.seite
	fw := &k.fw
	aktuell, hsv := e.s.AktuelleFarbe(), e.s.FarbeHSV()
	q, abst := gtx.Dp(24), gtx.Dp(6)
	y = e.zeichneFarbpaarZeile(gtx, x, y, breite, q)
	raster := 2*q + abst
	hoch := 4*q + 3*abst

	// Farbfläche links.
	fw.flaeche = image.Rect(x, y, x+breite-raster-gtx.Dp(10), y+hoch)
	bildIn(gtx, fw.flaechenBildFuer(hsv.H), fw.flaeche, paint.FilterLinear)
	paint.FillShape(gtx.Ops, farbeLinie, clip.Stroke{Path: clip.Rect(fw.flaeche).Path(), Width: float32(gtx.Dp(1))}.Op())
	mp := image.Pt(fw.flaeche.Min.X+int(math.Round(hsv.S*float64(fw.flaeche.Dx()))),
		fw.flaeche.Min.Y+int(math.Round((1-hsv.V)*float64(fw.flaeche.Dy()))))
	{
		rc := clip.Rect(fw.flaeche.Inset(-gtx.Dp(4))).Push(gtx.Ops)
		marke(gtx, image.Rectangle{Min: mp, Max: mp}.Inset(-gtx.Dp(4)))
		rc.Pop()
	}
	reglerAnmelden(gtx, &fw.flaeche, fw.flaeche)

	// Raster rechts: die zuletzt benutzten Farben (seit Karte #364; die eigenen Farbfelder stehen unter Hex und
	// Deckkraft, die Pipette seit Karte #318 in der Zeile des Farbpaars). Freie Plätze zeigen einen leeren Rahmen.
	rx := x + breite - raster
	platz := func(i int) image.Rectangle {
		px, py := rx+(i%2)*(q+abst), y+(i/2)*(q+abst)
		return image.Rect(px, py, px+q, py+q)
	}
	farben := e.s.Zuletzt()
	for j := range k.farbRahmen {
		k.farbRahmen[j] = image.Rectangle{}
		if j >= len(farben) {
			paint.FillShape(gtx.Ops, farbeLinie, clip.Stroke{Path: clip.Rect(platz(j)).Path(), Width: float32(gtx.Dp(1))}.Op())
		}
	}
	for j, c := range farben {
		if j >= farbPlaetze {
			break
		}
		r := platz(j)
		k.farbRahmen[j] = r
		e.farbFeld(gtx, c, r, c == aktuell, k.farben[j].Hovered())
		e.knopfAn(gtx, &k.farben[j], r)
	}
	y += hoch + gtx.Dp(12)

	// Farbton- und Deckkraft-Regler über die volle Breite.
	rh := gtx.Dp(12)
	fw.ton = image.Rect(x, y, x+breite, y+rh)
	bildIn(gtx, fw.tonBildHolen(), fw.ton, paint.FilterLinear)
	tx := fw.ton.Min.X + int(math.Round(hsv.H/360*float64(fw.ton.Dx())))
	marke(gtx, image.Rect(tx-gtx.Dp(2), fw.ton.Min.Y-gtx.Dp(2), tx+gtx.Dp(2), fw.ton.Max.Y+gtx.Dp(2)))
	reglerAnmelden(gtx, &fw.ton, fw.ton)
	y += rh + gtx.Dp(10)

	fw.deck = image.Rect(x, y, x+breite, y+rh)
	fw.schachIn(gtx, fw.deck, gtx.Dp(6))
	bildIn(gtx, fw.deckBildFuer(aktuell), fw.deck, paint.FilterLinear)
	dx := fw.deck.Min.X + int(math.Round(float64(aktuell.A)/255*float64(fw.deck.Dx())))
	marke(gtx, image.Rect(dx-gtx.Dp(2), fw.deck.Min.Y-gtx.Dp(2), dx+gtx.Dp(2), fw.deck.Max.Y+gtx.Dp(2)))
	reglerAnmelden(gtx, &fw.deck, fw.deck)
	y += rh + gtx.Dp(14)

	// Eingaben Hex und Deckkraft.
	fh := gtx.Dp(26)
	e.seiteText(gtx, i18n.T("Hex"), 13, farbeNebentext, image.Pt(x, y+gtx.Dp(5)), gtx.Dp(36), 1)
	fw.hex.rahmen = image.Rect(x+gtx.Dp(36), y, x+gtx.Dp(36)+gtx.Dp(100), y+fh)
	e.zeichneEingabe(gtx, &fw.hex)
	dl := x + spalte + gtx.Dp(64)
	e.seiteText(gtx, i18n.T("Opacity"), 13, farbeNebentext, image.Pt(dl, y+gtx.Dp(5)), x+breite-gtx.Dp(58)-dl, 1)
	fw.deckkraft.rahmen = image.Rect(x+breite-gtx.Dp(54), y, x+breite, y+fh)
	e.zeichneEingabe(gtx, &fw.deckkraft)
	return e.zeichneFarbfelder(gtx, x, y+fh+gtx.Dp(farbfelderAbstandDp), breite)
}

// zeichneFarbpaarZeile zeichnet über dem Farbwähler die Reiter der Farbfelder (ADR-0031 Nachtrag 13, Karte #369; vorher
// das Farbpaar mit dem Namen des aktiven Felds, ADR-0021 Nachtrag 23): je Feld ein Reiter mit Farbmuster und Namen —
// „Farbe“, „Füllung“ (nur, wo es eine gibt), „Schatten“ (nur, wenn er an ist) —, der Reiter des Felds, das der Farbwähler
// bearbeitet, hervorgehoben; zwischen Farbe und Füllung Tauschen, rechts die Pipette. „Keine“ steht seitdem vorn in den
// Farbfeldern (zeichneFarbfelder). Liefert das neue y.
func (e *editor) zeichneFarbpaarZeile(gtx layout.Context, x, y, breite, q int) int {
	k := &e.seite
	aktiv := e.s.AktivesFeld()
	fuellbar, schatten := e.s.Fuellbar(), e.s.Schatten().An
	h, abst, tauschB := gtx.Dp(28), gtx.Dp(4), gtx.Dp(18)
	for i := range k.paarRahmen {
		k.paarRahmen[i] = image.Rectangle{}
	}
	k.reiterAktiv = image.Rectangle{}
	fw := &k.fw
	fw.pipRahmen = image.Rect(x+breite-q, y+(h-q)/2, x+breite, y+(h-q)/2+q)
	ende := fw.pipRahmen.Min.X - gtx.Dp(8)
	felder := []canvas.Farbfeld{canvas.FeldVordergrund}
	if fuellbar {
		felder = append(felder, canvas.FeldFuellung)
	}
	if schatten {
		felder = append(felder, canvas.FeldSchatten)
	}
	frei := ende - x - (len(felder)-1)*abst
	if fuellbar {
		frei -= tauschB + abst
	}
	breiten := farbReiterBreiten(frei, func(f canvas.Farbfeld) int {
		return gtx.Dp(farbReiterRandDp) + e.feldTextBreite(gtx, farbReiterName(f), 12)
	}, felder)
	px := x
	reiter := func(i int, f canvas.Farbfeld) {
		w := breiten[slices.Index(felder, f)]
		r := image.Rect(px, y, px+w, y+h)
		k.paarRahmen[i] = r
		e.zeichneFarbReiter(gtx, &k.paar[i], r, f, farbReiterName(f), aktiv == f)
		px = r.Max.X + abst
	}
	reiter(paarVorn, canvas.FeldVordergrund)
	if fuellbar {
		k.paarRahmen[paarTausch] = image.Rect(px, y+(h-tauschB)/2, px+tauschB, y+(h-tauschB)/2+tauschB)
		e.tauschKnopf(gtx, &k.paar[paarTausch], k.paarRahmen[paarTausch])
		px += tauschB + abst
		reiter(paarFuellung, canvas.FeldFuellung)
	}
	if schatten {
		reiter(paarSchatten, canvas.FeldSchatten)
	}
	// Pipette rechts in der Zeile (Karte #318): Sie holt eine Farbe für das aktive Feld.
	{
		r, an := fw.pipRahmen, e.s.PipetteLaeuft()
		grund, rnd := farbeGrund, farbeLinie
		switch {
		case an:
			grund, rnd = farbeAuswahl, farbeAuswahl
		case fw.pipette.Hovered():
			rnd = farbeNebentext
		}
		paint.FillShape(gtx.Ops, grund, clip.Rect(r).Op())
		paint.FillShape(gtx.Ops, rnd, clip.Stroke{Path: clip.Rect(r).Path(), Width: float32(gtx.Dp(1))}.Op())
		seite := r.Dx() - 2*gtx.Dp(2)
		zeichneSymbol(gtx, icons.Pipette, r.Min.Add(image.Pt(gtx.Dp(2), gtx.Dp(2))), seite, farbeText, farbeAkzent)
		e.knopfAn(gtx, &fw.pipette, r)
	}
	return y + h + gtx.Dp(10)
}

// zeichneFarbReiter zeichnet einen Reiter des Farbziels in r: Farbmuster (Schachbrett unter Durchsichtigem, roter
// Schrägstrich bei „keine“) und Name; aktiv mit heller Fläche, grünem Rand und grüner Unterkante (Grün heißt aktiv,
// Orange bleibt dem Fokus).
func (e *editor) zeichneFarbReiter(gtx layout.Context, c *widget.Clickable, r image.Rectangle, feld canvas.Farbfeld, name string, aktiv bool) {
	farbe := e.s.Farbe(feld)
	rnd, text := farbeLinie, farbeNebentext
	switch {
	case aktiv:
		paint.FillShape(gtx.Ops, farbeGrund, clip.Rect(r).Op())
		rnd, text = farbeAuswahl, farbeText
	case c.Hovered():
		rnd = farbeNebentext
	}
	paint.FillShape(gtx.Ops, rnd, clip.Stroke{Path: clip.Rect(r).Path(), Width: float32(gtx.Dp(1))}.Op())
	if aktiv {
		paint.FillShape(gtx.Ops, farbeAuswahl, clip.Rect(image.Rect(r.Min.X, r.Max.Y-gtx.Dp(3), r.Max.X, r.Max.Y)).Op())
		e.seite.reiterAktiv = r
	}
	sq := gtx.Dp(12)
	mr := image.Rect(r.Min.X+gtx.Dp(5), r.Min.Y+(r.Dy()-sq)/2, r.Min.X+gtx.Dp(5)+sq, r.Min.Y+(r.Dy()-sq)/2+sq)
	e.farbFeld(gtx, farbe, mr, false, false)
	if farbe.A == 0 {
		zeichneKeine(gtx, mr)
	}
	e.zweiteFarbe(gtx, feld, mr)
	tx := mr.Max.X + gtx.Dp(4)
	e.seiteText(gtx, name, 12, text, image.Pt(tx, r.Min.Y+(r.Dy()-gtx.Dp(16))/2), max(0, r.Max.X-gtx.Dp(2)-tx), 1)
	e.knopfAn(gtx, c, r)
}

// farbReiterRandDp ist, was ein Reiter neben seinem Namen braucht: Rand, Farbmuster, Abstand, Rand (5 + 12 + 4 + 3).
const farbReiterRandDp = 24

// farbReiterBreiten verteilt frei Pixel auf die Reiter der Felder: Jeder bekommt, was sein Name braucht (bedarf), der
// Rest geht zu gleichen Teilen dazu; reicht frei nicht, gleich breit (die Namen kürzt dann die Leiste mit „…“).
func farbReiterBreiten(frei int, bedarf func(canvas.Farbfeld) int, felder []canvas.Farbfeld) []int {
	b := make([]int, len(felder))
	summe := 0
	for i, f := range felder {
		b[i] = bedarf(f)
		summe += b[i]
	}
	if summe > frei {
		for i := range b {
			b[i] = max(0, frei/len(felder))
		}
		return b
	}
	for i := range b {
		b[i] += (frei - summe) / len(felder)
	}
	return b
}

// farbReiterName ist der Name des Reiters zum Feld f (Karte #359, ADR-0031 Nachtrag 13).
func farbReiterName(f canvas.Farbfeld) string {
	switch f {
	case canvas.FeldFuellung:
		return i18n.TK("color field", "Fill")
	case canvas.FeldSchatten:
		return i18n.TK("color field", "Shadow")
	}
	return i18n.TK("color field", "Color")
}

// zeichneEingabe zeichnet ein Eingabefeld: Rand orange mit Fokus, rot bei ungültiger Eingabe, Schreibmarke am Ende.
func (e *editor) zeichneEingabe(gtx layout.Context, f *eingabe) {
	r := f.rahmen
	fokus := gtx.Focused(f)
	if fokus {
		e.seite.roll.fokus = r
	}
	text := e.feldWert(f)
	if fokus && !f.neu {
		text = f.text
	}
	f.gezeigt = text
	rnd := farbeLinie
	switch {
	case f.fehler:
		rnd = farbeRot
	case fokus:
		rnd = farbeAkzent
	}
	paint.FillShape(gtx.Ops, farbeGrund, clip.Rect(r).Op())
	paint.FillShape(gtx.Ops, rnd, clip.Stroke{Path: clip.Rect(r).Path(), Width: float32(gtx.Dp(1))}.Op())
	innen := r.Min.Add(image.Pt(gtx.Dp(6), gtx.Dp(5)))
	e.seiteText(gtx, text, 13, farbeText, innen, r.Dx()-gtx.Dp(10), 1)
	if fokus {
		// Schreibmarke hinter dem Text: Breite gemessen mit demselben Satz wie die Anzeige.
		w := e.feldTextBreite(gtx, text, 13)
		mx := min(innen.X+w+gtx.Dp(1), r.Max.X-gtx.Dp(3))
		paint.FillShape(gtx.Ops, farbeText, clip.Rect(image.Rect(mx, r.Min.Y+gtx.Dp(5), mx+gtx.Dp(1), r.Max.Y-gtx.Dp(5))).Op())
	}
	t := op.Offset(r.Min).Push(gtx.Ops)
	c := clip.Rect(image.Rectangle{Max: r.Size()}).Push(gtx.Ops)
	event.Op(gtx.Ops, f)
	pointer.CursorText.Add(gtx.Ops)
	c.Pop()
	t.Pop()
}

// feldTextBreite misst die Breite von text in sp (Maß der Leiste), ohne zu zeichnen.
func (e *editor) feldTextBreite(gtx layout.Context, text string, sp float32) int {
	lbl := material.Label(e.thema, unit.Sp(sp), text)
	lbl.MaxLines = 1
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(gtx.Dp(400), gtx.Dp(100))}
	m := op.Record(gtx.Ops)
	d := lbl.Layout(g)
	m.Stop()
	return d.Size.X
}
