// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/errprefix"
)

// ExportierePNG schreibt das sichtbare Bild als gewöhnliches PNG ohne Bearbeitungsdaten (zum Weitergeben).
func ExportierePNG(w io.Writer, sichtbar image.Image) (fehler error) {
	defer errprefix.Defer("fileformat", &fehler)
	return png.Encode(w, sichtbar)
}

// ExportiereJPG schreibt das sichtbare Bild als JPG (Qualität 1–100). Transparenz wird auf Weiß gelegt.
func ExportiereJPG(w io.Writer, sichtbar image.Image, qualitaet int) (fehler error) {
	defer errprefix.Defer("fileformat", &fehler)
	if qualitaet < 1 || qualitaet > 100 {
		return fmt.Errorf("fileformat: JPG-Qualität %d außerhalb 1–100", qualitaet)
	}
	b := sichtbar.Bounds()
	flach := image.NewRGBA(b)
	draw.Draw(flach, b, image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(flach, b, sichtbar, b.Min, draw.Over)
	return jpeg.Encode(w, flach, &jpeg.Options{Quality: qualitaet})
}

// intervalle liefert die sichtbaren Bereiche einer Achse (im Original), aus Zuschnitt und Streifen.
func intervalle(von, bis int, st []document.Streifen, zeilen bool) [][2]int {
	var res [][2]int
	start := von
	for _, s := range st {
		if s.Zeilen != zeilen || s.Bis <= von || s.Von >= bis {
			continue
		}
		if a := max(s.Von, von); a > start {
			res = append(res, [2]int{start, a})
		}
		start = max(start, min(s.Bis, bis))
	}
	if start < bis {
		res = append(res, [2]int{start, bis})
	}
	return res
}

// nahtArt ist der Name der Achse in den ids des SVG-Exports.
func nahtArt(zeilen bool) string {
	if zeilen {
		return "zeilen"
	}
	return "spalten"
}

// bandPfad schreibt ein Kantenband als Pfadangabe (M … L … Z).
func bandPfad(band []document.Punkt) string {
	var pfad strings.Builder
	for i, p := range band {
		if i == 0 {
			pfad.WriteString("M")
		} else {
			pfad.WriteString(" L")
		}
		pfad.WriteString(punkt(p))
	}
	pfad.WriteString(" Z")
	return pfad.String()
}

// naehteDefs schreibt die Kanten der Stauch-Nähte als Vorlagen (ADR-0023 Punkt 6, Nachträge 6 und 7): dieselben Vielecke
// wie im Renderer (document.Kantenband, in der Form des Dokuments), ohne Filter. Je Achse steht einmal das Band bei Lage
// 0 über die ganze Länge in <defs>: die Schichten (document.KantenSchichten, halbdurchsichtige mit fill-opacity) als
// Gruppe naht-zeilen bzw. naht-spalten, der Spalt (document.Spaltband) als Pfad spalt-zeilen bzw. spalt-spalten. Dazu
// die Maske spalt: weiß über die ganze Ansicht, je Naht der Spalt schwarz darin. Die Kacheln liegen unter dieser Maske
// (der Spalt bleibt leer, Bild und Objekte darin fallen weg), die Schichten darüber (naehteUse). Je Naht kostet das zwei
// kurze <use>-Zeilen statt aller Schichten über die ganze Bildbreite (F-183: vorher 41 MB nur für 1023 Nähte in
// 8192 px Breite). Ohne Naht schreibt es nichts und meldet false.
func (s *schreiber) naehteDefs(d *document.Dokument) bool {
	g := d.Groesse()
	naehte := d.Naehte()
	if len(naehte) == 0 {
		return false
	}
	spaltVon, spaltBis := document.Spaltband()
	beruehren := document.NaehteBeruehren(naehte, d.KantenFaktor())
	s.WriteString(`<defs>`)
	for _, zeilen := range []bool{false, true} {
		if !slices.ContainsFunc(naehte, func(n document.Naht) bool { return n.Zeilen == zeilen }) {
			continue
		}
		laenge := g.Y
		if zeilen {
			laenge = g.X
		}
		linie := document.NeueKantenlinie(d.Kante(), laenge, d.KantenFaktor()) // einmal je Achse (F-418)
		bei0 := document.Naht{Zeilen: zeilen}
		var band []document.Punkt
		// Berühren sich Nähte (F-438), liegen Fasersaum und Abdunklung (naht-…) und der Schatten (schatten-…) in zwei
		// Gruppen: Die erste legt naehteUse unter die Maske, der Schatten bleibt im Spalt stehen.
		gruppe := func(id string, nimm func(document.Schicht) bool) {
			s.WriteString(`<g`)
			s.attr("id", id+nahtArt(zeilen))
			s.WriteString(`>`)
			for _, sch := range document.KantenSchichten() {
				if !nimm(sch) {
					continue
				}
				band = linie.Band(band, bei0, sch.Von, sch.Bis)
				s.WriteString(`<path`)
				s.attr("d", bandPfad(band))
				s.farbeSVG("fill", sch.Farbe)
				s.WriteString(`/>`)
			}
			s.WriteString(`</g>`)
		}
		if beruehren {
			gruppe("naht-", func(sch document.Schicht) bool { return !sch.ImSpalt() })
			gruppe("schatten-", document.Schicht.ImSpalt)
		} else {
			gruppe("naht-", func(document.Schicht) bool { return true })
		}
		s.WriteString(`<path`)
		s.attr("id", "spalt-"+nahtArt(zeilen))
		band = linie.Band(band, bei0, spaltVon, spaltBis)
		s.attr("d", bandPfad(band))
		s.attr("fill", "#000")
		s.WriteString(`/>`)
	}
	s.WriteString(`<mask id="spalt" maskUnits="userSpaceOnUse" x="0" y="0"`)
	s.attr("width", strconv.Itoa(g.X))
	s.attr("height", strconv.Itoa(g.Y))
	s.WriteString(`><rect`)
	s.attr("width", strconv.Itoa(g.X))
	s.attr("height", strconv.Itoa(g.Y))
	s.WriteString(` fill="#fff"/>`)
	s.nahtVerweise(naehte, "spalt-", false)
	s.WriteString(`</mask></defs>`)
	return true
}

// naehteUse legt die Schichten jeder Naht über die Kacheln (siehe naehteDefs).
//
// Berühren sich Nähte (document.NaehteBeruehren, F-438, ADR-0023 Nachtrag 9), liegen die Schichten neben dem Spalt unter
// derselben Maske wie die Kacheln, damit der Fasersaum einer Naht nicht im Spalt der anderen steht, und der Schatten
// darüber, ohne Maske.
func (s *schreiber) naehteUse(d *document.Dokument) {
	naehte := d.Naehte()
	if !document.NaehteBeruehren(naehte, d.KantenFaktor()) {
		s.nahtVerweise(naehte, "naht-", true)
		return
	}
	s.WriteString(`<g mask="url(#spalt)">`)
	s.nahtVerweise(naehte, "naht-", true)
	s.WriteString(`</g>`)
	s.nahtVerweise(naehte, "schatten-", false)
}

// nahtVerweise schreibt je Naht ein <use> auf die Vorlage praefix+Achse, verschoben auf die Lage der Naht; mitArt
// schreibt in der Arbeitsdatei die Achse dazu (s:naht).
func (s *schreiber) nahtVerweise(naehte []document.Naht, praefix string, mitArt bool) {
	for _, n := range naehte {
		s.WriteString(`<use`)
		s.attr("href", "#"+praefix+nahtArt(n.Zeilen))
		if n.Zeilen {
			s.attr("y", strconv.Itoa(n.Lage))
		} else {
			s.attr("x", strconv.Itoa(n.Lage))
		}
		if mitArt {
			s.attr("s:naht", nahtArt(n.Zeilen))
		}
		s.WriteString(`/>`)
	}
}

// ExportiereSVG schreibt die Ansicht als eigenständiges SVG. Je sichtbarer Kachel (Zuschnitt und Stauchen teilen
// die Ansicht in Rechtecke) steht ein verschachteltes <svg> mit viewBox, das nur den Pixelausschnitt dieser Kachel als
// eingebettetes PNG trägt und die Objekte als Vektoren zeigt. Weggeschnittene oder weggestauchte Bereiche sind in
// der Datei nicht enthalten (F-60) — anders als in der Arbeitsdatei, die nicht zerstörend das ganze Original mitführt.
func ExportiereSVG(w io.Writer, d *document.Dokument) (fehler error) {
	defer errprefix.Defer("fileformat", &fehler)
	original := d.OriginalBild()
	g := d.Groesse()
	var s schreiber
	s.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" + `<svg xmlns="http://www.w3.org/2000/svg"`)
	s.attr("xmlns:s", ns)
	s.attr("width", strconv.Itoa(g.X))
	s.attr("height", strconv.Itoa(g.Y))
	s.attr("viewBox", fmt.Sprintf("0 0 %d %d", g.X, g.Y))
	z := d.Zuschnitt()
	st := d.Streifen()
	// Objekte, die ganz außerhalb aller sichtbaren Kacheln liegen (weggeschnitten oder weggestaucht), kommen nicht in
	// die Datei — sonst stünde etwa ein Text dort im Klartext im XML (F-66). Dieselbe Regel gilt im Renderer und im
	// Editor (Dokument.ObjektSichtbar, ADR-0023 Punkt 7).
	s.WriteString(`><defs><g id="objekte">`)
	s.objekte(d, d.ObjektSichtbar)
	s.WriteString(`</g></defs>`)
	// Gibt es Nähte, liegen die Kacheln unter der Maske, die den Spalt jeder Naht leer lässt (ADR-0023 Nachtrag 7).
	maske := s.naehteDefs(d)
	if maske {
		s.WriteString(`<g mask="url(#spalt)">`)
	}
	ay := 0
	for _, zy := range intervalle(z.Min.Y, z.Max.Y, st, true) {
		ax := 0
		for _, zx := range intervalle(z.Min.X, z.Max.X, st, false) {
			b, h := zx[1]-zx[0], zy[1]-zy[0]
			var kachel bytes.Buffer
			if err := png.Encode(&kachel, original.SubImage(image.Rect(zx[0], zy[0], zx[1], zy[1]))); err != nil {
				return err
			}
			s.WriteString(`<svg`)
			s.attr("x", strconv.Itoa(ax))
			s.attr("y", strconv.Itoa(ay))
			s.attr("width", strconv.Itoa(b))
			s.attr("height", strconv.Itoa(h))
			// viewBox ab 0,0 und Verschiebung per transform statt viewBox in Original-Koordinaten (F-225). Die Lage
			// des Zuschnitts im Original steht damit weiter in transform, im x/y des Bilds und in den Koordinaten der
			// Objekte (Original-Koordinaten); entfernt sind nur Programm-Namensraum und Bearbeitungsattribute.
			s.attr("viewBox", fmt.Sprintf("0 0 %d %d", b, h))
			s.WriteString(`><g`)
			s.attr("transform", fmt.Sprintf("translate(%d %d)", -zx[0], -zy[0]))
			s.WriteString(`><image`)
			s.attr("x", strconv.Itoa(zx[0]))
			s.attr("y", strconv.Itoa(zy[0]))
			s.attr("width", strconv.Itoa(b))
			s.attr("height", strconv.Itoa(h))
			s.attr("href", "data:image/png;base64,"+base64.StdEncoding.EncodeToString(kachel.Bytes()))
			s.WriteString(`/><use href="#objekte"/></g></svg>`)
			ax += b
		}
		ay += zy[1] - zy[0]
	}
	if maske {
		s.WriteString(`</g>`)
		s.naehteUse(d)
	}
	s.WriteString("</svg>\n")
	_, err := io.WriteString(w, s.String())
	return err
}
