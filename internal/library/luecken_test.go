// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/Andiministrator/schnapp/internal/fileformat"
)

// neuereFassung baut aus den Bytes einer Arbeitsdatei eine, deren SVG-Block die Formatversion 2 trägt (neuer als
// diese Fassung): PNG-Blöcke lesen, den Block snAP entpacken, die Version ersetzen, packen, mit neuer Prüfsumme
// schreiben.
func neuereFassung(t *testing.T, daten []byte) []byte {
	t.Helper()
	var aus bytes.Buffer
	aus.Write(daten[:8]) // PNG-Signatur
	ersetzt := false
	for p := 8; p < len(daten); {
		n := int(binary.BigEndian.Uint32(daten[p:]))
		typ := string(daten[p+4 : p+8])
		inhalt := daten[p+8 : p+8+n]
		if typ == "snAP" {
			zr, err := zlib.NewReader(bytes.NewReader(inhalt))
			if err != nil {
				t.Fatal(err)
			}
			svg, err := io.ReadAll(zr)
			if err != nil {
				t.Fatal(err)
			}
			neu := bytes.Replace(svg, []byte(`s:format="1"`), []byte(`s:format="2"`), 1)
			if bytes.Equal(neu, svg) {
				t.Fatalf("Formatversion nicht im SVG gefunden: %s", svg)
			}
			var z bytes.Buffer
			zw := zlib.NewWriter(&z)
			zw.Write(neu)
			zw.Close()
			inhalt, ersetzt = z.Bytes(), true
		}
		var kopf [8]byte
		binary.BigEndian.PutUint32(kopf[:], uint32(len(inhalt)))
		copy(kopf[4:], typ)
		aus.Write(kopf[:])
		aus.Write(inhalt)
		var crc [4]byte
		binary.BigEndian.PutUint32(crc[:], crc32.ChecksumIEEE(append([]byte(typ), inhalt...)))
		aus.Write(crc[:])
		p += 12 + n
	}
	if !ersetzt {
		t.Fatal("kein snAP-Block")
	}
	return aus.Bytes()
}

// ablageDatei liefert den vollen Pfad einer Aufnahme der Ablage.
func ablageDatei(b *Bibliothek, rel string) string {
	return filepath.Join(b.Orte.Ablage, filepath.FromSlash(rel))
}

func lies(t *testing.T, pfad string) []byte {
	t.Helper()
	d, err := os.ReadFile(pfad)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// F-111: Eine Datei aus einer neueren Fassung (Formatversion 2) wird nicht überschrieben: ErrFormatNeuer, nicht
// ErrFremd (der Editor weicht in beiden Fällen auf eine neue Aufnahme aus, die Fehler sind aber verschieden), und die
// Datei bleibt Byte für Byte erhalten. Gegenstück: dieselbe Datei in Version 1 lässt sich speichern.
func TestSpeichernFormatNeuer(t *testing.T) {
	b, _ := neueBibliothek(t)
	d := neuesDokument(t, 40, 30)
	e, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{})
	if err != nil {
		t.Fatal(err)
	}
	pfad := ablageDatei(b, e.Pfad)
	if _, _, err := b.Speichern(e.Pfad, standIn(t, b, e.Pfad), d, d.Ansichtsbild()); err != nil {
		t.Fatalf("Gegenstück (Version 1): %v", err)
	}
	if err := os.WriteFile(pfad, neuereFassung(t, lies(t, pfad)), 0o600); err != nil {
		t.Fatal(err)
	}
	vorher, stand := lies(t, pfad), standIn(t, b, e.Pfad)
	_, _, err = b.Speichern(e.Pfad, stand, d, d.Ansichtsbild())
	if !errors.Is(err, fileformat.ErrFormatNeuer) || errors.Is(err, ErrFremd) {
		t.Fatalf("Fehler %v, Soll fileformat.ErrFormatNeuer (nicht ErrFremd)", err)
	}
	if !bytes.Equal(lies(t, pfad), vorher) {
		t.Error("Datei aus neuerer Fassung verändert")
	}
	if idx, _ := b.Laden(); len(idx.Eintraege) != 1 || !idx.Eintraege[0].Bearbeitet || idx.Eintraege[0].Pfad != e.Pfad {
		t.Errorf("Index nach abgelehntem Speichern: %+v (Soll unverändert: der eine Eintrag, vom Gegenstück bearbeitet)", idx.Eintraege)
	}
	if standIn(t, b, e.Pfad) != stand {
		t.Error("Stand der Datei nach abgelehntem Speichern geändert")
	}
}

// F-111: Nur eine gewöhnliche Datei wird beschrieben. Ein Verweis in der Ablage (auch auf eine Aufnahme der Ablage)
// und ein Ordner mit dem Namen einer Aufnahme ergeben ErrPfad; das Ziel des Verweises bleibt unverändert.
func TestSpeichernNurGewoehnlicheDatei(t *testing.T) {
	b, heim := neueBibliothek(t)
	d := neuesDokument(t, 40, 30)
	e, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{})
	if err != nil {
		t.Fatal(err)
	}
	monat := filepath.Dir(ablageDatei(b, e.Pfad))
	// Verweis auf eine fremde Datei außerhalb der Ablage.
	aussen := filepath.Join(heim, "aussen.png")
	if err := os.WriteFile(aussen, lies(t, ablageDatei(b, e.Pfad)), 0o600); err != nil {
		t.Fatal(err)
	}
	// Verweis auf eine andere Aufnahme der Ablage, ein Ordner als „Aufnahme“.
	for name, einrichten := range map[string]func(pfad string) error{
		"nach-aussen.png": func(p string) error { return os.Symlink(aussen, p) },
		"nach-innen.png":  func(p string) error { return os.Symlink(filepath.Base(ablageDatei(b, e.Pfad)), p) },
		"ordner.png":      func(p string) error { return os.Mkdir(p, 0o700) },
	} {
		if err := einrichten(filepath.Join(monat, name)); err != nil {
			t.Fatal(err)
		}
		rel := filepath.ToSlash(filepath.Join(filepath.Base(monat), name))
		_, neuStand, err := b.Speichern(rel, Stand{}, d, d.Ansichtsbild())
		if !errors.Is(err, ErrPfad) || neuStand != (Stand{}) {
			t.Errorf("%s: Fehler %v, Stand %v, Soll ErrPfad ohne Stand", name, err, neuStand)
		}
	}
	if !bytes.Equal(lies(t, aussen), lies(t, ablageDatei(b, e.Pfad))) {
		t.Error("Ziel des Verweises außerhalb verändert")
	}
	if idx, _ := b.Laden(); len(idx.Eintraege) != 1 || idx.Eintraege[0].Bearbeitet {
		t.Errorf("Index nach abgelehntem Speichern: %+v", idx.Eintraege)
	}
}

// F-111: Scheitert das Umbenennen im letzten Schritt, bleibt die Aufnahme unverändert, es bleibt keine Restdatei, der
// Index ändert sich nicht, und der Fehler kommt zurück (ohne neuen Stand). Das Speichern danach gelingt.
func TestSpeichernScheiterndesUmbenennen(t *testing.T) {
	b, _ := neueBibliothek(t)
	d := neuesDokument(t, 40, 30)
	e, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{Schlagworte: []string{"x"}})
	if err != nil {
		t.Fatal(err)
	}
	pfad := ablageDatei(b, e.Pfad)
	vorher, namenVorher := lies(t, pfad), namen(t, filepath.Dir(pfad))
	kasten(t, d, 1)
	stand := standIn(t, b, e.Pfad)
	boese := errors.New("Attrappe: Umbenennen scheitert")
	alt := umbenenne
	umbenenne = func(string, string) error { return boese }
	_, neuStand, err := b.Speichern(e.Pfad, stand, d, d.Ansichtsbild())
	umbenenne = alt
	if !errors.Is(err, boese) || neuStand != (Stand{}) {
		t.Fatalf("Fehler %v, Stand %v, Soll die Attrappe ohne Stand", err, neuStand)
	}
	if !bytes.Equal(lies(t, pfad), vorher) {
		t.Error("Aufnahme nach gescheitertem Umbenennen verändert")
	}
	if n := namen(t, filepath.Dir(pfad)); len(n) != len(namenVorher) || n[0] != namenVorher[0] {
		t.Errorf("Ordner %v, vorher %v (Restdatei?)", n, namenVorher)
	}
	if idx, _ := b.Laden(); len(idx.Eintraege) != 1 || idx.Eintraege[0].Bearbeitet {
		t.Errorf("Index nach gescheitertem Speichern: %+v (Soll unverändert)", idx.Eintraege)
	}
	if standIn(t, b, e.Pfad) != stand {
		t.Error("Stand der Datei hat sich geändert")
	}
	// Gültiges Gegenstück: derselbe Aufruf ohne Fehler schreibt.
	if _, _, err := b.Speichern(e.Pfad, stand, d, d.Ansichtsbild()); err != nil {
		t.Fatalf("Gegenstück: %v", err)
	}
	if g, err := fileformat.Oeffne(pfad); err != nil || len(g.Dokument.Objekte()) != 1 {
		t.Errorf("Datei nach dem Gegenstück: %v", err)
	}
}

// F-111: Scheitert nur der Index, ist die Datei trotzdem geschrieben: Eintrag und neuer Stand kommen zurück, der Fehler
// ist ErrIndex (nicht ErrVorschau). Scheitern Index und Vorschau zugleich, meldet er beide.
func TestSpeichernIndexFehler(t *testing.T) {
	b, _ := neueBibliothek(t)
	d := neuesDokument(t, 40, 30)
	e, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{Schlagworte: []string{"Rechnung"}})
	if err != nil {
		t.Fatal(err)
	}
	pfad := ablageDatei(b, e.Pfad)
	kasten(t, d, 1)
	indexBoese := errors.New("Attrappe: Index nicht schreibbar")
	ersetzeUmbenenne(t, func(von, nach string) error {
		if nach == b.Orte.Index {
			return indexBoese
		}
		return os.Rename(von, nach)
	})
	neu, neuStand, err := b.Speichern(e.Pfad, standIn(t, b, e.Pfad), d, d.Ansichtsbild())
	if !errors.Is(err, ErrIndex) || errors.Is(err, ErrVorschau) {
		t.Fatalf("Fehler %v, Soll ErrIndex ohne ErrVorschau", err)
	}
	if neuStand != standIn(t, b, e.Pfad) || neuStand == (Stand{}) {
		t.Errorf("Stand %v, Soll der Stand der geschriebenen Datei", neuStand)
	}
	if !neu.Bearbeitet || neu.Pfad != e.Pfad || len(neu.Schlagworte) != 1 {
		t.Errorf("Eintrag %+v", neu)
	}
	if g, err := fileformat.Oeffne(pfad); err != nil || len(g.Dokument.Objekte()) != 1 || !g.Meta.Bearbeitet {
		t.Errorf("Datei nicht geschrieben: %v", err)
	}
	if idx, _ := b.Laden(); len(idx.Eintraege) != 1 || idx.Eintraege[0].Bearbeitet {
		t.Errorf("Index %+v: Soll der alte Stand (Schreiben scheiterte)", idx.Eintraege)
	}
	// Der Stand taugt für das nächste Speichern (kein Konflikt mit sich selbst).
	kasten(t, d, 6)
	if _, _, err := b.Speichern(e.Pfad, neuStand, d, d.Ansichtsbild()); !errors.Is(err, ErrIndex) || errors.Is(err, ErrGeaendert) {
		t.Errorf("zweites Speichern: %v, Soll wieder nur ErrIndex", err)
	}
	// Index und Vorschau zugleich.
	ersetzeVorschauNaht(t, func(io.Writer, image.Image) error { return errKodieren }, nil)
	_, _, err = b.Speichern(e.Pfad, standIn(t, b, e.Pfad), d, d.Ansichtsbild())
	if !errors.Is(err, ErrIndex) || !errors.Is(err, ErrVorschau) || errors.Is(err, ErrVorschauAlt) {
		t.Errorf("beide: %v, Soll ErrIndex und ErrVorschau ohne ErrVorschauAlt", err)
	}
}

// F-111: Scheitert nur die Vorschau, kommen Eintrag und Stand der geschriebenen Datei zurück, und der Index ist
// fortgeschrieben (Bearbeitet).
func TestSpeichernVorschauFehlerSchreibtTrotzdem(t *testing.T) {
	b, _ := neueBibliothek(t)
	d := neuesDokument(t, 40, 30)
	e, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{})
	if err != nil {
		t.Fatal(err)
	}
	kasten(t, d, 1)
	ersetzeVorschauNaht(t, func(io.Writer, image.Image) error { return errKodieren }, nil)
	neu, neuStand, err := b.Speichern(e.Pfad, standIn(t, b, e.Pfad), d, d.Ansichtsbild())
	if !errors.Is(err, ErrVorschau) || errors.Is(err, ErrIndex) {
		t.Fatalf("Fehler %v, Soll ErrVorschau ohne ErrIndex", err)
	}
	if neuStand != standIn(t, b, e.Pfad) || neuStand == (Stand{}) || !neu.Bearbeitet {
		t.Errorf("Stand %v, Eintrag %+v", neuStand, neu)
	}
	if idx, _ := b.Laden(); len(idx.Eintraege) != 1 || !idx.Eintraege[0].Bearbeitet {
		t.Errorf("Index %+v: Soll bearbeitet", idx.Eintraege)
	}
}

// F-127: Ablegen erneuert die Vorschau mit Entfernen des alten Bilds (erneuereVorschau), nicht nur mit Schreiben:
// Bleibt von einer früheren Aufnahme gleichen Namens eine Vorschau liegen und scheitert das Neuschreiben, darf die alte
// nicht stehen bleiben — sie zeigte sonst einen fremden, womöglich ungeschwärzten Inhalt (F-115, Kernprinzip 2).
func TestAblegenEntferntVeralteteVorschau(t *testing.T) {
	b, _ := neueBibliothek(t)
	d := neuesDokument(t, 40, 30)
	e, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{})
	if err != nil {
		t.Fatal(err)
	}
	vorschau := b.VorschauPfad(e.Pfad)
	// Die Aufnahme verschwindet (Datei weg), ihre Vorschau bleibt liegen; die nächste bekommt denselben Namen.
	if err := os.Remove(ablageDatei(b, e.Pfad)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(vorschau); err != nil {
		t.Fatalf("Vorschau der früheren Aufnahme fehlt: %v", err)
	}
	ersetzeVorschauNaht(t, func(io.Writer, image.Image) error { return errKodieren }, nil)
	neu, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{})
	if neu.Pfad != e.Pfad {
		t.Fatalf("Pfad %q, Soll der gleiche Name %q", neu.Pfad, e.Pfad)
	}
	if !errors.Is(err, ErrVorschau) || errors.Is(err, ErrVorschauAlt) {
		t.Fatalf("Fehler %v, Soll ErrVorschau ohne ErrVorschauAlt", err)
	}
	if _, err := os.Stat(vorschau); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("veraltete Vorschau liegt noch: %v", err)
	}
	// Lässt sich die alte nicht entfernen, meldet Ablegen ErrVorschauAlt (Eintrag und Datei gelten trotzdem).
	if err := os.Remove(ablageDatei(b, e.Pfad)); err != nil {
		t.Fatal(err)
	}
	ersetzeVorschauNaht(t, nil, func(string) error { return fs.ErrPermission })
	neu, err = b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{})
	if !errors.Is(err, ErrVorschauAlt) || !errors.Is(err, ErrVorschau) || neu.Pfad != e.Pfad {
		t.Errorf("Fehler %v, Eintrag %+v, Soll ErrVorschau und ErrVorschauAlt mit gültigem Eintrag", err, neu)
	}
	if _, err := os.Stat(ablageDatei(b, e.Pfad)); err != nil {
		t.Errorf("Aufnahme nicht geschrieben: %v", err)
	}
}
