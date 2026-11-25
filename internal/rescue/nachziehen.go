// SPDX-License-Identifier: GPL-3.0-or-later

package rescue

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Andiministrator/schnapp/internal/errprefix"
)

// umMarke ist die Endung der Marke „UmDatei läuft“ neben der Sperre (F-757). UmDatei hält sie als flock-Sperre, solange
// es arbeitet (F-771): Ein Prozess, der darin abstürzt, gibt sie mit dem Ende frei, eine liegen gebliebene Datei hält also
// niemanden auf. Aufraeumen entfernt eine freie Marke nach umMarkeAlt.
const umMarke = ".um"

// umMarkeAlt: Eine freie Marke, die älter ist (oder deren Änderungszeit in der Zukunft liegt), räumt Aufraeumen weg.
// Eine Minute, damit Aufräumen nicht in die Mikrosekunden zwischen Anlegen und Sperren einer neuen Marke greift.
const umMarkeAlt = time.Minute

func umMarkeName(pfad string) string { return strings.TrimSuffix(Name(pfad), ".json") + umMarke }

// UmDateiLaeuft meldet, ob gerade UmDatei für die Ausgangsdatei pfad läuft (die Marke <SHA>.um ist gesperrt). Fehlt die
// Marke, ist sie frei (Rest eines Absturzes) oder nicht prüfbar (kein flock, Symlink, Rechte), läuft keins: Ausfall in
// Richtung „nicht warten“, wie vor F-757.
func UmDateiLaeuft(ort, pfad string) bool {
	belegt, err := sperreGehalten(filepath.Join(ort, umMarkeName(pfad)))
	return err == nil && belegt
}

// WarteUmDatei wartet höchstens frist, solange UmDatei für die Ausgangsdatei pfad läuft (UmDateiLaeuft). Ein Fenster,
// das dieselbe Aufnahme öffnet, ruft es vor dem Laden der Datei auf (F-769): Es liest dann den neuen Stand, bekommt die
// Sperre und wendet die nachgezogene Kopie an, statt fremd zu werden und die Kopie beim nächsten Sichern beiseitezulegen
// (F-757). Ohne laufendes UmDatei kehrt es sofort zurück, auch bei einer liegen gebliebenen Marke (F-771).
func WarteUmDatei(ort, pfad string, frist time.Duration) {
	ende := time.Now().Add(frist)
	for time.Now().Before(ende) {
		if !UmDateiLaeuft(ort, pfad) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// UmDatei führt aendere aus, das die Datei pfad (absolut) nur in ihren Metadaten ändert (Schlagworte der Bibliothek),
// und zieht dabei die Kennung einer liegenden Rettungskopie nach (F-703): Passte die Kopie zum Stand der Datei vor
// aendere, trägt sie danach die Kennung des neuen Stands und bleibt anwendbar. Bild, Objekte und Schwärzungen der Datei
// ändern sich durch aendere nicht, die Kopie wendet also genau dasselbe an wie zuvor (Kernprinzip 2: nichts geht
// verloren, nichts Ungeschwärztes wird sichtbar).
//
// „Nur Metadaten“ prüft UmDatei nicht am Inhalt, sondern über den Stand: aendere bekommt unveraendert und ruft es
// unmittelbar vor dem Lesen der Datei auf; es meldet, ob Größe und Änderungszeit noch die der Kennung vor aendere sind.
// Ruft aendere es nicht auf oder meldet es false (ein anderer Schreiber hat die Datei dazwischen ersetzt), läuft aendere
// trotzdem, die Kopie wird aber nicht nachgezogen (F-757). Sie gilt danach wie bisher als fremd und bleibt bis zum
// Aufräumen liegen.
//
// Die Kopie wird nur angefasst, wenn niemand die Sperre der Ausgangsdatei hält: Eine gehaltene Sperre heißt, ein anderes
// Fenster führt die Kopie. Dann läuft aendere ohne Nachziehen, und die Kopie bleibt, wie sie ist. Während aendere läuft,
// hält UmDatei selbst die Sperre, damit kein Fenster die Kopie dazwischen anwendet oder beiseitelegt, und davor schon die
// Marke <SHA>.um, auf die ein öffnendes Fenster mit WarteUmDatei wartet (F-757, F-770). Eine Kopie, die nicht zum Stand vor aendere
// passte, eine unlesbare Kopie und ein Fehler beim Schreiben der neuen Kennung lassen die Kopie unverändert. Scheitert
// aendere, bleibt die Kopie ebenfalls unberührt.
func UmDatei(ort, pfad string, aendere func(unveraendert func() bool) error) (fehler error) {
	defer errprefix.Defer("rescue", &fehler)
	// Die Marke kommt vor der Sperre (F-770): Wer die Sperre belegt sieht, sieht auch die Marke und wartet. Sie wird nach
	// der Sperre freigegeben (Defer-Reihenfolge), sonst fände ein wartendes Fenster die Sperre noch belegt. Ist sie
	// belegt (ein zweites UmDatei) oder nicht anzulegen, läuft UmDatei ohne Marke weiter.
	if m, err := sperreAuf(ort, pfad, umMarkeName(pfad)); err == nil {
		defer func() { fehler = errors.Join(fehler, m.frei()) }()
	}
	sp, err := sperreNehmen(ort, pfad)
	if err != nil {
		// Belegt (anderes Fenster führt die Kopie), nicht prüfbar oder ohne flock: nichts nachziehen.
		return aendere(func() bool { return false })
	}
	defer func() { fehler = errors.Join(fehler, sp.frei()) }()
	kp, err := Lies(ort, pfad)
	if err != nil {
		return aendere(func() bool { return false }) // keine oder unlesbare Kopie: nichts nachzuziehen
	}
	vorher, err := KennungVon(pfad)
	passte := err == nil && kp.Datei.Gleich(vorher)
	bestaetigt := false
	unveraendert := func() bool {
		st, err := os.Lstat(pfad)
		bestaetigt = passte && err == nil && st.Mode().IsRegular() && st.Size() == vorher.Groesse && st.ModTime().Equal(vorher.Zeit)
		return bestaetigt
	}
	if err := aendere(unveraendert); err != nil {
		return err
	}
	if !passte || !bestaetigt {
		return nil
	}
	nachher, err := KennungVon(pfad)
	if err != nil {
		return nil // die Datei ist weg oder unlesbar: die Kopie bleibt mit der alten Kennung liegen
	}
	kp.Datei = nachher
	// Gelingt das Schreiben nicht, ist die Änderung von aendere trotzdem geschehen: kein Fehler, die alte Kopie bleibt.
	_ = Schreibe(ort, *kp)
	return nil
}
