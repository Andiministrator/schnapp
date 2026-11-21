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
)

// ExportierePNG schreibt das sichtbare Bild als gewöhnliches PNG ohne Bearbeitungsdaten (zum Weitergeben).
func ExportierePNG(w io.Writer, sichtbar image.Image) error { return png.Encode(w, sichtbar) }

// ExportiereJPG schreibt das sichtbare Bild als JPG (Qualität 1–100). Transparenz wird auf Weiß gelegt.
func ExportiereJPG(w io.Writer, sichtbar image.Image, qualitaet int) error {
	if qualitaet < 1 || qualitaet > 100 {
		return fmt.Errorf("datei: JPG-Qualität %d außerhalb 1–100", qualitaet)
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

// naehte schreibt die Zackenkanten der Stauch-Nähte in Ansichtskoordinaten über alle Kacheln (ADR-0023 Punkt 6):
// dieselben Vielecke wie im Renderer (document.Zackenband), hell außen, dunkel innen. Das Band jeder Achse steht einmal
// in <defs> (Naht bei Lage 0 über die ganze Länge, id naht-zeilen bzw. naht-spalten); jede Naht verweist darauf mit
// <use> und verschiebt es auf ihre Lage. So wächst die Datei je Naht um eine kurze Zeile statt um zwei Bänder über die
// ganze Bildbreite (F-183: vorher 41 MB nur für 1023 Nähte in 8192 px Breite).
func (s *schreiber) naehte(d *document.Dokument) {
	g := d.Groesse()
	naehte := d.Naehte()
	if len(naehte) == 0 {
		return
	}
	art := func(zeilen bool) string {
		if zeilen {
			return "zeilen"
		}
		return "spalten"
	}
	s.WriteString(`<defs>`)
	for _, zeilen := range []bool{false, true} {
		if !slices.ContainsFunc(naehte, func(n document.Naht) bool { return n.Zeilen == zeilen }) {
			continue
		}
		laenge := g.Y
		if zeilen {
			laenge = g.X
		}
		s.WriteString(`<g`)
		s.attr("id", "naht-"+art(zeilen))
		s.WriteString(`>`)
		for _, band := range []struct {
			halb  float64
			farbe color.NRGBA
		}{{document.ZackenRandHalb, document.ZackenRand}, {document.ZackenLinieHalb, document.ZackenLinie}} {
			var pfad strings.Builder
			for i, p := range document.Zackenband(document.Naht{Zeilen: zeilen}, laenge, d.ZackenFaktor(), band.halb) {
				if i == 0 {
					pfad.WriteString("M")
				} else {
					pfad.WriteString(" L")
				}
				pfad.WriteString(punkt(p))
			}
			pfad.WriteString(" Z")
			s.WriteString(`<path`)
			s.attr("d", pfad.String())
			s.farbeSVG("fill", band.farbe)
			s.WriteString(`/>`)
		}
		s.WriteString(`</g>`)
	}
	s.WriteString(`</defs>`)
	for _, n := range naehte {
		s.WriteString(`<use`)
		s.attr("href", "#naht-"+art(n.Zeilen))
		if n.Zeilen {
			s.attr("y", strconv.Itoa(n.Lage))
		} else {
			s.attr("x", strconv.Itoa(n.Lage))
		}
		s.attr("s:naht", art(n.Zeilen))
		s.WriteString(`/>`)
	}
}

// ExportiereSVG schreibt die Ansicht als eigenständiges SVG. Je sichtbarer Kachel (Zuschnitt und Stauchen teilen
// die Ansicht in Rechtecke) steht ein verschachteltes <svg> mit viewBox, das nur den Pixelausschnitt dieser Kachel als
// eingebettetes PNG trägt und die Objekte als Vektoren zeigt. Weggeschnittene oder weggestauchte Bereiche sind in
// der Datei nicht enthalten (F-60) — anders als in der Arbeitsdatei, die nicht zerstörend das ganze Original mitführt.
func ExportiereSVG(w io.Writer, d *document.Dokument) error {
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
	s.naehte(d)
	s.WriteString("</svg>\n")
	_, err := io.WriteString(w, s.String())
	return err
}
