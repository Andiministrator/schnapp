// SPDX-License-Identifier: GPL-3.0-or-later

package annotate

import (
	"context"
	"errors"
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

func dokument(t *testing.T, w, h int) *document.Dokument {
	t.Helper()
	o := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := range o.Pix {
		o.Pix[i] = 0x80
	}
	d, err := document.Neu(o)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func lies(t *testing.T, s string) []Eintrag {
	t.Helper()
	es, err := Lies(strings.NewReader(s))
	if err != nil {
		t.Fatalf("Lies: %v", err)
	}
	return es
}

// TestLiesStreng: Form der Liste (ADR-0030 Nachtrag 10 Punkt 2) — unbekannte oder unpassende Schlüssel, fehlende
// Pflichtwerte, null, Typfehler und Restdaten sind Fehler mit Index.
func TestLiesStreng(t *testing.T) {
	for _, tc := range []struct {
		spec, teil string
		index      int
	}{
		{`{}`, `missing "annotations"`, -1},
		{`{"annotations":[],"x":1}`, "invalid JSON", -1},
		{`{"annotations":[]} {}`, "invalid JSON", -1},
		{`[]`, "invalid JSON", -1},
		{`{"annotations":[{"type":"circle"}]}`, "unknown type", 0},
		{`{"annotations":[{"type":"arrow","from":[1,1]}]}`, `needs key "to"`, 0},
		{`{"annotations":[{"type":"step","at":[1,1]},{"type":"arrow","from":[1,1],"to":[2,2],"fill":"#000000"}]}`, `"fill" does not apply`, 1},
		{`{"annotations":[{"type":"arrow","from":[1,1],"to":[2,2],"colour":"#000000"}]}`, "invalid JSON", 0},
		{`{"annotations":[{"type":"step","at":null}]}`, "must not be null", 0},
		{`{"annotations":[{"type":"step","at":[1]}]}`, "invalid JSON", 0},
		{`{"annotations":[{"type":"step","at":"1,1"}]}`, "invalid JSON", 0},
	} {
		_, err := Lies(strings.NewReader(tc.spec))
		var f *Fehler
		if !errors.As(err, &f) || f.Index != tc.index || !strings.Contains(err.Error(), tc.teil) {
			t.Errorf("%s: %v (Index erwartet %d, Teil %q)", tc.spec, err, tc.index, tc.teil)
		}
	}
	if _, err := Lies(strings.NewReader(`{"annotations":[` + strings.Repeat(" ", MaxBytes) + `]}`)); err == nil ||
		!strings.Contains(err.Error(), "larger than") {
		t.Errorf("zu groß: %v", err)
	}
}

// TestWendeVoreinstellungen: Ohne Angaben gelten die Voreinstellungen des Editors — Pfeil und Kasten rot (Kasten ohne
// Füllung), Text weiß auf Rot, Nummer rot mit der nächsten Zahl (ADR-0021 Nachträge 7 und 11).
func TestWendeVoreinstellungen(t *testing.T) {
	d := dokument(t, 200, 100)
	es := lies(t, `{"annotations":[
		{"type":"arrow","from":[10,10],"to":[60,40]},
		{"type":"rect","from":[70,10],"to":[120,50]},
		{"type":"text","at":[10,60],"text":"Hallo"},
		{"type":"step","at":[150,50]},
		{"type":"step","at":[170,50]}]}`)
	erg, err := Wende(context.Background(), d, es)
	if err != nil || erg.Beschriftungen != 5 || erg.Schwaerzungen != 0 {
		t.Fatalf("%+v, %v", erg, err)
	}
	st := canvas.StilMitFaktor(d.KantenFaktor())
	rot := canvas.Beschriftungsfarben[0]
	o := d.Objekte()
	if p := o[0].(document.Pfeil); p.Strich != (document.Strich{Farbe: rot, Breite: st.Breite}) || p.A != document.Pt(10, 10) {
		t.Errorf("Pfeil %+v", p)
	}
	if k := o[1].(document.Kasten); k.Strich.Farbe != rot || k.Fuellung.A != 0 || k.Bereich != document.R(document.Pt(70, 10), document.Pt(120, 50)) {
		t.Errorf("Kasten %+v", k)
	}
	if x := o[2].(document.Text); x.Farbe != canvas.Weiss || x.Hintergrund != rot || x.Schriftgrad != st.Schriftgrad || x.Inhalt != "Hallo" {
		t.Errorf("Text %+v", x)
	}
	if n1, n2 := o[3].(document.Nummer), o[4].(document.Nummer); n1.Wert != 1 || n2.Wert != 2 || n1.Farbe != rot || n1.Radius != st.Radius {
		t.Errorf("Nummern %+v %+v", n1, n2)
	}
}

// TestWendeAngaben: Farben, Füllung „none“, Größen, Schrift und Ausrichtung kommen an.
func TestWendeAngaben(t *testing.T) {
	d := dokument(t, 200, 100)
	es := lies(t, `{"annotations":[
		{"type":"rect","from":[10,10],"to":[60,40],"color":"#112233","fill":"#44556680","width":7},
		{"type":"text","at":[10,60],"text":"a\nb","size":30,"color":"#000000","fill":"none","font":"Go Mono",
		 "bold":true,"italic":true,"underline":true,"strikethrough":true,"align":"right"},
		{"type":"step","at":[150,50],"color":"#2F6FDB","radius":20}]}`)
	if _, err := Wende(context.Background(), d, es); err != nil {
		t.Fatal(err)
	}
	o := d.Objekte()
	if k := o[0].(document.Kasten); k.Strich != (document.Strich{Farbe: color.NRGBA{0x11, 0x22, 0x33, 255}, Breite: 7}) ||
		k.Fuellung != (color.NRGBA{0x44, 0x55, 0x66, 0x80}) {
		t.Errorf("Kasten %+v", k)
	}
	x := o[1].(document.Text)
	if x.Farbe != (color.NRGBA{A: 255}) || x.Hintergrund.A != 0 || x.Schriftgrad != 30 || x.Schrift != "Go Mono" ||
		!x.Fett || !x.Kursiv || !x.Unterstrichen || !x.Durchgestrichen || x.Ausrichtung != document.AusrichtungRechts {
		t.Errorf("Text %+v", x)
	}
	if n := o[2].(document.Nummer); n.Farbe != canvas.Beschriftungsfarben[1] || n.Radius != 20 {
		t.Errorf("Nummer %+v", n)
	}
}

// ADR-0021 Nachtrag 20: start_head und end_head beim Pfeil; ohne sie Dreieck am Ende und keine Spitze am Anfang; bei
// rect sind sie unbekannt, unbekannte Arten ein Fehler mit der Liste der Namen.
func TestWendePfeilform(t *testing.T) {
	d := dokument(t, 200, 100)
	es := lies(t, `{"annotations":[
		{"type":"arrow","from":[10,10],"to":[60,40],"start_head":"dot","end_head":"notched"},
		{"type":"arrow","from":[10,60],"to":[60,90],"end_head":"none"},
		{"type":"arrow","from":[100,10],"to":[150,40]}]}`)
	if _, err := Wende(context.Background(), d, es); err != nil {
		t.Fatal(err)
	}
	o := d.Objekte()
	for i, soll := range []struct{ anfang, ende document.Kopfart }{
		{document.KopfPunkt, document.KopfKerbe}, {document.KopfKeine, document.KopfKeine},
		{document.KopfKeine, document.KopfDreieck},
	} {
		if p := o[i].(document.Pfeil); p.KopfAnfang() != soll.anfang || p.Ende != soll.ende {
			t.Errorf("Pfeil %d: %v %v, Soll %v %v", i, p.KopfAnfang(), p.Ende, soll.anfang, soll.ende)
		}
	}
	if _, err := Lies(strings.NewReader(`{"annotations":[{"type":"rect","from":[1,1],"to":[9,9],"end_head":"dot"}]}`)); err == nil {
		t.Error("end_head bei rect angenommen")
	}
	falsch := lies(t, `{"annotations":[{"type":"arrow","from":[10,10],"to":[60,40],"start_head":"spear"}]}`)
	if _, err := Wende(context.Background(), dokument(t, 200, 100), falsch); err == nil ||
		!strings.Contains(err.Error(), `"start_head" must be one of "triangle"`) || !strings.Contains(err.Error(), `or "none"`) {
		t.Errorf("unbekannte Spitze: %v", err)
	}
}

// ADR-0021 Nachtrag 25 Punkt 9: line_style bei arrow und rect; ohne ihn voll; bei text unbekannt, unbekannte Art ein
// Fehler mit der Liste der Namen.
func TestWendeStrichart(t *testing.T) {
	d := dokument(t, 200, 100)
	es := lies(t, `{"annotations":[
		{"type":"arrow","from":[10,10],"to":[60,40],"line_style":"dashed"},
		{"type":"rect","from":[70,10],"to":[120,40],"line_style":"dotted"},
		{"type":"rect","from":[130,10],"to":[180,40]}]}`)
	if _, err := Wende(context.Background(), d, es); err != nil {
		t.Fatal(err)
	}
	o := d.Objekte()
	if p := o[0].(document.Pfeil); p.Strich.Art != document.StrichGestrichelt {
		t.Errorf("Pfeil: %v", p.Strich.Art)
	}
	if k := o[1].(document.Kasten); k.Strich.Art != document.StrichGepunktet {
		t.Errorf("Kasten: %v", k.Strich.Art)
	}
	if k := o[2].(document.Kasten); k.Strich.Art != document.StrichVoll {
		t.Errorf("Kasten ohne line_style: %v", k.Strich.Art)
	}
	if _, err := Lies(strings.NewReader(`{"annotations":[{"type":"text","at":[1,1],"text":"x","line_style":"dashed"}]}`)); err == nil {
		t.Error("line_style bei text angenommen")
	}
	falsch := lies(t, `{"annotations":[{"type":"rect","from":[10,10],"to":[60,40],"line_style":"wavy"}]}`)
	if _, err := Wende(context.Background(), dokument(t, 200, 100), falsch); err == nil ||
		!strings.Contains(err.Error(), `"line_style" must be one of "solid", "dashed" or "dotted"`) {
		t.Errorf("unbekannte Strichart: %v", err)
	}
}

// ADR-0021 Nachtrag 26 Punkt 9: shadow true, false und ausführlich; bei redact unbekannt; ungültige Angaben ein Fehler.
func TestWendeSchatten(t *testing.T) {
	d := dokument(t, 200, 100)
	es := lies(t, `{"annotations":[
		{"type":"rect","from":[10,10],"to":[60,40],"shadow":true},
		{"type":"arrow","from":[70,10],"to":[120,40],"shadow":{"angle":90,"distance":4,"softness":0,"color":"#0000ff80"}},
		{"type":"step","at":[150,50],"shadow":false},
		{"type":"text","at":[10,60],"text":"Hi","shadow":{"distance":3}}]}`)
	if _, err := Wende(context.Background(), d, es); err != nil {
		t.Fatal(err)
	}
	o := d.Objekte()
	if s := o[0].Ebenenwerte().Schatten; s != document.StandardSchatten(1) {
		t.Errorf("true: %+v", s)
	}
	if s := o[1].Ebenenwerte().Schatten; !s.An || s.Winkel != 90 || s.Laenge != 4 || s.Weich != 0 || s.Farbe != (color.NRGBA{B: 255, A: 128}) {
		t.Errorf("ausführlich: %+v", s)
	}
	if o[2].Ebenenwerte().Schatten.An {
		t.Error("false: Schatten an")
	}
	if s := o[3].Ebenenwerte().Schatten; !s.An || s.Laenge != 3 || s.Winkel != 135 {
		t.Errorf("teilweise: %+v", s)
	}
	if _, err := Lies(strings.NewReader(`{"annotations":[{"type":"redact","from":[1,1],"to":[9,9],"shadow":true}]}`)); err == nil {
		t.Error("shadow bei redact angenommen")
	}
	for _, falsch := range []string{`"yes"`, `{"distance":-1}`, `{"softness":1000}`, `{"color":"black"}`, `{"blur":3}`} {
		es := lies(t, `{"annotations":[{"type":"rect","from":[10,10],"to":[60,40],"shadow":`+falsch+`}]}`)
		if _, err := Wende(context.Background(), dokument(t, 200, 100), es); err == nil || !strings.Contains(err.Error(), `"shadow"`) {
			t.Errorf("%s: %v", falsch, err)
		}
	}
}

// TestWendeFehler: Werte außerhalb, falsche Farben, leerer Text, gleiche Punkte — mit Index; der Fehler nennt den
// Eintrag.
func TestWendeFehler(t *testing.T) {
	for _, tc := range []struct{ eintrag, teil string }{
		{`{"type":"arrow","from":[10,10],"to":[201,40]}`, "outside the visible image"},
		{`{"type":"arrow","from":[-1,10],"to":[20,40]}`, "outside the visible image"},
		{`{"type":"arrow","from":[10,10],"to":[10,10]}`, "must differ"},
		{`{"type":"arrow","from":[10,10],"to":[20,20],"color":"red"}`, "#RRGGBB"},
		{`{"type":"arrow","from":[10,10],"to":[20,20],"color":"#FFF"}`, "#RRGGBB"},
		{`{"type":"rect","from":[10,10],"to":[20,20],"fill":"nope"}`, `"none"`},
		{`{"type":"arrow","from":[10,10],"to":[20,20],"width":0}`, "greater than 0"},
		{`{"type":"text","at":[10,10],"text":"  "}`, "must not be empty"},
		{`{"type":"text","at":[10,10],"text":"x","align":"middle"}`, `"align"`},
		{`{"type":"redact","from":[10,10],"to":[10,50]}`, "empty"},
		{`{"type":"arrow","from":[10,10],"to":[20,20],"end_head":"arrow"}`, `"end_head"`},
		{`{"type":"arrow","from":[10,10],"to":[20,20],"start_head":"tip"}`, `"start_head"`},
	} {
		d := dokument(t, 200, 100)
		_, err := Wende(context.Background(), d, lies(t, `{"annotations":[{"type":"step","at":[5,5]},`+tc.eintrag+`]}`))
		var f *Fehler
		if !errors.As(err, &f) || f.Index != 1 || !strings.Contains(err.Error(), tc.teil) {
			t.Errorf("%s: %v", tc.eintrag, err)
		}
	}
}

// TestWendeSchwaerzen: Schwärzen überschreibt die Pixel im Original (Kernprinzip 2), Verpixeln setzt ein Mosaik, ein
// zu schmaler Verpixeln-Bereich wird geschwärzt (ADR-0022 Punkt 2); gezählt als Schwärzungen.
func TestWendeSchwaerzen(t *testing.T) {
	d := dokument(t, 200, 100)
	erg, err := Wende(context.Background(), d, lies(t, `{"annotations":[
		{"type":"redact","from":[10,10],"to":[30,30]},
		{"type":"pixelate","from":[100,10],"to":[160,70]},
		{"type":"pixelate","from":[40,40],"to":[42,90]}]}`))
	if err != nil || erg.Schwaerzungen != 3 || erg.Beschriftungen != 0 {
		t.Fatalf("%+v, %v", erg, err)
	}
	o := d.OriginalBild()
	if c := o.NRGBAAt(20, 20); c != canvas.Schwaerzfarbe {
		t.Errorf("geschwärzt: %v", c)
	}
	// Das Mosaik verschiebt jeden Block um einen Zufallswert, der auch 0 sein kann (ADR-0022 Nachtrag 1): Ein
	// einzelnes Pixel des einfarbigen Testbilds darf deshalb unverändert aussehen (F-604). Geprüft wird der ganze
	// Bereich: nicht geschwärzt, und mindestens ein Block weicht vom Original ab.
	veraendert := false
	for y := 10; y < 70; y++ {
		for x := 100; x < 160; x++ {
			c := o.NRGBAAt(x, y)
			if c == canvas.Schwaerzfarbe {
				t.Fatalf("verpixelt (%d,%d) ist geschwärzt", x, y)
			}
			if c != (color.NRGBA{0x80, 0x80, 0x80, 0x80}) {
				veraendert = true
			}
		}
	}
	if !veraendert {
		t.Error("verpixelt: der ganze Bereich ist unverändert")
	}
	if c := o.NRGBAAt(41, 60); c != canvas.Schwaerzfarbe {
		t.Errorf("schmal verpixelt → geschwärzt: %v", c)
	}
	if c := o.NRGBAAt(5, 5); c != (color.NRGBA{0x80, 0x80, 0x80, 0x80}) {
		t.Errorf("außerhalb verändert: %v", c)
	}
	if !d.Geschwaerzt() {
		t.Error("Dokument nicht als geschwärzt gemeldet")
	}
}

// TestWendeSichtbaresBild: Koordinaten gelten im sichtbaren Bild — nach einem Zuschnitt landen sie verschoben im
// Original; Schwärzen trifft das Original an derselben Stelle (ADR-0030 Nachtrag 10 Punkt 2).
func TestWendeSichtbaresBild(t *testing.T) {
	d := dokument(t, 200, 100)
	if err := d.SetzeZuschnitt(image.Rect(50, 20, 150, 80)); err != nil {
		t.Fatal(err)
	}
	if _, err := Wende(context.Background(), d, lies(t, `{"annotations":[{"type":"step","at":[100,60]}]}`)); err != nil {
		t.Fatal(err)
	}
	if n := d.Objekte()[0].(document.Nummer); n.Mitte != document.Pt(150, 80) {
		t.Errorf("Nummer bei %v, erwartet (150, 80)", n.Mitte)
	}
	if _, err := Wende(context.Background(), d, lies(t, `{"annotations":[{"type":"step","at":[101,10]}]}`)); err == nil {
		t.Error("Punkt rechts neben dem sichtbaren Bild angenommen")
	}
	if _, err := Wende(context.Background(), d, lies(t, `{"annotations":[{"type":"redact","from":[0,0],"to":[10,10]}]}`)); err != nil {
		t.Fatal(err)
	}
	o := d.OriginalBild()
	if o.NRGBAAt(55, 25) != canvas.Schwaerzfarbe || o.NRGBAAt(45, 25) == canvas.Schwaerzfarbe {
		t.Errorf("Schwärzung am falschen Ort: in %v, davor %v", o.NRGBAAt(55, 25), o.NRGBAAt(45, 25))
	}
}

// TestWendeGrenze: Mehr Objekte, als ein Dokument nimmt, ergeben einen Fehler am ersten zu vielen Eintrag.
func TestWendeGrenze(t *testing.T) {
	d := dokument(t, 200, 100)
	for i := 0; i < document.MaxObjekte; i++ {
		if _, err := d.Hinzufuegen(document.Nummer{Mitte: document.Pt(10, 10), Wert: 1, Radius: 5, Farbe: canvas.Weiss}); err != nil {
			t.Fatal(err)
		}
	}
	_, err := Wende(context.Background(), d, lies(t, `{"annotations":[{"type":"arrow","from":[1,1],"to":[9,9]}]}`))
	if err == nil || !strings.Contains(err.Error(), "annotation 0") {
		t.Errorf("über der Objektgrenze: %v", err)
	}
}

// ADR-0047 Punkt 7: Typ ellipse mit den Feldern des Rechtecks, Form Ellipse, Voreinstellungen wie das Werkzeug.
func TestWendeEllipse(t *testing.T) {
	d := dokument(t, 200, 100)
	es := lies(t, `{"annotations":[
		{"type":"ellipse","from":[10,10],"to":[60,40]},
		{"type":"ellipse","from":[70,10],"to":[120,40],"fill":"#00ff0080","line_style":"dashed","width":4,"shadow":true},
		{"type":"rect","from":[130,10],"to":[180,40]}]}`)
	if _, err := Wende(context.Background(), d, es); err != nil {
		t.Fatal(err)
	}
	o := d.Objekte()
	k0, k1, k2 := o[0].(document.Kasten), o[1].(document.Kasten), o[2].(document.Kasten)
	if k0.Form != document.FormEllipse || k0.Fuellung.A != 0 || k0.Bereich != document.R(document.Pt(10, 10), document.Pt(60, 40)) {
		t.Errorf("Ellipse: %+v", k0)
	}
	if k1.Form != document.FormEllipse || k1.Fuellung.A != 0x80 || k1.Strich.Art != document.StrichGestrichelt ||
		k1.Strich.Breite != 4 || !k1.Ebene.Schatten.An {
		t.Errorf("Ellipse mit Angaben: %+v", k1)
	}
	if k2.Form != document.FormRechteck {
		t.Errorf("Rechteck: %+v", k2)
	}
	if _, err := Lies(strings.NewReader(`{"annotations":[{"type":"ellipse","from":[1,1],"to":[5,5],"start_head":"dot"}]}`)); err == nil {
		t.Error("start_head bei ellipse angenommen")
	}
}

// ADR-0048 Punkt 8: Typ highlight ist ein Textmarker (multiplizierender Pfeil ohne Köpfe) mit Gelb und Breite 20 als
// Voreinstellung; Köpfe und Füllung gibt es dort nicht.
func TestWendeTextmarker(t *testing.T) {
	d := dokument(t, 200, 100)
	es := lies(t, `{"annotations":[
		{"type":"highlight","from":[10,20],"to":[150,20]},
		{"type":"highlight","from":[10,60],"to":[150,70],"color":"#ff8ad8","width":12,"line_style":"dotted"}]}`)
	if _, err := Wende(context.Background(), d, es); err != nil {
		t.Fatal(err)
	}
	o := d.Objekte()
	m0, m1 := o[0].(document.Pfeil), o[1].(document.Pfeil)
	if !m0.Multipliziert() || m0.Ende != document.KopfKeine || m0.KopfAnfang() != document.KopfKeine ||
		m0.Strich.Farbe != canvas.TextmarkerGelb || m0.Strich.Breite != canvas.TextmarkerBreite {
		t.Errorf("Textmarker: %+v", m0)
	}
	if !m1.Multipliziert() || m1.Strich.Farbe != (color.NRGBA{R: 0xff, G: 0x8a, B: 0xd8, A: 0xff}) || m1.Strich.Breite != 12 ||
		m1.Strich.Art != document.StrichGepunktet {
		t.Errorf("Textmarker mit Angaben: %+v", m1)
	}
	for _, j := range []string{
		`{"annotations":[{"type":"highlight","from":[1,1],"to":[5,5],"end_head":"dot"}]}`,
		`{"annotations":[{"type":"highlight","from":[1,1],"to":[5,5],"fill":"#ffffff"}]}`,
		`{"annotations":[{"type":"highlight","from":[1,1]}]}`,
	} {
		if _, err := Lies(strings.NewReader(j)); err == nil {
			t.Errorf("angenommen: %s", j)
		}
	}
}

// F-1029 (Kritik #329 K-1): highlight mit shadow bleibt ein Textmarker (MitEbene ersetzte die ganze Ebene).
func TestHighlightMitSchattenBleibtTextmarker(t *testing.T) {
	d := dokument(t, 200, 100)
	es := lies(t, `{"annotations":[{"type":"highlight","from":[10,20],"to":[150,20],"shadow":true}]}`)
	if _, err := Wende(context.Background(), d, es); err != nil {
		t.Fatal(err)
	}
	m := d.Objekte()[0].(document.Pfeil)
	if !m.Ebenenwerte().Schatten.An || !m.Multipliziert() {
		t.Errorf("Schatten %v, Mischung %v", m.Ebenenwerte().Schatten.An, m.Ebenenwerte().Mischung)
	}
}

// ADR-0049 Punkt 7: freehand mit points (übernommen, nicht geglättet); highlight mit points ist ein freier Textmarker;
// highlight braucht from und to oder points, nicht beides.
func TestWendeFreihand(t *testing.T) {
	d := dokument(t, 200, 100)
	es := lies(t, `{"annotations":[
		{"type":"freehand","points":[[10,10],[30,40],[60,20]],"width":5,"line_style":"dashed"},
		{"type":"highlight","points":[[10,60],[80,62],[150,58]]}]}`)
	if _, err := Wende(context.Background(), d, es); err != nil {
		t.Fatal(err)
	}
	o := d.Objekte()
	f0, f1 := o[0].(document.Freihand), o[1].(document.Freihand)
	if len(f0.Punkte()) != 3 || f0.Strich.Breite != 5 || f0.Strich.Art != document.StrichGestrichelt || f0.Multipliziert() {
		t.Errorf("freehand: %+v %v", f0, f0.Punkte())
	}
	if !f1.Multipliziert() || f1.Strich.Farbe != canvas.TextmarkerGelb || f1.Strich.Breite != canvas.TextmarkerBreite {
		t.Errorf("highlight mit points: %+v", f1)
	}
	for _, j := range []string{
		`{"annotations":[{"type":"freehand","points":[[1,1]]}]}`,
		`{"annotations":[{"type":"freehand","from":[1,1],"to":[5,5]}]}`,
		`{"annotations":[{"type":"highlight","from":[1,1],"to":[5,5],"points":[[1,1],[2,2]]}]}`,
		`{"annotations":[{"type":"highlight","to":[5,5]}]}`,
		`{"annotations":[{"type":"freehand","points":[[1,1],[500,5]]}]}`,
		`{"annotations":[{"type":"freehand","points":[[5,5],[5,5]],"line_style":"dashed"}]}`, // ohne Länge (F-1040)
		`{"annotations":[{"type":"freehand","points":[[5,5],[5.004,5]]}]}`,                   // unter 1/100 px: ohne Länge (F-1045)
	} {
		es, err := Lies(strings.NewReader(j))
		if err == nil {
			_, err = Wende(context.Background(), dokument(t, 200, 100), es)
		}
		if err == nil {
			t.Errorf("angenommen: %s", j)
		}
	}
}
