// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// ADR-0046 (Karte #341): Datei → „Save As…“ schreibt eine Kopie als Arbeitsdatei an den gewählten Ort, das Fenster
// bleibt bei der Aufnahme in der Bibliothek; „New capture“ startet einen eigenen Prozess.

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gioui.org/io/key"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/portal"
)

// mitUnterDialog ersetzt den Speichern-Dialog für einen Test; die Anfragen werden gesammelt.
func mitUnterDialog(t *testing.T, erg portal.Ergebnis, err error) *[]portal.Anfrage {
	t.Helper()
	var anfragen []portal.Anfrage
	alt := unterDialog
	unterDialog = func(_ context.Context, a portal.Anfrage) (portal.Ergebnis, error) {
		anfragen = append(anfragen, a)
		return erg, err
	}
	t.Cleanup(func() { unterDialog = alt })
	return &anfragen
}

// kastenDazu legt einen Kasten ins Dokument (der Stand ist dann ungesichert).
func kastenDazu(t *testing.T, d *document.Dokument) {
	t.Helper()
	if _, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(2, 2), document.Pt(12, 9)),
		Strich: document.Strich{Farbe: color.NRGBA{R: 200, A: 255}, Breite: 2}}); err != nil {
		t.Fatal(err)
	}
}

// unterFertig wartet, bis Dialog und Schreiben vorbei sind.
func (l *leistenLauf) unterFertig() {
	l.t.Helper()
	l.warte(func() bool { return !l.e.unter.offen && !l.e.unter.laeuft })
}

func liesArbeitsdatei(t *testing.T, pfad string) *fileformat.Gelesen {
	t.Helper()
	f, err := os.Open(pfad)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	g, err := fileformat.LeseArbeitsdatei(f)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestSpeichernUnterSchreibtKopieUndBleibtBeiDerAufnahme(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.e.titel = "2026-11-24_09-00-00.png"
	kastenDazu(t, l.e.s.D)
	n := len(l.e.s.D.Objekte())
	quelle := l.e.pfad()
	vorher, err := os.ReadFile(quelle)
	if err != nil {
		t.Fatal(err)
	}
	ordner := t.TempDir()
	anfragen := mitUnterDialog(t, portal.Ergebnis{URI: "file://" + filepath.Join(ordner, "kopie")}, nil)
	l.taste("S", key.ModShortcut|key.ModShift)
	l.unterFertig()
	if len(*anfragen) != 1 {
		t.Fatalf("%d Dialoge", len(*anfragen))
	}
	a := (*anfragen)[0]
	if a.Dateiname != "2026-11-24_09-00-00.png" || len(a.Filter) != 1 || a.Filter[0].Globs[0] != "*.png" {
		t.Errorf("Anfrage %+v", a)
	}
	ziel := filepath.Join(ordner, "kopie.png") // Endung angehängt (Punkt 3)
	g := liesArbeitsdatei(t, ziel)
	if g.Fremd || len(g.Dokument.Objekte()) != n {
		t.Fatalf("Kopie: fremd %v, %d Objekte", g.Fremd, len(g.Dokument.Objekte()))
	}
	// Metadaten aus der Aufnahme (Punkt 2): ihre Aufnahmezeit, bearbeitet.
	if !g.Meta.Bearbeitet || !g.Meta.Erstellt.Equal(time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)) {
		t.Errorf("Metadaten %+v", g.Meta)
	}
	// Das Fenster bleibt bei der Aufnahme (Punkt 1): Titel, Pfad, ungesicherter Stand, Datei in der Bibliothek.
	if l.e.titel != "2026-11-24_09-00-00.png" || l.e.pfad() != quelle || !l.e.s.Geaendert() {
		t.Errorf("Fenster: Titel %q, Pfad %q, geändert %v", l.e.titel, l.e.pfad(), l.e.s.Geaendert())
	}
	if nachher, _ := os.ReadFile(quelle); !bytes.Equal(vorher, nachher) {
		t.Error("die Aufnahme in der Bibliothek hat sich geändert")
	}
	if l.e.meldung.Fehler || !strings.Contains(l.e.meldung.Text, "kopie.png") ||
		!strings.Contains(l.e.meldung.Text, "keeps editing") || strings.Contains(l.e.meldung.Text, "whole original") {
		t.Errorf("Meldung %q", l.e.meldung.Text)
	}
	// Die Kopie trägt das ganze Original: nur für den Eigentümer lesbar wie die Bibliothek (Kritik K-1).
	if fi, err := os.Stat(ziel); err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("Rechte der Kopie: %v %v", fi.Mode().Perm(), err)
	}
}

func mustLesen(t *testing.T, pfad string) []byte {
	t.Helper()
	b, err := os.ReadFile(pfad)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// Punkt 5: Mit Zuschnitt sagt die Fußzeile, dass die Kopie das ganze Original trägt; die Kopie hat den Zuschnitt.
func TestSpeichernUnterMitZuschnittWarnt(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 2)
	z := image.Rect(5, 5, 30, 20)
	if err := l.e.s.D.SetzeZuschnitt(z); err != nil {
		t.Fatal(err)
	}
	ziel := filepath.Join(t.TempDir(), "zugeschnitten.png")
	mitUnterDialog(t, portal.Ergebnis{URI: "file://" + ziel}, nil)
	l.taste("S", key.ModShortcut|key.ModShift)
	l.unterFertig()
	if !strings.Contains(l.e.meldung.Text, "whole original") || l.e.meldung.Fehler {
		t.Errorf("Meldung %q", l.e.meldung.Text)
	}
	if g := liesArbeitsdatei(t, ziel); g.Dokument.Zuschnitt() != z || g.Dokument.OriginalBild().Rect == z {
		t.Errorf("Kopie: Zuschnitt %v, Original %v", g.Dokument.Zuschnitt(), g.Dokument.OriginalBild().Rect)
	}
}

// Punkt 4: Die offene Aufnahme selbst wird nicht an der Speicherfunktion vorbei überschrieben.
func TestSpeichernUnterLehntDieOffeneAb(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 2)
	kastenDazu(t, l.e.s.D)
	quelle := l.e.pfad()
	vorher := mustLesen(t, quelle)
	geschrieben := false
	alt := unterSchreiben
	unterSchreiben = func(string, bool, func(io.Writer) error) error { geschrieben = true; return nil }
	t.Cleanup(func() { unterSchreiben = alt })
	mitUnterDialog(t, portal.Ergebnis{URI: "file://" + quelle}, nil)
	l.taste("S", key.ModShortcut|key.ModShift)
	l.unterFertig()
	if geschrieben || !bytes.Equal(vorher, mustLesen(t, quelle)) {
		t.Error("die offene Aufnahme wurde überschrieben")
	}
	if !l.e.meldung.Fehler || !strings.Contains(l.e.meldung.Text, "open in this window") {
		t.Errorf("Meldung %q", l.e.meldung.Text)
	}
	// Auch über einen harten Verweis (anderer Name, dieselbe Datei).
	verweis := filepath.Join(t.TempDir(), "verweis.png")
	if err := os.Link(quelle, verweis); err != nil {
		t.Skip("kein harter Verweis möglich:", err)
	}
	mitUnterDialog(t, portal.Ergebnis{URI: "file://" + verweis}, nil)
	l.taste("S", key.ModShortcut|key.ModShift)
	l.unterFertig()
	if geschrieben {
		t.Error("über den harten Verweis geschrieben")
	}
}

// Punkt 3: Eine angehängte Endung ersetzt keine vorhandene Datei; eine bestätigte schon.
func TestSpeichernUnterEndungErsetztNichtStill(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 2)
	ordner := t.TempDir()
	vorhanden := filepath.Join(ordner, "da.png")
	if err := os.WriteFile(vorhanden, []byte("alt"), 0o600); err != nil {
		t.Fatal(err)
	}
	mitUnterDialog(t, portal.Ergebnis{URI: "file://" + filepath.Join(ordner, "da")}, nil)
	l.taste("S", key.ModShortcut|key.ModShift)
	l.unterFertig()
	if string(mustLesen(t, vorhanden)) != "alt" || !l.e.meldung.Fehler || !strings.Contains(l.e.meldung.Text, "already exists") {
		t.Errorf("angehängt: Inhalt %q, Meldung %q", mustLesen(t, vorhanden), l.e.meldung.Text)
	}
	mitUnterDialog(t, portal.Ergebnis{URI: "file://" + vorhanden}, nil)
	l.taste("S", key.ModShortcut|key.ModShift)
	l.unterFertig()
	if g := liesArbeitsdatei(t, vorhanden); g.Fremd || l.e.meldung.Fehler {
		t.Errorf("bestätigt: fremd %v, Meldung %q", g.Fremd, l.e.meldung.Text)
	}
}

func TestSpeichernUnterAbbruchFehlerUndSchreibfehler(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 2)
	mitUnterDialog(t, portal.Ergebnis{}, portal.ErrAbgebrochen)
	l.taste("S", key.ModShortcut|key.ModShift)
	l.unterFertig()
	if l.e.meldung.Fehler || l.e.meldung.Text != "" {
		t.Errorf("Abbruch: %q", l.e.meldung.Text)
	}
	mitUnterDialog(t, portal.Ergebnis{}, portal.ErrKeinPortal)
	l.taste("S", key.ModShortcut|key.ModShift)
	l.unterFertig()
	if !l.e.meldung.Fehler || !strings.Contains(l.e.meldung.Text, "portal") {
		t.Errorf("ohne Portal: %q", l.e.meldung.Text)
	}
	mitUnterDialog(t, portal.Ergebnis{URI: "https://example.org/x.png"}, nil)
	l.taste("S", key.ModShortcut|key.ModShift)
	l.unterFertig()
	if !l.e.meldung.Fehler || !strings.Contains(l.e.meldung.Text, "local") {
		t.Errorf("entfernte Adresse: %q", l.e.meldung.Text)
	}
	alt := unterSchreiben
	unterSchreiben = func(string, bool, func(io.Writer) error) error { return errors.New("export: Platte voll") }
	t.Cleanup(func() { unterSchreiben = alt })
	mitUnterDialog(t, portal.Ergebnis{URI: "file://" + filepath.Join(t.TempDir(), "x.png")}, nil)
	l.taste("S", key.ModShortcut|key.ModShift)
	l.unterFertig()
	if !l.e.meldung.Fehler || !strings.Contains(l.e.meldung.Text, "Could not save a copy") {
		t.Errorf("Schreibfehler: %q", l.e.meldung.Text)
	}
}

// Punkt 6: Während des Schreibens geht das Bearbeiten weiter; die Kopie trägt den Stand bei der Wahl. Ein zweites
// Speichern unter und der Export warten, das Schließen wartet höchstens die Frist.
func TestSpeichernUnterImHintergrund(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 2)
	kastenDazu(t, l.e.s.D)
	n0 := len(l.e.s.D.Objekte())
	tor := make(chan struct{})
	alt := unterSchreiben
	unterSchreiben = func(p string, ohne bool, s func(io.Writer) error) error {
		<-tor
		return alt(p, ohne, s)
	}
	t.Cleanup(func() { unterSchreiben = alt })
	ziel := filepath.Join(t.TempDir(), "hinten.png")
	anfragen := mitUnterDialog(t, portal.Ergebnis{URI: "file://" + ziel}, nil)
	l.taste("S", key.ModShortcut|key.ModShift)
	l.warte(func() bool { return !l.e.unter.offen && l.e.unter.laeuft })
	kastenDazu(t, l.e.s.D) // Bearbeiten läuft weiter
	l.taste("S", key.ModShortcut|key.ModShift)
	l.bild()
	if len(*anfragen) != 1 || eintragVon(l.e.menueEintraegeRoh(menueDatei), aktSpeichernUnter).an {
		t.Errorf("zweites Speichern unter während des Schreibens: %d Dialoge", len(*anfragen))
	}
	close(tor)
	l.unterFertig()
	if n := len(liesArbeitsdatei(t, ziel).Dokument.Objekte()); n != n0 {
		t.Errorf("Kopie hat %d Objekte, Soll %d (Stand bei der Wahl)", n, n0)
	}
	if len(l.e.s.D.Objekte()) != n0+1 {
		t.Errorf("Dokument hat %d Objekte", len(l.e.s.D.Objekte()))
	}
}

func TestSpeichernUnterAbwartenBeimSchliessen(t *testing.T) {
	var aus bytes.Buffer
	altAus := fehlerAusgabe
	fehlerAusgabe = &aus
	t.Cleanup(func() { fehlerAusgabe = altAus })
	e := testEditor(t)
	e.unter.laeuft, e.unter.antwort = true, make(chan unterErgebnis, 1)
	e.speichernUnterAbwarten(20 * time.Millisecond)
	if !strings.Contains(aus.String(), "not finished") {
		t.Errorf("Frist abgelaufen: %q", aus.String())
	}
	// Je Fall ein eigener Kanal: Holt das Abwarten die Antwort nicht ab, blockiert das Senden nicht (Mutant SU14).
	for _, fall := range []struct {
		r    unterErgebnis
		soll string
	}{{unterErgebnis{err: errors.New("export: kaputt")}, "Could not save a copy"}, {unterErgebnis{name: "gut.png"}, ""}} {
		aus.Reset()
		e.unter.laeuft, e.unter.antwort = true, make(chan unterErgebnis, 1)
		e.unter.antwort <- fall.r
		e.speichernUnterAbwarten(time.Second)
		if e.unter.laeuft || len(e.unter.antwort) != 0 {
			t.Errorf("%+v: Antwort nicht abgeholt", fall.r)
		}
		if got := aus.String(); fall.soll == "" && got != "" || fall.soll != "" && !strings.Contains(got, fall.soll) {
			t.Errorf("%+v: Ausgabe %q", fall.r, got)
		}
	}
}

// Ohne lesbare Arbeitsdatei als Quelle (Datei von außen, kein Pfad): ohne Autor und Schlagworte, Zeit des Schreibens.
func TestKopieMetadatenOhneArbeitsdatei(t *testing.T) {
	fremd := filepath.Join(t.TempDir(), "fremd.png")
	if err := os.WriteFile(fremd, []byte("kein png"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{"", fremd, filepath.Join(t.TempDir(), "fehlt.png")} {
		m := kopieMetadaten(q)
		if !m.Bearbeitet || m.Autor != "" || len(m.Schlagworte) != 0 || time.Since(m.Erstellt) > time.Minute {
			t.Errorf("%q: %+v", q, m)
		}
	}
}

// --- Neue Aufnahme (Punkt 7) ---

// mitNeuerAufnahme ersetzt Programm und Start; die gestarteten Programme werden gesammelt.
func mitNeuerAufnahme(t *testing.T, startFehler error) *[]string {
	t.Helper()
	var gestartet []string
	altP, altS := neueAufnahmeProgramm, neueAufnahmeStart
	neueAufnahmeProgramm = func() (string, error) { return "/pfad/zu/schnapp", nil }
	neueAufnahmeStart = func(p string) error { gestartet = append(gestartet, p); return startFehler }
	t.Cleanup(func() { neueAufnahmeProgramm, neueAufnahmeStart = altP, altS })
	return &gestartet
}

func TestNeueAufnahmeStartetEigenenProzess(t *testing.T) {
	t.Setenv("WAYLAND_SOCKET", "")
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 2)
	kastenDazu(t, l.e.s.D)
	gestartet := mitNeuerAufnahme(t, nil)
	titel := l.e.titel
	l.taste("N", key.ModShortcut)
	l.bild()
	if len(*gestartet) != 1 || (*gestartet)[0] != "/pfad/zu/schnapp" {
		t.Fatalf("gestartet %v", *gestartet)
	}
	// Das Fenster bleibt, wie es ist.
	if l.e.titel != titel || !l.e.s.Geaendert() || l.e.meldung.Fehler {
		t.Errorf("Fenster verändert: Titel %q, geändert %v, Meldung %q", l.e.titel, l.e.s.Geaendert(), l.e.meldung.Text)
	}
	// N allein bleibt das Werkzeug Schritt-Nummer.
	l.taste("N", 0)
	l.bild()
	if len(*gestartet) != 1 {
		t.Errorf("N allein startete eine Aufnahme")
	}
	// Bei offener Bibliotheksansicht geht es auch.
	l.taste("L", key.ModShortcut|key.ModShift)
	l.bild()
	if !l.e.bibl.offen {
		t.Fatal("Ansicht nicht offen")
	}
	l.e.neuGestartet = time.Time{} // Sperre gegen Mehrfachstart (Kritik K-2) hier nicht prüfen
	l.taste("N", key.ModShortcut)
	l.bild()
	if len(*gestartet) != 2 {
		t.Errorf("bei offener Ansicht: %d Starts", len(*gestartet))
	}
}

func TestNeueAufnahmeFehlerUndGeerbterSocket(t *testing.T) {
	t.Setenv("WAYLAND_SOCKET", "")
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 2)
	mitNeuerAufnahme(t, errors.New("kein exec"))
	l.taste("N", key.ModShortcut)
	l.bild()
	if !l.e.meldung.Fehler || !strings.Contains(l.e.meldung.Text, "new capture") {
		t.Errorf("Startfehler: %q", l.e.meldung.Text)
	}
	gestartet := mitNeuerAufnahme(t, nil)
	t.Setenv("WAYLAND_SOCKET", "3")
	en := eintragVon(l.e.menueEintraegeRoh(menueDatei), aktNeueAufnahme)
	if en.an || en.erklaerung == "" {
		t.Errorf("mit WAYLAND_SOCKET: %+v", en)
	}
	l.taste("N", key.ModShortcut)
	l.bild()
	if len(*gestartet) != 0 {
		t.Errorf("mit WAYLAND_SOCKET gestartet")
	}
}

// eintragVon sucht den Eintrag mit der Handlung a.
func eintragVon(en []menueEintrag, a aktion) menueEintrag {
	for _, x := range en {
		if x.akt == a {
			return x
		}
	}
	return menueEintrag{}
}

// Punkt 8: Reihenfolge im Menü Datei.
func TestMenueDateiReihenfolge(t *testing.T) {
	t.Setenv("WAYLAND_SOCKET", "")
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 2)
	var akt []aktion
	for _, en := range l.e.menueEintraegeRoh(menueDatei) {
		akt = append(akt, en.akt)
	}
	soll := []aktion{aktOeffnen, aktNeueAufnahme, aktNeuAusZwischenablage, aktSpeichern, aktSpeichernUnter, aktExport,
		aktBildKopieren, aktEinstellungen, aktBibliothek, aktSchliessen}
	if len(akt) != len(soll) {
		t.Fatalf("Datei: %v", akt)
	}
	for i := range soll {
		if akt[i] != soll[i] {
			t.Fatalf("Datei: %v, Soll %v", akt, soll)
		}
	}
}

// Punkt 6: Ist beim Abholen eines Wechsels der Dialog von Speichern unter offen, bleibt die offene Aufnahme (wie beim
// Export, F-652); die Wahl gälte sonst einer anderen Aufnahme als der, für die der Dialog aufging.
func TestWechselSpeichernUnterOffenBeimAbholen(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	altD := l.e.s.D
	l.ladeAngehalten(0)
	l.e.unter.offen = true
	t.Cleanup(func() { l.e.unter.offen = false })
	close(l.tor)
	l.ruhig()
	if l.e.s.D != altD {
		t.Fatal("bei offenem Dialog von Speichern unter gewechselt")
	}
	if r := l.rett[l.pfade[0]]; r == nil || r.ende != 1 {
		t.Errorf("Rettung der geladenen: %+v", r)
	}
	if !strings.Contains(l.e.meldung.Text, "Save As dialog") {
		t.Errorf("Meldung %q", l.e.meldung.Text) // V-3 m4
	}
}

// Punkt 2 (Kritik K-8, K-9): Autor, Aufnahmezeit und Schlagworte aus der Quelle, das Quellprogramm nicht.
func TestKopieMetadatenAusQuelle(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 8, 6)))
	if err != nil {
		t.Fatal(err)
	}
	quelle := filepath.Join(t.TempDir(), "quelle.png")
	zeit := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	var b bytes.Buffer
	if err := fileformat.SchreibeArbeitsdatei(&b, d, d.Original(), fileformat.Metadaten{Autor: "Jo", Erstellt: zeit,
		Quelle: "browser", Schlagworte: []string{"rechnung"}}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(quelle, b.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	m := kopieMetadaten(quelle)
	if m.Autor != "Jo" || !m.Erstellt.Equal(zeit) || len(m.Schlagworte) != 1 || m.Schlagworte[0] != "rechnung" ||
		m.Quelle != "" || !m.Bearbeitet {
		t.Errorf("Metadaten %+v", m)
	}
}

// Punkt 4 (Kritik K-11): kein Ziel in der Ablage der Bibliothek, auch nicht unter neuem Namen.
func TestSpeichernUnterNichtInDieBibliothek(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 2)
	geschrieben := false
	alt := unterSchreiben
	unterSchreiben = func(string, bool, func(io.Writer) error) error { geschrieben = true; return nil }
	t.Cleanup(func() { unterSchreiben = alt })
	ziel := filepath.Join(filepath.Dir(l.e.pfad()), "neu-in-der-ablage.png")
	mitUnterDialog(t, portal.Ergebnis{URI: "file://" + ziel}, nil)
	l.taste("S", key.ModShortcut|key.ModShift)
	l.unterFertig()
	if geschrieben || !l.e.meldung.Fehler || !strings.Contains(l.e.meldung.Text, "outside the library") {
		t.Errorf("geschrieben %v, Meldung %q", geschrieben, l.e.meldung.Text)
	}
	if inOrdner(l.b.Orte.Ablage, filepath.Join(t.TempDir(), "x.png")) || !inOrdner(l.b.Orte.Ablage, ziel) {
		t.Error("inOrdner")
	}
}

// Punkt 3: Eine großgeschriebene Endung gilt, es wird nichts angehängt.
func TestSpeichernUnterGrosseEndung(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 2)
	ziel := filepath.Join(t.TempDir(), "GROSS.PNG")
	mitUnterDialog(t, portal.Ergebnis{URI: "file://" + ziel}, nil)
	l.taste("S", key.ModShortcut|key.ModShift)
	l.unterFertig()
	if g := liesArbeitsdatei(t, ziel); g.Fremd || l.e.meldung.Fehler {
		t.Errorf("fremd %v, Meldung %q", g.Fremd, l.e.meldung.Text)
	}
	// Bestätigte Endung (der Regelfall, SchreibeAtomarPrivat): auch 0600 (V-3).
	if fi, err := os.Stat(ziel); err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("Rechte %v %v", fi.Mode().Perm(), err)
	}
}

// Punkt 6: Die Dialoge sperren sich gegenseitig; „Saving a copy…“ verdrängt keine haltende Meldung (Kritik K-4); ein
// Wechsel wartet auf das Schreiben (Kritik K-5).
func TestSpeichernUnterSperren(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	anfragen := mitUnterDialog(t, portal.Ergebnis{}, portal.ErrAbgebrochen)
	l.e.exportOffen = true
	l.taste("S", key.ModShortcut|key.ModShift)
	l.e.exportOffen = false
	l.e.oeffnenOffen = true
	l.taste("S", key.ModShortcut|key.ModShift)
	l.e.oeffnenOffen = false
	if len(*anfragen) != 0 || l.e.unter.offen {
		t.Fatalf("Dialog neben Export/Öffnen: %d", len(*anfragen))
	}
	exportAnfragen := 0
	altExport := exportDialog
	exportDialog = func(context.Context, portal.Anfrage) (portal.Ergebnis, error) {
		exportAnfragen++
		return portal.Ergebnis{}, portal.ErrAbgebrochen
	}
	t.Cleanup(func() { exportDialog = altExport })
	oeffnenAnfragen := mitOeffnenDialog(t, portal.Ergebnis{}, portal.ErrAbgebrochen)
	l.e.unter.offen = true
	for _, a := range []aktion{aktExport, aktOeffnen} {
		if eintragVon(l.e.menueEintraegeRoh(menueDatei), a).an {
			t.Errorf("%v bei offenem Dialog an", a)
		}
	}
	l.taste("E", key.ModShortcut|key.ModShift)
	l.taste("O", key.ModShortcut)
	l.bild()
	if l.e.exportOffen || l.e.oeffnenOffen || exportAnfragen != 0 || len(*oeffnenAnfragen) != 0 {
		t.Error("Export- oder Öffnen-Dialog neben Speichern unter")
	}
	l.e.unter.offen = false
	l.warte(func() bool { return !l.e.exportOffen && !l.e.oeffnenOffen })

	tor := make(chan struct{})
	alt := unterSchreiben
	unterSchreiben = func(p string, ohne bool, s func(io.Writer) error) error { <-tor; return alt(p, ohne, s) }
	t.Cleanup(func() { unterSchreiben = alt })
	mitUnterDialog(t, portal.Ergebnis{URI: "file://" + filepath.Join(t.TempDir(), "k.png")}, nil)
	l.e.meldeHaltend(haltVerwerfen, "Verwerfen-Warnung", true)
	l.taste("S", key.ModShortcut|key.ModShift)
	l.warte(func() bool { return !l.e.unter.offen && l.e.unter.laeuft })
	if l.e.meldung.Text != "Verwerfen-Warnung" || l.e.meldungHaelt != haltVerwerfen {
		t.Errorf("haltende Meldung verdrängt: %q", l.e.meldung.Text)
	}
	if l.e.wechselAnfrage(l.pfade[0]) || len(l.aufrufe()) != 0 {
		t.Error("Wechsel während des Schreibens")
	}
	if l.e.meldung.Text != "Verwerfen-Warnung" { // die Sperrmeldung geht nebenbei (V-3 m8)
		t.Errorf("Sperrmeldung verdrängt die haltende: %q", l.e.meldung.Text)
	}
	close(tor)
	l.unterFertig()
}

// Kritik K-2: Gehaltenes Strg+N (Tastenwiederholung) oder ein Doppelklick startet nur eine Aufnahme.
func TestNeueAufnahmeNurEinmal(t *testing.T) {
	t.Setenv("WAYLAND_SOCKET", "")
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 2)
	gestartet := mitNeuerAufnahme(t, nil)
	for range 3 {
		l.taste("N", key.ModShortcut)
	}
	if len(*gestartet) != 1 {
		t.Fatalf("%d Starts", len(*gestartet))
	}
	l.e.neuGestartet = time.Now().Add(-neueAufnahmeSperre)
	l.taste("N", key.ModShortcut)
	if len(*gestartet) != 2 {
		t.Errorf("nach der Sperre: %d Starts", len(*gestartet))
	}
}

// Der echte Start ohne Fenster: das Programm bekommt das interne Argument; ein fehlendes scheitert (Setsid prüft der
// Test nicht, das bleibt der Echtprobe).
func TestNeueAufnahmeStartEcht(t *testing.T) {
	ordner := t.TempDir()
	marke := filepath.Join(ordner, "argument")
	skript := filepath.Join(ordner, "prog")
	if err := os.WriteFile(skript, []byte("#!/bin/sh\nprintf %s \"$1\" > "+marke+"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := neueAufnahmeStart(skript); err != nil {
		t.Fatal(err)
	}
	ende := time.Now().Add(5 * time.Second)
	for {
		if b, err := os.ReadFile(marke); err == nil && string(b) == NeueAufnahmeArgument {
			break
		}
		if time.Now().After(ende) {
			t.Fatal("Programm nicht mit dem Argument gestartet")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := neueAufnahmeStart(filepath.Join(ordner, "fehlt")); err == nil {
		t.Error("fehlendes Programm ohne Fehler")
	}
}

// V-1: Ein Verweis von außen auf eine andere Aufnahme der Ablage ist kein Ziel; die Aufnahme bleibt unverändert.
func TestSpeichernUnterSymlinkInDieAblage(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	kastenDazu(t, l.e.s.D)
	andere := l.pfade[2] // offen ist pfade[1]
	vorher := mustLesen(t, andere)
	link := filepath.Join(t.TempDir(), "link.png")
	if err := os.Symlink(andere, link); err != nil {
		t.Skip(err)
	}
	mitUnterDialog(t, portal.Ergebnis{URI: "file://" + link}, nil)
	l.taste("S", key.ModShortcut|key.ModShift)
	l.unterFertig()
	if !bytes.Equal(vorher, mustLesen(t, andere)) || !strings.Contains(l.e.meldung.Text, "outside the library") {
		t.Errorf("über den Verweis geschrieben oder falsche Meldung: %q", l.e.meldung.Text)
	}
}

// inOrdner mit Verweisen (Verify #341).
func TestInOrdnerVerweise(t *testing.T) {
	wurzel := t.TempDir()
	ablage := filepath.Join(wurzel, "ablage")
	monat := filepath.Join(ablage, "2026-11")
	aussen := filepath.Join(wurzel, "aussen")
	for _, o := range []string{monat, aussen, filepath.Join(ablage, "..x")} {
		if err := os.MkdirAll(o, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	hinein, ablageLink, hinaus := filepath.Join(aussen, "hinein"), filepath.Join(wurzel, "ablage-link"), filepath.Join(ablage, "hinaus")
	for _, v := range [][2]string{{monat, hinein}, {ablage, ablageLink}, {aussen, hinaus}} {
		if err := os.Symlink(v[0], v[1]); err != nil {
			t.Skip(err)
		}
	}
	for _, f := range []struct {
		name, ordner, pfad string
		soll               bool
	}{
		{"direkt", ablage, filepath.Join(ablage, "x.png"), true},
		{"monat", ablage, filepath.Join(monat, "x.png"), true},
		{"außen", ablage, filepath.Join(aussen, "x.png"), false},
		{"Ordnerverweis hinein", ablage, filepath.Join(hinein, "x.png"), true},
		{"Ablage als Verweis", ablageLink, filepath.Join(monat, "x.png"), true},
		{"Verweis hinaus", ablage, filepath.Join(hinaus, "x.png"), false},
		{"Präfix-Nachbar", ablage, filepath.Join(wurzel, "ablage2", "x.png"), false},
		{"Ordner ..x", ablage, filepath.Join(ablage, "..x", "y.png"), true},
	} {
		if g := inOrdner(f.ordner, f.pfad); g != f.soll {
			t.Errorf("%s: %v, Soll %v", f.name, g, f.soll)
		}
	}
}

// F-1008: Nach einem gescheiterten Start greift die Sperre nicht (Verify V-3 m5).
func TestNeueAufnahmeSperreNichtNachFehler(t *testing.T) {
	t.Setenv("WAYLAND_SOCKET", "")
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 2)
	mitNeuerAufnahme(t, errors.New("kein exec"))
	l.taste("N", key.ModShortcut)
	if !l.e.neuGestartet.IsZero() {
		t.Errorf("Sperre nach Fehlstart gesetzt")
	}
	gestartet := mitNeuerAufnahme(t, nil)
	l.taste("N", key.ModShortcut)
	if len(*gestartet) != 1 {
		t.Errorf("nach Fehlstart kein neuer Versuch: %d", len(*gestartet))
	}
}

// F-1009/F-1016: zerstoert wartet auf die Kopie, und zwar nach dem Notspeichern (Verify V-3 m1).
func TestZerstoertWartetNachNotspeichernAufKopie(t *testing.T) {
	var aus bytes.Buffer
	altAus, altFrist := fehlerAusgabe, unterFrist
	fehlerAusgabe, unterFrist = &aus, 300*time.Millisecond
	t.Cleanup(func() { fehlerAusgabe, unterFrist = altAus, altFrist })
	e := testEditor(t)
	n := 0
	e.unter.laeuft, e.unter.antwort = true, make(chan unterErgebnis, 1)
	e.speichern = func(*document.Dokument) (Gespeichert, error) {
		n++
		e.unter.antwort <- unterErgebnis{name: "x.png"} // die Kopie wird erst während des Notspeicherns fertig
		return Gespeichert{}, nil
	}
	kastenDazu(t, e.s.D)
	_ = e.zerstoert(nil)
	if n != 1 || e.unter.laeuft || aus.Len() != 0 {
		t.Errorf("Notspeichern %d, läuft %v, Ausgabe %q", n, e.unter.laeuft, aus.String())
	}
}

// V-2: Ein weiteres Signal beendet das Warten sofort; die Frist zählt ab Beginn des Schreibens.
func TestSpeichernUnterAbwartenSignalUndFrist(t *testing.T) {
	var aus bytes.Buffer
	altAus := fehlerAusgabe
	fehlerAusgabe = &aus
	t.Cleanup(func() { fehlerAusgabe = altAus })
	e := testEditor(t)
	e.weitereSignale = make(chan struct{}, 1)
	e.weitereSignale <- struct{}{}
	e.unter.laeuft, e.unter.antwort = true, make(chan unterErgebnis, 1)
	start := time.Now()
	e.speichernUnterAbwarten(3 * time.Second) // ohne Signal-Ausweg dauerte es die ganze Frist (Mutant SU29)
	if time.Since(start) > 2*time.Second || !strings.Contains(aus.String(), "not finished") {
		t.Errorf("Signal: %v, %q", time.Since(start), aus.String())
	}
	aus.Reset()
	e.unter.start = time.Now().Add(-time.Hour) // längst über der Frist
	start = time.Now()
	e.speichernUnterAbwarten(time.Minute)
	if time.Since(start) > 2*time.Second || !strings.Contains(aus.String(), "not finished") {
		t.Errorf("Frist ab Beginn: %v, %q", time.Since(start), aus.String())
	}
}

// F-1012: Fehlerausgabe nur zur Diagnose und nie als Pipe (Verify V-3 m3).
func TestStderrWeitergeben(t *testing.T) {
	for _, f := range []struct{ debug, pipe, soll bool }{{false, false, false}, {true, false, true}, {true, true, false}, {false, true, false}} {
		if g := stderrWeitergeben(f.debug, f.pipe); g != f.soll {
			t.Errorf("%+v: %v", f, g)
		}
	}
}
