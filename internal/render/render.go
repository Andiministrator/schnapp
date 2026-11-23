// SPDX-License-Identifier: GPL-3.0-or-later

// Package render setzt Ansichtsbild und Objekte eines Dokuments zu fertigen Pixeln zusammen (ADR-0020 Punkt 3):
// für das sichtbare Bild der Arbeitsdatei, PNG/JPG-Export und Zwischenablage. Ohne GPU, ohne Gio. Maße wie im
// SVG-Export (internal/fileformat/svg.go), damit Datei, Export und Bildschirm übereinstimmen: Pfeilspitze max(8, 4·Breite),
// Textzeilen im Abstand 1,3 × Schriftgrad, Nummer mit Schriftgrad 1,2 × Radius. Text wird wie im SVG-Export nicht
// umbrochen (F-70).
package render

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/errprefix"
	"github.com/Andiministrator/schnapp/internal/fonts"
	"github.com/Andiministrator/schnapp/internal/raster16"
	"github.com/Andiministrator/schnapp/internal/saveprogress"
)

// Bild liefert die Ansicht des Dokuments mit allen Objekten und den Kanten der Stauch-Nähte (ADR-0023). Objekte
// liegen im Original und werden über Dokument.ZuAnsicht abgebildet; was außerhalb der Ansicht liegt, fällt weg. Bild rastert immer vollständig und
// bricht nie wegen des Aufwands ab: Eine im Editor gemachte Bearbeitung darf nicht am Speichern scheitern (F-131),
// langsam ist besser als verloren. Gegen präparierte Dateien prüft der Editor beim Öffnen mit Pruefe (ADR-0020
// Nachtrag 1). Das Ergebnis trägt Alpha: Der Spalt jeder Stauch-Naht ist durchsichtig, ihr Schatten halbdurchsichtig
// (ADR-0023 Nachtrag 7). PNG, Zwischenablage, Arbeitsdatei und Vorschau behalten das; JPEG legt es auf Weiß
// (fileformat.ExportiereJPG).
func Bild(d *document.Dokument) (b *image.NRGBA, fehler error) {
	defer errprefix.Defer("render", &fehler) // ADR-0019 Nachtrag 2: auch Schrift- und Rasterfehler tragen das Präfix (F-379)
	b, _, err := bildMitAufwandOpt(d, true)
	return b, err
}

// PNG rendert das Dokument wie Bild und kodiert es als reines PNG für die Zwischenablage: nur IHDR, IDAT und IEND,
// ohne Bearbeitungsdaten (kein snAP/snAO) und ohne Metadaten (kein tEXt, iTXt, tIME), Kernprinzip 3. Wie Bild bricht es
// nie wegen des Aufwands ab. Der Editor ruft es in einer eigenen Goroutine auf einem Abzug des Dokuments auf, nicht im
// Fensterthread (F-178, ADR-0020 Nachtrag 6); das Dokument darf daneben nicht verändert werden.
func PNG(d *document.Dokument) (daten []byte, fehler error) {
	defer errprefix.Defer("render", &fehler) // F-379
	b, err := Bild(d)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, b); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ErrZuAufwendig meldet, dass das Rastern eines Dokuments das Rechenbudget (Budget) überschreiten würde: zu viele oder
// zu große Beschriftungen, etwa in einer präparierten Arbeitsdatei (F-125, F-132). Nur Pruefe liefert ihn, nie Bild.
var ErrZuAufwendig = errors.New("render: Dokument zu aufwendig zu rastern")

// Kostenmodell (F-125, F-131, F-132). Einheit ist die Rechenzeit, die das Rastern einer Glyphe je Schriftgrad² braucht,
// rund 0,3 ns (Messung 19.11.2026, Entwicklungsrechner, ein Kern, Karte #71):
//   - je Glyphe max(Schriftgrad², glyphGrund): Auch kleine Glyphen kosten 4 bis 10 µs (Gesichtsaufruf, Maske; Messung
//     23.11.2026, F-589: Grad 4 bis 24 rund 3,6 bis 5,5 µs, Grad 32 bis 64 rund 8 bis 10 µs, `BenchmarkGlyphKosten`), das sind
//     bei glyphGrund 5500 rund 0,6 bis 1,8 ns je Einheit, so viel wie bei großen Glyphen (rund 1 ns je Einheit);
//   - je Füllung fuellGewicht × Fläche ihres aufs Bild beschnittenen Rechtecks: rund 7 ns je Pixel mit einmischer
//     (F-130, Messung 20.11.2026; vorher rund 30 bis 36 ns über den allgemeinen Weg von x/image/vector, Gewicht 100);
//   - je Polygonecke punktGewicht (Geometrie, Ausrichten, LineTo; rund 30 ns) und je Kante kantenGewicht ×
//     Zeilenschritte (kantenSchritte, kreisSchritte): Der Rasterizer läuft jede Kante zeilenweise ab, auch die Zeilen
//     über dem Ausschnitt, und in jeder Zeile im Ausschnitt über ihre ganze waagerechte Spanne, auch außerhalb. Breite
//     Striche weit außerhalb des Bildes kosten so viel, obwohl der Ausschnitt winzig ist (F-132).
//
// Gemessen lag die Renderzeit bei Fällen auf kleinen Bildern bei 0,02 bis 0,5 ns je Einheit. Glyphen, die SICHTBAR
// aufs Bild gezeichnet werden, kosten mehr, als das Modell zählt (Übertragen der Maske): bis rund 9 ns je Einheit
// (Grad 24 bis 64 auf breitem Bild; „W“ Grad 4096 auf 8000×8000 5,5 ns, Sitzungs-QA 19.11.2026). Deshalb ist
// GlyphBudget eng gewählt (siehe unten).
const (
	glyphGrund    = 5500.0
	fuellGewicht  = 25.0
	kantenGewicht = 3.0
	punktGewicht  = 100.0
)

// Budget, GlyphBudget und KantenBudget sind die Obergrenzen des Rechenaufwands je Dokument, die der Editor beim
// Öffnen prüft (Pruefe): für die Summe, für die Glyphen und für Ecken und Kanten. Gemessen am 20.11.2026 nach F-130
// (Entwicklungsrechner, ein Kern; Benchmarks BenchmarkAufwand… in render_test.go, F-140):
//   - Ein alltagsnahes, großes Dokument (50 bildgroße Rahmen, 50 diagonale Pfeile, 100 Nummern, 200 Texte auf
//     7680×2160) kostet 4,06e10 Einheiten, fast alles Fläche, und rastert in rund 11 s (vorher 48 s). Budget ist gut
//     3-mal so groß; Glyphen (5,6e7, vor F-589 1,7e7) und Kanten (1,5e7) liegen beim 1/53 bzw. 1/650 ihrer Grenze. Eine echte
//     Bearbeitung wird beim Wiederöffnen also nicht abgelehnt — und beim Speichern ohnehin nie (Bild bricht nicht
//     ab; über dem Budget warnt der Editor, F-138). Auf 8000×8000 kostet ein bildgroßer Rahmen oder diagonaler Pfeil
//     1,6e9, ein gefüllter, umrandeter Kasten 3,2e9: Bis Budget/3 passen 26 bzw. 13 davon.
//   - Ein Dokument knapp unter Budget (39 gefüllte, umrandete Kästen auf 8000×8000, 1,25e11) rastert in rund 7 s
//     (vor ADR-0028 rund 35 s, vor F-135 bis knapp 5 min) — die Wartezeit, die eine präparierte Datei beim Speichern
//     erzwingen kann. Glyphen knapp unter GlyphBudget = 3e9 (seit F-589; vorher 1e9 bei glyphGrund 1700), sichtbar auf
//     8000×8000 als „W“ Grad 4096, brauchen rund 3,4 s (1,1 ns je Einheit), kleine Glyphen im Grad 64 knapp unter der
//     Grenze (rund 520 000) rund 5 s. Kanten knapp unter KantenBudget (vor ADR-0023 Nachtrag 7:
//     120 Abriss-Nähte auf 32768×2100, Kanten 9,4e9, Fläche 1,4e10) rasterten in rund 8 s (7,6 bis 8,9 s), davon rund
//     4 bis 5 s für die Kanten (rund 0,5 ns je Einheit, Vergleich mit 120 Zacken-Nähten: 3,2 bis 3,6 s); Messung
//     22.11.2026 nach F-418, ein Kern. Seit Nachtrag 7 (13 Füllungen je Naht, Band 32 px) kosten 120 Abriss-Nähte auf
//     dieser Breite 1,22e10 und werden abgelehnt; die Grenze liegt bei rund 98 Nähten (F-444). Die frühere Angabe
//     „rund 1,5 s“ stimmte für die Abrisskante nicht. Karte #177 (23.11.2026, F-430, F-429, F-444, F-408; ein Kern,
//     Benchmarks in naht_zeit_test.go): Bild zählt die Kosten nicht mehr mit, die Rasterschleifen nehmen min/max statt
//     math.Min/Max, ein Pixel (0,0,0,0) ohne Deckung wird übersprungen — bitgleich zu vorher (Prüfsummen), 120
//     Abriss-Nähte auf 32768×2100 statt 11,8 s in rund 6,5 s (liegen die Nähte eng, wie dort mit 17 px Abstand, wird
//     der Spalt für F-438 ein zweites Mal geleert: rund 7,3 s). Seit F-438 liegt die Grenze bei rund 91 Nähten auf
//     32768 px Länge und rund 364 auf 8192 px (TestNahtZahlGrenze); ein Dokument knapp unter allen drei Grenzen
//     zugleich (90 Nähte, 177 × „W“ Grad 4096, bildgroße Rahmen bis Budget) rastert in rund 14 s, nicht in 10 s
//     (BenchmarkNaehteAlleGrenzen); mit den teuersten kleinen Glyphen statt der „W“ (Grad 64, rund 520 000 Glyphen,
//     BenchmarkNaehteAlleGrenzenKleineGlyphen) in rund 16 s. Zusammen also rund 16 s (Messung 23.11.2026, F-589).
//     Die Gewichte unten stammen aus den Messungen vor ADR-0028 und bleiben (die Grenzen sind Obergrenzen).
//     So werden die präparierten Fälle aus F-125 und F-132 (10000 große Glyphen oder Nummern, 10000 sehr breite
//     Pfeile, 10000 Texte à 10000 winziger Glyphen, Hunderte bildgroßer Flächen) beim Öffnen abgelehnt.
//   - Pruefe rechnet nur die Geometrie, die Mittellinie einer Kante einmal je Achse (document.Kantenlinie), und bricht
//     ab, sobald eine Grenze überschritten ist (F-418): 120 Abriss-Nähte (vor Nachtrag 7 knapp unter KantenBudget) prüfte es
//     in rund 2 s, 1023 Abriss-Nähte auf 32768×2100 lehnt es nach rund 2 s ab (vorher 52 s, so lange hing das Öffnen).
//
// Kleiner lässt sich Budget nicht wählen, ohne den Abstand 3 zum Alltagsdokument aufzugeben: Für rund 30 s müsste es
// auf rund 1e11 sinken, das Alltagsdokument läge dann nur 2,5-fach darunter. Lieber eine präparierte Datei etwas
// länger speichern als eine echte Bearbeitung ablehnen (F-131).
//
// ZeichenBudget begrenzt die Zeichen aller Texte, sichtbar oder nicht (F-224): Ob ein Text in die Ansicht reicht,
// entscheidet seine vermessene Tinte (document.Reichweite), und Vermessen kostet rund 50 ns je Zeichen. Ganz
// verborgene Texte zählt das Kostenmodell sonst nicht (sie werden nicht gerastert); 400 Texte à 10 000 Zeichen außerhalb
// der Ansicht kosteten beim ersten Vermessen rund 0,2 s, an der Grenze der Dateigröße Sekunden. Der Zwischenspeicher
// der Tinte hält jedes weitere Bild kurz; die Grenze hält das erste Vermessen unter rund 0,05 s. Kein echtes Dokument
// eines Bildschirmfotos trägt eine Million Zeichen.
const (
	Budget        = 1.25e11
	GlyphBudget   = 3e9
	KantenBudget  = 1e10
	ZeichenBudget = 1_000_000
)

// grenzen sind die wirksamen Obergrenzen von Pruefe; nur Tests setzen sie kleiner.
var grenzen = struct {
	gesamt, glyph, kanten float64
	zeichen               int
}{Budget, GlyphBudget, KantenBudget, ZeichenBudget}

// aufwand zählt die Kosten nach dem Kostenmodell, getrennt nach Glyphen, Füllflächen und Kanten samt Ecken.
type aufwand struct {
	glyph, flaeche, kanten float64
	glyphen                int // Anzahl gerasterter Glyphen (Zählnahtstelle für Tests; im Trockenlauf 0)
}

// summe ist der ganze Rechenaufwand.
func (a aufwand) summe() float64 { return a.glyph + a.flaeche + a.kanten }

// Aufwand liefert den Rechenaufwand, den Bild für d hätte, ohne zu rastern: ein Trockenlauf desselben Zeichners, der
// die Geometrie berechnet und die Kosten zählt, aber weder Rasterizer noch Schrift aufruft. So laufen Kostenmodell
// und echtes Rendern nicht auseinander; Bild zählt dieselbe Summe.
func Aufwand(d *document.Dokument) float64 { return trockenlauf(d, false).summe() }

// trockenlauf zählt den Aufwand von d. Mit abbrechen hört er auf, sobald eine Grenze (grenzen) überschritten ist
// (Pruefe, F-418); die Summe ist dann nur ein Teil, aber schon zu groß.
func trockenlauf(d *document.Dokument, abbrechen bool) aufwand {
	z := zeichner{d: d, rand: image.Rectangle{Max: d.Groesse()}, auf: &aufwand{}, abbrechen: abbrechen}
	_ = z.alles() // ohne Schrift kein Fehler
	return *z.auf
}

// alles zeichnet die sichtbaren Objekte und darüber die Kanten der Stauch-Nähte (ADR-0023 Punkte 6 und 7, Nachträge 6
// und 7): zuerst wird der Spalt aller Nähte durchsichtig (Bild und Objekte darin fallen weg, Alpha 0), dann folgen je
// Naht alle Schichten (document.KantenSchichten) in der Form des Dokuments. Spalt und Schichten sind je eine Füllung
// und zählen im Trockenlauf (Pruefe) mit. Objekte ganz im Verborgenen (Dokument.ObjektSichtbar) fallen weg, wie im
// SVG-Export und im Editor.
func (z zeichner) alles() error {
	for _, o := range z.d.Objekte() {
		if z.abgebrochen() {
			return nil
		}
		if !z.d.ObjektSichtbar(o) || o.Ebenenwerte().Ausgeblendet { // ausgeblendete Ebenen fehlen (ADR-0032 Punkt 4)
			continue
		}
		if err := z.ebene(o); err != nil {
			return err
		}
	}
	z.kanten(true)
	return nil
}

// KantenOhneSchatten legt die Kanten der Stauch-Nähte von d über b, das Ansichtsbild von d (wird verändert), mit
// derselben Mischung wie Bild — aber ohne den Schatten im Spalt (document.Schicht.ImSpalt). Der Editor mischt die
// halbdurchsichtigen Schichten so im Renderer statt in Gio, das sie linear statt in sRGB mischt und den Fasersaum um
// im Mittel 22 von 255 zu hell zeigte (F-407, ADR-0023 Nachtrag 11); den durchsichtigen Spalt zeigt er weiter als
// Schachbrett mit dem Schatten darüber. Der Spalt ist danach durchsichtig (Alpha 0), wie in Bild.
func KantenOhneSchatten(b *image.NRGBA, d *document.Dokument) {
	if len(d.Naehte()) == 0 {
		return
	}
	z := zeichner{bild: b, rand: b.Bounds(), d: d, auf: &aufwand{}, ras: &raster16.Rasterizer{}, masken: neuerMaskenCache(),
		ohneZaehlen: true}
	z.kanten(false)
}

// kanten zeichnet die Kanten der Nähte: erst der Spalt aller Nähte durchsichtig, dann die Schichten; mit schatten auch
// den Schatten im Spalt (Bild), ohne ihn nur Fasersaum und Abdunklung (KantenOhneSchatten).
func (z zeichner) kanten(schatten bool) {
	g, f, k := z.d.Groesse(), z.d.KantenFaktor(), z.d.Kante()
	schichten := document.KantenSchichten()
	// Die Mittellinie je Achse einmal rechnen, die Schichten verschieben sie nur quer (F-418); ein Puffer für alle Bänder
	// (fuelle und raster behalten die Ecken nicht).
	var linien [2]*document.Kantenlinie
	var band []document.Punkt
	poly := make([][]document.Punkt, 1)
	naehte := z.d.Naehte()
	linie := func(n document.Naht) *document.Kantenlinie {
		achse, laenge := 0, g.X
		if !n.Zeilen {
			achse, laenge = 1, g.Y
		}
		if linien[achse] == nil {
			linien[achse] = document.NeueKantenlinie(k, laenge, f)
		}
		return linien[achse]
	}
	// Erst alle Spalte leeren, dann alle Schichten: So zeichnen Renderer, SVG-Export (Maske über den Kacheln) und
	// Editor (Schachbrett) auch eng beieinanderliegende Nähte gleich.
	spaltVon, spaltBis := document.Spaltband()
	spalte := func() bool {
		for _, n := range naehte {
			if z.abgebrochen() {
				return false
			}
			band = linie(n).Band(band, n, spaltVon, spaltBis)
			poly[0] = band
			z.radiere(poly)
		}
		return true
	}
	// schichtenDurchgang zeichnet die Schichten aller Nähte, die auswahl bejaht.
	schichtenDurchgang := func(auswahl func(document.Schicht) bool) bool {
		for _, n := range naehte {
			l := linie(n)
			for _, s := range schichten {
				if !auswahl(s) {
					continue
				}
				if z.abgebrochen() {
					return false
				}
				band = l.Band(band, n, s.Von, s.Bis)
				poly[0] = band
				z.fuelle(poly, nil, s.Farbe)
			}
		}
		return true
	}
	if !spalte() {
		return
	}
	if !document.NaehteBeruehren(naehte, f) {
		if schatten {
			schichtenDurchgang(func(document.Schicht) bool { return true })
		} else {
			schichtenDurchgang(func(s document.Schicht) bool { return !s.ImSpalt() })
		}
		return
	}
	// Kreuzen sich Nähte oder liegen sie eng beieinander, ragt der Fasersaum der einen in den Spalt der anderen (F-438,
	// ADR-0023 Nachtrag 9): Erst Fasersaum und Abdunklung, dann die Spalte noch einmal leeren, dann erst den Schatten
	// (er liegt im Spalt und bleibt dort, auch im Spalt der kreuzenden Naht).
	if !schichtenDurchgang(func(s document.Schicht) bool { return !s.ImSpalt() }) || !spalte() {
		return
	}
	if schatten {
		schichtenDurchgang(document.Schicht.ImSpalt)
	}
}

// abgebrochen meldet für den Trockenlauf von Pruefe, dass der gezählte Aufwand schon eine Grenze überschreitet: Mehr
// Zählen ändert das Urteil nicht mehr, nur die Wartezeit (F-418: 1023 Abriss-Nähte zählten sonst rund 50 s bis zur
// Ablehnung). Aufwand zählt immer vollständig; Bild zählt gar nicht (ohneZaehlen, F-430).
func (z zeichner) abgebrochen() bool {
	if !z.abbrechen {
		return false
	}
	a := z.auf
	return a.summe() > grenzen.gesamt || a.glyph > grenzen.glyph || a.kanten > grenzen.kanten
}

// Pruefe meldet ErrZuAufwendig, wenn das Rastern von d das Budget überschritte (F-125, F-132). Der Editor lehnt damit
// beim Öffnen ab; nach dem Speichern warnt er damit nur (F-138), abgebrochen wird das Speichern nie.
func Pruefe(d *document.Dokument) error {
	// Zuerst die Zeichen zählen, ohne zu vermessen: Der Trockenlauf fragt die Sichtbarkeit jedes Textes ab (F-224).
	if n := textZeichen(d); n > grenzen.zeichen {
		return fmt.Errorf("%w (Zeichen %d, Budget %d)", ErrZuAufwendig, n, grenzen.zeichen)
	}
	// Der Trockenlauf bricht ab, sobald eine Grenze überschritten ist (F-418): dasselbe Urteil, nur früher. Welche Grenze
	// die Meldung nennt, kann dann eine andere sein als nach dem vollen Zählen.
	a := trockenlauf(d, true)
	switch {
	case a.summe() > grenzen.gesamt:
		return fmt.Errorf("%w (Aufwand %.3g, Budget %.3g)", ErrZuAufwendig, a.summe(), grenzen.gesamt)
	case a.glyph > grenzen.glyph:
		return fmt.Errorf("%w (Glyphen %.3g, Budget %.3g)", ErrZuAufwendig, a.glyph, grenzen.glyph)
	case a.kanten > grenzen.kanten:
		return fmt.Errorf("%w (Kanten %.3g, Budget %.3g)", ErrZuAufwendig, a.kanten, grenzen.kanten)
	}
	return nil
}

// textZeichen zählt die Zeichen (Runen) aller Texte von d.
func textZeichen(d *document.Dokument) int {
	n := 0
	for _, o := range d.Objekte() {
		if t, ok := o.(document.Text); ok {
			n += utf8.RuneCountInString(t.Inhalt)
		}
	}
	return n
}

// bildMitAufwand ist Bild und liefert zusätzlich den Aufwand (für Tests).
func bildMitAufwand(d *document.Dokument) (*image.NRGBA, aufwand, error) {
	return bildMitAufwandOpt(d, false)
}

// bildMitAufwandOpt ist bildMitAufwand; mit ohneZaehlen zählt es die Kosten der Füllungen nicht (Bild braucht sie
// nicht, F-430: das Zählen kostete bei vielen Nähten rund ein Achtel der Zeit), der Aufwand ist dann unvollständig.
func bildMitAufwandOpt(d *document.Dokument, ohneZaehlen bool) (*image.NRGBA, aufwand, error) {
	b := d.Ansichtsbild()
	z := zeichner{bild: b, rand: b.Bounds(), d: d, faces: map[faceSchluessel]font.Face{}, auf: &aufwand{},
		ras: &raster16.Rasterizer{}, masken: neuerMaskenCache(), ohneZaehlen: ohneZaehlen}
	defer z.schliessen()
	if err := z.alles(); err != nil {
		return nil, *z.auf, err
	}
	return b, *z.auf, nil
}

// zeichner rastert die Objekte in bild; ist bild nil, ist es der Trockenlauf (Aufwand): nur Geometrie und Kosten.
type zeichner struct {
	bild  *image.NRGBA
	rand  image.Rectangle // Grenzen der Ansicht (auch im Trockenlauf)
	d     *document.Dokument
	faces map[faceSchluessel]font.Face
	auf   *aufwand
	// ras wird für jede Füllung zurückgesetzt statt neu angelegt: Seine Puffer (8 Byte je Pixel des Ausschnitts) nicht
	// je Füllung neu anzufordern, spart bei bildgroßen Flächen rund 5 % (F-130).
	ras *raster16.Rasterizer
	// masken merkt die Deckung großer Füllungen für gleiche Pfade (masken.go, F-430); nil im Trockenlauf.
	masken *maskenCache
	// abbrechen: nur im Trockenlauf von Pruefe — aufhören, sobald eine Grenze überschritten ist (abgebrochen, F-418).
	abbrechen bool
	// ohneZaehlen: nur Bild (F-430) — Füllungen rastern, ohne ihre Kosten zu zählen.
	ohneZaehlen bool
}

func (z zeichner) trocken() bool { return z.bild == nil }

// glyphKosten sind die Kosten von n Glyphen im Schriftgrad grad (Kostenmodell). Je Schrift wird nicht skaliert
// (F-576, Messung 23.11.2026, ein Kern): Einheiten je em spielen keine Rolle (der Grad ist in Pixeln); die dichtesten
// Glyphen einer Systemschrift (Noto Serif CJK „鬱龍蠶麤“) rastern bei Grad 8 mit rund 14 µs je Glyphe gegen rund 5,4 µs
// in Go (Faktor 2,6), bei Grad 40 mit rund 30 gegen 16 µs (Faktor 1,9) — das verlängert das Zeitversprechen
// (ADR-0020 Nachtrag 3) um einen Faktor unter 3 für den Glyphenanteil, es sprengt es nicht.
func glyphKosten(n int, grad float64) float64 { return float64(n) * math.Max(grad*grad, glyphGrund) }

// zeichenzahl zählt die Glyphen, die eine Zeile höchstens erzeugt: Tabulator wird zu vier Leerzeichen
// (document.Zeichenbar), entfernte Steuerzeichen zählen mit (obere Schranke, billig ohne Umwandeln).
func zeichenzahl(zeile string) int {
	return utf8.RuneCountInString(zeile) + 3*strings.Count(zeile, "\t")
}

func (z zeichner) schliessen() {
	for _, f := range z.faces {
		_ = f.Close()
	}
}

func (z zeichner) ansicht(p document.Punkt) document.Punkt { return z.d.ZuAnsicht(p) }

// ebene zeichnet o. Ist die Deckkraft der Ebene unter 100 %, entsteht das Objekt erst in einem durchsichtigen
// Zwischenbild in der Größe seines Umrisses und wird dann als Ganzes mit der Deckkraft eingemischt (Gruppendeckkraft,
// ADR-0032 Punkte 2 und 10): Schaft und Spitze, Schild und Schrift überlagern sich nicht doppelt. Im Trockenlauf zählt
// das Einmischen wie eine Füllung über dem Umriss.
func (z zeichner) ebene(o document.Objekt) error {
	p := o.Ebenenwerte().Deckkraft()
	if p >= 100 {
		return z.objekt(o)
	}
	r := z.ebenenRahmen(o)
	if r.Empty() || p <= 0 {
		return nil
	}
	if !z.ohneZaehlen {
		z.auf.flaeche += fuellGewicht * float64(r.Dx()) * float64(r.Dy())
	}
	if z.trocken() {
		return z.objekt(o)
	}
	zw := z
	zw.bild = image.NewNRGBA(r)
	if err := zw.objekt(o); err != nil {
		return err
	}
	mischeEbene(z.bild, zw.bild, p)
	return nil
}

// ebenenRahmen ist der Umriss von o in der Ansicht, aufs Bild begrenzt, mit Rand für die Pfeilspitze (sie ragt über
// Grenzen hinaus) und die Kantenglättung. Pfeil und Kasten bildet der Zeichner Punkt für Punkt ab, ihre Ecken reichen.
// Text und Nummer zeichnet er dagegen starr ab einem abgebildeten Anker (je Zeile bzw. die Mitte): Hinter einer
// Stauch-Naht liegt ihre Tinte dann weiter rechts bzw. unten als die abgebildete Ecke der Grenzen. Deshalb kommen die
// Grenzen, um die Verschiebung jedes Ankers versetzt, dazu (F-583).
func (z zeichner) ebenenRahmen(o document.Objekt) image.Rectangle {
	g := o.Grenzen()
	rand := 2.0
	switch v := o.(type) {
	case document.Pfeil:
		rand += math.Max(8, 4*v.Strich.Breite)
	case document.Kasten:
		// Der Strich liegt starr mit halber Breite um das abgebildete Rechteck; fällt seine Kante auf eine Naht, ragt er
		// über die abgebildeten Grenzen hinaus (Verify V23-1).
		rand += v.Strich.Breite / 2
	}
	r := document.R(z.ansicht(g.Min), z.ansicht(g.Max))
	starr := func(anker document.Punkt) {
		d := z.ansicht(anker).Minus(anker)
		r = r.Vereint(document.Rechteck{Min: g.Min.Plus(d), Max: g.Max.Plus(d)})
	}
	switch v := o.(type) {
	case document.Text:
		for _, zeile := range v.Zeilen() {
			starr(document.Pt(zeile.X, zeile.Grundlinie))
		}
	case document.Nummer:
		starr(v.Mitte)
	}
	return image.Rect(int(math.Floor(r.Min.X-rand)), int(math.Floor(r.Min.Y-rand)),
		int(math.Ceil(r.Max.X+rand)), int(math.Ceil(r.Max.Y+rand))).Intersect(z.rand)
}

// mischeEbene mischt das Zwischenbild q mit der Deckkraft p (Prozent) über bild, Pixel für Pixel mit derselben Formel
// wie jede Füllung (einmischer.misch, draw.Over in 16 Bit): das Pixel von q ist die Quelle, die Deckkraft die Deckung.
func mischeEbene(bild, q *image.NRGBA, p int) {
	ma := uint32(p) * 0xffff / 100
	r := q.Rect.Intersect(bild.Rect)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		saveprogress.Puls()
		qi, bi := q.PixOffset(r.Min.X, y), bild.PixOffset(r.Min.X, y)
		for x := 0; x < r.Dx(); x++ {
			s := q.Pix[qi+4*x : qi+4*x+4 : qi+4*x+4]
			if s[3] == 0 {
				continue
			}
			e := neuerEinmischer(bild, color.NRGBA{R: s[0], G: s[1], B: s[2], A: s[3]})
			e.misch(bild.Pix[bi+4*x:bi+4*x+4:bi+4*x+4], ma)
		}
	}
}

func (z zeichner) objekt(o document.Objekt) error {
	switch v := o.(type) {
	case document.Pfeil:
		const n = 48
		pts := make([]document.Punkt, 0, n+1)
		for i := 0; i <= n; i++ {
			pts = append(pts, z.ansicht(v.Punkt(float64(i)/n)))
		}
		z.strich(pts, false, v.Strich)
		r := v.RichtungAmEnde()
		g := math.Max(8, 4*v.Strich.Breite)
		nrm := document.Pt(-r.Y, r.X)
		fuss := v.B.Minus(r.Mal(g))
		z.fuelle([][]document.Punkt{{z.ansicht(v.B), z.ansicht(fuss.Plus(nrm.Mal(g / 2))),
			z.ansicht(fuss.Minus(nrm.Mal(g / 2)))}}, nil, v.Strich.Farbe)
	case document.Kasten:
		r := document.R(z.ansicht(v.Bereich.Min), z.ansicht(v.Bereich.Max))
		umriss := rundesRechteck(r, v.Eckradius)
		if v.Fuellung.A > 0 {
			z.fuelle([][]document.Punkt{umriss}, nil, v.Fuellung)
		}
		if v.Strich.Breite > 0 && v.Strich.Farbe.A > 0 {
			if document.WirksamerEckradius(v.Eckradius, r.Breite(), r.Hoehe()) == 0 {
				// Ohne Rundung spitze Außenecken (F-498): Strich als überlappungsfreie Rechtecke.
				var polys [][]document.Punkt
				for _, q := range document.EckigerRahmen(r, v.Strich.Breite) {
					polys = append(polys, []document.Punkt{q.Min, document.Pt(q.Max.X, q.Min.Y), q.Max, document.Pt(q.Min.X, q.Max.Y)})
				}
				z.fuelle(polys, nil, v.Strich.Farbe)
			} else {
				z.strich(umriss, true, v.Strich)
			}
		}
	case document.Text:
		if v.Hintergrund.A > 0 {
			g := v.Grenzen()
			r := document.R(z.ansicht(g.Min), z.ansicht(g.Max))
			z.fuelle([][]document.Punkt{rundesRechteck(r, 0)}, nil, v.Hintergrund)
		}
		var face font.Face
		if !z.trocken() {
			var err error
			if face, err = z.schrift(v.Schriftdatei(), v.Schriftgrad); err != nil {
				return err
			}
		}
		sch := v.Schriftdatei()
		for _, zeile := range v.Zeilen() {
			p := z.ansicht(document.Pt(zeile.X, zeile.Grundlinie))
			if !z.zeileSichtbar(sch, p.X, p.Y, v.Schriftgrad) {
				continue
			}
			z.auf.glyph += glyphKosten(zeichenzahl(zeile.Text), v.Schriftgrad)
			if z.trocken() {
				continue
			}
			// Tabulator als vier Leerzeichen, Wagenrücklauf weg (Zeichenbar in Zeilen): die Schriften haben für beide
			// keine Glyphe (Kritik A-3).
			z.text(face, sch, zeile.Text, p.X, p.Y, v.Farbe, v.Schriftgrad)
		}
		// Unter- und Durchstrich (ADR-0021 Nachtrag 10 Punkt 7): Rechtecke in der Textfarbe.
		for _, st := range v.Striche() {
			r := document.R(z.ansicht(st.Min), z.ansicht(st.Max))
			z.fuelle([][]document.Punkt{rundesRechteck(r, 0)}, nil, v.Farbe)
		}
	case document.Nummer:
		m := z.ansicht(v.Mitte)
		z.fuelle(nil, []kreisform{{m, v.Radius}}, v.Farbe)
		s := strconv.Itoa(v.Wert)
		grad := v.Radius * 1.2
		// Die Ziffern stehen mittig um m, weniger als len(s) × 0,8 em breit (Go Bold, mit Rand) und nicht höher als der
		// Schriftgrad; was dieses Rechteck nicht ins Bild bringt, kostet nichts und wird nicht gezeichnet.
		halb := grad * (0.4*float64(len(s)) + 1)
		if !image.Rect(int(math.Floor(m.X-halb)), int(math.Floor(m.Y-grad)),
			int(math.Ceil(m.X+halb)), int(math.Ceil(m.Y+grad))).Overlaps(z.rand) {
			return nil
		}
		z.auf.glyph += glyphKosten(len(s), grad)
		if z.trocken() {
			return nil
		}
		face, err := z.schrift(fonts.Laden(fonts.Go, fonts.Fett), grad)
		if err != nil {
			return err
		}
		breite := float64(font.MeasureString(face, s)) / 64
		// Senkrecht mittig wie dominant-baseline="central": Grundlinie um die halbe Ziffernhöhe unter der Mitte.
		hoehe := float64(face.Metrics().CapHeight) / 64
		z.text(face, fonts.Laden(fonts.Go, fonts.Fett), s, m.X-breite/2, m.Y+hoehe/2, color.NRGBA{R: 255, G: 255, B: 255, A: 255}, grad)
	}
	return nil
}

// zeileSichtbar meldet, ob eine Textzeile ab x auf dieser Grundlinie das Bild schneiden kann — dieselben Grenzen wie
// in text (senkrecht; rechts beginnt die erste Glyphe hinter dem Rand). Die Grenzen folgen der Schrift sch
// (fonts.Schrift.Huelle, F-576).
func (z zeichner) zeileSichtbar(sch *fonts.Schrift, x, grundlinie, grad float64) bool {
	oben, unten, seitlich := sch.Huelle()
	return grundlinie-grad*oben <= float64(z.rand.Max.Y) && grundlinie+grad*unten >= float64(z.rand.Min.Y) &&
		x <= float64(z.rand.Max.X)+seitlich*grad
}

// kreisform ist ein Kreis, der erst beim Rastern zum Polygon wird (kreis): Der Trockenlauf zählt seine Kosten aus
// Mitte und Radius, ohne bis zu 256 Ecken zu berechnen.
type kreisform struct {
	m document.Punkt
	r float64
}

// kreisEcken ist die Eckenzahl, mit der kreis einen Kreis vom Radius r annähert.
func kreisEcken(r float64) int { return int(math.Min(256, math.Max(16, r*2))) }

// fuelle füllt Polygone und Kreise (Ansichtskoordinaten) mit Kantenglättung. Alle Polygone werden gleich
// ausgerichtet, damit Überlappungen sich addieren statt aufzuheben; die Kreise kommen nach den Polygonen.
func (z zeichner) fuelle(polys [][]document.Punkt, kreise []kreisform, farbe color.NRGBA) {
	if farbe.A == 0 {
		return
	}
	z.fuelleMit(polys, kreise, farbe, false)
}

// radiere macht die Polygone durchsichtig (der Spalt einer Stauch-Naht, ADR-0023 Nachtrag 7): Die Deckung der Pixel
// sinkt um die Deckung der Fläche, mit Kantenglättung. Kosten wie eine Füllung.
func (z zeichner) radiere(polys [][]document.Punkt) { z.fuelleMit(polys, nil, color.NRGBA{}, true) }

// fuelleMit ist fuelle bzw. radiere (loeschen): Kosten zählen, dann rastern.
func (z zeichner) fuelleMit(polys [][]document.Punkt, kreise []kreisform, farbe color.NRGBA, loeschen bool) {
	if z.ohneZaehlen && !z.trocken() {
		for _, k := range kreise {
			polys = append(polys, kreis(k.m, k.r))
		}
		z.raster(polys, farbe, loeschen)
		return
	}
	// Kosten aus Polygonpunkten und Kreisen (Mitte ± Radius umschließt alle Ecken von kreis); Trockenlauf und
	// Rendern rechnen dieselbe Summe.
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	punkte := 0
	for _, p := range polys {
		if len(p) < 3 {
			continue
		}
		punkte += len(p)
		for _, q := range p {
			minX, minY = min(minX, q.X), min(minY, q.Y)
			maxX, maxY = max(maxX, q.X), max(maxY, q.Y)
		}
	}
	for _, k := range kreise {
		punkte += kreisEcken(k.r)
		minX, minY = min(minX, k.m.X-k.r), min(minY, k.m.Y-k.r)
		maxX, maxY = max(maxX, k.m.X+k.r), max(maxY, k.m.Y+k.r)
	}
	if minX > maxX {
		return
	}
	huelle := image.Rect(int(math.Floor(minX)), int(math.Floor(minY)), int(math.Ceil(maxX))+1, int(math.Ceil(maxY))+1).
		Intersect(z.rand)
	if huelle.Empty() {
		return // auch das (kleinere) Rechteck der Ecken liegt dann außerhalb
	}
	oy := float64(huelle.Min.Y)
	z.auf.flaeche += fuellGewicht * float64(huelle.Dx()) * float64(huelle.Dy())
	z.auf.kanten += punktGewicht*float64(punkte) +
		kantenGewicht*(kantenSchritte(polys, oy, huelle.Dy())+kreisSchritte(kreise, oy, huelle.Dy()))
	if z.trocken() {
		return
	}
	for _, k := range kreise {
		polys = append(polys, kreis(k.m, k.r))
	}
	z.raster(polys, farbe, loeschen)
}

// raster füllt die Polygone (mit loeschen: macht sie durchsichtig); gerastert wird nur das umschließende Rechteck ihrer
// Ecken (sonst kostet jedes Objekt das ganze Bild).
func (z zeichner) raster(polys [][]document.Punkt, farbe color.NRGBA, loeschen bool) {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, p := range polys {
		if len(p) < 3 {
			continue
		}
		for _, q := range p {
			minX, minY = min(minX, q.X), min(minY, q.Y)
			maxX, maxY = max(maxX, q.X), max(maxY, q.Y)
		}
	}
	if minX > maxX {
		return
	}
	ausschnitt := image.Rect(int(math.Floor(minX)), int(math.Floor(minY)), int(math.Ceil(maxX))+1, int(math.Ceil(maxY))+1).
		Intersect(z.rand)
	if ausschnitt.Empty() {
		return
	}
	// Je Füllung ein Puls: Flache Füllungen (unter 64 Zeilen) erreichen den Zeilenzähler des einmischers nie, viele davon
	// dauern trotzdem lange (Verify F-298 B-1).
	saveprogress.Puls()
	ox, oy := float64(ausschnitt.Min.X), float64(ausschnitt.Min.Y)
	pfade := func(r pfad) {
		for _, p := range polys {
			if len(p) < 3 {
				continue
			}
			if flaeche(p) < 0 {
				p = umgekehrt(p)
			}
			r.MoveTo(float32(p[0].X-ox), float32(p[0].Y-oy))
			for _, q := range p[1:] {
				r.LineTo(float32(q.X-ox), float32(q.Y-oy))
			}
			r.ClosePath()
		}
	}
	punkte := 0
	for _, p := range polys {
		punkte += len(p)
	}
	if z.rasterGemerkt(punkte, pfade, ausschnitt, farbe, loeschen) {
		return
	}
	if loeschen {
		ausradieren(z.ras, pfade, z.bild, ausschnitt)
		return
	}
	einmischen(z.ras, pfade, z.bild, ausschnitt, farbe)
}

// ausradieren senkt im Ausschnitt von bild die Deckung jedes Pixels um die Deckung der Pfade (draw.Op „Ziel außerhalb
// der Quelle“): A' = A · (1 − Deckung), auf 16 Bit gerechnet; die Farbe bleibt, ein ganz durchsichtiges Pixel wird
// (0,0,0,0) wie bei color.NRGBAModel. Für den Spalt der Stauch-Naht (ADR-0023 Nachtrag 7).
func ausradieren(ras *raster16.Rasterizer, pfade func(pfad), bild *image.NRGBA, ausschnitt image.Rectangle) {
	ras.Reset(ausschnitt.Dx(), ausschnitt.Dy())
	pfade(ras)
	sichtbar := ausschnitt.Intersect(bild.Rect)
	ras.Bloecke(64, func(y0 int, maske []uint32) {
		saveprogress.Puls()
		wendeBlockAn(bild, ausschnitt, sichtbar, y0, maske, radiereZeile[uint32])
	})
}

// pfad nimmt Strecken auf: raster16.Rasterizer, im Bitgleich-Test auch vector.Rasterizer.
type pfad interface {
	MoveTo(x, y float32)
	LineTo(x, y float32)
	ClosePath()
}

// einmischen zeichnet die Pfade einfarbig in den Ausschnitt von bild (draw.Over-Semantik). Paketvariable nur als
// Nahtstelle für den Bitgleich-Test gegen x/image/vector (TestFuellenBitgleich).
var einmischen = func(ras *raster16.Rasterizer, pfade func(pfad), bild *image.NRGBA, ausschnitt image.Rectangle, farbe color.NRGBA) {
	ras.Reset(ausschnitt.Dx(), ausschnitt.Dy())
	pfade(ras)
	e := neuerEinmischer(bild, farbe)
	sichtbar := ausschnitt.Intersect(bild.Rect)
	zeile := func(m []uint32, pix []byte) { mischeZeile(&e, m, pix) }
	// Je 64 Zeilen ein Block der Maske (bleibt im Cache) und ein Fortschritts-Puls fürs Speichern (F-298).
	ras.Bloecke(64, func(y0 int, maske []uint32) {
		saveprogress.Puls()
		wendeBlockAn(bild, ausschnitt, sichtbar, y0, maske, zeile)
	})
}

// einmischer mischt eine einfarbige Quelle in ein NRGBA-Bild, Pixel für Pixel mit der 16-Bit-Deckung aus
// raster16 (F-130, F-145). Er rechnet dieselbe Formel wie x/image/vector mit draw.Over auf *image.NRGBA
// (rasterizeOpOver, danach color.NRGBAModel) mit denselben uint32-Ausdrücken, also bitgleich (TestFuellenBitgleich),
// aber ohne Schnittstellenaufrufe je Pixel, und überspringt Pixel ohne Deckung auf deckendem Grund. Die Deckung kommt
// direkt aus der Maske (ADR-0028); vorher lief sie über den allgemeinen Weg von vector (At und Set je Pixel).
type einmischer struct {
	sr, sg, sb, sa uint32      // Quelle vormultipliziert, 16 Bit (color.NRGBA.RGBA)
	voll           color.NRGBA // Ergebnis bei voller Deckung einer deckenden Quelle
}

func neuerEinmischer(b *image.NRGBA, farbe color.NRGBA) einmischer {
	e := einmischer{voll: farbe}
	e.sr, e.sg, e.sb, e.sa = farbe.RGBA()
	return e
}

// mischeEins mischt ein Pixel mit der Deckung ma, mit den häufigen Fällen ohne Aufruf von misch: keine Deckung auf
// deckendem oder ganz durchsichtigem Grund, volle Deckung einer deckenden Quelle, deckender Grund (mischDeckend).
func (e *einmischer) mischeEins(ma uint32, p []byte) {
	if ma == 0 && (p[3] == 0xff || p[3]|p[0]|p[1]|p[2] == 0) {
		return // (0,0,0,0) bliebe (0,0,0,0), misch liefert dasselbe (F-430)
	}
	if ma == 0xffff && e.sa == 0xffff {
		p[0], p[1], p[2], p[3] = e.voll.R, e.voll.G, e.voll.B, 0xff
		return
	}
	if p[3] == 0xff {
		e.mischDeckend(p, ma)
		return
	}
	e.misch(p, ma)
}

// mischDeckend ist misch für ein deckendes Pixel (p[3] == 0xff, der Normalfall einer Aufnahme): Dort ist das Alpha des
// Ziels 0xffff, das Ergebnis-Alpha wird wieder 0xffff (a + sa·ma/0xffff = 0xffff, ohne Überlauf: höchstens
// 0x10000·0xffff), und die Umrechnung des Ziels ist c·0x101. Dieselben Zahlen wie misch ohne dessen Umwege, auch bei
// Deckung 0 (Pixel bleibt) und bei voller Deckung einer deckenden Quelle (Farbe der Quelle): ohne Fallunterscheidung,
// also ohne Fehlsprünge an den Rändern der Bänder (TestMischDeckendGleich).
func (e *einmischer) mischDeckend(p []byte, ma uint32) {
	a := 0xffff - (e.sa * ma / 0xffff)
	r := uint32(uint16((uint32(p[0])*0x101*a + e.sr*ma) / 0xffff))
	g := uint32(uint16((uint32(p[1])*0x101*a + e.sg*ma) / 0xffff))
	b := uint32(uint16((uint32(p[2])*0x101*a + e.sb*ma) / 0xffff))
	p[0], p[1], p[2] = uint8(r>>8), uint8(g>>8), uint8(b>>8)
}

// misch mischt die Quelle mit der Deckung ma (16 Bit) in das Pixel p (vier Bytes NRGBA).
func (e *einmischer) misch(p []byte, ma uint32) {
	switch {
	case ma == 0 && p[3] == 0xff:
		return // draw.Over ohne Deckung auf deckendem Pixel lässt es unverändert
	case ma == 0xffff && e.sa == 0xffff:
		p[0], p[1], p[2], p[3] = e.voll.R, e.voll.G, e.voll.B, 0xff
		return
	}
	// Wie vector.rasterizeOpOver: Ziel und Quelle vormultipliziert, 16 Bit.
	dr, dg, db, da := color.NRGBA{R: p[0], G: p[1], B: p[2], A: p[3]}.RGBA()
	a := 0xffff - (e.sa * ma / 0xffff)
	r := uint32(uint16((dr*a + e.sr*ma) / 0xffff))
	g := uint32(uint16((dg*a + e.sg*ma) / 0xffff))
	b := uint32(uint16((db*a + e.sb*ma) / 0xffff))
	al := uint32(uint16((da*a + e.sa*ma) / 0xffff))
	// Wie image.NRGBA.Set über color.NRGBAModel.
	switch al {
	case 0xffff:
		p[0], p[1], p[2], p[3] = uint8(r>>8), uint8(g>>8), uint8(b>>8), 0xff
	case 0:
		p[0], p[1], p[2], p[3] = 0, 0, 0, 0
	default:
		p[0], p[1], p[2], p[3] = uint8((r*0xffff/al)>>8), uint8((g*0xffff/al)>>8), uint8((b*0xffff/al)>>8), uint8(al>>8)
	}
}

// kantenSchritte zählt, wie viele Schritte der Rasterizer (x/image/vector, LineTo) für die Kanten der Polygone geht,
// bei einem Ausschnitt der Höhe hoehe ab Zeile oy: je Kante einen Aufruf, je Zeile von ihrem oberen Ende bis zum
// unteren Ende oder Ausschnittsrand einen Schritt (auch über dem Ausschnitt), und in den Zeilen im Ausschnitt die
// waagerechte Spanne in Pixeln plus zwei (vector zählt sie auch außerhalb der Breite ab).
func kantenSchritte(polys [][]document.Punkt, oy float64, hoehe int) float64 {
	h := float64(hoehe)
	var s float64
	for _, p := range polys {
		if len(p) < 3 {
			continue
		}
		for i, a := range p {
			b := p[(i+1)%len(p)]
			s++
			oben, unten := min(a.Y, b.Y)-oy, max(a.Y, b.Y)-oy
			if unten-oben <= 1e-6 {
				continue
			}
			von, bis := math.Floor(oben), min(math.Ceil(unten), h)
			if bis <= von {
				continue
			}
			innen := bis - max(von, 0)
			s += bis - von
			if innen > 0 {
				dx := math.Abs(b.X - a.X)
				s += 2*innen + min(dx, innen*dx/(unten-oben))
			}
		}
	}
	return s
}

// kreisSchritte schätzt kantenSchritte für die Ecken von kreis nach oben ab, ohne sie zu berechnen: Jede Hälfte läuft
// die Zeilen vom oberen Rand bis zum unteren Rand oder Ausschnittsrand ab (je Kante höchstens eine Zeile mehr), in
// den Zeilen im Ausschnitt mit zwei Schritten je Zeile und Kante, die waagerechte Spanne beider Hälften zusammen ist
// höchstens 4 r.
func kreisSchritte(kreise []kreisform, oy float64, hoehe int) float64 {
	h := float64(hoehe)
	var s float64
	for _, k := range kreise {
		n := float64(kreisEcken(k.r))
		s += 2 * n
		von, bis := math.Floor(k.m.Y-k.r-oy), min(math.Ceil(k.m.Y+k.r-oy), h)
		if bis <= von {
			continue
		}
		innen := bis - max(von, 0)
		s += 2 * (bis - von)
		if innen > 0 {
			s += 2*(2*innen+n) + 4*k.r
		}
	}
	return s
}

// strich zieht einen Linienzug der Breite s.Breite mit runden Enden und Ecken: je Strecke ein Viereck, je Punkt ein
// Kreis, alles gleich ausgerichtet und in einem Durchgang gefüllt (keine doppelte Deckung an den Stößen).
func (z zeichner) strich(pts []document.Punkt, geschlossen bool, s document.Strich) {
	hw := s.Breite / 2
	if hw <= 0 || len(pts) == 0 {
		return
	}
	var polys [][]document.Punkt
	n := len(pts)
	strecken := n - 1
	if geschlossen {
		strecken = n
	}
	for i := 0; i < strecken; i++ {
		a, b := pts[i], pts[(i+1)%n]
		d := b.Minus(a)
		l := d.Laenge()
		if l < 1e-9 {
			continue
		}
		nrm := document.Pt(-d.Y/l*hw, d.X/l*hw)
		polys = append(polys, []document.Punkt{a.Plus(nrm), b.Plus(nrm), b.Minus(nrm), a.Minus(nrm)})
	}
	kreise := make([]kreisform, len(pts))
	for i, p := range pts {
		kreise[i] = kreisform{p, hw}
	}
	z.fuelle(polys, kreise, s.Farbe)
}

// text zeichnet eine Zeile. Was das Bild nicht schneidet, wird nicht gerastert (F-118): eine Zeile, deren Höhe ganz
// über oder unter dem Bild liegt, entfällt, ebenso jede Glyphe, deren Ursprung mehr als die seitliche Hülle der Schrift (mindestens
// 1,25 Schriftgrade, fonts.Schrift.Huelle) links oder rechts vom Bild liegt (Vorschub und Unterschneidung laufen weiter, damit sichtbare Glyphen am selben Ort stehen).
// Die Kosten zählt der Aufrufer (glyphKosten), auch im Trockenlauf.
func (z zeichner) text(face font.Face, sch *fonts.Schrift, s string, x, grundlinie float64, farbe color.NRGBA, grad float64) {
	var puffer sfnt.Buffer // je Aufruf: Zeilen können nebenläufig gezeichnet werden
	sf := sch.Font
	rand := z.rand
	oben, unten, seitlich := sch.Huelle()
	if grundlinie-grad*oben > float64(rand.Max.Y) || grundlinie+grad*unten < float64(rand.Min.Y) {
		return
	}
	dot := fixed.Point26_6{X: fixed.Int26_6(math.Round(x * 64)), Y: fixed.Int26_6(math.Round(grundlinie * 64))}
	// Seitlicher Rand: mindestens 1,25 Schriftgrad (Glyphen sind bis gut 1,07 em breit, z. B. U+20A7), bei breiteren
	// Schriften ihr head-Rahmen (Huelle).
	links, rechts := float64(rand.Min.X)-seitlich*grad, float64(rand.Max.X)+seitlich*grad
	vorher := rune(-1)
	for _, c := range s {
		saveprogress.Puls() // je Zeichen: Glyphen kosten am Budget Sekunden (F-147, F-298)
		if vorher >= 0 {
			dot.X += document.KernPaar(sf, &puffer, vorher, c, grad) // im Grad wie die Messung (Sitzungs-QA B-4)
		}
		vorher = c
		if px := float64(dot.X) / 64; px > rechts {
			return // alles Weitere liegt rechts vom Bild
		} else if px < links {
			// Vorschub wie font.Drawer.DrawString: auch ohne Glyphe (ok = false) zählt der Vorschub des Ersatzkästchens.
			adv, _ := face.GlyphAdvance(c)
			dot.X += adv
			continue
		}
		z.auf.glyphen++
		// Wie font.Drawer.DrawString mit einem Zeichen (ohne Kerning, das steht oben), aber mit eigenem Einmischen
		// statt draw.DrawMask, das für *image.NRGBA nur den allgemeinen Weg kennt (F-147).
		gr, maske, mp, adv, _ := face.Glyph(dot, c)
		if !gr.Empty() {
			glyphMischen(z.bild, gr, maske, mp, farbe)
		}
		dot.X += adv
	}
}

// glyphMischen mischt die Glyphenmaske einfarbig in bild, bitgleich zu draw.DrawMask(bild, gr, image.NewUniform(farbe),
// image.Point{}, maske, mp, draw.Over) (TestGlyphMischenBitgleich): gleicher Zuschnitt, Deckung 0 lässt das Pixel
// unverändert, sonst dieselbe Formel wie einmischer.misch mit der 8-Bit-Deckung auf 16 Bit gedehnt (a·0x101, wie
// image.Alpha.RGBA64At). Andere Maskentypen als *image.Alpha gehen über draw.DrawMask.
func glyphMischen(bild *image.NRGBA, gr image.Rectangle, maske image.Image, mp image.Point, farbe color.NRGBA) {
	alpha, ok := maske.(*image.Alpha)
	if !ok {
		draw.DrawMask(bild, gr, image.NewUniform(farbe), image.Point{}, maske, mp, draw.Over)
		return
	}
	r := gr.Intersect(bild.Rect).Intersect(alpha.Rect.Add(gr.Min.Sub(mp)))
	if r.Empty() {
		return
	}
	mp = mp.Add(r.Min.Sub(gr.Min))
	e := neuerEinmischer(bild, farbe)
	for y := 0; y < r.Dy(); y++ {
		m := alpha.Pix[alpha.PixOffset(mp.X, mp.Y+y):][:r.Dx()]
		i := bild.PixOffset(r.Min.X, r.Min.Y+y)
		pix := bild.Pix[i : i+4*r.Dx()]
		for k, a := range m {
			if a == 0 {
				continue
			}
			e.misch(pix[4*k:4*k+4:4*k+4], uint32(a)*0x101)
		}
	}
}

// kreis nähert einen Kreis mit genug Ecken für glatte Ränder an.
func kreis(m document.Punkt, r float64) []document.Punkt {
	n := kreisEcken(r)
	pts := make([]document.Punkt, n)
	for i := range pts {
		w := 2 * math.Pi * float64(i) / float64(n)
		pts[i] = document.Pt(m.X+r*math.Cos(w), m.Y+r*math.Sin(w))
	}
	return pts
}

// rundesRechteck liefert den Umriss eines Rechtecks mit Eckradius (auf die halbe kürzere Seite begrenzt).
func rundesRechteck(r document.Rechteck, radius float64) []document.Punkt {
	radius = document.WirksamerEckradius(radius, r.Breite(), r.Hoehe())
	if radius == 0 {
		return []document.Punkt{r.Min, document.Pt(r.Max.X, r.Min.Y), r.Max, document.Pt(r.Min.X, r.Max.Y)}
	}
	ecken := []struct {
		m document.Punkt
		w float64
	}{
		{document.Pt(r.Max.X-radius, r.Min.Y+radius), -math.Pi / 2},
		{document.Pt(r.Max.X-radius, r.Max.Y-radius), 0},
		{document.Pt(r.Min.X+radius, r.Max.Y-radius), math.Pi / 2},
		{document.Pt(r.Min.X+radius, r.Min.Y+radius), math.Pi},
	}
	const schritte = 12
	var pts []document.Punkt
	for _, e := range ecken {
		for i := 0; i <= schritte; i++ {
			w := e.w + math.Pi/2*float64(i)/schritte
			pts = append(pts, document.Pt(e.m.X+radius*math.Cos(w), e.m.Y+radius*math.Sin(w)))
		}
	}
	return pts
}

func flaeche(p []document.Punkt) float64 {
	var s float64
	for i := range p {
		q := p[(i+1)%len(p)]
		s += p[i].X*q.Y - q.X*p[i].Y
	}
	return s / 2
}

func umgekehrt(p []document.Punkt) []document.Punkt {
	r := make([]document.Punkt, len(p))
	for i := range p {
		r[i] = p[len(p)-1-i]
	}
	return r
}

// Schriften kommen aus internal/fonts (ADR-0021 Nachtrag 10); sfnt-Fonts sind nebenläufig nutzbar. Faces sind es
// nicht — jeder Aufruf von Bild hält deshalb seine eigenen (zeichner.faces).
// neueFace legt ein Face an; der Test des Fehlerwegs tauscht es aus (die eingebauten Schriften lassen sich nicht
// kaputt machen).
var neueFace = func(f *sfnt.Font, o *opentype.FaceOptions) (font.Face, error) { return opentype.NewFace(f, o) }

type faceSchluessel struct {
	f    *sfnt.Font
	grad float64
}

func (z zeichner) schrift(s *fonts.Schrift, grad float64) (font.Face, error) {
	k := faceSchluessel{s.Font, grad}
	if f, ok := z.faces[k]; ok {
		return f, nil
	}
	face, err := neueFace(s.Font, &opentype.FaceOptions{Size: grad, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return nil, err
	}
	z.faces[k] = face
	return face, nil
}
