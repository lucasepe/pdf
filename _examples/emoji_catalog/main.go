package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	pdf "github.com/lucasepe/pdf"
)

const (
	twemojiVersion = "17.0.3"
	defaultBaseURL = "https://cdn.jsdelivr.net/gh/jdecked/twemoji@17.0.3/assets/72x72"
	columns        = 5
	rows           = 9
	itemsPerPage   = columns * rows
)

//go:embed catalog.txt
var catalogText string

type catalogEntry struct {
	filename string
	name     string
	sequence string
	image    image.Image
}

type catalogProvider map[string]image.Image

func (p catalogProvider) LookupEmoji(sequence string) (pdf.EmojiAsset, bool, error) {
	img, ok := p[sequence]
	if !ok {
		return pdf.EmojiAsset{}, false, nil
	}
	return pdf.EmojiAsset{Image: img, HeightEm: 1.15, BaselineEm: -.18, AdvanceEm: 1.22}, true, nil
}

func main() {
	entries, err := parseCatalog(catalogText)
	if err != nil {
		log.Fatal(err)
	}
	cacheDir, err := emojiCacheDir()
	if err != nil {
		log.Fatal(err)
	}
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("PDF_TWEMOJI_BASE_URL")), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if err := loadAll(entries, cacheDir, baseURL); err != nil {
		log.Fatal(err)
	}

	provider := make(catalogProvider, len(entries))
	for _, entry := range entries {
		provider[entry.sequence] = entry.image
	}
	doc := pdf.NewPDF("A4")
	doc.SetDocTitle("Twemoji catalog").
		SetDocSubject(fmt.Sprintf("%d unique emoji sequences recognized by ebookgen", len(entries))).
		SetDocCreator("github.com/lucasepe/pdf emoji catalog example")
	if err := doc.SetEmojiProvider(provider); err != nil {
		log.Fatal(err)
	}
	c := pdf.NewContext(&doc)
	totalPages := (len(entries) + itemsPerPage - 1) / itemsPerPage
	for index, entry := range entries {
		position := index % itemsPerPage
		if position == 0 {
			if index > 0 {
				c.AddPage()
			}
			drawHeader(c, len(entries), index/itemsPerPage+1, totalPages)
		}
		column := position % columns
		row := position / columns
		x := 31.0 + float64(column)*108
		y := 111.0 + float64(row)*76
		c.DrawEmojiAt(entry.sequence, x+37, y+31, 31)
		c.UseFontRole(pdf.FontRoleMonoBold, 6.8).
			DrawString(shorten(entry.name, 22), x, y+49)
		c.UseFontRole(pdf.FontRoleMono, 5.2).
			DrawString(shorten(entry.filename, 29), x, y+61)
	}

	if err := c.SaveFile("_examples/emoji_catalog/emoji-catalog.pdf"); err != nil {
		log.Fatal(err)
	}
	log.Printf("generated %d emoji across %d pages", len(entries), totalPages)
}

func drawHeader(c *pdf.Context, count, page, pages int) {
	c.UseFontRole(pdf.FontRoleMonoBold, 20).DrawString("Twemoji catalog", 31, 43)
	c.UseFontRole(pdf.FontRoleMono, 7.5).
		DrawString(fmt.Sprintf("%d ebookgen sequences · Twemoji %s · page %d/%d", count, twemojiVersion, page, pages), 31, 60)
	c.UseFontRole(pdf.FontRoleMono, 5.8).
		DrawString("Graphics: Twemoji project, CC BY 4.0 · github.com/jdecked/twemoji", 31, 78)
}

func parseCatalog(raw string) ([]*catalogEntry, error) {
	var entries []*catalogEntry
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "\t", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid catalog line %q", scanner.Text())
		}
		entries = append(entries, &catalogEntry{filename: parts[0], name: parts[1], sequence: parts[2]})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func emojiCacheDir() (string, error) {
	if value := strings.TrimSpace(os.Getenv("PDF_TWEMOJI_CACHE")); value != "" {
		return value, nil
	}
	root, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "lucasepe-pdf", "twemoji-"+twemojiVersion), nil
}

func loadAll(entries []*catalogEntry, cacheDir, baseURL string) error {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}
	jobs := make(chan *catalogEntry)
	errs := make(chan error, 1)
	client := &http.Client{Timeout: 30 * time.Second}
	var workers sync.WaitGroup
	for range 16 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for entry := range jobs {
				img, err := loadEmoji(client, cacheDir, baseURL, entry)
				if err != nil {
					select {
					case errs <- fmt.Errorf("load %s (%s): %w", entry.name, entry.filename, err):
					default:
					}
					continue
				}
				entry.image = img
			}
		}()
	}
	for _, entry := range entries {
		jobs <- entry
	}
	close(jobs)
	workers.Wait()
	select {
	case err := <-errs:
		return err
	default:
		return nil
	}
}

func loadEmoji(client *http.Client, cacheDir, baseURL string, entry *catalogEntry) (image.Image, error) {
	path := filepath.Join(cacheDir, entry.filename+".png")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		filenames := []string{entry.filename}
		if fallback := filenameWithVariationSelectors(entry.sequence); fallback != entry.filename {
			filenames = append(filenames, fallback)
		}
		for _, filename := range filenames {
			response, getErr := client.Get(baseURL + "/" + filename + ".png")
			if getErr != nil {
				return nil, getErr
			}
			if response.StatusCode == http.StatusOK {
				data, err = io.ReadAll(io.LimitReader(response.Body, 4<<20))
				_ = response.Body.Close()
				break
			}
			status := response.Status
			_ = response.Body.Close()
			if response.StatusCode != http.StatusNotFound {
				return nil, fmt.Errorf("HTTP %s", status)
			}
		}
		if len(data) == 0 && err == nil {
			return nil, fmt.Errorf("HTTP 404 Not Found")
		}
		if err == nil {
			err = os.WriteFile(path, data, 0o644)
		}
	}
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}

func filenameWithVariationSelectors(cluster string) string {
	var codepoints []string
	for _, r := range cluster {
		if r != '\ufe0e' {
			codepoints = append(codepoints, fmt.Sprintf("%x", r))
		}
	}
	return strings.Join(codepoints, "-")
}

func shorten(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit-1] + "~"
}
