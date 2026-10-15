package pdf

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math/bits"
	"sort"
)

// subsetProgram keeps the original glyph IDs. PDF CIDToGIDMap and composite
// component references therefore remain valid; unused outlines become empty
// loca entries. The resulting sfnt has a fresh directory and checksums.
func (f *pdfTTFont) subsetProgram() ([]byte, string, error) {
	used := make([]bool, f.numGlyphs)
	used[0] = true // .notdef is required even when no text was drawn.
	for _, gid := range f.cidToGID {
		if int(gid) >= len(used) {
			return nil, "", FontError{Kind: "malformed", Detail: "glyph ID exceeds maxp"}
		}
		used[gid] = true
	}
	loca, _ := f.table("loca", 0)
	glyf, _ := f.table("glyf", 0)
	head, _ := f.table("head", 54)
	longLoca := i16(head, 50) == 1
	for gid := range used {
		if used[gid] {
			if err := f.includeCompositeComponents(uint16(gid), used, loca, glyf, longLoca); err != nil {
				return nil, "", err
			}
		}
	}

	// Retain the core sfnt tables, plus the TrueType hint program when present.
	// Layout tables are unnecessary because text is already positioned by PDF.
	tags := []string{"OS/2", "cmap", "glyf", "head", "hhea", "hmtx", "loca", "maxp", "name", "post"}
	for _, optional := range []string{"cvt ", "fpgm", "prep", "gasp"} {
		if _, ok := f.tables[optional]; ok {
			tags = append(tags, optional)
		}
	}
	sort.Strings(tags)
	tables := make(map[string][]byte, len(tags))
	for _, tag := range tags {
		data, err := f.table(tag, 0)
		if err != nil {
			return nil, "", err
		}
		tables[tag] = append([]byte(nil), data...)
	}
	newHead := tables["head"]
	binary.BigEndian.PutUint32(newHead[8:12], 0)  // checkSumAdjustment
	binary.BigEndian.PutUint16(newHead[50:52], 1) // long loca offsets

	newLoca := make([]byte, (int(f.numGlyphs)+1)*4)
	newGlyf := make([]byte, 0, len(glyf)/4)
	for gid := range used {
		binary.BigEndian.PutUint32(newLoca[gid*4:], uint32(len(newGlyf)))
		if !used[gid] {
			continue
		}
		start, end := glyphOffsets(loca, uint16(gid), longLoca)
		newGlyf = append(newGlyf, glyf[start:end]...)
		for len(newGlyf)%4 != 0 {
			newGlyf = append(newGlyf, 0)
		}
	}
	binary.BigEndian.PutUint32(newLoca[len(used)*4:], uint32(len(newGlyf)))
	tables["loca"], tables["glyf"] = newLoca, newGlyf

	count := len(tags)
	if count > 0xffff || 12+count*16 > maxFontSize {
		return nil, "", FontError{Kind: "subset too large", Detail: f.logicalName}
	}
	result := make([]byte, 12+count*16)
	copy(result[:4], f.data[:4])
	binary.BigEndian.PutUint16(result[4:6], uint16(count))
	maxPower := 1 << (bits.Len(uint(count)) - 1)
	binary.BigEndian.PutUint16(result[6:8], uint16(maxPower*16))
	binary.BigEndian.PutUint16(result[8:10], uint16(bits.Len(uint(maxPower))-1))
	binary.BigEndian.PutUint16(result[10:12], uint16(count*16-maxPower*16))
	var headOffset int
	for index, tag := range tags {
		data := tables[tag]
		if len(result)+len(data)+3 > maxFontSize {
			return nil, "", FontError{Kind: "subset too large", Detail: f.logicalName}
		}
		entry := result[12+index*16 : 12+(index+1)*16]
		copy(entry[:4], tag)
		binary.BigEndian.PutUint32(entry[4:8], sfntChecksum(data))
		binary.BigEndian.PutUint32(entry[8:12], uint32(len(result)))
		binary.BigEndian.PutUint32(entry[12:16], uint32(len(data)))
		if tag == "head" {
			headOffset = len(result)
		}
		result = append(result, data...)
		for len(result)%4 != 0 {
			result = append(result, 0)
		}
	}
	binary.BigEndian.PutUint32(result[headOffset+8:headOffset+12], 0xB1B0AFBA-sfntChecksum(result))
	return result, f.subsetName(used), nil
}

func glyphOffsets(loca []byte, gid uint16, long bool) (int, int) {
	if long {
		return int(u32(loca, int(gid)*4)), int(u32(loca, (int(gid)+1)*4))
	}
	return int(u16(loca, int(gid)*2)) * 2, int(u16(loca, (int(gid)+1)*2)) * 2
}

func (f *pdfTTFont) includeCompositeComponents(gid uint16, used []bool, loca, glyf []byte, long bool) error {
	start, end := glyphOffsets(loca, gid, long)
	if start == end {
		return nil
	}
	data := glyf[start:end]
	if len(data) < 10 {
		return FontError{Kind: "malformed", Detail: fmt.Sprintf("short glyph %d", gid)}
	}
	if i16(data, 0) >= 0 {
		return nil
	}
	pos := 10
	for {
		if pos+4 > len(data) {
			return FontError{Kind: "malformed", Detail: fmt.Sprintf("short composite glyph %d", gid)}
		}
		flags, component := u16(data, pos), u16(data, pos+2)
		pos += 4
		if int(component) >= len(used) {
			return FontError{Kind: "malformed", Detail: fmt.Sprintf("component glyph %d exceeds maxp", component)}
		}
		if !used[component] {
			used[component] = true
			if err := f.includeCompositeComponents(component, used, loca, glyf, long); err != nil {
				return err
			}
		}
		if flags&0x0001 != 0 {
			pos += 4
		} else {
			pos += 2
		}
		switch {
		case flags&0x0008 != 0:
			pos += 2
		case flags&0x0040 != 0:
			pos += 4
		case flags&0x0080 != 0:
			pos += 8
		}
		if pos > len(data) {
			return FontError{Kind: "malformed", Detail: fmt.Sprintf("short composite glyph %d", gid)}
		}
		if flags&0x0020 == 0 {
			if flags&0x0100 != 0 {
				if pos+2 > len(data) || pos+2+int(u16(data, pos)) > len(data) {
					return FontError{Kind: "malformed", Detail: fmt.Sprintf("short composite instructions %d", gid)}
				}
			}
			break
		}
	}
	return nil
}

func sfntChecksum(data []byte) uint32 {
	var sum uint32
	for i := 0; i < len(data); i += 4 {
		var word [4]byte
		copy(word[:], data[i:])
		sum += binary.BigEndian.Uint32(word[:])
	}
	return sum
}

func (f *pdfTTFont) subsetName(used []bool) string {
	h := fnv.New64a()
	_, _ = h.Write(f.data)
	for gid, present := range used {
		if present {
			_, _ = h.Write([]byte{byte(gid >> 8), byte(gid)})
		}
	}
	value := h.Sum64()
	var prefix [6]byte
	for i := range prefix {
		prefix[i] = 'A' + byte(value%26)
		value /= 26
	}
	return string(prefix[:]) + "+" + sanitizePDFName(f.pdfName)
}
