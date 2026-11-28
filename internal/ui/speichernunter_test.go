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
	if strings.Contains(string(mustLesen(t, ziel)), ordner) {
		t.Error("die Kopie trägt einen Pfad")
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
	if !l.e.meldung.Fehler || !strings.Contains(l.e.meldung.Text, "Ctrl+S") {
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
}
