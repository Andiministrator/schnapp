// SPDX-License-Identifier: GPL-3.0-or-later

package bibliothek

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/datei"
)

// aufraeumBestand legt vier Aufnahmen an. Der Stichtag ist 2026-11-19 12:00 UTC, die Frist 90 Tage.
//
//	alt   : 2026-05-01, ohne Schlagworte, nicht bearbeitet  -> wird erfasst
//	wort  : 2026-05-01, mit Schlagwort                       -> geschützt
//	bearb : 2026-05-01, bearbeitet                           -> geschützt
//	jung  : 2026-11-01                                       -> geschützt (zu jung)
type aufraeumBestand struct {
	b                      *Bibliothek
	alt, wort, bearb, jung Eintrag
	jetzt                  time.Time
}

func neuerBestand(t *testing.T) aufraeumBestand {
	t.Helper()
	b, _ := neueBibliothek(t)
	alt := zeit("2026-05-01T10:00:00Z")
	return aufraeumBestand{
		b:     b,
		alt:   ablegen(t, b, datei.Metadaten{Erstellt: alt}),
		wort:  ablegen(t, b, datei.Metadaten{Erstellt: alt.Add(time.Minute), Schlagworte: []string{"behalten"}}),
		bearb: ablegen(t, b, datei.Metadaten{Erstellt: alt.Add(2 * time.Minute), Bearbeitet: true}),
		jung:  ablegen(t, b, datei.Metadaten{Erstellt: zeit("2026-11-01T10:00:00Z")}),
		jetzt: zeit("2026-11-19T12:00:00Z"),
	}
}

func (a aufraeumBestand) dateiDa(t *testing.T, e Eintrag) bool {
	t.Helper()
	_, err := os.Lstat(filepath.Join(a.b.Orte.Ablage, filepath.FromSlash(e.Pfad)))
	return err == nil
}

func (a aufraeumBestand) indexPfade(t *testing.T) []string {
	t.Helper()
	g, err := a.b.Laden()
	if err != nil {
		t.Fatal(err)
	}
	var p []string
	for _, e := range g.Eintraege {
		p = append(p, e.Pfad)
	}
	return p
}

func TestAufraeumenErfasstNurDasPassende(t *testing.T) {
	a := neuerBestand(t)
	erg, err := a.b.Aufraeumen(a.jetzt, Voreinstellung)
	if err != nil {
		t.Fatal(err)
	}
	if len(erg.Verschoben) != 1 || erg.Verschoben[0].Pfad != a.alt.Pfad || len(erg.Uebersprungen) != 0 {
		t.Fatalf("Ergebnis %+v", erg)
	}
	if a.dateiDa(t, a.alt) {
		t.Error("alte Aufnahme liegt noch in der Ablage")
	}
	for name, e := range map[string]Eintrag{"Schlagwort": a.wort, "Bearbeitet": a.bearb, "zu jung": a.jung} {
		if !a.dateiDa(t, e) {
			t.Errorf("%s schützt nicht: Datei ist weg", name)
		}
	}
	if got, want := a.indexPfade(t), []string{a.wort.Pfad, a.bearb.Pfad, a.jung.Pfad}; !reflect.DeepEqual(got, want) {
		t.Errorf("Index %v, soll %v", got, want)
	}
}

func TestAufraeumenPapierkorbInhalt(t *testing.T) {
	a := neuerBestand(t)
	if _, err := a.b.Aufraeumen(a.jetzt, Voreinstellung); err != nil {
		t.Fatal(err)
	}
	// Name in der Testzone (UTC+2): 2026-05-01 12:00:00.
	name := "2026-05-01_12-00-00.png"
	pk := a.b.Orte.Papierkorb
	if _, err := os.Stat(filepath.Join(pk, "files", name)); err != nil {
		t.Fatal(err)
	}
	info, err := os.ReadFile(filepath.Join(pk, "info", name+".trashinfo"))
	if err != nil {
		t.Fatal(err)
	}
	zeilen := strings.Split(string(info), "\n")
	// Sollwerte nach freedesktop-Trash-Spezifikation; Löschzeit lokal (Zone UTC+2): 14:00:00.
	if len(zeilen) != 4 || zeilen[0] != "[Trash Info]" || zeilen[2] != "DeletionDate=2026-11-19T14:00:00" || zeilen[3] != "" {
		t.Fatalf(".trashinfo = %q", info)
	}
	pfad, ok := strings.CutPrefix(zeilen[1], "Path=")
	if !ok || pfad != filepath.Join(a.b.Orte.Ablage, "2026-05", name) {
		t.Errorf("Path = %q", pfad)
	}
	for _, o := range []string{pk, filepath.Join(pk, "files"), filepath.Join(pk, "info")} {
		if m := modus(t, o); m != 0o700 {
			t.Errorf("%s hat %o, soll 700", o, m)
		}
	}
	if m := modus(t, filepath.Join(pk, "info", name+".trashinfo")); m != 0o600 {
		t.Errorf(".trashinfo hat %o", m)
	}
	// Die Datei im Papierkorb ist noch die ganze Arbeitsdatei.
	f, _ := os.Open(filepath.Join(pk, "files", name))
	defer f.Close()
	if g, err := datei.LeseArbeitsdatei(f); err != nil || g.Fremd {
		t.Errorf("Datei im Papierkorb: %v", err)
	}
	if n := namen(t, filepath.Join(pk, "info")); !reflect.DeepEqual(n, []string{name + ".trashinfo"}) {
		t.Errorf("info enthält %v", n)
	}
}

func TestAufraeumenPfadWirdProzentKodiert(t *testing.T) {
	env, heim := umgebung(t, nil)
	schreibe(t, filepath.Join(heim, ".config", "user-dirs.dirs"), "XDG_PICTURES_DIR=\"$HOME/Meine Bilder\"\n")
	o, err := OrteAus(env)
	if err != nil {
		t.Fatal(err)
	}
	b := Neu(o)
	b.Zone = time.UTC
	ablegen(t, b, datei.Metadaten{Erstellt: zeit("2026-05-01T10:00:00Z")})
	if _, err := b.Aufraeumen(zeit("2026-11-19T12:00:00Z"), Voreinstellung); err != nil {
		t.Fatal(err)
	}
	info, err := os.ReadFile(filepath.Join(o.Papierkorb, "info", "2026-05-01_10-00-00.png.trashinfo"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(info), "Path=/") || !strings.HasSuffix(strings.Split(string(info), "\n")[1], "/Meine%20Bilder/schnapp/2026-05/2026-05-01_10-00-00.png") {
		t.Errorf(".trashinfo = %q", info)
	}
}

func TestAufraeumenFristGrenze(t *testing.T) {
	b, _ := neueBibliothek(t)
	e := ablegen(t, b, datei.Metadaten{Erstellt: zeit("2026-08-21T12:00:00Z")}) // genau 90 Tage vor dem Stichtag
	erg, err := b.Aufraeumen(zeit("2026-11-19T12:00:00Z"), Voreinstellung)
	if err != nil || len(erg.Verschoben) != 0 {
		t.Fatalf("genau 90 Tage: %+v %v", erg, err)
	}
	erg, err = b.Aufraeumen(zeit("2026-11-19T12:00:01Z"), Voreinstellung) // eine Sekunde älter als die Frist
	if err != nil || len(erg.Verschoben) != 1 || erg.Verschoben[0].Pfad != e.Pfad {
		t.Fatalf("90 Tage + 1 s: %+v %v", erg, err)
	}
}

func TestAufraeumenAusWirkt(t *testing.T) {
	a := neuerBestand(t)
	erg, err := a.b.Aufraeumen(a.jetzt, Einstellung{An: false, Tage: 90})
	if err != nil || len(erg.Verschoben)+len(erg.Uebersprungen) != 0 {
		t.Fatalf("Aus: %+v %v", erg, err)
	}
	if !a.dateiDa(t, a.alt) {
		t.Error("Aus hat trotzdem verschoben")
	}
	if _, err := os.Stat(a.b.Orte.Papierkorb); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Aus hat den Papierkorb angelegt: %v", err)
	}
	// Gültiges Gegenstück: dieselbe Einstellung eingeschaltet erfasst die Aufnahme.
	if erg, _ := a.b.Aufraeumen(a.jetzt, Einstellung{An: true, Tage: 90}); len(erg.Verschoben) != 1 {
		t.Errorf("An: %+v", erg)
	}
}

func TestAufraeumenFristUngueltig(t *testing.T) {
	a := neuerBestand(t)
	for _, tage := range []int{0, -5} {
		if _, err := a.b.Aufraeumen(a.jetzt, Einstellung{An: true, Tage: tage}); !errors.Is(err, ErrFrist) {
			t.Errorf("Tage %d: %v", tage, err)
		}
	}
	if !a.dateiDa(t, a.alt) {
		t.Error("ungültige Frist hat verschoben")
	}
}

func TestAufraeumenKuerzereFrist(t *testing.T) {
	a := neuerBestand(t)
	erg, err := a.b.Aufraeumen(a.jetzt, Einstellung{An: true, Tage: 10}) // jung (18 Tage) fällt jetzt hinein
	if err != nil || len(erg.Verschoben) != 2 {
		t.Fatalf("%+v %v", erg, err)
	}
	if a.dateiDa(t, a.alt) || a.dateiDa(t, a.jung) || !a.dateiDa(t, a.wort) || !a.dateiDa(t, a.bearb) {
		t.Error("falsche Auswahl")
	}
}

// Der Index ist nur ein Cache: Trägt die Datei Schlagworte oder ist sie bearbeitet, schützt das, auch wenn der
// Index davon nichts weiß.
func TestAufraeumenPruftDieDateiNichtNurDenIndex(t *testing.T) {
	a := neuerBestand(t)
	// Die Datei bekommt ein Schlagwort, der Index wird auf den alten Stand (ohne) zurückgesetzt.
	if _, err := a.b.SetzeSchlagworte(a.alt.Pfad, []string{"nachtraeglich"}); err != nil {
		t.Fatal(err)
	}
	g, _ := a.b.Laden()
	for i := range g.Eintraege {
		if g.Eintraege[i].Pfad == a.alt.Pfad {
			g.Eintraege[i].Schlagworte = nil
		}
	}
	if err := a.b.schreibeIndex(g.Eintraege); err != nil {
		t.Fatal(err)
	}
	erg, err := a.b.Aufraeumen(a.jetzt, Voreinstellung)
	if err != nil || len(erg.Verschoben) != 0 || len(erg.Uebersprungen) != 1 || !errors.Is(erg.Uebersprungen[0].Grund, ErrGeschuetzt) {
		t.Fatalf("%+v %v", erg, err)
	}
	if !a.dateiDa(t, a.alt) {
		t.Error("Datei mit Schlagwort wurde verschoben")
	}
}

func TestAufraeumenFremdeDateiBleibt(t *testing.T) {
	a := neuerBestand(t)
	// Ein fremdes PNG, das ein manipulierter Index als alte Aufnahme ausgibt, wird nicht angerührt.
	fremd := filepath.Join(a.b.Orte.Ablage, "2026-05", "fremd.png")
	schreibe(t, fremd, "kein png")
	g, _ := a.b.Laden()
	g.Eintraege = append(g.Eintraege, Eintrag{Pfad: "2026-05/fremd.png", Erstellt: zeit("2026-01-01T00:00:00Z"), Breite: 1, Hoehe: 1})
	if err := a.b.schreibeIndex(g.Eintraege); err != nil {
		t.Fatal(err)
	}
	erg, err := a.b.Aufraeumen(a.jetzt, Voreinstellung)
	if err != nil || len(erg.Verschoben) != 1 || len(erg.Uebersprungen) != 1 || erg.Uebersprungen[0].Pfad != "2026-05/fremd.png" {
		t.Fatalf("%+v %v", erg, err)
	}
	if _, err := os.Stat(fremd); err != nil {
		t.Errorf("fremde Datei angefasst: %v", err)
	}
}

func TestAufraeumenNamenskollisionImPapierkorb(t *testing.T) {
	a := neuerBestand(t)
	name := "2026-05-01_12-00-00.png"
	pk := a.b.Orte.Papierkorb
	schreibe(t, filepath.Join(pk, "files", name), "vorhanden")
	schreibe(t, filepath.Join(pk, "info", name+".trashinfo"), "[Trash Info]\nPath=/x\nDeletionDate=2020-01-01T00:00:00\n")
	erg, err := a.b.Aufraeumen(a.jetzt, Voreinstellung)
	if err != nil || len(erg.Verschoben) != 1 {
		t.Fatalf("%+v %v", erg, err)
	}
	neu := "2026-05-01_12-00-00-2.png"
	if erg.Verschoben[0].Papierkorb != filepath.Join(pk, "files", neu) {
		t.Errorf("Papierkorb = %q", erg.Verschoben[0].Papierkorb)
	}
	if got, _ := os.ReadFile(filepath.Join(pk, "files", name)); string(got) != "vorhanden" {
		t.Errorf("vorhandene Datei im Papierkorb verändert: %q", got)
	}
	if got, _ := os.ReadFile(filepath.Join(pk, "info", name+".trashinfo")); !strings.Contains(string(got), "Path=/x") {
		t.Errorf("vorhandene .trashinfo verändert: %q", got)
	}
	if info, err := os.ReadFile(filepath.Join(pk, "info", neu+".trashinfo")); err != nil || !strings.HasPrefix(string(info), "[Trash Info]\nPath=/") {
		t.Errorf("neue .trashinfo: %q %v", info, err)
	}
}

func TestAufraeumenAnderesDateisystemBleibtLiegen(t *testing.T) {
	a := neuerBestand(t)
	ersetzeUmbenenne(t, func(von, nach string) error {
		if strings.HasPrefix(nach, a.b.Orte.Papierkorb) {
			return &os.LinkError{Op: "rename", Old: von, New: nach, Err: syscall.EXDEV}
		}
		return os.Rename(von, nach)
	})
	erg, err := a.b.Aufraeumen(a.jetzt, Voreinstellung)
	if err != nil {
		t.Fatal(err)
	}
	if len(erg.Verschoben) != 0 || len(erg.Uebersprungen) != 1 || !errors.Is(erg.Uebersprungen[0].Grund, ErrAnderesDateisystem) {
		t.Fatalf("%+v", erg)
	}
	if !a.dateiDa(t, a.alt) {
		t.Error("Aufnahme verschwand trotz EXDEV (kein Kopieren-und-Löschen erlaubt)")
	}
	if got := a.indexPfade(t); len(got) != 4 {
		t.Errorf("Index verlor Einträge: %v", got)
	}
	if n := namen(t, filepath.Join(a.b.Orte.Papierkorb, "files")); len(n) != 0 {
		t.Errorf("Papierkorb files: %v", n)
	}
}

func TestAufraeumenOhneTrashinfoWirdZurueckgelegt(t *testing.T) {
	a := neuerBestand(t)
	ersetzeUmbenenne(t, func(von, nach string) error {
		if strings.HasSuffix(nach, ".trashinfo") {
			return errors.New("Attrappe: .trashinfo nicht schreibbar")
		}
		return os.Rename(von, nach)
	})
	erg, err := a.b.Aufraeumen(a.jetzt, Voreinstellung)
	if err != nil || len(erg.Verschoben) != 0 || len(erg.Uebersprungen) != 1 {
		t.Fatalf("%+v %v", erg, err)
	}
	if !a.dateiDa(t, a.alt) {
		t.Error("Aufnahme nicht zurückgelegt")
	}
	pk := a.b.Orte.Papierkorb
	if n := append(namen(t, filepath.Join(pk, "files")), namen(t, filepath.Join(pk, "info"))...); len(n) != 0 {
		t.Errorf("Papierkorb enthält Reste: %v", n)
	}
}
