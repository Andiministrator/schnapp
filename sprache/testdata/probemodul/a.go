// SPDX-License-Identifier: GPL-3.0-or-later

// Probe für TestPruefungFindetLuecken; liegt unter testdata und wird nie gebaut.
package probemodul

import "github.com/Andiministrator/schnapp/sprache"

func texte(name string, n int) []string {
	return []string{
		sprache.T("Save"),
		sprache.T("Missing"),
		sprache.TK("menu", "Open"),
		sprache.N("%d image", "%d images", n),
		sprache.N("%d file", "%d files", n),
		sprache.T(name), // kein Literal: nicht prüfbar
		sprache.T(`Close`),
		sprache.Ausgangssprache,
	}
}
