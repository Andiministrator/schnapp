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
	if c := o.NRGBAAt(130, 40); c == canvas.Schwaerzfarbe || c == (color.NRGBA{0x80, 0x80, 0x80, 0x80}) {
		t.Errorf("verpixelt: %v (weder Mosaik noch unverändert erwartet)", c)
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
