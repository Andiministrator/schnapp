// SPDX-License-Identifier: GPL-3.0-or-later

package rescue

// Rettungskopie gegen Nebenwirkungen anderer Fenster (F-702, F-703): Das Aufräumen lässt die Kopie eines Fensters
// liegen, das die Ausgangsdatei noch hält, und eine Änderung der Metadaten der Datei (Schlagworte der Bibliothek) zieht
// die Kennung einer liegenden Kopie nach. Die Mutationsprobe scripts/mutanten/rescue.json prüft, dass die Tests die
// Fixes tragen.

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-702: Eine alte Kopie, deren Sperre ein Fenster hält, bleibt liegen; ohne Halter fällt sie wie bisher.
func TestAufraeumenLaesstKopieMitGehaltenerSperre(t *testing.T) {
	ort := t.TempDir()
	alt := time.Now().Add(-Haltezeit - 24*time.Hour)
	offen := Kennung{Pfad: "/offen"}
	zu := Kennung{Pfad: "/zu"}
	fremdeKopie(t, ort, offen)
	fremdeKopie(t, ort, zu)
	s := NeueSicherung(ort, offen) // hält die Sperre auf /offen
	defer s.Ende()
	for _, p := range []string{Name(offen.Pfad), Name(zu.Pfad)} {
		if err := os.Chtimes(filepath.Join(ort, p), alt, alt); err != nil {
			t.Fatal(err)
		}
	}
	if n, err := Aufraeumen(ort, time.Now()); n != 1 || err != nil {
		t.Errorf("%d entfernt, %v; erwartet 1 (nur /zu)", n, err)
	}
	if _, err := os.Stat(filepath.Join(ort, Name(offen.Pfad))); err != nil {
		t.Errorf("Kopie des offenen Fensters entfernt: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ort, Name(zu.Pfad))); !os.IsNotExist(err) {
		t.Errorf("Kopie ohne Halter liegt noch: %v", err)
	}
	// Die Sperrdatei bleibt, die Prüfung hat sie nicht entfernt und nichts angelegt.
	if _, err := os.Stat(filepath.Join(ort, sperrName(offen.Pfad))); err != nil {
		t.Errorf("Sperrdatei: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ort, sperrName(zu.Pfad))); !os.IsNotExist(err) {
		t.Errorf("Aufräumen hat eine Sperrdatei angelegt: %v", err)
	}
}

// F-702: Ist die Sperre nicht prüfbar (hier: ein Verzeichnis am Sperrnamen), bleibt die alte Kopie liegen.
func TestAufraeumenLaesstKopieBeiUnpruefbarerSperre(t *testing.T) {
	ort := t.TempDir()
	k := Kennung{Pfad: "/x"}
	fremdeKopie(t, ort, k)
	if err := os.Mkdir(filepath.Join(ort, sperrName(k.Pfad)), 0o700); err != nil {
		t.Fatal(err)
	}
	alt := time.Now().Add(-Haltezeit - time.Hour)
	if err := os.Chtimes(filepath.Join(ort, Name(k.Pfad)), alt, alt); err != nil {
		t.Fatal(err)
	}
	if n, _ := Aufraeumen(ort, time.Now()); n != 0 {
		t.Errorf("%d entfernt trotz unprüfbarer Sperre", n)
	}
	if _, err := os.Stat(filepath.Join(ort, Name(k.Pfad))); err != nil {
		t.Errorf("Kopie entfernt: %v", err)
	}
}

// aufnahmeMitKopie legt eine Ausgangsdatei und eine Rettungskopie zu ihr an (Sperre frei) und liefert den Inhalt der
// Kopie.
func aufnahmeMitKopie(t *testing.T) (pfad, ort string, kopie []byte) {
	t.Helper()
	pfad, k := ausgang(t, "bild mit schlagwort a")
	ort = t.TempDir()
	return pfad, ort, fremdeKopie(t, ort, k)
}

// ueberschreibe ersetzt den Inhalt der Datei pfad und setzt eine andere Änderungszeit (wie SetzeSchlagworte).
func ueberschreibe(t *testing.T, pfad, inhalt string) {
	t.Helper()
	if err := os.WriteFile(pfad, []byte(inhalt), 0o600); err != nil {
		t.Fatal(err)
	}
	z := time.Now().Add(time.Hour)
	if err := os.Chtimes(pfad, z, z); err != nil {
		t.Fatal(err)
	}
}

// F-703: Eine liegende Kopie bleibt nach einer Änderung der Metadaten anwendbar, mit allen Schwärzungen.
func TestUmDateiZiehtKennungNach(t *testing.T) {
	pfad, ort, vorher := aufnahmeMitKopie(t)
	err := UmDatei(ort, pfad, func(unv func() bool) error { unv(); ueberschreibe(t, pfad, "bild mit schlagwort b, länger"); return nil })
	if err != nil {
		t.Fatal(err)
	}
	k, _ := KennungVon(pfad)
	kp, err := Lies(ort, pfad)
	if err != nil || !kp.Datei.Gleich(k) {
		t.Fatalf("Kennung nicht nachgezogen: %v, %+v gegen %+v", err, kp, k)
	}
	if len(kp.Schwaerzungen) != 2 {
		t.Errorf("Schwärzungen verändert: %d", len(kp.Schwaerzungen))
	}
	if b, _ := os.ReadFile(filepath.Join(ort, Name(pfad))); bytes.Equal(b, vorher) {
		t.Error("Kopie nicht neu geschrieben")
	}
	if _, err := os.Stat(filepath.Join(ort, sperrName(pfad))); !os.IsNotExist(err) {
		t.Errorf("Sperre liegt nach UmDatei noch: %v", err)
	}
	// Das nächste Fenster wendet sie an.
	neu, _ := document.Neu(bild(64, 64))
	s := NeueSicherung(ort, k)
	defer s.Ende()
	if w := s.Wiederherstellen(neu, nil); !w.Angewendet || len(neu.Objekte()) != 2 {
		t.Errorf("nicht angewendet: %+v", w)
	}
}

// F-703: Passte die Kopie schon vor der Änderung nicht zur Datei, bleibt sie unverändert (sie gilt weiter als fremd).
func TestUmDateiLaesstFremdeKopieUnberuehrt(t *testing.T) {
	pfad, ort, vorher := aufnahmeMitKopie(t)
	ueberschreibe(t, pfad, "schon vorher anders")
	if err := UmDatei(ort, pfad, func(unv func() bool) error { unv(); ueberschreibe(t, pfad, "und nochmal anders"); return nil }); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(ort, Name(pfad))); !bytes.Equal(b, vorher) {
		t.Error("Kopie, die nicht zum Stand passte, wurde verändert")
	}
}

// F-703: Hält ein anderes Fenster die Sperre, führt es die Kopie: Sie bleibt unberührt, die Änderung läuft trotzdem.
func TestUmDateiMitGehaltenerSperre(t *testing.T) {
	pfad, k := ausgang(t, "bild")
	ort := t.TempDir()
	vorher := fremdeKopie(t, ort, k)
	s := NeueSicherung(ort, k)
	defer s.Ende()
	lief := false
	err := UmDatei(ort, pfad, func(unv func() bool) error { unv(); lief = true; ueberschreibe(t, pfad, "anders"); return nil })
	if err != nil || !lief {
		t.Fatalf("aendere lief nicht oder scheiterte: %v %v", lief, err)
	}
	if b, _ := os.ReadFile(filepath.Join(ort, Name(pfad))); !bytes.Equal(b, vorher) {
		t.Error("Kopie eines offenen Fensters verändert")
	}
}

// F-703: Scheitert die Änderung, bleibt die Kopie, und der Fehler kommt durch.
func TestUmDateiBeiFehlerVonAendere(t *testing.T) {
	pfad, ort, vorher := aufnahmeMitKopie(t)
	boese := errors.New("kaputt")
	if err := UmDatei(ort, pfad, func(func() bool) error { return boese }); !errors.Is(err, boese) {
		t.Errorf("Fehler: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(ort, Name(pfad))); !bytes.Equal(b, vorher) {
		t.Error("Kopie verändert")
	}
}

// F-703: Ohne Kopie läuft die Änderung und legt keine an.
func TestUmDateiOhneKopie(t *testing.T) {
	pfad, _ := ausgang(t, "bild")
	ort := t.TempDir()
	lief := false
	if err := UmDatei(ort, pfad, func(func() bool) error { lief = true; return nil }); err != nil || !lief {
		t.Fatalf("%v %v", lief, err)
	}
	if _, err := os.Stat(filepath.Join(ort, Name(pfad))); !os.IsNotExist(err) {
		t.Errorf("Kopie angelegt: %v", err)
	}
}
