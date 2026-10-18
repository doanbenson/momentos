package business

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var suffixByMIME = map[string]string{
	"image/jpeg":      ".jpg",
	"image/jpg":       ".jpg",
	"image/png":       ".png",
	"image/heic":      ".heic",
	"image/heif":      ".heif",
	"image/webp":      ".webp",
	"video/mp4":       ".mp4",
	"video/quicktime": ".mov",
	"video/webm":      ".webm",
	"audio/mp4":       ".m4a",
	"audio/m4a":       ".m4a",
	"audio/mpeg":      ".mp3",
	"audio/wav":       ".wav",
	"audio/x-wav":     ".wav",
	"audio/webm":      ".webm",
}

var suffixByAssetType = map[string]string{
	"photo": ".jpg",
	"video": ".mp4",
	"voice": ".m4a",
}

// SuffixFor picks a file extension from Content-Type, then Asset type fallback.
func SuffixFor(contentType, assetType string) string {
	if contentType != "" {
		mime := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
		if s, ok := suffixByMIME[mime]; ok {
			return s
		}
	}
	if s, ok := suffixByAssetType[strings.ToLower(strings.TrimSpace(assetType))]; ok {
		return s
	}
	return ".bin"
}

func isHEICPath(path, contentType string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".heic" || ext == ".heif" {
		return true
	}
	mime := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	return mime == "image/heic" || mime == "image/heif"
}

type downloadedFile struct {
	Path        string
	ContentType string
}

func downloadURLTyped(client *http.Client, rawURL, destBase, assetType string) (*downloadedFile, error) {
	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, rawURL)
	}
	ct := resp.Header.Get("Content-Type")
	suffix := SuffixFor(ct, assetType)
	dest := destBase + suffix
	f, err := os.Create(dest)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = os.Remove(dest)
		return nil, err
	}
	return &downloadedFile{Path: dest, ContentType: ct}, nil
}
