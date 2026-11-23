// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

// Freie Farbwahl mit Deckkraft, zuletzt benutzte Farben und Pipette (ADR-0021 Nachtrag 9, ADR-0031 Nachtrag 5). Die
// Oberfläche liefert nur Lagen in der Farbfläche und den Reglern (0 bis 1) und getippten Text; was daraus wird, steht
// hier ohne Gio.

import (
	"image/color"
	"math"
	"strconv"
	"strings"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ZuletztGrenze ist die Zahl der zuletzt benutzten Farben, die die Leiste zeigt.
const ZuletztGrenze = 3

// HSV ist eine Farbe als Farbton H (0 bis unter 360), Sättigung S und Hellwert V (je 0 bis 1). Nur Bedienung:
// gespeichert wird RGB.
type HSV struct{ H, S, V float64 }

// HSVAus rechnet eine Farbe nach HSV um (Deckkraft fällt weg). Grau hat den Farbton 0.
func HSVAus(c color.NRGBA) HSV {
	r, g, b := float64(c.R)/255, float64(c.G)/255, float64(c.B)/255
	mx, mn := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	d := mx - mn
	h := 0.0
	switch {
	case d == 0:
	case mx == r:
		h = 60 * math.Mod((g-b)/d, 6)
	case mx == g:
		h = 60 * ((b-r)/d + 2)
	default:
		h = 60 * ((r-g)/d + 4)
	}
	if h < 0 {
		h += 360
	}
	s := 0.0
	if mx > 0 {
		s = d / mx
	}
	return HSV{H: h, S: s, V: mx}
}

// Farbe liefert die Farbe zu h mit der Deckkraft a. Werte außerhalb werden begrenzt, der Farbton läuft im Kreis.
func (h HSV) Farbe(a uint8) color.NRGBA {
	ton := math.Mod(h.H, 360)
	if ton < 0 {
		ton += 360
	}
	s, v := klemme01(h.S), klemme01(h.V)
	c := v * s
	x := c * (1 - math.Abs(math.Mod(ton/60, 2)-1))
	m := v - c
	var r, g, b float64
	switch {
	case ton < 60:
		r, g, b = c, x, 0
	case ton < 120:
		r, g, b = x, c, 0
	case ton < 180:
		r, g, b = 0, c, x
	case ton < 240:
		r, g, b = 0, x, c
	case ton < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	k := func(f float64) uint8 { return uint8(math.Round((f + m) * 255)) }
	return color.NRGBA{R: k(r), G: k(g), B: k(b), A: a}
}

func klemme01(f float64) float64 { return math.Max(0, math.Min(1, f)) }

// LeseHex liest eine Farbe aus 3, 6 oder 8 Hex-Ziffern, mit oder ohne „#“ und Leerraum außen. Bei 3 und 6 Ziffern
// gilt die Deckkraft von alt, bei 8 die angegebene (ADR-0021 Nachtrag 9 Punkt 6).
func LeseHex(text string, alt color.NRGBA) (color.NRGBA, bool) {
	t := strings.TrimPrefix(strings.TrimSpace(text), "#")
	switch len(t) {
	case 3:
		t = string([]byte{t[0], t[0], t[1], t[1], t[2], t[2]})
	case 6, 8:
	default:
		return color.NRGBA{}, false
	}
	n, err := strconv.ParseUint(t, 16, 32)
	if err != nil {
		return color.NRGBA{}, false
	}
	if len(t) == 6 {
		return color.NRGBA{R: uint8(n >> 16), G: uint8(n >> 8), B: uint8(n), A: alt.A}, true
	}
	return color.NRGBA{R: uint8(n >> 24), G: uint8(n >> 16), B: uint8(n >> 8), A: uint8(n)}, true
}

// HexText ist die Anzeige einer Farbe: #RRGGBB, bei Deckkraft unter 100 % #RRGGBBAA.
func HexText(c color.NRGBA) string {
	if c.A == 255 {
		return "#" + strings.ToUpper(strconv.FormatUint(uint64(c.R)<<16|uint64(c.G)<<8|uint64(c.B)|1<<24, 16)[1:])
	}
	return "#" + strings.ToUpper(strconv.FormatUint(uint64(c.R)<<24|uint64(c.G)<<16|uint64(c.B)<<8|uint64(c.A)|1<<32, 16)[1:])
}

// LeseDeckkraft liest eine ganze Zahl 0 bis 100, mit oder ohne „%“, und liefert das Alpha dazu.
func LeseDeckkraft(text string) (uint8, bool) {
	t := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(text), "%"))
	p, err := strconv.Atoi(t)
	if err != nil || p < 0 || p > 100 {
		return 0, false
	}
	return uint8(math.Round(float64(p) * 255 / 100)), true
}

// AktuelleFarbe ist die Farbe des aktiven Felds, die die Leiste zeigt: die des Objekts, auf das die Farbwahl wirkt,
// sonst die des Werkzeugs.
func (s *Sitzung) AktuelleFarbe() color.NRGBA { return s.Farbe(s.AktivesFeld()) }

// FarbeHSV ist die aktuelle Farbe als HSV. Solange die Farbe zum gemerkten Wert passt, gilt der gemerkte Farbton und
// die gemerkte Sättigung: Grau, Schwarz und Weiß verlieren beim Ziehen den Ton nicht (ADR-0021 Nachtrag 9 Punkt 3).
func (s *Sitzung) FarbeHSV() HSV {
	c := s.AktuelleFarbe()
	if s.hsvGemerkt && s.hsv.Farbe(c.A) == c {
		return s.hsv
	}
	return HSVAus(c)
}

// Zuletzt sind die zuletzt benutzten Farben, die jüngste zuerst (höchstens ZuletztGrenze).
func (s *Sitzung) Zuletzt() []color.NRGBA { return append([]color.NRGBA(nil), s.zuletzt...) }

// merkeZuletzt setzt c vorn in die zuletzt benutzten Farben, wenn sie keine Beschriftungsfarbe ist.
func (s *Sitzung) merkeZuletzt(c color.NRGBA) {
	for _, b := range Beschriftungsfarben {
		if b == c {
			return
		}
	}
	neu := []color.NRGBA{c}
	for _, z := range s.zuletzt {
		if z != c && len(neu) < ZuletztGrenze {
			neu = append(neu, z)
		}
	}
	s.zuletzt = neu
}

// setzeEigeneFarbe setzt eine Farbe, die nicht aus einem Farbfeld kommt (Hex, Deckkraft, Pipette): ein Schritt wie
// SetzeFarbe, dazu in die zuletzt benutzten Farben. Liefert, ob die Farbe übernommen wurde (nicht gesperrt).
func (s *Sitzung) setzeEigeneFarbe(c color.NRGBA) bool {
	if s.Zieht() || s.gesperrt() {
		return false
	}
	s.SetzeFarbe(c)
	s.merkeZuletzt(c)
	s.hsvGemerkt = false
	return true
}

// SetzeHex übernimmt eine Hex-Eingabe (LeseHex); false: ungültig oder gesperrt, nichts geändert. Ist die aktuelle
// Farbe ganz durchsichtig (Füllung „keine“), wird eine Eingabe ohne Deckkraft deckend (ADR-0021 Nachtrag 11).
func (s *Sitzung) SetzeHex(text string) bool {
	alt := s.AktuelleFarbe()
	if alt.A == 0 {
		alt.A = 255
	}
	c, ok := LeseHex(text, alt)
	return ok && s.setzeEigeneFarbe(c)
}

// SetzeDeckkraftText übernimmt eine Deckkraft-Eingabe (LeseDeckkraft); false: ungültig oder gesperrt.
func (s *Sitzung) SetzeDeckkraftText(text string) bool {
	a, ok := LeseDeckkraft(text)
	if !ok {
		return false
	}
	c := s.AktuelleFarbe()
	c.A = a
	return s.setzeEigeneFarbe(c)
}

// --- Ziehen in Farbfläche und Reglern ---

// FarbZugLaeuft meldet, ob in der Farbfläche oder an einem Regler gezogen wird.
func (s *Sitzung) FarbZugLaeuft() bool { return s.farbZug }

// BeginneFarbZug beginnt das Ziehen in der Farbfläche oder an einem Regler: Das gewählte Objekt färbt sich bis zum
// Loslassen mit und wird dann ein Rückgängig-Schritt (ADR-0021 Nachtrag 9 Punkt 2). Während der Texteingabe gehört
// das Färben zur Eingabe. Liefert false, wenn gerade nicht gezogen werden darf.
func (s *Sitzung) BeginneFarbZug() bool {
	if s.Zieht() || s.gesperrt() {
		return false
	}
	s.hsv, s.hsvGemerkt = s.FarbeHSV(), true
	s.farbZug, s.farbZugObjekt, s.farbZugText = true, false, nil
	if s.zug == zugText {
		if o, ok := s.D.Objekt(s.textID); ok {
			s.farbZugText = o
		}
	} else if o, ok := s.farbObjekt(); ok && s.D.BeginneZug(o.Kennung()) == nil {
		s.farbZugObjekt, s.farbZugVor = true, s.D.Stand()
	}
	return true
}

// FarbZugHSV setzt während des Ziehens die Farbe aus HSV und Deckkraft a.
func (s *Sitzung) FarbZugHSV(h HSV, a uint8) {
	if !s.farbZug {
		return
	}
	s.hsv, s.hsvGemerkt = h, true
	c := h.Farbe(a)
	f := s.AktivesFeld()
	s.merkePaarFarbe(s.zielWerkzeug(), f, c)
	switch {
	case s.zug == zugText:
		if o, ok := s.D.Objekt(s.textID); ok {
			_ = s.D.ZugSetze(UmgefaerbtIn(o, f, c))
		}
	case s.farbZugObjekt:
		if o, ok := s.D.Objekt(s.Auswahl); ok {
			_ = s.D.ZugSetze(UmgefaerbtIn(o, f, c))
		}
	}
}

// BeendeFarbZug schließt das Ziehen ab: ein Rückgängig-Schritt am Objekt (keiner, wenn die Farbe gleich blieb), die
// Farbe kommt in die zuletzt benutzten.
func (s *Sitzung) BeendeFarbZug() {
	if !s.farbZug {
		return
	}
	s.farbZug = false
	if s.farbZugObjekt {
		s.farbZugObjekt = false
		_ = s.D.BeendeZug()
		s.merkeAenderung(s.farbZugVor)
	}
	s.merkeZuletzt(s.AktuelleFarbe())
}

// BrecheFarbZugAb stellt das Objekt wie vor dem Ziehen her (Zeiger-Cancel); die Farbe des Werkzeugs bleibt die zuletzt
// gezogene.
func (s *Sitzung) BrecheFarbZugAb() {
	if !s.farbZug {
		return
	}
	s.farbZug = false
	if s.farbZugObjekt {
		s.farbZugObjekt = false
		_ = s.D.BrecheZugAb()
	}
	if s.zug == zugText && s.farbZugText != nil { // in der Texteingabe: Text wie vor dem Zug (Verify V-3)
		_ = s.D.ZugSetze(s.farbZugText)
	}
	s.farbZugText = nil
}

// --- Pipette ---

// PipetteLaeuft meldet, ob der nächste Klick ins Bild eine Farbe aufnimmt.
func (s *Sitzung) PipetteLaeuft() bool { return s.pipette }

// SchaltePipette schaltet die Pipette ein oder aus (nicht während eines Zugs oder des Speicherns). Der Klick ins Bild
// bestätigt eine laufende Texteingabe, wie jeder Klick auf die Leinwand.
func (s *Sitzung) SchaltePipette() bool {
	if s.Zieht() || s.gesperrt() {
		return false
	}
	s.pipette = !s.pipette
	return true
}

// pipetteKlick nimmt die Farbe am Fensterpunkt p auf (Original mit Schwärzungen, ohne Beschriftungen, deckend) und
// beendet die Pipette. Den Bildpunkt bestimmt dieselbe Umrechnung wie die Fußzeile (ZeigerOrt; beim Zuschneiden die
// Vollansicht, Sitzungs-QA A-6). Neben dem Bild oder auf einem ganz durchsichtigen Bildpunkt endet sie ohne Wirkung:
// Dessen Farbe sieht niemand (A-9, ADR-0021 Nachtrag 9 Punkt 5).
func (s *Sitzung) pipetteKlick(p document.Punkt) {
	s.pipette = false
	ort, ok := s.ZeigerOrt(s.Abb.ZuAnsicht(p))
	o := s.D.OriginalBild()
	if !ok || !ort.In(o.Rect) {
		return
	}
	c := o.NRGBAAt(ort.X, ort.Y)
	if c.A == 0 {
		return
	}
	c.A = 255
	s.TextBestaetigen()
	s.setzeEigeneFarbe(c)
}
