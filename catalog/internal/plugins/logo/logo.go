package logo

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
)

// maxLogoBytes bounds the size of a decoded data-URI logo. Logos are icons
// (current catalog entries are all well under 10KB); this cap prevents an
// oversized data URI from forcing a large per-request in-memory allocation.
const maxLogoBytes = 1 << 20 // 1 MiB

// allowedLogoTypes restricts the Content-Type served for data-URI logos to
// known image mediatypes. Because this endpoint serves bytes from the catalog's
// own origin (so downstream systems can reference logos by URL), an attacker who
// controls a logo value could otherwise have us serve text/html or other active
// content under our origin. See svgContentType handling for the SVG case.
var allowedLogoTypes = map[string]bool{
	"image/png":                true,
	"image/jpeg":               true,
	"image/gif":                true,
	"image/webp":               true,
	"image/svg+xml":            true,
	"image/x-icon":             true,
	"image/vnd.microsoft.icon": true,
}

const svgContentType = "image/svg+xml"

// Serve returns a catalog logo as image bytes or redirects to its external URL.
func Serve(w http.ResponseWriter, r *http.Request, value string) {
	// The URI scheme is case-insensitive (RFC 2397), so normalize before
	// deciding how to handle the value.
	if strings.HasPrefix(strings.ToLower(value), "data:") {
		serveDataURI(w, value)
		return
	}

	// Plain-URL logos are redirected. Restrict to http(s) so the value
	// cannot be used as an open redirect into other schemes (javascript:,
	// data:, etc.) or non-URL garbage.
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		http.Error(w, "unsupported logo URL", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, value, http.StatusFound)
}

func serveDataURI(w http.ResponseWriter, dataURI string) {
	// data:[<mediatype>][;base64],<data>
	// The caller has already verified the "data:" prefix (case-insensitively),
	// so strip it by length rather than a case-sensitive TrimPrefix.
	rest := dataURI[len("data:"):]
	metaAndData, data, found := strings.Cut(rest, ",")
	if !found {
		http.Error(w, "malformed data URI", http.StatusBadRequest)
		return
	}

	mimeType := ""
	isBase64 := false
	for i, part := range strings.Split(metaAndData, ";") {
		if i == 0 && strings.Contains(part, "/") {
			mimeType = strings.ToLower(strings.TrimSpace(part))
		} else if part == "base64" {
			isBase64 = true
		}
	}

	// Only serve known image mediatypes. This endpoint returns the bytes from
	// our own origin, so serving an attacker-chosen Content-Type (e.g.
	// text/html) would enable stored XSS in the catalog origin.
	if !allowedLogoTypes[mimeType] {
		http.Error(w, "unsupported logo media type", http.StatusUnsupportedMediaType)
		return
	}

	var body []byte
	if isBase64 {
		// Reject before decoding if the payload would exceed the cap, then
		// clamp after decoding as a defense-in-depth bound.
		if base64.StdEncoding.DecodedLen(len(data)) > maxLogoBytes {
			http.Error(w, "logo too large", http.StatusRequestEntityTooLarge)
			return
		}
		var err error
		body, err = base64.StdEncoding.DecodeString(data)
		if err != nil {
			body, err = base64.RawStdEncoding.DecodeString(data)
			if err != nil {
				http.Error(w, "invalid base64 data", http.StatusBadRequest)
				return
			}
		}
	} else {
		// Reject before unescaping if the payload is oversized. Percent-decoding
		// never expands the input (each %XX triple decodes to one byte), so the
		// encoded length is a safe upper bound on the decoded size and lets us
		// bail out before allocating the decoded copy.
		if len(data) > maxLogoBytes {
			http.Error(w, "logo too large", http.StatusRequestEntityTooLarge)
			return
		}
		// Non-base64 data URIs percent-encode their payload (RFC 2397), e.g.
		// data:image/svg+xml,%3Csvg%3E...%3C/svg%3E. Decode it so the browser
		// receives the real bytes instead of the literal %XX escapes.
		decoded, err := url.PathUnescape(data)
		if err != nil {
			http.Error(w, "invalid data URI encoding", http.StatusBadRequest)
			return
		}
		body = []byte(decoded)
	}
	if len(body) > maxLogoBytes {
		http.Error(w, "logo too large", http.StatusRequestEntityTooLarge)
		return
	}

	// Prevent MIME sniffing from reinterpreting the bytes as a different type.
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// SVG is an active document format: if navigated to directly it can run
	// embedded scripts in our origin. A restrictive CSP keeps the image
	// renderable via <img> while neutralizing scripting/external references.
	if mimeType == svgContentType {
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")
	}

	w.Header().Set("Content-Type", mimeType)
	// Make the intended rendering explicit rather than relying on browser
	// defaults: logos are meant to be displayed inline (e.g. via <img>).
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(body)
}
