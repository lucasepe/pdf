package pdf

import "image/color"

// ToColor returns an RGBA color value from a web/X11 color name
// (e.g. "HONEY DEW") or HTML color value such as "#191970"
// If the name or code is unknown or invalid, returns zero value (black).
func (p *PDF) ToColor(nameOrHTMLColor string) (color.RGBA, error) {

	s := p.toUpperLettersDigits(nameOrHTMLColor, "#")
	if len(s) >= 7 && s[0] == '#' {
		var hex [6]byte
		for i, r := range s[1:7] {
			switch {
			case r >= '0' && r <= '9':
				hex[i] = byte(r - '0')
			case r >= 'A' && r <= 'F':
				hex[i] = byte(r - 'A' + 10)
			default:
				return pdfBlack, pdfError{id: 0xEED50B, src: "ToColor",
					msg: "Bad color code", val: nameOrHTMLColor}
			}
		}
		return color.RGBA{
			hex[0]*16 + hex[1],
			hex[2]*16 + hex[3],
			hex[4]*16 + hex[5], 255}, nil
	}
	if cl, found := PDFColorNames[s]; found {
		return color.RGBA{cl.R, cl.G, cl.B, 255}, nil
	}
	for k, v := range PDFColorNames {
		if p.toUpperLettersDigits(k, "") == s {
			return v, nil
		}
	}
	return pdfBlack, pdfError{id: 0xE00982, src: "ToColor",
		msg: "Unknown color name", val: nameOrHTMLColor}
}

// PDFColorNames maps web (X11) color names to values.
// From https://en.wikipedia.org/wiki/X11_color_names
var PDFColorNames = map[string]color.RGBA{
	"ALICE BLUE":             {R: 240, G: 248, B: 255, A: 255},
	"ANTIQUE WHITE":          {R: 250, G: 235, B: 215, A: 255},
	"AQUA":                   {R: 000, G: 255, B: 255, A: 255},
	"AQUAMARINE":             {R: 127, G: 255, B: 212, A: 255},
	"AZURE":                  {R: 240, G: 255, B: 255, A: 255},
	"BEIGE":                  {R: 245, G: 245, B: 220, A: 255},
	"BISQUE":                 {R: 255, G: 228, B: 196, A: 255},
	"BLACK":                  {R: 000, G: 000, B: 000, A: 255},
	"BLANCHED ALMOND":        {R: 255, G: 235, B: 205, A: 255},
	"BLUE":                   {R: 000, G: 000, B: 255, A: 255},
	"BLUE VIOLET":            {R: 138, G: 43, B: 226, A: 255},
	"BROWN":                  {R: 165, G: 42, B: 42, A: 255},
	"BURLYWOOD":              {R: 222, G: 184, B: 135, A: 255},
	"CADET BLUE":             {R: 95, G: 158, B: 160, A: 255},
	"CHARTREUSE":             {R: 127, G: 255, B: 000, A: 255},
	"CHOCOLATE":              {R: 210, G: 105, B: 30, A: 255},
	"CORAL":                  {R: 255, G: 127, B: 80, A: 255},
	"CORNFLOWER BLUE":        {R: 100, G: 149, B: 237, A: 255},
	"CORNSILK":               {R: 255, G: 248, B: 220, A: 255},
	"CRIMSON":                {R: 220, G: 20, B: 60, A: 255},
	"CYAN":                   {R: 000, G: 255, B: 255, A: 255},
	"DARK BLUE":              {R: 000, G: 000, B: 139, A: 255},
	"DARK CYAN":              {R: 000, G: 139, B: 139, A: 255},
	"DARK GOLDEN ROD":        {R: 184, G: 134, B: 11, A: 255},
	"DARK GRAY":              {R: 169, G: 169, B: 169, A: 255},
	"DARK GREEN":             {R: 000, G: 100, B: 000, A: 255},
	"DARK KHAKI":             {R: 189, G: 183, B: 107, A: 255},
	"DARK MAGENTA":           {R: 139, G: 000, B: 139, A: 255},
	"DARK OLIVE GREEN":       {R: 85, G: 107, B: 47, A: 255},
	"DARK ORANGE":            {R: 255, G: 140, B: 000, A: 255},
	"DARK ORCHID":            {R: 153, G: 50, B: 204, A: 255},
	"DARK RED":               {R: 139, G: 000, B: 000, A: 255},
	"DARK SALMON":            {R: 233, G: 150, B: 122, A: 255},
	"DARK SEA GREEN":         {R: 143, G: 188, B: 143, A: 255},
	"DARK SLATE BLUE":        {R: 72, G: 61, B: 139, A: 255},
	"DARK SLATE GRAY":        {R: 47, G: 79, B: 79, A: 255},
	"DARK TURQUOISE":         {R: 000, G: 206, B: 209, A: 255},
	"DARK VIOLET":            {R: 148, G: 000, B: 211, A: 255},
	"DEEP PINK":              {R: 255, G: 20, B: 147, A: 255},
	"DEEP SKY BLUE":          {R: 000, G: 191, B: 255, A: 255},
	"DIM GRAY":               {R: 105, G: 105, B: 105, A: 255},
	"DODGER BLUE":            {R: 30, G: 144, B: 255, A: 255},
	"FIRE BRICK":             {R: 178, G: 34, B: 34, A: 255},
	"FLORAL WHITE":           {R: 255, G: 250, B: 240, A: 255},
	"FOREST GREEN":           {R: 34, G: 139, B: 34, A: 255},
	"FUCHSIA":                {R: 255, G: 000, B: 255, A: 255},
	"GAINSBORO":              {R: 220, G: 220, B: 220, A: 255},
	"GHOST WHITE":            {R: 248, G: 248, B: 255, A: 255},
	"GOLD":                   {R: 255, G: 215, B: 000, A: 255},
	"GOLDEN ROD":             {R: 218, G: 165, B: 32, A: 255},
	"GRAY":                   {R: 190, G: 190, B: 190, A: 255},
	"GREEN":                  {R: 000, G: 255, B: 000, A: 255},
	"GREEN YELLOW":           {R: 173, G: 255, B: 47, A: 255},
	"HONEY DEW":              {R: 240, G: 255, B: 240, A: 255},
	"HOT PINK":               {R: 255, G: 105, B: 180, A: 255},
	"INDIAN RED":             {R: 205, G: 92, B: 92, A: 255},
	"INDIGO":                 {R: 75, G: 000, B: 130, A: 255},
	"IVORY":                  {R: 255, G: 255, B: 240, A: 255},
	"KHAKI":                  {R: 240, G: 230, B: 140, A: 255},
	"LAVENDER":               {R: 230, G: 230, B: 250, A: 255},
	"LAVENDER BLUSH":         {R: 255, G: 240, B: 245, A: 255},
	"LAWN GREEN":             {R: 124, G: 252, B: 000, A: 255},
	"LEMON CHIFFON":          {R: 255, G: 250, B: 205, A: 255},
	"LIGHT BLUE":             {R: 173, G: 216, B: 230, A: 255},
	"LIGHT CORAL":            {R: 240, G: 128, B: 128, A: 255},
	"LIGHT CYAN":             {R: 224, G: 255, B: 255, A: 255},
	"LIGHT GOLDENROD YELLOW": {R: 250, G: 250, B: 210, A: 255},
	"LIGHT GRAY":             {R: 211, G: 211, B: 211, A: 255},
	"LIGHT GREEN":            {R: 144, G: 238, B: 144, A: 255},
	"LIGHT PINK":             {R: 255, G: 182, B: 193, A: 255},
	"LIGHT SALMON":           {R: 255, G: 160, B: 122, A: 255},
	"LIGHT SEA GREEN":        {R: 32, G: 178, B: 170, A: 255},
	"LIGHT SKY BLUE":         {R: 135, G: 206, B: 250, A: 255},
	"LIGHT SLATE GRAY":       {R: 119, G: 136, B: 153, A: 255},
	"LIGHT STEEL BLUE":       {R: 176, G: 196, B: 222, A: 255},
	"LIGHT YELLOW":           {R: 255, G: 255, B: 224, A: 255},
	"LIME":                   {R: 000, G: 255, B: 000, A: 255},
	"LIME GREEN":             {R: 50, G: 205, B: 50, A: 255},
	"LINEN":                  {R: 250, G: 240, B: 230, A: 255},
	"MAGENTA":                {R: 255, G: 000, B: 255, A: 255},
	"MAROON":                 {R: 176, G: 48, B: 96, A: 255},
	"MEDIUM AQUAMARINE":      {R: 102, G: 205, B: 170, A: 255},
	"MEDIUM BLUE":            {R: 000, G: 000, B: 205, A: 255},
	"MEDIUM ORCHID":          {R: 186, G: 85, B: 211, A: 255},
	"MEDIUM PURPLE":          {R: 147, G: 112, B: 219, A: 255},
	"MEDIUM SEA GREEN":       {R: 60, G: 179, B: 113, A: 255},
	"MEDIUM SLATE BLUE":      {R: 123, G: 104, B: 238, A: 255},
	"MEDIUM SPRING GREEN":    {R: 000, G: 250, B: 154, A: 255},
	"MEDIUM TURQUOISE":       {R: 72, G: 209, B: 204, A: 255},
	"MEDIUM VIOLET RED":      {R: 199, G: 21, B: 133, A: 255},
	"MIDNIGHT BLUE":          {R: 25, G: 25, B: 112, A: 255},
	"MINT CREAM":             {R: 245, G: 255, B: 250, A: 255},
	"MISTY ROSE":             {R: 255, G: 228, B: 225, A: 255},
	"MOCCASIN":               {R: 255, G: 228, B: 181, A: 255},
	"NAVAJO WHITE":           {R: 255, G: 222, B: 173, A: 255},
	"NAVY":                   {R: 000, G: 000, B: 128, A: 255},
	"OLD LACE":               {R: 253, G: 245, B: 230, A: 255},
	"OLIVE":                  {R: 128, G: 128, B: 000, A: 255},
	"OLIVE DRAB":             {R: 107, G: 142, B: 35, A: 255},
	"ORANGE":                 {R: 255, G: 165, B: 000, A: 255},
	"ORANGE RED":             {R: 255, G: 69, B: 000, A: 255},
	"ORCHID":                 {R: 218, G: 112, B: 214, A: 255},
	"PALE GOLDEN ROD":        {R: 238, G: 232, B: 170, A: 255},
	"PALE GREEN":             {R: 152, G: 251, B: 152, A: 255},
	"PALE TURQUOISE":         {R: 175, G: 238, B: 238, A: 255},
	"PALE VIOLET RED":        {R: 219, G: 112, B: 147, A: 255},
	"PAPAYA WHIP":            {R: 255, G: 239, B: 213, A: 255},
	"PEACH PUFF":             {R: 255, G: 218, B: 185, A: 255},
	"PERU":                   {R: 205, G: 133, B: 63, A: 255},
	"PINK":                   {R: 255, G: 192, B: 203, A: 255},
	"PLUM":                   {R: 221, G: 160, B: 221, A: 255},
	"POWDER BLUE":            {R: 176, G: 224, B: 230, A: 255},
	"PURPLE":                 {R: 160, G: 32, B: 240, A: 255},
	"REBECCA PURPLE":         {R: 102, G: 51, B: 153, A: 255},
	"RED":                    {R: 255, G: 000, B: 000, A: 255},
	"ROSY BROWN":             {R: 188, G: 143, B: 143, A: 255},
	"ROYAL BLUE":             {R: 65, G: 105, B: 225, A: 255},
	"SADDLE BROWN":           {R: 139, G: 69, B: 19, A: 255},
	"SALMON":                 {R: 250, G: 128, B: 114, A: 255},
	"SANDY BROWN":            {R: 244, G: 164, B: 96, A: 255},
	"SEA GREEN":              {R: 46, G: 139, B: 87, A: 255},
	"SEASHELL":               {R: 255, G: 245, B: 238, A: 255},
	"SIENNA":                 {R: 160, G: 82, B: 45, A: 255},
	"SILVER":                 {R: 192, G: 192, B: 192, A: 255},
	"SKY BLUE":               {R: 135, G: 206, B: 235, A: 255},
	"SLATE BLUE":             {R: 106, G: 90, B: 205, A: 255},
	"SLATE GRAY":             {R: 112, G: 128, B: 144, A: 255},
	"SNOW":                   {R: 255, G: 250, B: 250, A: 255},
	"SPRING GREEN":           {R: 000, G: 255, B: 127, A: 255},
	"STEEL BLUE":             {R: 70, G: 130, B: 180, A: 255},
	"TAN":                    {R: 210, G: 180, B: 140, A: 255},
	"TEAL":                   {R: 000, G: 128, B: 128, A: 255},
	"THISTLE":                {R: 216, G: 191, B: 216, A: 255},
	"TOMATO":                 {R: 255, G: 99, B: 71, A: 255},
	"TURQUOISE":              {R: 64, G: 224, B: 208, A: 255},
	"VIOLET":                 {R: 238, G: 130, B: 238, A: 255},
	"WEB GRAY":               {R: 128, G: 128, B: 128, A: 255},
	"WEB GREEN":              {R: 000, G: 128, B: 000, A: 255},
	"WEB MAROON":             {R: 127, G: 000, B: 000, A: 255},
	"WEB PURPLE":             {R: 127, G: 000, B: 127, A: 255},
	"WHEAT":                  {R: 245, G: 222, B: 179, A: 255},
	"WHITE":                  {R: 255, G: 255, B: 255, A: 255},
	"WHITE SMOKE":            {R: 245, G: 245, B: 245, A: 255},
	"YELLOW":                 {R: 255, G: 255, B: 000, A: 255},
	"YELLOW GREEN":           {R: 154, G: 205, B: 50, A: 255},
} //                                                               PDFColorNames

var pdfBlack = color.RGBA{A: 255}
