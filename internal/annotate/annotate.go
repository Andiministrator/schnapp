// SPDX-License-Identifier: GPL-3.0-or-later

// Package annotate liest eine Liste von Beschriftungen und Schwärzungen (JSON, ADR-0030 Nachtrag 10) und legt sie auf
// ein Dokument, ohne Fenster: für `schnapp --annotate=SPEC DATEI` und das MCP-Werkzeug `annotate`. Die Werte, die eine
// Liste offen lässt, und die Abbildung eines Schwärzungsrahmens aufs Original kommen aus canvas, damit Kommandozeile
// und Editor gleich rechnen. Fehlertexte sind englisch (sie gehen auch an Agenten, ADR-0030 Nachtrag 5 Punkt 6) und
// nennen den Index des Eintrags.
package annotate

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"io"
	"math"
	"strings"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// Grenzen der Liste: MaxBytes für die JSON-Datei, MaxEintraege für die Zahl der Einträge (mehr Objekte nimmt ein
// Dokument ohnehin nicht, document.MaxObjekte).
const (
	MaxBytes     = 1 << 20
	MaxEintraege = document.MaxObjekte
)

// Fehler ist ein Fehler in der Liste; Index ist der Eintrag (ab 0), -1 die Liste als Ganzes.
type Fehler struct {
	Index int
	Grund string
}

func (f *Fehler) Error() string {
	if f.Index < 0 {
		return "annotation spec: " + f.Grund
	}
	return fmt.Sprintf("annotation %d: %s", f.Index, f.Grund)
}

// Eintrag ist ein gelesener Eintrag der Liste. Zeiger sind „nicht angegeben“ (dann gilt die Voreinstellung).
type Eintrag struct {
	Type          string   `json:"type"`
	From          *Punkt   `json:"from,omitempty"`
	To            *Punkt   `json:"to,omitempty"`
	At            *Punkt   `json:"at,omitempty"`
	Text          *string  `json:"text,omitempty"`
	Color         *string  `json:"color,omitempty"`
	Fill          *string  `json:"fill,omitempty"`
	Width         *float64 `json:"width,omitempty"`
	Size          *float64 `json:"size,omitempty"`
	Radius        *float64 `json:"radius,omitempty"`
	Font          *string  `json:"font,omitempty"`
	Bold          *bool    `json:"bold,omitempty"`
	Italic        *bool    `json:"italic,omitempty"`
	Underline     *bool    `json:"underline,omitempty"`
	Strikethrough *bool    `json:"strikethrough,omitempty"`
	Align         *string  `json:"align,omitempty"`
}

// Punkt ist ein Punkt [x, y] im sichtbaren Bild; genau zwei Zahlen (encoding/json füllte ein kürzeres Array still mit
// Nullen auf).
type Punkt [2]float64

func (p *Punkt) UnmarshalJSON(b []byte) error {
	var l []float64
	if err := json.Unmarshal(b, &l); err != nil {
		return err
	}
	if len(l) != 2 {
		return fmt.Errorf("a point needs exactly two numbers [x, y], got %d", len(l))
	}
	*p = Punkt{l[0], l[1]}
	return nil
}

// erlaubt sind die Schlüssel je Typ; Pflicht die, ohne die ein Eintrag nicht geht.
var (
	erlaubt = map[string][]string{
		"arrow":    {"from", "to", "color", "width"},
		"rect":     {"from", "to", "color", "fill", "width"},
		"text":     {"at", "text", "size", "color", "fill", "font", "bold", "italic", "underline", "strikethrough", "align"},
		"step":     {"at", "color", "radius"},
		"redact":   {"from", "to"},
		"pixelate": {"from", "to"},
	}
	pflicht = map[string][]string{
		"arrow": {"from", "to"}, "rect": {"from", "to"}, "text": {"at", "text"}, "step": {"at"},
		"redact": {"from", "to"}, "pixelate": {"from", "to"},
	}
)

// Lies liest die Liste aus r: ein Objekt mit genau dem Schlüssel "annotations" (ein Array). Unbekannte Schlüssel und
// Typen, fehlende Pflichtwerte und Schlüssel, die zum Typ nicht passen, sind Fehler — ein Tippfehler soll nicht still
// wirken.
func Lies(r io.Reader) ([]Eintrag, error) {
	b, err := io.ReadAll(io.LimitReader(r, MaxBytes+1))
	if err != nil {
		return nil, &Fehler{-1, "cannot read: " + err.Error()}
	}
	if len(b) > MaxBytes {
		return nil, &Fehler{-1, fmt.Sprintf("larger than %d bytes", MaxBytes)}
	}
	var huelle struct {
		Annotations []json.RawMessage `json:"annotations"`
	}
	if err := streng(b, &huelle); err != nil {
		return nil, &Fehler{-1, err.Error()}
	}
	if huelle.Annotations == nil {
		return nil, &Fehler{-1, `missing "annotations" array`}
	}
	if len(huelle.Annotations) > MaxEintraege {
		return nil, &Fehler{-1, fmt.Sprintf("more than %d annotations", MaxEintraege)}
	}
	es := make([]Eintrag, 0, len(huelle.Annotations))
	for i, roh := range huelle.Annotations {
		var e Eintrag
		if err := streng(roh, &e); err != nil {
			return nil, &Fehler{i, err.Error()}
		}
		var schluessel map[string]json.RawMessage
		_ = json.Unmarshal(roh, &schluessel) // gelang eben als Eintrag, ist also ein Objekt
		ok, bekannt := erlaubt[e.Type]
		if !bekannt {
			return nil, &Fehler{i, fmt.Sprintf("unknown type %q (arrow, rect, text, step, redact, pixelate)", e.Type)}
		}
		for k, v := range schluessel {
			if k == "type" {
				continue
			}
			if !enthaelt(ok, k) {
				return nil, &Fehler{i, fmt.Sprintf("key %q does not apply to type %q", k, e.Type)}
			}
			if string(bytes.TrimSpace(v)) == "null" {
				return nil, &Fehler{i, fmt.Sprintf("key %q must not be null", k)}
			}
		}
		for _, k := range pflicht[e.Type] {
			if _, da := schluessel[k]; !da {
				return nil, &Fehler{i, fmt.Sprintf("type %q needs key %q", e.Type, k)}
			}
		}
		es = append(es, e)
	}
	return es, nil
}

// streng dekodiert genau einen JSON-Wert ohne unbekannte Felder.
func streng(b []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %v", err)
	}
	if dec.More() {
		return errors.New("invalid JSON: more than one value")
	}
	if _, err := dec.Token(); err != io.EOF {
		return errors.New("invalid JSON: trailing data")
	}
	return nil
}

func enthaelt(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}

// Ergebnis zählt, was Wende angelegt hat.
type Ergebnis struct {
	Beschriftungen int // Pfeile, Kästen, Texte, Nummern
	Schwaerzungen  int // Schwärzen und Verpixeln
}

// Wende legt die Einträge der Reihe nach auf d. Koordinaten gelten im sichtbaren Bild (d.Groesse()) und müssen darin
// liegen (Rand eingeschlossen). Bei einem Fehler ist d teilweise geändert: Der Aufrufer speichert dann nicht (ADR-0030
// Nachtrag 10 Punkt 3).
func Wende(d *document.Dokument, es []Eintrag) (Ergebnis, error) {
	var erg Ergebnis
	st := canvas.StilMitFaktor(d.KantenFaktor())
	for i, e := range es {
		f := func(format string, a ...any) error { return &Fehler{i, fmt.Sprintf(format, a...)} }
		punkt := func(name string, p *Punkt) (document.Punkt, error) {
			g := d.Groesse()
			if !endlich(p[0]) || !endlich(p[1]) || p[0] < 0 || p[1] < 0 || p[0] > float64(g.X) || p[1] > float64(g.Y) {
				return document.Punkt{}, f("%q %v lies outside the visible image (0..%d, 0..%d)", name, *p, g.X, g.Y)
			}
			return d.ZuOriginal(document.Pt(p[0], p[1])), nil
		}
		farbe := func(name string, s *string, vor color.NRGBA, keineErlaubt bool) (color.NRGBA, error) {
			if s == nil {
				return vor, nil
			}
			if keineErlaubt && *s == "none" {
				return color.NRGBA{}, nil
			}
			c, ok := leseFarbe(*s)
			if !ok {
				if keineErlaubt {
					return c, f("%q must be #RRGGBB, #RRGGBBAA or \"none\"", name)
				}
				return c, f("%q must be #RRGGBB or #RRGGBBAA", name)
			}
			return c, nil
		}
		groesse := func(name string, v *float64, vor float64) (float64, error) {
			if v == nil {
				return vor, nil
			}
			if !endlich(*v) || *v <= 0 || *v > document.MaxGroesse {
				return 0, f("%q must be greater than 0 and at most %g", name, document.MaxGroesse)
			}
			return *v, nil
		}
		var o document.Objekt
		switch e.Type {
		case "arrow", "rect":
			w := canvas.WerkzeugPfeil
			if e.Type == "rect" {
				w = canvas.WerkzeugKasten
			}
			vorn, fuel := canvas.StandardPaar(w)
			a, err := punkt("from", e.From)
			if err != nil {
				return erg, err
			}
			b, err := punkt("to", e.To)
			if err != nil {
				return erg, err
			}
			if *e.From == *e.To {
				return erg, f(`"from" and "to" must differ`)
			}
			c, err := farbe("color", e.Color, vorn, false)
			if err != nil {
				return erg, err
			}
			breite, err := groesse("width", e.Width, st.Breite)
			if err != nil {
				return erg, err
			}
			strich := document.Strich{Farbe: c, Breite: breite}
			if e.Type == "arrow" {
				o = document.GeraderPfeil(a, b, strich)
				break
			}
			fuellung, err := farbe("fill", e.Fill, fuel, true)
			if err != nil {
				return erg, err
			}
			o = document.Kasten{Bereich: document.R(a, b), Strich: strich, Fuellung: fuellung, Eckradius: st.Eckradius}
		case "text":
			vorn, fuel := canvas.StandardPaar(canvas.WerkzeugText)
			p, err := punkt("at", e.At)
			if err != nil {
				return erg, err
			}
			if strings.TrimSpace(*e.Text) == "" {
				return erg, f(`"text" must not be empty`)
			}
			c, err := farbe("color", e.Color, vorn, false)
			if err != nil {
				return erg, err
			}
			fuellung, err := farbe("fill", e.Fill, fuel, true)
			if err != nil {
				return erg, err
			}
			grad, err := groesse("size", e.Size, st.Schriftgrad)
			if err != nil {
				return erg, err
			}
			t := document.Text{Ort: p, Inhalt: *e.Text, Schriftgrad: grad, Farbe: c, Hintergrund: fuellung}
			if e.Font != nil {
				t.Schrift = *e.Font
			}
			t.Fett, t.Kursiv = wahr(e.Bold), wahr(e.Italic)
			t.Unterstrichen, t.Durchgestrichen = wahr(e.Underline), wahr(e.Strikethrough)
			if e.Align != nil {
				switch *e.Align {
				case "left":
				case "center":
					t.Ausrichtung = document.AusrichtungMitte
				case "right":
					t.Ausrichtung = document.AusrichtungRechts
				default:
					return erg, f(`"align" must be "left", "center" or "right"`)
				}
			}
			o = t
		case "step":
			vorn, _ := canvas.StandardPaar(canvas.WerkzeugNummer)
			p, err := punkt("at", e.At)
			if err != nil {
				return erg, err
			}
			c, err := farbe("color", e.Color, vorn, false)
			if err != nil {
				return erg, err
			}
			r, err := groesse("radius", e.Radius, st.Radius)
			if err != nil {
				return erg, err
			}
			n := d.NaechsteNummer()
			if n > document.MaxNummer {
				return erg, f("no step number left (at most %d)", document.MaxNummer)
			}
			o = document.Nummer{Mitte: p, Wert: n, Radius: r, Farbe: c}
		case "redact", "pixelate":
			for _, q := range []struct {
				name string
				p    *Punkt
			}{{"from", e.From}, {"to", e.To}} {
				if _, err := punkt(q.name, q.p); err != nil {
					return erg, err
				}
			}
			r := document.R(document.Pt(e.From[0], e.From[1]), document.Pt(e.To[0], e.To[1]))
			px, sw, _, ok := canvas.SchwaerzBereich(d, r, e.Type == "pixelate")
			if !ok || r.Breite() <= 0 || r.Hoehe() <= 0 {
				return erg, f("the area is empty")
			}
			if err := d.Schwaerzen(px, sw); err != nil {
				return erg, f("cannot %s this area", e.Type)
			}
			erg.Schwaerzungen++
			continue
		}
		if _, err := d.Hinzufuegen(o); err != nil {
			return erg, f("rejected by the document (values or limits out of range)")
		}
		erg.Beschriftungen++
	}
	return erg, nil
}

func endlich(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

func wahr(b *bool) bool { return b != nil && *b }

// leseFarbe liest #RRGGBB (deckend) oder #RRGGBBAA.
func leseFarbe(s string) (color.NRGBA, bool) {
	if !strings.HasPrefix(s, "#") || (len(s) != 7 && len(s) != 9) {
		return color.NRGBA{}, false
	}
	return canvas.LeseHex(s, color.NRGBA{A: 255})
}
