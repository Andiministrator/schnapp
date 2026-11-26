// SPDX-License-Identifier: GPL-3.0-or-later

// Package annotate liest eine Liste von Beschriftungen und Schwärzungen (JSON, ADR-0030 Nachtrag 10) und legt sie auf
// ein Dokument, ohne Fenster: für `schnapp --annotate=SPEC DATEI` und das MCP-Werkzeug `annotate`. Die Werte, die eine
// Liste offen lässt, und die Abbildung eines Schwärzungsrahmens aufs Original kommen aus canvas, damit Kommandozeile
// und Editor gleich rechnen. Fehlertexte sind englisch (sie gehen auch an Agenten, ADR-0030 Nachtrag 5 Punkt 6) und
// nennen den Index des Eintrags.
package annotate

import (
	"bytes"
	"context"
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

// LeseFehler: Die Liste ließ sich nicht lesen (Ordner, E/A-Fehler auf stdin) — kein Fehler in der Liste, sondern einer
// der Umgebung (Exit 1 statt 2, Kritik #202 B-7). Err ist die Ursache.
type LeseFehler struct{ Err error }

func (l *LeseFehler) Error() string { return "annotation spec: cannot read: " + l.Err.Error() }
func (l *LeseFehler) Unwrap() error { return l.Err }

// Eintrag ist ein gelesener Eintrag der Liste. Zeiger sind „nicht angegeben“ (dann gilt die Voreinstellung).
type Eintrag struct {
	Type          string   `json:"type"`
	From          *Punkt   `json:"from,omitempty"`
	To            *Punkt   `json:"to,omitempty"`
	At            *Punkt   `json:"at,omitempty"`
	Tip           *Punkt   `json:"tip,omitempty"`
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
	StartHead     *string  `json:"start_head,omitempty"` // arrow: head at "from" (default none; ADR-0021 Nachtrag 20)
	EndHead       *string  `json:"end_head,omitempty"`   // arrow: head at "to" (default triangle)
	LineStyle     *string  `json:"line_style,omitempty"` // arrow, rect: solid, dashed, dotted (ADR-0021 Nachtrag 25)
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
		"arrow":    {"from", "to", "color", "width", "start_head", "end_head", "line_style"},
		"rect":     {"from", "to", "color", "fill", "width", "line_style"},
		"text":     {"at", "text", "size", "color", "fill", "font", "bold", "italic", "underline", "strikethrough", "align", "tip"},
		"step":     {"at", "color", "radius"},
		"redact":   {"from", "to"},
		"pixelate": {"from", "to"},
	}
	pflicht = map[string][]string{
		"arrow": {"from", "to"}, "rect": {"from", "to"}, "text": {"at", "text"}, "step": {"at"},
		"redact": {"from", "to"}, "pixelate": {"from", "to"},
	}
)

// Lies liest die Liste aus r: ein Objekt mit genau dem Schlüssel "annotations" (ein nicht leeres Array). Unbekannte
// Schlüssel und Typen, fehlende Pflichtwerte, Schlüssel, die zum Typ nicht passen, und doppelte Schlüssel sind Fehler —
// ein Tippfehler soll nicht still wirken. Schlüssel gelten nur in genau dieser Schreibung (encoding/json nähme sonst
// "ANNOTATIONS" an und ließe bei doppelten den letzten still gewinnen, Kritik #202 B-6). Eine leere Liste ist ein
// Fehler: Sie speicherte sonst ohne Änderung eine neue Aufnahme (B-5). Lässt sich r nicht lesen, kommt *LeseFehler.
func Lies(r io.Reader) ([]Eintrag, error) {
	b, err := io.ReadAll(io.LimitReader(r, MaxBytes+1))
	if err != nil {
		return nil, &LeseFehler{err}
	}
	if len(b) > MaxBytes {
		return nil, &Fehler{-1, fmt.Sprintf("larger than %d bytes", MaxBytes)}
	}
	var huelle map[string]json.RawMessage
	if err := streng(b, &huelle); err != nil {
		return nil, &Fehler{-1, err.Error()}
	}
	if err := doppelte(b, false); err != nil { // die Einträge prüft die Schleife, mit ihrem Index
		return nil, &Fehler{-1, err.Error()}
	}
	for k := range huelle {
		if k != "annotations" {
			return nil, &Fehler{-1, fmt.Sprintf(`invalid JSON: unknown key %q (only "annotations")`, k)}
		}
	}
	var liste []json.RawMessage
	if roh, da := huelle["annotations"]; da {
		if err := json.Unmarshal(roh, &liste); err != nil {
			return nil, &Fehler{-1, `"annotations" must be an array`}
		}
	}
	if liste == nil {
		return nil, &Fehler{-1, `missing "annotations" array`}
	}
	if len(liste) == 0 {
		return nil, &Fehler{-1, `"annotations" is empty: nothing to do`}
	}
	if len(liste) > MaxEintraege {
		return nil, &Fehler{-1, fmt.Sprintf("more than %d annotations", MaxEintraege)}
	}
	es := make([]Eintrag, 0, len(liste))
	for i, roh := range liste {
		var e Eintrag
		if err := streng(roh, &e); err != nil {
			return nil, &Fehler{i, err.Error()}
		}
		if err := KeineDoppelten(roh); err != nil {
			return nil, &Fehler{i, err.Error()}
		}
		var schluessel map[string]json.RawMessage
		_ = json.Unmarshal(roh, &schluessel) // gelang eben als Eintrag, ist also ein Objekt
		if _, da := schluessel["type"]; !da {
			return nil, &Fehler{i, `missing key "type"`}
		}
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

// KeineDoppelten meldet einen Fehler, wenn ein Objekt im JSON-Wert b (auf jeder Ebene) einen Schlüssel zweimal trägt.
// encoding/json ließe sonst still den letzten gewinnen (Kritik #202 B-6). b muss gültiges JSON sein (vorher dekodiert):
// Die Tiefe ist dann durch die Grenze von encoding/json beschränkt.
func KeineDoppelten(b []byte) error { return doppelte(b, true) }

// doppelte ist KeineDoppelten; tief = false prüft nur das äußerste Objekt.
func doppelte(b []byte, tief bool) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	return doppelteWert(dec, tief, true)
}

func doppelteWert(dec *json.Decoder, tief, pruefen bool) error {
	t, err := dec.Token()
	if err != nil {
		return fmt.Errorf("invalid JSON: %v", err)
	}
	d, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	gesehen := map[string]bool{}
	for dec.More() {
		if d == '{' {
			k, err := dec.Token()
			if err != nil {
				return fmt.Errorf("invalid JSON: %v", err)
			}
			s, _ := k.(string)
			if pruefen && gesehen[s] {
				return fmt.Errorf("invalid JSON: duplicate key %q", s)
			}
			gesehen[s] = true
		}
		if err := doppelteWert(dec, tief, tief); err != nil {
			return err
		}
	}
	if _, err := dec.Token(); err != nil { // schließende Klammer
		return fmt.Errorf("invalid JSON: %v", err)
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

// imBild meldet, ob p im sichtbaren Bild von d liegt (Rand eingeschlossen).
func imBild(d *document.Dokument, p *Punkt) bool {
	g := d.Groesse()
	return endlich(p[0]) && endlich(p[1]) && p[0] >= 0 && p[1] >= 0 && p[0] <= float64(g.X) && p[1] <= float64(g.Y)
}

// pruefeSchwaerzFlaeche summiert vorab die Fläche aller Schwärzungen und Verpixelungen der Liste im Original und lehnt
// die Liste ab, wenn sie mehr Pixelkopien in den Rückgängig-Verlauf legte, als er ohne Verwerfen aufnimmt (4 Bytes je
// Pixel ≤ d.VerlaufPlatz(), derselbe Maßstab wie beim Anwenden einer Rettungskopie, F-412). Das begrenzt zugleich die
// Rechenzeit (Kritik #202 B-3: 10 000 ganzflächige Verpixelungen dauerten eine Viertelstunde). Einträge mit ungültigen
// Punkten oder leerem Bereich überspringt sie: Deren Fehler meldet Wende in der Reihenfolge der Liste.
func pruefeSchwaerzFlaeche(d *document.Dokument, es []Eintrag) error {
	platz := d.VerlaufPlatz()
	var bedarf int64
	for i, e := range es {
		if e.Type != "redact" && e.Type != "pixelate" || e.From == nil || e.To == nil || !imBild(d, e.From) || !imBild(d, e.To) {
			continue
		}
		r := document.R(document.Pt(e.From[0], e.From[1]), document.Pt(e.To[0], e.To[1]))
		px, sw, _, ok := canvas.SchwaerzBereich(d, r, e.Type == "pixelate")
		if !ok {
			continue
		}
		q, err := d.PruefeSchwaerzung(px, sw)
		if err != nil {
			continue
		}
		if bedarf += 4 * int64(q.Dx()) * int64(q.Dy()); bedarf > platz { // q liegt im Bild: kein Überlauf
			return &Fehler{i, fmt.Sprintf("the redacted and pixelated areas of this list together exceed %d megapixels; "+
				"use several lists", platz/4/1_000_000)}
		}
	}
	return nil
}

// Wende legt die Einträge der Reihe nach auf d. Koordinaten gelten im sichtbaren Bild (d.Groesse()) und müssen darin
// liegen (Rand eingeschlossen); der Fehlertext nennt die Größe des Bilds nicht (ein Agent soll über eine fremde Datei
// nichts erfahren, Kritik #202 Anmerkung). Vorab prüft Wende das Flächenbudget der Schwärzungen
// (pruefeSchwaerzFlaeche); zwischen zwei Einträgen bricht es ab, sobald ctx endet (dann ctx.Err()). Bei einem Fehler
// ist d teilweise geändert: Der Aufrufer speichert dann nicht (ADR-0030 Nachtrag 10 Punkt 3).
func Wende(ctx context.Context, d *document.Dokument, es []Eintrag) (Ergebnis, error) {
	var erg Ergebnis
	if err := pruefeSchwaerzFlaeche(d, es); err != nil {
		return erg, err
	}
	st := canvas.StilMitFaktor(d.KantenFaktor())
	for i, e := range es {
		if err := ctx.Err(); err != nil {
			return erg, err
		}
		f := func(format string, a ...any) error { return &Fehler{i, fmt.Sprintf(format, a...)} }
		punkt := func(name string, p *Punkt) (document.Punkt, error) {
			if !imBild(d, p) {
				return document.Punkt{}, f("%q %v lies outside the visible image", name, *p)
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
			if e.LineStyle != nil {
				art, ok := document.StrichartAus(*e.LineStyle)
				if !ok {
					return erg, f(`"line_style" must be one of %s`, namenListe(document.StrichartNamen()))
				}
				strich.Art = art
			}
			if e.Type == "arrow" {
				pf := document.GeraderPfeil(a, b, strich)
				anfang, ende := document.KopfKeine, document.KopfDreieck
				for _, k := range []struct {
					name string
					wert *string
					ziel *document.Kopfart
				}{{"start_head", e.StartHead, &anfang}, {"end_head", e.EndHead, &ende}} {
					if k.wert == nil {
						continue
					}
					art, ok := document.KopfartAus(*k.wert)
					if !ok {
						return erg, f(`"%s" must be one of %s`, k.name, kopfListe())
					}
					*k.ziel = art
				}
				pf = pf.MitKoepfen(anfang, ende)
				o = pf
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
			if e.Tip != nil { // Sprechblase (ADR-0021 Nachtrag 13 Punkt 8): gleiche Prüfung wie "at"
				sp, err := punkt("tip", e.Tip)
				if err != nil {
					return erg, err
				}
				t.Spitze = &sp
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

// kopfListe nennt die Namen der Spitzenarten für Fehlertexte: "triangle", "narrow", … or "none".
func kopfListe() string { return namenListe(document.KopfNamen()) }

// namenListe ist die Liste der Namen n für Fehlertexte: "a", "b" or "c".
func namenListe(n []string) string {
	q := make([]string, len(n))
	for i, x := range n {
		q[i] = `"` + x + `"`
	}
	return strings.Join(q[:len(q)-1], ", ") + " or " + q[len(q)-1]
}
