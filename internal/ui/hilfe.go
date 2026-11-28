// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Übersicht der Tastenkürzel (Menü Hilfe, F1; ADR-0031 Punkt 1). Sie entsteht aus denselben Quellen wie Menü und
// Werkzeugleiste (Tastentabelle, Menüeinträge), damit sie nie von ihnen abweicht.

import (
	"image"
	"strings"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/Andiministrator/schnapp/internal/i18n"
)

// Version ist die Version des Programms für „Über Schnapp“; cmd/schnapp setzt sie beim Start.
var Version = "0.0.0-dev"

// hilfeZeile ist eine Zeile der Übersicht: Handlung und Taste; ueberschrift: Menütitel ohne Taste.
type hilfeZeile struct {
	text, taste  string
	ueberschrift bool
}

// hilfeZeilen sammelt alle Kürzel: je Menü die Einträge mit Taste, die Werkzeuge mit Buchstaben, dazu die Tasten ohne
// Menüeintrag (Esc, F10).
func (e *editor) hilfeZeilen() []hilfeZeile {
	var z []hilfeZeile
	for i := 0; i < menueAnzahl; i++ {
		var teil []hilfeZeile
		for _, en := range e.menueEintraegeVon(i) {
			if _, zweite := hilfePaarZweite[en.akt]; zweite {
				continue // steht mit der ersten auf einer Zeile
			}
			k := menueKuerzel(en)
			if !en.hatWerkzeug && en.akt != aktTextBearbeiten {
				k = strings.Join(alleKuerzel(en.akt), ", ") // auch die zweite Taste (Strg+Y, Rücktaste)
			}
			if p, ok := hilfePaare[en.akt]; ok && k != "" {
				teil = append(teil, hilfeZeile{text: p.text(), taste: paarTaste(k, strings.Join(alleKuerzel(p.zweite), ", "))})
				continue
			}
			if k != "" {
				teil = append(teil, hilfeZeile{text: en.text, taste: k})
			}
		}
		if len(teil) > 0 {
			z = append(z, hilfeZeile{text: menueTitel(i), ueberschrift: true})
			z = append(z, teil...)
		}
	}
	return append(z,
		hilfeZeile{text: i18n.T("Other keys"), ueberschrift: true},
		hilfeZeile{text: i18n.T("Menu bar"), taste: "F10"},
		hilfeZeile{text: i18n.T("Context menu"), taste: i18n.T("Shift") + "+F10"},
		hilfeZeile{text: i18n.T("Cancel, deselect, Select tool"), taste: "Esc"},
		hilfeZeile{text: i18n.T("New line in a text"), taste: i18n.T("Shift") + "+" + i18n.T("Enter")},
		hilfeZeile{text: i18n.T("Snapping off while dragging"), taste: "Alt"},
		hilfeZeile{text: i18n.T("Move the view in the navigator"), taste: i18n.T("Click, drag")},
		// Mausrad auf der Leinwand (ADR-0040): rollt, mit Umschalt waagerecht, mit Strg zoomt es. Nur zwei Zeilen: mit einer
		// eigenen Zeile für Umschalt oder den Navigator (Rad zoomt dort, steht in der README) rollte die Übersicht im
		// Startfenster (F-705).
		hilfeZeile{text: i18n.T("Scroll the view (Shift: sideways)"), taste: i18n.T("Wheel")},
		hilfeZeile{text: i18n.T("Zoom at the pointer"), taste: i18n.T("Ctrl") + "+" + i18n.T("Wheel")},
		// Mehrfachauswahl auf der Leinwand und Ansicht verschieben (ADR-0039 mit Nachtrag 1)
		hilfeZeile{text: i18n.T("Add or remove an object"), taste: i18n.T("Shift") + "+" + i18n.T("Click")},
		hilfeZeile{text: i18n.T("Select several objects"), taste: i18n.T("Drag in the image")},
		hilfeZeile{text: i18n.T("Move the view"), taste: i18n.T("Space") + "+" + i18n.T("Drag")},
		// Mehrere Vorschauen der Bibliothek markieren (ADR-0036 Zusatz Karte #361)
		hilfeZeile{text: i18n.T("Select several captures"), taste: i18n.T("Ctrl/Shift+Click")},
		// … und per Tastatur an einer Vorschau mit Fokus (F-891); je Taste eine Zeile, eine passte nicht in die Spalte.
		hilfeZeile{text: i18n.T("Select a capture"), taste: i18n.T("Ctrl") + "+" + i18n.T("Space")},
		hilfeZeile{text: i18n.T("Extend the selection"), taste: i18n.T("Shift") + "+" + i18n.T("Arrow")},
		hilfeZeile{text: i18n.T("Select all captures"), taste: i18n.T("Ctrl") + "+A"},
	)
}

// hilfePaar legt zwei Gegenstücke auf eine Zeile der Übersicht (F-705: sonst rollte sie im Startfenster, seit den
// Gruppen, ADR-0044, zwei Zeilen mehr).
type hilfePaar struct {
	zweite aktion
	text   func() string
}

var hilfePaare = map[aktion]hilfePaar{
	aktEbeneVor:      {aktEbeneZurueck, func() string { return i18n.T("Bring forward / send backward") }},
	aktEbeneDarueber: {aktEbeneDarunter, func() string { return i18n.T("Select layer above / below") }},
}

// hilfePaarZweite sind die zweiten Handlungen der Paare.
var hilfePaarZweite = func() map[aktion]bool {
	m := map[aktion]bool{}
	for _, p := range hilfePaare {
		m[p.zweite] = true
	}
	return m
}()

// paarTaste fasst zwei Kürzel zusammen: gleicher Umschalter → „Strg+Bild↑/Bild↓“, sonst „a / b“.
func paarTaste(a, b string) string {
	ia, ib := strings.LastIndex(a, "+"), strings.LastIndex(b, "+")
	if ia >= 0 && ib >= 0 && a[:ia] == b[:ib] {
		return a + "/" + b[ib+1:]
	}
	return a + " / " + b
}

// Maße der Übersicht in dp (F-705: so, dass alle Zeilen im Startfenster ohne Rollen in zwei Spalten stehen).
const (
	hilfeZeileDp = 20
	hilfeTextDp  = 220
	hilfeTasteDp = 170
)

// hilfeAufteilung teilt n Zeilen auf Spalten: zwei nur, wenn eine nicht in die Höhe passt und zwei in die Breite;
// schmale Fenster verkürzen die Textspalte (nicht unter 120/250 der gegebenen Breite). Alle Maße in Pixeln. Passt auch
// das nicht in die Höhe, rollt die Liste.
func hilfeAufteilung(n int, flaeche image.Point, hoehe, innen, kopf, spalteText, spalteTaste int) (spalten, pro, text int) {
	pro, spalten = n, 1
	if kopf+innen+pro*hoehe > flaeche.Y && innen*2+2*(spalteText+spalteTaste+innen) <= flaeche.X {
		spalten, pro = 2, (n+1)/2
	}
	text = spalteText
	if frei := flaeche.X - innen*2 - spalten*(spalteTaste+innen); text*spalten > frei {
		text = max(spalteText*120/hilfeTextDp, frei/spalten)
	}
	return spalten, pro, text
}

// zeichneHilfe legt die Übersicht über die Leinwand; ein Klick irgendwohin im Fenster oder Esc schließt sie.
func (e *editor) zeichneHilfe(gtx layout.Context) {
	if !e.hilfeOffen {
		e.hilfeRoll.rollen, e.hilfeRoll.rest = 0, 0
		return
	}
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &e.hilfeOffen, Kinds: pointer.Press})
		if !ok {
			break
		}
		if _, ok := ev.(pointer.Event); ok {
			e.hilfeOffen = false
			return
		}
	}
	flaeche := e.leinwandFlaeche(gtx)
	// Modal wie die Einstellungstafel (ADR-0031 Nachtrag 11, F-707): Das Zeigerziel deckt das ganze Fenster, Menüzeile,
	// Werkzeugleiste, Leisten und Fußfelder bekommen keinen Klick, ein Klick irgendwo schließt nur die Tafel.
	ha := clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops)
	event.Op(gtx.Ops, &e.hilfeOffen)
	ha.Pop()

	zeilen := e.hilfeZeilen()
	hoehe, innen := gtx.Dp(hilfeZeileDp), gtx.Dp(12)
	kopf := innen + gtx.Dp(14) // Oberkante der Liste in der Tafel; darüber steht die feste Überschrift
	spalten, pro, spalteText := hilfeAufteilung(len(zeilen), flaeche.Size(), hoehe, innen, kopf, gtx.Dp(hilfeTextDp), gtx.Dp(hilfeTasteDp))
	spalteTaste := gtx.Dp(hilfeTasteDp)
	spalteB := spalteText + spalteTaste + innen
	groesse := image.Pt(min(flaeche.Dx(), innen*2+spalten*spalteB), min(flaeche.Dy(), kopf+innen+pro*hoehe))
	e.hilfeGroesse = groesse
	ecke := flaeche.Min.Add(flaeche.Size().Sub(groesse).Div(2))
	t := op.Offset(ecke).Push(gtx.Ops)
	r := image.Rectangle{Max: groesse}
	rand := clip.Rect(r).Push(gtx.Ops) // nichts läuft über die Tafel hinaus
	paint.FillShape(gtx.Ops, farbeLeisteVoll, clip.Rect(r).Op())
	paint.FillShape(gtx.Ops, farbeLinie, clip.Stroke{Path: clip.Rect(r).Path(), Width: float32(gtx.Dp(1))}.Op())
	schreibe := func(s string, x, y, breite int, sp unit.Sp, c, fett bool) {
		lbl := material.Label(e.thema, sp, s)
		lbl.Color = farbeText
		if c {
			lbl.Color = farbeNebentext
		}
		if fett {
			lbl.Color = farbeGruenText
		}
		lbl.MaxLines = 1
		g := gtx
		g.Constraints.Min = image.Point{}
		g.Constraints.Max.X = breite
		m := op.Record(gtx.Ops)
		lbl.Layout(g)
		call := m.Stop()
		tt := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		tt.Pop()
	}
	// Die Liste rollt unter der festen Überschrift (F-705): Passt sie nicht in die Tafel, rollt sie mit Mausrad und
	// Tasten; die Rechnung steht in seitenrollen.go/tafelrollen.go.
	k := &e.hilfeRoll
	liste := image.Rect(0, kopf, groesse.X, groesse.Y)
	lc := clip.Rect(liste).Push(gtx.Ops)
	k.anmelden(liste, gtx)
	k.radLesen(gtx)
	// Das Rad-Ziel deckt die Liste und nimmt damit auch Klicks: ein Klick auf die Liste schließt die Übersicht wie
	// überall sonst (Kritik B-2, Regression aus F-705; ADR-0031 Nachtrag 11).
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &k.flaeche, Kinds: pointer.Press})
		if !ok {
			break
		}
		if pe, ok := ev.(pointer.Event); ok && pe.Kind == pointer.Press {
			e.hilfeOffen = false
		}
	}
	k.rollen = rollBegrenze(k.rollen, k.rollMax)
	for i, z := range zeilen {
		sp, zeile := i/pro, i%pro
		x := innen + sp*spalteB
		y := kopf + zeile*hoehe - k.rollen
		if y+hoehe < liste.Min.Y || y > liste.Max.Y {
			continue
		}
		if z.ueberschrift {
			schreibe(z.text, x, y, spalteText-gtx.Dp(12), unit.Sp(12), false, true)
			continue
		}
		schreibe(z.text, x+gtx.Dp(8), y, spalteText-gtx.Dp(20), unit.Sp(12), false, false)
		schreibe(z.taste, x+spalteText, y, spalteTaste, unit.Sp(12), true, false)
	}
	k.abschluss(gtx, kopf+pro*hoehe+innen/2, 0)
	k.balken(gtx, liste)
	lc.Pop()
	schreibe(i18n.T("Keyboard shortcuts")+" · "+i18n.T("click or Esc to close"), innen, innen/2, groesse.X-2*innen, unit.Sp(12), true, false)
	rand.Pop()
	t.Pop()
}
