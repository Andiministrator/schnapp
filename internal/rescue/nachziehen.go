// SPDX-License-Identifier: GPL-3.0-or-later

package rescue

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Andiministrator/schnapp/internal/errprefix"
)

// umMarke ist die Endung der Marke „UmDatei läuft“ neben der Sperre (F-757).
const umMarke = ".um"

// umMarkeAlt: Eine ältere Marke gilt als Rest eines abgestürzten Prozesses und hält niemanden auf.
const umMarkeAlt = time.Minute

func umMarkeName(pfad string) string { return strings.TrimSuffix(Name(pfad), ".json") + umMarke }

// WarteUmDatei wartet höchstens frist, solange UmDatei für die Ausgangsdatei pfad läuft (Marke da, Sperre gehalten, Marke
// jünger als eine Minute). Ein Fenster, das dieselbe Aufnahme öffnet, ruft es vor KennungVon und NeueSicherung auf: Es
// bekommt dann die Sperre, liest den neuen Stand und wendet die nachgezogene Kopie an, statt fremd zu werden und die
// Kopie beim nächsten Sichern beiseitezulegen (F-757). Ohne Marke kehrt es sofort zurück.
func WarteUmDatei(ort, pfad string, frist time.Duration) {
	ende := time.Now().Add(frist)
	for time.Now().Before(ende) {
		st, err := os.Lstat(filepath.Join(ort, umMarkeName(pfad)))
		if err != nil || time.Since(st.ModTime()) > umMarkeAlt {
			return
		}
		if belegt, err := sperreGehalten(filepath.Join(ort, sperrName(pfad))); err != nil || !belegt {
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
// hält UmDatei selbst die Sperre, damit kein Fenster die Kopie dazwischen anwendet oder beiseitelegt, und legt die Marke
// <SHA>.um an, auf die ein öffnendes Fenster mit WarteUmDatei wartet (F-757). Eine Kopie, die nicht zum Stand vor aendere
// passte, eine unlesbare Kopie und ein Fehler beim Schreiben der neuen Kennung lassen die Kopie unverändert. Scheitert
// aendere, bleibt die Kopie ebenfalls unberührt.
func UmDatei(ort, pfad string, aendere func(unveraendert func() bool) error) (fehler error) {
	defer errprefix.Defer("rescue", &fehler)
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
	marke := filepath.Join(ort, umMarkeName(pfad))
	if f, err := os.OpenFile(marke, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|syscall.O_NOFOLLOW, 0o600); err == nil {
		_ = f.Close()
		// Die Marke wird vor der Sperre entfernt (Defer-Reihenfolge), sonst sähe ein wartendes Fenster sie ohne Sperre.
		defer os.Remove(marke)
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
