// SPDX-License-Identifier: GPL-3.0-or-later

package annotate

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// TestWendeSpitze: `tip` macht den Text zur Sprechblase (ADR-0021 Nachtrag 13 Punkt 8), Koordinaten im sichtbaren Bild
// wie `at`; ohne `tip` bleibt es ein einfacher Text.
func TestWendeSpitze(t *testing.T) {
	d := dokument(t, 200, 100)
	es := lies(t, `{"annotations":[
		{"type":"text","at":[10,10],"text":"Hallo","tip":[120,80]},
		{"type":"text","at":[10,50],"text":"Ohne"}]}`)
	if _, err := Wende(context.Background(), d, es); err != nil {
		t.Fatal(err)
	}
	o := d.Objekte()
	a, b := o[0].(document.Text), o[1].(document.Text)
	if a.Spitze == nil || *a.Spitze != (document.Punkt{X: 120, Y: 80}) || b.Spitze != nil {
		t.Errorf("Spitzen: %v, %v", a.Spitze, b.Spitze)
	}
}

// TestSpitzeNurBeiText: `tip` gehört nur zum Typ text; außerhalb des Bilds, ein Punkt mit einer Zahl und null sind
// Fehler mit dem Index des Eintrags.
func TestSpitzeFehler(t *testing.T) {
	_, err := Lies(strings.NewReader(`{"annotations":[{"type":"step","at":[1,1],"tip":[2,2]}]}`))
	var f *Fehler
	if !errors.As(err, &f) || f.Index != 0 || !strings.Contains(err.Error(), `"tip" does not apply`) {
		t.Errorf("tip an step: %v", err)
	}
	for _, tc := range []struct{ eintrag, teil string }{
		{`{"type":"text","at":[10,10],"text":"x","tip":[201,10]}`, "outside the visible image"},
		{`{"type":"text","at":[10,10],"text":"x","tip":[10,-0.5]}`, "outside the visible image"},
	} {
		d := dokument(t, 200, 100)
		_, err := Wende(context.Background(), d, lies(t, `{"annotations":[{"type":"step","at":[5,5]},`+tc.eintrag+`]}`))
		if !errors.As(err, &f) || f.Index != 1 || !strings.Contains(err.Error(), tc.teil) {
			t.Errorf("%s: %v", tc.eintrag, err)
		}
	}
	for _, spec := range []string{
		`{"annotations":[{"type":"text","at":[1,1],"text":"x","tip":[5]}]}`,
		`{"annotations":[{"type":"text","at":[1,1],"text":"x","tip":null}]}`,
		`{"annotations":[{"type":"text","at":[1,1],"text":"x","tip":[5,5],"tip":[6,6]}]}`,
	} {
		if _, err := Lies(strings.NewReader(spec)); err == nil {
			t.Errorf("%s: angenommen", spec)
		}
	}
}
