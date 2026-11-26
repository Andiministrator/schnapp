// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Eigene Farbfelder unter Hex und Deckkraft (ADR-0021 Nachtrag 24, Karte #364): ein Klick setzt die Farbe, „+“ nimmt die
// aktuelle Farbe auf, Rechtsklick oder Entf entfernt ein Feld. Gespeichert in colors.ini (settings.AendereFarben);
// ohne Datei gelten canvas.StandardFarbfelder.

import (
	"fmt"
	"image"
	"image/color"
	"slices"

	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// Nähte für Tests (TestMain setzt XDG_CONFIG_HOME, Tests können sie ersetzen).
var (
	ladeFarben    = settings.LadeFarben
	aendereFarben = settings.AendereFarben
)

// Maße der Farbfelder in dp.
const (
	farbfeldDp          = 20
	farbfeldAbstDp      = 6
	farbfelderAbstandDp = 8 // zwischen Hex/Deckkraft und der ersten Reihe
)

// farbfeldLeiste ist der Zustand der Farbfelder; die Rahmen halten die Lage beim letzten Zeichnen (Tests).
type farbfeldLeiste struct {
	farben     []color.NRGBA // nur gültig mit geladen; sonst gelten canvas.StandardFarbfelder (Editor ohne Laden, Tests)
	geladen    bool
	knoepfe    [settings.MaxFarbfelder]widget.Clickable
	rechts     [settings.MaxFarbfelder]bool // Ziele der Rechtsklicks (nur die Adresse zählt)
	plus       widget.Clickable
	rahmen     [settings.MaxFarbfelder]image.Rectangle
	plusRahmen image.Rectangle
	hinweis    string // Text der Statuszeile zum Feld mit Fokus oder Zeiger
}

func (f *farbfeldLeiste) alle() []*widget.Clickable {
	f.liste()
	a := make([]*widget.Clickable, 0, len(f.farben)+1)
	for i := range f.farben {
		a = append(a, &f.knoepfe[i])
	}
	return append(a, &f.plus)
}

// farbfelderLaden liest colors.ini beim Öffnen des Editors; Warnungen gehen auf die Fehlerausgabe wie bei styles.ini.
func (e *editor) farbfelderLaden() {
	farben, da, warn := ladeFarben()
	for _, w := range warn {
		fmt.Fprintln(fehlerAusgabe, w)
	}
	if !da {
		farben = slices.Clone(canvas.StandardFarbfelder)
	}
	e.seite.ff.farben, e.seite.ff.geladen = farben, true
}

// farbfelder sind die Farbfelder dieses Editors; vor dem Laden die eingebauten.
func (f *farbfeldLeiste) liste() []color.NRGBA {
	if !f.geladen {
		f.farben, f.geladen = slices.Clone(canvas.StandardFarbfelder), true
	}
	return f.farben
}

// farbfelderAendern ändert colors.ini um genau die eigene Änderung (unter einer Sperre neu gelesen, damit ein zweites
// Fenster nichts verliert); ohne Datei beginnt sie bei den eingebauten Feldern. Danach gilt der Stand der Datei.
func (e *editor) farbfelderAendern(erfolg string, aend func([]color.NRGBA) []color.NRGBA) {
	neu, err := aendereFarben(func(alt []color.NRGBA, da bool) []color.NRGBA {
		if !da {
			alt = slices.Clone(canvas.StandardFarbfelder)
		}
		return aend(alt)
	})
	if err != nil {
		e.melde(fmt.Sprintf(i18n.T("Could not save the color swatches: %v"), fehlerGrund(err)), true)
		return
	}
	e.seite.ff.farben, e.seite.ff.geladen = neu, true
	e.melde(erfolg, false)
}

// farbfelderAuswerten verarbeitet Klicks, Rechtsklicks und „+“ (aus seiteAuswerten).
func (e *editor) farbfelderAuswerten(gtx layout.Context) {
	f := &e.seite.ff
	f.liste()
	// Nach einem Entfernen gilt die Lage des letzten Bilds nicht mehr (die Felder rücken nach): weitere Klicks dieses
	// Bilds verfallen, sonst träfen sie den Nachbarn oder lägen hinter dem Ende der kürzeren Liste (F-836).
	n := len(f.farben)
	for j := 0; j < n; j++ {
		if f.knoepfe[j].Clicked(gtx) && e.seiteAn() && j < len(f.farben) && len(f.farben) == n {
			e.s.SetzeFarbe(f.farben[j])
		}
		for {
			ev, ok := gtx.Event(pointer.Filter{Target: &f.rechts[j], Kinds: pointer.Press})
			if !ok {
				break
			}
			if pe, ok := ev.(pointer.Event); ok && pe.Buttons == pointer.ButtonSecondary && e.seiteAn() && len(f.farben) == n {
				e.farbfeldEntferne(gtx, j)
			}
		}
	}
	if f.plus.Clicked(gtx) && e.seiteAn() && e.kannFarbfeldHinzu() {
		c := e.s.AktuelleFarbe()
		e.farbfelderAendern(i18n.T("Color swatch added"), func(alt []color.NRGBA) []color.NRGBA {
			return settings.FarbeHinzu(alt, c)
		})
	}
}

// kannFarbfeldHinzu meldet, ob „+“ etwas täte: die aktuelle Farbe ist sichtbar, noch kein Feld und es ist Platz.
func (e *editor) kannFarbfeldHinzu() bool {
	f := &e.seite.ff
	c := e.s.AktuelleFarbe()
	return c.A > 0 && !slices.Contains(f.liste(), c) && len(f.farben) < settings.MaxFarbfelder
}

// farbfeldEntferne entfernt das Feld j (Rechtsklick, Entf). Der Fokus bleibt an der Stelle, beim letzten Feld rückt er
// auf „+“.
func (e *editor) farbfeldEntferne(gtx layout.Context, j int) {
	f := &e.seite.ff
	if j < 0 || j >= len(f.farben) {
		return
	}
	weg := f.farben[j]
	if gtx.Focused(&f.knoepfe[j]) && j == len(f.farben)-1 {
		gtx.Execute(key.FocusCmd{Tag: &f.plus})
	}
	e.farbfelderAendern(i18n.T("Color swatch removed"), func(alt []color.NRGBA) []color.NRGBA {
		return settings.FarbeEntfernt(alt, weg)
	})
}

// farbfeldTaste behandelt Entf, solange ein Farbfeld den Fokus hat; liefert, ob die Taste verbraucht ist. Entf gehört
// dem Feld: sonst löschte es das gewählte Objekt.
func (e *editor) farbfeldTaste(gtx layout.Context, taste key.Event) bool {
	f := &e.seite.ff
	if taste.Name != key.NameDeleteForward || taste.Modifiers != 0 {
		return false
	}
	for j := range f.farben {
		if gtx.Focused(&f.knoepfe[j]) {
			e.farbfeldEntferne(gtx, j)
			return true
		}
	}
	return false
}

// zeichneFarbfelder zeichnet die Farbfelder ab y in Reihen über die Breite, „+“ am Ende, und liefert das neue y.
func (e *editor) zeichneFarbfelder(gtx layout.Context, x, y, breite int) int {
	f := &e.seite.ff
	f.rahmen, f.plusRahmen, f.hinweis = [settings.MaxFarbfelder]image.Rectangle{}, image.Rectangle{}, ""
	f.liste()
	q, abst := gtx.Dp(farbfeldDp), gtx.Dp(farbfeldAbstDp)
	je := max(1, (breite+abst)/(q+abst))
	platz := func(i int) image.Rectangle {
		px, py := x+(i%je)*(q+abst), y+(i/je)*(q+abst)
		return image.Rect(px, py, px+q, py+q)
	}
	aktuell := e.s.AktuelleFarbe()
	// „Keine“ vorn, solange die Füllung das Ziel ist (ADR-0031 Nachtrag 13; vorher in der Zeile des Farbpaars): weiß mit
	// rotem Schrägstrich wie Inkscape/Photoshop, ein Klick macht die Füllung durchsichtig.
	vorn := 0
	if e.keineZeigen() {
		k := &e.seite
		rk, c := platz(0), &k.paar[paarKeine]
		k.paarRahmen[paarKeine] = rk
		e.farbFeld(gtx, farbeText, rk, aktuell.A == 0, c.Hovered())
		zeichneKeine(gtx, rk)
		e.knopfAn(gtx, c, rk)
		if c.Hovered() || gtx.Focused(c) {
			f.hinweis = i18n.T("No fill")
		}
		vorn = 1
	}
	for j, c := range f.farben {
		r := platz(j + vorn)
		f.rahmen[j] = r
		e.farbFeld(gtx, c, r, c == aktuell, f.knoepfe[j].Hovered())
		e.knopfAn(gtx, &f.knoepfe[j], r)
		rechtsAnmelden(gtx, &f.rechts[j], r)
		if f.knoepfe[j].Hovered() || gtx.Focused(&f.knoepfe[j]) {
			f.hinweis = fmt.Sprintf(i18n.T("Color swatch %s. Right-click or Del removes it."), canvas.HexText(c))
		}
	}
	pr := platz(len(f.farben) + vorn)
	f.plusRahmen = pr
	an := e.kannFarbfeldHinzu()
	zeichnePlus(gtx, pr, an, f.plus.Hovered())
	e.knopfAn(gtx, &f.plus, pr)
	if f.plus.Hovered() || gtx.Focused(&f.plus) {
		f.hinweis = e.farbfeldPlusText()
	}
	return pr.Max.Y
}

// farbfelderHoehe ist die Höhe, die die Farbfelder unter Hex und Deckkraft dem Farbwähler hinzufügen (Abstand und Reihen
// einschließlich „+“), in Pixeln bei der Breite breite: Die Schwelle für das Einklappen der Ebenen rechnet damit
// (farbwaehlerHoeheDp reicht bis unter Hex und Deckkraft; ADR-0021 Nachtrag 24 Punkt 7).
func (e *editor) farbfelderHoehe(gtx layout.Context, breite int) int {
	q, abst := gtx.Dp(farbfeldDp), gtx.Dp(farbfeldAbstDp)
	je := max(1, (breite+abst)/(q+abst))
	vorn := 0
	if e.keineZeigen() {
		vorn = 1
	}
	reihen := (len(e.seite.ff.liste()) + vorn + 1 + je - 1) / je
	return gtx.Dp(farbfelderAbstandDp) + reihen*(q+abst) - abst
}

// keineZeigen meldet, ob vorn in den Farbfeldern „Keine“ steht: wenn die Füllung das Ziel des Farbwählers ist.
func (e *editor) keineZeigen() bool {
	return e.s.Fuellbar() && e.s.AktivesFeld() == canvas.FeldFuellung
}

// farbfeldPlusText ist der Hinweis zu „+“.
func (e *editor) farbfeldPlusText() string {
	f := &e.seite.ff
	c := e.s.AktuelleFarbe()
	f.liste()
	switch {
	case c.A == 0:
		return i18n.T("A fully transparent color cannot be a swatch.")
	case slices.Contains(f.farben, c):
		return i18n.T("The current color is already a swatch.")
	case len(f.farben) >= settings.MaxFarbfelder:
		return fmt.Sprintf(i18n.T("At most %d color swatches. Remove one first."), settings.MaxFarbfelder)
	}
	return i18n.T("Adds the current color as a swatch.")
}
