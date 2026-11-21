// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/fileformat"
)

// Nacharbeit nach dem Verify (Fable) zu Karte #114: Z-1 bis Z-3, temporäre Reste ohne Altersgrenze, beim Entfernen und
// im Vorschau-Ordner. Alles in eigenen Testverzeichnissen (neueBibliothek).

// setzeAlter setzt die Änderungszeit von pfad auf jetzt+um (wirkt bei einem Hardlink auf den gemeinsamen Inode).
func setzeAlter(t *testing.T, pfad string, um time.Duration) {
	t.Helper()
	z := time.Now().Add(um)
	if err := os.Chtimes(pfad, z, z); err != nil {
		t.Fatal(err)
	}
}

// dateienMitInhalt nennt die regulären Dateien im Ordner, deren Inhalt genau alt ist.
func dateienMitInhalt(t *testing.T, ordner string, alt []byte) []string {
	t.Helper()
	var namen []string
	es, err := os.ReadDir(ordner)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range es {
		if !x.Type().IsRegular() {
			continue
		}
		roh, err := os.ReadFile(filepath.Join(ordner, x.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Equal(roh, alt) {
			namen = append(namen, x.Name())
		}
	}
	return namen
}

// ablegenMitRest legt ab, wobei das Entfernen der temporären Datei scheitert: genau der Weg, den F-274 beschreibt.
// Zurück kommen die Aufnahme und der Rest (ein Hardlink der Aufnahme mit deren Änderungszeit).
func ablegenMitRest(t *testing.T, b *Bibliothek) (e Eintrag, rest string) {
	t.Helper()
	orig := entferneTemp
	entferneTemp = func(string) error { return errors.New("Entfernen scheitert") }
	d := neuesDokument(t, 40, 30)
	e, err := b.Ablegen(d, d.Original(), fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")})
	entferneTemp = orig
	if !errors.Is(err, ErrTempRest) {
		t.Fatalf("Testaufbau: %v", err)
	}
	es, err := os.ReadDir(filepath.Dir(ablageDatei(b, e.Pfad)))
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range es {
		if len(x.Name()) > len(tempPraefix) && x.Name()[:len(tempPraefix)] == tempPraefix {
			rest = filepath.Join(filepath.Dir(ablageDatei(b, e.Pfad)), x.Name())
		}
	}
	if rest == "" {
		t.Fatal("Testaufbau: kein Rest")
	}
	return e, rest
}

// Z-1: Der Rest nach einem gescheiterten Entfernen ist ein Hardlink mit der Änderungszeit der Aufnahme, also jünger
// als eine Minute. Wer sofort schwärzt und speichert, behält ihn nicht: Kein Name trägt danach den alten Inhalt.
func TestSpeichernRaeumtJungenRestMitGleichemInode(t *testing.T) {
	b, _ := neueBibliothek(t)
	e, rest := ablegenMitRest(t, b)
	voll := ablageDatei(b, e.Pfad)
	alt := lies(t, voll)
	neu := neuesDokument(t, 12, 9) // „geschwärzter“ Stand
	if _, _, err := b.Speichern(e.Pfad, standIn(t, b, e.Pfad), neu, neu.Original()); err != nil {
		t.Fatalf("Speichern: %v (Soll: Rest geräumt, keine Meldung)", err)
	}
	if _, err := os.Lstat(rest); !os.IsNotExist(err) {
		t.Errorf("junger Rest bleibt: %v", err)
	}
	if namen := dateienMitInhalt(t, filepath.Dir(voll), alt); len(namen) != 0 {
		t.Errorf("Dateien mit altem Inhalt: %v", namen)
	}
}

// Z-1b: Auch ein Rest mit Änderungszeit in der Zukunft (verstellte Uhr, Netzlaufwerk) geht, wenn er denselben Inode
// hat wie die zu ersetzende Datei.
func TestSpeichernRaeumtRestMitZukunftszeitUndGleichemInode(t *testing.T) {
	b, _ := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")})
	voll := ablageDatei(b, e.Pfad)
	alt := lies(t, voll)
	rest := filepath.Join(filepath.Dir(voll), tempPraefix+"zukunft")
	if err := os.Link(voll, rest); err != nil {
		t.Fatal(err)
	}
	setzeAlter(t, rest, 48*time.Hour)
	neu := neuesDokument(t, 12, 9)
	if _, _, err := b.Speichern(e.Pfad, standIn(t, b, e.Pfad), neu, neu.Original()); err != nil {
		t.Fatalf("Speichern: %v", err)
	}
	if namen := dateienMitInhalt(t, filepath.Dir(voll), alt); len(namen) != 0 {
		t.Errorf("Dateien mit altem Inhalt: %v", namen)
	}
}

// Z-1c: Ein Rest mit anderem Inode wird nach Alter beurteilt: Älter als eine Minute geht er, jünger bleibt er und wird
// gemeldet (nicht gelöscht), mit Zeit weit in der Zukunft ebenso. Die Altersgrenze ist damit festgenagelt: Wer sie
// entfernt, löscht den jungen Rest, wer sie nie greifen lässt, lässt den alten liegen.
func TestSpeichernAltersgrenzeBeiFremdemRest(t *testing.T) {
	b, _ := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")})
	ordner := filepath.Dir(ablageDatei(b, e.Pfad))
	rest := func(name string, um time.Duration) string {
		p := filepath.Join(ordner, tempPraefix+name)
		schreibe(t, p, "halb geschrieben")
		setzeAlter(t, p, um)
		return p
	}
	alt, grenze := rest("alt", -time.Hour), rest("grenze", -61*time.Second)
	jung, zukunft := rest("jung", -30*time.Second), rest("zukunft", 48*time.Hour)
	neu := neuesDokument(t, 12, 9)
	_, _, err := b.Speichern(e.Pfad, standIn(t, b, e.Pfad), neu, neu.Original())
	if !errors.Is(err, ErrTempRest) {
		t.Fatalf("Fehler %v (Soll: ErrTempRest für den jungen und den zukunftsdatierten Rest)", err)
	}
	for _, p := range []string{alt, grenze} {
		if _, err := os.Lstat(p); !os.IsNotExist(err) {
			t.Errorf("%s bleibt: %v", filepath.Base(p), err)
		}
	}
	for _, p := range []string{jung, zukunft} {
		if _, err := os.Lstat(p); err != nil {
			t.Errorf("%s gelöscht: %v", filepath.Base(p), err)
		}
	}
	for _, name := range []string{"jung", "zukunft"} {
		if !bytes.Contains([]byte(err.Error()), []byte(tempPraefix+name)) {
			t.Errorf("Meldung nennt %s nicht: %v", name, err)
		}
	}
}

// Schutzprobe: Speichern räumt nur reguläre Dateien mit dem Präfix. Ordner, Symlinks, ähnliche Namen, Aufnahmen,
// fremde Dateien und Dateien in anderen Ordnern bleiben.
func TestSpeichernLoeschtNurEigeneReste(t *testing.T) {
	b, _ := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")})
	ordner := filepath.Dir(ablageDatei(b, e.Pfad))
	bleibt := map[string]string{}
	datei := func(name string, um time.Duration) string {
		p := filepath.Join(ordner, name)
		schreibe(t, p, "inhalt "+name)
		setzeAlter(t, p, um)
		return p
	}
	bleibt["ähnlich ohne Strich"] = datei(".schnapp-tmp", -time.Hour)
	bleibt["Präfix davor"] = datei("x"+tempPraefix+"1", -time.Hour)
	bleibt["Großschreibung"] = datei(".Schnapp-tmp-1", -time.Hour)
	bleibt["notiz"] = datei("notiz.txt", -time.Hour)
	bleibt["andere Aufnahme"] = datei("2026-11-19_10-00-00.png", -time.Hour)
	dir := filepath.Join(ordner, tempPraefix+"ordner")
	bleibt["Ordner"] = dir
	bleibt["Datei im Ordner"] = filepath.Join(dir, tempPraefix+"innen")
	schreibe(t, bleibt["Datei im Ordner"], "x")
	setzeAlter(t, bleibt["Datei im Ordner"], -time.Hour)
	leer := filepath.Join(ordner, tempPraefix+"leer")
	if err := os.Mkdir(leer, 0o700); err != nil {
		t.Fatal(err)
	}
	setzeAlter(t, leer, -time.Hour)
	bleibt["leerer Ordner"] = leer
	aussen := filepath.Join(t.TempDir(), "wertvoll.txt")
	schreibe(t, aussen, "wertvoll")
	setzeAlter(t, aussen, -time.Hour)
	verweis := filepath.Join(ordner, tempPraefix+"verweis")
	if err := os.Symlink(aussen, verweis); err != nil {
		t.Fatal(err)
	}
	bleibt["Symlink"], bleibt["Symlink-Ziel"] = verweis, aussen
	bleibt["Nachbarordner"] = filepath.Join(b.Orte.Ablage, "2026-10", tempPraefix+"alt")
	schreibe(t, bleibt["Nachbarordner"], "x")
	setzeAlter(t, bleibt["Nachbarordner"], -time.Hour)
	weg := datei(tempPraefix+"alt", -time.Hour)

	neu := neuesDokument(t, 12, 9)
	if _, _, err := b.Speichern(e.Pfad, standIn(t, b, e.Pfad), neu, neu.Original()); err != nil {
		t.Fatalf("Speichern: %v", err)
	}
	for was, p := range bleibt {
		if _, err := os.Lstat(p); err != nil {
			t.Errorf("%s gelöscht: %v", was, err)
		}
	}
	if _, err := os.Lstat(weg); !os.IsNotExist(err) {
		t.Errorf("alter Rest bleibt: %v", err)
	}
	if got := lies(t, aussen); string(got) != "wertvoll" {
		t.Error("Ziel des Symlinks angefasst")
	}
}

// Rennen zwischen Prüfen und Entfernen: Wird der Name zwischen ReadDir und Remove gegen einen Symlink getauscht,
// trifft os.Remove den Symlink selbst, nie das Ziel.
func TestRaeumeTempResteTauschGegenSymlink(t *testing.T) {
	b, _ := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")})
	ordner := filepath.Dir(ablageDatei(b, e.Pfad))
	aussen := filepath.Join(t.TempDir(), "wertvoll.txt")
	schreibe(t, aussen, "wertvoll")
	rest := filepath.Join(ordner, tempPraefix+"x")
	schreibe(t, rest, "x")
	setzeAlter(t, rest, -time.Hour)
	orig := entferneTemp
	t.Cleanup(func() { entferneTemp = orig })
	entferneTemp = func(p string) error {
		if err := os.Remove(p); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(aussen, p); err != nil {
			t.Fatal(err)
		}
		return orig(p)
	}
	n, err := raeumeTempReste(ordner, time.Now(), true)
	if n != 1 || err != nil {
		t.Fatalf("%d %v", n, err)
	}
	if got := lies(t, aussen); string(got) != "wertvoll" {
		t.Error("Ziel des Symlinks angefasst")
	}
}

// Z-1: Neuaufbau meldet einen zukunftsdatierten Rest (er wird nie alt genug), löscht ihn aber nicht.
func TestNeuaufbauMeldetZukunftsRest(t *testing.T) {
	b, _ := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")})
	rest := filepath.Join(filepath.Dir(ablageDatei(b, e.Pfad)), tempPraefix+"zukunft")
	schreibe(t, rest, "x")
	setzeAlter(t, rest, 48*time.Hour)
	erg, err := b.Neuaufbau()
	if err != nil || len(erg.TempReste) != 1 {
		t.Fatalf("Neuaufbau %+v %v (Soll: ein gemeldeter Rest)", erg, err)
	}
	if _, err := os.Lstat(rest); err != nil {
		t.Errorf("Rest gelöscht: %v", err)
	}
}

// Z-2: Entfernen räumt die temporären Kopien der Aufnahme mit (gleicher Inode, auch jung), sonst bliebe der
// ungeschwärzte Inhalt unter dem Namen der Kopie.
func TestEntfernenRaeumtTempRestDerAufnahme(t *testing.T) {
	b, _ := neueBibliothek(t)
	e, rest := ablegenMitRest(t, b)
	voll := ablageDatei(b, e.Pfad)
	alt := lies(t, voll)
	geloescht, err := b.Entfernen(e.Pfad, standIn(t, b, e.Pfad))
	if err != nil || !geloescht {
		t.Fatalf("Entfernen: %v %v", geloescht, err)
	}
	if _, err := os.Lstat(rest); !os.IsNotExist(err) {
		t.Errorf("Rest bleibt: %v", err)
	}
	if namen := dateienMitInhalt(t, filepath.Dir(voll), alt); len(namen) != 0 {
		t.Errorf("Dateien mit altem Inhalt: %v", namen)
	}
}

// Z-2: Lässt sich der Rest beim Entfernen nicht löschen, ist die Datei weg, und ErrTempRest sagt es (nicht still).
func TestEntfernenMeldetTempRest(t *testing.T) {
	b, _ := neueBibliothek(t)
	e, rest := ablegenMitRest(t, b)
	orig := entferneTemp
	entferneTemp = func(string) error { return errors.New("Entfernen scheitert") }
	t.Cleanup(func() { entferneTemp = orig })
	geloescht, err := b.Entfernen(e.Pfad, standIn(t, b, e.Pfad))
	if !geloescht || !errors.Is(err, ErrTempRest) {
		t.Fatalf("Entfernen: %v %v (Soll: gelöscht und ErrTempRest)", geloescht, err)
	}
	if _, err := os.Lstat(rest); err != nil {
		t.Errorf("Rest unerwartet weg: %v", err)
	}
}

// Z-3: Ein alter Rest im Vorschau-Ordner (Absturz zwischen Schreiben und Umbenennen) trägt das Bild des alten Stands.
// Speichern räumt ihn; ein junger bleibt (er kann zu einem zweiten Prozess gehören) und löst keine Meldung aus.
func TestSpeichernRaeumtAltenRestImVorschauOrdner(t *testing.T) {
	b, _ := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")})
	altVorschau := lies(t, b.VorschauPfad(e.Pfad))
	alt := filepath.Join(b.Orte.Vorschau, tempPraefix+"123")
	jung := filepath.Join(b.Orte.Vorschau, tempPraefix+"456")
	for _, p := range []string{alt, jung} {
		if err := os.WriteFile(p, altVorschau, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	setzeAlter(t, alt, -time.Hour)
	setzeAlter(t, jung, -10*time.Second)
	neu := neuesDokument(t, 12, 9)
	if _, _, err := b.Speichern(e.Pfad, standIn(t, b, e.Pfad), neu, neu.Original()); err != nil {
		t.Fatalf("Speichern: %v", err)
	}
	if _, err := os.Lstat(alt); !os.IsNotExist(err) {
		t.Errorf("alter Rest bleibt: %v", err)
	}
	if _, err := os.Lstat(jung); err != nil {
		t.Errorf("junger Rest gelöscht: %v", err)
	}
	if got := dateienMitInhalt(t, b.Orte.Vorschau, altVorschau); len(got) != 1 || got[0] != filepath.Base(jung) {
		t.Errorf("Dateien mit altem Vorschaubild: %v (Soll nur der junge Rest)", got)
	}
}

// Z-3: Ein Rest im Vorschau-Ordner, der sich nicht entfernen lässt, und einer mit Zeit weit in der Zukunft kommen als
// ErrTempRest zurück; das Speichern gilt trotzdem.
func TestSpeichernMeldetTempRestImVorschauOrdner(t *testing.T) {
	b, _ := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")})
	alt := filepath.Join(b.Orte.Vorschau, tempPraefix+"123")
	zukunft := filepath.Join(b.Orte.Vorschau, tempPraefix+"789")
	for _, p := range []string{alt, zukunft} {
		schreibe(t, p, "x")
	}
	setzeAlter(t, alt, -time.Hour)
	setzeAlter(t, zukunft, 48*time.Hour)
	orig := entferneTemp
	entferneTemp = func(string) error { return errors.New("Entfernen scheitert") }
	t.Cleanup(func() { entferneTemp = orig })
	neu := neuesDokument(t, 12, 9)
	_, _, err := b.Speichern(e.Pfad, standIn(t, b, e.Pfad), neu, neu.Original())
	if !errors.Is(err, ErrTempRest) {
		t.Fatalf("Fehler %v (Soll: ErrTempRest)", err)
	}
	for _, p := range []string{alt, zukunft} {
		if !bytes.Contains([]byte(err.Error()), []byte(filepath.Base(p))) {
			t.Errorf("Meldung nennt %s nicht: %v", filepath.Base(p), err)
		}
	}
	if g, err := fileformat.Oeffne(ablageDatei(b, e.Pfad)); err != nil || g.Dokument.Groesse().X != 12 {
		t.Errorf("Speichern hat nicht geschrieben: %v", err)
	}
}
