// SPDX-License-Identifier: GPL-3.0-or-later

package rescue

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	kette    chan struct{} // geschlossen, wenn der zuletzt eingereihte Auftrag fertig ist; nil: noch keiner
	aus      bool          // eine nicht angewendete Kopie ließ sich nicht beiseitelegen: diese Sitzung schreibt und löscht nichts
	fremd    bool          // ein anderes Fenster hält die Sperre auf die Ausgangsdatei (F-413)
	gestoert bool          // die Sperre ließ sich nicht prüfen (Symlink, Verzeichnis, Rechte, ...): Sitzung stumm (F-440)
	lageNr   uint64        // zählt Änderungen von fremd und gestoert (Lage, F-454)
	sperre   *sperre       // gehaltene Sperre auf kettenBasis; nil: keine
	wechsel  int           // NeueBasis-Aufträge, die noch in der Kette stehen: bis dahin entscheidet die Kette, nicht Sichere (F-446)

	// kettenBasis ist die Ausgangsdatei, für die fremd, gestoert, aus und sperre gelten: der Stand der Kette, nicht der
	// Aufrufe. basis setzt NeueBasis sofort, kettenBasis erst ihr Auftrag in der Kette (F-456). Aufträge der Kette lesen
	// nur kettenBasis oder einen Pfad, den ihr Aufruf mitgegeben hat, nie basis.
	kettenBasis string

	// Höchstens eine wartende Kopie (F-446): Sichere ersetzt sie, ein einziger Auftrag je Epoche (gen) schreibt sie. Bei
	// hängender Platte hält die Sitzung so eine Kopie, nicht eine je Aufruf.
	wartend    *Kopie
	wartendGen uint64      // gen, zu dem wartend gebildet wurde
	eingereiht bool        // für gen steht ein Schreibauftrag in der Kette
	Diagnose   func(error) // Fehler beim Schreiben und Löschen (nur Diagnose; die Oberfläche bleibt ruhig)
	jetzt      func() time.Time

	// Nähte für Tests (F-417): Schreiben und Löschen der Kopie.
	schreibe func(ort string, kp Kopie) error
	loesche  func(ort, pfad string) error
}

// NeueSicherung legt die Sicherung für die Ausgangsdatei basis im Ordner ort an und nimmt die Sperre auf sie. Hält ein
// anderes Fenster sie, ist die Sitzung fremd (Wiederherstellung.Fremd). Kann das Dateisystem kein flock (ENOLCK,
// ENOTSUP), läuft die Sitzung ohne Sperre wie vor F-413; scheitert das Anlegen oder Sperren anders (Ordner nicht
// beschreibbar, Symlink oder Verzeichnis am Sperrnamen), ist sie stumm (F-440).
func NeueSicherung(ort string, basis Kennung) *Sicherung {
	s := &Sicherung{ort: ort, basis: basis, Diagnose: func(error) {}, jetzt: time.Now, schreibe: Schreibe, loesche: Loesche}
	s.sperreFuer(basis.Pfad)
	return s
}

// sperreFuer nimmt die Sperre auf die Ausgangsdatei pfad (behält sie, wenn sie schon gehalten ist) und gibt eine Sperre
// auf eine andere frei. Aufrufer: NeueSicherung oder ein Auftrag der Kette.
func (s *Sicherung) sperreFuer(pfad string) {
	s.mu.Lock()
	alt, warFremd := s.sperre, s.fremd
	s.mu.Unlock()
	if alt != nil && alt.pfad == pfad {
		s.setzeSperre(pfad, false, false, alt)
		return
	}
	if alt != nil {
		if err := alt.frei(); err != nil {
			s.Diagnose(err)
		}
	}
	sp, err := sperreNehmen(s.ort, pfad)
	fremd := errors.Is(err, ErrBelegt)
	// Ein erneuter Versuch einer fremden Sitzung meldet nicht jedes Mal von Neuem.
	if err != nil && !(fremd && warFremd) {
		s.Diagnose(err)
	}
	s.setzeSperre(pfad, fremd, err != nil && !fremd && !errors.Is(err, errKeinFlock), sp)
}

// setzeSperre hält den Stand der Sperre für die Ausgangsdatei pfad fest (kettenBasis, F-456).
func (s *Sicherung) setzeSperre(pfad string, fremd, gestoert bool, sp *sperre) {
	s.mu.Lock()
	if fremd != s.fremd || gestoert != s.gestoert {
		s.lageNr++ // F-454: Die Oberfläche sieht am Zähler, dass sich etwas geändert hat
	}
	s.kettenBasis, s.fremd, s.gestoert, s.sperre = pfad, fremd, gestoert, sp
	s.mu.Unlock()
}

// Lage ist der Stand der Sperre dieser Sitzung für die Oberfläche (F-454).
type Lage struct {
	Fremd    bool   // ein anderes Fenster hält die Sperre: diese Sitzung sichert nicht
	Gestoert bool   // die Sperre ließ sich nicht prüfen: diese Sitzung sichert nicht
	Nr       uint64 // wächst mit jeder Änderung von Fremd oder Gestoert (0: seit dem Anlegen keine)
}

// Lage liefert den Stand der Sperre. Fremd wird die Sitzung bei NeueBasis oder beim Anlegen, frei wieder, wenn sie die
// Sperre später nachholt (F-436); beides läuft in der Kette, die Oberfläche fragt deshalb nach (ui.Rettung.Lage).
func (s *Sicherung) Lage() Lage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Lage{Fremd: s.fremd, Gestoert: s.gestoert, Nr: s.lageNr}
}

// uebernehmen versucht, die Sperre nachzuholen, wenn ein anderes Fenster sie hatte (F-436): ein Aufruf von flock ohne
// Warten, keine Suche auf der Platte. Gelingt es, übernimmt diese Sitzung die Ausgangsdatei; eine Kopie, die das andere
// Fenster hinterlassen hat (es endete mit ungesicherten Änderungen oder stürzte ab), legt sie vorher beiseite, statt sie
// zu überschreiben. Auftrag der Kette.
//
// Nur ein Auftrag der laufenden Epoche gen übernimmt (F-456): Ein verfallener schriebe nichts, und eine Übernahme der
// Datei, die die Sitzung gerade verlässt, legte die Kopie des anderen Fensters nur beiseite, wo sie beim nächsten
// Öffnen nicht mehr angewendet wird. Nachgeholt wird die Sperre der Datei, zu der die Sitzung fremd wurde
// (kettenBasis), nie die einer neuen Basis, deren NeueBasis-Auftrag noch aussteht: Sonst gälte die Sitzung als nicht
// fremd, und dieser Auftrag löschte danach die Kopie der alten Datei, die das andere Fenster noch hält.
func (s *Sicherung) uebernehmen(gen uint64) {
	s.mu.Lock()
	fremd, pfad, verfallen := s.fremd, s.kettenBasis, gen != s.gen
	s.mu.Unlock()
	if !fremd || verfallen {
		return
	}
	s.sperreFuer(pfad)
	s.mu.Lock()
	frei := !s.fremd && !s.gestoert
	s.mu.Unlock()
	if !frei {
		return
	}
	if _, err := os.Lstat(filepath.Join(s.ort, Name(pfad))); err != nil {
		return
	}
	if _, err := legeBeiseite(s.ort, pfad, s.jetzt()); err != nil {
		s.Diagnose(err)
		s.mu.Lock()
		s.aus = true
		s.mu.Unlock()
	}
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
func (s *Sicherung) stumm() bool { return s.aus || s.fremd || s.gestoert }

// Sichere bildet die Rettungskopie von d (Aufrufer: Ereignisschleife, kein Speichern) und schreibt sie im
// Hintergrund. Ein laufender Zug zählt nicht: Die Kopie trägt den Stand d.Stand() ohne das Objekt im Zug (F-409). Ist
// der Verlauf unvollständig (ErrUnfertig) oder die Kopie über dem Flächenbudget (ErrZuGross, F-431), bleibt die
// zuletzt geschriebene Kopie stehen.
func (s *Sicherung) Sichere(d *document.Dokument) {
	s.mu.Lock()
	// Entscheidend ist die Kette (F-446, F-436, F-457): Ein NeueBasis-Auftrag setzt aus, fremd und gestoert erst dort
	// zurück, und eine fremde Sitzung holt die Sperre dort nach. Nur was ohne ausstehenden Wechsel stumm ist, fällt hier
	// schon weg.
	if (s.gestoert || s.aus) && s.wechsel == 0 {
		s.mu.Unlock()
		return
	}
	kp, err := Erzeuge(d, s.basis, s.jetzt())
	if err != nil {
		s.mu.Unlock()
		s.Diagnose(err)
		return
	}
	gen := s.gen
	s.wartend, s.wartendGen = &kp, gen
	if !s.eingereiht {
		s.eingereiht = true
		s.reihe(func() { s.schreibeWartende(gen) })
	}
	s.mu.Unlock()
}

// schreibeWartende schreibt die wartende Kopie der Epoche gen (Auftrag der Kette). Sie hat sie vorher aus der Hand
// gegeben: Zur Zeit eines hängenden Schreibens ersetzt Sichere nur wartend, ein Auftrag folgt erst danach.
func (s *Sicherung) schreibeWartende(gen uint64) {
	s.uebernehmen(gen)
	s.mu.Lock()
	kp := s.wartend
	if kp == nil || s.wartendGen != gen {
		// Verworfen oder neue Basis: Eine Kopie einer späteren Epoche hat ihren eigenen Auftrag.
		s.mu.Unlock()
		return
	}
	s.wartend, s.eingereiht = nil, false
	veraltet := s.stumm() || gen != s.gen
	s.mu.Unlock()
	if veraltet {
		return
	}
	if err := s.schreibe(s.ort, *kp); err != nil {
		s.Diagnose(err)
	}
}

// Verwirf löscht die Rettungskopie (nach einem gelungenen Speichern, beim Schließen ohne Änderung, zurück auf dem
// gespeicherten Stand); noch nicht geschriebene Kopien verfallen. Das Löschen läuft in der Kette: Verwirf kehrt sofort
// zurück, auch wenn ein Schreiben hängt (F-417), und eine danach gewollte Kopie wird erst nach dem Löschen geschrieben.
func (s *Sicherung) Verwirf() {
	s.mu.Lock()
	s.gen++
	s.wartend, s.eingereiht = nil, false
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
	s.wartend, s.eingereiht = nil, false
	alt := s.basis.Pfad
	s.basis = k
	s.wechsel++
	s.reihe(func() {
		s.loescheKopie(alt)
		s.mu.Lock()
		s.aus = false // die neue Ausgangsdatei hat keine beiseitegelegte Kopie im Weg
		s.mu.Unlock()
		s.sperreFuer(k.Pfad)
		s.mu.Lock()
		s.wechsel--
		s.mu.Unlock()
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
	Gestoert   bool      // die Sperre ließ sich nicht prüfen: nichts gelesen, diese Sitzung sichert nicht (F-440)
}

// Wiederherstellen wendet beim Öffnen eine passende Rettungskopie auf d an (ADR-0029 Punkt 6). Passt die Kennung
// nicht oder scheitert das Anwenden, bleibt die Kopie liegen (bis zum Aufräumen nach Haltezeit) und d unverändert. Ist
// die Sitzung fremd, bleibt die Kopie des anderen Fensters unberührt. Aufruf nur direkt nach NeueSicherung, vor
// NeueBasis: Es liest basis und den Stand der Sperre zugleich, beide gehören nur bis dahin sicher zusammen (F-456).
func (s *Sicherung) Wiederherstellen(d *document.Dokument, pruefe func(*document.Dokument) error) Wiederherstellung {
	s.mu.Lock()
	basis, fremd, gestoert := s.basis, s.fremd, s.gestoert
	s.mu.Unlock()
	if gestoert {
		return Wiederherstellung{Gestoert: true}
	}
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
