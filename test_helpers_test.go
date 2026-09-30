package pdf

import (
	"bytes"
	"fmt"

	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// callerList returns a human-friendly list of strings showing the
// call stack with each calling method or function's name and line number.
//
// The most immediate callers are listed first, followed by their callers,
// and so on. For brevity, 'runtime.*' and 'syscall.*'
// and other top-level callers are not included.
func callerList() []string {
	var ret []string
	i := 0
mainLoop:
	for {
		i++
		programCounter, filename, lineNo, _ := runtime.Caller(i)
		funcName := runtime.FuncForPC(programCounter).Name()

		for _, s := range []string{
			"", "runtime.goexit", "runtime.main", "testing.tRunner",
		} {
			if funcName == s {
				break mainLoop
			}
		}
		if strings.Contains(funcName, "HandlerFunc.ServeHTTP") {
			break
		}

		for _, s := range []string{
			".Callers", ".callerList", ".Error", ".Log", ".logAsync",
			"mismatch", "runtime.", "syscall.",
		} {
			if strings.Contains(funcName, s) {
				continue mainLoop
			}
		}
		switch showFileNames {
		case 1:
			filename = filepath.Base(filename)
		case 2:

			if string(os.PathSeparator) != "/" {
				filename = strings.ReplaceAll(filename, "/",
					string(os.PathSeparator))
			}
		}

		if index := strings.LastIndex(funcName, "/"); index != -1 {
			funcName = funcName[index+1:]
		}
		if strings.Count(funcName, ".") > 1 {
			funcName = funcName[strings.Index(funcName, ".")+1:]
		}

		for _, find := range []string{"(", ")", "*"} {
			if strings.Contains(funcName, find) {
				funcName = strings.ReplaceAll(funcName, find, "")
			}
		}
		line := fmt.Sprintf(":%d %s()", lineNo, funcName)
		if showFileNames > 0 {
			line = filename + line
		}
		ret = append(ret, line)
	}
	return ret
}

// failIfHasErrors raises a test failure if the supplied PDF has errors
func failIfHasErrors(t *testing.T, errors func() []error) {
	if len(errors()) == 0 {
		return
	}
	for i, err := range errors() {
		t.Errorf("ERROR %d: %s\n\n", i+1, err)
	}
	t.Fail()
}

// floatStr returns a float64 as a string, with val rounded to 3 decimals
func floatStr(val float64) string {
	return fmt.Sprintf("%0.3f", val)
}

// formatLines accepts an uncompressed PDF document as a string,
// and returns an array of trimmed, non-empty lines
func formatLines(s string, formatStreams bool) []string {

	if formatStreams {
		s = pdfFormatStreams(s)
	}

	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	for _, space := range "\a\b\f\t\v" {
		s = strings.ReplaceAll(s, string(space), " ")
	}

	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	// trim and copy non-blank lines to result
	// also, continue lines that end with '\'
	var (
		ar   = strings.Split(s, "\n")
		ret  = make([]string, 0, len(ar))
		prev = ""
	)
	for _, line := range ar {
		line = strings.Trim(line, " \a\b\f\n\r\t\v")
		if line == "" {
			continue
		}

		if prev != "" {
			line = prev + line
		}
		if strings.HasSuffix(line, "\\") {
			prev = strings.TrimRight(line, "\\")
			continue
		}

		prev = ""
		ret = append(ret, line)
	}
	return ret
}

// getStack returns a list of line numbers and function names on the call stack
func getStack() string {
	buf := make([]byte, 8192)
	runtime.Stack(buf, true)
	var ar []string
	for _, s := range strings.Split(string(buf), "\n") {
		if strings.Contains(s, "\t") && !strings.Contains(s, "/testing.go") {
			ar = append(ar, "<- "+filepath.Base(s))
		}
	}
	return strings.Join(ar, "\n")
}

// mismatch formats and raises a test error
func mismatch(t *testing.T, tag string, want, got any) {
	ws := fmt.Sprintf("%v", want)
	gs := fmt.Sprintf("%v", got)
	t.Errorf("%s mismatch: expected: %s got: %s\n%s",
		tag, ws, gs, getStack())
}

// pdfCompare compares generated result bytes to the expected PDF content:
// - convert result ('got') to a string
// - format both result and expected string using formatLines()
// - compare result and expected lines ('got' and 'want')
// - raise an error if there are diffs (report up to 5 differences)
func pdfCompare(t *testing.T, got []byte, want string) {
	//
	const formatStreams = true
	var (
		gotAr    = canonicalPDFLines(formatLines(string(got), formatStreams))
		wantAr   = canonicalPDFLines(formatLines(want, !formatStreams))
		errCount = 0
		mismatch = false
		max      = len(gotAr)
	)
	if max < len(wantAr) {
		max = len(wantAr)
	}
	for i := 0; i < max; i++ {
		//
		// get the expected and the result line at i
		// if the slice is too short, leave it blank
		var want, got string
		if i < len(wantAr) {
			want = wantAr[i]
		}
		if i < len(gotAr) {
			got = gotAr[i]
		}
		if want == got {
			continue
		}

		mismatch = true
		errCount++
		if errCount > 5 {
			break
		}
		t.Errorf("%s",
			"\n"+
				"/*\n"+
				"LOCATION: "+tCaller()+":\n"+
				"MISMATCH: L"+strconv.Itoa(i+1)+":\n"+
				"EXPECTED: "+want+"\n"+
				"PRODUCED: "+got+"\n"+
				"*/\n")
	}
	if mismatch {
		t.Errorf("%s",
			"\n"+
				"// RETURNED-PDF:\n"+
				"// "+tCaller()+"\n"+
				"`\n"+
				strings.Join(gotAr, "\n")+"\n"+
				"`\n")
	}
}

var (
	mediaBoxPattern = regexp.MustCompile(`/MediaBox\[0 0 [^]]+\]`)
	lengthPattern   = regexp.MustCompile(`/Length [0-9]+`)
	xrefPattern     = regexp.MustCompile(`^[0-9]{10} 00000 n$`)
)

func canonicalPDFLines(lines []string) []string {
	ret := make([]string, 0, len(lines))
	skipCompressedData := false
	normalizeStartXRef := false
	for _, line := range lines {
		if skipCompressedData {
			if line == "endstream" {
				skipCompressedData = false
				ret = append(ret, line)
			}
			continue
		}
		if strings.Contains(line, "/Filter/FlateDecode") && strings.HasSuffix(line, "stream") {
			line = lengthPattern.ReplaceAllString(line, "/Length #")
			ret = append(ret, line)
			skipCompressedData = true
			continue
		}
		if strings.Contains(line, "/MediaBox[") {
			line = mediaBoxPattern.ReplaceAllString(line, "/MediaBox[0 0 # #]")
		}
		if strings.Contains(line, "/Length ") && strings.HasSuffix(line, "stream") {
			line = lengthPattern.ReplaceAllString(line, "/Length #")
		}
		if strings.HasPrefix(line, "/Encoding/") {
			if strings.HasSuffix(line, ">>") {
				ret = append(ret, ">>")
			}
			continue
		}
		if line == "/Resources <<>> >>" && len(ret) > 0 {
			ret[len(ret)-1] += ">>"
			continue
		}
		if xrefPattern.MatchString(line) {
			ret = append(ret, "########## 00000 n")
			continue
		}
		if normalizeStartXRef {
			ret = append(ret, "#")
			normalizeStartXRef = false
			continue
		}
		ret = append(ret, line)
		if line == "startxref" {
			normalizeStartXRef = true
		}
	}
	return ret
}

// pdfFormatStreams formats content of all streams in s as hex strings
func pdfFormatStreams(s string) string {
	const (
		STREAM    = ">> stream"
		ENDSTREAM = "endstream"
		BPL       = 16 // bytes per line
	)
	buf := bytes.NewBuffer(make([]byte, 0, len(s)))
	for part, s := range strings.Split(s, " obj ") {
		if part > 0 {
			buf.WriteString(" obj ")
		}

		i := strings.Index(s, STREAM)
		if i == -1 ||
			(!strings.Contains(s[:i], "/FlateDecode") &&
				!strings.Contains(s[:i], "/Image")) {
			buf.WriteString(s)
			continue
		}

		i += len(STREAM)
		buf.WriteString(s[:i])
		s = s[i:]

		buf.WriteString("\n")
		n := strings.Index(s, ENDSTREAM)
		if n == -1 {
			n = len(s)
		}
		c := 0
		for _, b := range []byte(s[:n]) {
			buf.WriteString(fmt.Sprintf(" %02X", b))
			c++
			if c >= BPL {
				buf.WriteString("\n")
				c = 0
			}
		}
		buf.WriteString("\n")

		s = s[n:]
		if len(s) > 0 {
			buf.WriteString(s)
		}
	}
	return buf.String()
}

// permuteStrings returns all combinations of strings in 'parts'
func permuteStrings(parts ...[]string) (ret []string) {
	{
		n := 1
		for _, ar := range parts {
			n *= len(ar)
		}
		ret = make([]string, 0, n)
	}
	at := make([]int, len(parts))
	var buf bytes.Buffer
loop:
	for {

		for i := len(parts) - 1; i >= 0; i-- {
			if at[i] > 0 && at[i] >= len(parts[i]) {
				if i == 0 || (i == 1 && at[i-1] == len(parts[0])-1) {
					break loop
				}
				at[i] = 0
				at[i-1]++
			}
		}

		buf.Reset()
		for i, ar := range parts {
			j := at[i]
			if j >= 0 && j < len(ar) {
				buf.WriteString(ar[j])
			}
		}
		ret = append(ret, buf.String())
		at[len(parts)-1]++
	}
	return ret
}

// tCaller returns the name of the unit test function.
func tCaller() string {
	for _, caller := range callerList() {
		if strings.Contains(caller, "util.tCaller") ||
			strings.Contains(caller, "util.tEqual") ||
			strings.Contains(caller, "util.pdfCompare") {
			continue
		}
		return caller
	}
	return "<no-caller>"
}

const (
	showFileNames = 1
)

// tEqual asserts that 'got' is equal to 'want'.
// Provides a slightly-altered tEqual() function (and functions it uses)
// from Zircon-Go lib: github.com/balacode/zr
func tEqual(t *testing.T, got any, want any) bool {
	makeStr := func(value any) string {
		switch v := value.(type) {
		case nil:
			{
				return "nil"
			}
		case bool:
			{
				if v {
					return "true"
				}
				return "false"
			}
		case int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64, uintptr:
			{
				return fmt.Sprintf("%d", v)
			}
		case float64, float32:
			{
				s := fmt.Sprintf("%.4f", v)
				if strings.Contains(s, ".") {
					for strings.HasSuffix(s, "0") {
						s = s[:len(s)-1]
					}
					for strings.HasSuffix(s, ".") {
						s = s[:len(s)-1]
					}
				}
				return s
			}
		case error:
			{
				return v.Error()
			}
		case string:
			{
				return v
			}
		case time.Time:
			{
				s := v.Format(time.RFC3339)[:19]
				if strings.HasSuffix(s, "T00:00:00") {
					s = s[:10]
				}
				return s
			}
		case fmt.Stringer:
			{
				return v.String()
			}
		case fmt.GoStringer:
			return v.GoString()
		}
		return fmt.Sprintf("(type: %v value: %v)", reflect.TypeOf(value), value)
	}
	if makeStr(got) != makeStr(want) {
		t.Logf("\n"+"LOCATION: %s\n"+"EXPECTED: %s\n"+"RETURNED: %s\n",
			tCaller(), makeStr(want), makeStr(got))
		t.Fail()
		return false
	}
	return true
}
