// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"errors"

	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// fehlerGrund liefert den Grund eines Fehlers für die Fußzeile (F-623): Die eigenen Fehlerwerte des Pakets settings
// (und der Tafel) werden über errors.Is auf übersetzte Texte abgebildet; die Pakete selbst liefern weiter englische
// Texte mit Paketpräfix (ADR-0019 Nachträge 2 und 3). Alles andere, vor allem Fehler des Betriebssystems, bleibt
// englisch und verliert nur das Paketpräfix.
func fehlerGrund(err error) string {
	switch {
	case errors.Is(err, settings.ErrZuGross):
		return i18n.T("the settings file is larger than 64 KiB")
	case errors.Is(err, settings.ErrKeineDatei):
		return i18n.T("the settings file is not a regular file")
	case errors.Is(err, settings.ErrKeinUTF8):
		return i18n.T("the settings file is not valid UTF-8")
	case errors.Is(err, settings.ErrUngueltig):
		return i18n.T("a setting has an invalid value")
	case errors.Is(err, settings.ErrVerweisLeer):
		return i18n.T("the settings file is a link to nothing")
	case errors.Is(err, settings.ErrKeinHeim):
		return i18n.T("the home directory of the user is unknown")
	case errors.Is(err, errKeinOrt):
		return i18n.T("the settings file cannot be located")
	case errors.Is(err, settings.ErrStileZuGross):
		return i18n.T("the styles file would be larger than 64 KiB")
	case errors.Is(err, settings.ErrStileKeineDatei):
		return i18n.T("the styles file is not a regular file")
	case errors.Is(err, settings.ErrStileVerweis):
		return i18n.T("the styles file is a link to nothing")
	case errors.Is(err, settings.ErrStileUnlesbar):
		return i18n.T("the existing styles file cannot be read, so it is left as it is")
	case errors.Is(err, settings.ErrStileSicherung):
		return i18n.T("the existing styles file has parts that cannot be read and styles.ini.bak already exists, so it is left as it is")
	case errors.Is(err, settings.ErrStileGesperrt):
		return i18n.T("the styles file is locked by another window")
	}
	return ohnePaketpraefix(err.Error())
}
