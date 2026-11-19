// SPDX-License-Identifier: GPL-3.0-or-later

package probemodul

// Ohne Import des Pakets: eine Variable namens sprache darf keinen Treffer erzeugen.
type fremd struct{}

func (fremd) T(s string) string { return s }

func ohneImport() string {
	var sprache fremd
	return sprache.T("Not a UI text")
}
