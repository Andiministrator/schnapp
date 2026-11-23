// SPDX-License-Identifier: GPL-3.0-or-later

package annotate

// Tests zu den Befunden der Kritik an Karte #202 (B-3, B-4, B-5, B-6, B-7, Anmerkung).

import (
	"context"
	"errors"
	"image/color"
	"strconv"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// TestLiesSchluesselExakt (B-6): Schlüssel gelten nur in genau ihrer Schreibung, doppelte Schlüssel sind ein Fehler —
// in der Hülle und in den Einträgen (dort mit Index).
func TestLiesSchluesselExakt(t *testing.T) {
	for _, tc := range []struct {
		spec, teil string
		index      int
	}{
		{`{"ANNOTATIONS":[{"type":"step","at":[1,1]}]}`, `unknown key "ANNOTATIONS"`, -1},
		{`{"annotations":[{"type":"step","at":[1,1]}],"annotations":[]}`, `duplicate key "annotations"`, -1},
		{`{"annotations":[{"type":"step","at":[1,1],"at":[2,2]}]}`, `duplicate key "at"`, 0},
		{`{"annotations":[{"type":"step","at":[1,1]},{"type":"step","type":"arrow","at":[1,1]}]}`, `duplicate key "type"`, 1},
		{`{"annotations":[{"TYPE":"step","at":[1,1]}]}`, `missing key "type"`, 0},
		{`{"annotations":[{"type":"step","AT":[1,1]}]}`, `"AT" does not apply`, 0},
		{`{"annotations":{"type":"step"}}`, "must be an array", -1},
	} {
		_, err := Lies(strings.NewReader(tc.spec))
		var f *Fehler
		if !errors.As(err, &f) || f.Index != tc.index || !strings.Contains(err.Error(), tc.teil) {
			t.Errorf("%s: %v (Index erwartet %d, Teil %q)", tc.spec, err, tc.index, tc.teil)
		}
	}
}

// TestKeineDoppelten: auf jeder Ebene, auch in Feldern; gleiche Schlüssel in verschiedenen Objekten sind erlaubt.
func TestKeineDoppelten(t *testing.T) {
	for _, s := range []string{`{"a":1,"a":2}`, `{"x":{"b":[1,{"c":1,"c":1}]}}`, `[{"a":1},{"b":{"d":1,"d":2}}]`} {
		if err := KeineDoppelten([]byte(s)); err == nil || !strings.Contains(err.Error(), "duplicate key") {
			t.Errorf("%s: %v", s, err)
		}
	}
	for _, s := range []string{`{"a":1,"b":{"a":1}}`, `[{"a":1},{"a":2}]`, `"a"`, `{}`, `{"a":[]}`} {
		if err := KeineDoppelten([]byte(s)); err != nil {
			t.Errorf("%s: %v", s, err)
		}
	}
}

// TestLiesLeer (B-5): Eine leere Liste ist ein Fehler, keine Aufnahme ohne Änderung.
func TestLiesLeer(t *testing.T) {
	_, err := Lies(strings.NewReader(`{"annotations":[]}`))
	var f *Fehler
	if !errors.As(err, &f) || f.Index != -1 || !strings.Contains(err.Error(), "empty") {
		t.Errorf("leere Liste: %v", err)
	}
}

// TestLiesLeseFehler (B-7): Ein Lesefehler ist kein *Fehler der Liste, sondern ein *LeseFehler mit der Ursache.
func TestLiesLeseFehler(t *testing.T) {
	ursache := errors.New("E/A")
	_, err := Lies(iotest.ErrReader(ursache))
	var f *Fehler
	var l *LeseFehler
	if errors.As(err, &f) || !errors.As(err, &l) || !errors.Is(err, ursache) {
		t.Errorf("Lesefehler: %T %v", err, err)
	}
}

// TestLiesMaxEintraege (M7): MaxEintraege Einträge gehen, einer mehr nicht.
func TestLiesMaxEintraege(t *testing.T) {
	liste := func(n int) string {
		var b strings.Builder
		b.WriteString(`{"annotations":[`)
		for i := 0; i < n; i++ {
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString(`{"type":"step","at":[1,1]}`)
		}
		b.WriteString("]}")
		return b.String()
	}
	if es, err := Lies(strings.NewReader(liste(MaxEintraege))); err != nil || len(es) != MaxEintraege {
		t.Errorf("genau an der Grenze: %d, %v", len(es), err)
	}
	if _, err := Lies(strings.NewReader(liste(MaxEintraege + 1))); err == nil || !strings.Contains(err.Error(), "more than") {
		t.Errorf("über der Grenze: %v", err)
	}
}

// TestWendeNoneNurFuerFuellung (M17): "none" gilt nur für fill, nie für color.
func TestWendeNoneNurFuerFuellung(t *testing.T) {
	for _, e := range []string{
		`{"type":"arrow","from":[1,1],"to":[20,20],"color":"none"}`,
		`{"type":"rect","from":[1,1],"to":[20,20],"color":"none"}`,
		`{"type":"text","at":[1,1],"text":"x","color":"none"}`,
		`{"type":"step","at":[10,10],"color":"none"}`,
	} {
		d := dokument(t, 100, 50)
		if _, err := Wende(context.Background(), d, lies(t, `{"annotations":[`+e+`]}`)); err == nil || !strings.Contains(err.Error(), "#RRGGBB") {
			t.Errorf("%s: %v", e, err)
		}
	}
	d := dokument(t, 100, 50)
	if _, err := Wende(context.Background(), d, lies(t, `{"annotations":[{"type":"rect","from":[1,1],"to":[20,20],"fill":"none"}]}`)); err != nil {
		t.Errorf("fill none: %v", err)
	}
}

// TestWendeFehlerOhneBildgroesse (Anmerkung): Der Fehler zu einem Punkt außerhalb nennt die Maße des Bilds nicht.
func TestWendeFehlerOhneBildgroesse(t *testing.T) {
	d := dokument(t, 217, 131)
	_, err := Wende(context.Background(), d, lies(t, `{"annotations":[{"type":"step","at":[300,5]}]}`))
	if err == nil || !strings.Contains(err.Error(), "outside the visible image") || strings.Contains(err.Error(), "217") ||
		strings.Contains(err.Error(), "131") {
		t.Errorf("Fehlertext: %v", err)
	}
}

// TestWendeSchwaerzBudget (B-3): 2000 ganzflächige Verpixelungen werden vorab abgelehnt, ohne ein Pixel anzufassen;
// eine Liste knapp unter dem Budget geht.
func TestWendeSchwaerzBudget(t *testing.T) {
	d := dokument(t, 4000, 3000)
	var l strings.Builder
	l.WriteString(`{"annotations":[{"type":"step","at":[1,1]}`)
	for i := 0; i < 2000; i++ {
		l.WriteString(`,{"type":"pixelate","from":[0,0],"to":[4000,3000]}`)
	}
	l.WriteString("]}")
	start := time.Now()
	_, err := Wende(context.Background(), d, lies(t, l.String()))
	var f *Fehler
	if !errors.As(err, &f) || !strings.Contains(err.Error(), "megapixels") || time.Since(start) > 5*time.Second {
		t.Fatalf("2000 ganzflächige Verpixelungen: %v nach %v", err, time.Since(start))
	}
	// Der Index ist der Eintrag, mit dem die Summe die Grenze überschreitet: VerlaufPlatz / (4 · 12 MPx) ganze passen.
	passen := int(d.VerlaufPlatz() / (4 * 4000 * 3000))
	if f.Index != passen+1 {
		t.Errorf("Index %d, erwartet %d", f.Index, passen+1)
	}
	if d.Geschwaerzt() || len(d.Objekte()) != 0 {
		t.Error("vor der Ablehnung schon angewendet")
	}
	klein := dokument(t, 400, 300)
	l.Reset()
	l.WriteString(`{"annotations":[`)
	for i := 0; i < 10; i++ {
		if i > 0 {
			l.WriteString(",")
		}
		l.WriteString(`{"type":"redact","from":[0,0],"to":[400,300]}`)
	}
	l.WriteString("]}")
	if erg, err := Wende(context.Background(), klein, lies(t, l.String())); err != nil || erg.Schwaerzungen != 10 {
		t.Errorf("unter dem Budget: %+v, %v", erg, err)
	}
}

// TestWendeAbbruch (B-3): Ist ctx abgebrochen, legt Wende nichts mehr an und meldet ctx.Err(), keinen *Fehler.
func TestWendeAbbruch(t *testing.T) {
	d := dokument(t, 100, 50)
	ctx, abbrechen := context.WithCancel(context.Background())
	abbrechen()
	_, err := Wende(ctx, d, lies(t, `{"annotations":[{"type":"redact","from":[0,0],"to":[10,10]}]}`))
	var f *Fehler
	if !errors.Is(err, context.Canceled) || errors.As(err, &f) || d.Geschwaerzt() {
		t.Errorf("Abbruch: %v, geschwärzt %v", err, d.Geschwaerzt())
	}
}

// TestWendeStreifen (B-4): Mit gestauchten Spalten 80..120 deckt eine Schwärzung über die Naht den Streifen mit; eine
// bis an die Naht nicht, eine dahinter landet verschoben im Original.
func TestWendeStreifen(t *testing.T) {
	grau := color.NRGBA{0x80, 0x80, 0x80, 0x80}
	for _, tc := range []struct {
		von, bis     int
		schwarz, nie []int // Spalten im Original (Zeile 15)
	}{
		{70, 90, []int{70, 85, 100, 119, 125, 129}, []int{69, 130}},
		{60, 80, []int{60, 79}, []int{59, 80, 100, 119, 120}},
		{80, 90, []int{120, 129}, []int{79, 100, 119, 130}},
	} {
		d := dokument(t, 200, 100)
		if err := d.SetzeStreifen([]document.Streifen{{Von: 80, Bis: 120}}); err != nil {
			t.Fatal(err)
		}
		e := `{"annotations":[{"type":"redact","from":[` + strconv.Itoa(tc.von) + `,10],"to":[` + strconv.Itoa(tc.bis) + `,20]}]}`
		if _, err := Wende(context.Background(), d, lies(t, e)); err != nil {
			t.Fatal(err)
		}
		o := d.OriginalBild()
		for _, x := range tc.schwarz {
			if c := o.NRGBAAt(x, 15); c != canvas.Schwaerzfarbe {
				t.Errorf("%d..%d: Spalte %d nicht geschwärzt (%v)", tc.von, tc.bis, x, c)
			}
		}
		for _, x := range tc.nie {
			if c := o.NRGBAAt(x, 15); c != grau {
				t.Errorf("%d..%d: Spalte %d verändert (%v)", tc.von, tc.bis, x, c)
			}
		}
	}
}
