# Schnapp working file format

**Format version: 2**

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
    <s:ansicht s:zuschnitt="0 0 64 48" s:kante="tear">
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
| `s:format` | format version, a decimal integer; this document describes `1` and `2` (version `2` only adds the object type `lupe`, 8.7, the attribute `s:spitze` of the text, 8.3, the attributes `s:ende`/`s:anfang` of the arrow, 8.1, the attribute `s:strichart` of arrow, box and magnifier, 8.8, and the layer attribute `s:schatten`, 8.5; writers use the lowest version that carries the content, section 12) | MUST |
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

`s:ansicht` is REQUIRED (a file without it is damaged). Attributes:

| Attribute | Value |
|---|---|
| `s:zuschnitt` | rectangle `x0 y0 x1 y1` of the original that stays visible. The whole image is `0 0 <width> <height>`. |
| `s:kante` | style of the edge drawn at every squeeze seam (7.2): `tear` (torn paper), `wave` or `zigzag`. Writers SHOULD emit it; files without it remain valid version-1 files, and readers treat a missing value as `zigzag`. |

The rectangle MUST be non-empty and lie within the original. A reader canonicalises reversed corners (`x0 > x1`);
writers MUST write the smaller value first.

`s:kante` is presentation of the document, not of a single seam. If it is missing (files written before the attribute
existed) or holds any other value (including other letter case or an empty value, or a style of a later version), a
reader MUST use `zigzag` and MUST NOT treat the file as damaged. Adding the attribute did not change the format
version: a reader that does not know it ignores it (section 11, rule 3).

### 7.2 Element `s:streifen` (squeeze strips)

Zero or more `s:streifen` children of `s:ansicht` remove parts of the cropped image ("squeeze"). Schnapp draws an edge
in the style `s:kante` at every seam inside the view; that edge is a rendering, not stored data.

Informative: the edge follows a centre line `f(t)` across the seam, `t` running along it, in view pixels:
`tear` = `0.9 x (2.6 sin(2πt/71 + 0.3) + 1.6 sin(2πt/29 + 1.9) + 1.0 sin(2πt/13 + 4.1) + 0.6 sin(2πt/7.3 + 2.2)
+ 0.35 sin(2πt/3.1 + 5.7))` sampled every pixel, `wave` = `4 sin(2πt/32)` sampled every 2 pixels, `zigzag` = ±4 with
a tooth width of 16. Around the line lies a transparent gap 16 pixels wide: the image and annotations under it are not
drawn, so the visible image carries an alpha channel there (a JPEG export fills the gap with white). The edges of both
image parts look like torn paper: a light fibre fringe of about 1.6 pixels and a slight darkening, plus a fine
semi-transparent shadow of the upper or left part into the gap, built from a few semi-transparent bands of the same
shape. All sizes grow by an integer factor on large images.

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

A quadratic Bézier curve from `A` to `B` with control point `K`. A straight arrow has `K` at the midpoint of `A` and
`B`. Each end has its own head style; by default there is a filled triangle at `B` and no head at `A`. With no head at
either end the arrow is a plain line.

| Attribute | Value |
|---|---|
| `s:a`, `s:b`, `s:k` | points: start, end, control point |
| `s:strich` | stroke: colour and width of the line and the heads |
| `s:ende` | OPTIONAL, format version 2 only. Head at `B`: `triangle`, `narrow`, `notched`, `open`, `half`, `dot`, `diamond`, `square`, `bar` or `none`. Missing or any other value: `triangle`; writers omit it for `triangle` |
| `s:anfang` | OPTIONAL, format version 2 only. Head at `A`, same values. Missing or any other value: `none`; writers omit it for `none` |
| `s:strichart` | OPTIONAL, format version 2 only: line style of the line (8.8); the heads stay solid |

`s:ende` and `s:anfang` are part of format version 2: a writer emits `s:format="2"` if an arrow has a head other than
the defaults (section 12), so that a version-1 reader reports the file as newer instead of showing a different arrow. In
a file with `s:format="1"` they are unknown content (section 11, rule 3) and are ignored. The same holds for
`s:strichart` (8.8).

Informative (w = stroke width, all in original pixels): `triangle` is `3w + 5` long and `4.5w + 6` wide at its base;
`narrow` is `4w + 7` long and `3w + 4` wide; `notched` is `3.5w + 7` long and `4.5w + 7` wide with a notch cut `0.35`
of its length into the back; `open` is a chevron: the two legs of the triangle, each w thick, without the base; `half`
is the triangle with only one barb: for the unit direction `(dx, dy)` from the line to the end, the barb lies on the
side `(-dy, dx)` (with y pointing down: below an arrow that points right); `dot` is a disc of radius `1.5w + 3` centred on
the end; `diamond` is `4w + 8` long and `3w + 6` wide with its front corner on the end; `square` has a side of
`2.6w + 5`, centred on the end; `bar` is w thick and `5w + 6` long, centred on the end and across the direction. The line stops under the head, at
the point of the curve whose distance from the end equals the head's setback: the length for `triangle`, `narrow`
and `half`; for `notched` and `diamond` the point where the edges of the line meet the rear edges of the head; for
`open` `w/sin(a) + w/(2·tan(a))` with `a` the half angle at the tip; for `dot` `sqrt(r² − w²/4)`; for `square` half
its side; for `bar` `w/2`; the head's
direction is the chord from that point to the end. If the curve is shorter than a head, the line is left out.

Informative: the SVG export and the presentation in a work file draw an arrow with a head as the line (a stroke), one
polygon per head and, at an end without a head, a half disc. Head and half disc reach 1 pixel back into the line, so that
renderers that smooth each shape on its own leave no light seam; with a translucent colour these parts are drawn opaque
inside a group with `opacity`, so the overlap is covered once.

### 8.2 `"kasten"` box

| Attribute | Value |
|---|---|
| `s:bereich` | area of the box; `x0 <= x1`, `y0 <= y1` |
| `s:strich` | stroke of the outline, centred on the edge |
| `s:fuellung` | fill colour; alpha `00` means no fill |
| `s:eckradius` | corner radius, number `>= 0` |
| `s:strichart` | OPTIONAL, format version 2 only: line style of the outline (8.8) |

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
| `s:schrift` | OPTIONAL. Font family name: 1 to 128 Unicode scalar values, not only white space, no control characters (Unicode category Cc: U+0000 to U+001F, U+007F to U+009F) and none of U+FFFD, U+FFFE, U+FFFF. Missing, empty, `Go`, or not a valid name: the built-in family Go; writers omit it for Go |
| `s:stil` | OPTIONAL. Words separated by spaces from `bold`, `italic`, `underline`, `strike`; unknown words are ignored. Missing: none |
| `s:ausrichtung` | OPTIONAL. Alignment of each line: `left`, `center` or `right`. Missing or any other value: `left` |
| `s:spitze` | OPTIONAL, format version 2 only. Point (5.2): the tip of a speech bubble, in original pixels, which makes the text a callout (below). Missing: a plain text. In a file with `s:format="1"` the attribute is unknown content (section 11, rule 3) and is ignored, whatever it holds |

`s:inhalt` is the text as typed. Line feed (U+000A) separates lines. The value MUST be valid UTF-8 with at most
`max_text_chars` characters (Unicode scalar values) and MUST NOT contain control characters below U+0020 other than
tab, line feed and carriage return, nor U+FFFE or U+FFFF. Writers MUST escape line feed, carriage return and tab as
character references (`&#xA;`, `&#xD;`, `&#x9;`) so that XML attribute-value normalisation keeps them.

The three optional attributes `s:schrift`, `s:stil` and `s:ausrichtung` were added without changing the format version (like `s:kante`, 7.1): writers emit
them only when they differ from the default, a reader that does not know them ignores them (section 11, rule 3) and
shows the text in Go Regular, and no value of them makes a file damaged. The family is a name, never a path; a reader
that lacks the family sets the text in Go and SHOULD keep the name when it saves the file.

Informative: Schnapp sets the text in the named family (system fonts or the built-in Go and Go Mono) in the face
picked by `bold` and `italic`, falling back to the nearest available face of the family and to Go if the family is
missing; there is no shaping, only advance widths and pair kerning (from `GPOS`, otherwise the `kern` table),
scaled with `grad`. The first baseline lies `grad` below
`ort.y`, each further line `1.3 x grad` lower; each line is placed left, centred or right within the advance width
of the longest line; a tab is shown as four spaces. `underline` draws a bar from `0.1 x grad` below the baseline,
`strike` a bar centred `0.3 x grad` above it, both `max(1, 0.06 x grad)` thick and as wide as the line's advance.
The background label spans the measured text ink and bars plus `0.25 x grad` padding on every side. Example with all
three attributes: `testdata/format/text-style`.

Informative: the reference writer's presentation (section 5) shows the alignment the same way as Schnapp's SVG
export. A centred or right-aligned text carries `text-anchor="middle"` or `"end"`, and the `x` of each line is the
middle or the end of that line as Schnapp sets it, so that an SVG viewer with a different font keeps the alignment; a
left-aligned text has no `text-anchor` and `x` at the line start. Like all presentation, readers ignore it.

**Callout (speech bubble).** A text with `s:spitze` is a callout; there is no separate object type. The tip may lie
anywhere inside the coordinate range (inside or outside the image and the label). `s:spitze` is part of format version
2: a writer emits `s:format="2"` if the document contains a magnifier, a text with `s:spitze`, an arrow with
`s:ende`/`s:anfang`, a stroke with `s:strichart` or an object with `s:schatten` (section 12), and a
version-1 reader that meets a version-2 file reports it as newer instead of dropping the tip silently. A value that
does not parse as a point makes a version-2 file damaged (section 11, rule 4).

Informative: the callout is drawn as one polygon, the label (the rectangle of the background label) with a triangle
inserted. The triangle's base lies on the side of the label that the ray from the label centre to the tip crosses; its
centre is that crossing, moved so that the whole base lies on the side and keeps at least `0.25 x grad` from each
corner. The base is `grad` wide, at most half of the side's length minus the two corner margins. The triangle is
omitted (the polygon is the label) if the tip is inside the label or closer to it than `0.25 x grad`, or if the side
is too short for a base. The polygon is filled with the label colour (`s:hintergrund`). If that has alpha `00`, the
reference renderer draws the two legs of the triangle (base corner, tip, base corner) as a line in the text colour,
`max(1, 0.06 x grad)` wide with round ends and joins, so that the bubble still points. The presentation (section 5)
shows this as one `path`. Example: `testdata/format/callout`.

### 8.4 `"nummer"` step number

A filled circle with a number in it.

| Attribute | Value |
|---|---|
| `s:mitte` | point: centre of the circle |
| `s:wert` | integer, `0 <= wert <= max_number` |
| `s:radius` | circle radius, `0 < radius <= max_font_size` |
| `s:farbe` | fill colour of the circle |

Informative: the number is drawn in bold white, centred, with font size `1.2 x radius`.

### 8.5 Layer attributes (all object types)

Every object is a layer above the image; stacking order is document order (section 8). Five OPTIONAL attributes may
appear on the element of any object type:

| Attribute | Value |
|---|---|
| `s:name` | Name given by the user: at most 100 Unicode scalar values, no control characters (category Cc) and none of U+FFFE, U+FFFF. Missing or invalid: no name (the program shows one derived from type and content) |
| `s:ausgeblendet` | `true`: the layer is hidden. Any other value or missing: visible |
| `s:gesperrt` | `true`: the layer is locked against changes with the pointer. Any other value or missing: not locked |
| `s:deckkraft` | Layer opacity in percent, integer `0` to `100` (syntax of 5.2). Missing or invalid: `100` |
| `s:schatten` | Drop shadow, format version 2 only: `ANGLE DISTANCE SOFTNESS COLOUR`, three numbers and a colour (5.2) separated by single spaces. Missing, invalid or in a version-1 file: no shadow (see below) |

Writers emit them only when they differ from the default. The first four were added without changing the format
version (like the text attributes of 8.3): a reader that does not know them ignores them (section 11, rule 3) and shows
every object visible, unlocked and opaque. `s:schatten` belongs to format version 2 (below). No value of any of them
makes a file damaged.

A hidden layer is **kept** in the working file but drawn nowhere: not in the visible image, the exports, the
clipboard or the library preview. Hiding is not redaction; the content of a hidden object stays readable in `snAP`.
The layer opacity applies to the object as a whole (group opacity: overlapping parts of one object are not
compounded), in addition to the alpha of its colours.

Informative: for viewers of the `snAP` document the reference writer adds `display="none"` to a hidden object and
`opacity` (the percentage as a fraction) to a translucent one; the SVG export omits hidden objects and objects
with opacity 0, and writes `opacity`. Example: `testdata/format/layers`.

**Drop shadow (`s:schatten`).** `ANGLE` is the direction the light comes from, in degrees, 0 from the right and
counter-clockwise (readers reduce it to `0 <= angle < 360`); the shadow is the object's coverage moved by `DISTANCE`
away from the light, that is by `(−cos(angle)·d, +sin(angle)·d)` with y pointing down, blurred with a Gaussian of
standard deviation `SOFTNESS / 2` and filled with `COLOUR` (alpha included); it lies under the object, and the layer
opacity applies to object and shadow together. `0 <= DISTANCE <= 1000`, `0 <= SOFTNESS <= 200`, both in original pixels;
a value outside these, or a value that does not parse, means no shadow and does not make the file damaged. The attribute
belongs to format version 2: a writer emits `s:format="2"` if any object has it (section 12), and in a version-1 file it
is ignored. Informative: the reference renderer moves the coverage by whole pixels of the view and approximates the
Gaussian with three box filters; the SVG export and the presentation in a working file use a `filter` element
(`feGaussianBlur`, `feOffset`, `feFlood`, `feComposite`, `feMerge`) referenced from the object. Example:
`testdata/format/shadow`.

### 8.6 Unknown object types

A reader that meets an unknown `s:typ` MUST NOT silently drop the object (that would lose data on the next save). It
treats the file as damaged and offers the visible image instead (section 11).

### 8.7 `"lupe"` magnifier (format version 2)

An enlarged copy of a part of the image, shown in a frame of its own with a line to where it comes from. The type
exists from format version 2 on: in a file with `s:format="1"` it is an unknown object type (8.6), and the file is
damaged.

| Attribute | Value |
|---|---|
| `s:quelle` | area (5.2): the source, the part of the original that is enlarged; `x0 < x1`, `y0 < y1` |
| `s:ziel` | area: the target, where the enlarged copy is shown; `x0 < x1`, `y0 < y1` |
| `s:strich` | stroke of both frames and of the connecting line |
| `s:strichart` | OPTIONAL: line style of both frames and of the connecting line (8.8) |

Width and height of both areas MUST be greater than 0 and at most `max_size`. The magnification is not stored: it is
`target width / source width` horizontally and `target height / source height` vertically, with the source measured
**in the view** (section 7, see below). Schnapp keeps both equal; a file in which they differ is valid and is drawn
stretched.

**The content is not stored, and it is the view that is magnified.** Both areas are stored in pixels of the original,
like all objects, but the target shows the source as the view shows it, read anew from the original (`snAO`) whenever
the object is drawn. A redaction in the source therefore shows in the magnifier at once, and no part of the file keeps a
copy of the redacted content (section 9). The source is mapped into the view per axis without clamping to the crop
(`view(v) = v - crop.min - removed`, where `removed` counts the squeezed-away pixels before `v` inside the crop, and a
point inside a strip falls on its edge): strips squeezed away inside the source drop out, so the target has no empty
band for them, and what lies before or after the crop keeps its width. Only what the view shows is magnified: a target
pixel whose view pixel lies outside the view (outside the crop or the original) or whose centre lies in the transparent
gap of a seam (7.2) stays transparent, so that a magnifier never carries anything cropped or squeezed away, nor
anything the gap hides, into an export. Annotations under the source are not part of the content. The `snAP` document
carries the frames and the line as presentation, never pixels.

Drawing, bottom to top, all in the colour and width of `s:strich`:

1. the connecting line between the centres of source and target, cut off where it enters either area (from the edge of
   the source to the edge of the target), with round caps; no line if the areas overlap;
2. the frame around the source;
3. the enlarged content in the target, by **nearest neighbour** without smoothing: with `qa` the source in the view
   (both corners mapped as above), the target pixel whose centre is at `z` (original pixels) shows the view pixel
   `floor(qa.min + (z - ziel.min) x qa.size / ziel.size)`, per axis, and that is the original pixel the view shows
   there; if `qa` has no extent on an axis (the source lies inside a strip), the target stays transparent;
4. the frame around the target.

Without crop offset and strips the view is the original, and step 3 reads
`floor(quelle.min + (z - ziel.min) x quelle.size / ziel.size)`. Each frame is centred on the edge of its area and has
square outer corners, like a `kasten` with corner radius 0. The layer attributes (8.5) apply to the object as a whole.
Example: `testdata/format/magnifier`.

Informative: the SVG export embeds, inside a nested `svg` element at the target that clips to it
(`preserveAspectRatio="none"`, `image-rendering:pixelated`), the smaller of two PNG images, both with the hidden pixels
transparent: the whole view pixels under `qa`, limited to the view, with `qa` as `viewBox`; or, if it has fewer pixels
(magnification below 1), the finished content on the pixel grid of the original inside the target and the crop, with
the target as `viewBox`. A magnifier whose target lies in no visible tile carries no image. Since the export embeds up
to the source area, Schnapp counts that area in its rendering budget. An image that an earlier magnifier already
embedded (the same view pixels, or the same finished content) is written once with an `id` and repeated with a `use`
element. Schnapp refuses an SVG export whose magnifier images (each distinct image once) would hold more than four
times the pixels of the view, unless told to ignore its limits.

### 8.8 Line style (`s:strichart`, format version 2)

Arrow (8.1), box (8.2) and magnifier (8.7) MAY carry `s:strichart`: `dashed` or `dotted`. Missing or any other value:
`solid`; writers omit it for `solid`. A writer emits `s:format="2"` if any object has it (section 12); in a file with
`s:format="1"` it is unknown content (section 11, rule 3) and is ignored. The heads of an arrow and the legs of a callout
are always solid.

Informative (w = stroke width, original pixels): `dashed` has dashes `3w + 2` long with gaps of `2w + 2` and square
(butt) ends; `dotted` has discs of diameter w, `2w + 2` apart from centre to centre. The number of dashes or dots is
rounded and all lengths are scaled by the same factor, so that the pattern fits:

* an open line (the arrow's line between its heads, the magnifier's connecting line) starts and ends with a whole dash or
  a dot; a line shorter than one dash is solid, a dotted line shorter than w is one dot in its middle;
* an outline (box, magnifier frames) is fitted side by side, from corner to corner, with half a dash or a dot on each
  corner, so that every corner is covered; for rounded corners the sections run from the middle of one corner arc to the
  middle of the next.

On a square outline the horizontal dashes include the corners (like the solid stroke). Where an arrow has a head, the
line's last dash ends under the head; a dotted line has, instead of its last dot, the half disc behind the head.
Informative: the reference renderer draws a stroke solid when its pattern would have more than 16384 pieces (estimated
per stroke — the arrow's line, a box outline, each magnifier frame and its line — as length ÷ (dash + gap) + 1, per side
for square outlines); the SVG export draws the
pieces explicitly (dashes as one path, dots as circles), not with `stroke-dasharray`; the presentation in a working file
only approximates the pattern with `stroke-dasharray` on the solid shape (magnifier frames solid), which keeps `snAP`
small — readers use `s:strichart` alone.

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
| `format_version` | 2 | highest value of `s:format` read and written by this version (`2` only with a `lupe`, an `s:spitze`, an arrow head other than the defaults, an `s:strichart` or an `s:schatten`, section 12) |
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
   reader is reported as newer, not as damaged. Within a supported version, only the object types of that version
   exist: a `lupe` (8.7) in a file of version 1 is an unknown object type, and the file is damaged.
3. **Unknown content.** Attributes or elements that are not defined here and are not an object (`s:typ`) MUST be
   ignored, whatever their namespace; so are the presentation elements and attributes. Position matters for two elements: `s:streifen` counts only as a direct child of `s:ansicht`, and `s:schlagwort` only as a direct child of `s:meta`; anywhere else they are unknown content and have no effect (they are neither data nor damage). Readers MUST NOT accept them elsewhere, so that all readers see the same strips and keywords. Unknown *object types* are not
   ignored (8.6). Unknown content is not preserved when the reader saves the file.
4. **Strictness.** A required attribute that is missing, a value that does not parse, a value outside the ranges of
   sections 7, 8 and 10 (except `s:kante`, whose unknown values mean `zigzag`, 7.1), a rectangle outside the original,
   overlapping strips, a view that would be empty, an `s:ansicht` that is missing, or an original whose size differs from the root's `width`/`height` makes
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
2. Emit `s:format`, `viewBox`, the `image` placeholder, `s:ansicht` with `s:zuschnitt` (and `s:kante`, which a writer SHOULD emit, 7.1), and the object
   layer even when there is nothing in them.
3. Write redacted pixels (section 9).
4. Do not add anything that identifies the machine or user (section 13).
5. Write objects in stacking order, bottom first.
6. A program that stores a changed view of the file SHOULD rewrite all of it (visible image, `snAP`, `snAO`) rather
   than patch single chunks.
7. Write the lowest format version that carries the content: `s:format="2"` if and only if the document contains a
   `lupe` (8.7), a text with `s:spitze` (8.3), an arrow with `s:ende` or `s:anfang` (8.1) or an object with
   `s:strichart` (8.8) or `s:schatten` (8.5), otherwise `1`. A reader that knows only version 1 then still reads
   every file without them, and reports a file with one as newer instead of damaged.

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
| `view` | crop plus one row strip and one column strip, one box, edge style `wave` |
| `no-edge-style` | like `minimal`, but `s:ansicht` has no `s:kante` (a valid version-1 file written before the attribute existed; readers use `zigzag`) |
| `redaction` | one filled and one pixelated redaction, applied to the pixels |
| `text-style` | one text with all optional text attributes: a family that is not installed, `bold italic underline strike`, `center` |
| `layers` | three objects with layer attributes: a named, half-opaque box, a hidden and locked arrow, a step number without them |
| `arrow-heads` | format version 2: four arrows: `open` head at `B`, `dot` at `A` only on a curve, `diamond` at `A` and `notched` at `B`, and a curved line without heads |
| `line-styles` | format version 2: a `dashed` arrow with the default head, a curved `dotted` line, a `dotted` square box and a `dashed` box with rounded corners |
| `shadow` | format version 2: a filled box with a soft black shadow at 40 % opacity down and to the right and an arrow with a sharp blue shadow 3 pixels to the right (light from the left, angle 180) |
| `callout` | format version 2: two texts with `s:spitze`, one with a label (filled polygon) and one without (legs as a line) |
| `magnifier` | format version 2: one magnifier with a non-integer source over a column strip (the strip drops out of the magnified view, 8.7); the visible image shows no annotations, as in all examples |

In the examples the visible image is the view *without* annotations (annotation rendering depends on a font
rasteriser and is not part of the format).

`go test ./internal/fileformat` checks that

* the writer reproduces every example byte for byte (`-update` rewrites them after an intended change);
* the reader reads every example with the expected content, and rejects the damaged variants of section 11;
* the constants of section 10 and the format version equal those of the code, and every `s:` item the writer emits is
  described in this document.

An implementation is conformant if it reads all examples with the same result and, as a writer, produces files that
the reference reader accepts and that follow sections 2 to 9.
