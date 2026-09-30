package pdf

import "fmt"

// TextEncodingError reports a rune that cannot be represented by a PDF text
// encoding supported by this package.
type TextEncodingError struct {
	Encoding   string
	Font       string
	Rune       rune
	ByteOffset int
	RuneIndex  int
}

func (e TextEncodingError) Error() string {
	font := ""
	if e.Font != "" {
		font = fmt.Sprintf(" for font %q", e.Font)
	}
	return fmt.Sprintf("%s cannot encode %U%s at byte %d (rune %d)",
		e.Encoding, e.Rune, font, e.ByteOffset, e.RuneIndex)
}

func encodeWinAnsi(s string) ([]byte, error) {
	return encodeSingleByte(s, "WinAnsiEncoding", winAnsiByte)
}

func encodePDFDoc(s string) ([]byte, error) {
	return encodeSingleByte(s, "PDFDocEncoding", pdfDocByte)
}

func encodeSingleByte(s, name string, lookup func(rune) (byte, bool)) ([]byte, error) {
	out := make([]byte, 0, len(s))
	runeIndex := 0
	for byteOffset, r := range s {
		b, ok := lookup(r)
		if !ok {
			return nil, TextEncodingError{
				Encoding:   name,
				Rune:       r,
				ByteOffset: byteOffset,
				RuneIndex:  runeIndex,
			}
		}
		out = append(out, b)
		runeIndex++
	}
	return out, nil
}

func winAnsiByte(r rune) (byte, bool) {
	if r >= 0x20 && r <= 0x7e {
		return byte(r), true
	}
	if r >= 0xa0 && r <= 0xff {
		return byte(r), true
	}
	b, ok := winAnsiSpecial[r]
	return b, ok
}

var winAnsiSpecial = map[rune]byte{
	'€': 0x80,
	'‚': 0x82,
	'ƒ': 0x83,
	'„': 0x84,
	'…': 0x85,
	'†': 0x86,
	'‡': 0x87,
	'ˆ': 0x88,
	'‰': 0x89,
	'Š': 0x8a,
	'‹': 0x8b,
	'Œ': 0x8c,
	'Ž': 0x8e,
	'‘': 0x91,
	'’': 0x92,
	'“': 0x93,
	'”': 0x94,
	'•': 0x95,
	'–': 0x96,
	'—': 0x97,
	'˜': 0x98,
	'™': 0x99,
	'š': 0x9a,
	'›': 0x9b,
	'œ': 0x9c,
	'ž': 0x9e,
	'Ÿ': 0x9f,
}

func pdfDocByte(r rune) (byte, bool) {
	if r >= 0x20 && r <= 0x7e {
		return byte(r), true
	}
	if r >= 0xa1 && r <= 0xff {
		return byte(r), true
	}
	b, ok := pdfDocSpecial[r]
	return b, ok
}

var pdfDocSpecial = map[rune]byte{
	'˘': 0x18,
	'ˇ': 0x19,
	'ˆ': 0x1a,
	'˙': 0x1b,
	'˝': 0x1c,
	'˛': 0x1d,
	'˚': 0x1e,
	'˜': 0x1f,
	'•': 0x80,
	'†': 0x81,
	'‡': 0x82,
	'…': 0x83,
	'—': 0x84,
	'–': 0x85,
	'ƒ': 0x86,
	'⁄': 0x87,
	'‹': 0x88,
	'›': 0x89,
	'−': 0x8a,
	'‰': 0x8b,
	'„': 0x8c,
	'“': 0x8d,
	'”': 0x8e,
	'‘': 0x8f,
	'’': 0x90,
	'‚': 0x91,
	'™': 0x92,
	'ﬁ': 0x93,
	'ﬂ': 0x94,
	'Ł': 0x95,
	'Œ': 0x96,
	'Š': 0x97,
	'Ÿ': 0x98,
	'Ž': 0x99,
	'ı': 0x9a,
	'ł': 0x9b,
	'œ': 0x9c,
	'š': 0x9d,
	'ž': 0x9e,
	'€': 0xa0,
}

func escapePDFBytes(src []byte) string {
	out := make([]byte, 0, len(src))
	for _, b := range src {
		if b == '(' || b == ')' || b == '\\' {
			out = append(out, '\\')
		}
		out = append(out, b)
	}
	return string(out)
}
