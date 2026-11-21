// SPDX-License-Identifier: GPL-3.0-or-later

//go:build !appengine && gc && !noasm

package raster16

// vierer: x/image/vector akkumuliert unter dieser Build-Bedingung auf amd64 mit SSE4.1 in Viererblöcken (acc_amd64.go
// dort, CPUID direkt abgefragt); raster16 rechnet dieselbe Summenfolge in Go nach, damit die Maske bitgleich bleibt.
// Fest eingeschaltet statt über golang.org/x/sys/cpu: Dessen Abfrage folgt GODEBUG=cpu.sse41=off, die von vector nicht
// (Sitzungs-QA Sprint 10, K-3). Nur auf amd64-CPUs ohne SSE4.1 (Intel vor 2008, AMD vor 2011) weicht die Maske daher
// in einzelnen Pixeln um eine Stufe von vector ab; dort schlagen die Vergleichstests fehl, das Bild ist aber dasselbe
// wie auf allen anderen amd64-Rechnern. Die Nachbildung selbst braucht kein SSE4.1.
const vierer = true
