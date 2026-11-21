// SPDX-License-Identifier: GPL-3.0-or-later

package probemodul

// Ohne Import des Pakets: eine Variable namens i18n darf keinen Treffer erzeugen.
type fremd struct{}

func (fremd) T(s string) string { return s }

func ohneImport() string {
	var i18n fremd
	return i18n.T("Not a UI text")
}
