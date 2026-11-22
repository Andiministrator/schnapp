// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"image"
	"image/png"
	"io"
	"reflect"
	"testing"
	"unsafe"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-331: Scheitert die nebenläufige Kodierung des Originals (das Bild gelingt), meldet SchreibeArbeitsdatei den Fehler
// und schreibt nichts. Ein Dokument mit leerem Original gibt es über die Schnittstelle nicht (document.Neu lehnt es ab);
// der Test setzt das private Feld über reflect und unsafe, nur hier, um die Kodierung scheitern zu lassen.
func TestSchreibeArbeitsdateiOriginalFehler(t *testing.T) {
	d, err := document.Neu(rauschbild(8, 8, 1))
	if err != nil {
		t.Fatal(err)
	}
	f := reflect.ValueOf(d).Elem().FieldByName("original")
	leer := image.NewNRGBA(image.Rect(0, 0, 0, 0))
	*(**image.NRGBA)(unsafe.Pointer(f.UnsafeAddr())) = leer
	want := png.Encode(io.Discard, leer)
	if want == nil {
		t.Fatal("Vorbedingung: leeres Bild lässt sich kodieren")
	}
	var z zaehlSchreiber
	err = SchreibeArbeitsdatei(&z, d, rauschbild(8, 8, 2), Metadaten{})
	if err == nil {
		t.Fatal("Fehler der Original-Kodierung verschluckt: keine Fehlermeldung")
	}
	// Hülle mit Paketpräfix (ADR-0019 Nachtrag 2, F-362): Der Text der Kodierung folgt unverändert dahinter.
	if err.Error() != "fileformat: "+want.Error() {
		t.Errorf("Fehler %q, erwartet „fileformat: “ vor dem der Original-Kodierung %q", err, want)
	}
	if z.aufrufe != 0 {
		t.Errorf("trotz Fehler %d Schreibaufrufe", z.aufrufe)
	}
}
