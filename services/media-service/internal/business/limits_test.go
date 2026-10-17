package business

import (
	"testing"

	"github.com/doanbenson/momentos/services/media-service/internal/database"
)

func TestMimeAllowlistAndCaps(t *testing.T) {
	cases := []struct {
		typ   database.AssetType
		mime  string
		ok    bool
		max   int64
	}{
		{database.AssetTypePhoto, "image/jpeg", true, MaxPhotoBytes},
		{database.AssetTypePhoto, "image/gif", false, MaxPhotoBytes},
		{database.AssetTypeVideo, "video/mp4", true, MaxVideoBytes},
		{database.AssetTypeVideo, "video/webm", false, MaxVideoBytes},
		{database.AssetTypeVoice, "audio/m4a", true, MaxVoiceBytes},
		{database.AssetTypeVoice, "audio/wav", false, MaxVoiceBytes},
	}
	for _, tc := range cases {
		if got := mimeAllowed(tc.typ, tc.mime); got != tc.ok {
			t.Fatalf("%s %s: allowed=%v want %v", tc.typ, tc.mime, got, tc.ok)
		}
		if got := maxBytesFor(tc.typ); got != tc.max {
			t.Fatalf("%s max=%d want %d", tc.typ, got, tc.max)
		}
	}
}
