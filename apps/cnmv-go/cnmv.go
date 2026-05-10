package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// CNMVClient scrapes the public listing page and downloads monthly ZIPs.
// Download URLs are tokenized — the only deterministic identifier is the
// "title" attribute on the <a> tag (Spanish month name).
type CNMVClient struct {
	listURL    string
	httpClient *http.Client
	userAgent  string
}

func NewCNMVClient() *CNMVClient {
	return &CNMVClient{
		listURL: "https://www.cnmv.es/portal/Publicaciones/Descarga-Informacion-Individual.aspx",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		// Some CNMV endpoints return 403 without a realistic UA.
		userAgent: "ticker-lab-cnmv-go/1.0 (+https://tickerlab.dev)",
	}
}

// monthNumber translates Spanish month names (case-insensitive) to 1..12.
// CNMV uses these as the `title` attribute on the download links.
var monthNumber = map[string]int{
	"enero": 1, "febrero": 2, "marzo": 3, "abril": 4,
	"mayo": 5, "junio": 6, "julio": 7, "agosto": 8,
	"septiembre": 9, "setiembre": 9, "octubre": 10,
	"noviembre": 11, "diciembre": 12,
}

// linkPattern matches each monthly download link on the page. The id
// suffix (_ctlNN_lnkZip) varies but is stable in structure; the title
// carries the month name.
var linkPattern = regexp.MustCompile(
	`<a[^>]*id="[^"]*_ctl\d+_lnkZip"[^>]*href="(https://www\.cnmv\.es/webservices/verdocumento/ver\?e=[^"]+)"[^>]*title="([^"]+)"`,
)

// ListMonthlyZips fetches the listing for `year` and returns one entry per
// month found.
func (c *CNMVClient) ListMonthlyZips(year int) ([]MonthlyZip, error) {
	url := fmt.Sprintf("%s?ano=%d", c.listURL, year)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build list request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read list body: %w", err)
	}

	var out []MonthlyZip
	matches := linkPattern.FindAllSubmatch(body, -1)
	for _, m := range matches {
		// m[1] = URL, m[2] = title (e.g. "Noviembre")
		title := strings.ToLower(strings.TrimSpace(string(m[2])))
		// Spanish month names sometimes have accents; strip them.
		title = removeAccents(title)
		month, ok := monthNumber[title]
		if !ok {
			continue
		}
		out = append(out, MonthlyZip{
			Year:  year,
			Month: month,
			URL:   string(m[1]),
		})
	}
	return out, nil
}

// removeAccents is a tiny ASCII-folder for the Spanish month names we care
// about. Pulling in a full Unicode library would be overkill.
func removeAccents(s string) string {
	r := strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u",
		"Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U",
	)
	return r.Replace(s)
}

// DownloadZip fetches a tokenized URL and returns the FONDREGISTRO and
// FONDMENS XML bytes from the archive. Other files in the ZIP (XSDs, PDFs)
// are ignored.
func (c *CNMVClient) DownloadZip(url string) (registroXML, mensXML []byte, err error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("build zip request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/zip,application/octet-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("zip request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("zip returned %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read zip body: %w", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, nil, fmt.Errorf("open zip: %w", err)
	}

	for _, f := range zr.File {
		name := f.Name
		switch {
		case strings.HasPrefix(name, "FONDREGISTRO_") && strings.HasSuffix(name, ".xml"):
			registroXML, err = readZipFile(f)
		case strings.HasPrefix(name, "FONDMENS_") && strings.HasSuffix(name, ".xml"):
			mensXML, err = readZipFile(f)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("read %s: %w", name, err)
		}
	}
	if registroXML == nil || mensXML == nil {
		return nil, nil, fmt.Errorf("zip missing FONDREGISTRO or FONDMENS")
	}
	return registroXML, mensXML, nil
}

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}
