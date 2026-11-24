// SPDX-License-Identifier: GPL-3.0-or-later

package settings

import (
	"errors"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func kastenStil(breite float64) Stil {
	return Stil{Art: ArtKasten, Farbe: color.NRGBA{R: 1, G: 2, B: 3, A: 255}, Breite: breite}
}

func stileInDatei(t *testing.T, pfad string) []Stil {
	t.Helper()
	l, warn := LadeStileDatei(pfad)
	if len(warn) != 0 {
		t.Fatalf("Warnungen: %q", warn)
	}
	return l
}

func hinzuStil(s Stil) func([]Stil) []Stil {
	return func(alt []Stil) []Stil { return StilHinzu(alt, s) }
}

func wegStil(s Stil) func([]Stil) []Stil {
	return func(alt []Stil) []Stil { return StilEntfernt(alt, s) }
}

// F-627: Zwei Sitzungen, je ein „+“ ohne Kenntnis der anderen: beide Stile stehen danach in der Datei; ein von
// Sitzung 1 entfernter Stil kommt durch Sitzung 2 nicht zurück.
func TestAendereStileZweiSitzungen(t *testing.T) {
	pfad := filepath.Join(t.TempDir(), "k", "styles.ini")
	a, b := kastenStil(5), kastenStil(6)
	if _, _, err := AendereStileDatei(pfad, hinzuStil(a)); err != nil {
		t.Fatal(err)
	}
	neu, gesichert, err := AendereStileDatei(pfad, hinzuStil(b)) // Sitzung 2 kennt a nicht
	if err != nil || gesichert {
		t.Fatalf("%v gesichert %v", err, gesichert)
	}
	if got := stileInDatei(t, pfad); len(got) != 2 || got[0] != a.Bereinigt() || got[1] != b.Bereinigt() || len(neu) != 2 {
		t.Errorf("Datei: %+v, Rückgabe %+v", got, neu)
	}
	// Sitzung 1 entfernt a; Sitzung 2 hat a noch in ihrer Liste, fügt c hinzu: a bleibt weg.
	if _, _, err := AendereStileDatei(pfad, wegStil(a)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := AendereStileDatei(pfad, hinzuStil(kastenStil(7))); err != nil {
		t.Fatal(err)
	}
	got := stileInDatei(t, pfad)
	if len(got) != 2 || got[0] != b.Bereinigt() || got[1] != kastenStil(7).Bereinigt() {
		t.Errorf("Datei nach Entfernen: %+v", got)
	}
	// Doppeltes und Entfernen eines fehlenden Stils ändern nichts.
	if _, _, err := AendereStileDatei(pfad, hinzuStil(b)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := AendereStileDatei(pfad, wegStil(a)); err != nil {
		t.Fatal(err)
	}
	if got := stileInDatei(t, pfad); len(got) != 2 {
		t.Errorf("Datei: %+v", got)
	}
	if st, err := os.Stat(pfad + ".lock"); err != nil || st.Mode().Perm() != 0o600 {
		t.Errorf("Sperrdatei: %v %v", st, err)
	}
}

// Gleichzeitig: Viele Läufer fügen je einen Stil hinzu, keiner geht verloren.
func TestAendereStileGleichzeitig(t *testing.T) {
	pfad := filepath.Join(t.TempDir(), "styles.ini")
	const n = MaxEigeneStile
	fertig := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() {
			_, _, err := AendereStileDatei(pfad, hinzuStil(kastenStil(float64(10+i))))
			fertig <- err
		}()
	}
	for i := 0; i < n; i++ {
		if err := <-fertig; err != nil {
			t.Fatal(err)
		}
	}
	if got := stileInDatei(t, pfad); len(got) != n {
		t.Errorf("%d Stile, Soll %d", len(got), n)
	}
}

// Die Sperre hält: Solange die Änderung des einen Fensters läuft, beginnt die des anderen nicht.
func TestAendereStileSperreHaelt(t *testing.T) {
	pfad := filepath.Join(t.TempDir(), "styles.ini")
	drin, weiter, zweiteDrin := make(chan struct{}), make(chan struct{}), make(chan struct{})
	fertig := make(chan error, 2)
	go func() {
		_, _, err := AendereStileDatei(pfad, func(alt []Stil) []Stil {
			close(drin)
			<-weiter
			return StilHinzu(alt, kastenStil(5))
		})
		fertig <- err
	}()
	<-drin
	go func() {
		_, _, err := AendereStileDatei(pfad, func(alt []Stil) []Stil {
			close(zweiteDrin)
			return StilHinzu(alt, kastenStil(6))
		})
		fertig <- err
	}()
	select {
	case <-zweiteDrin:
		t.Error("die zweite Änderung begann, obwohl die erste die Sperre hält")
	case <-time.After(300 * time.Millisecond):
	}
	close(weiter)
	for i := 0; i < 2; i++ {
		if err := <-fertig; err != nil {
			t.Fatal(err)
		}
	}
	if got := stileInDatei(t, pfad); len(got) != 2 {
		t.Errorf("%d Stile, Soll 2", len(got))
	}
}

// Ohne flock-Unterstützung läuft das Schreiben ohne Sperre weiter; ein anderer Fehler beim Sperren bricht ab.
func TestAendereStileOhneFlock(t *testing.T) {
	alt := stileFlock
	t.Cleanup(func() { stileFlock = alt })
	stileFlock = func(int, int) error { return syscall.ENOLCK }
	pfad := filepath.Join(t.TempDir(), "styles.ini")
	if _, _, err := AendereStileDatei(pfad, hinzuStil(kastenStil(5))); err != nil {
		t.Fatalf("ENOLCK: %v", err)
	}
	stileFlock = func(int, int) error { return syscall.EIO }
	if _, _, err := AendereStileDatei(pfad, func(l []Stil) []Stil { return l }); !errors.Is(err, syscall.EIO) {
		t.Errorf("EIO: %v", err)
	}
}

// F-629 b: Verworfene Stile in der Datei: vor dem Ersetzen entsteht styles.ini.bak mit dem alten Inhalt (0600); gibt
// es sie schon, bleibt die Datei unberührt und die Sicherung wird nicht überschrieben.
func TestAendereStileSichertUnlesbares(t *testing.T) {
	pfad := filepath.Join(t.TempDir(), "styles.ini")
	alt := "[box]\ncolor = #010203\nwidth = 5\n\n[box]\nwidth = nonsense\ncolor = #010203\n"
	if err := os.WriteFile(pfad, []byte(alt), 0o600); err != nil {
		t.Fatal(err)
	}
	neu, gesichert, err := AendereStileDatei(pfad, hinzuStil(kastenStil(9)))
	if err != nil || !gesichert || len(neu) != 2 {
		t.Fatalf("%v gesichert %v, %d Stile", err, gesichert, len(neu))
	}
	b, err := os.ReadFile(pfad + ".bak")
	if err != nil || string(b) != alt {
		t.Errorf("Sicherung: %q %v", b, err)
	}
	if st, _ := os.Stat(pfad + ".bak"); st.Mode().Perm() != 0o600 {
		t.Errorf("Rechte der Sicherung: %v", st.Mode().Perm())
	}
	// Die Datei ist jetzt sauber: weiteres Schreiben sichert nicht erneut.
	if _, gesichert, err := AendereStileDatei(pfad, hinzuStil(kastenStil(10))); err != nil || gesichert {
		t.Errorf("zweiter Lauf: %v %v", err, gesichert)
	}
	// Wieder Unlesbares, die Sicherung gibt es schon: verweigert, Datei und Sicherung unverändert.
	kaputt := "[box]\ncolor = #zzz\n"
	if err := os.WriteFile(pfad, []byte(kaputt), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := AendereStileDatei(pfad, hinzuStil(kastenStil(9))); !errors.Is(err, ErrStileSicherung) {
		t.Fatalf("Sicherung vorhanden: %v", err)
	}
	if d, _ := os.ReadFile(pfad); string(d) != kaputt {
		t.Errorf("Datei verändert: %q", d)
	}
	if d, _ := os.ReadFile(pfad + ".bak"); string(d) != alt {
		t.Errorf("Sicherung überschrieben: %q", d)
	}
}

// F-629 b: Eine Datei, die sich gar nicht lesen lässt (Verzeichnis, zu groß), bleibt unberührt.
func TestAendereStileUnlesbareDatei(t *testing.T) {
	d := t.TempDir()
	pfad := filepath.Join(d, "styles.ini")
	if err := os.Mkdir(pfad, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, _, err := AendereStileDatei(pfad, func(l []Stil) []Stil { return l }); !errors.Is(err, ErrStileUnlesbar) {
		t.Errorf("Verzeichnis: %v", err)
	}
	pfad2 := filepath.Join(d, "gross", "styles.ini")
	if err := os.MkdirAll(filepath.Dir(pfad2), 0o700); err != nil {
		t.Fatal(err)
	}
	gross := strings.Repeat("# x\n", maxDateigroesse/4+10)
	if err := os.WriteFile(pfad2, []byte(gross), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := AendereStileDatei(pfad2, func(l []Stil) []Stil { return l }); !errors.Is(err, ErrStileUnlesbar) {
		t.Errorf("zu groß: %v", err)
	}
	if b, _ := os.ReadFile(pfad2); string(b) != gross {
		t.Errorf("zu große Datei verändert")
	}
	if _, err := os.Stat(pfad2 + ".bak"); err == nil {
		t.Errorf("Sicherung ohne Anlass")
	}
}

func TestPruefeStilOeffentlich(t *testing.T) {
	if PruefeStil(kastenStil(0)) == nil || PruefeStil(Stil{Art: ArtText, Schriftgrad: 20, Schrift: " X"}) == nil || PruefeStil(kastenStil(4)) != nil {
		t.Errorf("PruefeStil")
	}
}

// StilHinzu hält die Grenze je Art und lässt die übergebene Liste unverändert.
func TestStilHinzuGrenze(t *testing.T) {
	var l []Stil
	for i := 0; i < MaxEigeneStile+3; i++ {
		l = StilHinzu(l, kastenStil(float64(5+i)))
	}
	if len(l) != MaxEigeneStile {
		t.Errorf("%d Stile", len(l))
	}
	l2 := StilHinzu(l, Stil{Art: ArtLupe, Farbe: color.NRGBA{A: 255}, Breite: 4})
	if len(l2) != MaxEigeneStile+1 || len(l) != MaxEigeneStile {
		t.Errorf("andere Art: %d, Original %d", len(l2), len(l))
	}
}
