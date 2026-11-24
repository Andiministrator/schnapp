// SPDX-License-Identifier: GPL-3.0-or-later

package rescue

import (
	"errors"

	"github.com/Andiministrator/schnapp/internal/errprefix"
)

// UmDatei führt aendere aus, das die Datei pfad (absolut) nur in ihren Metadaten ändert (Schlagworte der Bibliothek),
// und zieht dabei die Kennung einer liegenden Rettungskopie nach (F-703): Passte die Kopie zum Stand der Datei vor
// aendere, trägt sie danach die Kennung des neuen Stands und bleibt anwendbar. Bild, Objekte und Schwärzungen der Datei
// ändern sich durch aendere nicht, die Kopie wendet also genau dasselbe an wie zuvor (Kernprinzip 2: nichts geht
// verloren, nichts Ungeschwärztes wird sichtbar).
//
// Die Kopie wird nur angefasst, wenn niemand die Sperre der Ausgangsdatei hält: Eine gehaltene Sperre heißt, ein anderes
// Fenster führt die Kopie. Dann läuft aendere ohne Nachziehen, und die Kopie bleibt, wie sie ist. Während aendere läuft,
// hält UmDatei selbst die Sperre, damit kein Fenster die Kopie dazwischen anwendet oder beiseitelegt. Eine Kopie, die
// nicht zum Stand vor aendere passte, eine unlesbare Kopie und ein Fehler beim Schreiben der neuen Kennung lassen die
// Kopie unverändert: Sie gilt danach wie bisher als fremd und bleibt bis zum Aufräumen liegen. Scheitert aendere, bleibt
// die Kopie ebenfalls unberührt.
func UmDatei(ort, pfad string, aendere func() error) (fehler error) {
	defer errprefix.Defer("rescue", &fehler)
	sp, err := sperreNehmen(ort, pfad)
	if err != nil {
		// Belegt (anderes Fenster führt die Kopie), nicht prüfbar oder ohne flock: nichts nachziehen.
		return aendere()
	}
	defer func() { fehler = errors.Join(fehler, sp.frei()) }()
	kp, err := Lies(ort, pfad)
	if err != nil {
		return aendere() // keine oder unlesbare Kopie: nichts nachzuziehen
	}
	vorher, err := KennungVon(pfad)
	passte := err == nil && kp.Datei.Gleich(vorher)
	if err := aendere(); err != nil {
		return err
	}
	if !passte {
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
