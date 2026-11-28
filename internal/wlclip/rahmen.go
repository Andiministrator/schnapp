// SPDX-License-Identifier: GPL-3.0-or-later

package wlclip

import (
	"encoding/binary"
	"errors"
)

// ErrRahmen meldet einen Rahmen aus Packe, der nicht stimmt.
var ErrRahmen = errors.New("wlclip: Rahmen der Objekte ungültig")

// Packe fasst die beiden Teile einer Objekt-Kopie (ArtObjekte, ADR-0045 Punkt 2) für die Standardeingabe des Halters
// zusammen: 8 Byte Länge des SVG (big endian), das SVG, dann das PNG bis zum Ende.
func Packe(svg, png []byte) []byte {
	b := make([]byte, 8, 8+len(svg)+len(png))
	binary.BigEndian.PutUint64(b, uint64(len(svg)))
	return append(append(b, svg...), png...)
}

// Entpacke ist die Umkehrung von Packe. Beide Teile müssen Inhalt haben.
func Entpacke(daten []byte) (svg, png []byte, err error) {
	if len(daten) < 8 {
		return nil, nil, ErrRahmen
	}
	n := binary.BigEndian.Uint64(daten)
	rest := daten[8:]
	if n == 0 || n >= uint64(len(rest)) {
		return nil, nil, ErrRahmen
	}
	return rest[:n], rest[n:], nil
}
