// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"encoding/xml"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

// FormatVersion ist die Version des SVG-Blocks (ADR-0006, ADR-0008). Ältere Versionen müssen lesbar bleiben.
const FormatVersion = 1

// ns ist der Namensraum der Schnapp-Attribute. Er kennzeichnet das Format, nicht den Rechner oder Nutzer.
const ns = "urn:x-schnapp:1"

// maxTiefe ist die größte Verschachtelungstiefe des SVG beim Lesen (docs/format.md, Abschnitt Limits).
const maxTiefe = 64

// Metadaten stehen im SVG-Block; die Bibliothek baut ihren Index daraus neu auf (ADR-0016). Keine Pfade, keine
// Rechner- oder Nutzernamen (Kernprinzip 3); den Autor trägt nur ein, wer ihn einstellt.
type Metadaten struct {
	Autor       string
	Erstellt    time.Time // in UTC gespeichert, sekundengenau
	Quelle      string    // Quellprogramm, wenn bekannt (nur bei Fenster-Aufnahmen verlässlich)
	Schlagworte []string
	Bearbeitet  bool
}

// --- Schreiben ---

// schreiber baut SVG-Text. Mit rundlauf (Arbeitsdatei, snAP-Block) stehen die Bearbeitungsattribute im Namensraum
// ns dabei, damit die Datei wieder lesbar ist; ohne (Export, ADR-0025 Punkt 4, F-225) fehlen Namensraum und alle
// Attribute mit Präfix `s:` — der Export ist ein gewöhnliches SVG ohne Spuren des Programms.
type schreiber struct {
	strings.Builder
	rundlauf bool
}

func (s *schreiber) attr(name, wert string) {
	if !s.rundlauf && (strings.HasPrefix(name, "s:") || name == "xmlns:s") {
		return
	}
	s.WriteString(" " + name + `="`)
	xml.EscapeText(s, []byte(wert))
	s.WriteString(`"`)
}

func zahl(f float64) string         { return strconv.FormatFloat(f, 'g', -1, 64) }
func punkt(p document.Punkt) string { return zahl(p.X) + "," + zahl(p.Y) }

// farbeHex schreibt #rrggbbaa (exakt, für die Daten).
func farbeHex(c color.NRGBA) string { return fmt.Sprintf("#%02x%02x%02x%02x", c.R, c.G, c.B, c.A) }

// farbeSVG schreibt Farbe und Deckkraft für die Darstellung.
func (s *schreiber) farbeSVG(art string, c color.NRGBA) {
	if c.A == 0 {
		s.attr(art, "none")
		return
	}
	s.attr(art, fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B))
	if c.A != 255 {
		s.attr(art+"-opacity", zahl(float64(c.A)/255))
	}
}

func strich(st document.Strich) string { return farbeHex(st.Farbe) + " " + zahl(st.Breite) }

// svgSchrift ist die Schrift der Beschriftungen im SVG: Renderer und Editor setzen Go Regular und Go Bold (Familie „Go“);
// ein Betrachter ohne diese Schrift fällt auf eine serifenlose Schrift zurück (F-108).
const svgSchrift = "Go, Go Regular, sans-serif"

// objekte schreibt die Objekte in Original-Koordinaten; mit behalten != nil nur die, für die behalten true liefert
// (Export: nur Objekte, die in die Ansicht reichen).
func (s *schreiber) objekte(d *document.Dokument, behalten func(document.Objekt) bool) {
	s.WriteString(`<g`)
	s.attr("s:ebene", "objekte")
	s.WriteString(`>`)
	for _, o := range d.Objekte() {
		if behalten != nil && !behalten(o) {
			continue
		}
		switch o := o.(type) {
		case document.Pfeil:
			s.WriteString(`<g`)
			s.attr("s:typ", "pfeil")
			s.attr("s:a", punkt(o.A))
			s.attr("s:b", punkt(o.B))
			s.attr("s:k", punkt(o.K))
			s.attr("s:strich", strich(o.Strich))
			s.WriteString(`><path`)
			s.attr("d", "M"+punkt(o.A)+" Q"+punkt(o.K)+" "+punkt(o.B))
			s.attr("fill", "none")
			s.farbeSVG("stroke", o.Strich.Farbe)
			s.attr("stroke-width", zahl(o.Strich.Breite))
			s.attr("stroke-linecap", "round")
			s.WriteString(`/><path`)
			r := o.RichtungAmEnde()
			g := math.Max(8, 4*o.Strich.Breite)
			n := document.Pt(-r.Y, r.X)
			fuss := o.B.Minus(r.Mal(g))
			s.attr("d", "M"+punkt(o.B)+" L"+punkt(fuss.Plus(n.Mal(g/2)))+" L"+punkt(fuss.Minus(n.Mal(g/2)))+" Z")
			s.farbeSVG("fill", o.Strich.Farbe)
			s.WriteString(`/></g>`)
		case document.Kasten:
			b := o.Bereich
			s.WriteString(`<rect`)
			s.attr("s:typ", "kasten")
			s.attr("s:bereich", punkt(b.Min)+" "+punkt(b.Max))
			s.attr("s:strich", strich(o.Strich))
			s.attr("s:fuellung", farbeHex(o.Fuellung))
			s.attr("s:eckradius", zahl(o.Eckradius))
			s.attr("x", zahl(b.Min.X))
			s.attr("y", zahl(b.Min.Y))
			s.attr("width", zahl(b.Breite()))
			s.attr("height", zahl(b.Hoehe()))
			// Rundung wie der Renderer: gleiche Begrenzung (nie elliptisch) und runde Ecken des Strichs (F-108).
			s.attr("rx", zahl(document.WirksamerEckradius(o.Eckradius, b.Breite(), b.Hoehe())))
			s.farbeSVG("fill", o.Fuellung)
			s.farbeSVG("stroke", o.Strich.Farbe)
			s.attr("stroke-width", zahl(o.Strich.Breite))
			s.attr("stroke-linejoin", "round")
			s.WriteString(`/>`)
		case document.Text:
			s.WriteString(`<g`)
			s.attr("s:typ", "text")
			s.attr("s:ort", punkt(o.Ort))
			s.attr("s:inhalt", o.Inhalt)
			s.attr("s:grad", zahl(o.Schriftgrad))
			s.attr("s:farbe", farbeHex(o.Farbe))
			s.attr("s:hintergrund", farbeHex(o.Hintergrund))
			s.attr("s:umbruch", zahl(o.BreiteGrenze))
			s.WriteString(`>`)
			if o.Hintergrund.A > 0 {
				g := o.Grenzen()
				s.WriteString(`<rect`)
				s.attr("x", zahl(g.Min.X))
				s.attr("y", zahl(g.Min.Y))
				s.attr("width", zahl(g.Breite()))
				s.attr("height", zahl(g.Hoehe()))
				s.farbeSVG("fill", o.Hintergrund)
				s.WriteString(`/>`)
			}
			s.WriteString(`<text`)
			s.attr("xml:space", "preserve") // sonst falten Betrachter mehrere und führende Leerzeichen (F-343)
			s.attr("font-family", svgSchrift)
			s.attr("font-size", zahl(o.Schriftgrad))
			s.farbeSVG("fill", o.Farbe)
			s.WriteString(`>`)
			for i, zeile := range strings.Split(o.Inhalt, "\n") {
				s.WriteString(`<tspan`)
				s.attr("x", zahl(o.Ort.X))
				s.attr("y", zahl(o.Grundlinie(i)))
				s.WriteString(`>`)
				xml.EscapeText(s, []byte(document.Zeichenbar(zeile))) // sichtbar wie im Renderer: Tabulator als vier Leerzeichen (F-343); s:inhalt behält das Original
				s.WriteString(`</tspan>`)
			}
			s.WriteString(`</text></g>`)
		case document.Nummer:
			s.WriteString(`<g`)
			s.attr("s:typ", "nummer")
			s.attr("s:mitte", punkt(o.Mitte))
			s.attr("s:wert", strconv.Itoa(o.Wert))
			s.attr("s:radius", zahl(o.Radius))
			s.attr("s:farbe", farbeHex(o.Farbe))
			s.WriteString(`><circle`)
			s.attr("cx", zahl(o.Mitte.X))
			s.attr("cy", zahl(o.Mitte.Y))
			s.attr("r", zahl(o.Radius))
			s.farbeSVG("fill", o.Farbe)
			s.WriteString(`/><text`)
			s.attr("x", zahl(o.Mitte.X))
			s.attr("y", zahl(o.Grundlinie()))
			s.attr("font-family", svgSchrift)
			s.attr("font-size", zahl(o.Radius*1.2))
			s.attr("font-weight", "bold")
			s.attr("text-anchor", "middle")
			s.attr("fill", "#ffffff")
			s.WriteString(`>` + strconv.Itoa(o.Wert) + `</text></g>`)
		}
	}
	s.WriteString(`</g>`)
}

// svgArbeitsdatei erzeugt das SVG des snAP-Blocks: Original (Verweis auf snAO) und Objekte in Original-Koordinaten,
// Ansicht und Metadaten in <metadata>.
func svgArbeitsdatei(d *document.Dokument, m Metadaten) []byte {
	o := image.Rectangle{Max: d.OriginalGroesse()}
	s := schreiber{rundlauf: true}
	s.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" + `<svg xmlns="http://www.w3.org/2000/svg"`)
	s.attr("xmlns:s", ns)
	s.attr("s:format", strconv.Itoa(FormatVersion))
	s.attr("width", strconv.Itoa(o.Dx()))
	s.attr("height", strconv.Itoa(o.Dy()))
	s.attr("viewBox", fmt.Sprintf("0 0 %d %d", o.Dx(), o.Dy()))
	s.WriteString(`><metadata><s:meta`)
	if m.Autor != "" {
		s.attr("s:autor", m.Autor)
	}
	if !m.Erstellt.IsZero() {
		s.attr("s:erstellt", m.Erstellt.UTC().Truncate(time.Second).Format(time.RFC3339))
	}
	if m.Quelle != "" {
		s.attr("s:quelle", m.Quelle)
	}
	s.attr("s:bearbeitet", strconv.FormatBool(m.Bearbeitet))
	s.WriteString(`>`)
	for _, w := range m.Schlagworte {
		s.WriteString(`<s:schlagwort>`)
		xml.EscapeText(&s, []byte(w))
		s.WriteString(`</s:schlagwort>`)
	}
	s.WriteString(`</s:meta><s:ansicht`)
	z := d.Zuschnitt()
	s.attr("s:zuschnitt", fmt.Sprintf("%d %d %d %d", z.Min.X, z.Min.Y, z.Max.X, z.Max.Y))
	s.WriteString(`>`)
	for _, st := range d.Streifen() {
		s.WriteString(`<s:streifen`)
		s.attr("s:zeilen", strconv.FormatBool(st.Zeilen))
		s.attr("s:von", strconv.Itoa(st.Von))
		s.attr("s:bis", strconv.Itoa(st.Bis))
		s.WriteString(`/>`)
	}
	s.WriteString(`</s:ansicht></metadata><image`)
	s.attr("x", "0")
	s.attr("y", "0")
	s.attr("width", strconv.Itoa(o.Dx()))
	s.attr("height", strconv.Itoa(o.Dy()))
	s.attr("href", "schnapp:original") // steht im Block snAO
	s.WriteString(`/>`)
	s.objekte(d, nil)
	s.WriteString("</svg>\n")
	return []byte(s.String())
}

// --- Lesen ---

// inhalt ist, was das SVG eines snAP-Blocks beschreibt.
type inhalt struct {
	breite, hoehe int
	zuschnitt     image.Rectangle
	streifen      []document.Streifen
	objekte       []document.Objekt
	meta          Metadaten
}

func attrs(e xml.StartElement) map[string]string {
	m := map[string]string{}
	for _, a := range e.Attr {
		if a.Name.Space == ns {
			m[a.Name.Local] = a.Value
		} else if a.Name.Space == "" {
			m["svg:"+a.Name.Local] = a.Value
		}
	}
	return m
}

type leser struct {
	a   map[string]string
	err error
}

func (l *leser) fehler(format string, args ...any) {
	if l.err == nil {
		l.err = fmt.Errorf("%w: "+format, append([]any{ErrBeschaedigt}, args...)...)
	}
}

func (l *leser) text(name string) string {
	v, ok := l.a[name]
	if !ok {
		l.fehler("Attribut %s fehlt", name)
	}
	return v
}

func (l *leser) zahl(name string) float64 {
	f, err := strconv.ParseFloat(l.text(name), 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		l.fehler("Zahl %s ungültig", name)
	}
	return f
}

func (l *leser) ganz(name string) int {
	n, err := strconv.Atoi(l.text(name))
	if err != nil {
		l.fehler("Ganzzahl %s ungültig", name)
	}
	return n
}

func (l *leser) punkt(name string) document.Punkt {
	x, y, ok := strings.Cut(l.text(name), ",")
	fx, e1 := strconv.ParseFloat(x, 64)
	fy, e2 := strconv.ParseFloat(y, 64)
	if !ok || e1 != nil || e2 != nil {
		l.fehler("Punkt %s ungültig", name)
	}
	return document.Pt(fx, fy)
}

func (l *leser) farbe(name string) color.NRGBA {
	v := l.text(name)
	var c color.NRGBA
	if n, err := fmt.Sscanf(v, "#%02x%02x%02x%02x", &c.R, &c.G, &c.B, &c.A); err != nil || n != 4 || len(v) != 9 {
		l.fehler("Farbe %s ungültig", name)
	}
	return c
}

func (l *leser) strich(name string) document.Strich {
	f, b, _ := strings.Cut(l.text(name), " ")
	l.a["_f"], l.a["_b"] = f, b
	return document.Strich{Farbe: l.farbe("_f"), Breite: l.zahl("_b")}
}

func (l *leser) objekt(typ string) document.Objekt {
	switch typ {
	case "pfeil":
		return document.Pfeil{A: l.punkt("a"), B: l.punkt("b"), K: l.punkt("k"), Strich: l.strich("strich")}
	case "kasten":
		min, max, _ := strings.Cut(l.text("bereich"), " ")
		l.a["_min"], l.a["_max"] = min, max
		return document.Kasten{Bereich: document.Rechteck{Min: l.punkt("_min"), Max: l.punkt("_max")},
			Strich: l.strich("strich"), Fuellung: l.farbe("fuellung"), Eckradius: l.zahl("eckradius")}
	case "text":
		return document.Text{Ort: l.punkt("ort"), Inhalt: l.text("inhalt"), Schriftgrad: l.zahl("grad"),
			Farbe: l.farbe("farbe"), Hintergrund: l.farbe("hintergrund"), BreiteGrenze: l.zahl("umbruch")}
	case "nummer":
		return document.Nummer{Mitte: l.punkt("mitte"), Wert: l.ganz("wert"), Radius: l.zahl("radius"),
			Farbe: l.farbe("farbe")}
	}
	// Unbekannte Objekttypen aus neueren Fassungen: als Fehler melden statt still verlieren.
	l.fehler("unbekannter Objekttyp %q", typ)
	return nil
}

// leseSVG liest das SVG eines snAP-Blocks streng: nur die eigenen Attribute zählen, Darstellungselemente werden
// übergangen, Verweise werden nie aufgelöst.
func leseSVG(r io.Reader) (*inhalt, error) {
	dec := xml.NewDecoder(io.LimitReader(r, maxSVG))
	dec.Strict = true
	var (
		in              inhalt
		tiefe, elemente int
		version         = -1
		hatZuschnitt    bool
		inSchlagwort    bool
		inObjekt        int // Tiefe des aktuellen Objekts, 0 = keins
	)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: SVG: %v", ErrBeschaedigt, err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			tiefe++
			if elemente++; elemente > maxElemente || tiefe > maxTiefe {
				return nil, fmt.Errorf("%w: SVG zu groß oder zu tief", ErrZuGross)
			}
			if inObjekt > 0 {
				continue // Darstellung eines Objekts
			}
			l := &leser{a: attrs(t)}
			switch {
			case t.Name.Local == "svg" && tiefe == 1:
				version = l.ganz("format")
				in.breite, in.hoehe = l.ganz("svg:width"), l.ganz("svg:height")
			case tiefe == 1:
				return nil, fmt.Errorf("%w: Wurzel ist kein svg", ErrBeschaedigt)
			case t.Name.Space == ns && t.Name.Local == "meta":
				in.meta = Metadaten{Autor: l.a["autor"], Quelle: l.a["quelle"], Bearbeitet: l.a["bearbeitet"] == "true"}
				if e := l.a["erstellt"]; e != "" {
					zeit, err := time.Parse(time.RFC3339, e)
					if err != nil {
						l.fehler("Datum ungültig")
					}
					in.meta.Erstellt = zeit
				}
			case t.Name.Space == ns && t.Name.Local == "schlagwort":
				inSchlagwort = true
				in.meta.Schlagworte = append(in.meta.Schlagworte, "")
			case t.Name.Space == ns && t.Name.Local == "ansicht":
				var z [4]int
				if n, err := fmt.Sscanf(l.text("zuschnitt"), "%d %d %d %d", &z[0], &z[1], &z[2], &z[3]); err != nil || n != 4 {
					l.fehler("Zuschnitt ungültig")
				}
				in.zuschnitt, hatZuschnitt = image.Rect(z[0], z[1], z[2], z[3]), true
			case t.Name.Space == ns && t.Name.Local == "streifen":
				in.streifen = append(in.streifen, document.Streifen{Zeilen: l.text("zeilen") == "true",
					Von: l.ganz("von"), Bis: l.ganz("bis")})
			case l.a["typ"] != "":
				if o := l.objekt(l.a["typ"]); o != nil {
					in.objekte = append(in.objekte, o)
				}
				inObjekt = tiefe
			}
			if l.err != nil {
				return nil, l.err
			}
		case xml.EndElement:
			if inObjekt == tiefe {
				inObjekt = 0
			}
			inSchlagwort = false
			tiefe--
		case xml.CharData:
			if inSchlagwort {
				in.meta.Schlagworte[len(in.meta.Schlagworte)-1] += string(t)
			}
		}
	}
	switch {
	case version < 0:
		return nil, fmt.Errorf("%w: Formatversion fehlt", ErrBeschaedigt)
	case version > FormatVersion:
		return nil, fmt.Errorf("%w: %d > %d", ErrFormatNeuer, version, FormatVersion)
	case !hatZuschnitt:
		return nil, fmt.Errorf("%w: Ansicht fehlt", ErrBeschaedigt)
	}
	return &in, nil
}
