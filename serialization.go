package pdf

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
)

// Compression returns the current compression mode. If it is true,
// all PDF content will be compressed when the PDF is generated. If
// false, most PDF content (excluding images) will be in plain text,
// which is useful for debugging or to study PDF commands.
func (p *PDF) Compression() bool { p.init(); return p.compression }

// SetCompression sets the compression mode used to generate the PDF.
// If set to true, all PDF steams will be compressed when the PDF is
// generated. If false, most content (excluding images) will be in
// plain text, which is useful for debugging or to study PDF commands.
func (p *PDF) SetCompression(val bool) *PDF {
	p.init()
	p.compression = val
	return p
}

// Bytes generates the PDF document and returns bytes identical to the content
// of a PDF file. Serialization errors are added to Errors. Use WriteTo when the
// caller needs an explicit error result.
func (p *PDF) Bytes() []byte {
	data, err := p.buildBytes()
	if err != nil {
		p.errors = append(p.errors, err)
	}
	return data
}

// WriteTo generates the PDF document and writes it to wr.
func (p *PDF) WriteTo(wr io.Writer) (int64, error) {
	if wr == nil {
		return 0, fmt.Errorf("nil PDF writer")
	}
	data, err := p.buildBytes()
	if err != nil {
		return 0, err
	}
	n, err := wr.Write(data)
	if err != nil {
		return int64(n), err
	}
	if n != len(data) {
		return int64(n), io.ErrShortWrite
	}
	return int64(n), nil
}

func (p *PDF) buildBytes() ([]byte, error) {

	p.reservePage()
	hasInfo := p.docTitle != "" || p.docSubject != "" ||
		p.docKeywords != "" || p.docAuthor != "" || p.docCreator != ""
	fontObjectCounts := make([]int, len(p.fonts))
	for i, font := range p.fonts {
		fontObjectCounts[i] = 1
		if font.handler != nil {
			fontObjectCounts[i] = font.handler.objectCount()
		}
	}
	imageObjectCounts := make([]int, len(p.images))
	for i, img := range p.images {
		imageObjectCounts[i] = 1
		if len(img.alpha) > 0 {
			imageObjectCounts[i] = 2
		}
	}
	plan := newPDFObjectPlan(len(p.pages), fontObjectCounts, imageObjectCounts, hasInfo)
	prevWriter := p.writer
	p.content.Reset()
	p.writer = &p.content
	p.objOffsets = make([]int, plan.count+1)
	p.writeErr = nil
	p.write("%PDF-1.4\n\n").
		writeObjAt(plan.catalog, "/Catalog").
		write("/Pages ", plan.pages, " 0 R>>\n"+"endobj\n\n")

	p.writePages(plan)

	for i, font := range p.fonts {
		if font.handler == nil {
			p.writeObjAt(plan.fonts[i][0], "/Font").write("/Subtype/Type1/Name/FNT", font.id, "\n",
				"/BaseFont/", font.name, "\n")
			if font.name != "Symbol" && font.name != "ZapfDingbats" {
				p.write("/Encoding/WinAnsiEncoding")
			}
			p.write(">>\n" + "endobj\n")
		} else {
			font.handler.writeFontObjects(plan.fonts[i])
		}
	}

	for i, img := range p.images {
		colorSpace := "DeviceRGB"
		if img.isGray {
			colorSpace = "DeviceGray"
		}
		old := p.compression
		p.compression = true
		p.writeObjAt(plan.images[i][0], "/XObject").
			write("/Subtype/Image\n",
				"/Width ", img.widthPx, "/Height ", img.heightPx,
				"/ColorSpace/", colorSpace, "/BitsPerComponent 8\n")
		if len(img.alpha) > 0 {
			p.write("/SMask ", plan.images[i][1], " 0 R\n")
		}
		if img.filter != "" {
			p.write("/Filter/", img.filter, "/Length ", len(img.data), ">> stream\n",
				string(img.data), "\nendstream\n")
		} else {
			p.writeStreamData(img.data)
		}
		p.write("\n" + "endobj\n\n")
		if len(img.alpha) > 0 {
			p.writeObjAt(plan.images[i][1], "/XObject").
				write("/Subtype/Image\n/Width ", img.widthPx, "/Height ", img.heightPx,
					"/ColorSpace/DeviceGray/BitsPerComponent 8\n").
				writeStreamData(img.alpha).write("\nendobj\n\n")
		}
		p.compression = old
	}
	// write info object
	var metadataErr error
	if hasInfo {

		p.writeObjAt(plan.info, "/Info")
		for _, tuple := range [][]string{
			{"/Title ", p.docTitle}, {"/Subject ", p.docSubject},
			{"/Keywords ", p.docKeywords}, {"/Author ", p.docAuthor},
			{"/Creator ", p.docCreator},
		} {
			if tuple[1] != "" {
				encoded, err := encodePDFDoc(tuple[1])
				if err != nil {
					p.errors = append(p.errors, err)
					if metadataErr == nil {
						metadataErr = err
					}
					continue
				}
				p.write(tuple[0], "(", escapePDFBytes(encoded), ")")
			}
		}
		p.write(">>\n" + "endobj\n\n")
	}

	start := p.content.Len()
	p.write("xref\n"+
		"0 ", len(p.objOffsets), "\n"+"0000000000 65535 f \n")
	for _, offset := range p.objOffsets[1:] {
		p.write(fmt.Sprintf("%010d 00000 n \n", offset))
	}

	p.write("trailer\n"+"<</Size ", len(p.objOffsets), "/Root ", plan.catalog, " 0 R")
	if plan.info > 0 {
		p.write("/Info ", plan.info, " 0 R")
	}
	p.write(">>\n"+"startxref\n", start, "\n", "%%EOF\n")
	p.writer = prevWriter
	if p.writeErr == nil {
		p.writeErr = metadataErr
	}
	return append([]byte(nil), p.content.Bytes()...), p.writeErr
}

// SaveFile generates and saves the PDF document to a file.
func (p *PDF) SaveFile(filename string) error {
	data, err := p.buildBytes()
	if err == nil {
		err = os.WriteFile(filename, data, 0644)
	}
	if err != nil {
		p.putError(0xED3F6D, "Failed writing file", err.Error())
	}
	return err
}

type pdfObjectPlan struct {
	catalog, pages int
	page, stream   []int
	fonts, images  [][]int
	info, count    int
}

func newPDFObjectPlan(pageCount int, fontObjectCounts, imageObjectCounts []int, hasInfo bool) pdfObjectPlan {
	next := 0
	reserve := func() int {
		next++
		return next
	}
	plan := pdfObjectPlan{catalog: reserve(), pages: reserve()}
	for i := 0; i < pageCount; i++ {
		plan.page = append(plan.page, reserve())
		plan.stream = append(plan.stream, reserve())
	}
	for _, count := range fontObjectCounts {
		ids := make([]int, count)
		for i := range ids {
			ids[i] = reserve()
		}
		plan.fonts = append(plan.fonts, ids)
	}
	for _, count := range imageObjectCounts {
		ids := make([]int, count)
		for i := range ids {
			ids[i] = reserve()
		}
		plan.images = append(plan.images, ids)
	}
	if hasInfo {
		plan.info = reserve()
	}
	plan.count = next
	return plan
}

// write writes strings and numbers to the current page's content
// stream or to the final generated PDF, if there is no active page
func (p *PDF) write(a ...any) *PDF {
	p.reservePage()
	if p.writeErr == nil {
		_, p.writeErr = p.writeTo(p.writer, a...)
	}
	return p
}

// writeObjAt writes an object header at a previously reserved object number.
func (p *PDF) writeObjAt(id int, objType string) *PDF {
	p.beginObjAt(id)
	if p.writeErr != nil {
		return p
	}
	return p.write("<</Type", objType)
}

// beginObjAt starts an object at a previously reserved object number.
func (p *PDF) beginObjAt(id int) *PDF {
	if id <= 0 || id >= len(p.objOffsets) {
		if p.writeErr == nil {
			p.writeErr = fmt.Errorf("invalid PDF object number %d", id)
		}
		return p
	}
	p.objOffsets[id] = p.content.Len()
	return p.write(id, " 0 obj ")
}

// writePages writes all PDF pages
func (p *PDF) writePages(plan pdfObjectPlan) *PDF {
	p.writeObjAt(plan.pages, "/Pages").write("/Count ", len(p.pages), "/MediaBox[0 0 ",
		pdfNumber(p.paperSize.widthPt), " ", pdfNumber(p.paperSize.heightPt), "]")

	if len(p.pages) > 0 {
		p.write("/Kids[")
		for i := range p.pages {
			if i > 0 {
				p.write(" ")
			}
			p.write(plan.page[i], " 0 R")
		}
		p.write("]")
	}
	p.write(">>\n" + "endobj\n\n")
	for pageNo, pg := range p.pages {
		p.writeObjAt(plan.page[pageNo], "/Page").
			write("/Parent ", plan.pages, " 0 R/Contents ", plan.stream[pageNo], " 0 R")
		p.write("\n/Resources <<")
		if len(pg.fontIDs) > 0 {
			p.write("/Font <<")
			for _, id := range pg.fontIDs {
				if len(pg.fontIDs) > 1 {
					p.write("\n")
				}
				p.write("/FNT", id, " ", plan.fonts[id-1][0], " 0 R")
			}
			p.write(">> ")
		}
		if len(pg.imageIDs) > 0 {
			p.write("/XObject <<")
			for _, id := range pg.imageIDs {
				if len(pg.imageIDs) > 1 {
					p.write("\n")
				}
				p.write("/IMG", id, " ", plan.images[id][0], " 0 R")
			}
			p.write(">> ")
		}
		p.write(">> ")
		if len(pg.links) > 0 {
			p.write("/Annots[")
			for _, link := range pg.links {
				r := link.rect
				p.write("<</Type/Annot/Subtype/Link/Rect[", pdfNumber(r.X), " ", pdfNumber(r.Y), " ",
					pdfNumber(r.X+r.Width), " ", pdfNumber(r.Y+r.Height), "]/Border[0 0 0]/A<<")
				if link.uri != "" {
					p.write("/S/URI/URI(", escapePDFBytes([]byte(link.uri)), ")")
				} else {
					destination, ok := p.destinations[link.target]
					if !ok || destination.page < 0 || destination.page >= len(plan.page) {
						if p.writeErr == nil {
							p.writeErr = fmt.Errorf("unresolved PDF destination %q", link.target)
						}
						return p
					}
					p.write("/S/GoTo/D[", plan.page[destination.page], " 0 R/XYZ ", pdfNumber(destination.x), " ", pdfNumber(destination.y), " null]")
				}
				p.write(">>>>")
			}
			p.write("] ")
		}
		p.write(">>\n" + "endobj\n\n")
		p.writeStreamObjAt(plan.stream[pageNo], pg.content.Bytes())
	}
	return p
}

// writeStreamData writes a stream or image stream
func (p *PDF) writeStreamData(ar []byte) *PDF {
	var filter string
	if p.compression {
		var (
			buf    bytes.Buffer
			wr     = zlib.NewWriter(&buf)
			_, err = wr.Write(ar)
		)
		if err != nil {
			return p.putError(0xE782A2, "Failed compressing", err.Error())
		}
		if err = wr.Close(); err != nil {
			return p.putError(0xE782A2, "Failed closing compressor", err.Error())
		}
		ar = buf.Bytes()
		filter = "/Filter/FlateDecode"
	}
	return p.write(filter, "/Length ", len(ar), ">> stream\n",
		string(ar), "\n"+"endstream\n")
}

// writeStreamObjAt outputs a stream at a previously reserved object number.
func (p *PDF) writeStreamObjAt(id int, ar []byte) *PDF {
	return p.beginObjAt(id).write("<<").
		writeStreamData(ar).write("\n" + "endobj\n\n")
}

// escape escapes special characters '(', '(' and '\' in strings
// in order to avoid them interfering with PDF commands
func (*PDF) escape(s string) string {
	has := strings.Contains
	if !has(s, "(") && !has(s, ")") && !has(s, "\\") {
		return s
	}
	buf := bytes.NewBuffer(make([]byte, 0, len(s)))
	for _, r := range s {
		if r == '(' || r == ')' || r == '\\' {
			buf.WriteRune('\\')
		}
		buf.WriteRune(r)
	}
	return buf.String()
}

// writeTo writes multiple strings and numbers specified in 'args' using
// writer 'wr'. Returns total bytes written and the first error if any.
type pdfNumber float64

func (*PDF) writeTo(wr io.Writer, args ...any) (count int, err error) {
	for _, any := range args {
		n, err := 0, error(nil)
		switch val := any.(type) {
		case string:
			n, err = io.WriteString(wr, val)
		case float64:
			n, err = io.WriteString(wr, strconv.FormatFloat(val, 'f', 3, 64))
		case pdfNumber:
			n, err = io.WriteString(wr, strconv.FormatFloat(float64(val), 'f', -1, 64))
		case int:
			n, err = io.WriteString(wr, strconv.FormatInt(int64(val), 10))
		case int16:
			n, err = io.WriteString(wr, strconv.FormatInt(int64(val), 10))
		case uint:
			n, err = io.WriteString(wr, strconv.FormatInt(int64(val), 10))
		case uint16:
			n, err = io.WriteString(wr, strconv.FormatInt(int64(val), 10))
		case *bytes.Buffer:
			if val != nil {
				n, err = wr.Write(val.Bytes())
			}
		case []byte:
			n, err = wr.Write(val)
		default:
			n, err = 0,
				fmt.Errorf("Invalid type %s = %v", reflect.TypeOf(val), val)
		}
		count += n
		if err != nil {
			return count, err
		}
	}
	return count, nil
}
