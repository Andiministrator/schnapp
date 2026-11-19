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
	"strconv"

	"github.com/Andiministrator/schnapp/document"
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

// ExportiereSVG schreibt die Ansicht als eigenständiges SVG. Je sichtbarer Kachel (Zuschnitt und Stauchen teilen
// die Ansicht in Rechtecke) steht ein verschachteltes <svg> mit viewBox, das nur den Pixelausschnitt dieser Kachel als
// eingebettetes PNG trägt und die Objekte als Vektoren zeigt. Weggeschnittene oder weggestauchte Bereiche sind in
// der Datei nicht enthalten (F-60) — anders als in der Arbeitsdatei, die nicht zerstörend das ganze Original mitführt.
func ExportiereSVG(w io.Writer, d *document.Dokument) error {
	original := d.Original()
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
	// die Datei — sonst stünde etwa ein Text dort im Klartext im XML (F-66).
	var kacheln []document.Rechteck
	for _, zy := range intervalle(z.Min.Y, z.Max.Y, st, true) {
		for _, zx := range intervalle(z.Min.X, z.Max.X, st, false) {
			kacheln = append(kacheln, document.Rechteck{Min: document.Pt(float64(zx[0]), float64(zy[0])),
				Max: document.Pt(float64(zx[1]), float64(zy[1]))})
		}
	}
	inAnsicht := func(g document.Rechteck) bool {
		for _, k := range kacheln {
			if g.Min.X < k.Max.X && g.Max.X > k.Min.X && g.Min.Y < k.Max.Y && g.Max.Y > k.Min.Y {
				return true
			}
		}
		return false
	}
	s.WriteString(`><defs><g id="objekte">`)
	s.objekte(d, inAnsicht)
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
			s.attr("viewBox", fmt.Sprintf("%d %d %d %d", zx[0], zy[0], b, h))
			s.WriteString(`><image`)
			s.attr("x", strconv.Itoa(zx[0]))
			s.attr("y", strconv.Itoa(zy[0]))
			s.attr("width", strconv.Itoa(b))
			s.attr("height", strconv.Itoa(h))
			s.attr("href", "data:image/png;base64,"+base64.StdEncoding.EncodeToString(kachel.Bytes()))
			s.WriteString(`/><use href="#objekte"/></svg>`)
			ax += b
		}
		ay += zy[1] - zy[0]
	}
	s.WriteString("</svg>\n")
	_, err := io.WriteString(w, s.String())
	return err
}
