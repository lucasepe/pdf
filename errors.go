package pdf

import (
	"fmt"
	"runtime"
	"strings"
	"unicode"
)

// Clean clears all accumulated errors.
func (p *PDF) Clean() *PDF { p.errors = nil; return p }

// ErrorInfo extracts and returns additional error details from PDF errors
func (*PDF) ErrorInfo(err error) (ret struct {
	ID            int
	Msg, Src, Val string
}) {
	if err, isT := err.(pdfError); isT {
		ret.ID, ret.Msg, ret.Src, ret.Val = err.id, err.msg, err.src, err.val
	}
	return ret
}

// Errors returns a slice of all accumulated errors.
func (p *PDF) Errors() []error { return p.errors }

// PullError removes and returns the first error from the errors collection.
func (p *PDF) PullError() error {
	if len(p.errors) == 0 {
		return nil
	}
	ret := p.errors[0]
	p.errors = p.errors[1:]
	return ret
}

// pdfError stores extended error details for errors in this package.
type pdfError struct {
	id            int    // unique ID of the error (only within package)
	msg, src, val string // the error message, source method and invalid value
} //                                                                    pdfError

// Error creates and returns an error message from pdfError details
func (err pdfError) Error() string {
	ret := fmt.Sprintf("%s %q", err.msg, err.val)
	if err.src != "" {
		ret += " @" + err.src
	}
	return ret
}

// putError appends an error to the errors collection
func (p *PDF) putError(id int, msg, val string) *PDF {
	var fn string //                                  get the public method name
	for i := 0; i < 10; i++ {
		programCounter, _, _, _ := runtime.Caller(i)
		fn = runtime.FuncForPC(programCounter).Name()
		fn = fn[strings.LastIndex(fn, ".")+1:]
		if unicode.IsLower(rune(fn[0])) {
			continue
		}
		break
	}
	p.errors = append(p.errors, pdfError{id: id, src: fn, msg: msg, val: val})
	return p
}
