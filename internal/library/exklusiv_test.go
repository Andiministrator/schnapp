// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/fileformat"
)

// restdateien zählt temporäre Dateien von Schnapp im Ordner.
func restdateien(t *testing.T, ordner string) []string {
	t.Helper()
	var r []string
	for _, n := range namen(t, ordner) {
		if strings.HasPrefix(n, tempPraefix) {
			r = append(r, n)
		}
	}
	return r
}

// F-48: Zwei Bibliotheken (wie zwei Schnapp-Prozesse) auf derselben Ablage legen in derselben Sekunde parallel ab:
// Es entstehen genauso viele verschiedene, vollständige Dateien wie Aufrufe; keine überschreibt eine andere.
func TestAblegenZweiInstanzenGleicheSekundeParallel(t *testing.T) {
	b1, _ := neueBibliothek(t)
	b2 := Neu(b1.Orte) // eigener Mutex, gleiche Ablage
	b2.Zone, b2.Jetzt = b1.Zone, b1.Jetzt
	m := fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")}

	const je = 6
	var wg sync.WaitGroup
	start := make(chan struct{})
	pfade := make(chan string, 2*je)
	fehler := make(chan error, 2*je)
	for _, b := range []*Bibliothek{b1, b2} {
		for i := 0; i < je; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				d := neuesDokument(t, 40, 30)
				<-start
				e, err := b.Ablegen(d, d.Original(), m)
				if err != nil {
					fehler <- err
					return
				}
				pfade <- e.Pfad
			}()
		}
	}
	close(start)
	wg.Wait()
	close(pfade)
	close(fehler)
	for err := range fehler {
		t.Errorf("Ablegen: %v", err)
	}
	gesehen := map[string]bool{}
	for p := range pfade {
		if gesehen[p] {
			t.Errorf("Pfad %q doppelt vergeben", p)
		}
		gesehen[p] = true
		if g, err := fileformat.Oeffne(ablageDatei(b1, p)); err != nil || g.Fremd {
			t.Errorf("%s nicht vollständig lesbar: %v", p, err)
		}
	}
	if len(gesehen) != 2*je {
		t.Errorf("%d verschiedene Dateien, soll %d", len(gesehen), 2*je)
	}
	if r := restdateien(t, filepath.Join(b1.Orte.Ablage, "2026-11")); len(r) != 0 {
		t.Errorf("Restdateien %v", r)
	}
}

// F-48: Eine schon vorhandene Datei mit dem Wunschnamen (von anderer Seite angelegt) wird nie überschrieben, auf dem
// Link-Weg wie auf dem Rückfallweg ohne Hardlinks.
func TestAblegenUeberschreibtNichtWege(t *testing.T) {
	for _, fall := range []struct {
		name string
		link func(string, string) error
	}{
		{"link", nil},
		{"ohne-hardlinks", func(string, string) error { return syscall.EPERM }},
		{"nicht-unterstuetzt", func(string, string) error { return syscall.ENOTSUP }},
	} {
		t.Run(fall.name, func(t *testing.T) {
			b, _ := neueBibliothek(t)
			if fall.link != nil {
				ersetzeVerknuepfe(t, fall.link)
			}
			fremd := filepath.Join(b.Orte.Ablage, "2026-11", "2026-11-19_14-35-12.png")
			schreibe(t, fremd, "fremder Inhalt")
			e := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")})
			if e.Pfad != "2026-11/2026-11-19_14-35-12-2.png" {
				t.Errorf("Pfad %q", e.Pfad)
			}
			if got, _ := os.ReadFile(fremd); string(got) != "fremder Inhalt" {
				t.Errorf("fremde Datei verändert: %q", got)
			}
			if g, err := fileformat.Oeffne(ablageDatei(b, e.Pfad)); err != nil || g.Fremd {
				t.Errorf("neue Datei: %v", err)
			}
			if modus(t, ablageDatei(b, e.Pfad)) != 0o600 {
				t.Errorf("Rechte %o", modus(t, ablageDatei(b, e.Pfad)))
			}
			if r := restdateien(t, filepath.Dir(fremd)); len(r) != 0 {
				t.Errorf("Restdateien %v", r)
			}
		})
	}
}

// F-48: Scheitert das Verknüpfen aus einem anderen Grund (E/A-Fehler), gibt es keinen Rückfall: Fehler kommt zurück,
// es bleibt keine Datei unter dem Zielnamen und keine temporäre, der Index ändert sich nicht.
func TestAblegenVerknuepfenScheitertOhneRueckfall(t *testing.T) {
	b, _ := neueBibliothek(t)
	ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")})
	ordner := filepath.Join(b.Orte.Ablage, "2026-11")
	vorher := namen(t, ordner)
	boese := errors.New("Attrappe: Verknüpfen scheitert")
	ersetzeVerknuepfe(t, func(string, string) error { return boese })
	d := neuesDokument(t, 40, 30)
	if _, err := b.Ablegen(d, d.Original(), fileformat.Metadaten{Erstellt: zeit("2026-11-19T13:00:00Z")}); !errors.Is(err, boese) {
		t.Fatalf("Fehler %v, soll die Attrappe sein", err)
	}
	if n := namen(t, ordner); len(n) != len(vorher) || n[0] != vorher[0] {
		t.Errorf("Ordner %v, vorher %v", n, vorher)
	}
	if g, _ := b.Laden(); len(g.Eintraege) != 1 {
		t.Errorf("Index hat %d Einträge, soll 1", len(g.Eintraege))
	}
}

// F-48: Scheitert auf dem Rückfallweg das Umbenennen, verschwindet auch die eigene Reservierung wieder.
func TestAblegenRueckfallUmbenennenScheitert(t *testing.T) {
	b, _ := neueBibliothek(t)
	ersetzeVerknuepfe(t, func(string, string) error { return syscall.EPERM })
	ersetzeUmbenenne(t, func(string, string) error { return errors.New("Attrappe: Umbenennen scheitert") })
	d := neuesDokument(t, 40, 30)
	if _, err := b.Ablegen(d, d.Original(), fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")}); err == nil {
		t.Fatal("Fehler erwartet")
	}
	if n := namen(t, filepath.Join(b.Orte.Ablage, "2026-11")); len(n) != 0 {
		t.Errorf("Ordner nach Fehlschlag %v, soll leer sein", n)
	}
}

// Die Uhr bleibt in diesem Test fest; zusätzlich ein Gegenstück ohne feste Erstellzeit.
func TestAblegenFesteUhrZweiBibliotheken(t *testing.T) {
	b1, _ := neueBibliothek(t)
	b2 := Neu(b1.Orte)
	b2.Zone = b1.Zone
	b2.Jetzt = func() time.Time { return time.Date(2026, 11, 19, 12, 35, 12, 0, time.UTC) }
	e1 := ablegen(t, b1, fileformat.Metadaten{})
	e2 := ablegen(t, b2, fileformat.Metadaten{})
	if e1.Pfad == e2.Pfad {
		t.Errorf("beide %q", e1.Pfad)
	}
}
