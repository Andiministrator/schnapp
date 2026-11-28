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
	"github.com/Andiministrator/schnapp/internal/fonts"
)

// FormatVersion ist die höchste Version des SVG-Blocks, die dieser Stand liest und schreibt (ADR-0006, ADR-0008).
// Ältere Versionen müssen lesbar bleiben. Geschrieben wird die kleinste Version, die den Inhalt trägt (schreibVersion):
// 2 nur mit einer Lupe (ADR-0033 Punkt 5), einer Sprechblasen-Spitze (ADR-0021 Nachtrag 13 Punkt 7), einem Pfeil,
// dessen Spitzen von der Voreinstellung abweichen (ADR-0021 Nachtrag 20), einem gemusterten Strich (ADR-0021
// Nachtrag 25) oder einem Schatten (ADR-0021 Nachtrag 26), sonst 1 — Dateien ohne all das bleiben für ältere Leser lesbar.
const FormatVersion = 2

// versionLupe ist die erste Formatversion mit dem Objekttyp „lupe“ (ADR-0033 Punkt 5); versionSpitze die erste mit
// dem Attribut s:spitze am Text (ADR-0021 Nachtrag 13 Punkt 7); versionPfeilKoepfe die erste mit s:ende/s:anfang am
// Pfeil (ADR-0021 Nachtrag 20); versionStrichart die erste mit s:strichart an Pfeil, Kasten und Lupe (ADR-0021
// Nachtrag 25); versionSchatten die erste mit s:schatten an jeder Ebene (ADR-0021 Nachtrag 26 Punkt 6); versionGruppe
// die erste mit s:gruppe (ADR-0044 Punkt 6). Alle sind
// Version 2.
const (
	versionLupe        = 2
	versionSpitze      = 2
	versionPfeilKoepfe = 2
	versionStrichart   = 2
	versionSchatten    = 2
	versionGruppe      = 2
)

// schreibVersion ist die Version, die der Schreiber für die Objekte objekte angibt.
func schreibVersion(objekte []document.Objekt) int {
	gruppen := wirksameGruppen(objekte)
	for _, o := range objekte {
		if o.Ebenenwerte().Schatten.An {
			return versionSchatten
		}
		if gruppen[o.Ebenenwerte().Gruppe] {
			return versionGruppe
		}
		switch v := o.(type) {
		case document.Lupe:
			return versionLupe
		case document.Text:
			if v.Spitze != nil {
				return versionSpitze
			}
		case document.Pfeil:
			if v.Ende != document.KopfDreieck || v.KopfAnfang() != document.KopfKeine {
				return versionPfeilKoepfe
			}
			if v.Strich.Art != document.StrichVoll {
				return versionStrichart
			}
		case document.Kasten:
			if v.Strich.Art != document.StrichVoll {
				return versionStrichart
			}
		}
	}
	return 1
}

// wirksameGruppen sind die Gruppen mit mindestens zwei Mitgliedern (ADR-0044 Punkt 1): Nur sie stehen in der Datei;
// die Kennung eines einzelnen Restmitglieds bleibt weg und hebt die Formatversion nicht (F-962).
func wirksameGruppen(objekte []document.Objekt) map[int]bool {
	n := map[int]int{}
	for _, o := range objekte {
		if g := o.Ebenenwerte().Gruppe; g != 0 {
			n[g]++
		}
	}
	r := map[int]bool{}
	for g, k := range n {
		if k >= 2 {
			r[g] = true
		}
	}
	return r
}

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
	// lupenBilder: schon eingebettete Bilder der Lupen und ihre id (ADR-0033 Nachtrag 5), nur im Export.
	lupenBilder map[lupenSchluessel]string
	// ohneText: Text ohne `<text>` und Striche schreiben (nur Schild bzw. Dreieck), nur im Export: eine Sprechblase,
	// von der nur das Dreieck sichtbar ist (F-692). nil schreibt jeden Text ganz.
	ohneText func(document.Text) bool
	// schattenNr zählt die Filter der Schatten, schattenID ist der Filter des Objekts, das gerade geschrieben wird
	// (ADR-0021 Nachtrag 26 Punkt 5); ebene setzt ihn als filter-Attribut.
	schattenNr int
	schattenID string
	// gruppen sind die wirksamen Gruppen des Dokuments (wirksameGruppen), gesetzt in objekte.
	gruppen map[int]bool
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

// kastenOhneFlaeche schreibt einen Kasten mit Breite oder Höhe 0 (etwa ein Griff, der eine Kante auf die gegenüberliegende
// zieht): Der Renderer zeichnet ihn als Balken (document.EckigerRahmen, Strichbreite über die Länge), ein `<rect>` mit Maß
// 0 aber ist im SVG unsichtbar (Kritik A-5). Darum trägt eine Gruppe die Daten und ein gefüllter Pfad die Darstellung;
// die Daten (s:bereich usw.) sind dieselben wie beim Rechteck, der Leser unterscheidet die Elemente nicht.
func (s *schreiber) kastenOhneFlaeche(o document.Kasten) {
	s.WriteString(`<g`)
	s.attr("s:typ", "kasten")
	s.ebene(o.Ebenenwerte())
	s.attr("s:bereich", punkt(o.Bereich.Min)+" "+punkt(o.Bereich.Max))
	s.attr("s:strich", strich(o.Strich))
	s.attr("s:fuellung", farbeHex(o.Fuellung))
	s.attr("s:eckradius", zahl(o.Eckradius))
	s.strichart(o.Strich)
	s.WriteString(`>`)
	if o.Strich.Breite > 0 && o.Strich.Farbe.A > 0 {
		s.WriteString(`<path`)
		var pfad strings.Builder
		for _, r := range document.EckigerRahmen(o.Bereich, o.Strich.Breite) {
			pfad.WriteString("M" + punkt(r.Min) + " L" + punkt(document.Pt(r.Max.X, r.Min.Y)) + " L" + punkt(r.Max) +
				" L" + punkt(document.Pt(r.Min.X, r.Max.Y)) + " Z")
		}
		s.attr("d", pfad.String())
		s.farbeSVG("fill", o.Strich.Farbe)
		s.WriteString(`/>`)
	}
	s.WriteString(`</g>`)
}

// ebene schreibt die Angaben der Ebene (ADR-0032 Punkt 9, docs/format.md 8.5): die Daten nur, wo sie von der
// Voreinstellung abweichen, und für Betrachter display="none" bzw. opacity. Der Export lässt ausgeblendete Objekte
// vorher weg (behalten), dort steht nur opacity.
func (s *schreiber) ebene(e document.Ebene) {
	if e.Name != "" {
		s.attr("s:name", e.Name)
	}
	if e.Ausgeblendet {
		s.attr("s:ausgeblendet", "true")
		s.attr("display", "none")
	}
	if e.Gesperrt {
		s.attr("s:gesperrt", "true")
	}
	if p := e.Deckkraft(); p < 100 {
		s.attr("s:deckkraft", strconv.Itoa(p))
		s.attr("opacity", zahl(float64(p)/100))
	}
	if sch := e.Schatten; sch.An { // ADR-0021 Nachtrag 26 Punkt 6, docs/format.md 8.5
		s.attr("s:schatten", zahl(sch.Winkel)+" "+zahl(sch.Laenge)+" "+zahl(sch.Weich)+" "+farbeHex(sch.Farbe))
	}
	if s.gruppen[e.Gruppe] { // ADR-0044 Punkt 6, docs/format.md 8.5; nur wirksame (F-962)
		s.attr("s:gruppe", strconv.Itoa(e.Gruppe))
	}
	if s.schattenID != "" {
		s.attr("filter", "url(#"+s.schattenID+")")
		s.schattenID = ""
	}
}

// leseEbene übernimmt s:name, s:ausgeblendet, s:gesperrt, s:deckkraft und (ab Version 2) s:schatten und s:gruppe. Alle sind
// freiwillig: fehlend oder ungültig heißt kein Name, sichtbar, offen, deckend, kein Schatten — kein Schaden (wie
// s:schrift).
func leseEbene(a map[string]string, version int) document.Ebene {
	var e document.Ebene
	if n := a["name"]; document.EbenenNameGueltig(n) {
		e.Name = n
	}
	e.Ausgeblendet = a["ausgeblendet"] == "true"
	e.Gesperrt = a["gesperrt"] == "true"
	if p, ok := parseGanz(a["deckkraft"]); ok && p >= 0 && p <= 100 {
		e = e.MitDeckkraft(p)
	}
	if v, da := a["schatten"]; da && version >= versionSchatten {
		e.Schatten = leseSchatten(v)
	}
	if g, ok := parseGanz(a["gruppe"]); ok && g >= 1 && g <= document.MaxGruppe && version >= versionGruppe {
		e.Gruppe = g
	}
	return e
}

func strich(st document.Strich) string { return farbeHex(st.Farbe) + " " + zahl(st.Breite) }

// svgSchrift ist die Schrift der Beschriftungen im SVG: Renderer und Editor setzen Go Regular und Go Bold (Familie „Go“);
// ein Betrachter ohne diese Schrift fällt auf eine serifenlose Schrift zurück (F-108).
const svgSchrift = "Go, Go Regular, sans-serif"

// svgSchriftFuer ist font-family für einen Text der Familie name (leer = Go): die Familie als CSS-Zeichenkette vor Go
// und sans-serif (ADR-0021 Nachtrag 10 Punkt 8).
func svgSchriftFuer(name string) string {
	if name == "" {
		return svgSchrift
	}
	return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(name) + "', " + svgSchrift
}

// Stil und Ausrichtung als Wörter der Arbeitsdatei (docs/format.md 8.3).
func textStilText(t document.Text) string {
	var w []string
	for _, x := range []struct {
		an   bool
		wort string
	}{{t.Fett, "bold"}, {t.Kursiv, "italic"}, {t.Unterstrichen, "underline"}, {t.Durchgestrichen, "strike"}} {
		if x.an {
			w = append(w, x.wort)
		}
	}
	return strings.Join(w, " ")
}

func ausrichtungText(a document.Ausrichtung) string {
	switch a {
	case document.AusrichtungMitte:
		return "center"
	case document.AusrichtungRechts:
		return "right"
	}
	return ""
}

// textAnker liefert text-anchor und den Anteil des Zeilenvorschubs, um den das x einer Zeile vom Anfang aus nach rechts
// rückt (Mitte 0,5, rechts 1). Links: kein Attribut, x am Zeilenanfang. Eine Rechnung für Export und Arbeitsdatei.
func textAnker(a document.Ausrichtung) (string, float64) {
	switch a {
	case document.AusrichtungMitte:
		return "middle", 0.5
	case document.AusrichtungRechts:
		return "end", 1
	}
	return "", 0
}

// leseTextStil übernimmt s:schrift, s:stil und s:ausrichtung. Alle drei sind freiwillig: fehlend, ungültig oder
// unbekannt heißt Go, normal, links — kein Schaden (wie s:kante); unbekannte Stil-Wörter werden übergangen.
func leseTextStil(t *document.Text, a map[string]string) {
	if n := a["schrift"]; n != fonts.Go && fonts.NameGueltig(n) {
		t.Schrift = n
	}
	for _, w := range strings.Fields(a["stil"]) {
		switch w {
		case "bold":
			t.Fett = true
		case "italic":
			t.Kursiv = true
		case "underline":
			t.Unterstrichen = true
		case "strike":
			t.Durchgestrichen = true
		}
	}
	switch a["ausrichtung"] {
	case "center":
		t.Ausrichtung = document.AusrichtungMitte
	case "right":
		t.Ausrichtung = document.AusrichtungRechts
	}
}

// objekte schreibt die Objekte in Original-Koordinaten; mit behalten != nil nur die, für die behalten true liefert
// (Export: nur Objekte, die in die Ansicht reichen).
func (s *schreiber) objekte(d *document.Dokument, behalten func(document.Objekt) bool) {
	s.WriteString(`<g`)
	s.attr("s:ebene", "objekte")
	s.WriteString(`>`)
	s.gruppen = wirksameGruppen(d.ObjekteOhneZug())
	for _, o := range d.ObjekteOhneZug() { // ein laufender Zug zählt nicht (F-409)
		if behalten != nil && !behalten(o) {
			continue
		}
		s.schattenFilter(o)
		switch o := o.(type) {
		case document.Pfeil:
			s.WriteString(`<g`)
			s.attr("s:typ", "pfeil")
			s.ebene(o.Ebenenwerte())
			s.attr("s:a", punkt(o.A))
			s.attr("s:b", punkt(o.B))
			s.attr("s:k", punkt(o.K))
			s.attr("s:strich", strich(o.Strich))
			// Freiwillig, fehlen bei Voreinstellung (Dreieck bei B, keine bei A; docs/format.md 8.1); sonst Version 2.
			if o.Ende != document.KopfDreieck {
				s.attr("s:ende", o.Ende.Name())
			}
			if a := o.KopfAnfang(); a != document.KopfKeine {
				s.attr("s:anfang", a.Name())
			}
			s.strichart(o.Strich)
			s.WriteString(`>`)
			s.pfeilDarstellung(o)
			s.WriteString(`</g>`)
		case document.Kasten:
			b := o.Bereich
			if b.Breite() == 0 || b.Hoehe() == 0 {
				s.kastenOhneFlaeche(o)
				break
			}
			if o.Strich.Art != document.StrichVoll && !s.rundlauf {
				s.kastenGemustert(o)
				break
			}
			s.WriteString(`<rect`)
			s.attr("s:typ", "kasten")
			s.ebene(o.Ebenenwerte())
			s.attr("s:bereich", punkt(b.Min)+" "+punkt(b.Max))
			s.attr("s:strich", strich(o.Strich))
			s.attr("s:fuellung", farbeHex(o.Fuellung))
			s.attr("s:eckradius", zahl(o.Eckradius))
			s.strichart(o.Strich)
			s.attr("x", zahl(b.Min.X))
			s.attr("y", zahl(b.Min.Y))
			s.attr("width", zahl(b.Breite()))
			s.attr("height", zahl(b.Hoehe()))
			// Rundung wie der Renderer: gleiche Begrenzung (nie elliptisch) und runde Ecken des Strichs (F-108).
			s.attr("rx", zahl(document.WirksamerEckradius(o.Eckradius, b.Breite(), b.Hoehe())))
			s.farbeSVG("fill", o.Fuellung)
			s.farbeSVG("stroke", o.Strich.Farbe)
			s.attr("stroke-width", zahl(o.Strich.Breite))
			if document.WirksamerEckradius(o.Eckradius, b.Breite(), b.Hoehe()) == 0 {
				s.attr("stroke-linejoin", "miter") // spitze Außenecken wie der Renderer (F-498)
			} else {
				s.attr("stroke-linejoin", "round")
			}
			s.strichNaeherung(o.Strich, false)
			s.WriteString(`/>`)
		case document.Text:
			s.WriteString(`<g`)
			s.attr("s:typ", "text")
			s.ebene(o.Ebenenwerte())
			s.attr("s:ort", punkt(o.Ort))
			s.attr("s:inhalt", o.Inhalt)
			s.attr("s:grad", zahl(o.Schriftgrad))
			s.attr("s:farbe", farbeHex(o.Farbe))
			s.attr("s:hintergrund", farbeHex(o.Hintergrund))
			s.attr("s:umbruch", zahl(o.BreiteGrenze))
			if o.Spitze != nil {
				s.attr("s:spitze", punkt(*o.Spitze))
			}
			// Schrift, Stil und Ausrichtung nur, wo sie von der Voreinstellung abweichen (docs/format.md 8.3).
			if o.Schrift != "" && o.Schrift != fonts.Go { // Go ist die Voreinstellung (Sitzungs-QA B-11)
				s.attr("s:schrift", o.Schrift)
			}
			if st := textStilText(o); st != "" {
				s.attr("s:stil", st)
			}
			if a := ausrichtungText(o.Ausrichtung); a != "" {
				s.attr("s:ausrichtung", a)
			}
			s.WriteString(`>`)
			if a, sp, b, ok := o.Spitzendreieck(); ok {
				// Sprechblase (ADR-0021 Nachtrag 13 Punkte 2 und 3): ein Pfad aus Schild und Dreieck in der Füllfarbe,
				// ohne Füllung nur die beiden Schenkel als Linie in der Schriftfarbe (wie document.Sprechblase im Renderer).
				s.WriteString(`<path`)
				if o.Hintergrund.A > 0 && s.ohneText != nil && s.ohneText(o) {
					// Nur das Dreieck ist sichtbar: auch der Umriss des Schilds bleibt draußen, sonst stünden dessen
					// Breite und Höhe (ungefähre Länge und Zeilenzahl des Texts) in der Datei (F-728).
					s.attr("d", "M"+punkt(a)+" L"+punkt(sp)+" L"+punkt(b)+" Z")
					s.farbeSVG("fill", o.Hintergrund)
					s.attr("stroke-linejoin", "miter")
				} else if o.Hintergrund.A > 0 {
					var pfad strings.Builder
					for i, p := range document.Sprechblase(o) {
						if i == 0 {
							pfad.WriteString("M" + punkt(p))
						} else {
							pfad.WriteString(" L" + punkt(p))
						}
					}
					s.attr("d", pfad.String()+" Z")
					s.farbeSVG("fill", o.Hintergrund)
					s.attr("stroke-linejoin", "miter")
				} else {
					s.attr("d", "M"+punkt(a)+" L"+punkt(sp)+" L"+punkt(b))
					s.attr("fill", "none")
					s.farbeSVG("stroke", o.Farbe)
					s.attr("stroke-width", zahl(o.SchenkelBreite()))
					s.attr("stroke-linecap", "round")
					s.attr("stroke-linejoin", "round")
				}
				s.WriteString(`/>`)
			} else if o.Hintergrund.A > 0 {
				g := o.Grenzen()
				s.WriteString(`<rect`)
				s.attr("x", zahl(g.Min.X))
				s.attr("y", zahl(g.Min.Y))
				s.attr("width", zahl(g.Breite()))
				s.attr("height", zahl(g.Hoehe()))
				s.farbeSVG("fill", o.Hintergrund)
				s.WriteString(`/>`)
			}
			if s.ohneText != nil && s.ohneText(o) {
				s.WriteString(`</g>`) // nur das Dreieck ist sichtbar: kein Text im Klartext (F-692)
				break
			}
			s.WriteString(`<text`)
			s.attr("xml:space", "preserve") // sonst falten Betrachter mehrere und führende Leerzeichen (F-343)
			s.attr("font-family", svgSchriftFuer(o.Schrift))
			s.attr("font-size", zahl(o.Schriftgrad))
			// Gewicht und Stil nach dem Schnitt, den Renderer und Editor wirklich setzen (fehlt der verlangte, steht der
			// Ersatz, ADR-0021 Nachtrag 10 Punkt 4), nicht nach dem verlangten (F-577).
			schnitt := o.Schriftdatei().Schnitt
			if schnitt&fonts.Fett != 0 {
				s.attr("font-weight", "bold")
			}
			if schnitt&fonts.Kursiv != 0 {
				s.attr("font-style", "italic")
			}
			// Export und Darstellung der Arbeitsdatei verankern jede Zeile an Mitte bzw. rechtem Ende: Setzt ein
			// Betrachter eine andere Schrift, bleibt die Ausrichtung erhalten (F-577, F-607).
			wort, anker := textAnker(o.Ausrichtung)
			if wort != "" {
				s.attr("text-anchor", wort)
			}
			s.farbeSVG("fill", o.Farbe)
			s.WriteString(`>`)
			for _, zeile := range o.Zeilen() { // Anfang nach der Ausrichtung (ADR-0021 Nachtrag 10 Punkt 6)
				s.WriteString(`<tspan`)
				s.attr("x", zahl(zeile.X+anker*zeile.Vorschub))
				s.attr("y", zahl(zeile.Grundlinie))
				s.WriteString(`>`)
				xml.EscapeText(s, []byte(zeile.Text)) // sichtbar wie im Renderer: Tabulator als vier Leerzeichen (F-343); s:inhalt behält das Original
				s.WriteString(`</tspan>`)
			}
			s.WriteString(`</text>`)
			for _, st := range o.Striche() { // als Rechtecke, nicht text-decoration (Punkt 7)
				s.WriteString(`<rect`)
				s.attr("x", zahl(st.Min.X))
				s.attr("y", zahl(st.Min.Y))
				s.attr("width", zahl(st.Breite()))
				s.attr("height", zahl(st.Hoehe()))
				s.farbeSVG("fill", o.Farbe)
				s.WriteString(`/>`)
			}
			s.WriteString(`</g>`)
		case document.Nummer:
			s.WriteString(`<g`)
			s.attr("s:typ", "nummer")
			s.ebene(o.Ebenenwerte())
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
		case document.Lupe:
			s.lupe(d, o)
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
	s.attr("s:format", strconv.Itoa(schreibVersion(d.ObjekteOhneZug()))) // dieselbe Liste wie s.objekte
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
	s.attr("s:kante", d.Kante().Name()) // Form der Stauch-Kante (ADR-0023 Nachtrag 6, docs/format.md 7.1)
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
	kante         document.Kante // s:kante; fehlt es oder ist es unbekannt, KanteZacken (ADR-0023 Nachtrag 6)
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
	a       map[string]string
	err     error
	version int // Formatversion der Datei; 0 = unbekannt (nur für Angaben, die erst ab einer Version gelten)
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

// dezimal prüft die Schreibweise einer Zahl nach docs/format.md 5.2: nur Ziffern, Punkt, Minus und Exponent (`e`/`E`),
// ein Plus nur als Vorzeichen des Exponenten (`1e+06`, wie der Schreiber es mit %g erzeugt). Hexadezimale Gleitkomma-
// zahlen, Unterstriche, `+7`, `Inf` und `NaN` fallen hier heraus, obwohl strconv sie annähme.
func dezimal(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c >= '0' && c <= '9', c == '.', c == '-', c == 'e', c == 'E':
		case c == '+' && i > 0 && (s[i-1] == 'e' || s[i-1] == 'E'):
		default:
			return false
		}
	}
	return true
}

// parseZahl liest eine Zahl in der Schreibweise von dezimal; ok ist falsch bei NaN, Inf und anderer Schreibweise.
func parseZahl(s string) (float64, bool) {
	if !dezimal(s) {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil && !math.IsNaN(f) && !math.IsInf(f, 0)
}

// parseGanz liest eine ganze Zahl: optionales Minus, Ziffern, kein Plus (docs/format.md 5.2).
func parseGanz(s string) (int, bool) {
	if s == "" || s[0] == '+' {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	return n, err == nil
}

func (l *leser) zahl(name string) float64 {
	f, ok := parseZahl(l.text(name))
	if !ok {
		l.fehler("Zahl %s ungültig", name)
	}
	return f
}

func (l *leser) ganz(name string) int {
	n, ok := parseGanz(l.text(name))
	if !ok {
		l.fehler("Ganzzahl %s ungültig", name)
	}
	return n
}

func (l *leser) punkt(name string) document.Punkt {
	x, y, ok := strings.Cut(l.text(name), ",")
	fx, ok1 := parseZahl(x)
	fy, ok2 := parseZahl(y)
	if !ok || !ok1 || !ok2 {
		l.fehler("Punkt %s ungültig", name)
	}
	return document.Pt(fx, fy)
}

func (l *leser) farbe(name string) color.NRGBA {
	v := l.text(name)
	// genau `#` und acht Hexziffern, in beiden Schreibweisen (docs/format.md 5.2); kein Leerraum, kein Vorzeichen
	if len(v) != 9 || v[0] != '#' {
		l.fehler("Farbe %s ungültig", name)
		return color.NRGBA{}
	}
	n, err := strconv.ParseUint(v[1:], 16, 32)
	if err != nil {
		l.fehler("Farbe %s ungültig", name)
		return color.NRGBA{}
	}
	return color.NRGBA{R: uint8(n >> 24), G: uint8(n >> 16), B: uint8(n >> 8), A: uint8(n)}
}

func (l *leser) strich(name string) document.Strich {
	f, b, _ := strings.Cut(l.text(name), " ")
	l.a["_f"], l.a["_b"] = f, b
	return document.Strich{Farbe: l.farbe("_f"), Breite: l.zahl("_b")}
}

// strichart liest s:strichart (ADR-0021 Nachtrag 25): erst ab Version 2; fehlend, unbekannt oder in Version 1 voll
// (docs/format.md 11, Regel 3).
func (l *leser) strichart() document.Strichart {
	if l.version < versionStrichart {
		return document.StrichVoll
	}
	a, _ := document.StrichartAus(l.a["strichart"])
	return a
}

// strichMitArt ist strich samt s:strichart.
func (l *leser) strichMitArt(name string) document.Strich {
	st := l.strich(name)
	st.Art = l.strichart()
	return st
}

// bereich liest eine Fläche `x0,y0 x1,y1` (docs/format.md 5.2); die Ordnung der Ecken prüft das Objekt.
func (l *leser) bereich(name string) document.Rechteck {
	min, max, _ := strings.Cut(l.text(name), " ")
	l.a["_min"], l.a["_max"] = min, max
	return document.Rechteck{Min: l.punkt("_min"), Max: l.punkt("_max")}
}

func (l *leser) objekt(typ string) document.Objekt {
	o := l.objektOhneEbene(typ)
	if o == nil {
		return nil
	}
	return document.MitEbene(o, leseEbene(l.a, l.version))
}

func (l *leser) objektOhneEbene(typ string) document.Objekt {
	switch typ {
	case "pfeil":
		// s:ende und s:anfang gibt es erst ab Version 2 (ADR-0021 Nachtrag 20); fehlend oder unbekannt: Dreieck bei B,
		// keine bei A. In Version 1 sind sie unbekannter Inhalt (docs/format.md 11, Regel 3).
		p := document.Pfeil{A: l.punkt("a"), B: l.punkt("b"), K: l.punkt("k"), Strich: l.strich("strich")}
		if l.version >= versionPfeilKoepfe {
			ende, anfang := document.KopfDreieck, document.KopfKeine
			if v, da := l.a["ende"]; da {
				if k, ok := document.KopfartAus(v); ok {
					ende = k
				}
			}
			if v, da := l.a["anfang"]; da {
				if k, ok := document.KopfartAus(v); ok {
					anfang = k
				}
			}
			p = p.MitKoepfen(anfang, ende)
		}
		p.Strich.Art = l.strichart()
		return p
	case "kasten":
		return document.Kasten{Bereich: l.bereich("bereich"), Strich: l.strichMitArt("strich"), Fuellung: l.farbe("fuellung"),
			Eckradius: l.zahl("eckradius")}
	case "text":
		t := document.Text{Ort: l.punkt("ort"), Inhalt: l.text("inhalt"), Schriftgrad: l.zahl("grad"),
			Farbe: l.farbe("farbe"), Hintergrund: l.farbe("hintergrund"), BreiteGrenze: l.zahl("umbruch")}
		leseTextStil(&t, l.a)
		// s:spitze gibt es erst ab Version 2 (ADR-0021 Nachtrag 13 Punkt 7); in Version 1 ist es unbekannter Inhalt,
		// ignoriert (docs/format.md 11, Regel 3).
		if _, da := l.a["spitze"]; da && l.version >= versionSpitze {
			sp := l.punkt("spitze")
			t.Spitze = &sp
		}
		return t
	case "nummer":
		return document.Nummer{Mitte: l.punkt("mitte"), Wert: l.ganz("wert"), Radius: l.zahl("radius"),
			Farbe: l.farbe("farbe")}
	case "lupe":
		return document.Lupe{Quelle: l.bereich("quelle"), Ziel: l.bereich("ziel"), Strich: l.strichMitArt("strich")}
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
		inObjekt        int        // Tiefe des aktuellen Objekts, 0 = keins
		eltern          []xml.Name // Vorfahren des aktuellen Elements
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
			elternteil := xml.Name{}
			if len(eltern) > 0 {
				elternteil = eltern[len(eltern)-1]
			}
			eltern = append(eltern, t.Name)
			if inObjekt > 0 {
				continue // Darstellung eines Objekts
			}
			l := &leser{a: attrs(t), version: version}
			// Ein Element mit s:typ ist ein Objekt (docs/format.md 8); die Strukturelemente tragen es nie, auch nicht
			// leer. Ein leeres s:typ ist ebenfalls beschädigt, nicht „kein Objekt“.
			_, hatTyp := l.a["typ"]
			strukturell := tiefe == 1 || t.Name.Space == ns && (t.Name.Local == "meta" || t.Name.Local == "ansicht" ||
				t.Name.Local == "streifen" && elternteil == (xml.Name{Space: ns, Local: "ansicht"}) ||
				t.Name.Local == "schlagwort" && elternteil == (xml.Name{Space: ns, Local: "meta"}))
			if hatTyp && strukturell {
				return nil, fmt.Errorf("%w: s:typ an einem Strukturelement (%s)", ErrBeschaedigt, t.Name.Local)
			}
			if hatTyp && l.a["typ"] == "" {
				return nil, fmt.Errorf("%w: leeres s:typ", ErrBeschaedigt)
			}
			switch {
			case t.Name.Local == "svg" && tiefe == 1:
				// Die Version entscheidet vor jedem Inhalt (docs/format.md 11, Regel 2, F-387): eine neuere Fassung
				// darf nicht an ihren unbekannten Objekten als beschädigt scheitern; Version 0 gab es nie.
				version = l.ganz("format")
				if l.err != nil {
					return nil, l.err
				}
				if version < 1 {
					return nil, fmt.Errorf("%w: Formatversion %d ungültig", ErrBeschaedigt, version)
				}
				if version > FormatVersion {
					return nil, fmt.Errorf("%w: %d > %d", ErrFormatNeuer, version, FormatVersion)
				}
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
			case t.Name.Space == ns && t.Name.Local == "schlagwort" && elternteil == (xml.Name{Space: ns, Local: "meta"}):
				inSchlagwort = true
				in.meta.Schlagworte = append(in.meta.Schlagworte, "")
			case t.Name.Space == ns && t.Name.Local == "ansicht":
				// vier ganze Zahlen, einzelne Leerzeichen, nichts davor oder danach (docs/format.md 5.2)
				var z [4]int
				teile := strings.Split(l.text("zuschnitt"), " ")
				if len(teile) != 4 {
					l.fehler("Zuschnitt ungültig")
				} else {
					for i, teil := range teile {
						var ok bool
						if z[i], ok = parseGanz(teil); !ok {
							l.fehler("Zuschnitt ungültig")
						}
					}
				}
				in.zuschnitt, hatZuschnitt = image.Rect(z[0], z[1], z[2], z[3]), true
				// Form der Kante: Fehlt sie (ältere Dateien) oder ist sie unbekannt (neuere Form), gilt zigzag — kein
				// Schaden, die Kante ist Darstellung (docs/format.md 7.1, Regel 3 in 11).
				in.kante, _ = document.KanteAus(l.a["kante"])
			case t.Name.Space == ns && t.Name.Local == "streifen" && elternteil == (xml.Name{Space: ns, Local: "ansicht"}):
				in.streifen = append(in.streifen, document.Streifen{Zeilen: l.text("zeilen") == "true",
					Von: l.ganz("von"), Bis: l.ganz("bis")})
			case hatTyp:
				// Die Lupe gibt es erst ab Version 2 (ADR-0033 Punkt 5): In einer Datei der Version 1 ist sie ein
				// unbekannter Objekttyp, wie für einen Leser dieser Version.
				if l.a["typ"] == "lupe" && version < versionLupe {
					return nil, fmt.Errorf("%w: Objekttyp \"lupe\" erst ab Formatversion %d", ErrBeschaedigt, versionLupe)
				}
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
			eltern = eltern[:len(eltern)-1]
			tiefe--
		case xml.CharData:
			if inSchlagwort {
				in.meta.Schlagworte[len(in.meta.Schlagworte)-1] += string(t)
			}
		}
	}
	switch {
	case version < 1:
		return nil, fmt.Errorf("%w: Formatversion fehlt", ErrBeschaedigt)
	case !hatZuschnitt:
		return nil, fmt.Errorf("%w: Ansicht fehlt", ErrBeschaedigt)
	}
	return &in, nil
}

// pfeilDarstellung schreibt die sichtbare Gestalt eines Pfeils (ADR-0021 Nachtrag 19): ohne Kopf die Kurve als Strich mit
// runden Enden; mit Kopf den gekürzten Schaft als Strich mit geraden Enden, je Kopf ein gefülltes Vieleck und an einem
// Ende ohne Kopf eine halbe Scheibe.
//
// Kopf und halbe Scheibe reichen um pfeilFuge (höchstens die Länge des Schafts) in den Schaft zurück: SVG-Renderer glätten
// jede Form für sich, an einer gemeinsamen Kante bliebe eine helle Fuge (F-826). Damit eine durchscheinende Farbe die
// Überlappung nicht doppelt deckt, steht sie dann deckend in einer Gruppe mit opacity — wie im Renderer, der den Pfeil
// in einer Füllung zeichnet.
func (s *schreiber) pfeilDarstellung(o document.Pfeil) {
	f := o.Form()
	hw := o.Strich.Breite / 2
	farbe := o.Strich.Farbe
	if o.Strich.Art != document.StrichVoll && !s.rundlauf {
		s.pfeilGemustert(o, f)
		return
	}
	if len(f.Koepfe) == 0 {
		s.WriteString(`<path`)
		s.attr("d", "M"+punkt(o.A)+" Q"+punkt(o.K)+" "+punkt(o.B))
		s.attr("fill", "none")
		s.farbeSVG("stroke", farbe)
		s.attr("stroke-width", zahl(o.Strich.Breite))
		s.attr("stroke-linecap", "round")
		s.strichNaeherung(o.Strich, true)
		s.WriteString(`/>`)
		return
	}
	if farbe.A > 0 && farbe.A < 255 {
		s.WriteString(`<g`)
		s.attr("opacity", zahl(float64(farbe.A)/255))
		s.WriteString(`>`)
		farbe.A = 255
		defer s.WriteString(`</g>`)
	}
	fuge := 0.0
	if f.TB > f.TA {
		fuge = min(pfeilFuge, linienLaenge(f.Schaft))
		a, k, b := o.Teilstueck(f.TA, f.TB)
		s.WriteString(`<path`)
		s.attr("d", "M"+punkt(a)+" Q"+punkt(k)+" "+punkt(b))
		s.attr("fill", "none")
		s.farbeSVG("stroke", farbe)
		s.attr("stroke-width", zahl(o.Strich.Breite))
		s.strichNaeherung(o.Strich, false)
		s.WriteString(`/>`)
		if len(f.Koepfe) == 1 && len(f.Schaft) >= 2 { // das andere Ende rund
			ende, nach := f.Schaft[0], f.Schaft[1]
			if f.Koepfe[0].AmAnfang {
				ende, nach = f.Schaft[len(f.Schaft)-1], f.Schaft[len(f.Schaft)-2]
			}
			s.halbeScheibe(ende, nach, hw, fuge, farbe)
		}
	}
	for _, k := range f.Koepfe {
		u := k.Umriss(hw)
		if fuge > 0 {
			// Zurück in den Schaft: entgegen der Richtung nach außen, die zur Schaftnormalen n gehört (links(r) = n).
			n := k.SchaftNormale
			zurueck := document.Pt(-n.Y, n.X).Mal(fuge)
			u = append(append([]document.Punkt{u[0].Plus(zurueck)}, u...), u[len(u)-1].Plus(zurueck))
		}
		d := "M" + punkt(u[0])
		for _, q := range u[1:] {
			d += " L" + punkt(q)
		}
		s.WriteString(`<path`)
		s.attr("d", d+" Z")
		s.farbeSVG("fill", farbe)
		s.WriteString(`/>`)
	}
}

// pfeilFuge ist, wie weit Kopf und halbe Scheibe eines Pfeils im SVG in den Schaft zurückreichen (Pixel des Originals):
// ein ganzes Pixel, damit jedes Pixel an der Kante von einer der beiden Formen ganz gedeckt ist (F-826).
const pfeilFuge = 1.0

// linienLaenge ist die Länge des Linienzugs pts.
func linienLaenge(pts []document.Punkt) float64 {
	l := 0.0
	for i := 1; i < len(pts); i++ {
		l += pts[i].Minus(pts[i-1]).Laenge()
	}
	return l
}

// halbeScheibe schreibt die runde Kappe am Ende ende eines Strichs der halben Breite hw, dessen Nachbarpunkt nach ist;
// sie reicht um fuge in den Strich zurück (pfeilDarstellung).
func (s *schreiber) halbeScheibe(ende, nach document.Punkt, hw, fuge float64, farbe color.NRGBA) {
	d := ende.Minus(nach)
	l := d.Laenge()
	if l < 1e-9 || !(hw > 0) {
		return
	}
	r := d.Mal(1 / l)
	n := document.Pt(-r.Y, r.X)
	von, bis := ende.Minus(n.Mal(hw)), ende.Plus(n.Mal(hw))
	zurueck := r.Mal(-fuge)
	// Von rechts (−n) über die Außenseite ende + r·hw nach links (+n) wächst der Winkel immer: SVG-Flag 1.
	pfad := "M" + punkt(von.Plus(zurueck)) + " L" + punkt(von) + " A" + zahl(hw) + " " + zahl(hw) + " 0 0 1 " + punkt(bis) +
		" L" + punkt(bis.Plus(zurueck)) + " Z"
	if fuge <= 0 {
		pfad = "M" + punkt(von) + " A" + zahl(hw) + " " + zahl(hw) + " 0 0 1 " + punkt(bis) + " Z"
	}
	s.WriteString(`<path`)
	s.attr("d", pfad)
	s.farbeSVG("fill", farbe)
	s.WriteString(`/>`)
}
