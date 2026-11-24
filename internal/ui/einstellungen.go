// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Einstellungstafel im Editor (ADR-0034): Menü Datei → „Settings…“, Strg+,. Die Tafel liegt modal über dem Editor wie
// die Hilfe: Darunter ist alles abgedunkelt, ein Zeigerziel über dem ganzen Fenster fängt Klicks und Mausrad ab, und
// solange sie offen ist, gehören die Tasten ihr (einstellungenAuswerten läuft vor feldTasten und tasten; tasten
// verwirft den Rest). Den Tastaturfokus hält die Tafel selbst (Gio-Fokus auf &e.einst, die Stelle in tafel.fokus):
// So bleibt Tab in der Tafel und läuft nicht in die Leisten darunter. Auswahllisten laufen über die Liste des Menüs
// (listeEinstellung). Bedeutung und Zustand stehen in einstellungen_logik.go.

import (
	"errors"
	"fmt"
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/font"
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

	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// Maße der Tafel in dp (Muster #175).
const (
	tafelBreiteDp = 800
	tafelHoeheDp  = 640
	tafelKopfDp   = 50
	tafelFussDp   = 60
	tafelLinksDp  = 180 // Spalte der Reiter
	tafelFeldDp   = 30  // Höhe eines Felds
)

var (
	farbeWarnGrund = color.NRGBA{R: 0x3D, G: 0x33, B: 0x24, A: 0xFF} // orange-900 auf grau, Warnkasten der Agenten
	farbeWeiss     = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	// farbeTafelDunkel dunkelt den Editor unter der Tafel ab (Muster #175: 60 % grau-950). Seit ADR-0020 Nachtrag 14
	// mischt Gio in sRGB wie das Muster; 0x99 ergibt dieselbe Helligkeit wie dort.
	farbeTafelDunkel = color.NRGBA{R: 0x16, G: 0x18, B: 0x1B, A: 0x99}
)

// tafelRahmen ist die Lage der Teile beim letzten Zeichnen in Fensterpixeln (Tests, Drehbücher, Liste am Feld).
type tafelRahmen struct {
	tafel, kreuz, abbrechen, ok, minus, plus image.Rectangle
	hinweis                                  image.Rectangle // Hinweis zum Ort der Agenten (F-625)
	reiter                                   [reiterAnzahl]image.Rectangle
	feld                                     [len(tafelZeilen)]image.Rectangle
	warnung                                  [len(tafelZeilen)]image.Rectangle
}

// schreiber ist ein gemessener, noch nicht gezeichneter Text.
type schreiber struct {
	call    op.CallOp
	groesse image.Point
}

func (s schreiber) setze(gtx layout.Context, at image.Point) {
	t := op.Offset(at).Push(gtx.Ops)
	s.call.Add(gtx.Ops)
	t.Pop()
}

// tafelText misst s (umbrechend bis breite, zeilen = 0: beliebig viele Zeilen), ohne es zu zeichnen.
func (e *editor) tafelText(gtx layout.Context, s string, sp unit.Sp, c color.NRGBA, breite, zeilen int, fett bool) schreiber {
	lbl := material.Label(e.thema, sp, s)
	lbl.Color = c
	lbl.MaxLines = zeilen
	if fett {
		lbl.Font.Weight = font.SemiBold
	}
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(max(1, breite), 1<<20)}
	m := op.Record(gtx.Ops)
	d := lbl.Layout(g)
	return schreiber{call: m.Stop(), groesse: d.Size}
}

// tafelKnopf meldet c auf r an (Zeiger, Klick); den Fokusring zeichnet die Tafel selbst.
func tafelKnopf(gtx layout.Context, c *widget.Clickable, r image.Rectangle) {
	t := op.Offset(r.Min).Push(gtx.Ops)
	g := gtx
	g.Constraints = layout.Exact(r.Size())
	c.Layout(g, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: r.Size()} })
	pointer.CursorPointer.Add(gtx.Ops)
	t.Pop()
}

// einstellungenOeffnen liest die Einstellungen frisch (ADR-0034 Punkt 4 der Karte: beim Öffnen) und öffnet die Tafel.
func (e *editor) einstellungenOeffnen(gtx layout.Context) {
	werte, warn := settings.Lade()
	pfad, pfadFehler := settings.Pfad()
	e.einst.oeffne(werte, warn, pfad, tafelSprachen())
	e.einst.pfadFehler = pfadFehler
	ag, _ := settings.AgentenPfad() // der Ort, den `schnapp mcp` liest (ADR-0034 Nachtrag 1, F-625)
	e.einst.agentenOrt(ag, settings.LadeDatei)
	e.hilfeOffen = false
	gtx.Execute(key.FocusCmd{Tag: &e.einst})
}

// einstellungenSchliessen schließt die Tafel ohne zu speichern; der Fokus geht an die Leinwand.
func (e *editor) einstellungenSchliessen(gtx layout.Context) {
	e.einst.offen = false
	if e.menue.offen && e.menue.welches == listeEinstellung {
		e.menue.offen, e.menue.fokusNach = false, nil
	}
	if e.bibl.offen { // über der vollen Ansicht der Bibliothek geöffnet: Fokus zurück an sie (ADR-0036)
		gtx.Execute(key.FocusCmd{Tag: &e.bibl})
		return
	}
	gtx.Execute(key.FocusCmd{Tag: e})
}

// schreibeEinstellungen ist der Schreiber der Tafel (Tests tauschen ihn nicht: sie setzen XDG_CONFIG_HOME und
// settings.BenutzerHeim).
var schreibeEinstellungen = settings.SchreibeDatei

// merkbar sind die Schlüssel (Abschnitt, Schlüssel), die merkeEinstellung außerhalb der Tafel schreiben darf.
var merkbar = map[[2]string]bool{{"library", "bar"}: true}

// errNichtMerkbar: merkeEinstellung bekam einen Schlüssel außerhalb von merkbar (ein Programmfehler).
var errNichtMerkbar = errors.New("setting cannot be remembered outside the settings panel")

// merkeEinstellung schreibt einen einzelnen Schlüssel mit dem Schreiber der Tafel (schreibeEinstellungen) an den Ort
// von settings.Pfad: der Menüschalter der Leiste der letzten Aufnahmen (ADR-0036 Punkt 2, ADR-0034 Nachtrag 2). Er
// steht hier, weil nur die Tafel schreibt (ADR-0034 Punkt 6, TestMCPErreichtDenSchreiberNicht). Er nimmt nur die
// Schlüssel aus merkbar an (F-658): Sonst wäre er aus jeder Datei des Pakets ein Schreiber für beliebige Schlüssel,
// auch die Einwilligungen der Agenten.
func merkeEinstellung(a settings.Aenderung) error {
	if !merkbar[[2]string{a.Abschnitt, a.Schluessel}] {
		return fmt.Errorf("%w: %s.%s", errNichtMerkbar, a.Abschnitt, a.Schluessel)
	}
	pfad, err := settings.Pfad()
	if err != nil {
		return err
	}
	return schreibeEinstellungen(pfad, []settings.Aenderung{a})
}

// einstellungenSpeichern schreibt die Änderungen; gelingt es, schließt die Tafel und die Fußzeile sagt es. Die im
// laufenden Editor schon wirksamen Werte bleiben, wie sie sind (ADR-0034 Punkt 3).
func (e *editor) einstellungenSpeichern(gtx layout.Context) {
	leisteVorher := e.einst.geladen.Leiste // speichere setzt geladen auf den geschriebenen Stand
	geschrieben, ok := e.einst.speichere(schreibeEinstellungen)
	if !ok {
		return
	}
	e.einstellungenSchliessen(gtx)
	if geschrieben {
		e.melde(i18n.T("Settings saved"), false)
	}
	// Ausnahme von ADR-0034 Punkt 3: Die Leiste der letzten Aufnahmen folgt der Tafel sofort, wie dem Menüschalter
	// (ADR-0034 Nachtrag 2, ADR-0036 Punkt 2) — aber nur, wenn die Tafel diesen Schlüssel geändert hat (F-655): Weicht
	// das Fenster von der Datei ab (Probe-Lauf startet ohne Leiste, Merken scheiterte, ein zweites Fenster schaltete),
	// kippte sonst jedes Speichern die Leiste.
	if e.unten.l != nil && e.einst.wert.Leiste != leisteVorher && e.unten.an != e.einst.wert.Leiste {
		e.unten.an = e.einst.wert.Leiste
	}
}

// einstellungenListe öffnet die Auswahlliste der Zeile z; per Tastatur steht der Fokus auf dem gewählten Eintrag.
func (e *editor) einstellungenListe(z int, perTaste bool) {
	t := &e.einst
	t.listeZeile = z
	e.menueOeffne(listeEinstellung, perTaste)
	if i := t.gewaehlt(z); perTaste && e.menue.offen && i >= 0 {
		e.menue.cursor, e.menue.fokusNach = i, &e.menue.eintraege[i]
	}
}

// einstellungsListe sind die Einträge der offenen Auswahlliste (menueEintraegeVon).
func (e *editor) einstellungsListe() []menueEintrag {
	t := &e.einst
	z := t.listeZeile
	if z < 0 || z >= len(tafelZeilen) {
		return nil
	}
	var en []menueEintrag
	g := t.gewaehlt(z)
	for j, o := range t.optionen(z) {
		en = append(en, menueEintrag{hatWahl: true, wahl: j, text: o.text, an: true, aktiv: j == g})
	}
	return en
}

// einstellungenFilter sind die Tasten der Tafel (ADR-0034 Punkt 4). Sie gelten ohne Fokusbindung, damit keine Taste
// in die Leisten darunter gerät; was hier fehlt, verwirft tasten.
func (e *editor) einstellungenFilter() []event.Filter {
	f := []event.Filter{key.FocusFilter{Target: &e.einst}}
	for _, n := range []key.Name{key.NameEscape, key.NameReturn, key.NameEnter, key.NameSpace, key.NameTab,
		key.NameUpArrow, key.NameDownArrow, key.NameDeleteBackward, "+", "-"} {
		f = append(f, key.Filter{Name: n, Optional: key.ModShift})
	}
	for _, n := range []key.Name{key.NamePageUp, key.NamePageDown} {
		f = append(f, key.Filter{Name: n, Required: key.ModShortcut},
			key.Filter{Name: n}) // ohne Umschalter: der Inhalt rollt seitenweise (F-709)
	}
	return f
}

// einstellungenAuswerten verarbeitet Klicks und Tasten der offenen Tafel; läuft im Bild vor allen anderen.
func (e *editor) einstellungenAuswerten(gtx layout.Context) {
	t := &e.einst
	if !t.offen {
		return
	}
	t.roll.radLesen(gtx) // Mausrad über dem Inhalt rollt ihn (F-709)
	for {                // Klicks und Mausrad auf die Fläche über dem Fenster: abholen, ohne Wirkung (modal)
		if _, ok := gtx.Event(pointer.Filter{Target: t, Kinds: pointer.Press | pointer.Scroll,
			ScrollX: pointer.ScrollRange{Min: -1 << 20, Max: 1 << 20},
			ScrollY: pointer.ScrollRange{Min: -1 << 20, Max: 1 << 20}}); !ok {
			break
		}
	}
	for i := range t.reiterKnopf {
		if t.reiterKnopf[i].Clicked(gtx) {
			t.waehleReiter(tafelReiter(i))
		}
	}
	for z := range t.feldKnopf {
		if !t.feldKnopf[z].Clicked(gtx) || tafelZeilen[z].reiter != t.reiter {
			continue
		}
		t.fokusAuf(z)
		switch tafelZeilen[z].art {
		case artSchalter:
			t.schalte(z)
		case artAuswahl:
			e.einstellungenListe(z, false)
		}
	}
	if t.minus.Clicked(gtx) {
		t.fokusAuf(zahlZeile())
		t.zahlSchritt(-1)
	}
	if t.plus.Clicked(gtx) {
		t.fokusAuf(zahlZeile())
		t.zahlSchritt(1)
	}
	if t.kreuz.Clicked(gtx) || t.abbrechen.Clicked(gtx) {
		e.einstellungenSchliessen(gtx)
		return
	}
	if t.ok.Clicked(gtx) {
		e.einstellungenSpeichern(gtx)
		if !t.offen {
			return
		}
	}
	if e.menue.offen {
		// Die offene Auswahlliste bedient tasten wie jedes Menü. Das Fokusziel der Tafel bleibt trotzdem angemeldet:
		// Sonst verwirft Gio es am Ende des Bilds, und der Fokus, den menueSchliesse beim Schließen der Liste an die
		// Tafel gibt, ginge ins Leere.
		for {
			if _, ok := gtx.Event(key.FocusFilter{Target: t}); !ok {
				return
			}
		}
	}
	if !gtx.Focused(t) { // nach dem Schließen einer Liste oder dem Öffnen: Fokus (und getippte Ziffern) zur Tafel
		gtx.Execute(key.FocusCmd{Tag: t})
	}
	filter := e.einstellungenFilter()
	for {
		ev, ok := gtx.Event(filter...)
		if !ok {
			return
		}
		switch ev := ev.(type) {
		case key.EditEvent:
			t.tippe(ev.Text)
		case key.Event:
			if ev.State != key.Press {
				continue
			}
			if ev.Modifiers == 0 && (ev.Name == key.NamePageUp || ev.Name == key.NamePageDown) {
				k := &t.roll
				k.rollen, _ = rollTaste(ev.Name, k.rollen, k.rollMax, gtx.Dp(tafelFeldDp), k.sicht.Dy())
				gtx.Execute(op.InvalidateCmd{})
				continue
			}
			switch t.taste(ev.Name, ev.Modifiers) {
			case folgeSchliessen:
				e.einstellungenSchliessen(gtx)
				return
			case folgeSpeichern:
				e.einstellungenSpeichern(gtx)
				if !t.offen {
					return
				}
			case folgeListe:
				e.einstellungenListe(t.listeZeile, true)
				return // die weiteren Tasten gehören der Liste (tasten)
			}
		}
	}
}

// zeichneEinstellungen zeichnet die offene Tafel über dem Editor (unter der offenen Auswahlliste).
func (e *editor) zeichneEinstellungen(gtx layout.Context) {
	t := &e.einst
	t.rahmen = tafelRahmen{}
	if !t.offen {
		return
	}
	mx := gtx.Constraints.Max
	ganz := image.Rectangle{Max: mx}
	paint.FillShape(gtx.Ops, farbeTafelDunkel, clip.Rect(ganz).Op())
	ha := clip.Rect(ganz).Push(gtx.Ops)
	event.Op(gtx.Ops, t)
	ha.Pop()

	dp := gtx.Dp
	groesse := image.Pt(max(dp(200), min(dp(tafelBreiteDp), mx.X-dp(32))), max(dp(200), min(dp(tafelHoeheDp), mx.Y-dp(32))))
	ecke := ganz.Size().Sub(groesse).Div(2)
	ecke = image.Pt(max(0, ecke.X), max(0, ecke.Y))
	W, H := groesse.X, groesse.Y
	kopf, fuss := dp(tafelKopfDp), dp(tafelFussDp)
	links := min(dp(tafelLinksDp), W/3)
	off := op.Offset(ecke).Push(gtx.Ops)
	rand := clip.Rect(image.Rectangle{Max: groesse}).Push(gtx.Ops)
	fenster := func(r image.Rectangle) image.Rectangle { return r.Add(ecke) }
	t.rahmen.tafel = fenster(image.Rectangle{Max: groesse})
	ring := func(r image.Rectangle) { // Tastaturfokus: 2 px orange außen (corporate-design.md §6, wie die rechte Leiste)
		paint.FillShape(gtx.Ops, farbeAkzent, clip.Stroke{Path: clip.Rect(r.Inset(-dp(3))).Path(), Width: float32(dp(2))}.Op())
	}
	ziel := t.ziel()

	// Grund, Linien
	paint.FillShape(gtx.Ops, farbeLeisteVoll, clip.Rect(image.Rectangle{Max: groesse}).Op())
	inhalt := image.Rect(links, kopf, W, H-fuss)
	paint.FillShape(gtx.Ops, farbeFenster, clip.Rect(inhalt).Op())
	linie := func(r image.Rectangle) { paint.FillShape(gtx.Ops, farbeLinie, clip.Rect(r).Op()) }
	linie(image.Rect(0, kopf-dp(1), W, kopf))
	linie(image.Rect(0, H-fuss, W, H-fuss+dp(1)))
	linie(image.Rect(links-dp(1), kopf, links, H-fuss))
	paint.FillShape(gtx.Ops, farbeLinie, clip.Stroke{Path: clip.Rect(image.Rectangle{Max: groesse}).Path(), Width: float32(dp(1))}.Op())

	// Kopf: Titel und Kreuz
	ti := e.tafelText(gtx, i18n.T("Settings"), unit.Sp(17), farbeText, W-dp(80), 1, true)
	ti.setze(gtx, image.Pt(dp(20), (kopf-ti.groesse.Y)/2))
	kr := image.Rect(W-dp(16)-dp(32), (kopf-dp(32))/2, W-dp(16), (kopf-dp(32))/2+dp(32))
	if t.kreuz.Hovered() {
		paint.FillShape(gtx.Ops, farbeGrund, clip.Rect(kr).Op())
	}
	zeichneKreuz(gtx, kr, dp(12), farbeText)
	tafelKnopf(gtx, &t.kreuz, kr)
	t.rahmen.kreuz = fenster(kr)

	// Reiter
	for i := range reiterAnzahl {
		r := image.Rect(dp(10), kopf+dp(14)+int(i)*dp(38), links-dp(10), kopf+dp(14)+int(i)*dp(38)+dp(32))
		switch {
		case i == t.reiter:
			paint.FillShape(gtx.Ops, farbeAuswahl, clip.Rect(r).Op())
		case t.reiterKnopf[i].Hovered():
			paint.FillShape(gtx.Ops, farbeGrund, clip.Rect(r).Op())
		}
		c := farbeText
		if i == t.reiter {
			c = farbeWeiss
		}
		s := e.tafelText(gtx, reiterName(i), unit.Sp(14), c, r.Dx()-dp(24), 1, false)
		s.setze(gtx, image.Pt(r.Min.X+dp(12), r.Min.Y+(r.Dy()-s.groesse.Y)/2))
		tafelKnopf(gtx, &t.reiterKnopf[i], r)
		t.rahmen.reiter[i] = fenster(r)
	}

	// Zeilen des Reiters
	ic := clip.Rect(inhalt).Push(gtx.Ops)
	rk := &t.roll
	rk.anmelden(inhalt, gtx)
	rk.rollen = rollBegrenze(rk.rollen, rk.rollMax)
	vers := op.Offset(image.Pt(0, -rk.rollen)).Push(gtx.Ops) // der Inhalt rollt (F-709), Lagen in Inhaltskoordinaten
	fensterR := func(r image.Rectangle) image.Rectangle { return fenster(r.Sub(image.Pt(0, rk.rollen))) }
	x0 := links + dp(20)
	feldX := x0 + min(dp(240), (W-x0)/3)
	feldB := max(dp(60), min(dp(300), W-feldX-dp(24)))
	breite := W - x0 - dp(24)
	y := kopf + dp(20)
	k := e.tafelText(gtx, reiterKopf(t.reiter), unit.Sp(11), farbeNebentext, breite, 1, true)
	k.setze(gtx, image.Pt(x0, y))
	y += k.groesse.Y + dp(10)
	if s := reiterEinleitung(t.reiter); s != "" {
		ei := e.tafelText(gtx, s, unit.Sp(12), farbeNebentext, min(breite, dp(440)), 0, false)
		ei.setze(gtx, image.Pt(x0, y))
		y += ei.groesse.Y + dp(12)
	}
	if s := t.agentenHinweis(); s != "" && t.reiter == reiterAgenten {
		// Der Server liest eine andere Datei als die übrigen Reiter (oder keine): Kasten wie die Warnungen (F-625).
		wb := min(breite, dp(560))
		ht := e.tafelText(gtx, s, unit.Sp(12), farbeText, wb-dp(30), 0, false)
		hr := image.Rect(x0, y, x0+wb, y+ht.groesse.Y+dp(20))
		paint.FillShape(gtx.Ops, farbeWarnGrund, clip.Rect(hr).Op())
		paint.FillShape(gtx.Ops, farbeAkzent, clip.Rect(image.Rect(hr.Min.X, hr.Min.Y, hr.Min.X+dp(3), hr.Max.Y)).Op())
		ht.setze(gtx, image.Pt(hr.Min.X+dp(14), hr.Min.Y+dp(10)))
		t.rahmen.hinweis = fensterR(hr)
		y = hr.Max.Y + dp(12)
	}
	for _, z := range zeilenDes(t.reiter) {
		fh := dp(tafelFeldDp)
		ti := e.tafelText(gtx, zeileTitel(z), unit.Sp(14), farbeText, feldX-x0-dp(12), 1, false)
		ti.setze(gtx, image.Pt(x0, y+(fh-ti.groesse.Y)/2))
		r := image.Rect(feldX, y, feldX+feldB, y+fh)
		switch tafelZeilen[z].art {
		case artAuswahl:
			paint.FillShape(gtx.Ops, farbeGrund, clip.Rect(r).Op())
			paint.FillShape(gtx.Ops, farbeLinie, clip.Stroke{Path: clip.Rect(r).Path(), Width: float32(dp(1))}.Op())
			s := e.tafelText(gtx, t.optionText(z), unit.Sp(14), farbeText, r.Dx()-dp(40), 1, false)
			s.setze(gtx, image.Pt(r.Min.X+dp(10), r.Min.Y+(fh-s.groesse.Y)/2))
			zeichneAufklappPfeil(gtx, image.Pt(r.Max.X-dp(15), r.Min.Y+fh/2), farbeText)
		case artSchalter:
			q := dp(20)
			kr := image.Rect(feldX, y+(fh-q)/2, feldX+q, y+(fh-q)/2+q)
			s := e.tafelText(gtx, zeileSchalterText(z), unit.Sp(14), farbeText, feldB-q-dp(10), 1, false)
			s.setze(gtx, image.Pt(kr.Max.X+dp(10), y+(fh-s.groesse.Y)/2))
			zeichneKaestchen(gtx, kr, t.schalter(z))
			r.Max.X = min(r.Max.X, kr.Max.X+dp(10)+s.groesse.X+dp(4))
			if ziel == z {
				ring(kr)
				rk.fokus = kr
			}
		case artZahl:
			kn := dp(tafelFeldDp)
			mr := image.Rect(feldX, y, feldX+kn, y+fh)
			wr := image.Rect(mr.Max.X+dp(8), y, mr.Max.X+dp(8)+dp(70), y+fh)
			pr := image.Rect(wr.Max.X+dp(8), y, wr.Max.X+dp(8)+kn, y+fh)
			for _, kk := range []struct {
				c *widget.Clickable
				r image.Rectangle
				s string
			}{{&t.minus, mr, "−"}, {&t.plus, pr, "+"}} {
				hg := farbeLinie
				if kk.c.Hovered() {
					hg = farbeGrund
				}
				paint.FillShape(gtx.Ops, hg, clip.Rect(kk.r).Op())
				s := e.tafelText(gtx, kk.s, unit.Sp(14), farbeText, kk.r.Dx(), 1, false)
				s.setze(gtx, kk.r.Min.Add(kk.r.Size().Sub(s.groesse).Div(2)))
				tafelKnopf(gtx, kk.c, kk.r)
			}
			t.rahmen.minus, t.rahmen.plus = fensterR(mr), fensterR(pr)
			paint.FillShape(gtx.Ops, farbeGrund, clip.Rect(wr).Op())
			rc := farbeLinie
			if t.zahlFehler {
				rc = farbeRot
			}
			paint.FillShape(gtx.Ops, rc, clip.Stroke{Path: clip.Rect(wr).Path(), Width: float32(dp(1))}.Op())
			s := e.tafelText(gtx, t.zahlAnzeige(), unit.Sp(14), farbeText, wr.Dx()-dp(16), 1, false)
			s.setze(gtx, image.Pt(wr.Min.X+dp(10), y+(fh-s.groesse.Y)/2))
			r = wr
		}
		if ziel == z && tafelZeilen[z].art != artSchalter {
			ring(r)
			rk.fokus = r
		}
		tafelKnopf(gtx, &t.feldKnopf[z], r)
		t.rahmen.feld[z] = fensterR(r)
		y += fh
		if s := zeileErklaerung(z); s != "" {
			er := e.tafelText(gtx, s, unit.Sp(12), farbeNebentext, max(feldB, W-feldX-dp(24)), 0, false)
			er.setze(gtx, image.Pt(feldX, y+dp(4)))
			y += dp(4) + er.groesse.Y
		}
		if t.warnungSichtbar(z) {
			wb := W - x0 - dp(24)
			wt := e.tafelText(gtx, zeileWarnung(z), unit.Sp(12), farbeText, wb-dp(30), 0, false)
			wr := image.Rect(x0, y+dp(8), x0+wb, y+dp(8)+wt.groesse.Y+dp(20))
			paint.FillShape(gtx.Ops, farbeWarnGrund, clip.Rect(wr).Op())
			paint.FillShape(gtx.Ops, farbeAkzent, clip.Rect(image.Rect(wr.Min.X, wr.Min.Y, wr.Min.X+dp(3), wr.Max.Y)).Op())
			wt.setze(gtx, image.Pt(wr.Min.X+dp(14), wr.Min.Y+dp(10)))
			t.rahmen.warnung[z] = fensterR(wr)
			y = wr.Max.Y
		}
		y += dp(10)
	}
	vers.Pop()
	rk.abschluss(gtx, y, dp(8)) // y ist das Ende in Inhaltskoordinaten
	rk.balken(gtx, inhalt)
	ic.Pop()

	// Fuß: Wirkung bzw. Fehler links, Abbrechen und Speichern rechts
	fy := H - fuss
	ok := image.Rect(W-dp(20)-dp(108), fy+(fuss-dp(32))/2, W-dp(20), fy+(fuss-dp(32))/2+dp(32))
	ab := image.Rect(ok.Min.X-dp(12)-dp(110), ok.Min.Y, ok.Min.X-dp(12), ok.Max.Y)
	for _, kk := range []struct {
		c      *widget.Clickable
		r      image.Rectangle
		s      string
		grund  color.NRGBA
		zielNr int
	}{{&t.abbrechen, ab, i18n.T("Cancel"), farbeLinie, zielAbbrechen}, {&t.ok, ok, i18n.T("Save"), farbeAuswahl, zielSpeichern}} {
		g := kk.grund
		if kk.c.Hovered() {
			g = farbeGrund
			if kk.zielNr == zielSpeichern {
				g = farbeGruenText
			}
		}
		paint.FillShape(gtx.Ops, g, clip.Rect(kk.r).Op())
		c := farbeText
		if kk.zielNr == zielSpeichern {
			c = farbeWeiss
			if kk.c.Hovered() {
				c = farbeGrund
			}
		}
		s := e.tafelText(gtx, kk.s, unit.Sp(14), c, kk.r.Dx(), 1, false)
		s.setze(gtx, kk.r.Min.Add(kk.r.Size().Sub(s.groesse).Div(2)))
		if ziel == kk.zielNr {
			ring(kk.r)
		}
		tafelKnopf(gtx, kk.c, kk.r)
	}
	t.rahmen.abbrechen, t.rahmen.ok = fenster(ab), fenster(ok)
	tb := ab.Min.X - dp(40)
	t.gezeigteFehler = ""
	if t.fehler != "" {
		s := e.tafelText(gtx, t.fehler, unit.Sp(12), farbeRot, tb, 2, false)
		s.setze(gtx, image.Pt(dp(20), fy+(fuss-s.groesse.Y)/2))
		t.gezeigteFehler = t.fehler
	} else {
		h := e.tafelText(gtx, reiterWirkung(t.reiter), unit.Sp(12), farbeNebentext, tb, 1, false)
		if w := t.fussWarnung(); w != "" {
			s := e.tafelText(gtx, w, unit.Sp(12), farbeAkzent, tb, 1, false)
			h.setze(gtx, image.Pt(dp(20), fy+fuss/2-h.groesse.Y-dp(1)))
			s.setze(gtx, image.Pt(dp(20), fy+fuss/2+dp(1)))
		} else {
			h.setze(gtx, image.Pt(dp(20), fy+(fuss-h.groesse.Y)/2))
		}
	}
	rand.Pop()
	off.Pop()
}

// zeichneKaestchen zeichnet einen eckigen Schalter: leer mit grauem Rand, an grün mit weißem Haken.
func zeichneKaestchen(gtx layout.Context, r image.Rectangle, an bool) {
	if !an {
		paint.FillShape(gtx.Ops, farbeNebentext, clip.Stroke{Path: clip.Rect(r.Inset(gtx.Dp(1))).Path(),
			Width: float32(gtx.Dp(1)) * 1.5}.Op())
		return
	}
	paint.FillShape(gtx.Ops, farbeAuswahl, clip.Rect(r).Op())
	w, h := float32(r.Dx()), float32(r.Dy())
	o := f32.Pt(float32(r.Min.X), float32(r.Min.Y))
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(o.Add(f32.Pt(w*0.22, h*0.52)))
	p.LineTo(o.Add(f32.Pt(w*0.42, h*0.72)))
	p.LineTo(o.Add(f32.Pt(w*0.78, h*0.30)))
	paint.FillShape(gtx.Ops, farbeWeiss, clip.Stroke{Path: p.End(), Width: float32(gtx.Dp(2))}.Op())
}

// zeichneKreuz zeichnet ein Kreuz der Kantenlänge a mittig in r.
func zeichneKreuz(gtx layout.Context, r image.Rectangle, a int, c color.NRGBA) {
	m := f32.Pt(float32(r.Min.X+r.Max.X)/2, float32(r.Min.Y+r.Max.Y)/2)
	h := float32(a) / 2
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(m.Add(f32.Pt(-h, -h)))
	p.LineTo(m.Add(f32.Pt(h, h)))
	p.MoveTo(m.Add(f32.Pt(h, -h)))
	p.LineTo(m.Add(f32.Pt(-h, h)))
	paint.FillShape(gtx.Ops, c, clip.Stroke{Path: p.End(), Width: float32(gtx.Dp(1)) * 1.5}.Op())
}
