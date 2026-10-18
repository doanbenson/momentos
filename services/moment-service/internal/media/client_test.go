package media

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestSignedAssetUnmarshalJSON_MediaSignDownloadShape(t *testing.T) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	raw := []byte(`{
		"assetId":"11111111-1111-1111-1111-111111111111",
		"type":"voice",
		"durationMs":3200,
		"downloadUrl":"https://example.test/signed/voice.m4a"
	}`)
	var a SignedAsset
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Fatal(err)
	}
	if a.ID != id {
		t.Fatalf("id=%v want %v", a.ID, id)
	}
	if a.StorageURL != "https://example.test/signed/voice.m4a" {
		t.Fatalf("storageUrl=%q", a.StorageURL)
	}
	if a.Type != "voice" {
		t.Fatalf("type=%q", a.Type)
	}
	if a.DurationMs == nil || *a.DurationMs != 3200 {
		t.Fatalf("durationMs=%v", a.DurationMs)
	}
}

func TestSignedAssetUnmarshalJSON_LegacyShape(t *testing.T) {
	raw := []byte(`{
		"id":"22222222-2222-2222-2222-222222222222",
		"type":"photo",
		"storageUrl":"https://example.test/legacy.jpg"
	}`)
	var a SignedAsset
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Fatal(err)
	}
	if a.StorageURL != "https://example.test/legacy.jpg" {
		t.Fatalf("storageUrl=%q", a.StorageURL)
	}
}
