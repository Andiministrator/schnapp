// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"errors"
	"io/fs"
	"os"
	"strings"

	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// fehlerGrund liefert den Grund eines Fehlers für die Fußzeile (F-623): Die eigenen Fehlerwerte des Pakets settings
// (und der Tafel) werden über errors.Is auf übersetzte Texte abgebildet; die Pakete selbst liefern weiter englische
// Texte mit Paketpräfix (ADR-0019 Nachträge 2 und 3). Alles andere, vor allem Fehler des Betriebssystems, bleibt
// englisch und verliert das Paketpräfix; bei einem Dateifehler bleibt nur die Ursache, nie der Pfad (dateiUrsache,
// F-661). Wo die Übersetzung bewusst endet, sagt ADR-0019 Nachtrag 4.
//
// Zusatz (C-5): Erhalten bleibt nur, was im Text HINTER dem Fehlerwert steht (etwa „: capture.seam“ bei
// ErrUngueltig). Angaben davor ("settings: styles.ini line 7: %w") und weitere Zweige eines errors.Join fallen weg bzw.
// stehen roh dahinter. Das Paket settings hüllt seine Fehlerwerte nur in der Form „%w: Zusatz“ (Präfix davor ist
// nur „settings: “); wer eine andere Hülle einführt, passt diese Funktion und ihren Test an.
func fehlerGrund(err error) string {
	text := ohnePaketpraefix(err.Error())
	for _, f := range []struct {
		wert  error
		grund string
	}{
		{settings.ErrZuGross, i18n.T("the settings file is larger than 64 KiB")},
		{settings.ErrKeineDatei, i18n.T("the settings file is not a regular file")},
		{settings.ErrKeinUTF8, i18n.T("the settings file is not valid UTF-8")},
		{settings.ErrUngueltig, i18n.T("a setting has an invalid value")},
		{settings.ErrVerweisLeer, i18n.T("the settings file is a link to nothing")},
		{settings.ErrKeinHeim, i18n.T("the home directory of the user is unknown")},
		{errKeinOrt, i18n.T("the settings file cannot be located")},
		{settings.ErrStileZuGross, i18n.T("the styles file would be larger than 64 KiB")},
		{settings.ErrStileKeineDatei, i18n.T("the styles file is not a regular file")},
		{settings.ErrStileVerweis, i18n.T("the styles file is a link to nothing")},
		{settings.ErrStileUnlesbar, i18n.T("the existing styles file cannot be read, so it is left as it is")},
		{settings.ErrStileSicherung, i18n.T("the existing styles file has parts that cannot be read and styles.ini.bak already exists, so it is left as it is")},
		{settings.ErrStileGesperrt, i18n.T("the styles file is locked by another window")},
		{settings.ErrFarbenUnlesbar, i18n.T("the color swatches file cannot be read, so it is left as it is")},
		{settings.ErrFarbenGesperrt, i18n.T("the color swatches file is locked by another window")},
		{settings.ErrFarbenSicherung, i18n.T("the color swatches file has parts that cannot be read and colors.ini.bak already exists, so it is left as it is")},
		// Bibliothek (volle Ansicht: Schlagworte, Papierkorb; Sitzungs-QA 24.11. abends)
		{library.ErrGeaendert, i18n.T("the capture was changed in another window or program in the meantime")},
		{library.ErrPfad, i18n.T("the capture is not inside the library folder")},
		{library.ErrFremd, i18n.T("the file is not a Schnapp capture")},
		{library.ErrAnderesDateisystem, i18n.T("the trash is on another file system")},
	} {
		if errors.Is(err, f.wert) {
			// Was hinter dem Fehlerwert steht (etwa der Schlüssel bei ErrUngueltig), bleibt als Zusatz stehen.
			if _, rest, ok := strings.Cut(text, ohnePaketpraefix(f.wert.Error())); ok && rest != "" { // Präfixe auf beiden Seiten weg
				return f.grund + rest
			}
			return f.grund
		}
	}
	if u, ok := dateiUrsache(err); ok {
		return u
	}
	return text
}

// dateiUrsache liefert bei einem Dateifehler (*fs.PathError, *os.LinkError) nur die Ursache des Betriebssystems, etwa
// „permission denied“: Der Pfad enthielte Ordner- und Benutzernamen und temporäre Namen, und das Fenster wird oft
// fotografiert (Kernprinzip 3, A-6, F-661). Gemeinsam für fehlerGrund und exportFehlerText.
func dateiUrsache(err error) (string, bool) {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err.Error(), true
	}
	var le *os.LinkError
	if errors.As(err, &le) {
		return le.Err.Error(), true
	}
	return "", false
}
