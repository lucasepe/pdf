package pdf

import (
	"fmt"
	"math"
)

// FontRole identifies a semantic document typeface role.
type FontRole string

const (
	FontRoleBody           FontRole = "body"
	FontRoleBodyBold       FontRole = "body-bold"
	FontRoleBodyItalic     FontRole = "body-italic"
	FontRoleBodyBoldItalic FontRole = "body-bold-italic"
	FontRoleMono           FontRole = "mono"
	FontRoleMonoBold       FontRole = "mono-bold"
)

// FontRoleError reports an invalid or unbound semantic font role.
type FontRoleError struct {
	Role   FontRole
	Detail string
}

func (e FontRoleError) Error() string {
	return fmt.Sprintf("font role %q: %s", e.Role, e.Detail)
}

// BindFontRole associates a semantic role with a registered embedded font or
// a PDF base-14 font name.
func (p *PDF) BindFontRole(role FontRole, fontName string) error {
	p.init()
	if !validFontRole(role) {
		return FontRoleError{Role: role, Detail: "unknown role"}
	}
	if fontName == "" {
		return FontRoleError{Role: role, Detail: "empty font name"}
	}
	key := p.toUpperLettersDigits(fontName, "")
	if isBodyFontRole(role) && p.registeredFonts[key] == nil {
		return FontRoleError{Role: role, Detail: fmt.Sprintf("body font %q is not registered", fontName)}
	}
	if _, builtIn := p.currentBuiltInFontNamed(fontName); !builtIn && p.registeredFonts[key] == nil {
		return FontRoleError{Role: role, Detail: fmt.Sprintf("font %q is not registered", fontName)}
	}
	p.fontRoles[role] = fontName
	return nil
}

func isBodyFontRole(role FontRole) bool {
	switch role {
	case FontRoleBody, FontRoleBodyBold, FontRoleBodyItalic, FontRoleBodyBoldItalic:
		return true
	default:
		return false
	}
}

// FontForRole returns the font name currently associated with role.
func (p *PDF) FontForRole(role FontRole) (string, error) {
	p.init()
	if !validFontRole(role) {
		return "", FontRoleError{Role: role, Detail: "unknown role"}
	}
	name := p.fontRoles[role]
	if name == "" {
		return "", FontRoleError{Role: role, Detail: "not bound"}
	}
	return name, nil
}

// UseFontRole selects the font associated with role at size points.
func (p *PDF) UseFontRole(role FontRole, size float64) error {
	if size <= 0 || math.IsNaN(size) || math.IsInf(size, 0) {
		return FontRoleError{Role: role, Detail: "font size must be finite and positive"}
	}
	name, err := p.FontForRole(role)
	if err != nil {
		return err
	}
	p.SetFont(name, size)
	return nil
}

func validFontRole(role FontRole) bool {
	switch role {
	case FontRoleBody, FontRoleBodyBold, FontRoleBodyItalic,
		FontRoleBodyBoldItalic, FontRoleMono, FontRoleMonoBold:
		return true
	default:
		return false
	}
}
