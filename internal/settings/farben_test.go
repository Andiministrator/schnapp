// SPDX-License-Identifier: GPL-3.0-or-later

package settings

// Eigene Farbfelder in colors.ini (ADR-0021 Nachtrag 24, Karte #364).

import (
	"errors"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
)

var (
	ffRot  = color.NRGBA{R: 0xD9, G: 0x3A, B: 0x33, A: 0xFF}
	ffHalb = color.NRGBA{R: 0x11, G: 0x22, B: 0x33, A: 0x80}
)

func TestParseFarben(t *testing.T) {
	inhalt := "\xef\xbb\xbf; Kommentar\n[swatches]\ncolor = #D93A33\ncolor=#11223380\ncolor = #D93A33FF\ncolor = rot\nfarbe = #000000\n[andere]\ncolor = #FFFFFF\n"
	farben, warn := ParseFarben("colors.ini", inhalt)
	if !slices.Equal(farben, []color.NRGBA{ffRot, ffHalb}) {
		t.Errorf("Farben %v", farben)
	}
	if len(warn) != 3 { // „rot“, „farbe“, [andere]
		t.Errorf("%d Warnungen: %q", len(warn), warn)
	}
	for _, w := range warn {
		if !strings.HasPrefix(w, "colors.ini:") {
			t.Errorf("Warnung ohne Datei und Zeile: %q", w)
		}
	}
	if _, warn := ParseFarben("x", "color = #FFFFFF\n"); len(warn) != 1 {
		t.Errorf("Zeile vor [swatches]: %q", warn)
	}
	// „#“ ist ein Kommentar wie in styles.ini (F-837), auch innerhalb des Abschnitts.
	if farben, warn := ParseFarben("x", "# meine Farben\n[swatches]\n# rot\ncolor = #FF0000\n"); len(warn) != 0 || len(farben) != 1 {
		t.Errorf("„#“-Kommentar: %v, Warnungen %q", farben, warn)
	}
}

func TestParseFarbenGrenze(t *testing.T) {
	var b strings.Builder
	b.WriteString("[swatches]\n")
	for i := range MaxFarbfelder + 2 {
		fmt.Fprintf(&b, "color = #0000%02X\n", i)
	}
	farben, warn := ParseFarben("x", b.String())
	if len(farben) != MaxFarbfelder || len(warn) != 2 {
		t.Errorf("%d Farben, %d Warnungen", len(farben), len(warn))
	}
}

func TestFormatFarbenRundlauf(t *testing.T) {
	alt := []color.NRGBA{ffRot, ffHalb}
	neu, warn := ParseFarben("x", FormatFarben(alt))
	if !slices.Equal(neu, alt) || len(warn) != 0 {
		t.Errorf("Rundlauf: %v, Warnungen %q", neu, warn)
	}
	if neu, _ := ParseFarben("x", FormatFarben(nil)); len(neu) != 0 {
		t.Errorf("leere Palette: %v", neu)
	}
}

func TestLadeFarbenDateiFehltUndLeer(t *testing.T) {
	ordner := t.TempDir()
	pfad := filepath.Join(ordner, "colors.ini")
	if farben, da, warn := LadeFarbenDatei(pfad); da || farben != nil || warn != nil {
		t.Errorf("ohne Datei: %v, da %v, %q", farben, da, warn)
	}
	if err := os.WriteFile(pfad, []byte(FormatFarben(nil)), 0o600); err != nil {
		t.Fatal(err)
	}
	if farben, da, _ := LadeFarbenDatei(pfad); !da || len(farben) != 0 {
		t.Errorf("leere Datei: %v, da %v (alle entfernt heißt: keine Farbfelder)", farben, da)
	}
}

func TestAendereFarbenDatei(t *testing.T) {
	ordner := filepath.Join(t.TempDir(), "schnapp")
	pfad := filepath.Join(ordner, "colors.ini")
	neu, err := AendereFarbenDatei(pfad, func(alt []color.NRGBA, da bool) []color.NRGBA {
		if da || len(alt) != 0 {
			t.Errorf("erste Änderung: da %v, alt %v", da, alt)
		}
		return []color.NRGBA{ffRot}
	})
	if err != nil || !slices.Equal(neu, []color.NRGBA{ffRot}) {
		t.Fatalf("anlegen: %v, %v", neu, err)
	}
	info, err := os.Stat(pfad)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("Datei: %v, %v", info, err)
	}
	if oi, err := os.Stat(ordner); err != nil || oi.Mode().Perm() != 0o700 {
		t.Errorf("Ordner: %v, %v", oi, err)
	}
	neu, err = AendereFarbenDatei(pfad, func(alt []color.NRGBA, da bool) []color.NRGBA {
		if !da || !slices.Equal(alt, []color.NRGBA{ffRot}) {
			t.Errorf("zweite Änderung: da %v, alt %v", da, alt)
		}
		return FarbeHinzu(alt, ffHalb)
	})
	if err != nil || !slices.Equal(neu, []color.NRGBA{ffRot, ffHalb}) {
		t.Fatalf("hinzu: %v, %v", neu, err)
	}
	if farben, _, _ := LadeFarbenDatei(pfad); !slices.Equal(farben, neu) {
		t.Errorf("gelesen %v, geschrieben %v", farben, neu)
	}
}

// Unlesbares wird vor dem Schreiben als colors.ini.bak gesichert; gibt es die Sicherung schon, bleibt alles.
func TestAendereFarbenSichertUnlesbares(t *testing.T) {
	ordner := t.TempDir()
	pfad := filepath.Join(ordner, "colors.ini")
	roh := "[swatches]\ncolor = #D93A33\ncolor = kaputt\n"
	if err := os.WriteFile(pfad, []byte(roh), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := AendereFarbenDatei(pfad, func(alt []color.NRGBA, _ bool) []color.NRGBA { return alt }); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(pfad + ".bak"); err != nil || string(b) != roh {
		t.Errorf("Sicherung: %q, %v", b, err)
	}
	if err := os.WriteFile(pfad, []byte(roh), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := AendereFarbenDatei(pfad, func(alt []color.NRGBA, _ bool) []color.NRGBA { return nil })
	if !errors.Is(err, ErrFarbenSicherung) {
		t.Errorf("zweite Sicherung: %v, Soll ErrFarbenSicherung", err)
	}
	if b, _ := os.ReadFile(pfad); string(b) != roh {
		t.Errorf("Datei verändert trotz Fehler: %q", b)
	}
}

func TestAendereFarbenUnlesbarBleibt(t *testing.T) {
	ordner := t.TempDir()
	pfad := filepath.Join(ordner, "colors.ini")
	if err := os.Mkdir(pfad, 0o700); err != nil { // keine reguläre Datei
		t.Fatal(err)
	}
	_, err := AendereFarbenDatei(pfad, func(alt []color.NRGBA, _ bool) []color.NRGBA { return alt })
	if !errors.Is(err, ErrFarbenUnlesbar) {
		t.Errorf("Ordner statt Datei: %v", err)
	}
	// Der Fehler trägt keinen Fehlerwert der Stildatei: fehlerGrund spräche sonst von styles.ini (F-837).
	if errors.Is(err, ErrStileKeineDatei) || err != nil && strings.Contains(err.Error(), "styles") {
		t.Errorf("Fehler nennt die Stildatei: %v", err)
	}
}

// F-835 (K-4): Hält ein anderes Fenster die Sperre, bleibt die Datei, wie sie ist (ErrFarbenGesperrt).
func TestAendereFarbenGesperrt(t *testing.T) {
	alt := stileFlock
	stileFlock = func(int, int) error { return syscall.EWOULDBLOCK }
	t.Cleanup(func() { stileFlock = alt })
	pfad := filepath.Join(t.TempDir(), "colors.ini")
	roh := FormatFarben([]color.NRGBA{ffRot})
	if err := os.WriteFile(pfad, []byte(roh), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := AendereFarbenDatei(pfad, func([]color.NRGBA, bool) []color.NRGBA { return nil })
	if !errors.Is(err, ErrFarbenGesperrt) {
		t.Errorf("gesperrt: %v", err)
	}
	if b, _ := os.ReadFile(pfad); string(b) != roh {
		t.Errorf("trotz Sperre geschrieben: %q", b)
	}
}

// F-835 (K-4): Ist colors.ini ein Verweis, wird das Ziel geschrieben und der Verweis bleibt; ein hängender Verweis
// bleibt unberührt.
func TestAendereFarbenVerweis(t *testing.T) {
	ordner := t.TempDir()
	ziel := filepath.Join(ordner, "woanders", "farben.ini")
	if err := os.MkdirAll(filepath.Dir(ziel), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ziel, []byte(FormatFarben(nil)), 0o600); err != nil {
		t.Fatal(err)
	}
	pfad := filepath.Join(ordner, "colors.ini")
	if err := os.Symlink(ziel, pfad); err != nil {
		t.Fatal(err)
	}
	if _, err := AendereFarbenDatei(pfad, func(alt []color.NRGBA, _ bool) []color.NRGBA { return FarbeHinzu(alt, ffRot) }); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(pfad); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("Verweis ersetzt: %v, %v", info, err)
	}
	if farben, _, _ := LadeFarbenDatei(ziel); !slices.Equal(farben, []color.NRGBA{ffRot}) {
		t.Errorf("Ziel: %v", farben)
	}
	haengend := filepath.Join(ordner, "haengend.ini")
	if err := os.Symlink(filepath.Join(ordner, "fehlt", "x.ini"), haengend); err != nil {
		t.Fatal(err)
	}
	if _, err := AendereFarbenDatei(haengend, func(alt []color.NRGBA, _ bool) []color.NRGBA { return alt }); err == nil {
		t.Error("hängender Verweis: kein Fehler")
	}
	if info, err := os.Lstat(haengend); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("hängender Verweis ersetzt: %v, %v", info, err)
	}
}

// F-835 (K-4): Liefert die Änderung mehr als MaxFarbfelder, wird beim Schreiben gekürzt.
func TestAendereFarbenKuerzt(t *testing.T) {
	pfad := filepath.Join(t.TempDir(), "colors.ini")
	neu, err := AendereFarbenDatei(pfad, func([]color.NRGBA, bool) []color.NRGBA {
		viele := make([]color.NRGBA, MaxFarbfelder+6)
		for i := range viele {
			viele[i] = color.NRGBA{G: uint8(i), A: 255}
		}
		return viele
	})
	if err != nil || len(neu) != MaxFarbfelder {
		t.Fatalf("%d Farben, %v", len(neu), err)
	}
	if farben, _, warn := LadeFarbenDatei(pfad); len(farben) != MaxFarbfelder || len(warn) != 0 {
		t.Errorf("gelesen %d, Warnungen %q", len(farben), warn)
	}
}

func TestFarbeHinzuUndEntfernt(t *testing.T) {
	alt := []color.NRGBA{ffRot}
	if neu := FarbeHinzu(alt, ffRot); !slices.Equal(neu, alt) {
		t.Errorf("doppelt: %v", neu)
	}
	if neu := FarbeHinzu(alt, color.NRGBA{R: 9}); !slices.Equal(neu, alt) {
		t.Errorf("durchsichtig: %v", neu)
	}
	voll := make([]color.NRGBA, MaxFarbfelder)
	for i := range voll {
		voll[i] = color.NRGBA{B: uint8(i), A: 255}
	}
	if neu := FarbeHinzu(voll, ffRot); len(neu) != MaxFarbfelder {
		t.Errorf("voll: %d", len(neu))
	}
	if neu := FarbeHinzu(alt, ffHalb); !slices.Equal(neu, []color.NRGBA{ffRot, ffHalb}) || len(alt) != 1 {
		t.Errorf("hinzu: %v (alt %v)", neu, alt)
	}
	if neu := FarbeEntfernt([]color.NRGBA{ffRot, ffHalb}, ffRot); !slices.Equal(neu, []color.NRGBA{ffHalb}) {
		t.Errorf("entfernt: %v", neu)
	}
	if neu := FarbeEntfernt(alt, ffHalb); !slices.Equal(neu, alt) {
		t.Errorf("fehlt: %v", neu)
	}
}
