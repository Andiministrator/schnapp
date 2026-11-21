// SPDX-License-Identifier: GPL-3.0-or-later

//go:build !amd64 || appengine || !gc || noasm

package raster16

// vierer: Außerhalb von amd64 (oder ohne Assembler) akkumuliert x/image/vector skalar.
const vierer = false
