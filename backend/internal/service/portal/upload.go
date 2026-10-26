package portal

import "net/http"

// Per-file upload limits.
const (
	MaxAttachmentBytes = 5 << 20
	MaxAssetBytes      = 10 << 20
)

// DetectFileType sniffs the real content type from the file's first bytes and
// accepts only PDF, PNG and JPEG. The client-supplied type and name are never trusted.
func DetectFileType(head []byte) (mime, ext string, ok bool) {
	if len(head) == 0 {
		return "", "", false
	}
	switch detected := http.DetectContentType(head); detected {
	case "application/pdf":
		return detected, ".pdf", true
	case "image/png":
		return detected, ".png", true
	case "image/jpeg":
		return detected, ".jpg", true
	default:
		return "", "", false
	}
}

// IsImage reports whether the mime type is a supported image.
func IsImage(mime string) bool {
	return mime == "image/png" || mime == "image/jpeg"
}
