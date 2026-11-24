// SPDX-License-Identifier: GPL-3.0-or-later

package library

// Härtung der Bibliothek nach der Sitzungs-QA vom 24.11.2026 (F-673, F-675, F-678, F-685, F-686, F-689, F-690).

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/saveprogress"
)

func indexPfade(t *testing.T, b *Bibliothek) map[string]bool {
	t.Helper()
	g, err := b.Laden()
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]bool{}
	for _, e := range g.Eintraege {
		m[e.Pfad] = true
	}
	return m
}

func vorschauZahl(t *testing.T, b *Bibliothek) int {
	t.Helper()
	es, err := os.ReadDir(b.Orte.Vorschau)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
	return len(es)
}

// F-685: Fehlt die Ablage (nicht eingehängt, umbenannt) oder ist sie leer (leerer Einhängepunkt), verwirft Abgleich
// weder Index noch Vorschauen, sondern meldet ErrAblageLeer. Nach der Rückkehr der Ablage ist alles wie vorher.
func TestAbgleichOhneErreichbareAblage(t *testing.T) {
	for _, fall := range []string{"fehlt", "leer", "Verweis ins Leere"} {
		t.Run(fall, func(t *testing.T) {
			b, _ := neueBibliothek(t)
			for i := range 3 {
				ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T10:00:00Z").Add(time.Duration(i) * time.Hour)})
			}
			weg := b.Orte.Ablage + ".weg"
			if err := os.Rename(b.Orte.Ablage, weg); err != nil {
				t.Fatal(err)
			}
			switch fall {
			case "leer":
				if err := os.Mkdir(b.Orte.Ablage, 0o700); err != nil {
					t.Fatal(err)
				}
			case "Verweis ins Leere":
				if err := os.Symlink(b.Orte.Ablage+".nirgends", b.Orte.Ablage); err != nil {
					t.Fatal(err)
				}
			}
			n, err := b.Abgleich()
			if !errors.Is(err, ErrAblageLeer) || n != 0 {
				t.Errorf("Abgleich = %d, %v; soll 0, ErrAblageLeer", n, err)
			}
			if got := len(indexPfade(t, b)); got != 3 {
				t.Errorf("Index %d Einträge, soll 3", got)
			}
			if got := vorschauZahl(t, b); got != 3 {
				t.Errorf("%d Vorschauen, soll 3", got)
			}
		})
	}
}

// F-685, Gegenstück: Liegt noch eine Aufnahme in der Ablage, fallen fehlende Einträge wie gewohnt weg.
func TestAbgleichMitRestAufnahme(t *testing.T) {
	b, _ := neueBibliothek(t)
	e1 := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T10:00:00Z")})
	e2 := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T11:00:00Z")})
	if err := os.Remove(filepath.Join(b.Orte.Ablage, filepath.FromSlash(e1.Pfad))); err != nil {
		t.Fatal(err)
	}
	if n, err := b.Abgleich(); err != nil || n != 1 {
		t.Fatalf("Abgleich = %d, %v", n, err)
	}
	if p := indexPfade(t, b); len(p) != 1 || !p[e2.Pfad] {
		t.Errorf("Index %v", p)
	}
}

// ADR-0016 Nachtrag 6: Fehlt die Indexdatei, die Ablage hat aber Aufnahmen, baut Abgleich den Index neu auf (Weg zum
// Wiederherstellen). Ohne Aufnahmen legt er nichts an.
func TestAbgleichBautFehlendenIndexNeu(t *testing.T) {
	b, _ := neueBibliothek(t)
	if n, err := b.Abgleich(); err != nil || n != 0 {
		t.Fatalf("erster Start: %d, %v", n, err)
	}
	if _, err := os.Lstat(b.Orte.Index); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("erster Start legt einen Index an: %v", err)
	}
	e1 := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T10:00:00Z")})
	e2 := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T11:00:00Z")})
	if err := os.Remove(b.Orte.Index); err != nil {
		t.Fatal(err)
	}
	if n, err := b.Abgleich(); err != nil || n != 0 {
		t.Fatalf("Abgleich: %d, %v", n, err)
	}
	if p := indexPfade(t, b); len(p) != 2 || !p[e1.Pfad] || !p[e2.Pfad] {
		t.Errorf("Index nach Neuaufbau %v", p)
	}
	// Ein leerer, aber vorhandener Index bleibt leer (kein Neuaufbau bei jedem Start).
	if err := os.WriteFile(b.Orte.Index, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if n, err := b.Abgleich(); err != nil || n != 0 || len(indexPfade(t, b)) != 0 {
		t.Errorf("leerer Index: %d, %v, %v", n, err, indexPfade(t, b))
	}
}

// F-673: Ein zweiter Prozess speichert eine Schwärzung, während SetzeSchlagworte die Datei neu kodiert.
// SetzeSchlagworte darf sie nicht überschreiben (ErrGeaendert), die Schwärzung bleibt.
func TestSetzeSchlagworteUeberschreibtKeineSchwaerzung(t *testing.T) {
	b, _ := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-05-01T10:00:00Z")})
	voll := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
	zweiter := Neu(b.Orte) // der andere Prozess: eigener Mutex
	zweiter.Zone, zweiter.Jetzt = b.Zone, b.Jetzt

	f, err := os.Open(voll)
	if err != nil {
		t.Fatal(err)
	}
	g, err := fileformat.LeseArbeitsdatei(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	fi, err := os.Lstat(voll)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Dokument.Schwaerzen(image.Rect(0, 0, 40, 30), document.Schwaerzung{Farbe: color.NRGBA{A: 255}}); err != nil {
		t.Fatal(err)
	}
	einmal := false // der Haken läuft auch im Speichern des zweiten: nur beim ersten Mal eingreifen
	var errZweiter error
	alt := saveprogress.SetzeHaken(func(schritt string) {
		if schritt != "bild-kodiert" || einmal {
			return
		}
		einmal = true
		_, _, errZweiter = zweiter.Speichern(e.Pfad, standAus(fi), g.Dokument, g.Dokument.Original())
	})
	t.Cleanup(func() { saveprogress.SetzeHaken(alt) })

	_, err = b.SetzeSchlagworte(e.Pfad, []string{"rechnung"})
	// Der zweite sieht die temporäre Datei des ersten im Ordner (zu jung zum Räumen) und warnt nur (ErrTempRest).
	if !einmal || (errZweiter != nil && !errors.Is(errZweiter, ErrTempRest)) {
		t.Fatalf("zweiter Prozess: lief %v, %v", einmal, errZweiter)
	}
	if !errors.Is(err, ErrGeaendert) {
		t.Errorf("SetzeSchlagworte = %v, soll ErrGeaendert", err)
	}
	f, err = os.Open(voll)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	nach, err := fileformat.LeseArbeitsdatei(f)
	if err != nil {
		t.Fatal(err)
	}
	if r, gg, bb, _ := nach.Dokument.Original().At(4, 4).RGBA(); r != 0 || gg != 0 || bb != 0 || !nach.Meta.Bearbeitet {
		t.Errorf("Schwärzung überschrieben: Pixel %d/%d/%d, Bearbeitet %v", r>>8, gg>>8, bb>>8, nach.Meta.Bearbeitet)
	}
	if len(nach.Meta.Schlagworte) != 0 {
		t.Errorf("Schlagworte %v", nach.Meta.Schlagworte)
	}
}

// F-675: Ein Monatsordner, der als Verweis aus der Ablage führt: SetzeSchlagworte und InPapierkorb lehnen ab
// (ErrPfad) und lassen die Dateien draußen unberührt. Dazu: ein Ordner mit Endung .png wird nicht verschoben.
func TestVerweisOrdnerAusDerAblage(t *testing.T) {
	b, heim := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-05-01T10:00:00Z")})
	daten, err := os.ReadFile(filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad)))
	if err != nil {
		t.Fatal(err)
	}
	aussen := filepath.Join(heim, "woanders")
	if err := os.MkdirAll(aussen, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"a.png", "b.png"} {
		if err := os.WriteFile(filepath.Join(aussen, n), daten, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(aussen, filepath.Join(b.Orte.Ablage, "2020-01")); err != nil {
		t.Fatal(err)
	}
	stand := func(n string) Stand {
		fi, err := os.Lstat(filepath.Join(aussen, n))
		if err != nil {
			t.Fatalf("%s: %v", n, err)
		}
		return standAus(fi)
	}
	vorA, vorB := stand("a.png"), stand("b.png")
	if _, err := b.SetzeSchlagworte("2020-01/a.png", []string{"x"}); !errors.Is(err, ErrPfad) {
		t.Errorf("SetzeSchlagworte = %v, soll ErrPfad", err)
	}
	if _, err := b.InPapierkorb("2020-01/b.png", zeit("2026-11-19T12:00:00Z")); !errors.Is(err, ErrPfad) {
		t.Errorf("InPapierkorb = %v, soll ErrPfad", err)
	}
	if stand("a.png") != vorA || stand("b.png") != vorB {
		t.Error("Datei außerhalb der Ablage verändert oder verschoben")
	}
	ordner := filepath.Join(b.Orte.Ablage, "2026-05", "ordner.png")
	if err := os.Mkdir(ordner, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := b.InPapierkorb("2026-05/ordner.png", zeit("2026-11-19T12:00:00Z")); !errors.Is(err, ErrPfad) {
		t.Errorf("Ordner: %v, soll ErrPfad", err)
	}
	if fi, err := os.Lstat(ordner); err != nil || !fi.IsDir() {
		t.Errorf("Ordner bewegt: %v", err)
	}
}

// F-678 Teil 2: Ist die Datei verschoben, aber der Index nicht nachgeführt, kommt ErrIndex mit gefülltem Verschoben;
// ein Fehlschlag vor dem Verschieben trägt nie ErrIndex.
func TestInPapierkorbIndexfehlerNachVerschieben(t *testing.T) {
	b, _ := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-05-01T10:00:00Z")})
	f := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-05-01T11:00:00Z")})
	alt := flockFn
	flockFn = func(int, int) error { return syscall.EIO }
	v, err := b.InPapierkorb(e.Pfad, zeit("2026-11-19T12:00:00Z"))
	flockFn = alt
	if !errors.Is(err, ErrIndex) || v.Papierkorb == "" || v.Pfad != e.Pfad {
		t.Fatalf("InPapierkorb = %+v, %v; soll gefüllt mit ErrIndex", v, err)
	}
	if _, err := os.Lstat(v.Papierkorb); err != nil {
		t.Errorf("nicht im Papierkorb: %v", err)
	}
	if b.fehltInAblage(e.Pfad) != true {
		t.Error("Datei liegt noch in der Ablage")
	}
	// Fehlschlag vor dem Verschieben: kein ErrIndex.
	ersetzeUmbenenne(t, func(string, string) error { return &os.LinkError{Op: "rename", Err: syscall.EXDEV} })
	if v, err := b.InPapierkorb(f.Pfad, zeit("2026-11-19T12:00:00Z")); err == nil || errors.Is(err, ErrIndex) || v.Papierkorb != "" {
		t.Errorf("gescheitertes Verschieben = %+v, %v", v, err)
	}
	if _, err := b.InPapierkorb("2026-05/gibtsnicht.png", zeit("2026-11-19T12:00:00Z")); err == nil || errors.Is(err, ErrIndex) {
		t.Errorf("fehlende Datei: %v", err)
	}
}

// F-686: Jede Operation, die den Index liest, ändert und ersetzt, hält die Sperre bis nach dem Umbenennen. Ein zweiter
// Prozess (zweite Bibliothek, eigener Mutex), der genau in diesem Augenblick eine Zeile anhängt, wartet und verliert
// sie nicht.
func TestIndexsperreGegenZweitenProzess(t *testing.T) {
	jetzt := zeit("2026-11-19T12:00:00Z")
	faelle := []struct {
		name string
		tu   func(t *testing.T, b *Bibliothek, es []Eintrag) error
	}{
		{"Abgleich", func(t *testing.T, b *Bibliothek, es []Eintrag) error {
			if err := os.Remove(filepath.Join(b.Orte.Ablage, filepath.FromSlash(es[0].Pfad))); err != nil {
				t.Fatal(err)
			}
			_, err := b.Abgleich()
			return err
		}},
		{"SetzeSchlagworte", func(t *testing.T, b *Bibliothek, es []Eintrag) error {
			_, err := b.SetzeSchlagworte(es[0].Pfad, []string{"x"})
			return err
		}},
		{"Speichern", func(t *testing.T, b *Bibliothek, es []Eintrag) error {
			fi, err := os.Lstat(filepath.Join(b.Orte.Ablage, filepath.FromSlash(es[0].Pfad)))
			if err != nil {
				t.Fatal(err)
			}
			d := neuesDokument(t, 40, 30)
			_, _, err = b.Speichern(es[0].Pfad, standAus(fi), d, d.Original())
			return err
		}},
		{"InPapierkorb", func(t *testing.T, b *Bibliothek, es []Eintrag) error {
			_, err := b.InPapierkorb(es[0].Pfad, jetzt)
			return err
		}},
		{"Entfernen", func(t *testing.T, b *Bibliothek, es []Eintrag) error {
			fi, err := os.Lstat(filepath.Join(b.Orte.Ablage, filepath.FromSlash(es[0].Pfad)))
			if err != nil {
				t.Fatal(err)
			}
			_, err = b.Entfernen(es[0].Pfad, standAus(fi))
			return err
		}},
		{"Aufraeumen", func(t *testing.T, b *Bibliothek, es []Eintrag) error {
			erg, err := b.Aufraeumen(jetzt, Einstellung{An: true, Tage: 1})
			if err == nil && len(erg.Verschoben) == 0 {
				t.Fatal("nichts aufgeräumt")
			}
			return err
		}},
		{"Neuaufbau", func(t *testing.T, b *Bibliothek, es []Eintrag) error {
			_, err := b.Neuaufbau()
			return err
		}},
	}
	for _, fall := range faelle {
		t.Run(fall.name, func(t *testing.T) {
			b, _ := neueBibliothek(t)
			var es []Eintrag
			for i := range 2 {
				es = append(es, ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-05-01T10:00:00Z").Add(time.Duration(i) * time.Hour)}))
			}
			zweiter := Neu(b.Orte)
			fremd := Eintrag{Pfad: "2026-05/zweiter-prozess.png", Erstellt: jetzt, Breite: 4, Hoehe: 3}
			fertig := make(chan error, 1)
			var einmal sync.Once
			ersetzeUmbenenne(t, func(von, nach string) error {
				if nach == b.Orte.Index {
					einmal.Do(func() {
						go func() { fertig <- zweiter.haengeAn(fremd) }()
						// Ohne Sperre hängt der zweite sofort an, und das Umbenennen verwirft die Zeile.
						select {
						case err := <-fertig:
							fertig <- err
						case <-time.After(300 * time.Millisecond):
						}
					})
				}
				return os.Rename(von, nach)
			})
			if err := fall.tu(t, b, es); err != nil {
				t.Fatalf("%s: %v", fall.name, err)
			}
			select {
			case err := <-fertig:
				if err != nil {
					t.Fatalf("zweiter Prozess: %v", err)
				}
			case <-time.After(sperrFrist + 5*time.Second):
				t.Fatal("zweiter Prozess hängt")
			}
			if !indexPfade(t, b)[fremd.Pfad] {
				t.Error("Zeile des zweiten Prozesses verloren")
			}
		})
	}
}

// F-686: Die Sperrdatei liegt neben dem Index mit 0600; ein Dateisystem ohne flock läuft ohne Sperre weiter, ein
// anderer Fehler beim Sperren scheitert (ErrIndex beim Ablegen, die Aufnahme liegt).
func TestIndexsperreDateiUndOhneFlock(t *testing.T) {
	b, _ := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-05-01T10:00:00Z")})
	fi, err := os.Lstat(b.Orte.Index + ".lock")
	if err != nil || fi.Mode().Perm() != 0o600 || !fi.Mode().IsRegular() {
		t.Fatalf("Sperrdatei: %v, %v", fi, err)
	}
	alt := flockFn
	t.Cleanup(func() { flockFn = alt })
	for _, kein := range []error{syscall.ENOLCK, syscall.ENOTSUP, syscall.EOPNOTSUPP} {
		flockFn = func(int, int) error { return kein }
		if _, err := b.SetzeSchlagworte(e.Pfad, []string{kein.Error()}); err != nil {
			t.Errorf("ohne flock (%v): %v", kein, err)
		}
	}
	flockFn = func(int, int) error { return syscall.EIO }
	d := neuesDokument(t, 40, 30)
	neu, err := b.Ablegen(d, d.Original(), fileformat.Metadaten{Erstellt: zeit("2026-05-01T11:00:00Z")})
	if !errors.Is(err, ErrIndex) || neu.Pfad == "" || b.fehltInAblage(neu.Pfad) {
		t.Errorf("Ablegen bei Sperrfehler: %+v, %v", neu, err)
	}
}

// F-686: Hält ein anderer die Sperre über sperrFrist, scheitert das Anhängen mit errIndexBelegt statt zu hängen. Die
// Frist ist hier nicht verkürzt; geprüft wird nur, dass ein EWOULDBLOCK bis zur Frist wiederholt wird.
func TestIndexsperreWartetUndGibtAuf(t *testing.T) {
	b, _ := neueBibliothek(t)
	alt := flockFn
	t.Cleanup(func() { flockFn = alt })
	versuche := 0
	flockFn = func(fd, art int) error {
		if art&syscall.LOCK_UN != 0 {
			return nil
		}
		versuche++
		if versuche < 4 {
			return syscall.EWOULDBLOCK
		}
		return nil
	}
	if err := b.haengeAn(Eintrag{Pfad: "2026-05/a.png", Erstellt: zeit("2026-05-01T10:00:00Z"), Breite: 1, Hoehe: 1}); err != nil || versuche != 4 {
		t.Errorf("haengeAn = %v nach %d Versuchen", err, versuche)
	}
}

// F-689: Teilwort faltet wie das ganze Schlagwort (EqualFold); ein ganzes Wort als Teilwort trifft genau dann, wenn es
// als Schlagwort trifft. Ein Teilwort nur aus Leerraum gilt als keins.
func TestTeilwortFaltetWieSchlagwort(t *testing.T) {
	faelle := []struct {
		schlagwort, suche string
		soll              bool
	}{
		{"ΟΔΟΣ", "οδος", true},   // Schluss-Sigma
		{"Meſſe", "messe", true}, // langes s
		{"Kelvin", "Kelvin", true},
		{"İstanbul", "istanbul", false}, // keine einfache Faltung
		{"Straße", "STRASSE", false},    // keine einfache Faltung
		{"ÄRGER", "ärger", true},
	}
	for _, f := range faelle {
		e := Eintrag{Schlagworte: []string{f.schlagwort}}
		if got := hatTeilwort(e, f.suche); got != f.soll || got != hatSchlagwort(e, f.suche) {
			t.Errorf("%q mit %q: Teilwort %v, Schlagwort %v, soll %v", f.schlagwort, f.suche, got, hatSchlagwort(e, f.suche), f.soll)
		}
	}
	if !hatTeilwort(Eintrag{Schlagworte: []string{"ΟΔΟΣ"}}, "δος") {
		t.Error("Teilwort mit Schluss-Sigma trifft nicht")
	}
	b, _ := neueBibliothek(t)
	mit := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-05-01T10:00:00Z"), Schlagworte: []string{"a"}})
	ohne := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-05-01T11:00:00Z")})
	es, err := b.Suchen(Anfrage{Teilwort: " \t "})
	if err != nil || len(es) != 2 || es[0].Pfad != ohne.Pfad || es[1].Pfad != mit.Pfad {
		t.Errorf("Teilwort aus Leerraum: %v, %v", es, err)
	}
}

// F-690: Die Dateiprüfung des Abgleichs läuft ohne b.mu und ohne die Sperre des Index; was in dieser Zeit neu in den
// Index kommt, bleibt nach dem Abgleich stehen.
func TestAbgleichPrueftOhneSperren(t *testing.T) {
	b, _ := neueBibliothek(t)
	var es []Eintrag
	for i := range 3 {
		es = append(es, ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-05-01T10:00:00Z").Add(time.Duration(i) * time.Hour)}))
	}
	if err := os.Remove(filepath.Join(b.Orte.Ablage, filepath.FromSlash(es[0].Pfad))); err != nil {
		t.Fatal(err)
	}
	zweiter := Neu(b.Orte)
	fremd := Eintrag{Pfad: "2026-05/zweiter-prozess.png", Erstellt: zeit("2026-11-19T12:00:00Z"), Breite: 4, Hoehe: 3}
	var frei []bool
	alt := lstatAblage
	lstatAblage = func(p string) (os.FileInfo, error) {
		ok := b.mu.TryLock()
		if ok {
			b.mu.Unlock()
		}
		// Die Sperre des Index: ein eigener Deskriptor bekäme sie nur, wenn niemand sie hält.
		if f, err := os.OpenFile(b.sperrPfad(), os.O_RDWR, 0); err == nil {
			if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
				ok = false
			}
			f.Close()
		}
		frei = append(frei, ok)
		if len(frei) == 1 {
			if err := zweiter.haengeAn(fremd); err != nil {
				t.Errorf("zweiter Prozess: %v", err)
			}
		}
		return alt(p)
	}
	t.Cleanup(func() { lstatAblage = alt })
	if n, err := b.Abgleich(); err != nil || n != 1 {
		t.Fatalf("Abgleich = %d, %v", n, err)
	}
	if len(frei) < len(es) {
		t.Fatalf("%d Prüfungen, soll mindestens %d", len(frei), len(es))
	}
	for i := range es {
		if !frei[i] {
			t.Errorf("Prüfung %d lief unter einer Sperre", i)
		}
	}
	p := indexPfade(t, b)
	if !p[fremd.Pfad] || p[es[0].Pfad] || !p[es[1].Pfad] || !p[es[2].Pfad] {
		t.Errorf("Index nach Abgleich %v", p)
	}
}

// BenchmarkAbgleich10000 misst Abgleich über 10 000 Einträge, deren Dateien alle liegen (der Regelfall bei jedem Start
// des Editors und jedem Öffnen der Ansicht, F-690). Gemessen am 24.11.2026 auf lokaler SSD: rund 31 ms, davon rund
// 15 ms Lesen des Index unter b.mu (BenchmarkSuchen10000), der Rest Lstat ohne Sperre. Aufruf:
//
//	go test ./internal/library -run '^$' -bench Abgleich10000 -benchmem
func BenchmarkAbgleich10000(b *testing.B) {
	dir := b.TempDir()
	bib := Neu(Orte{
		Ablage:     filepath.Join(dir, "ablage"),
		Index:      filepath.Join(dir, "daten", "schnapp", "bibliothek.jsonl"),
		Vorschau:   filepath.Join(dir, "cache"),
		Papierkorb: filepath.Join(dir, "daten", "Trash"),
	})
	start := time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC)
	es := make([]Eintrag, 10000)
	for i := range es {
		t := start.Add(time.Duration(i) * 97 * time.Minute)
		pfad := fmt.Sprintf("%s/schnapp-%s.png", t.Format("2006-01"), t.Format("2006-01-02_15-04-05"))
		voll := filepath.Join(bib.Orte.Ablage, filepath.FromSlash(pfad))
		if err := os.MkdirAll(filepath.Dir(voll), 0o700); err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(voll, nil, 0o600); err != nil {
			b.Fatal(err)
		}
		es[i] = Eintrag{Pfad: pfad, Erstellt: t, Breite: 2560, Hoehe: 1440, Schlagworte: []string{"probe"}}
	}
	if err := bib.schreibeIndex(es); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for b.Loop() {
		if n, err := bib.Abgleich(); err != nil || n != 0 {
			b.Fatalf("%d, %v", n, err)
		}
	}
}
