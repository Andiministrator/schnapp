// SPDX-License-Identifier: GPL-3.0-or-later

// Package po bettet die Übersetzungen der Oberfläche ein: eine gettext-PO-Datei je Sprache, "<sprache>.po"
// (ADR-0014). Englisch ist die Ausgangssprache und hat keine Datei.
package po

import "embed"

// Dateien enthält alle PO-Dateien dieses Ordners.
//
//go:embed *.po
var Dateien embed.FS
