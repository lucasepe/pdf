package pdf

import (
	"fmt"
	"math"
	"strings"
)

type pdfDestination struct {
	page int
	x, y float64
}

type pdfLink struct {
	rect        Rect
	uri, target string
}

// AddDestination registers name at the transformed point (x, y) on the
// current page. Internal links may refer to destinations declared later.
func (c *Context) AddDestination(name string, x, y float64) *Context {
	if c.err != nil {
		return c
	}
	if strings.TrimSpace(name) == "" || strings.ContainsRune(name, 0) {
		return c.fail("add destination", "name must be non-empty and contain no NUL byte")
	}
	if !finite(x, y) {
		return c.fail("add destination", "non-finite point")
	}
	c.doc.reservePage()
	if _, exists := c.doc.destinations[name]; exists {
		return c.fail("add destination", fmt.Sprintf("duplicate destination %q", name))
	}
	tx, ty := c.state.transform.TransformPoint(x, y)
	c.doc.destinations[name] = pdfDestination{page: c.doc.pageNo, x: tx, y: c.doc.paperSize.heightPt - ty}
	return c
}

// AddExternalLink adds a clickable URI annotation over rect on the current
// page. Rect uses the context's transformed top-left coordinate system.
func (c *Context) AddExternalLink(uri string, rect Rect) *Context {
	if strings.TrimSpace(uri) == "" || strings.ContainsRune(uri, 0) {
		return c.fail("add external link", "URI must be non-empty and contain no NUL byte")
	}
	return c.addLink(pdfLink{rect: rect, uri: uri}, "add external link")
}

// AddInternalLink adds a clickable GoTo annotation over rect. The destination
// may be registered before or after the link.
func (c *Context) AddInternalLink(destination string, rect Rect) *Context {
	if strings.TrimSpace(destination) == "" || strings.ContainsRune(destination, 0) {
		return c.fail("add internal link", "destination must be non-empty and contain no NUL byte")
	}
	return c.addLink(pdfLink{rect: rect, target: destination}, "add internal link")
}

func (c *Context) addLink(link pdfLink, operation string) *Context {
	if c.err != nil {
		return c
	}
	r := link.rect
	if !finite(r.X, r.Y, r.Width, r.Height) || r.Width <= 0 || r.Height <= 0 {
		return c.fail(operation, "rectangle must be finite and have positive dimensions")
	}
	c.doc.reservePage()
	points := [][2]float64{{r.X, r.Y}, {r.X + r.Width, r.Y}, {r.X, r.Y + r.Height}, {r.X + r.Width, r.Y + r.Height}}
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, point := range points {
		x, y := c.state.transform.TransformPoint(point[0], point[1])
		y = c.doc.paperSize.heightPt - y
		minX, maxX = math.Min(minX, x), math.Max(maxX, x)
		minY, maxY = math.Min(minY, y), math.Max(maxY, y)
	}
	link.rect = Rect{X: minX, Y: minY, Width: maxX - minX, Height: maxY - minY}
	c.doc.page.links = append(c.doc.page.links, link)
	return c
}
