// SPDX-License-Identifier: GPL-3.0-or-later

package rescue

import (
	"errors"
	"sync"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Sicherung führt die Rettungskopie einer Editor-Sitzung (ADR-0029 Punkte 4 und 5): Sichere bildet die Kopie auf der
// Ereignisschleife und schreibt sie in einer Goroutine; Verwirf löscht sie; NeueBasis setzt nach einem gelungenen
// Speichern die neue Ausgangsdatei. Schreiben und Löschen laufen nacheinander: Eine ältere Kopie überschreibt nie eine
// jüngere, und nach Verwirf oder NeueBasis schreibt keine früher begonnene Kopie mehr.
type Sicherung struct {
	ort string

	// schreibt reiht Schreiben und Löschen der Datei; mu schützt die Felder und wird nie über einen Dateizugriff
	// gehalten, damit Sichere die Ereignisschleife nicht aufhält.
	schreibt sync.Mutex
	mu       sync.Mutex
	basis    Kennung
	gen      uint64 // erhöht von Verwirf und NeueBasis: ältere Schreibaufträge verfallen
	naechste uint64 // Nummer des nächsten Schreibauftrags
	letzte   uint64 // Nummer des zuletzt geschriebenen
	laeuft   sync.WaitGroup
	aus      bool        // eine nicht angewendete Kopie ließ sich nicht beiseitelegen: diese Sitzung schreibt und löscht nichts
	Diagnose func(error) // Fehler beim Schreiben und Löschen (nur Diagnose; die Oberfläche bleibt ruhig)
	jetzt    func() time.Time
}

// NeueSicherung legt die Sicherung für die Ausgangsdatei basis im Ordner ort an.
func NeueSicherung(ort string, basis Kennung) *Sicherung {
	return &Sicherung{ort: ort, basis: basis, Diagnose: func(error) {}, jetzt: time.Now}
}

// Basis liefert die aktuelle Ausgangsdatei.
func (s *Sicherung) Basis() Kennung {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.basis
}

// Ort liefert den Ordner der Rettungskopien.
func (s *Sicherung) Ort() string { return s.ort }

// Sichere bildet die Rettungskopie von d (Aufrufer: Ereignisschleife, kein Zug, kein Speichern) und schreibt sie im
// Hintergrund. Ist der Verlauf unvollständig (ErrUnfertig), bleibt die zuletzt geschriebene Kopie stehen.
func (s *Sicherung) Sichere(d *document.Dokument) {
	s.mu.Lock()
	if s.aus {
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
	s.laeuft.Add(1)
	s.mu.Unlock()
	go func() {
		defer s.laeuft.Done()
		s.schreibt.Lock()
		defer s.schreibt.Unlock()
		s.mu.Lock()
		veraltet := gen != s.gen || nr <= s.letzte // verworfen, neue Basis oder schon eine jüngere Kopie geschrieben
		s.mu.Unlock()
		if veraltet {
			return
		}
		if err := Schreibe(s.ort, kp); err != nil {
			s.Diagnose(err)
			return
		}
		s.mu.Lock()
		s.letzte = nr
		s.mu.Unlock()
	}()
}

// Verwirf löscht die Rettungskopie (nach einem gelungenen Speichern, beim Schließen ohne Änderung, zurück auf dem
// gespeicherten Stand); noch nicht geschriebene Kopien verfallen.
func (s *Sicherung) Verwirf() {
	s.schreibt.Lock() // ein laufendes Schreiben erst beenden, sonst legte es die Kopie nach dem Löschen wieder ab
	defer s.schreibt.Unlock()
	s.mu.Lock()
	s.gen++
	pfad, aus := s.basis.Pfad, s.aus
	s.mu.Unlock()
	if aus {
		return
	}
	if err := Loesche(s.ort, pfad); err != nil {
		s.Diagnose(err)
	}
}

// NeueBasis setzt nach einem gelungenen Speichern die Ausgangsdatei (sie kann eine andere sein: neue Aufnahme der
// Bibliothek) und löscht die Kopie der bisherigen; noch nicht geschriebene Kopien verfallen.
func (s *Sicherung) NeueBasis(k Kennung) {
	s.schreibt.Lock()
	defer s.schreibt.Unlock()
	s.mu.Lock()
	s.gen++
	pfad, aus := s.basis.Pfad, s.aus
	s.basis, s.aus = k, false // die neue Ausgangsdatei hat keine beiseitegelegte Kopie im Weg
	s.mu.Unlock()
	if aus {
		return
	}
	if err := Loesche(s.ort, pfad); err != nil {
		s.Diagnose(err)
	}
}

// Ende wartet, bis begonnene Schreibaufträge fertig sind (beim Zerstören des Fensters).
func (s *Sicherung) Ende() { s.laeuft.Wait() }

// Wiederherstellung ist das Ergebnis von Wiederherstellen.
type Wiederherstellung struct {
	Angewendet bool      // die Kopie wurde angewendet
	Gesichert  time.Time // Zeitpunkt der Kopie (angewendet oder nicht)
	Liegt      bool      // eine Kopie liegt, wurde aber nicht angewendet (Datei geändert, unlesbar, Grenzen)
	Grund      error     // warum nicht angewendet
	Beiseite   string    // Dateiname, unter dem die nicht angewendete Kopie im Ordner bleibt (leer: Umbenennen scheiterte)
}

// Wiederherstellen wendet beim Öffnen eine passende Rettungskopie auf d an (ADR-0029 Punkt 6). Passt die Kennung
// nicht oder scheitert das Anwenden, bleibt die Kopie liegen (bis zum Aufräumen nach Haltezeit) und d unverändert.
func (s *Sicherung) Wiederherstellen(d *document.Dokument, pruefe func(*document.Dokument) error) Wiederherstellung {
	s.mu.Lock()
	basis := s.basis
	s.mu.Unlock()
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
