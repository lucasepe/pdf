# PDF examples

These programs are small, runnable demonstrations of the public API, following
the spirit of `fogleman/gg`'s `_examples` gallery. Their generated PDF files are
intentional persistent artifacts: open them in a PDF viewer and keep them in the
repository for visual inspection.

Regenerate everything from the repository root:

```sh
go run ./_examples/basic_shapes
go run ./_examples/font_sample
go run ./_examples/emoji_catalog
go run ./_examples/images_and_emoji
go run ./_examples/text_flow_6x9
go run ./_examples/tables_multipage
go run ./_examples/image_flow_6x9
go run ./_examples/ascii_diagrams
```

The examples cover:

- paths, curves, fills, strokes, dashes, clipping, transforms, and state;
- all six embedded ebook fonts: four Gentium Book faces and two Liberation Mono
  faces, with their OFL license texts kept beside the binaries;
- the complete 1,812-sequence Markdown emoji catalog used by `ebookgen`, drawn
  from Twemoji 17.0.3 across 41 pages;
- transparent `image.Image` resources and provider-backed inline emoji.
- 6×9 page geometry, mixed-style wrapping, automatic/explicit pagination,
  mirrored binding margins, headers, footers, and inline emoji.
- measured table columns, wrapped cells, atomic rows, repeated headers, and
  stable zebra striping across page breaks.
- verbatim JPEG embedding, transparent PNG-style resources, `scale` sizing,
  row-wrapping image groups, and atomic image pagination.
- nine `ebookgen` ASCII-diagram fixtures rendered directly as PDF vectors,
  including rounded corners, dotted borders, arrows, labels, and raster emoji.

Most layout examples use the small bundled Bitstream Vera test fixture. The
dedicated `font_sample` example instead embeds the production Gentium Book and
Liberation Mono files, so it doubles as a visual and structural font reference.

The emoji example downloads graphics on first use and caches them outside the
repository. Subsequent runs are offline and deterministic:

```sh
PDF_TWEMOJI_CACHE=/path/to/cache go run ./_examples/emoji_catalog
```

`catalog.txt` is generated from `ebookgen`'s Markdown `:emoji:` table. Refresh
it after that table changes with:

```sh
go run ./_examples/emoji_catalog/generate_catalog.go \
  -source /path/to/ebookgen/internal/markdown/zemoji.go \
  -output _examples/emoji_catalog/catalog.txt
```

Twemoji graphics are provided by the Twemoji project under CC BY 4.0. The
generated PDF carries the attribution on every page.
