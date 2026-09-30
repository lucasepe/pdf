package pdf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"unicode/utf16"
)

const maxFontSize = 64 << 20

// FontError reports an invalid, unsupported, or non-embeddable font.
type FontError struct {
	Kind   string
	Detail string
}

func (e FontError) Error() string {
	if e.Detail == "" {
		return "font: " + e.Kind
	}
	return "font: " + e.Kind + ": " + e.Detail
}

type fontTable struct{ offset, length uint32 }

type pdfTTFont struct {
	pdf             *PDF
	logicalName     string
	pdfName         string
	data            []byte
	tables          map[string]fontTable
	unitsPerEm      uint16
	xMin, yMin      int16
	xMax, yMax      int16
	ascent, descent int16
	capHeight       int16
	italicAngle     int32
	weight          uint16
	fixedPitch      bool
	numGlyphs       uint16
	widths          []uint16
	cmap            map[rune]uint16
	runeToCID       map[rune]uint16
	cidToRune       map[uint16]rune
	cidToGID        map[uint16]uint16
}

// RegisterFont registers caller-provided TrueType font data under name. Source
// may be a []byte or io.Reader. The library never performs system font discovery.
func (p *PDF) RegisterFont(name string, source any) error {
	p.init()
	key := p.toUpperLettersDigits(name, "")
	if key == "" {
		return FontError{Kind: "invalid name", Detail: name}
	}
	if _, builtIn := p.currentBuiltInFontNamed(name); builtIn {
		return FontError{Kind: "reserved name", Detail: name}
	}
	if _, exists := p.registeredFonts[key]; exists {
		return FontError{Kind: "duplicate name", Detail: name}
	}
	data, err := readFontSource(source)
	if err != nil {
		return err
	}
	font, err := parseTrueType(data)
	if err != nil {
		return err
	}
	font.pdf, font.logicalName = p, name
	if p.registeredFonts == nil {
		p.registeredFonts = make(map[string]*pdfTTFont)
	}
	p.registeredFonts[key] = font
	return nil
}

func readFontSource(source any) ([]byte, error) {
	switch src := source.(type) {
	case []byte:
		if len(src) > maxFontSize {
			return nil, FontError{Kind: "too large", Detail: fmt.Sprintf("%d bytes", len(src))}
		}
		return append([]byte(nil), src...), nil
	case io.Reader:
		value := reflect.ValueOf(src)
		if (value.Kind() == reflect.Ptr || value.Kind() == reflect.Interface) && value.IsNil() {
			return nil, FontError{Kind: "invalid source", Detail: "nil reader"}
		}
		data, err := io.ReadAll(io.LimitReader(src, maxFontSize+1))
		if err != nil {
			return nil, FontError{Kind: "read failed", Detail: err.Error()}
		}
		if len(data) > maxFontSize {
			return nil, FontError{Kind: "too large", Detail: fmt.Sprintf("more than %d bytes", maxFontSize)}
		}
		return data, nil
	default:
		return nil, FontError{Kind: "invalid source", Detail: fmt.Sprintf("%T", source)}
	}
}

func parseTrueType(data []byte) (*pdfTTFont, error) {
	if len(data) < 12 {
		return nil, FontError{Kind: "malformed", Detail: "truncated sfnt header"}
	}
	signature := string(data[:4])
	switch signature {
	case "OTTO":
		return nil, FontError{Kind: "unsupported outlines", Detail: "CFF/CFF2"}
	case "ttcf":
		return nil, FontError{Kind: "unsupported container", Detail: "TrueType collection"}
	case "wOFF", "wOF2":
		return nil, FontError{Kind: "unsupported container", Detail: signature}
	}
	if !bytes.Equal(data[:4], []byte{0, 1, 0, 0}) && signature != "true" {
		return nil, FontError{Kind: "unsupported sfnt version", Detail: fmt.Sprintf("%x", data[:4])}
	}
	numTables := int(u16(data, 4))
	if numTables == 0 || numTables > 256 || 12+numTables*16 > len(data) {
		return nil, FontError{Kind: "malformed", Detail: "invalid table directory"}
	}
	f := &pdfTTFont{data: data, tables: make(map[string]fontTable), cmap: make(map[rune]uint16), runeToCID: make(map[rune]uint16), cidToRune: make(map[uint16]rune), cidToGID: make(map[uint16]uint16)}
	for i := 0; i < numTables; i++ {
		pos := 12 + i*16
		tag := string(data[pos : pos+4])
		off, length := u32(data, pos+8), u32(data, pos+12)
		if uint64(off)+uint64(length) > uint64(len(data)) {
			return nil, FontError{Kind: "malformed", Detail: "table " + tag + " exceeds file"}
		}
		f.tables[tag] = fontTable{off, length}
	}
	for _, tag := range []string{"head", "hhea", "maxp", "hmtx", "cmap", "name", "OS/2", "post", "loca", "glyf"} {
		if _, ok := f.tables[tag]; !ok {
			if tag == "glyf" || tag == "loca" {
				return nil, FontError{Kind: "unsupported outlines", Detail: "missing glyf/loca"}
			}
			return nil, FontError{Kind: "malformed", Detail: "missing " + tag + " table"}
		}
	}
	if _, variable := f.tables["fvar"]; variable {
		return nil, FontError{Kind: "unsupported variation font", Detail: "select a static TrueType instance"}
	}
	if err := f.parseMetrics(); err != nil {
		return nil, err
	}
	if err := f.parseCmap(); err != nil {
		return nil, err
	}
	if err := f.parseName(); err != nil {
		return nil, err
	}
	if err := f.parseOS2(); err != nil {
		return nil, err
	}
	f.parsePost()
	return f, nil
}

func (f *pdfTTFont) table(tag string, minimum int) ([]byte, error) {
	t, ok := f.tables[tag]
	if !ok || int(t.length) < minimum {
		return nil, FontError{Kind: "malformed", Detail: "short " + tag + " table"}
	}
	return f.data[int(t.offset):int(t.offset+t.length)], nil
}

func (f *pdfTTFont) parseMetrics() error {
	head, err := f.table("head", 54)
	if err != nil {
		return err
	}
	f.unitsPerEm = u16(head, 18)
	if f.unitsPerEm < 16 || f.unitsPerEm > 16384 {
		return FontError{Kind: "malformed", Detail: "invalid unitsPerEm"}
	}
	f.xMin, f.yMin, f.xMax, f.yMax = i16(head, 36), i16(head, 38), i16(head, 40), i16(head, 42)
	locaFormat := i16(head, 50)
	if locaFormat != 0 && locaFormat != 1 {
		return FontError{Kind: "malformed", Detail: "invalid loca format"}
	}
	hhea, err := f.table("hhea", 36)
	if err != nil {
		return err
	}
	f.ascent, f.descent = i16(hhea, 4), i16(hhea, 6)
	metricCount := int(u16(hhea, 34))
	maxp, err := f.table("maxp", 6)
	if err != nil {
		return err
	}
	f.numGlyphs = u16(maxp, 4)
	if f.numGlyphs == 0 || metricCount == 0 || metricCount > int(f.numGlyphs) {
		return FontError{Kind: "malformed", Detail: "invalid glyph or metric count"}
	}
	hmtx, err := f.table("hmtx", metricCount*4+(int(f.numGlyphs)-metricCount)*2)
	if err != nil {
		return err
	}
	f.widths = make([]uint16, f.numGlyphs)
	for i := 0; i < metricCount; i++ {
		f.widths[i] = u16(hmtx, i*4)
	}
	for i := metricCount; i < int(f.numGlyphs); i++ {
		f.widths[i] = f.widths[metricCount-1]
	}
	locaEntrySize := 2
	if locaFormat == 1 {
		locaEntrySize = 4
	}
	loca, err := f.table("loca", (int(f.numGlyphs)+1)*locaEntrySize)
	if err != nil {
		return err
	}
	glyf := f.tables["glyf"]
	var previous uint32
	for i := 0; i <= int(f.numGlyphs); i++ {
		var offset uint32
		if locaFormat == 0 {
			offset = uint32(u16(loca, i*2)) * 2
		} else {
			offset = u32(loca, i*4)
		}
		if offset < previous || offset > glyf.length {
			return FontError{Kind: "malformed", Detail: "invalid loca offsets"}
		}
		previous = offset
	}
	return nil
}

func (f *pdfTTFont) parseCmap() error {
	cmap, err := f.table("cmap", 4)
	if err != nil {
		return err
	}
	count := int(u16(cmap, 2))
	if count == 0 || count > 128 || 4+count*8 > len(cmap) {
		return FontError{Kind: "malformed", Detail: "invalid cmap directory"}
	}
	type candidate struct {
		score int
		data  []byte
	}
	var choices []candidate
	for i := 0; i < count; i++ {
		pos := 4 + i*8
		platform, encoding, offset := u16(cmap, pos), u16(cmap, pos+2), u32(cmap, pos+4)
		if int(offset)+2 > len(cmap) {
			return FontError{Kind: "malformed", Detail: "cmap offset exceeds table"}
		}
		format, score := u16(cmap, int(offset)), 0
		if format == 12 && platform == 3 && encoding == 10 {
			score = 4
		} else if format == 12 && platform == 0 {
			score = 3
		} else if format == 4 && platform == 3 && encoding == 1 {
			score = 2
		} else if format == 4 && platform == 0 {
			score = 1
		}
		if score > 0 {
			choices = append(choices, candidate{score, cmap[int(offset):]})
		}
	}
	if len(choices) == 0 {
		return FontError{Kind: "unsupported cmap", Detail: "need Unicode format 4 or 12"}
	}
	sort.Slice(choices, func(i, j int) bool { return choices[i].score > choices[j].score })
	if u16(choices[0].data, 0) == 12 {
		err = f.parseCmap12(choices[0].data)
	} else {
		err = f.parseCmap4(choices[0].data)
	}
	if err != nil {
		return err
	}
	if len(f.cmap) == 0 {
		return FontError{Kind: "unsupported cmap", Detail: "no usable mappings"}
	}
	return nil
}

func (f *pdfTTFont) parseCmap12(table []byte) error {
	if len(table) < 16 {
		return FontError{Kind: "malformed", Detail: "short cmap format 12"}
	}
	length, groups := int(u32(table, 4)), int(u32(table, 12))
	if length > len(table) || groups > 1<<20 || 16+groups*12 > length {
		return FontError{Kind: "malformed", Detail: "invalid cmap format 12"}
	}
	for i := 0; i < groups; i++ {
		pos := 16 + i*12
		start, end, gid := u32(table, pos), u32(table, pos+4), u32(table, pos+8)
		if end < start || end-start > 1<<20 {
			return FontError{Kind: "malformed", Detail: "invalid cmap group"}
		}
		for cp := start; cp <= end && cp <= 0xffff; cp++ {
			g := gid + (cp - start)
			if g < uint32(f.numGlyphs) && g != 0 {
				f.cmap[rune(cp)] = uint16(g)
			}
		}
	}
	return nil
}

func (f *pdfTTFont) parseCmap4(table []byte) error {
	if len(table) < 16 {
		return FontError{Kind: "malformed", Detail: "short cmap format 4"}
	}
	length, segCount := int(u16(table, 2)), int(u16(table, 6))/2
	if length > len(table) || segCount == 0 || segCount > 8192 || 16+segCount*8 > length {
		return FontError{Kind: "malformed", Detail: "invalid cmap format 4"}
	}
	endBase := 14
	startBase := endBase + segCount*2 + 2
	deltaBase := startBase + segCount*2
	rangeBase := deltaBase + segCount*2
	for i := 0; i < segCount; i++ {
		start, end := u16(table, startBase+i*2), u16(table, endBase+i*2)
		if end < start {
			return FontError{Kind: "malformed", Detail: "invalid cmap segment"}
		}
		delta, ro := i16(table, deltaBase+i*2), u16(table, rangeBase+i*2)
		for cp := uint32(start); cp <= uint32(end) && cp < 0xffff; cp++ {
			var gid uint16
			if ro == 0 {
				gid = uint16(int32(cp) + int32(delta))
			} else {
				glyphPos := rangeBase + i*2 + int(ro) + int(cp-uint32(start))*2
				if glyphPos+2 > length {
					return FontError{Kind: "malformed", Detail: "cmap glyph offset exceeds table"}
				}
				gid = u16(table, glyphPos)
				if gid != 0 {
					gid = uint16(int32(gid) + int32(delta))
				}
			}
			if gid != 0 && gid < f.numGlyphs {
				f.cmap[rune(cp)] = gid
			}
		}
	}
	return nil
}

func (f *pdfTTFont) parseName() error {
	name, err := f.table("name", 6)
	if err != nil {
		return err
	}
	count, storage := int(u16(name, 2)), int(u16(name, 4))
	if count > 4096 || 6+count*12 > len(name) || storage > len(name) {
		return FontError{Kind: "malformed", Detail: "invalid name table"}
	}
	type foundName struct {
		score int
		value string
	}
	best := foundName{}
	for i := 0; i < count; i++ {
		pos := 6 + i*12
		platform, nameID := u16(name, pos), u16(name, pos+6)
		if nameID != 6 {
			continue
		}
		length, off := int(u16(name, pos+8)), storage+int(u16(name, pos+10))
		if off < storage || off+length > len(name) {
			return FontError{Kind: "malformed", Detail: "name string exceeds table"}
		}
		value, score := "", 1
		if platform == 0 || platform == 3 {
			value = decodeUTF16BE(name[off : off+length])
			score = 2
		} else {
			value = string(name[off : off+length])
		}
		value = sanitizePDFName(value)
		if value != "" && score > best.score {
			best = foundName{score, value}
		}
	}
	if best.value == "" {
		return FontError{Kind: "malformed", Detail: "missing PostScript name"}
	}
	f.pdfName = best.value
	return nil
}

func (f *pdfTTFont) parseOS2() error {
	os2, err := f.table("OS/2", 10)
	if err != nil {
		return err
	}
	f.weight = u16(os2, 4)
	fsType := u16(os2, 8)
	if fsType&0x0002 != 0 || fsType&0x0200 != 0 {
		return FontError{Kind: "embedding prohibited", Detail: fmt.Sprintf("OS/2 fsType 0x%04X", fsType)}
	}
	if len(os2) >= 74 {
		f.ascent, f.descent = i16(os2, 68), i16(os2, 70)
	}
	if len(os2) >= 90 && u16(os2, 0) >= 2 {
		f.capHeight = i16(os2, 88)
	} else {
		f.capHeight = f.ascent
	}
	return nil
}

func (f *pdfTTFont) parsePost() {
	post, err := f.table("post", 16)
	if err != nil {
		return
	}
	f.italicAngle = int32(u32(post, 4))
	f.fixedPitch = u32(post, 12) != 0
}

func (f *pdfTTFont) encodeRun(s string) ([]uint16, error) {
	out := make([]uint16, 0, len(s))
	runeIndex := 0
	for byteOffset, r := range s {
		if _, ok := winAnsiByte(r); !ok {
			return nil, TextEncodingError{Encoding: "embedded Latin", Font: f.logicalName, Rune: r, ByteOffset: byteOffset, RuneIndex: runeIndex}
		}
		gid, ok := f.cmap[r]
		if !ok || gid == 0 {
			return nil, TextEncodingError{Encoding: "font cmap", Font: f.logicalName, Rune: r, ByteOffset: byteOffset, RuneIndex: runeIndex}
		}
		cid, ok := f.runeToCID[r]
		if !ok {
			if len(f.runeToCID) >= 65534 {
				return nil, FontError{Kind: "too many CIDs", Detail: f.logicalName}
			}
			cid = uint16(len(f.runeToCID) + 1)
			f.runeToCID[r], f.cidToRune[cid], f.cidToGID[cid] = cid, r, gid
		}
		out = append(out, cid)
		runeIndex++
	}
	return out, nil
}

func (f *pdfTTFont) textWidthPt(s string) (float64, error) {
	run, err := f.encodeRun(s)
	if err != nil {
		return 0, err
	}
	var total uint64
	for _, cid := range run {
		total += uint64(f.widths[f.cidToGID[cid]])
	}
	return float64(total) / float64(f.unitsPerEm) * f.pdf.fontSizePt * float64(f.pdf.horzScaling) / 100, nil
}
func (f *pdfTTFont) writeText(s string) error {
	run, err := f.encodeRun(s)
	if err != nil {
		return err
	}
	var hex strings.Builder
	for _, cid := range run {
		fmt.Fprintf(&hex, "%04X", cid)
	}
	f.pdf.write("BT ", int(f.pdf.page.x), " ", int(f.pdf.page.y), " Td <", hex.String(), "> Tj ET\n")
	return nil
}
func (*pdfTTFont) objectCount() int { return 6 }

func (f *pdfTTFont) writeFontObjects(ids []int) {
	if len(ids) != 6 {
		f.pdf.writeErr = FontError{Kind: "internal object plan", Detail: f.logicalName}
		return
	}
	p := f.pdf
	name := sanitizePDFName(f.pdfName)
	p.beginObjAt(ids[0]).write("<</Type/Font/Subtype/Type0/BaseFont/", name, "/Encoding/Identity-H/DescendantFonts[", ids[1], " 0 R]/ToUnicode ", ids[5], " 0 R>>\nendobj\n\n")
	p.beginObjAt(ids[1]).write("<</Type/Font/Subtype/CIDFontType2/BaseFont/", name, "/CIDSystemInfo<</Registry(Adobe)/Ordering(Identity)/Supplement 0>>/FontDescriptor ", ids[2], " 0 R/DW 1000/W[")
	cids := f.sortedCIDs()
	for _, cid := range cids {
		p.write(cid, "[", f.pdfWidth(f.widths[f.cidToGID[cid]]), "]")
	}
	p.write("]/CIDToGIDMap ", ids[4], " 0 R>>\nendobj\n\n")
	flags := 32
	if f.fixedPitch {
		flags |= 1
	}
	if f.italicAngle != 0 {
		flags |= 64
	}
	p.beginObjAt(ids[2]).write("<</Type/FontDescriptor/FontName/", name, "/Flags ", flags, "/FontBBox[", f.pdfMetric(f.xMin), " ", f.pdfMetric(f.yMin), " ", f.pdfMetric(f.xMax), " ", f.pdfMetric(f.yMax), "]/ItalicAngle ", pdfNumber(float64(f.italicAngle)/65536), "/Ascent ", f.pdfMetric(f.ascent), "/Descent ", f.pdfMetric(f.descent), "/CapHeight ", f.pdfMetric(f.capHeight), "/StemV ", stemV(f.weight), "/FontFile2 ", ids[3], " 0 R>>\nendobj\n\n")
	p.beginObjAt(ids[3]).write("<</Length1 ", len(f.data)).writeStreamData(f.data).write("\nendobj\n\n")
	gids := make([]byte, (len(cids)+1)*2)
	for _, cid := range cids {
		binary.BigEndian.PutUint16(gids[int(cid)*2:], f.cidToGID[cid])
	}
	p.writeStreamObjAt(ids[4], gids)
	p.writeStreamObjAt(ids[5], f.toUnicodeCMap(cids))
}

func (f *pdfTTFont) sortedCIDs() []uint16 {
	out := make([]uint16, 0, len(f.cidToRune))
	for cid := range f.cidToRune {
		out = append(out, cid)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
func (f *pdfTTFont) pdfWidth(v uint16) int {
	return int((uint64(v)*1000 + uint64(f.unitsPerEm)/2) / uint64(f.unitsPerEm))
}
func (f *pdfTTFont) pdfMetric(v int16) int {
	if v >= 0 {
		return int((int64(v)*1000 + int64(f.unitsPerEm)/2) / int64(f.unitsPerEm))
	}
	return int((int64(v)*1000 - int64(f.unitsPerEm)/2) / int64(f.unitsPerEm))
}
func stemV(weight uint16) int {
	if weight == 0 {
		return 80
	}
	return 50 + int(weight)/5
}

func (f *pdfTTFont) toUnicodeCMap(cids []uint16) []byte {
	var b strings.Builder
	b.WriteString("/CIDInit /ProcSet findresource begin\n12 dict begin\nbegincmap\n/CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def\n/CMapName /Adobe-Identity-UCS def\n/CMapType 2 def\n1 begincodespacerange\n<0000> <FFFF>\nendcodespacerange\n")
	for start := 0; start < len(cids); start += 100 {
		end := start + 100
		if end > len(cids) {
			end = len(cids)
		}
		fmt.Fprintf(&b, "%d beginbfchar\n", end-start)
		for _, cid := range cids[start:end] {
			fmt.Fprintf(&b, "<%04X> <%s>\n", cid, utf16Hex(f.cidToRune[cid]))
		}
		b.WriteString("endbfchar\n")
	}
	b.WriteString("endcmap\nCMapName currentdict /CMap defineresource pop\nend\nend\n")
	return []byte(b.String())
}

func utf16Hex(r rune) string {
	units := utf16.Encode([]rune{r})
	var b strings.Builder
	for _, u := range units {
		fmt.Fprintf(&b, "%04X", u)
	}
	return b.String()
}
func decodeUTF16BE(data []byte) string {
	if len(data)%2 != 0 {
		return ""
	}
	units := make([]uint16, len(data)/2)
	for i := range units {
		units[i] = binary.BigEndian.Uint16(data[i*2:])
	}
	return string(utf16.Decode(units))
}
func sanitizePDFName(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
func u16(data []byte, off int) uint16 { return binary.BigEndian.Uint16(data[off : off+2]) }
func i16(data []byte, off int) int16  { return int16(u16(data, off)) }
func u32(data []byte, off int) uint32 { return binary.BigEndian.Uint32(data[off : off+4]) }
