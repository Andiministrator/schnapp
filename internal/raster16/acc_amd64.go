// SPDX-License-Identifier: GPL-3.0-or-later

//go:build !appengine && gc && !noasm

package raster16

import "golang.org/x/sys/cpu"

// vierer: x/image/vector akkumuliert unter genau dieser Bedingung mit SSE4.1 in Viererblöcken (acc_amd64.go dort);
// raster16 folgt derselben Summenfolge, damit die Maske bitgleich bleibt.
var vierer = cpu.X86.HasSSE41
