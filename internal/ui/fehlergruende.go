// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"errors"
	"strings"

	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// fehlerGrund liefert den Grund eines Fehlers für die Fußzeile (F-623): Die eigenen Fehlerwerte des Pakets settings
// (und der Tafel) werden über errors.Is auf übersetzte Texte abgebildet; die Pakete selbst liefern weiter englische
// Texte mit Paketpräfix (ADR-0019 Nachträge 2 und 3). Alles andere, vor allem Fehler des Betriebssystems, bleibt
// englisch und verliert nur das Paketpräfix. Ein Zusatz hinter dem Fehlerwert (etwa „: capture.seam“) bleibt stehen.
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
	} {
		if errors.Is(err, f.wert) {
			// Was hinter dem Fehlerwert steht (etwa der Schlüssel bei ErrUngueltig), bleibt als Zusatz stehen.
			if _, rest, ok := strings.Cut(text, f.wert.Error()); ok && rest != "" {
				return f.grund + rest
			}
			return f.grund
		}
	}
	return text
}
