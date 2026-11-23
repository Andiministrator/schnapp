// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0023 Punkt 6, Nachtrag 6: Der SVG-Export zeichnet je Naht alle Schichten der Kante (document.KantenSchichten) mit
// denselben Punkten wie der Renderer, in der Form des Dokuments, halbdurchsichtige mit fill-opacity, ohne Filter, in
// Ansichtskoordinaten nach den Kacheln (also obenauf); ein Streifen am Rand ergibt keine. Seit Nachtrag 7 bleibt der
// Spalt leer (pruefeSpaltLeer).
func TestExportSVGKante(t *testing.T) {
	for _, form := range []document.Kante{document.KanteAbriss, document.KanteWelle, document.KanteZacken} {
		d, err := document.Neu(testbild(80, 60))
		if err != nil {
			t.Fatal(err)
		}
		if err := d.VorgabeKante(form); err != nil {
			t.Fatal(err)
		}
		if err := d.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 20, Bis: 25}, {Von: 30, Bis: 33},
			{Von: 77, Bis: 80}}); err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		if err := ExportiereSVG(&buf, d); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		if strings.Contains(s, "filter") || strings.Contains(s, "s:") {
			t.Errorf("%v: Filter oder s:-Attribut im Export", form)
		}
		// Je Achse ein Band bei Lage 0 in <defs> (F-183), je Naht ein <use>, das es auf die Lage verschiebt.
		gruppen := regexp.MustCompile(`<g id="(?:naht|schatten)-(zeilen|spalten)">(.*?)</g>`).FindAllStringSubmatch(s, -1)
		verweise := regexp.MustCompile(`<use href="#naht-(zeilen|spalten)" (x|y)="(-?\d+)"/>`).FindAllStringSubmatch(s, -1)
		// Nähte beider Achsen berühren sich (F-438): je Achse zwei Gruppen (Saum und Abdunklung, Schatten), die
		// Verweise der ersten unter der Maske, die des Schattens danach.
		if len(gruppen) != 4 || len(verweise) != 2 {
			t.Fatalf("%v: %d Bänder, %d Nähte im Export, Soll 4 und 2:\n%.2000s", form, len(gruppen), len(verweise), s)
		}
		if !regexp.MustCompile(`<g mask="url\(#spalt\)"><use href="#naht-spalten" x="\d+"/><use href="#naht-zeilen" y="\d+"/></g>` +
			`<use href="#schatten-spalten" x="\d+"/><use href="#schatten-zeilen" y="\d+"/></svg>`).MatchString(s) {
			t.Errorf("%v: Verweise unter Maske und Schatten fehlen am Ende:\n%.400s", form, s[len(s)-400:])
		}
		// Je Achse ein Pfad je Breite (band-<Achse>-<Breite>, ohne Füllung), je Stück der Kante (document.KantenFlach) ein
		// <use> darauf mit Versatz, Farbe und Deckkraft (F-422).
		basisRe := regexp.MustCompile(`<path id="band-(zeilen|spalten)-(\d+)" d="([^"]*)"/>`)
		basen := map[string]string{}
		for _, m := range basisRe.FindAllStringSubmatch(s, -1) {
			basen[m[1]+"-"+m[2]] = m[3]
		}
		stueckRe := regexp.MustCompile(`<use href="#band-(zeilen|spalten)-(\d+)" (?:x|y)="(-?[0-9.e+-]+)" fill="#([0-9a-f]{6})"(?: fill-opacity="([0-9.e+-]+)")?/>`)
		stuecke := document.KantenFlach()
		g := d.Groesse()
		for i, n := range d.Naehte() {
			art, achse, laenge := "spalten", "x", g.Y
			if n.Zeilen {
				art, achse, laenge = "zeilen", "y", g.X
			}
			if v := verweise[i]; v[1] != art || v[2] != achse || v[3] != strconv.Itoa(n.Lage) {
				t.Errorf("%v Naht %d: %v", form, i, v[0])
			}
			var grp string // Saum und Abdunklung, dann der Schatten: die Reihenfolge der Stücke
			for _, gr := range gruppen {
				if gr[1] == art {
					grp += gr[2]
				}
			}
			use := stueckRe.FindAllStringSubmatch(grp, -1)
			if len(use) != len(stuecke) {
				t.Fatalf("%v Band %s: %d Stücke, Soll %d", form, art, len(use), len(stuecke))
			}
			f := d.KantenFaktor()
			for j, sch := range stuecke {
				u := use[j]
				if want := fmt.Sprintf("%02x%02x%02x", sch.Farbe.R, sch.Farbe.G, sch.Farbe.B); u[4] != want {
					t.Errorf("%v Stück %d: Farbe %s, Soll %s", form, j, u[4], want)
				}
				if want := strconv.FormatFloat(float64(sch.Farbe.A)/255, 'g', -1, 64); u[5] != want {
					t.Errorf("%v Stück %d: Deckkraft %q, Soll %q", form, j, u[5], want)
				}
				vonH, bisH := int(math.Round(sch.Von*100)), int(math.Round(sch.Bis*100))
				if want := strconv.FormatFloat(float64(vonH*f)/100, 'g', -1, 64); u[3] != want {
					t.Errorf("%v Stück %d: Versatz %s, Soll %s", form, j, u[3], want)
				}
				if u[1] != art || u[2] != strconv.Itoa(bisH-vonH) {
					t.Errorf("%v Stück %d: Verweis auf %s-%s, Soll %s-%d", form, j, u[1], u[2], art, bisH-vonH)
				}
				// Der Pfad ist das Band von 0 bis zur Breite der vereinfachten Linie (dieselbe wie im Export) ...
				bei0 := document.Naht{Zeilen: n.Zeilen}
				linie := document.NeueKantenlinie(form, laenge, f).Vereinfacht(KantenToleranz)
				soll := linie.Band(nil, bei0, 0, float64(bisH-vonH)/100)
				var teile []string
				for _, q := range soll {
					teile = append(teile, punkt(q))
				}
				if basen[art+"-"+u[2]] != "M"+strings.Join(teile, " L")+" Z" {
					t.Errorf("%v Band %s Stück %d: %.200s", form, art, j, basen[art+"-"+u[2]])
				}
				// ... und weicht von der vollen Linie des Renderers an keiner Stelle um mehr als die Toleranz ab.
				voll := document.Kantenband(form, bei0, laenge, f, 0, float64(bisH-vonH)/100)
				abweichung := kantenAbweichung(voll[:len(voll)/2], soll[:len(soll)/2], n.Zeilen)
				if abweichung > KantenToleranz+1e-9 {
					t.Errorf("%v Band %s Stück %d: Abweichung %v > %v", form, art, j, abweichung, KantenToleranz)
				}
			}
		}
		if strings.LastIndex(s, `<use href="#objekte"/>`) > strings.Index(s, `<use href="#naht-`) {
			t.Errorf("%v: Kante liegt unter einer Kachel", form)
		}
		pruefeSpaltLeer(t, form, s, d)
	}
}

// kantenAbweichung ist der größte Querabstand der Punkte voll von dem Linienzug einfach (beide entlang der Naht
// aufsteigend, einfach hat weniger Punkte): Abstand quer zur Naht an der Stelle des jeweiligen Punkts.
func kantenAbweichung(voll, einfach []document.Punkt, zeilen bool) float64 {
	entlang := func(p document.Punkt) (t, q float64) {
		if zeilen {
			return p.X, p.Y
		}
		return p.Y, p.X
	}
	weit, k := 0.0, 0
	for _, p := range voll {
		t, q := entlang(p)
		for k+2 < len(einfach) {
			if t2, _ := entlang(einfach[k+1]); t2 >= t {
				break
			}
			k++
		}
		t0, q0 := entlang(einfach[k])
		t1, q1 := entlang(einfach[k+1])
		soll := q0 + (q1-q0)*(t-t0)/(t1-t0)
		weit = max(weit, math.Abs(q-soll))
	}
	return weit
}

// ADR-0023 Nachtrag 7: Der Spalt bleibt im SVG leer. Je Achse steht er einmal als schwarzer Pfad in <defs> (dieselben
// Punkte wie das Spaltband ±8 des Renderers), die Maske „spalt“ ist weiß über die ganze Ansicht und verweist je Naht auf
// ihn; alle Kacheln (Bild und Objekte) liegen unter dieser Maske, die Schichten darüber. Außerhalb der Maske wird der
// Spalt nirgends gemalt, und keine Schicht ist deckend (die alte graue Spaltfüllung).
func pruefeSpaltLeer(t *testing.T, form document.Kante, s string, d *document.Dokument) {
	t.Helper()
	g := d.Groesse()
	spaltRe := regexp.MustCompile(`<path id="spalt-(zeilen|spalten)" d="([^"]*)" fill="#000"/>`)
	pfade := map[string]string{}
	for _, m := range spaltRe.FindAllStringSubmatch(s, -1) {
		pfade[m[1]] = m[2]
	}
	if len(pfade) != 2 {
		t.Fatalf("%v: %d Spaltpfade, Soll 2", form, len(pfade))
	}
	for art, zeilen := range map[string]bool{"zeilen": true, "spalten": false} {
		laenge := g.Y
		if zeilen {
			laenge = g.X
		}
		var soll []string
		bei0 := document.Naht{Zeilen: zeilen}
		voll := document.Kantenband(form, bei0, laenge, d.KantenFaktor(), -8, 8)
		vereinfacht := document.NeueKantenlinie(form, laenge, d.KantenFaktor()).Vereinfacht(KantenToleranz).Band(nil, bei0, -8, 8)
		if a := kantenAbweichung(voll[:len(voll)/2], vereinfacht[:len(vereinfacht)/2], zeilen); a > KantenToleranz+1e-9 {
			t.Errorf("%v Spalt %s: Abweichung %v > %v", form, art, a, KantenToleranz)
		}
		for _, q := range vereinfacht {
			soll = append(soll, punkt(q))
		}
		if pfade[art] != "M"+strings.Join(soll, " L")+" Z" {
			t.Errorf("%v Spalt %s: %.200s", form, art, pfade[art])
		}
	}
	maske := regexp.MustCompile(`<mask id="spalt" maskUnits="userSpaceOnUse" x="0" y="0" width="(\d+)" height="(\d+)">` +
		`<rect width="(\d+)" height="(\d+)" fill="#fff"/>((?:<use href="#spalt-(?:zeilen|spalten)" (?:x|y)="-?\d+"/>)*)</mask>`).
		FindStringSubmatch(s)
	if maske == nil {
		t.Fatalf("%v: Maske fehlt:\n%.3000s", form, s)
	}
	if b := strconv.Itoa(g.X); maske[1] != b || maske[3] != b || maske[2] != strconv.Itoa(g.Y) || maske[4] != strconv.Itoa(g.Y) {
		t.Errorf("%v: Maske %s×%s, Soll %v", form, maske[1], maske[2], g)
	}
	var soll strings.Builder
	for _, n := range d.Naehte() {
		if n.Zeilen {
			fmt.Fprintf(&soll, `<use href="#spalt-zeilen" y="%d"/>`, n.Lage)
		} else {
			fmt.Fprintf(&soll, `<use href="#spalt-spalten" x="%d"/>`, n.Lage)
		}
	}
	if maske[5] != soll.String() {
		t.Errorf("%v: Verweise der Maske %q, Soll %q", form, maske[5], soll.String())
	}
	if strings.Count(s, `href="#spalt-`) != len(d.Naehte()) {
		t.Errorf("%v: Spalt außerhalb der Maske verwendet", form)
	}
	// Alle Kacheln unter der Maske: die erste Kachel nach dem Öffnen der Gruppe, die letzte vor ihrem Schluss.
	auf := strings.Index(s, `<g mask="url(#spalt)">`)
	zu := strings.Index(s, `</g><use href="#naht-`)
	if zu < 0 { // berühren sich Nähte (F-438), liegen die Schichten unter einer Maske
		zu = strings.Index(s, `</g><g mask="url(#spalt)"><use href="#naht-`)
	}
	if auf < 0 || zu < auf || strings.Index(s, `<image`) < auf || strings.LastIndex(s, `<image`) > zu ||
		strings.Count(s, `mask="url(#spalt)"`) != 1+strings.Count(s, `<g mask="url(#spalt)"><use`) {
		t.Errorf("%v: Kacheln nicht unter der Maske", form)
	}
	if strings.Contains(s, `fill="#d3d7dc"`) {
		t.Errorf("%v: graue Spaltfüllung noch im Export", form)
	}
}

// F-438 (ADR-0023 Nachtrag 9): Eine einzelne Naht oder weit auseinanderliegende bleiben im Export wie zuvor (eine
// Gruppe, ein Verweis, keine zweite Maske); kreuzen oder berühren sich Nähte, liegen Saum und Abdunklung unter der
// Maske und der Schatten in einer eigenen Gruppe darüber.
func TestExportSVGNaehteBeruehrenSchatten(t *testing.T) {
	export := func(st ...document.Streifen) string {
		d, err := document.Neu(testbild(200, 200))
		if err != nil {
			t.Fatal(err)
		}
		if err := d.SetzeStreifen(st); err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		if err := ExportiereSVG(&buf, d); err != nil {
			t.Fatal(err)
		}
		return buf.String()
	}
	for name, c := range map[string]struct {
		st   []document.Streifen
		soll bool
	}{
		"eine Naht":        {[]document.Streifen{{Zeilen: true, Von: 50, Bis: 55}}, false},
		"weit auseinander": {[]document.Streifen{{Zeilen: true, Von: 40, Bis: 45}, {Zeilen: true, Von: 120, Bis: 125}}, false},
		"eng beieinander":  {[]document.Streifen{{Zeilen: true, Von: 40, Bis: 45}, {Zeilen: true, Von: 60, Bis: 65}}, true},
		"kreuzend":         {[]document.Streifen{{Zeilen: true, Von: 40, Bis: 45}, {Von: 100, Bis: 105}}, true},
	} {
		s := export(c.st...)
		hat := strings.Contains(s, `id="schatten-`) && strings.Contains(s, `<g mask="url(#spalt)"><use`) &&
			strings.Contains(s, `<use href="#schatten-`)
		if hat != c.soll {
			t.Errorf("%s: Schattengruppe %v, Soll %v", name, hat, c.soll)
		}
		if !c.soll && (strings.Contains(s, "schatten") || strings.Count(s, `mask="url(#spalt)"`) != 1) {
			t.Errorf("%s: Export weicht vom Zustand vor F-438 ab", name)
		}
	}
}

// Ohne Naht kein Band, keine Maske (ADR-0023 Nachtrag 7): Der Export bleibt wie zuvor.
func TestExportSVGOhneNahtOhneMaske(t *testing.T) {
	d, err := document.Neu(testbild(40, 30))
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	if s := buf.String(); strings.Contains(s, "mask") || strings.Contains(s, "spalt") || strings.Contains(s, "naht") {
		t.Errorf("ohne Naht Kante im Export:\n%s", s)
	}
}

// F-183: Viele Nähte kosten im SVG je Naht nur eine kurze Zeile (vorher 41 MB für 1023 Zeilen-Nähte in 8192 px
// Breite): ein Band je Achse in <defs>, je Naht ein <use>. Das Band selbst (alle Schichten des Abrisses, je Pixel ein
// Stützpunkt, ADR-0023 Nachtrag 6) steht einmal und hält sich mit Hundertsteln kurz.
func TestExportSVGVieleNaehteKompakt(t *testing.T) {
	export := func(anzahl int) (gesamt, band int) {
		d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 8192, 2047)))
		if err != nil {
			t.Fatal(err)
		}
		var st []document.Streifen
		for i := 0; i < anzahl; i++ {
			st = append(st, document.Streifen{Zeilen: true, Von: 1 + 2*i, Bis: 2 + 2*i})
		}
		if err := d.SetzeStreifen(st); err != nil {
			t.Fatal(err)
		}
		if n := len(d.Naehte()); n != anzahl {
			t.Fatalf("%d Nähte, Soll %d", n, anzahl)
		}
		var buf bytes.Buffer
		if err := ExportiereSVG(&buf, d); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		// Seit ADR-0023 Nachtrag 7 stehen Bänder und Maske vor den Kacheln, die Verweise der Schichten danach. Gezählt
		// wird ohne die Kacheln (deren PNG wächst mit jeder Naht um eine Kachel, nicht durch die Kante).
		start := strings.Index(s, `<defs><path id="band-`)
		maske := strings.Index(s, `<mask id="spalt"`)
		kachelnEnde := strings.Index(s, `</g><use href="#naht-`)
		if kachelnEnde < 0 { // eng beieinander (F-438): Schichten unter Maske
			kachelnEnde = strings.Index(s, `</g><g mask="url(#spalt)"><use href="#naht-`)
		}
		if start < 0 || maske < start || kachelnEnde < maske {
			t.Fatal("keine Nähte im Export")
		}
		ende := maske + strings.Index(s[maske:], `</defs>`)
		return (ende - maske) + (len(s) - kachelnEnde), maske - start
	}
	eins, band1 := export(2) // zwei eng liegende Nähte berühren sich schon (F-438), wie die 1023
	viele, band := export(1023)
	if band != band1 {
		t.Errorf("Band in <defs> hängt an der Zahl der Nähte: %d gegen %d Byte", band, band1)
	}
	// Je Naht drei Verweise: einer in der Maske (Spalt), einer auf Saum und Abdunklung, einer auf den Schatten (die Nähte
	// liegen hier eng beieinander, F-438, ADR-0023 Nachtrag 9); vor Nachtrag 7 einer,
	// Grenze damals 64 Byte.
	if je := (viele - eins) / 1021; je > 120 {
		t.Errorf("je weitere Naht %d Byte, Soll höchstens 120", je)
	}
	const soll = 4 << 20
	if band > soll {
		t.Errorf("Band belegt %d Byte, Soll höchstens %d", band, soll)
	}
}

// Rundlauf (ADR-0023): Zuschnitt und Streifen — auch verborgene außerhalb des Zuschnitts und anstoßende — kommen
// unverändert zurück, die Ansicht samt Nähten ist dieselbe, das sichtbare Bild der Datei ist das übergebene.
func TestRundlaufZuschnittUndStreifen(t *testing.T) {
	d, err := document.Neu(testbild(80, 60))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Von: 2, Bis: 4}, {Von: 30, Bis: 33}, {Von: 33, Bis: 35},
		{Zeilen: true, Von: 40, Bis: 45}}); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeZuschnitt(image.Rect(10, 5, 70, 55)); err != nil { // der Streifen 2..4 liegt außerhalb
		t.Fatal(err)
	}
	daten := schreibe(t, d, meta)
	g, err := LeseArbeitsdatei(bytes.NewReader(daten))
	if err != nil {
		t.Fatal(err)
	}
	e := g.Dokument
	if e.Zuschnitt() != d.Zuschnitt() || !reflect.DeepEqual(e.Streifen(), d.Streifen()) {
		t.Errorf("Ansicht: %v %v, Soll %v %v", e.Zuschnitt(), e.Streifen(), d.Zuschnitt(), d.Streifen())
	}
	if e.Groesse() != d.Groesse() || !reflect.DeepEqual(e.Naehte(), d.Naehte()) {
		t.Errorf("Größe %v Nähte %v, Soll %v %v", e.Groesse(), e.Naehte(), d.Groesse(), d.Naehte())
	}
	if !bytes.Equal(e.Ansichtsbild().Pix, d.Ansichtsbild().Pix) {
		t.Error("Ansichtsbild weicht ab")
	}
	sichtbar, err := LeseBild(bytes.NewReader(daten))
	if err != nil {
		t.Fatal(err)
	}
	if !gleich(sichtbar, d.Ansichtsbild()) {
		t.Error("sichtbares Bild der Datei weicht ab")
	}
}

// F-173: Bei 2200×2200 Ansicht gilt Faktor 2; das Band in <defs> hat doppelte Zackenhöhe und Spaltbreite (Form Zacken,
// Naht bei Lage 0: erster Punkt des Spalts bei x = -8 - 16 = -24 statt -4 - 8 = -12, der zweite bei y = 16; ADR-0023
// Nachtrag 7).
func TestExportSVGKanteFaktor2(t *testing.T) {
	d, err := document.Neu(testbild(2200, 2200))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.VorgabeKante(document.KanteZacken); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Von: 100, Bis: 110}}); err != nil {
		t.Fatal(err)
	}
	if f := d.KantenFaktor(); f != 2 {
		t.Fatalf("Faktor %d, Soll 2", f)
	}
	var buf bytes.Buffer
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	// Spalt ±8 um die Zacken ±4, beides doppelt: erster Punkt bei x = −8 − 16 = −24, der zweite bei y = 16 und x = 8 − 16.
	if !strings.Contains(buf.String(), `<path id="spalt-spalten" d="M-24,0 L-8,16 `) {
		t.Errorf("Band bei Faktor 2 nicht gefunden:\n%.600s", buf.String())
	}
}
