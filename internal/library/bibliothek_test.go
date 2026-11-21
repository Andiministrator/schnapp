// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
)

func modus(t *testing.T, pfad string) os.FileMode {
	t.Helper()
	st, err := os.Stat(pfad)
	if err != nil {
		t.Fatal(err)
	}
	return st.Mode().Perm()
}

func namen(t *testing.T, ordner string) []string {
	t.Helper()
	es, err := os.ReadDir(ordner)
	if err != nil {
		t.Fatal(err)
	}
	var n []string
	for _, e := range es {
		n = append(n, e.Name())
	}
	return n
}

func TestAblegenPfadUndRechte(t *testing.T) {
	b, _ := neueBibliothek(t)
	d := neuesDokument(t, 600, 300)
	// 12:35:12 UTC ist in der Testzone (UTC+2) 14:35:12.
	e, err := b.Ablegen(d, d.Original(), fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z"), Quelle: "Fenster", Schlagworte: []string{"a", " ", "A", "b"}})
	if err != nil {
		t.Fatal(err)
	}
	if e.Pfad != "2026-11/2026-11-19_14-35-12.png" || e.Breite != 600 || e.Hoehe != 300 || e.Quelle != "Fenster" ||
		!reflect.DeepEqual(e.Schlagworte, []string{"a", "b"}) || !e.Erstellt.Equal(zeit("2026-11-19T12:35:12Z")) {
		t.Errorf("Eintrag = %+v", e)
	}
	datei1 := filepath.Join(b.Orte.Ablage, "2026-11", "2026-11-19_14-35-12.png")
	// Sollwerte der Rechte (ADR-0016): Ordner 0700, Dateien 0600.
	for _, o := range []string{b.Orte.Ablage, filepath.Dir(datei1), filepath.Dir(b.Orte.Index), b.Orte.Vorschau} {
		if m := modus(t, o); m != 0o700 {
			t.Errorf("Ordner %s hat %o, soll 700", o, m)
		}
	}
	for _, f := range []string{datei1, b.Orte.Index, b.VorschauPfad(e.Pfad)} {
		if m := modus(t, f); m != 0o600 {
			t.Errorf("Datei %s hat %o, soll 600", f, m)
		}
	}
	// Die Datei ist eine lesbare Arbeitsdatei mit den Metadaten.
	f, err := os.Open(datei1)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	g, err := fileformat.LeseArbeitsdatei(f)
	if err != nil || g.Fremd || g.Meta.Quelle != "Fenster" || !g.Meta.Erstellt.Equal(e.Erstellt) {
		t.Fatalf("Arbeitsdatei: %v %+v", err, g)
	}
	// Vorschau: längste Seite 256, Seitenverhältnis erhalten.
	vf, err := os.Open(b.VorschauPfad(e.Pfad))
	if err != nil {
		t.Fatal(err)
	}
	defer vf.Close()
	cfg, err := png.DecodeConfig(vf)
	if err != nil || cfg.Width != 256 || cfg.Height != 128 {
		t.Errorf("Vorschau %dx%d (%v), soll 256x128", cfg.Width, cfg.Height, err)
	}
	// Index enthält genau diese Zeile.
	gl, err := b.Laden()
	if err != nil || gl.Defekt != 0 || len(gl.Eintraege) != 1 || gl.Eintraege[0].Pfad != e.Pfad {
		t.Errorf("Laden: %v %+v", err, gl)
	}
	// Keine Restdatei der Zwischenschritte.
	if n := namen(t, filepath.Dir(datei1)); !reflect.DeepEqual(n, []string{"2026-11-19_14-35-12.png"}) {
		t.Errorf("Ordnerinhalt %v", n)
	}
}

func TestVorschauNieVergroessern(t *testing.T) {
	k := verkleinere(muster(40, 30), 256)
	if k.Bounds().Dx() != 40 || k.Bounds().Dy() != 30 {
		t.Errorf("Größe %v", k.Bounds())
	}
	// Mittelwert zweier Pixel (Schwarz, Weiß) beim Halbieren der Breite: Grau 127 oder 128 (Rundung).
	sw := image.NewNRGBA(image.Rect(0, 0, 4, 1))
	sw.SetNRGBA(1, 0, color.NRGBA{255, 255, 255, 255})
	sw.SetNRGBA(3, 0, color.NRGBA{255, 255, 255, 255})
	for x := 0; x < 4; x += 2 {
		sw.SetNRGBA(x, 0, color.NRGBA{0, 0, 0, 255})
	}
	k = verkleinere(sw, 2)
	if k.Bounds().Dx() != 2 || k.Bounds().Dy() != 1 {
		t.Fatalf("Größe %v", k.Bounds())
	}
	for x := 0; x < 2; x++ {
		if c := k.NRGBAAt(x, 0); c.R < 127 || c.R > 128 || c.G != c.R || c.B != c.R || c.A != 255 {
			t.Errorf("Pixel %d = %+v, soll Grau um 127", x, c)
		}
	}
}

func TestAblegenErstelltFehltNimmtUhr(t *testing.T) {
	b, _ := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{})
	if e.Pfad != "2026-11/2026-11-19_14-35-12.png" {
		t.Errorf("Pfad %q", e.Pfad)
	}
}

func TestAblegenKollisionInDerselbenSekunde(t *testing.T) {
	b, _ := neueBibliothek(t)
	m := fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")}
	var pfade []string
	for i := 0; i < 3; i++ {
		pfade = append(pfade, ablegen(t, b, m).Pfad)
	}
	want := []string{"2026-11/2026-11-19_14-35-12.png", "2026-11/2026-11-19_14-35-12-2.png", "2026-11/2026-11-19_14-35-12-3.png"}
	if !reflect.DeepEqual(pfade, want) {
		t.Errorf("Pfade %v, soll %v", pfade, want)
	}
	for _, p := range want { // keine überschrieben: alle da
		if _, err := os.Stat(filepath.Join(b.Orte.Ablage, filepath.FromSlash(p))); err != nil {
			t.Error(err)
		}
	}
	if g, _ := b.Laden(); len(g.Eintraege) != 3 {
		t.Errorf("Index hat %d Einträge", len(g.Eintraege))
	}
}

func TestAblegenAtomarKeineRestdatei(t *testing.T) {
	b, _ := neueBibliothek(t)
	ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")}) // Ordner existiert, mit einer Datei
	ordner := filepath.Join(b.Orte.Ablage, "2026-11")
	vorher := namen(t, ordner)

	// Fehlschlag im letzten Schritt (Umbenennen) auf dem Rückfallweg ohne Hardlinks (F-48).
	ersetzeVerknuepfe(t, func(string, string) error { return syscall.EPERM })
	ersetzeUmbenenne(t, func(string, string) error { return errors.New("Attrappe: Umbenennen scheitert") })
	d := neuesDokument(t, 40, 30)
	_, err := b.Ablegen(d, d.Original(), fileformat.Metadaten{Erstellt: zeit("2026-11-19T13:00:00Z")})
	if err == nil {
		t.Fatal("Fehler erwartet")
	}
	if n := namen(t, ordner); !reflect.DeepEqual(n, vorher) {
		t.Errorf("Ordner nach Fehlschlag %v, vorher %v", n, vorher)
	}
	if g, _ := b.Laden(); len(g.Eintraege) != 1 {
		t.Errorf("Index hat nach Fehlschlag %d Einträge, soll 1", len(g.Eintraege))
	}
}

func TestSchreibeAtomarSchreibfehler(t *testing.T) {
	ordner := t.TempDir()
	ziel := filepath.Join(ordner, "ziel.txt")
	schreibe(t, ziel, "alt")
	fremd := filepath.Join(ordner, "fremd.tmp") // gehört nicht zu uns und bleibt
	schreibe(t, fremd, "x")

	err := schreibeAtomar(ziel, func(w io.Writer) error {
		io.WriteString(w, "halb")
		return errors.New("Attrappe: Schreiben scheitert")
	})
	if err == nil {
		t.Fatal("Fehler erwartet")
	}
	if got, _ := os.ReadFile(ziel); string(got) != "alt" {
		t.Errorf("Ziel = %q, soll unverändert", got)
	}
	if n := namen(t, ordner); !reflect.DeepEqual(n, []string{"fremd.tmp", "ziel.txt"}) {
		t.Errorf("Ordnerinhalt %v", n)
	}
	// Gültiges Gegenstück: derselbe Aufruf ohne Fehler ersetzt das Ziel mit 0600.
	if err := schreibeAtomar(ziel, func(w io.Writer) error { _, err := io.WriteString(w, "neu"); return err }); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(ziel); string(got) != "neu" || modus(t, ziel) != 0o600 {
		t.Errorf("Ziel = %q (%o)", got, modus(t, ziel))
	}
}

func TestLadenToleriertKaputteZeilen(t *testing.T) {
	b, _ := neueBibliothek(t)
	gut1 := `{"pfad":"2026-11/a.png","erstellt":"2026-11-19T12:35:12Z","breite":40,"hoehe":30}`
	gut2 := `{"pfad":"2026-11/b.png","erstellt":"2026-11-19T12:36:12Z","breite":40,"hoehe":30,"schlagworte":["x"]}`
	zeilen := []string{
		gut1,
		`{"pfad":"2026-11/c.png","erstellt":`, // abgeschnitten
		`kein json`,                           // Müll
		`{"pfad":"../ausserhalb.png","erstellt":"2026-11-19T12:35:12Z","breite":1,"hoehe":1}`, // Pfad verlässt die Ablage
		`{"pfad":"/etc/x.png","erstellt":"2026-11-19T12:35:12Z","breite":1,"hoehe":1}`,        // absolut
		`{"pfad":"2026-11/d.txt","erstellt":"2026-11-19T12:35:12Z","breite":1,"hoehe":1}`,     // keine PNG
		`{"pfad":"2026-11/e.png","erstellt":"2026-11-19T12:35:12Z","breite":0,"hoehe":1}`,     // Größe fehlt
		``, // leere Zeile zählt nicht
		gut2,
	}
	schreibe(t, b.Orte.Index, strings.Join(zeilen, "\n")+"\n")
	g, err := b.Laden()
	if err != nil {
		t.Fatal(err)
	}
	if g.Defekt != 6 || len(g.Eintraege) != 2 || g.Eintraege[0].Pfad != "2026-11/a.png" || g.Eintraege[1].Schlagworte[0] != "x" {
		t.Errorf("Geladen = %+v", g)
	}
}

func TestLadenOhneIndex(t *testing.T) {
	b, _ := neueBibliothek(t)
	g, err := b.Laden()
	if err != nil || len(g.Eintraege) != 0 || g.Defekt != 0 {
		t.Errorf("Laden = %+v, %v", g, err)
	}
}

func TestAnhaengenNachAbgebrochenerZeile(t *testing.T) {
	b, _ := neueBibliothek(t)
	schreibe(t, b.Orte.Index, `{"pfad":"2026-11/a.png","erstell`) // Absturz mitten in der Zeile, ohne Umbruch
	ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")})
	g, _ := b.Laden()
	if g.Defekt != 1 || len(g.Eintraege) != 1 {
		t.Errorf("Geladen = %+v: die neue Zeile darf nicht an der kaputten kleben", g)
	}
}

func TestNeuaufbauAusDateien(t *testing.T) {
	b, _ := neueBibliothek(t)
	ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-10-02T08:00:00Z"), Quelle: "Fenster", Schlagworte: []string{"x"}})
	ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z"), Bearbeitet: true})
	ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")}) // -2
	vorher, _ := b.Laden()

	// Fremdes PNG und beschädigte Datei in der Ablage: übersprungen, nicht verändert.
	var fremd bytes.Buffer
	if err := png.Encode(&fremd, muster(10, 10)); err != nil {
		t.Fatal(err)
	}
	fremdPfad := filepath.Join(b.Orte.Ablage, "2026-11", "fremd.png")
	schreibe(t, fremdPfad, fremd.String())
	kaputtPfad := filepath.Join(b.Orte.Ablage, "2026-11", "kaputt.png")
	schreibe(t, kaputtPfad, fremd.String()[:20])
	schreibe(t, filepath.Join(b.Orte.Ablage, "2026-11", "notiz.txt"), "keine PNG")

	for name, zerstoere := range map[string]func(){
		"Index gelöscht":    func() { os.Remove(b.Orte.Index) },
		"Index zerstört":    func() { schreibe(t, b.Orte.Index, "Müll\n\x00\x01{{{\n") },
		"Index unverändert": func() {},
	} {
		zerstoere()
		erg, err := b.Neuaufbau()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if erg.Aufgenommen != 3 || !reflect.DeepEqual(erg.Uebersprungen, []string{"2026-11/fremd.png", "2026-11/kaputt.png"}) {
			t.Errorf("%s: Ergebnis %+v", name, erg)
		}
		nachher, err := b.Laden()
		if err != nil || nachher.Defekt != 0 {
			t.Fatalf("%s: %v %+v", name, err, nachher)
		}
		if !reflect.DeepEqual(nachher.Eintraege, sortiertKopie(vorher.Eintraege)) {
			t.Errorf("%s: Index nach Neuaufbau\n%+v\nsoll\n%+v", name, nachher.Eintraege, sortiertKopie(vorher.Eintraege))
		}
		if m := modus(t, b.Orte.Index); m != 0o600 {
			t.Errorf("%s: Index hat %o", name, m)
		}
	}
	for _, p := range []string{fremdPfad, kaputtPfad} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("übersprungene Datei angefasst: %v", err)
		}
	}
}

func sortiertKopie(es []Eintrag) []Eintrag {
	k := append([]Eintrag(nil), es...)
	sortiere(k)
	return k
}

func TestNeuaufbauOhneAblage(t *testing.T) {
	b, _ := neueBibliothek(t)
	erg, err := b.Neuaufbau()
	if err != nil || erg.Aufgenommen != 0 {
		t.Errorf("Neuaufbau = %+v, %v", erg, err)
	}
	if g, err := b.Laden(); err != nil || len(g.Eintraege) != 0 {
		t.Errorf("Laden = %+v, %v", g, err)
	}
}

func TestAbgleichLaesstFehlendeDateienFallen(t *testing.T) {
	b, _ := neueBibliothek(t)
	e1 := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")})
	e2 := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:13Z")})
	// Gegenstück: nichts fehlt → nichts entfernt.
	if n, err := b.Abgleich(); err != nil || n != 0 {
		t.Fatalf("Abgleich ohne Änderung: %d, %v", n, err)
	}
	// Der Nutzer löscht eine Aufnahme im Dateimanager (hier: im eigenen Testverzeichnis).
	if err := os.Remove(filepath.Join(b.Orte.Ablage, filepath.FromSlash(e1.Pfad))); err != nil {
		t.Fatal(err)
	}
	n, err := b.Abgleich()
	if err != nil || n != 1 {
		t.Fatalf("Abgleich: %d, %v", n, err)
	}
	g, _ := b.Laden()
	if len(g.Eintraege) != 1 || g.Eintraege[0].Pfad != e2.Pfad {
		t.Errorf("Index nach Abgleich %+v", g.Eintraege)
	}
}

func TestSuchen(t *testing.T) {
	b, _ := neueBibliothek(t)
	a := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-10-01T10:00:00Z"), Schlagworte: []string{"Rechnung", "Bank"}})
	c := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-01T10:00:00Z"), Schlagworte: []string{"bank"}})
	d := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-10T10:00:00Z")})

	pfade := func(a Anfrage) []string {
		es, err := b.Suchen(a)
		if err != nil {
			t.Fatal(err)
		}
		var p []string
		for _, e := range es {
			p = append(p, e.Pfad)
		}
		return p
	}
	faelle := []struct {
		name string
		a    Anfrage
		want []string
	}{
		{"alles, neueste zuerst", Anfrage{}, []string{d.Pfad, c.Pfad, a.Pfad}},
		{"Schlagwort ohne Groß/Klein", Anfrage{Schlagwort: "BANK"}, []string{c.Pfad, a.Pfad}},
		{"nur ganzes Schlagwort", Anfrage{Schlagwort: "ban"}, nil},
		{"anderes Schlagwort", Anfrage{Schlagwort: "rechnung"}, []string{a.Pfad}},
		{"Von inklusive", Anfrage{Von: zeit("2026-11-01T10:00:00Z")}, []string{d.Pfad, c.Pfad}},
		{"Bis exklusive", Anfrage{Bis: zeit("2026-11-01T10:00:00Z")}, []string{a.Pfad}},
		{"Zeitraum und Schlagwort", Anfrage{Schlagwort: "bank", Von: zeit("2026-10-15T00:00:00Z"), Bis: zeit("2026-12-01T00:00:00Z")}, []string{c.Pfad}},
	}
	for _, f := range faelle {
		if got := pfade(f.a); !reflect.DeepEqual(got, f.want) {
			t.Errorf("%s: %v, soll %v", f.name, got, f.want)
		}
	}
}

func TestSetzeSchlagworte(t *testing.T) {
	b, _ := neueBibliothek(t)
	d := neuesDokument(t, 40, 30)
	if _, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(5, 5), document.Pt(20, 15)), Strich: document.Strich{Farbe: color.NRGBA{255, 0, 0, 255}, Breite: 2}}); err != nil {
		t.Fatal(err)
	}
	m := fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z"), Quelle: "Fenster", Autor: "Tester"}
	e, err := b.Ablegen(d, d.Original(), m)
	if err != nil {
		t.Fatal(err)
	}
	pfad := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))

	neu, err := b.SetzeSchlagworte(e.Pfad, []string{"Wichtig", "wichtig", "Bank"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(neu.Schlagworte, []string{"Wichtig", "Bank"}) {
		t.Errorf("Schlagworte %v", neu.Schlagworte)
	}
	// Datei: Schlagworte, sonst alles wie vorher.
	f, _ := os.Open(pfad)
	defer f.Close()
	g, err := fileformat.LeseArbeitsdatei(f)
	if err != nil || g.Fremd {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.Meta.Schlagworte, []string{"Wichtig", "Bank"}) || g.Meta.Quelle != "Fenster" || g.Meta.Autor != "Tester" ||
		!g.Meta.Erstellt.Equal(m.Erstellt) || g.Meta.Bearbeitet {
		t.Errorf("Meta %+v", g.Meta)
	}
	if !bytes.Equal(g.Dokument.Original().Pix, d.Original().Pix) || len(g.Dokument.Objekte()) != 1 {
		t.Error("Original oder Objekte haben sich verändert")
	}
	if modus(t, pfad) != 0o600 {
		t.Errorf("Rechte %o", modus(t, pfad))
	}
	// Index folgt, Suche findet.
	if es, _ := b.Suchen(Anfrage{Schlagwort: "bank"}); len(es) != 1 || es[0].Pfad != e.Pfad {
		t.Errorf("Suche %+v", es)
	}
	if gl, _ := b.Laden(); len(gl.Eintraege) != 1 {
		t.Errorf("Index %+v", gl.Eintraege)
	}
	// Leeren geht auch.
	if neu, err := b.SetzeSchlagworte(e.Pfad, nil); err != nil || len(neu.Schlagworte) != 0 {
		t.Errorf("Leeren: %+v %v", neu, err)
	}
}

func TestSetzeSchlagworteLehntFremdesAb(t *testing.T) {
	b, _ := neueBibliothek(t)
	var fremd bytes.Buffer
	png.Encode(&fremd, muster(10, 10))
	schreibe(t, filepath.Join(b.Orte.Ablage, "2026-11", "fremd.png"), fremd.String())
	if _, err := b.SetzeSchlagworte("2026-11/fremd.png", []string{"x"}); !errors.Is(err, ErrFremd) {
		t.Errorf("Fremd: %v", err)
	}
	if got, _ := os.ReadFile(filepath.Join(b.Orte.Ablage, "2026-11", "fremd.png")); !bytes.Equal(got, fremd.Bytes()) {
		t.Error("fremde Datei verändert")
	}
	for _, p := range []string{"../x.png", "/etc/x.png", "", "2026-11/x.txt"} {
		if _, err := b.SetzeSchlagworte(p, nil); !errors.Is(err, ErrPfad) {
			t.Errorf("Pfad %q: %v", p, err)
		}
	}
}
