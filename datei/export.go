// SPDX-License-Identifier: GPL-3.0-or-later

package datei

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
	"strconv"

	"github.com/Andiministrator/schnapp/dokument"
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
func intervalle(von, bis int, st []dokument.Streifen, zeilen bool) [][2]int {
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

// ExportiereSVG schreibt die Ansicht als eigenständiges SVG: das Original als eingebettetes PNG (mit allen
// Schwärzungen) und die Objekte als Vektoren. Zuschnitt und Stauchen werden als Ausschnitte dargestellt — je
// sichtbarer Kachel ein verschachteltes <svg> mit viewBox, das auf einen gemeinsamen Inhalt verweist.
func ExportiereSVG(w io.Writer, d *dokument.Dokument) error {
	var orig bytes.Buffer
	if err := png.Encode(&orig, d.Original()); err != nil {
		return err
	}
	o := d.Original().Rect
	g := d.Groesse()
	var s schreiber
	s.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" + `<svg xmlns="http://www.w3.org/2000/svg"`)
	s.attr("xmlns:s", ns)
	s.attr("width", strconv.Itoa(g.X))
	s.attr("height", strconv.Itoa(g.Y))
	s.attr("viewBox", fmt.Sprintf("0 0 %d %d", g.X, g.Y))
	s.WriteString(`><defs><g id="inhalt"><image`)
	s.attr("x", "0")
	s.attr("y", "0")
	s.attr("width", strconv.Itoa(o.Dx()))
	s.attr("height", strconv.Itoa(o.Dy()))
	s.attr("href", "data:image/png;base64,"+base64.StdEncoding.EncodeToString(orig.Bytes()))
	s.WriteString(`/>`)
	s.objekte(d)
	s.WriteString(`</g></defs>`)
	z := d.Zuschnitt()
	st := d.Streifen()
	ay := 0
	for _, zy := range intervalle(z.Min.Y, z.Max.Y, st, true) {
		ax := 0
		for _, zx := range intervalle(z.Min.X, z.Max.X, st, false) {
			b, h := zx[1]-zx[0], zy[1]-zy[0]
			s.WriteString(`<svg`)
			s.attr("x", strconv.Itoa(ax))
			s.attr("y", strconv.Itoa(ay))
			s.attr("width", strconv.Itoa(b))
			s.attr("height", strconv.Itoa(h))
			s.attr("viewBox", fmt.Sprintf("%d %d %d %d", zx[0], zy[0], b, h))
			s.WriteString(`><use href="#inhalt"/></svg>`)
			ax += b
		}
		ay += zy[1] - zy[0]
	}
	s.WriteString("</svg>\n")
	_, err := io.WriteString(w, s.String())
	return err
}
