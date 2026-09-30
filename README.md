# pdf

[![Go Reference](https://pkg.go.dev/badge/github.com/lucasepe/pdf.svg)](https://pkg.go.dev/github.com/lucasepe/pdf)
[![Go Report Card](https://goreportcard.com/badge/github.com/lucasepe/pdf)](https://goreportcard.com/report/github.com/lucasepe/pdf)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)

`pdf` is a compact, pure Go PDF generation library with a drawing API inspired by `fogleman/gg` and document layout primitives for paginated content. It writes PDF directly: no LaTeX executable, cgo, browser, or platform PDF framework is
required.

## Features

- A top left, point-based drawing context with paths, curves, clipping and affine transforms.
- Automatic paragraph wrapping, multi page flow, tables, image groups and
  direct ASCII diagrams.
- Embedded static TrueType fonts with metrics, CID mapping and searchable
  `ToUnicode` text.
- UTF-8 input for an explicit Western European Latin repertoire, with precise errors for unsupported runes.
- Transparent raster images, JPEG passthrough and provider backed inline emoji.
- Internal destinations, clickable URI links, metadata, page decorators and mirrored binding margins.
- Built-in PDF fonts, 144 named colors, HTML/RGB colors, configurable units, metadata and optional stream compression.
- Output to files, byte slices or any `io.Writer`.

## Not yet supported

- General Unicode text outside the WinAnsi Latin repertoire
- CFF/CFF2, variable, collection, WOFF/WOFF2, and color-font embedding
- PDF encryption

## Installation

The library requires Go 1.27 or newer.

```bash
go get github.com/lucasepe/pdf
```

## Hello World

```go
package main

import "github.com/lucasepe/pdf"

func main() {
    // create a new PDF using 'A4' page size
    doc := pdf.NewPDF("A4")

    // set the measurement units to centimeters
    doc.SetUnits("cm")

    // draw a grid to help us align stuff (just a guide, not necessary)
    doc.DrawUnitGrid()

    // draw the word 'HELLO' in orange, using 100pt bold Helvetica font
    // - text is placed on top of, not below the Y-coordinate
    // - you can use method chaining
    doc.SetFont("Helvetica-Bold", 100).
        SetXY(5, 5).
        SetColor("Orange").
        DrawText("HELLO")

    // draw the word 'WORLD' in blue-violet, using 100pt Helvetica font
    // note that here we use the colo(u)r hex code instead
    // of its name, using the CSS/HTML format: #RRGGBB
    doc.SetXY(5, 9).
        SetColor("#8A2BE2").
        SetFont("Helvetica", 100).
        DrawText("WORLD!")

    // draw a flower icon using 300pt Zapf-Dingbats font
    doc.SetX(7).SetY(17).
        SetColorRGB(255, 0, 0).
        SetFont("ZapfDingbats", 300).
        DrawText("a")

    // save the file:
    // if the file exists, it will be overwritten
    // if the file is in use, prints an error message
    if err := doc.SaveFile("hello.pdf"); err != nil {
        panic(err)
    }
}
```

## Embedded TrueType fonts

Font discovery belongs to the application. Register font bytes or an
`io.Reader` with a logical name, then select that name like a built-in font:

```go
fontData, err := os.ReadFile("GentiumBook-Regular.ttf")
if err != nil {
    return err
}
if err := doc.RegisterFont("Body", fontData); err != nil {
    return err
}
doc.SetFont("Body", 11).DrawText("Perché l’unità è già pronta — € 12,50")
```

The first embedding path accepts static TTF/OpenType fonts containing
TrueType `glyf`/`loca` outlines. Unsupported formats and fonts whose embedding
permissions prohibit use return `FontError`.

Semantic font roles keep document styling independent from physical files.
Body roles must be bound to registered embedded faces. Monospace roles default
to the PDF base-14 Courier faces, but applications should bind an embedded
TrueType monospace when byte-for-byte consistent rendering across devices is
required:

```go
if err := doc.BindFontRole(pdf.FontRoleBody, "Body"); err != nil {
    return err
}
if err := doc.UseFontRole(pdf.FontRoleMono, 9); err != nil {
    return err
}
doc.DrawText("inline_code()")
```

Color emoji are supplied independently through `EmojiProvider`. A provider
resolves a complete Unicode sequence to an `image.Image` and em-relative
geometry. `DrawEmoji` preserves alpha with a PDF soft mask and reuses identical
pixel resources throughout the document. The `ebookgen` adapter resolves a
pinned Twemoji release on demand; a typical inline configuration uses
`HeightEm: 1.15` and `BaselineEm: -0.18`, with `AdvanceEm` derived from the
image aspect ratio and desired spacing.

To verify an installed Gentium Book build manually:

```bash
PDF_TEST_FONT=/path/to/GentiumBook-Regular.ttf go test -run TestManualExternalTrueType -v
PDF_TEST_EMOJI=/path/to/twemoji.png go test -run TestManualEmojiPNG -v
```

## gg-like drawing context

`Context` offers a top-left, point-based API for paths, fills, strokes,
clipping, transforms, text, and transparent `image.Image` resources:

```go
doc := pdf.NewPDF("A4")
c := pdf.NewContext(&doc)

c.SetFillColor(color.RGBA{R: 47, G: 128, B: 237, A: 255}).
    DrawRoundedRectangle(40, 40, 220, 100, 16).
    Fill()

c.SetStrokeColor(color.Black).
    SetLineWidth(3).
    MoveTo(40, 180).
    CubicTo(140, 100, 260, 260, 380, 180).
    Stroke()

if err := c.SaveFile("drawing.pdf"); err != nil {
    return err
}
```

The Context is intentionally recognizable rather than source-compatible with
`fogleman/gg`: PDF page lifecycle and explicit finalization errors remain part
of the API.

## Flow layout and pagination

`Flow` sits above `Context` and keeps paragraphs inside a configurable page
body. It supports mixed body/monospace runs, atomic provider-backed
emoji, explicit newlines, long-token fallback, alignment, automatic pagination,
explicit page breaks, mirrored binding offsets, and a per-page decorator for
headers and footers:

```go
spec := pdf.PageSpec{
    Width: pdf.Inches(6), Height: pdf.Inches(9),
    MarginTop: 18, MarginRight: 18,
    MarginBottom: 18, MarginLeft: 18,
    HeaderHeight: 24, FooterHeight: 24,
}
doc := pdf.NewPDF(spec.PaperSize())
c := pdf.NewContext(&doc)
flow, err := pdf.NewFlow(c, spec, nil)
if err != nil {
    return err
}
flow.DrawParagraph([]pdf.TextRun{
    {Text: "Testo che va a capo; codice ", Style: pdf.TextStyle{FontName: "Helvetica", FontSize: 11}},
    {Text: "cfg.OutputPath", Style: pdf.TextStyle{FontRole: pdf.FontRoleMono, FontSize: 9, Wrap: pdf.WrapCode}},
}, pdf.ParagraphStyle{LineHeight: 15})
if err := flow.Error(); err != nil {
    return err
}
```

The layout layer is backend-neutral: it does not parse Markdown and leaves
tables and other block semantics to higher-level renderers.

## Clickable links and destinations

The context can attach URI or internal `GoTo` annotations to any rectangular
area. Internal destinations may be declared before or after their links; a
missing destination is reported when the PDF is finalized. Linked `TextRun`
values automatically receive one annotation per wrapped token and line:

```go
c.AddDestination("chapter-2", flow.Body().X, flow.CursorY())
flow.DrawParagraph([]pdf.TextRun{
    {Text: "Open the project", URI: "https://example.com", Style: body},
    {Text: " or jump to chapter 2", LinkDestination: "chapter-2", Style: body},
}, pdf.ParagraphStyle{LineHeight: 15})

// The same primitives can annotate images or custom drawings.
c.AddExternalLink("https://example.com/image", pdf.Rect{X: 40, Y: 80, Width: 120, Height: 90})
```

Annotation geometry uses the same transformed, top-left coordinate system as
the rest of `Context`; PDF object references remain private.

## Paginated tables

`Flow.DrawTable` measures columns and wrapped cells before drawing. Body rows
are atomic and the optional header is repeated after every automatic page
break. This maps directly onto a Markdown table AST without making the PDF
library depend on a Markdown parser:

```go
body := pdf.TextStyle{FontName: "Helvetica", FontSize: 9}
table := pdf.Table{
    Columns: []pdf.TableColumn{
        {Width: 42, Align: pdf.AlignRight},
        {Weight: 1},
        {Weight: 2},
    },
    Header: []pdf.TableCell{
        pdf.TextTableCell("ID", body),
        pdf.TextTableCell("Titolo", body),
        pdf.TextTableCell("Descrizione", body),
    },
    Rows: []pdf.TableRow{
        {Cells: []pdf.TableCell{
            pdf.TextTableCell("01", body),
            pdf.TextTableCell("Introduzione", body),
            pdf.TextTableCell("Testo Latin che può andare a capo.", body),
        }},
    },
}
flow.DrawTable(table)
if err := flow.Error(); err != nil {
    return err
}
```

A row taller than the usable page body returns `TableError`; it is never
silently clipped or divided between pages.

## Image resources and image flow

Register encoded bytes or an `io.Reader` when the original representation is
available. Compatible Gray/RGB JPEG data is embedded verbatim with
`/DCTDecode`; decoded images retain the existing alpha-mask path:

```go
data, err := os.ReadFile("photo.jpg")
if err != nil {
    return err
}
photo, err := doc.RegisterImage(data)
if err != nil {
    return err
}

flow.DrawImage(photo, pdf.ImageOptions{Scale: 0.75})
```

The default sizing matches ebookgen: width is `0.78 × scale × body width`, clamped to `0.98 × body width`, centered, with aspect ratio preserved. Image groups reproduce the consecutive `flow` behavior with a default horizontal gap of `0.02 × body width`:

```go
flow.DrawImageGroup([]pdf.ImageItem{
    {Image: front, Scale: 0.32},
    {Image: detail, Scale: 0.32},
    {Image: pillow, Scale: 0.22},
}, pdf.ImageGroupOptions{})
```

Each standalone image or group row is atomic during pagination. Upscaling only changes displayed size; `ImageResource.EffectiveDPI` can be used to assess the available source resolution.

## Direct ASCII diagrams

`Flow.DrawASCIIDiagram` turns the diagram alphabet used by `ebookgen` into PDF vectors without TikZ or LaTeX. It supports horizontal, vertical, and diagonal segments; `=`/`:` dotted borders; junctions; sharp or rounded corners; and four arrow directions. Labels use the configured monospace role, while complete emoji sequences are
resolved through the document's `EmojiProvider`:

```go
flow.DrawASCIIDiagram(`.----------.
| producer |----> consumer
'----------'`, pdf.ASCIIDiagramOptions{})
```

The diagram is measured as a Unicode-cell grid, uniformly reduced when needed,
centered by default, and kept atomic across page breaks. This is deliberately a
small diagram syntax, not a general-purpose monospace-art parser; tabs,
ambiguous East Asian display widths, and arbitrary Unicode line-drawing glyphs
are outside the current contract.

## Example gallery

Runnable sources and persistent PDFs are available under [`_examples`](./_examples):

- [basic shapes PDF](./_examples/basic_shapes/output.pdf)
- [embedded font sample PDF](./_examples/font_sample/font-sample.pdf)
- [1,812-sequence Twemoji catalog PDF](./_examples/emoji_catalog/emoji-catalog.pdf)
- [images and emoji PDF](./_examples/images_and_emoji/output.pdf)
- [6×9 flow layout PDF](./_examples/text_flow_6x9/output.pdf)
- [multipage table PDF](./_examples/tables_multipage/output.pdf)
- [6×9 image flow PDF](./_examples/image_flow_6x9/output.pdf)
- [nine direct ASCII diagram fixtures](./_examples/ascii_diagrams/output.pdf)

Regenerate them from the repository root with the commands documented in
[`_examples/README.md`](./_examples/README.md). These PDFs are intentionally
kept for visual inspection.
