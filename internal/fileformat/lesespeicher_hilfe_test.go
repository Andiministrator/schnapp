// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import "runtime"

// runtimeStats liest die insgesamt angelegten Bytes (kumulativ) aus der Laufzeit.
type runtimeStats struct{ gesamt uint64 }

func (r *runtimeStats) lies() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	r.gesamt = m.TotalAlloc
}
