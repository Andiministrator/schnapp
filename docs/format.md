# Schnapp working file format

**Format version: 1**

This document specifies the working file written and read by Schnapp: a PNG image that looks like a finished
screenshot in every viewer and carries all editable layers (annotations, crop and squeeze, the unmodified original) in
two private chunks. It is meant for developers who want to read, write or validate such files. It is normative; the
reference implementation is the Go package `internal/fileformat` of this repository, and
[`testdata/format/`](../testdata/format/) holds small example files that the implementation reproduces byte for byte
(`internal/fileformat/format_konform_test.go`).

The key words MUST, MUST NOT, SHOULD, SHOULD NOT and MAY are used as in RFC 2119. "Writer" means a program that
produces a working file, "reader" one that consumes it. Where this text and the implementation differ, the
implementation is authoritative and this text is a defect to be fixed.

**Stability.** Until Schnapp 1.0.0 the format may still change incompatibly; every such change raises the format
version and changes this document in the same commit. From 1.0.0 on, every release reads all earlier format versions
(ADR-0006). The format version is independent of the program version.

Contents: [1 Overview](#1-overview) · [2 PNG container](#2-png-container) · [3 Chunk `snAP`](#3-chunk-snap-svg-document)
· [4 Chunk `snAO`](#4-chunk-snao-original-pixels) · [5 SVG document](#5-svg-document) · [6 Metadata](#6-metadata)
· [7 View: crop and squeeze](#7-view-crop-and-squeeze) · [8 Objects](#8-objects) · [9 Redaction](#9-redaction)
· [10 Limits](#10-limits-and-constants) · [11 Reader rules](#11-reader-rules) · [12 Writer rules](#12-writer-rules)
· [13 What is not stored](#13-what-is-not-stored) · [14 Examples](#14-examples-and-conformance)

## 1. Overview

A working file consists of three parts:

| Part | Where | Content |
|---|---|---|
| Visible image | ordinary PNG chunks (`IHDR`, `IDAT`, ...) | the *view*: original cropped and squeezed, annotations drawn in |
| Edit data | chunk `snAP` | zlib-compressed SVG document: metadata, crop, squeeze strips, annotation objects |
| Original | chunk `snAO` | the uncropped original pixels as a complete PNG datastream |

All coordinates in the SVG are **pixels of the original** (the `snAO` image), origin top left, x to the right, y
downwards. Annotations never change original pixels. The only operation that does is redaction (section 9).

A viewer that knows nothing about Schnapp shows the visible image and ignores both private chunks. A program that
changes the visible image's pixels MUST drop both chunks, because they would no longer match (see 2.2).

## 2. PNG container

### 2.1 Structure

The file is a PNG datastream as defined by the PNG specification (W3C, ISO/IEC 15948): the 8-byte signature
`89 50 4E 47 0D 0A 1A 0A`, then chunks (length, type, data, CRC-32), ending with `IEND`.

A working file written by Schnapp contains exactly these chunks, in this order:

```
IHDR  IDAT...  snAP  snAO  IEND
```

* The two private chunks MUST appear exactly once each and MUST precede `IEND`. A writer MUST place them after the last
  `IDAT` and SHOULD place `snAP` before `snAO`. Readers MUST accept them anywhere before `IEND` (the reference reader
  does not look at their position).
* The writer MUST NOT add other ancillary chunks that carry information about the machine, the user or the program
  (`tEXt`, `zTXt`, `iTXt`, `eXIf`, `tIME`, ...); the reference writer emits none (section 13).
* Nothing may follow `IEND`. A file with data after `IEND` that contains Schnapp chunks is damaged; a plain PNG
  (no Schnapp chunks) with trailing data is read like any other PNG.

### 2.2 Chunk names

| Name | Meaning | PNG property bits |
|---|---|---|
| `snAP` | SVG edit data | ancillary (`s` lower case), private (`n` lower case), reserved bit valid (`A` upper case), **unsafe to copy** (`P` upper case) |
| `snAO` | original pixels | the same |

"Unsafe to copy" is deliberate: a program that modifies the visible pixels without understanding the chunks MUST
NOT copy them to its output, otherwise a later reader would lay stale layers over a foreign edit.

### 2.3 Visible image

The visible image is any valid PNG image. Schnapp writes the view (section 7) with all annotations rendered, 8 bits per
channel. A writer SHOULD make its size equal the view size computed in section 7.3. A reader MUST NOT depend on the
visible image to reconstruct the document: the original comes from `snAO`, everything else from `snAP`. (The
reference reader does not compare the visible image with the view; it is only used by programs that open the file as a
plain picture.)

### 2.4 Plain PNG and damaged files

* A PNG with **neither** chunk is a plain image ("foreign file"); a reader treats its visible image as the original of a
  new document.
* A PNG with exactly one of the chunks, with a chunk twice, or with data after `IEND` together with Schnapp chunks is
  **damaged**. Readers MUST NOT guess the missing part. The visible image is still available as an ordinary picture.

## 3. Chunk `snAP`: SVG document

The chunk data is one **zlib stream** (RFC 1950: two-byte header, DEFLATE data, Adler-32 trailer, not raw DEFLATE)
whose decompressed content is the SVG document of section 5, encoded in UTF-8. The reference writer uses compression
level 9 (best compression), which yields the header bytes `78 DA`; readers MUST accept any valid zlib stream.

## 4. Chunk `snAO`: original pixels

The chunk data is a **complete, self-contained PNG datastream** (signature, `IHDR`, `IDAT`..., `IEND`) holding the
unchanged original image, without any Schnapp chunks of its own. It is the PNG as produced by an ordinary encoder; no
extra framing and no re-compression of the chunk. The reference writer encodes the original as a non-interlaced
8-bit image; readers SHOULD accept every PNG colour type and bit depth a standard decoder supports and MUST convert the
pixels to non-premultiplied 8-bit RGBA.

Its width and height MUST equal the `width` and `height` of the SVG root (section 5.1); otherwise the file is damaged.
The size limits of section 10 apply.

## 5. SVG document

The document is XML 1.0 in UTF-8, wellformed, using the SVG namespace as default namespace and one additional
namespace for Schnapp's own attributes and elements, written with the prefix `s`:

| Namespace | Prefix | Use |
|---|---|---|
| `http://www.w3.org/2000/svg` | (default) | root and presentation elements |
| `urn:x-schnapp:1` | `s` | edit data; the name identifies the format, nothing else |

Schnapp's data is carried only by attributes and elements in the Schnapp namespace (written below as `s:name`). SVG
presentation elements and attributes are an informative picture of the same content so that a generic SVG tool shows
something sensible; **a reader MUST ignore them** and MUST NOT resolve any reference in them (`href`, `url(...)`,
entities).

### 5.1 Skeleton

```xml
<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" xmlns:s="urn:x-schnapp:1" s:format="1"
     width="64" height="48" viewBox="0 0 64 48">
  <metadata>
    <s:meta s:autor="..." s:erstellt="..." s:quelle="..." s:bearbeitet="true">
      <s:schlagwort>keyword</s:schlagwort>
    </s:meta>
    <s:ansicht s:zuschnitt="0 0 64 48">
      <s:streifen s:zeilen="true" s:von="20" s:bis="25"/>
    </s:ansicht>
  </metadata>
  <image x="0" y="0" width="64" height="48" href="schnapp:original"/>
  <g s:ebene="objekte">
    <!-- objects, bottom to top -->
  </g>
</svg>
```

(The reference writer emits no whitespace between elements; the example is indented for reading.)

Root element `svg`:

| Attribute | Value | Required |
|---|---|---|
| `s:format` | format version, a decimal integer; this document describes `1` | MUST |
| `width`, `height` | size of the original in pixels, plain decimal integers without unit | MUST |
| `viewBox` | `0 0 <width> <height>` | writer MUST, reader ignores |
| `xmlns`, `xmlns:s` | the namespaces above | writer MUST |

* `image` with `href="schnapp:original"` marks where the original sits. It has no data of its own: the pixels are in
  `snAO`. Writers MUST include it (`x="0" y="0"` and the original's `width`/`height`); readers MUST ignore it and
  MUST NOT try to load anything from `href`.
* `g` with the attribute `s:ebene` (value `objekte`) is the object layer (section 8). Writers MUST emit it even when it is empty.
* The names of the `s:` items are German identifiers (`ansicht` = view, `zuschnitt` = crop, `streifen` = strip,
  `zeilen` = rows, `von`/`bis` = from/to, `ebene` = layer, `typ` = type, ...). They are part of the format and are not
  translated.

### 5.2 Value syntax

| Kind | Syntax | Notes |
|---|---|---|
| number | decimal floating point as written by the shortest round-trip form, e.g. `2.5`, `-1`, `1e-05`, `1e+06` | characters only digits, `.`, `-`, `e`, `E`, and `+` directly after the exponent mark; readers MUST accept any such decimal float, including exponent notation, and MUST reject everything else (a leading `+`, `NaN`, `Inf`, hexadecimal floats, `_`, white space) |
| integer | optional minus sign, decimal digits | no `+`, no white space, no decimal point |
| boolean | `true` or `false` | a reader treats every value other than `true` as false |
| point | `x,y` (two numbers, one comma, no spaces) | |
| area | `x0,y0 x1,y1` (two points, one space) | `x0 <= x1` and `y0 <= y1` |
| colour | `#rrggbbaa`, exactly 8 hexadecimal digits, no white space | red, green, blue, alpha; **non-premultiplied**, 8 bits each; `aa` = `00` transparent, `ff` opaque; writers use lower case, readers accept both |
| stroke | `<colour> <width>` (one space) | `width` is a number in original pixels |
| rectangle | `x0 y0 x1 y1` (four integers separated by single spaces, nothing before, between or after) | half-open: `[x0,x1) x [y0,y1)` |

Colours carry no colour-space tag; pixel values are used as stored. Units are always pixels of the original.

Presentation attributes (`fill`, `stroke`, `fill-opacity`, ...) split a colour into `#rrggbb` and an opacity; they are
informative only.

## 6. Metadata

`s:meta` (inside `metadata`) is optional for readers; writers MUST emit it. Attributes are optional unless noted:

| Attribute | Value |
|---|---|
| `s:autor` | free text; written only if the user configured an author |
| `s:erstellt` | creation time, RFC 3339, UTC with second resolution, e.g. `2026-01-02T03:04:05Z`; omitted if unknown |
| `s:quelle` | name of the application whose window was captured, if known (not the program that wrote the file); omitted if unknown |
| `s:bearbeitet` | `true` if the file was edited after capture; writers MUST emit it |

Child elements `s:schlagwort` hold one keyword each as character data (leading and trailing white space is part of
the keyword). Order is preserved.

## 7. View: crop and squeeze

The *view* is what the visible image shows. It is derived from the original in two steps: **crop**, then **squeeze**.

### 7.1 Element `s:ansicht`

`s:ansicht` is REQUIRED (a file without it is damaged). Attribute:

| Attribute | Value |
|---|---|
| `s:zuschnitt` | rectangle `x0 y0 x1 y1` of the original that stays visible. The whole image is `0 0 <width> <height>`. |

The rectangle MUST be non-empty and lie within the original. A reader canonicalises reversed corners (`x0 > x1`);
writers MUST write the smaller value first.

### 7.2 Element `s:streifen` (squeeze strips)

Zero or more `s:streifen` children of `s:ansicht` remove parts of the cropped image ("squeeze"). Schnapp draws a zigzag
edge at every seam in the view; that edge is a rendering, not stored data.

| Attribute | Value |
|---|---|
| `s:zeilen` | `true`: the strip removes rows `[s:von, s:bis)` (the image becomes lower); `false`: it removes columns (the image becomes narrower) |
| `s:von`, `s:bis` | integers in pixels of the original, `0 <= von < bis <= height` (rows) or `<= width` (columns) |

* Strips of the same axis MUST NOT overlap; touching (`bis` of one equal to `von` of the next) is allowed.
* Writers MUST write column strips first, then row strips, each ordered by ascending `s:von`. Readers MUST accept
  any order and sort.
* A strip may reach beyond the crop rectangle; only the part inside the crop has an effect.
* The view MUST NOT become empty in either dimension.
* At most `max_stripes` strips, and `(row strips + 1) x (column strips + 1) <= max_tiles` (section 10).

### 7.3 View geometry

With crop `[cx0,cx1) x [cy0,cy1)`:

* view width = `(cx1 - cx0)` minus the total length of all column strips clipped to `[cx0,cx1)`;
* view height = `(cy1 - cy0)` minus the total length of all row strips clipped to `[cy0,cy1)`.

The view pixel `(vx, vy)` is the original pixel `(ox, oy)` where `ox` runs through the columns of the crop that lie in
no column strip, in order (likewise `oy` for rows). Objects keep their original coordinates. When an object is drawn
into the view, a point is mapped to `ox - cx0 - (removed columns before ox)`; a point inside a strip falls on the seam
(the strip's edge).

## 8. Objects

All objects are children of `g s:ebene="objekte"` (readers accept an object element at any place outside another
object). Document order is stacking order: the first object is at the bottom, the last on top. Identifiers are not
stored; a reader assigns 1, 2, 3, ... in document order.

An object is the element that has the attribute `s:typ`. The value selects the type; an empty value is invalid (the file is damaged), and the root, `s:meta`, `s:ansicht`, `s:streifen` and `s:schlagwort` MUST NOT carry `s:typ` (a reader treats the file as damaged); **the `s:` attributes are the
object's complete data**. The element's children (`path`, `rect`, `text`, `circle`, ...) are presentation and are
ignored on reading. Coordinates in objects are original pixels and may lie partly outside the image, but each
coordinate MUST satisfy `|value| <= max_coordinate`.

Common rules: all attributes listed as required MUST be present; a missing or invalid attribute makes the file
damaged. Sizes and widths are numbers with `0 <= value <= max_size`.

### 8.1 `"pfeil"` arrow

A quadratic Bézier curve from `A` to `B` with control point `K`; the tip is at `B`. A straight arrow has `K` at the
midpoint of `A` and `B`.

| Attribute | Value |
|---|---|
| `s:a`, `s:b`, `s:k` | points: start, tip, control point |
| `s:strich` | stroke: colour and width of the line and the head |

Informative: the head is a filled triangle at `B`, of length `max(8, 4 x width)` and the same base width, aligned with
the curve's tangent at `B` (direction `B - K`, or `B - A` if `K` equals `B`).

### 8.2 `"kasten"` box

| Attribute | Value |
|---|---|
| `s:bereich` | area of the box; `x0 <= x1`, `y0 <= y1` |
| `s:strich` | stroke of the outline, centred on the edge |
| `s:fuellung` | fill colour; alpha `00` means no fill |
| `s:eckradius` | corner radius, number `>= 0` |

Informative: the effective radius is `min(radius, min(w, h) / 2)`, and below 0.5 pixel the corners are square; the
rounding is circular, never elliptic.

### 8.3 `"text"` text

| Attribute | Value |
|---|---|
| `s:ort` | point: top-left corner of the text block (before padding) |
| `s:inhalt` | the text, see below |
| `s:grad` | font size in original pixels, `0 < grad <= max_font_size` |
| `s:farbe` | text colour |
| `s:hintergrund` | colour of the label behind the text; alpha `00` means none |
| `s:umbruch` | wrap width, number `>= 0`; `0` = no wrapping. Stored, but **not applied** by the current implementation |

`s:inhalt` is the text as typed. Line feed (U+000A) separates lines. The value MUST be valid UTF-8 with at most
`max_text_chars` characters (Unicode scalar values) and MUST NOT contain control characters below U+0020 other than
tab, line feed and carriage return, nor U+FFFE or U+FFFF. Writers MUST escape line feed, carriage return and tab as
character references (`&#xA;`, `&#xD;`, `&#x9;`) so that XML attribute-value normalisation keeps them.

Informative: font Go Regular (family "Go"); the first baseline lies `grad` below `ort.y`, each further line
`1.3 x grad` lower; a tab is shown as four spaces; the background label spans the measured text ink plus
`0.25 x grad` padding on every side.

### 8.4 `"nummer"` step number

A filled circle with a number in it.

| Attribute | Value |
|---|---|
| `s:mitte` | point: centre of the circle |
| `s:wert` | integer, `0 <= wert <= max_number` |
| `s:radius` | circle radius, `0 < radius <= max_font_size` |
| `s:farbe` | fill colour of the circle |

Informative: the number is drawn in bold white, centred, with font size `1.2 x radius`.

### 8.5 Unknown object types

A reader that meets an unknown `s:typ` MUST NOT silently drop the object (that would lose data on the next save). It
treats the file as damaged and offers the visible image instead (section 11).

## 9. Redaction

Redacting (blacking out or pixelating) is **not** an object. It overwrites pixels of the original, and the result is
what `snAO` holds; the visible image is derived from the same pixels. After a redaction no part of the file, neither
the visible image, `snAO` nor `snAP`, contains the covered content, and no undo information is stored.

Consequences for implementations:

* A writer MUST write redacted pixels, never a layer that merely hides them.
* A pixelation SHOULD NOT be invertible from what remains. Block means, even with random noise added per block, can
  be computed back for small and medium text. The reference editor therefore fills every block with the mean of the
  *whole* redacted area, shifted by a random offset per block (the same offset for R, G and B), so that no block
  carries information about the pixels it covers. The example `redaction` uses plain block means so that it can be
  generated reproducibly, and is not a model for a real redaction.
* A reader has nothing to interpret: a redaction is indistinguishable from ordinary pixels. Programs MUST NOT try to
  reconstruct covered content.

Crop and squeeze are the opposite: they are non-destructive. The uncropped original, including everything cropped or
squeezed away, stays in `snAO`. **Whoever passes a working file on passes the whole original with it.** Schnapp's
exports (flat PNG/JPG, SVG) contain only the visible part.

## 10. Limits and constants

Readers MUST reject a file that exceeds a limit and SHOULD reject it before allocating memory for the excess (no
allocation of an announced size in advance; memory grows with the bytes actually read). Writers MUST stay within the
limits. The table is machine-checked against the implementation by `TestFormatSpezifikationKonstanten`; a change to a
value here and to the code must happen in the same commit.

| Key | Value | Meaning |
|---|---|---|
| `format_version` | 1 | value of `s:format` written by this version |
| `namespace` | `urn:x-schnapp:1` | Schnapp XML namespace |
| `chunk_svg` | `snAP` | chunk with the compressed SVG |
| `chunk_orig` | `snAO` | chunk with the original PNG |
| `original_ref` | `schnapp:original` | `href` of the `image` element |
| `max_side` | 32768 | largest width or height of the original (2^15) |
| `max_pixels` | 268435456 | largest area of the original (2^28; 1 GiB as 8-bit RGBA) |
| `max_file_bytes` | 1073741824 | largest file (1 GiB) |
| `max_chunk_bytes` | 536870912 | largest single chunk (512 MiB) |
| `max_svg_bytes` | 67108864 | largest decompressed SVG (64 MiB) |
| `max_svg_elements` | 200000 | most XML elements |
| `max_svg_depth` | 64 | deepest element nesting |
| `max_objects` | 10000 | most objects |
| `max_text_chars` | 10000 | most characters in one text object |
| `max_stripes` | 1024 | most squeeze strips |
| `max_tiles` | 4096 | most tiles: (row strips + 1) x (column strips + 1) |
| `max_coordinate` | 1000000 | largest absolute value of a coordinate |
| `max_size` | 100000 | largest width, radius or corner radius |
| `max_font_size` | 4096 | largest `s:grad` and `s:radius` |
| `max_number` | 1000000 | largest `s:wert` |

Original width and height are at least 1.

## 11. Reader rules

1. **Verify the container**: signature, chunk lengths and CRCs. Reject on any mismatch (damaged).
2. **Version.** Read `s:format` first. A version greater than the highest one the reader supports MUST be rejected as
   "newer"; the reader MUST NOT interpret such a file (the layout may differ in any respect) and SHOULD offer the
   visible image as a plain picture; it MUST NOT save over the file, or the newer layers would be lost. A version
   that is missing, not an integer, or less than 1 is damaged. Older supported versions MUST be read. The version is
   decided before any other content is judged, so a newer file that also contains an object type unknown to the
   reader is reported as newer, not as damaged.
3. **Unknown content.** Attributes or elements that are not defined here and are not an object (`s:typ`) MUST be
   ignored, whatever their namespace; so are the presentation elements and attributes. Position matters for two elements: `s:streifen` counts only as a direct child of `s:ansicht`, and `s:schlagwort` only as a direct child of `s:meta`; anywhere else they are unknown content and have no effect (they are neither data nor damage). Readers MUST NOT accept them elsewhere, so that all readers see the same strips and keywords. Unknown *object types* are not
   ignored (8.5). Unknown content is not preserved when the reader saves the file.
4. **Strictness.** A required attribute that is missing, a value that does not parse, a value outside the ranges of
   sections 7, 8 and 10, a rectangle outside the original, overlapping strips, a view that would be
   empty, an `s:ansicht` that is missing, or an original whose size differs from the root's `width`/`height` makes
   the file **damaged**. A reader MUST apply the same validation whether it loads pixels or only reads metadata.
5. **Safety.** Parse XML strictly. Do not resolve external or internal entities (only the five predefined entities
   and character references occur), do not fetch anything, do not follow `href`. Enforce the limits of section 10
   while reading, including the element count and depth.
6. **Damaged or newer file**: still open the visible image as a plain picture, and do not write back to the file.
7. **Metadata only** (library index): a reader MAY read only `snAP`, `IHDR` and the head of `snAO` without decoding any
   pixels, with the same decisions as above.

## 12. Writer rules

1. Write the chunks of section 2 and the document of section 5; use the formats of section 5.2 (lower-case colours,
   points without spaces, any decimal number form that round-trips; the reference writer uses Go's shortest `%g`
   form, e.g. `1e+06`).
2. Emit `s:format`, `viewBox`, the `image` placeholder, `s:ansicht` with `s:zuschnitt` and the object layer even when
   there is nothing in them.
3. Write redacted pixels (section 9).
4. Do not add anything that identifies the machine or user (section 13).
5. Write objects in stacking order, bottom first.
6. A program that stores a changed view of the file SHOULD rewrite all of it (visible image, `snAP`, `snAO`) rather
   than patch single chunks.

## 13. What is not stored

* No file-system paths, no user, host or domain names, no program or version names of the writer, no time stamps other
  than the optional `s:erstellt`, no GPS or device data. The namespace string and the chunk names identify the
  format, not a person or machine.
* No `tEXt`/`iTXt`/`zTXt`/`eXIf`/`tIME` chunks.
* `s:autor` is written only if the user set an author; `s:quelle` names the application whose window was captured
  and only when known.
* No history: undo information, the rescue copy (a separate file in the user's state directory) and the object
  identifiers are not part of the file.
* Exports are different files: a flat PNG or JPG carries only pixels, and the SVG export contains no `s:` attributes
  and no namespace of Schnapp at all.

## 14. Examples and conformance

The directory [`testdata/format/`](../testdata/format/) contains synthetic examples (64 x 48 pixels, `minimal` 32 x 24; generated, no real
screenshots). For each there is the working file `NAME.png` and the decompressed `snAP` document `NAME.snap.svg`:

| Name | Shows |
|---|---|
| `minimal` | smallest valid file: no objects, no metadata, whole image visible |
| `objects` | one object of each type, full metadata with keywords |
| `view` | crop plus one row strip and one column strip, one box |
| `redaction` | one filled and one pixelated redaction, applied to the pixels |

In the examples the visible image is the view *without* annotations (annotation rendering depends on a font
rasteriser and is not part of the format).

`go test ./internal/fileformat` checks that

* the writer reproduces every example byte for byte (`-update` rewrites them after an intended change);
* the reader reads every example with the expected content, and rejects the damaged variants of section 11;
* the constants of section 10 and the format version equal those of the code, and every `s:` item the writer emits is
  described in this document.

An implementation is conformant if it reads all examples with the same result and, as a writer, produces files that
the reference reader accepts and that follow sections 2 to 9.
