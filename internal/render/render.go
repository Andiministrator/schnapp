// SPDX-License-Identifier: GPL-3.0-or-later

// Package render setzt Ansichtsbild und Objekte eines Dokuments zu fertigen Pixeln zusammen (ADR-0020 Punkt 3):
// für das sichtbare Bild der Arbeitsdatei, PNG/JPG-Export und Zwischenablage. Ohne GPU, ohne Gio. Maße wie im
// SVG-Export (internal/fileformat/svg.go), damit Datei, Export und Bildschirm übereinstimmen: Pfeilspitzen aus
// document.Pfeil.Form (ADR-0021 Nachtrag 19),
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
//     bei glyphGrund 5500 rund 0,6 bis 1,8 ns je Einheit, so viel wie bei großen Glyphen (rund 1 ns je Einheit; die
//     „rund 0,3 ns“ oben gelten seit ADR-0028 nicht mehr). Das gilt für gemischten Fließtext. Die Form max(…) zählt in
//     der Mitte zu wenig: Dichte Glyphen an der Knickstelle (Grad 74, 74² = glyphGrund) kosten bis rund 20 µs, also
//     rund 3,7 ns je Einheit („@“; „W“ rund 3 ns, Fließtext rund 2 ns; Kritik zu Karte #224, 23.11.2026), bewusst so
//     belassen: Das Zeitversprechen rund 21 s gilt (ADR-0020 Nachtrag 12), gemessen mit eingebauten und Systemschriften,
//     seit Glyphen aus Systemschriften mit systemFaktor zählen (ADR-0020 Nachtrag 13, F-601);
//   - je Füllung fuellGewicht × Fläche ihres aufs Bild beschnittenen Rechtecks: rund 7 ns je Pixel mit einmischer
//     (F-130, Messung 20.11.2026; vorher rund 30 bis 36 ns über den allgemeinen Weg von x/image/vector, Gewicht 100);
//     eine Lupe, deren Ziel in die Ansicht reicht, zählt so ihr Ziel und zusätzlich ihre aufs Original begrenzte Quelle,
//     weil der SVG-Export bis zu so viele Pixel einbettet (ADR-0033 Nachtrag 2 Punkt 3, K193-1);
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
	// aufrufGewicht sind die Grundkosten je Füllung (Rasterizer zurücksetzen, Pfad, Ausschnitt, Puls), unabhängig von
	// Fläche und Kanten: rund 5,5 µs, bei 0,5 ns je Kanten-Einheit 11 000, aufgerundet. Im Alltag fällt das nicht ins
	// Gewicht (wenige Tausend Füllungen), aber über vielen Nähten zeichnet jedes schräge Objekt je Bereich eine eigene
	// Füllung: 1000 Sprechblasen über 4096 Kacheln rasterten in 26 s bei 63 % von KantenBudget (F-750, ADR-0023
	// Nachtrag 16). Gezählt bei den Kanten.
	aufrufGewicht = 12000.0
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
//     Grenze (rund 520 000) rund 5 s, das teuerste gefundene Muster („@“ im Grad 74, rund 530 000 Glyphen) rund 11 s.
//     Kanten knapp unter KantenBudget (vor ADR-0023 Nachtrag 7:
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
//     zugleich (90 Nähte, 177 × „W“ Grad 4096, bildgroße Rahmen bis Budget) rastert in rund 12 bis 14 s, nicht in 10 s
//     (BenchmarkNaehteAlleGrenzen); mit kleinen Glyphen statt der „W“ (BenchmarkNaehteAlleGrenzenKleineGlyphen) in
//     rund 16 s (Fließtext Grad 64, rund 520 000 Glyphen) und mit dem teuersten gefundenen Muster („@“ Grad 74, rund
//     530 000 Glyphen) in rund 21 s. Zusammen also rund 21 s (Messung 23.11.2026, F-589 und Kritik zu Karte #224).
//     Gemessen mit eingebauten und Systemschriften (ADR-0020 Nachtrag 13, Karte #238, F-601): Dasselbe Dokument mit
//     „Noto Serif CJK JP“ „龘“ (BenchmarkNaehteAlleGrenzenSystemschrift; Glyphen mit systemFaktor gezählt, rund
//     210 000 statt 530 000) rastert in 20,9 bis 22,1 s, Go „@“ im selben Lauf 20,5 bis 20,9 s (24.11.2026, ein Kern,
//     Last rund 2 durch andere Läufe); ohne den Faktor waren es 41,4 und 42,5 s.
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
	einmischungen          int // Aufrufe zum Einmischen von Glyphen (Zählnahtstelle für Tests, F-766; im Trockenlauf 0)
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
	if err := z.objekte(); err != nil {
		return err
	}
	z.kanten(true)
	return nil
}

// objekte zeichnet die sichtbaren Objekte ohne die Kanten der Nähte (alles, Objektebene).
func (z zeichner) objekte() error {
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
	return nil
}

// KantenOhneSchatten legt die Kanten der Stauch-Nähte von d über b, das Ansichtsbild von d (wird verändert), mit
// derselben Mischung wie Bild — aber ohne den Schatten im Spalt (document.Schicht.ImSpalt). Der Editor mischt die
// halbdurchsichtigen Schichten so im Renderer statt in Gio, das sie bis ADR-0020 Nachtrag 14 linear statt in sRGB
// mischte und den Fasersaum um im Mittel 22 von 255 zu hell zeigte (F-407, ADR-0023 Nachtrag 11); seit Nachtrag 14
// nicht mehr nötig, bleibt aber; den durchsichtigen Spalt zeigt er weiter als
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

// glyphKosten sind die Kosten von n Glyphen im Schriftgrad grad in einer eingebauten Schrift (Kostenmodell, Faktor 1).
// Einheiten je em spielen keine Rolle (der Grad ist in Pixeln, F-576).
func glyphKosten(n int, grad float64) float64 { return float64(n) * math.Max(grad*grad, glyphGrund) }

// systemFaktor ist der feste Kostenfaktor für Glyphen aus Systemschriften, also jeder Familie außer den eingebauten
// Go und Go Mono (ADR-0020 Nachtrag 13, Karte #238, F-601). glyphGrund und GlyphBudget sind an Go geeicht; Systemschriften
// rastern je Glyphe teurer, gemessen (ein Kern, ungestört; Sammelkritik Sprint 12 KS-2, F-576) bis 2,39-mal so lange
// wie Go: „Noto Serif CJK JP“ „龘“ im Grad 74 im Budget-Dokument 41,4 und 42,5 s gegen 19,8 bis 20,5 s für Go „@“,
// lateinische Systemschriften bis 1,21-mal (dichte CJK-Glyphen im Grad 8 einzeln rund 14 gegen 5,4 µs, F-576). Gezählt
// wird pessimistisch mit 2,5 statt je Schrift aus den Umrissen gemessen (bewusst nicht Teil von Karte #238): Ein
// Dokument mit Systemschrift erreicht GlyphBudget so schon bei 1/2,5 der Glyphen und hält das Zeitversprechen von rund
// 21 s (CJK-Budget-Dokument 20,9 bis 22,1 s, siehe Budget); das Alltagsdokument bleibt auch ganz in einer
// Systemschrift rund 21-mal unter GlyphBudget (TestAlltagsdokumentSystemschriftWeitUnterDemBudget).
const systemFaktor = 2.5

// glyphKostenIn sind die Kosten von n Glyphen im Grad grad in der Schrift sch: glyphKosten, für Systemschriften mal
// systemFaktor. Ersatz (die Familie fehlt, gerastert wird Go) zählt wie Go.
func glyphKostenIn(sch *fonts.Schrift, n int, grad float64) float64 {
	if sch.Eingebaut() {
		return glyphKosten(n, grad)
	}
	return systemFaktor * glyphKosten(n, grad)
}

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

// jeStueck ruft f je sichtbarem Bereich der Ansicht, den r (Original) berührt (document.Stuecke), mit einem Zeichner,
// der auf diesen Bereich beschnitten ist, und der Abbildung Original → Ansicht in diesem Bereich (p + Versatz). So
// zeichnen Renderer und Editor schräge Kanten über einer Naht wie der SVG-Export und der Treffer: abgeschnitten, nicht
// zwischen abgebildeten Ecken gerade verbunden (F-695, ADR-0023 Nachtrag 14). Für achsparallele Rechtecke ist das
// dasselbe wie die Abbildung der Ecken.
func (z zeichner) jeStueck(r document.Rechteck, f func(zs zeichner, abb func(document.Punkt) document.Punkt)) {
	for _, st := range z.d.Stuecke(r) {
		zs, ok := z.aufStueck(st)
		if !ok {
			continue
		}
		v := st.Versatz
		f(zs, func(p document.Punkt) document.Punkt { return p.Plus(v) })
	}
}

// aufStueck liefert den Zeichner, der auf den Bereich st beschnitten ist (Ziel in der Ansicht, ganze Pixel), und ob
// davon etwas im Bild liegt.
func (z zeichner) aufStueck(st document.Stueck) (zeichner, bool) {
	ziel := st.Ziel()
	zs := z
	zs.rand = image.Rect(int(math.Round(ziel.Min.X)), int(math.Round(ziel.Min.Y)),
		int(math.Round(ziel.Max.X)), int(math.Round(ziel.Max.Y))).Intersect(z.rand)
	return zs, !zs.rand.Empty()
}

// huelleVon ist das umschließende Rechteck der Punkte, um rand vergrößert (Original).
func huelleVon(pts []document.Punkt, rand float64) document.Rechteck {
	r := document.Rechteck{Min: pts[0], Max: pts[0]}
	for _, p := range pts[1:] {
		r = r.Vereint(document.Rechteck{Min: p, Max: p})
	}
	return document.Rechteck{Min: r.Min.Minus(document.Pt(rand, rand)), Max: r.Max.Plus(document.Pt(rand, rand))}
}

// ebene zeichnet o. Ist die Deckkraft der Ebene unter 100 % oder mischt sie multiplizierend (ADR-0048), entsteht das
// Objekt erst in einem durchsichtigen Zwischenbild in der Größe seines Umrisses und wird dann als Ganzes mit der
// Deckkraft eingemischt (Gruppendeckkraft, ADR-0032 Punkte 2 und 10): Schaft und Spitze, Schild und Schrift überlagern
// sich nicht doppelt. Im Trockenlauf zählt das Einmischen wie eine Füllung über dem Umriss.
func (z zeichner) ebene(o document.Objekt) error {
	p := o.Ebenenwerte().Deckkraft()
	schatten := o.Ebenenwerte().Schatten.Sichtbar()
	mult := o.Ebenenwerte().Multipliziert()
	if p >= 100 && !schatten && !mult {
		return z.objekt(o)
	}
	r := z.ebenenRahmen(o)
	if r.Empty() || p <= 0 {
		return nil
	}
	if schatten { // Schatten und Objekt zusammen im Zwischenbild (ADR-0021 Nachtrag 26 Punkt 3)
		return z.schattenEbene(o, r, p)
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
	mischeEbene(z.bild, zw.bild, p, mult)
	return nil
}

// ebenenRahmen ist der Umriss von o in der Ansicht, aufs Bild begrenzt, mit den Köpfen eines Pfeils (sie ragen über
// Grenzen hinaus, Pfeil.Form) und Rand für die Kantenglättung. Pfeil und Kasten bildet der Zeichner Punkt für Punkt ab, ihre Ecken reichen.
// Text und Nummer zeichnete er früher starr ab einem abgebildeten Anker; seit ADR-0023 Nachträgen 13 und 15 zeichnet er
// sie je sichtbarem Bereich, ihre Tinte liegt also innerhalb der abgebildeten Grenzen. Die Erweiterung um die starren
// Anker bleibt als großzügiges Obermaß (F-583).
func (z zeichner) ebenenRahmen(o document.Objekt) image.Rectangle {
	g := o.Grenzen()
	rand := 2.0
	if t, ok := o.(document.Text); ok && t.Spitze != nil {
		// Sprechblase: Schild plus Spitze, die Schenkel ragen mit halber Linienbreite darüber (ADR-0021 Nachtrag 13 Punkt 4).
		g = t.Huelle()
		rand += t.SchenkelBreite() / 2
	}
	switch v := o.(type) {
	case document.Pfeil:
		// Schaft und Köpfe aus derselben Gestalt wie das Zeichnen (ADR-0021 Nachtrag 19, Kritik K-5): Die Schenkel der
		// offenen Spitze reichen bis 4,67w+5 hinter das Ende, das frühere Obermaß max(8, 4w) schnitt sie ab.
		g = g.Vereint(v.Form().Huelle(v.Strich.Breite))
	case document.Kasten:
		// Der Strich liegt starr mit halber Breite um das abgebildete Rechteck; fällt seine Kante auf eine Naht, ragt er
		// über die abgebildeten Grenzen hinaus (Verify V23-1).
		rand += v.Strich.Breite / 2
	case document.Lupe: // Rahmen wie beim Kasten (ADR-0033 Punkt 4)
		rand += 0.5 * v.Strich.Breite
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
// mult mischt multiplizierend (ADR-0048 Punkt 1, mischeMultipliziert).
func mischeEbene(bild, q *image.NRGBA, p int, mult bool) {
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
			if mult {
				mischeMultipliziert(bild.Pix[bi+4*x:bi+4*x+4:bi+4*x+4], color.NRGBA{R: s[0], G: s[1], B: s[2], A: s[3]}, ma)
				continue
			}
			e := neuerEinmischer(bild, color.NRGBA{R: s[0], G: s[1], B: s[2], A: s[3]})
			e.misch(bild.Pix[bi+4*x:bi+4*x+4:bi+4*x+4], ma)
		}
	}
}

// mischeMultipliziert mischt die Quelle s mit der Deckung ma (16 Bit) multiplizierend in das Pixel p (NRGBA), in
// sRGB-Werten wie jede Mischung (ADR-0048 Punkt 1, W3C Compositing „multiply“ mit source-over). Vormultipliziert und
// auf 0xffff bezogen: co = cs·(1−ab) + cb·(1−as) + cs·cb, ao = as + ab − as·ab. Auf deckendem Grund ist das
// cb·(1 − as·(1 − Cs)) — dasselbe wie Gios Mischung src·DstColor + dst·(1 − srcA) (paint.PushMultiply).
func mischeMultipliziert(p []byte, s color.NRGBA, ma uint32) {
	sr, sg, sb, sa := s.RGBA()
	sr, sg, sb, sa = sr*ma/0xffff, sg*ma/0xffff, sb*ma/0xffff, sa*ma/0xffff
	if sa == 0 {
		return
	}
	dr, dg, db, da := color.NRGBA{R: p[0], G: p[1], B: p[2], A: p[3]}.RGBA()
	kanal := func(cs, cb uint32) uint32 {
		return (cs*(0xffff-da) + cb*(0xffff-sa) + cs*cb) / 0xffff
	}
	r, g, b := kanal(sr, dr), kanal(sg, dg), kanal(sb, db)
	al := sa + da - sa*da/0xffff
	r, g, b, al = min(r, al), min(g, al), min(b, al), min(al, 0xffff)
	switch al {
	case 0xffff:
		p[0], p[1], p[2], p[3] = uint8(r>>8), uint8(g>>8), uint8(b>>8), 0xff
	case 0:
		p[0], p[1], p[2], p[3] = 0, 0, 0, 0
	default:
		p[0], p[1], p[2], p[3] = uint8((r*0xffff/al)>>8), uint8((g*0xffff/al)>>8), uint8((b*0xffff/al)>>8), uint8(al>>8)
	}
}

func (z zeichner) objekt(o document.Objekt) error {
	switch v := o.(type) {
	case document.Pfeil:
		// Abgeschnitten wie im SVG-Export (F-748, ADR-0023 Nachtrag 17): Jeder sichtbare Bereich zeichnet die Strecken in
		// seiner Nähe um seinen Versatz verschoben und auf sich beschnitten (strichJeStueck). Über vielen Nähten baut so
		// kein Bereich den ganzen Pfeil (F-136). Die Köpfe ersetzen die runden Kappen des Schafts (ADR-0021 Nachtrag 19).
		z.pfeil(v)
	case document.Kasten:
		if v.Ellipse() {
			// Ellipse (ADR-0047 Punkt 2): je Bereich abgeschnitten wie der runde Kasten; Füllung im abgetasteten Umriss,
			// Strich als überlappungsfreie Teile des geschlossenen Linienzugs.
			b := v.Bereich
			strich := v.Strich.Breite > 0 && v.Strich.Farbe.A > 0
			hw := 0.0
			if strich {
				hw = v.Strich.Breite / 2
			}
			if !document.EllipseSichtbar(v, z.originalRahmen()) {
				break // Ring und Füllung berühren das Bild nicht: nicht abtasten (F-1023)
			}
			umriss := document.EllipsenUmriss(b)
			z.ellipsenKosten(len(umriss))
			if v.Fuellung.A > 0 {
				z.jeStueck(huelleVon(umriss, 1), func(zs zeichner, abb func(document.Punkt) document.Punkt) {
					pts := make([]document.Punkt, len(umriss))
					for i, p := range umriss {
						pts[i] = abb(p)
					}
					zs.fuelle([][]document.Punkt{pts}, nil, v.Fuellung)
				})
			}
			if strich && document.EllipsenRingSichtbar(b, hw+1, z.originalRahmen()) {
				sz := ZerlegeEllipse(b, v.Strich)
				z.ellipsenKosten(len(sz.Teile)) // auch beim vollen Strich: das Zerlegen kostet je Teil (F-1023)
				z.strichJeStueckMit(sz.ls, v.Strich, sz.Suche)
			}
			break
		}
		if b := document.R(v.Bereich.Min, v.Bereich.Max); document.WirksamerEckradius(v.Eckradius, b.Breite(), b.Hoehe()) > 0 {
			// Gerundet: je Bereich abgeschnitten wie im SVG-Export, nicht über die abgebildeten Ecken neu gerundet (F-749,
			// ADR-0023 Nachtrag 17). Ohne Naht ist das dasselbe.
			umriss := rundesRechteck(b, v.Eckradius)
			if v.Fuellung.A > 0 {
				z.jeStueck(huelleVon(umriss, 1), func(zs zeichner, abb func(document.Punkt) document.Punkt) {
					pts := make([]document.Punkt, len(umriss))
					for i, p := range umriss {
						pts[i] = abb(p)
					}
					zs.fuelle([][]document.Punkt{pts}, nil, v.Fuellung)
				})
			}
			if v.Strich.Breite > 0 && v.Strich.Farbe.A > 0 {
				if v.Strich.Art != document.StrichVoll { // gemustert (ADR-0021 Nachtrag 25)
					sz := ZerlegeRundenRahmen(b, v.Eckradius, v.Strich)
					z.musterKosten(v.Strich, len(sz.Teile))
					z.strichJeStueckMit(sz.ls, v.Strich, sz.Suche)
				} else {
					z.kastenStrich(document.NeuerKastenStrich(b, v.Eckradius, v.Strich.Breite/2), v.Strich.Farbe)
				}
			}
			break
		}
		r := document.R(z.ansicht(v.Bereich.Min), z.ansicht(v.Bereich.Max))
		umriss := rundesRechteck(r, v.Eckradius)
		if v.Fuellung.A > 0 {
			z.fuelle([][]document.Punkt{umriss}, nil, v.Fuellung)
		}
		if v.Strich.Breite > 0 && v.Strich.Farbe.A > 0 {
			if document.WirksamerEckradius(v.Eckradius, r.Breite(), r.Hoehe()) == 0 {
				// Ohne Rundung spitze Außenecken (F-498): Strich als überlappungsfreie Rechtecke, gemustert auch Scheiben
				// (ADR-0021 Nachtrag 25).
				z.rahmen(r, v.Strich)
			} else {
				z.strich(umriss, true, v.Strich)
			}
		}
	case document.Text:
		// Nur die Spitze sichtbar (Schild weggeschnitten oder ganz im Streifen): nur das Dreieck, keine Zeile (F-735).
		nurSpitze := z.d.NurSpitzeSichtbar(v)
		if sa, sp, sb, ok := v.Spitzendreieck(); ok {
			// Sprechblase (ADR-0021 Nachtrag 13 Punkte 2 und 3): Schild samt Dreieck als ein Vieleck in der Füllfarbe, ohne
			// Füllung nur die beiden Schenkel als Linie in der Schriftfarbe. Je sichtbarem Bereich gezeichnet und auf ihn
			// beschnitten wie im SVG-Export und im Treffer: Eine schräge Kante über einer Naht ist dort abgeschnitten, nicht
			// zwischen abgebildeten Ecken neu gezogen (F-695, ADR-0023 Nachtrag 14).
			umriss := document.Sprechblase(v)
			if nurSpitze || v.Hintergrund.A == 0 {
				umriss = []document.Punkt{sa, sp, sb}
			}
			z.jeStueck(huelleVon(umriss, v.SchenkelBreite()/2+1), func(zs zeichner, abb func(document.Punkt) document.Punkt) {
				pts := make([]document.Punkt, len(umriss))
				for i, p := range umriss {
					pts[i] = abb(p)
				}
				if v.Hintergrund.A > 0 {
					zs.fuelle([][]document.Punkt{pts}, nil, v.Hintergrund)
				} else {
					zs.strich(pts, false, document.Strich{Farbe: v.Farbe, Breite: v.SchenkelBreite()})
				}
			})
		} else if v.Hintergrund.A > 0 && !nurSpitze {
			g := v.Grenzen()
			r := document.R(z.ansicht(g.Min), z.ansicht(g.Max))
			z.fuelle([][]document.Punkt{rundesRechteck(r, 0)}, nil, v.Hintergrund)
		}
		if nurSpitze {
			return nil
		}
		var face font.Face
		if !z.trocken() {
			var err error
			if face, err = z.schrift(v.Schriftdatei(), v.Schriftgrad); err != nil {
				return err
			}
		}
		sch := v.Schriftdatei()
		// Glyphen je sichtbarem Bereich (document.TextStuecke, F-739, ADR-0023 Nachtrag 13): verschoben um dessen
		// Versatz und, wenn der Text über eine Naht reicht, auf den Bereich beschnitten. So ist Text an der Naht
		// abgeschnitten wie das Bild darunter, statt starr hinter der Naht zu stehen.
		// Immer auf den Bereich beschnitten, auch bei nur einem: Ein Text, der im Streifen endet, stünde sonst mit dem Teil
		// im Streifen hinter der Naht (Kritik A-1, [S]). Zeilenweise über die Abschnitte statt je Bereich über alle Zeilen,
		// jede Glyphe einmal gerastert (textBaender, F-747, F-750, ADR-0023 Nachtrag 16).
		xs, ys := z.d.TextAbschnitte(v)
		spalten := baender(xs, z.rand.Min.X, z.rand.Max.X)
		reihen := baender(ys, z.rand.Min.Y, z.rand.Max.Y)
		if len(spalten) > 0 && len(reihen) > 0 {
			oben, unten, seitlich := sch.Huelle()
			grad := v.Schriftgrad
			rechts := float64(spalten[len(spalten)-1].B) + seitlich*grad
			for _, zeile := range v.Zeilen() {
				// Sichtbar wie zeileSichtbar (Testhilfe): senkrecht in einem Abschnitt, Anfang nicht rechts aller Bereiche.
				rs := baenderIn(reihen, zeile.Grundlinie-grad*oben, zeile.Grundlinie+grad*unten)
				if len(rs) == 0 || zeile.X > rechts {
					continue
				}
				// Kosten einmal je sichtbarer Zeile, nicht je Bereich (F-136).
				z.auf.glyph += glyphKostenIn(sch, zeichenzahl(zeile.Text), grad)
				if z.trocken() {
					continue
				}
				// Tabulator als vier Leerzeichen, Wagenrücklauf weg (Zeichenbar in Zeilen): die Schriften haben für beide
				// keine Glyphe (Kritik A-3).
				z.textBaender(face, sch, zeile.Text, zeile.X, zeile.Grundlinie, v.Farbe, grad, spalten, rs)
			}
		}
		// Unter- und Durchstrich (ADR-0021 Nachtrag 10 Punkt 7): Rechtecke in der Textfarbe.
		for _, st := range v.Striche() {
			r := document.R(z.ansicht(st.Min), z.ansicht(st.Max))
			z.fuelle([][]document.Punkt{rundesRechteck(r, 0)}, nil, v.Farbe)
		}
	case document.Nummer:
		// Je sichtbarem Bereich und auf ihn beschnitten wie der SVG-Export (F-745, ADR-0023 Nachtrag 15): der Kreis je
		// Bereich, die Ziffern zeilenweise über die Abschnitte, jede einmal gerastert (F-750, ADR-0023 Nachtrag 16).
		z.jeStueck(document.Reichweite(v), func(zs zeichner, abb func(document.Punkt) document.Punkt) {
			zs.fuelle(nil, []kreisform{{abb(v.Mitte), v.Radius}}, v.Farbe)
		})
		if err := z.nummer(v); err != nil {
			return err
		}
	case document.Lupe:
		z.lupe(v)
	}
	return nil
}

// nummer zeichnet die Ziffern der Schritt-Nummer v in jeden sichtbaren Bereich (document.Abschnitte), weiß, mittig um
// die Mitte. Die Ziffern stehen weniger als len(s) × 0,8 em breit (Go Bold, mit Rand) und nicht höher als der Schriftgrad
// um die Mitte; berührt dieses Rechteck keinen Bereich, kosten sie nichts und werden nicht gezeichnet. Sonst zählen sie
// einmal, wie Text (F-750).
func (z zeichner) nummer(v document.Nummer) error {
	s := strconv.Itoa(v.Wert)
	grad := v.Radius * 1.2
	halb := grad * (0.4*float64(len(s)) + 1)
	m := v.Mitte
	xs, ys := z.d.Abschnitte(document.Reichweite(v))
	spalten := baenderIn(baender(xs, z.rand.Min.X, z.rand.Max.X), math.Floor(m.X-halb), math.Ceil(m.X+halb))
	reihen := baenderIn(baender(ys, z.rand.Min.Y, z.rand.Max.Y), math.Floor(m.Y-grad), math.Ceil(m.Y+grad))
	if len(spalten) == 0 || len(reihen) == 0 {
		return nil
	}
	z.auf.glyph += glyphKosten(len(s), grad) // Ziffern immer in Go Fett
	if z.trocken() {
		return nil
	}
	sch := fonts.Laden(fonts.Go, fonts.Fett)
	face, err := z.schrift(sch, grad)
	if err != nil {
		return err
	}
	breite := zeilenBreite(face, sch, s, grad)
	// Senkrecht mittig wie dominant-baseline="central": Grundlinie um die halbe Ziffernhöhe unter der Mitte.
	hoehe := float64(face.Metrics().CapHeight) / 64
	z.textBaender(face, sch, s, m.X-breite/2, m.Y+hoehe/2, color.NRGBA{R: 255, G: 255, B: 255, A: 255}, grad, spalten, reihen)
	return nil
}

// zeilenBreite ist die Breite von s in Pixeln, wie text sie setzt: Vorschub je Zeichen und Paarunterschneidung im Grad
// (document.KernPaar). font.MeasureString nähme face.Kern, der nicht mit dem Grad skaliert (F-577).
func zeilenBreite(face font.Face, sch *fonts.Schrift, s string, grad float64) float64 {
	var puffer sfnt.Buffer
	var b fixed.Int26_6
	vorher := rune(-1)
	for _, c := range s {
		if vorher >= 0 {
			b += document.KernPaar(sch.Font, &puffer, vorher, c, grad)
		}
		vorher = c
		a, _ := face.GlyphAdvance(c)
		b += a
	}
	return float64(b) / 64
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
	z.auf.kanten += aufrufGewicht + punktGewicht*float64(punkte) +
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

// strich zieht einen Linienzug (Ansicht) der Breite s.Breite mit runden Enden und Gelenken: die überlappungsfreien
// Teile aus neuerLinienStrich in einer Füllung (F-796, ADR-0023 Nachtrag 22).
func (z zeichner) strich(pts []document.Punkt, geschlossen bool, s document.Strich) {
	z.fuelle(neuerLinienStrich(pts, geschlossen, s.Breite/2).teile, nil, s.Farbe)
}

// strichJeStueck zeichnet den Strich der Linie pts (Original) wie strich, aber je sichtbarem Bereich: nur die Teile der
// Strecken in seiner Nähe (document.StreckenStuecke) samt Fächern und Kappen an ihren Enden, um seinen Versatz
// verschoben und auf ihn beschnitten. Über einer Naht ist der Strich so abgeschnitten wie im SVG-Export (F-748,
// ADR-0023 Nachtrag 17); ohne Naht ist es eine Füllung aller Teile wie strich. Jeder Teil liegt höchstens hw von seiner
// Strecke entfernt; die Suche mit hw+1 findet also jeden Bereich, den er berührt.
func (z zeichner) strichJeStueck(pts []document.Punkt, geschlossen bool, s document.Strich) {
	z.strichJeStueckMit(neuerLinienStrich(pts, geschlossen, s.Breite/2), s, s.Breite/2+1)
}

// strichJeStueckMit ist strichJeStueck für den fertig zerlegten Strich ls; suche ist der Abstand, bis zu dem ein Teil von
// seiner Strecke reicht, plus 1 (bei Köpfen eines Pfeils größer als die halbe Breite).
func (z zeichner) strichJeStueckMit(ls linienStrich, s document.Strich, suche float64) {
	if len(ls.teile) == 0 {
		return
	}
	pts, geschlossen := ls.pts, ls.zu // ohne doppelte Punkte; die Strecken zählen wie ls.strecke
	n := len(pts)
	gesehen, marke := make([]int, n), 0
	// Ein Puffer für die verschobenen Ecken, für jeden Bereich wiederverwendet (über vielen Nähten sonst eine Zuweisung je
	// Teil und Bereich): so groß wie alle Teile zusammen, je Bereich kommt jeder Teil höchstens einmal hinein, Anhängen
	// zieht also nie um. Die Füllung hält nichts davon über ihren Aufruf hinaus fest.
	gesamt := 0
	for _, t := range ls.teile {
		gesamt += len(t)
	}
	punkte := make([]document.Punkt, 0, gesamt)
	var polys [][]document.Punkt
	for _, st := range z.d.StreckenStuecke(pts, geschlossen, suche) {
		zs, ok := z.aufStueck(st.Stueck)
		if !ok {
			continue
		}
		marke++
		v := st.Versatz
		punkte, polys = punkte[:0], polys[:0]
		verschoben := func(t []document.Punkt) {
			von := len(punkte)
			for _, p := range t {
				punkte = append(punkte, p.Plus(v))
			}
			polys = append(polys, punkte[von:len(punkte):len(punkte)])
		}
		for _, i := range st.Strecken {
			if i < len(ls.strecke) && ls.strecke[i] >= 0 { // −1: Lücke eines gemusterten Strichs
				verschoben(ls.teile[ls.strecke[i]])
			}
			for _, k := range []int{i, (i + 1) % n} {
				if gesehen[k] != marke && ls.gelenk[k] >= 0 {
					gesehen[k] = marke // Fächer und Kappen einmal je Bereich
					verschoben(ls.teile[ls.gelenk[k]])
				}
			}
		}
		zs.fuelle(polys, nil, s.Farbe)
	}
}

// kastenStrich zeichnet den Strich eines runden Kastens aus seinen Teilen ks (Original; document.NeuerKastenStrich)
// je sichtbarem Bereich, den sein Umriss berührt (document.StreckenStuecke mit ks.Rand): dort die Teile der Strecken in
// der Nähe, um den Versatz verschoben und auf den Bereich beschnitten, in einer Füllung. Die Teile überlappen nicht und
// laufen gleich herum; ihre gemeinsamen Kanten heben sich im Rasterizer genau auf, die Füllung deckt jedes Pixel wie die
// Fläche des Strichs (F-788, ADR-0023 Nachtrag 21). Vorher füllte der Renderer je Strecke ein Rechteck und je Gelenk
// einen Kreis; deren Kantenglättung addierte sich, die äußeren Randpixel einer Rundung standen bis 255 statt rund 128.
// Editor und Vorschau der Ebenenliste füllen dieselben Teile.
func (z zeichner) kastenStrich(ks document.KastenStrich, farbe color.NRGBA) {
	if len(ks.Teile) == 0 {
		return
	}
	gesehen, marke := make([]int, len(ks.Teile)), 0
	for _, st := range z.d.StreckenStuecke(ks.Umriss, true, ks.Rand) {
		zs, ok := z.aufStueck(st.Stueck)
		if !ok {
			continue
		}
		marke++
		var polys [][]document.Punkt
		for _, i := range st.Strecken {
			t := ks.TeilVon[i]
			if gesehen[t] == marke {
				continue // Fächer mehrerer Strecken: einmal je Bereich
			}
			gesehen[t] = marke
			teil := make([]document.Punkt, len(ks.Teile[t]))
			for k, p := range ks.Teile[t] {
				teil[k] = p.Plus(st.Versatz)
			}
			polys = append(polys, teil)
		}
		zs.fuelle(polys, nil, farbe)
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

// rundesRechteck liefert den Umriss eines Rechtecks mit Eckradius (auf die halbe kürzere Seite begrenzt); dieselbe
// Rechnung wie die Bereichssuche des Editors (document.RundesRechteck, F-765).
func rundesRechteck(r document.Rechteck, radius float64) []document.Punkt {
	return document.RundesRechteck(r, radius)
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
