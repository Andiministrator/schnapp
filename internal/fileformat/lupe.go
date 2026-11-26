// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/png"
	"math"
	"strconv"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/errprefix"
)

// lupe schreibt eine Lupe (ADR-0033 Punkte 4 bis 6, docs/format.md 8.7): die Daten s:quelle, s:ziel, s:strich und die
// Darstellung von unten nach oben — Verbindungslinie, Rahmen um die Quelle, Bild im Ziel, Rahmen um das Ziel. In der
// Arbeitsdatei (rundlauf) steht kein Bild: Die Pixel liegen im Original (snAO), eine Kopie trüge nach einer späteren
// Schwärzung den Klartext weiter. Der Export bettet die sichtbaren Pixel als PNG ein, vergrößert ohne Glätten.
func (s *schreiber) lupe(d *document.Dokument, o document.Lupe) {
	s.WriteString(`<g`)
	s.attr("s:typ", "lupe")
	s.ebene(o.Ebenenwerte())
	s.attr("s:quelle", punkt(o.Quelle.Min)+" "+punkt(o.Quelle.Max))
	s.attr("s:ziel", punkt(o.Ziel.Min)+" "+punkt(o.Ziel.Max))
	s.attr("s:strich", strich(o.Strich))
	s.strichart(o.Strich)
	s.WriteString(`>`)
	strichDa := o.Strich.Breite > 0 && o.Strich.Farbe.A > 0
	rahmen := func(r document.Rechteck) { // gemustert auch Punkte (ADR-0021 Nachtrag 25); in der Arbeitsdatei voll (F-841)
		st := o.Strich
		if s.rundlauf {
			st.Art = document.StrichVoll
		}
		s.rahmenSVG(r, st)
	}
	if strichDa {
		if a, b, ok := o.Linie(); ok && o.Strich.Art != document.StrichVoll && !s.rundlauf {
			s.musterSVG(document.StrichAbschnitte([]document.Punkt{a, b}, o.Strich.Art, o.Strich.Breite), o.Strich.Breite,
				o.Strich.Farbe)
		} else if ok {
			s.WriteString(`<path`)
			s.attr("d", "M"+punkt(a)+" L"+punkt(b))
			s.attr("fill", "none")
			s.farbeSVG("stroke", o.Strich.Farbe)
			s.attr("stroke-width", zahl(o.Strich.Breite))
			s.attr("stroke-linecap", "round")
			s.strichNaeherung(o.Strich, true)
			s.WriteString(`/>`)
		}
		rahmen(o.Quelle)
	}
	if !s.rundlauf {
		s.lupenBild(d, o)
	}
	if strichDa {
		rahmen(o.Ziel)
	}
	s.WriteString(`</g>`)
}

// LupenExportFaktor begrenzt, was der SVG-Export für die Lupen einbettet (ADR-0033 Nachtrag 5, Sitzungs-QA QS-2): Die
// Summe der Pixel aller eingebetteten Lupenbilder (jedes gleiche Bild einmal) darf höchstens LupenExportFaktor-mal die
// Fläche der Ansicht sein. Ein Bild einer Lupe ist nie größer als die Ansicht, mindestens so viele Lupen mit
// verschiedenen Quellen passen also immer.
const LupenExportFaktor = 4

// ErrLupenZuGross: Die Lupen eines Dokuments betteten im SVG-Export mehr als LupenExportFaktor × Ansichtsfläche Pixel
// ein. Der Export lehnt vor dem Kodieren ab; mit --ignore-limit schreibt ExportiereSVGOhneGrenze trotzdem.
var ErrLupenZuGross = errors.New("fileformat: die Lupen betten für den SVG-Export zu viele Pixel ein")

// lupenEinbettung ist der Plan für das Bild im Ziel einer Lupe, ohne ein Pixel zu lesen: welches Bild (Ausschnitt der
// Ansicht oder fertiges Zielbild), seine Grenzen und die viewBox. Gleiche Schlüssel tragen gleiche Pixel.
type lupenEinbettung struct {
	box        document.Rechteck
	rect       image.Rectangle
	schluessel lupenSchluessel
}

// lupenSchluessel macht gleiche eingebettete Bilder erkennbar. Der Ansichtsausschnitt hängt nur von seinen Grenzen ab
// (document.LupenAusschnitt); das Zielbild von Quelle in der Ansicht, Ziel und Grenzen.
type lupenSchluessel struct {
	zielbild bool
	rect     image.Rectangle
	qa, ziel document.Rechteck
}

// planeLupenBild entscheidet, was der Export für das Ziel von o einbettet (ADR-0033 Punkt 6, Nachtrag 2 Punkte 1 und 3):
// nur, was das Ziel braucht, und davon das Kleinere — die Ansichtspixel unter der Quelle (viewBox = Quelle in der
// Ansicht) oder, bei weniger Pixeln (Vergrößerung kleiner 1), das fertige Bild auf dem Pixelraster des Ziels (viewBox =
// Ziel). ok = false, wenn nichts einzubetten ist: Ziel in keiner Kachel der Ansicht, Quelle ganz in einem Streifen oder
// leeres Bild.
func planeLupenBild(d *document.Dokument, o document.Lupe) (lupenEinbettung, bool) {
	if !d.Sichtbar(o.Ziel) {
		return lupenEinbettung{}, false
	}
	qa, px := d.LupenAusschnitt(o)
	if !(qa.Breite() > 0 && qa.Hoehe() > 0) {
		return lupenEinbettung{}, false // Quelle ganz in einem Streifen: das Ziel bleibt leer
	}
	e := lupenEinbettung{box: qa, rect: px, schluessel: lupenSchluessel{rect: px}}
	z := o.Ziel
	zr := image.Rect(int(math.Floor(z.Min.X)), int(math.Floor(z.Min.Y)), int(math.Ceil(z.Max.X)), int(math.Ceil(z.Max.Y))).
		Intersect(d.Zuschnitt())
	if flaeche(zr) < flaeche(px) {
		e = lupenEinbettung{box: z, rect: zr, schluessel: lupenSchluessel{zielbild: true, rect: zr, qa: qa, ziel: z}}
	}
	if e.rect.Empty() {
		return lupenEinbettung{}, false
	}
	return e, true
}

// LupenExportPixel ist die Summe der Pixel, die der SVG-Export für die Bilder der Lupen von d einbettet, jedes gleiche
// Bild einmal; gerechnet ohne ein Pixel zu lesen (ADR-0033 Nachtrag 5).
func LupenExportPixel(d *document.Dokument) int64 {
	gesehen := map[lupenSchluessel]bool{}
	var n int64
	for _, o := range d.ObjekteOhneZug() {
		l, ok := o.(document.Lupe)
		if !ok || !exportiert(d, o) {
			continue
		}
		e, ok := planeLupenBild(d, l)
		if !ok || gesehen[e.schluessel] {
			continue
		}
		gesehen[e.schluessel] = true
		n += int64(flaeche(e.rect))
	}
	return n
}

// PruefeSVGExport lehnt ein Dokument ab, dessen Lupen im SVG-Export mehr als LupenExportFaktor × Ansichtsfläche Pixel
// einbetteten (ErrLupenZuGross). Billig: kein Pixel wird gelesen, nichts kodiert. Gilt nur für SVG; PNG und JPG betten
// nichts ein (ihre Kosten begrenzt render.Pruefe).
func PruefeSVGExport(d *document.Dokument) (fehler error) {
	defer errprefix.Defer("fileformat", &fehler)
	g := d.Groesse()
	if LupenExportPixel(d) > LupenExportFaktor*int64(g.X)*int64(g.Y) {
		return ErrLupenZuGross
	}
	return nil
}

// lupenBild schreibt das Bild im Ziel für den Export (planeLupenBild): ein <svg> in der Lage des Ziels, das auf das Ziel
// beschneidet, darin ein PNG mit image-rendering pixelated. document.LupenQuelle rechnet die Pixel, so zeigt jedes
// Zielpixel dasselbe wie im Renderer; Unsichtbares (vor dem Zuschnitt, im Spalt einer Naht) ist durchsichtig. Ein Bild,
// das schon eine frühere Lupe eingebettet hat (gleicher Schlüssel), steht nur einmal in der Datei und wird mit <use>
// wiederholt (ADR-0033 Nachtrag 5): Gleiche Lupen kosten dann kein Kodieren und kaum Platz.
func (s *schreiber) lupenBild(d *document.Dokument, o document.Lupe) {
	e, ok := planeLupenBild(d, o)
	if !ok {
		return
	}
	id, da := s.lupenBilder[e.schluessel]
	var kodiert bytes.Buffer
	if !da {
		q := d.LupenQuelle(o)
		var px *image.NRGBA
		if e.schluessel.zielbild {
			px = q.Zielbild(e.rect)
		} else {
			px = q.AnsichtsAusschnitt()
		}
		if err := png.Encode(&kodiert, px); err != nil {
			return // ein NRGBA im Speicher kodiert immer; ohne Bild bleibt das Ziel leer
		}
	}
	z := o.Ziel
	s.WriteString(`<svg`)
	s.attr("x", zahl(z.Min.X))
	s.attr("y", zahl(z.Min.Y))
	s.attr("width", zahl(z.Breite()))
	s.attr("height", zahl(z.Hoehe()))
	s.attr("viewBox", zahl(e.box.Min.X)+" "+zahl(e.box.Min.Y)+" "+zahl(e.box.Breite())+" "+zahl(e.box.Hoehe()))
	s.attr("preserveAspectRatio", "none")
	if da {
		s.WriteString(`><use`)
		s.attr("href", "#"+id)
		s.WriteString(`/></svg>`)
		return
	}
	if s.lupenBilder == nil {
		s.lupenBilder = map[lupenSchluessel]string{}
	}
	id = "lupenbild-" + strconv.Itoa(len(s.lupenBilder)+1)
	s.lupenBilder[e.schluessel] = id
	s.WriteString(`><image`)
	s.attr("x", zahl(float64(e.rect.Min.X)))
	s.attr("y", zahl(float64(e.rect.Min.Y)))
	s.attr("width", zahl(float64(e.rect.Dx())))
	s.attr("height", zahl(float64(e.rect.Dy())))
	s.attr("preserveAspectRatio", "none")
	s.attr("style", "image-rendering:pixelated")
	s.attr("href", "data:image/png;base64,"+base64.StdEncoding.EncodeToString(kodiert.Bytes()))
	s.attr("id", id)
	s.WriteString(`/></svg>`)
}

// flaeche ist die Pixelzahl von r.
func flaeche(r image.Rectangle) int { return r.Dx() * r.Dy() }
