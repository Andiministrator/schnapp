// SPDX-License-Identifier: GPL-3.0-or-later

package settings

import (
	"errors"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

func beispielStile() []Stil {
	return []Stil{
		{Art: ArtPfeil, Farbe: color.NRGBA{R: 0xD9, G: 0x3A, B: 0x33, A: 0xFF}, Breite: 8},
		{Art: ArtKasten, Farbe: color.NRGBA{R: 0xFF, G: 0xD6, B: 0x0A, A: 0xFF}, Fuellung: color.NRGBA{R: 0xFF, G: 0xD6, B: 0x0A, A: 0x66}, Breite: 4},
		{Art: ArtText, Farbe: color.NRGBA{A: 0xFF}, Fuellung: color.NRGBA{R: 0xFF, G: 0xD6, B: 0x0A, A: 0xFF},
			Schriftgrad: 24.5, Schrift: "Test Sans", Fett: true, Kursiv: true, Unterstr: true, Durchgestr: true,
			Ausrichtung: document.AusrichtungMitte},
		{Art: ArtSprechblase, Farbe: color.NRGBA{A: 0xFF}, Schriftgrad: 20, Ausrichtung: document.AusrichtungRechts},
		{Art: ArtNummer, Farbe: color.NRGBA{R: 0x2F, G: 0x6F, B: 0xDB, A: 0xFF}, Radius: 14},
		{Art: ArtLupe, Farbe: color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}, Breite: 4},
		{Art: ArtKasten, Farbe: color.NRGBA{R: 0x11, A: 0xFF}, Breite: 6, // mit Schatten (ADR-0021 Nachtrag 26 Punkt 8)
			Schatten: document.Schatten{An: true, Winkel: 112.5, Laenge: 6, Weich: 0, Farbe: color.NRGBA{B: 0x40, A: 0x66}}},
	}
}

// Schreiben und Lesen sind zueinander umgekehrt; die Reihenfolge der Abschnitte bleibt.
func TestStileRundlauf(t *testing.T) {
	ein := beispielStile()
	text, err := FormatStile(ein)
	if err != nil {
		t.Fatal(err)
	}
	aus, warn := ParseStile("styles.ini", text)
	if len(warn) != 0 {
		t.Fatalf("Warnungen: %q", warn)
	}
	if !reflect.DeepEqual(aus, ein) {
		t.Errorf("Rundlauf: %+v, Soll %+v", aus, ein)
	}
	for _, s := range []string{"color = #D93A33FF", "fill = #FFD60A66", "size = 24.5", "align = center", "font = Test Sans",
		"shadow = 112.5 6 0 #00004066"} {
		if !strings.Contains(text, s) {
			t.Errorf("Datei ohne %q:\n%s", s, text)
		}
	}
	if strings.Count(text, "shadow =") != 1 {
		t.Errorf("shadow steht an Stilen ohne Schatten:\n%s", text)
	}
	if !strings.HasPrefix(text, "# Quick styles") {
		t.Errorf("Kopfkommentar fehlt")
	}
}

func TestStileParseRegeln(t *testing.T) {
	fall := []struct {
		name, ein string
		stile     int    // Zahl der behaltenen Stile
		warn      string // Teil der (einzigen) Warnung, leer = keine
	}{
		{"leer", "", 0, ""},
		{"nur Farbe, Rest Vorgabe", "[arrow]\ncolor = #112233\n", 1, ""},
		{"Farbe fehlt", "[arrow]\nwidth = 4\n", 0, "key \"color\" is missing"},
		{"unbekannter Schlüssel", "[arrow]\ncolor = #112233FF\nglow = 3\n", 0, "unknown key \"glow\""},
		{"Schlüssel einer anderen Art", "[arrow]\ncolor = #112233FF\nfill = #112233FF\n", 0, "unknown key \"fill\""},
		{"doppelter Schlüssel", "[arrow]\ncolor = #112233FF\ncolor = #112233FF\n", 0, "set twice"},
		{"ungültige Farbe", "[box]\ncolor = rot\n", 0, "invalid value \"rot\""},
		{"Farbe ohne Raute", "[box]\ncolor = a112233\n", 0, "invalid value"},
		{"Farbe mit falscher Länge", "[box]\ncolor = #12345\n", 0, "invalid value"},
		{"Breite null", "[arrow]\ncolor = #112233\nwidth = 0\n", 0, "invalid value"},
		{"Breite NaN", "[arrow]\ncolor = #112233\nwidth = NaN\n", 0, "invalid value"},
		{"Breite unendlich", "[arrow]\ncolor = #112233\nwidth = Inf\n", 0, "invalid value"},
		{"Breite zu groß", "[arrow]\ncolor = #112233\nwidth = 1e9\n", 0, "invalid value"},
		{"Ausrichtung ungültig", "[text]\ncolor = #112233\nalign = oben\n", 0, "invalid value"},
		{"Schrift mit Steuerzeichen", "[text]\ncolor = #112233\nfont = a\x01b\n", 0, "invalid value"},
		{"Wahrheitswert ungültig", "[text]\ncolor = #112233\nbold = vielleicht\n", 0, "invalid value"},
		{"unbekannter Abschnitt", "[triangle]\ncolor = #112233\n", 0, "Unknown section [triangle]"},
		{"kaputter Kopf", "[arrow\ncolor = #112233\n", 0, "Broken section header"},
		{"Schlüssel vor jedem Abschnitt", "color = #112233\n[arrow]\ncolor = #112233\n", 1, "outside of any section"},
		{"ein kaputter Stil lässt die anderen stehen", "[arrow]\ncolor = x\n[box]\ncolor = #112233\n[step]\ncolor = #112233\n", 2, "invalid value"},
		{"Schatten", "[step]\ncolor = #112233\nshadow = 135 6 8 #00000066\n", 1, ""},
		{"Schatten mit Winkel über 360", "[arrow]\ncolor = #112233\nshadow = 495 0 0 #000000\n", 1, ""},
		{"Schatten mit drei Werten", "[arrow]\ncolor = #112233\nshadow = 135 6 #00000066\n", 0, "invalid value"},
		{"Schatten negativ", "[box]\ncolor = #112233\nshadow = 135 -1 8 #00000066\n", 0, "invalid value"},
		{"Schatten zu lang", "[box]\ncolor = #112233\nshadow = 135 1001 8 #00000066\n", 0, "invalid value"},
		{"Schatten zu weich", "[box]\ncolor = #112233\nshadow = 135 6 201 #00000066\n", 0, "invalid value"},
		{"Schatten NaN", "[box]\ncolor = #112233\nshadow = NaN 6 8 #00000066\n", 0, "invalid value"},
		{"Schatten ohne Farbe", "[box]\ncolor = #112233\nshadow = 135 6 8 schwarz\n", 0, "invalid value"},
		{"Kommentare, Leerzeilen, Byte-Marke", "\xef\xbb\xbf# k\n; k\n\n[step]\n color = #112233 \n radius = 10\n", 1, ""},
	}
	for _, f := range fall {
		got, warn := ParseStile("x.ini", f.ein)
		if len(got) != f.stile {
			t.Errorf("%s: %d Stile, Soll %d", f.name, len(got), f.stile)
		}
		if f.warn == "" && len(warn) != 0 {
			t.Errorf("%s: unerwartete Warnung %q", f.name, warn)
		}
		if f.warn != "" && (len(warn) != 1 || !strings.Contains(warn[0], f.warn)) {
			t.Errorf("%s: Warnungen %q, Soll eine mit %q", f.name, warn, f.warn)
		}
	}
}

func TestStileVorgabenBeiFehlendemSchluessel(t *testing.T) {
	got, _ := ParseStile("x", "[text]\ncolor = #112233\n[step]\ncolor = #112233\n[arrow]\ncolor = #112233\n")
	if len(got) != 3 || got[0].Schriftgrad != 20 || got[1].Radius != 14 || got[2].Breite != 6 {
		t.Errorf("Vorgaben: %+v", got)
	}
	if got[0].Fuellung.A != 0 {
		t.Errorf("Füllung ohne Angabe ist durchsichtig, nicht %+v", got[0].Fuellung)
	}
}

// Höchstens 16 je Art; der 17. fällt mit Warnung weg, andere Arten zählen getrennt.
func TestStileGrenzeJeArt(t *testing.T) {
	var b strings.Builder
	for i := 0; i < MaxEigeneStile+1; i++ {
		b.WriteString("[arrow]\ncolor = #112233\n")
	}
	b.WriteString("[box]\ncolor = #112233\n")
	got, warn := ParseStile("x", b.String())
	if len(got) != MaxEigeneStile+1 || len(warn) != 1 || !strings.Contains(warn[0], "more than 16") {
		t.Errorf("%d Stile, Warnungen %q", len(got), warn)
	}
}

func TestStileWarnungsGrenze(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 40; i++ {
		b.WriteString("[arrow]\ncolor = rot\n")
	}
	_, warn := ParseStile("x", b.String())
	if len(warn) != maxWarnungen+1 || !strings.Contains(warn[len(warn)-1], "20 more") {
		t.Errorf("%d Warnungen, letzte %q", len(warn), warn[len(warn)-1])
	}
}

func stileMitHome(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", d)
	t.Setenv("HOME", filepath.Join(d, "heim"))
	return d
}

func TestStileSchreibenLesenDatei(t *testing.T) {
	d := stileMitHome(t)
	if got, warn := LadeStile(); len(got) != 0 || len(warn) != 0 {
		t.Fatalf("fehlende Datei: %v %q", got, warn)
	}
	ein := beispielStile()
	if err := SchreibeStile(ein); err != nil {
		t.Fatal(err)
	}
	pfad := filepath.Join(d, "schnapp", "styles.ini")
	if p, err := StilePfad(); err != nil || p != pfad {
		t.Errorf("Pfad %q %v, Soll %q", p, err, pfad)
	}
	info, err := os.Stat(pfad)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("Datei: %v %v, Soll 0600", info, err)
	}
	if o, err := os.Stat(filepath.Dir(pfad)); err != nil || o.Mode().Perm() != 0o700 {
		t.Errorf("Ordner: %v %v, Soll 0700", o, err)
	}
	got, warn := LadeStile()
	if len(warn) != 0 || !reflect.DeepEqual(got, ein) {
		t.Errorf("gelesen %+v, Warnungen %q", got, warn)
	}
	// ganz neu: Handschrift geht verloren, nichts bleibt liegen
	if err := os.WriteFile(pfad, []byte("# meine Notiz\n[arrow]\ncolor = #112233\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := SchreibeStile(ein[:1]); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(pfad); strings.Contains(string(b), "Notiz") {
		t.Errorf("Datei nicht ganz neu:\n%s", b)
	}
	if es, _ := os.ReadDir(filepath.Dir(pfad)); len(es) != 1 {
		t.Errorf("Rest im Ordner: %v", es)
	}
}

func TestStileSchreibenFehler(t *testing.T) {
	stileMitHome(t)
	for name, s := range map[string]Stil{
		"unbekannte Art":   {Art: "x", Breite: 1},
		"Breite 0":         {Art: ArtPfeil},
		"Schriftname leer": {Art: ArtText, Schriftgrad: 20, Schrift: " x"},
		"Ausrichtung":      {Art: ArtText, Schriftgrad: 20, Ausrichtung: 9},
		"Schrift Go":       {Art: ArtText, Schriftgrad: 20, Schrift: "Go"},
	} {
		err := SchreibeStile([]Stil{s})
		if err == nil || !strings.HasPrefix(err.Error(), "settings: ") {
			t.Errorf("%s: Fehler %v, Soll mit Präfix \"settings: \"", name, err)
		}
	}
	if p, _ := StilePfad(); func() bool { _, err := os.Stat(p); return err == nil }() {
		t.Errorf("fehlerhafter Aufruf hat die Datei angelegt")
	}
	// zu groß: 3000 Stile mit langem Schriftnamen
	var viele []Stil
	for i := 0; i < 3000; i++ {
		viele = append(viele, Stil{Art: ArtText, Schriftgrad: 20, Schrift: strings.Repeat("a", 100)})
	}
	if err := SchreibeStile(viele); !errors.Is(err, ErrStileZuGross) {
		t.Errorf("zu große Datei: %v", err)
	}
}

func TestStileLesenGrenzen(t *testing.T) {
	d := stileMitHome(t)
	pfad := filepath.Join(d, "schnapp", "styles.ini")
	if err := os.MkdirAll(filepath.Dir(pfad), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pfad, []byte(strings.Repeat("# x\n", 20000)), 0o600); err != nil {
		t.Fatal(err)
	}
	got, warn := LadeStile()
	if len(got) != 0 || len(warn) != 1 || !strings.Contains(warn[0], "64 KiB") {
		t.Errorf("große Datei: %v %q", got, warn)
	}
	// Verzeichnis statt Datei
	if err := os.Remove(pfad); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(pfad, 0o700); err != nil {
		t.Fatal(err)
	}
	if got, warn = LadeStile(); len(got) != 0 || len(warn) != 1 {
		t.Errorf("Verzeichnis: %v %q", got, warn)
	}
	if err := SchreibeStile(nil); !errors.Is(err, ErrStileKeineDatei) {
		t.Errorf("Schreiben auf Verzeichnis: %v", err)
	}
}

func TestStileVerweisBleibt(t *testing.T) {
	d := stileMitHome(t)
	ordner := filepath.Join(d, "schnapp")
	if err := os.MkdirAll(ordner, 0o700); err != nil {
		t.Fatal(err)
	}
	ziel := filepath.Join(d, "punkte.ini")
	if err := os.WriteFile(ziel, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	pfad := filepath.Join(ordner, "styles.ini")
	if err := os.Symlink(ziel, pfad); err != nil {
		t.Fatal(err)
	}
	if err := SchreibeStile(beispielStile()[:1]); err != nil {
		t.Fatal(err)
	}
	if l, err := os.Readlink(pfad); err != nil || l != ziel {
		t.Errorf("Verweis weg: %q %v", l, err)
	}
	if b, _ := os.ReadFile(ziel); !strings.Contains(string(b), "[arrow]") {
		t.Errorf("Ziel nicht geschrieben: %q", b)
	}
	// Verweis ins Leere
	if err := os.Remove(ziel); err != nil {
		t.Fatal(err)
	}
	if err := SchreibeStile(nil); !errors.Is(err, ErrStileVerweis) {
		t.Errorf("Verweis ins Leere: %v", err)
	}
}

func TestStilePfadFehlerPraefix(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	if _, err := StilePfad(); err == nil || !strings.HasPrefix(err.Error(), "settings: ") || strings.Count(err.Error(), "settings:") != 1 {
		t.Errorf("Fehler %v", err)
	}
	if err := SchreibeStile(nil); err == nil || strings.Count(err.Error(), "settings:") != 1 {
		t.Errorf("Schreiben ohne Heim: %v", err)
	}
	if _, warn := LadeStile(); len(warn) != 1 || strings.Contains(warn[0], "settings:") {
		t.Errorf("Warnung %q", warn)
	}
}

// Ein Winkel über 360 kommt auf 0 bis 360; ein Schatten, der sich nicht schreiben ließe, lehnt PruefeStil ab.
func TestStileSchattenGrenzen(t *testing.T) {
	aus, _ := ParseStile("x.ini", "[arrow]\ncolor = #112233\nshadow = 495 0 0 #000000\n")
	if len(aus) != 1 || aus[0].Schatten != (document.Schatten{An: true, Winkel: 135, Farbe: color.NRGBA{A: 0xFF}}) {
		t.Fatalf("gelesen %+v", aus)
	}
	for _, sch := range []document.Schatten{{An: true, Laenge: -1}, {An: true, Weich: 500}, {An: true, Winkel: 400}} {
		if PruefeStil(Stil{Art: ArtPfeil, Breite: 4, Schatten: sch}) == nil {
			t.Errorf("Schatten %+v angenommen", sch)
		}
	}
	// Ausgeschaltet zählt der Schatten nicht: Er fällt beim Bereinigen weg und wird nicht geschrieben.
	if s := (Stil{Art: ArtPfeil, Breite: 4, Schatten: document.Schatten{Laenge: -1}}); PruefeStil(s) != nil || s.Bereinigt().Schatten != (document.Schatten{}) {
		t.Error("ausgeschalteter Schatten stört")
	}
}
