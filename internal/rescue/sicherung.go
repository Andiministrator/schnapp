// SPDX-License-Identifier: GPL-3.0-or-later

package rescue

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

// endeFrist ist, wie lange Ende höchstens auf begonnene Aufträge wartet (F-417); Naht für Tests.
var endeFrist = 5 * time.Second

// ErrFrist meldet, dass Ende nicht abgewartet hat: Ein Auftrag (etwa ein Sync auf einem hängenden Zustandsordner)
// läuft noch, die zuletzt gewollte Kopie ist womöglich nicht geschrieben oder nicht gelöscht.
var ErrFrist = errors.New("rescue: Frist beim Beenden abgelaufen, Rettungskopie womöglich nicht auf dem letzten Stand")

// Sicherung führt die Rettungskopie einer Editor-Sitzung (ADR-0029 Punkte 4 und 5): Sichere bildet die Kopie auf der
// Ereignisschleife; Verwirf löscht sie; NeueBasis setzt nach einem gelungenen Speichern die neue Ausgangsdatei. Die
// Dateiarbeit läuft als Kette von Aufträgen in Goroutinen, streng in der Reihenfolge der Aufrufe (F-417): Kein Aufruf
// wartet auf die Platte, eine ältere Kopie überschreibt nie eine jüngere, ein Löschen trifft nie eine später gewollte
// Kopie, und nach Verwirf oder NeueBasis schreibt keine früher begonnene Kopie mehr.
//
// Die Sitzung hält eine Sperre auf ihre Ausgangsdatei (sperre.go, F-413). Hält ein anderes Fenster sie, ist diese
// Sitzung fremd: Sie wendet nichts an und schreibt und löscht keine Kopie.
type Sicherung struct {
	ort string

	// mu schützt die Felder und wird nie über einen Dateizugriff gehalten, damit kein Aufruf die Ereignisschleife
	// aufhält.
	mu       sync.Mutex
	basis    Kennung
	gen      uint64        // erhöht von Verwirf und NeueBasis: ältere Schreibaufträge verfallen
	naechste uint64        // Nummer des zuletzt eingereihten Schreibauftrags
	kette    chan struct{} // geschlossen, wenn der zuletzt eingereihte Auftrag fertig ist; nil: noch keiner
	aus      bool          // eine nicht angewendete Kopie ließ sich nicht beiseitelegen: diese Sitzung schreibt und löscht nichts
	fremd    bool          // ein anderes Fenster hält die Sperre auf die Ausgangsdatei (F-413)
	sperre   *sperre       // gehaltene Sperre auf basis.Pfad; nil: keine
	Diagnose func(error)   // Fehler beim Schreiben und Löschen (nur Diagnose; die Oberfläche bleibt ruhig)
	jetzt    func() time.Time

	// Nähte für Tests (F-417): Schreiben und Löschen der Kopie.
	schreibe func(ort string, kp Kopie) error
	loesche  func(ort, pfad string) error
}

// NeueSicherung legt die Sicherung für die Ausgangsdatei basis im Ordner ort an und nimmt die Sperre auf sie. Hält ein
// anderes Fenster sie, ist die Sitzung fremd (Wiederherstellung.Fremd). Lässt sich keine Sperre anlegen (Ordner nicht
// beschreibbar, Dateisystem ohne flock), läuft die Sitzung ohne Sperre wie vor F-413.
func NeueSicherung(ort string, basis Kennung) *Sicherung {
	s := &Sicherung{ort: ort, basis: basis, Diagnose: func(error) {}, jetzt: time.Now, schreibe: Schreibe, loesche: Loesche}
	s.sperreFuer(basis.Pfad)
	return s
}

// sperreFuer nimmt die Sperre auf die Ausgangsdatei pfad (behält sie, wenn sie schon gehalten ist) und gibt eine Sperre
// auf eine andere frei. Aufrufer: NeueSicherung oder ein Auftrag der Kette.
func (s *Sicherung) sperreFuer(pfad string) {
	s.mu.Lock()
	alt := s.sperre
	s.mu.Unlock()
	if alt != nil && alt.pfad == pfad {
		s.setzeFremd(false, alt)
		return
	}
	if alt != nil {
		if err := alt.frei(); err != nil {
			s.Diagnose(err)
		}
	}
	sp, err := sperreNehmen(s.ort, pfad)
	if err != nil {
		s.Diagnose(err)
	}
	s.setzeFremd(errors.Is(err, ErrBelegt), sp)
}

func (s *Sicherung) setzeFremd(fremd bool, sp *sperre) {
	s.mu.Lock()
	s.fremd, s.sperre = fremd, sp
	s.mu.Unlock()
}

// Basis liefert die aktuelle Ausgangsdatei.
func (s *Sicherung) Basis() Kennung {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.basis
}

// Ort liefert den Ordner der Rettungskopien.
func (s *Sicherung) Ort() string { return s.ort }

// reihe hängt auftrag an die Kette (Aufrufer hält mu): Er läuft in einer eigenen Goroutine, sobald der vorige fertig
// ist. So wartet kein Aufrufer auf die Platte, und die Aufträge laufen in der Reihenfolge der Aufrufe.
func (s *Sicherung) reihe(auftrag func()) {
	vorher, fertig := s.kette, make(chan struct{})
	s.kette = fertig
	go func() {
		defer close(fertig)
		if vorher != nil {
			<-vorher
		}
		auftrag()
	}()
}

// stumm meldet, ob diese Sitzung keine Kopie schreiben oder löschen darf (Aufrufer hält mu).
func (s *Sicherung) stumm() bool { return s.aus || s.fremd }

// Sichere bildet die Rettungskopie von d (Aufrufer: Ereignisschleife, kein Speichern) und schreibt sie im
// Hintergrund. Ein laufender Zug zählt nicht: Die Kopie trägt den Stand d.Stand() ohne das Objekt im Zug (F-409). Ist
// der Verlauf unvollständig (ErrUnfertig) oder die Kopie über dem Flächenbudget (ErrZuGross, F-431), bleibt die
// zuletzt geschriebene Kopie stehen.
func (s *Sicherung) Sichere(d *document.Dokument) {
	s.mu.Lock()
	if s.stumm() {
		s.mu.Unlock()
		return
	}
	kp, err := Erzeuge(d, s.basis, s.jetzt())
	if err != nil {
		s.mu.Unlock()
		s.Diagnose(err)
		return
	}
	s.naechste++
	nr, gen := s.naechste, s.gen
	s.reihe(func() {
		s.mu.Lock()
		// Verworfen, neue Basis, fremd, oder eine jüngere Kopie ist schon eingereiht (sie folgt in der Kette).
		veraltet := s.stumm() || gen != s.gen || nr != s.naechste
		s.mu.Unlock()
		if veraltet {
			return
		}
		if err := s.schreibe(s.ort, kp); err != nil {
			s.Diagnose(err)
		}
	})
	s.mu.Unlock()
}

// Verwirf löscht die Rettungskopie (nach einem gelungenen Speichern, beim Schließen ohne Änderung, zurück auf dem
// gespeicherten Stand); noch nicht geschriebene Kopien verfallen. Das Löschen läuft in der Kette: Verwirf kehrt sofort
// zurück, auch wenn ein Schreiben hängt (F-417), und eine danach gewollte Kopie wird erst nach dem Löschen geschrieben.
func (s *Sicherung) Verwirf() {
	s.mu.Lock()
	s.gen++
	pfad := s.basis.Pfad
	s.reihe(func() { s.loescheKopie(pfad) })
	s.mu.Unlock()
}

// loescheKopie löscht die Kopie zur Ausgangsdatei pfad, wenn diese Sitzung darf (Auftrag der Kette).
func (s *Sicherung) loescheKopie(pfad string) {
	s.mu.Lock()
	stumm := s.stumm()
	s.mu.Unlock()
	if stumm {
		return
	}
	if err := s.loesche(s.ort, pfad); err != nil {
		s.Diagnose(err)
	}
}

// NeueBasis setzt nach einem gelungenen Speichern die Ausgangsdatei (sie kann eine andere sein: neue Aufnahme der
// Bibliothek), löscht die Kopie der bisherigen und nimmt die Sperre auf die neue; noch nicht geschriebene Kopien
// verfallen. Die Dateiarbeit läuft in der Kette.
func (s *Sicherung) NeueBasis(k Kennung) {
	s.mu.Lock()
	s.gen++
	alt := s.basis.Pfad
	s.basis = k
	s.reihe(func() {
		s.loescheKopie(alt)
		s.mu.Lock()
		s.aus = false // die neue Ausgangsdatei hat keine beiseitegelegte Kopie im Weg
		s.mu.Unlock()
		s.sperreFuer(k.Pfad)
	})
	s.mu.Unlock()
}

// Ende wartet beim Zerstören des Fensters auf begonnene Aufträge, höchstens endeFrist (F-417), und gibt dann die Sperre
// frei. Läuft die Frist ab, meldet es ErrFrist an die Diagnose und liefert false; die Sperre bleibt bis zum Ende des
// Prozesses. Danach ruft die Oberfläche nichts mehr (Tests dürfen weiterarbeiten, dann ohne Sperre).
func (s *Sicherung) Ende() bool {
	s.mu.Lock()
	kette := s.kette
	s.mu.Unlock()
	if kette != nil {
		select {
		case <-kette:
		case <-time.After(endeFrist):
			s.Diagnose(fmt.Errorf("%w (%v)", ErrFrist, endeFrist))
			return false
		}
	}
	s.mu.Lock()
	sp := s.sperre
	s.sperre = nil
	s.mu.Unlock()
	if sp != nil {
		if err := sp.frei(); err != nil {
			s.Diagnose(err)
		}
	}
	return true
}

// Wiederherstellung ist das Ergebnis von Wiederherstellen.
type Wiederherstellung struct {
	Angewendet bool      // die Kopie wurde angewendet
	Gesichert  time.Time // Zeitpunkt der Kopie (angewendet oder nicht)
	Liegt      bool      // eine Kopie liegt, wurde aber nicht angewendet (Datei geändert, unlesbar, Grenzen)
	Grund      error     // warum nicht angewendet
	Beiseite   string    // Dateiname, unter dem die nicht angewendete Kopie im Ordner bleibt (leer: Umbenennen scheiterte)
	Fremd      bool      // ein anderes Fenster hat die Datei offen: nichts gelesen, diese Sitzung sichert nicht (F-413)
}

// Wiederherstellen wendet beim Öffnen eine passende Rettungskopie auf d an (ADR-0029 Punkt 6). Passt die Kennung
// nicht oder scheitert das Anwenden, bleibt die Kopie liegen (bis zum Aufräumen nach Haltezeit) und d unverändert. Ist
// die Sitzung fremd, bleibt die Kopie des anderen Fensters unberührt.
func (s *Sicherung) Wiederherstellen(d *document.Dokument, pruefe func(*document.Dokument) error) Wiederherstellung {
	s.mu.Lock()
	basis, fremd := s.basis, s.fremd
	s.mu.Unlock()
	if fremd {
		return Wiederherstellung{Fremd: true}
	}
	kp, err := Lies(s.ort, basis.Pfad)
	if errors.Is(err, ErrKeine) {
		return Wiederherstellung{}
	}
	w := Wiederherstellung{Liegt: true, Grund: err}
	if err == nil {
		w.Gesichert = kp.Gesichert
		if !kp.Datei.Gleich(basis) {
			w.Grund = ErrAnders
		} else if w.Grund = Anwenden(d, kp, pruefe); w.Grund == nil {
			return Wiederherstellung{Angewendet: true, Gesichert: kp.Gesichert}
		}
	}
	// Nicht angewendet: beiseitelegen, damit diese Sitzung sie weder überschreibt noch beim Schließen löscht; sie bleibt
	// bis zum Aufräumen nach Haltezeit (ADR-0029 Punkt 6, Verify 7, V7-1). Scheitert das Umbenennen, schreibt und löscht
	// diese Sitzung nichts.
	name, err := legeBeiseite(s.ort, basis.Pfad, s.jetzt())
	if err != nil {
		s.Diagnose(err)
		s.mu.Lock()
		s.aus = true
		s.mu.Unlock()
		return w
	}
	w.Beiseite = name
	return w
}
