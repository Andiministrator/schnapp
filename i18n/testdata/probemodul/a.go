// SPDX-License-Identifier: GPL-3.0-or-later

// Probe für TestPruefungFindetLuecken; liegt unter testdata und wird nie gebaut.
package probemodul

import "github.com/Andiministrator/schnapp/i18n"

func texte(name string, n int) []string {
	return []string{
		i18n.T("Save"),
		i18n.T("Missing"),
		i18n.TK("menu", "Open"),
		i18n.N("%d image", "%d images", n),
		i18n.N("%d file", "%d files", n),
		i18n.T(name), // kein Literal: nicht prüfbar
		i18n.T(`Close`),
		i18n.Ausgangssprache,
	}
}
