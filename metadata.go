package pdf

// DocAuthor returns the optional 'document author' metadata property.
func (p *PDF) DocAuthor() string { p.init(); return p.docAuthor }

// SetDocAuthor sets the optional 'document author' metadata property.
func (p *PDF) SetDocAuthor(s string) *PDF { p.docAuthor = s; return p }

// DocCreator returns the optional 'document creator' metadata property.
func (p *PDF) DocCreator() string { p.init(); return p.docCreator }

// SetDocCreator sets the optional 'document creator' metadata property.
func (p *PDF) SetDocCreator(s string) *PDF { p.docCreator = s; return p }

// DocKeywords returns the optional 'document keywords' metadata property.
func (p *PDF) DocKeywords() string { p.init(); return p.docKeywords }

// SetDocKeywords sets the optional 'document keywords' metadata property.
func (p *PDF) SetDocKeywords(s string) *PDF { p.docKeywords = s; return p }

// DocSubject returns the optional 'document subject' metadata property.
func (p *PDF) DocSubject() string { p.init(); return p.docSubject }

// SetDocSubject sets the optional 'document subject' metadata property.
func (p *PDF) SetDocSubject(s string) *PDF { p.docSubject = s; return p }

// DocTitle returns the optional 'document subject' metadata property.
func (p *PDF) DocTitle() string { p.init(); return p.docTitle }

// SetDocTitle sets the optional 'document title' metadata property.
func (p *PDF) SetDocTitle(s string) *PDF { p.docTitle = s; return p }
