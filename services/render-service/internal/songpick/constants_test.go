package songpick

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMVPIdentity(t *testing.T) {
	if SongPickID != "live-a-life-you-will-remember" {
		t.Fatalf("SongPickID = %q", SongPickID)
	}
	if StyleProfileID != "life-on-film-v1" {
		t.Fatalf("StyleProfileID = %q", StyleProfileID)
	}
}

func TestLegalSpanBeats(t *testing.T) {
	want := []int{1, 2, 4}
	if len(LegalSpanBeats) != len(want) {
		t.Fatalf("LegalSpanBeats len = %d", len(LegalSpanBeats))
	}
	for i, v := range want {
		if LegalSpanBeats[i] != v {
			t.Fatalf("LegalSpanBeats[%d] = %d, want %d", i, LegalSpanBeats[i], v)
		}
	}
	if !IsLegalSpanBeats(4) || IsLegalSpanBeats(3) {
		t.Fatal("IsLegalSpanBeats mismatch")
	}
	if HeroOutroSpanBeats != 4 {
		t.Fatalf("HeroOutroSpanBeats = %d", HeroOutroSpanBeats)
	}
}

func TestConstantsMatchContractsParityFile(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "packages", "contracts", "mvp-constants.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read parity file: %v", err)
	}
	var expected struct {
		SongPickID                string  `json:"songPickId"`
		SongPickTitle             string  `json:"songPickTitle"`
		StyleProfileID            string  `json:"styleProfileId"`
		VibeID                    string  `json:"vibeId"`
		NominalBPM                int     `json:"nominalBpm"`
		LegalSpanBeats            []int   `json:"legalSpanBeats"`
		HeroOutroSpanBeats        int     `json:"heroOutroSpanBeats"`
		PhotoHoldBeats            int     `json:"photoHoldBeats"`
		ChorusDropBeatIndex       int     `json:"chorusDropBeatIndex"`
		JCutPreRollSec            float64 `json:"jCutPreRollSec"`
		DropOffsetAfterSpeechSec  float64 `json:"dropOffsetAfterSpeechSec"`
	}
	if err := json.Unmarshal(raw, &expected); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if SongPickID != expected.SongPickID {
		t.Fatalf("SongPickID parity: got %q want %q", SongPickID, expected.SongPickID)
	}
	if SongPickTitle != expected.SongPickTitle {
		t.Fatalf("SongPickTitle parity")
	}
	if StyleProfileID != expected.StyleProfileID {
		t.Fatalf("StyleProfileID parity")
	}
	if VibeID != expected.VibeID {
		t.Fatalf("VibeID parity")
	}
	if NominalBPM != expected.NominalBPM {
		t.Fatalf("NominalBPM parity")
	}
	if HeroOutroSpanBeats != expected.HeroOutroSpanBeats {
		t.Fatalf("HeroOutroSpanBeats parity")
	}
	if PhotoHoldBeats != expected.PhotoHoldBeats {
		t.Fatalf("PhotoHoldBeats parity")
	}
	if ChorusDropBeatIndex != expected.ChorusDropBeatIndex {
		t.Fatalf("ChorusDropBeatIndex parity")
	}
	if JCutPreRollSec != expected.JCutPreRollSec {
		t.Fatalf("JCutPreRollSec parity: got %v want %v", JCutPreRollSec, expected.JCutPreRollSec)
	}
	if DropOffsetAfterSpeechSec != expected.DropOffsetAfterSpeechSec {
		t.Fatalf("DropOffsetAfterSpeechSec parity: got %v want %v", DropOffsetAfterSpeechSec, expected.DropOffsetAfterSpeechSec)
	}
	if len(LegalSpanBeats) != len(expected.LegalSpanBeats) {
		t.Fatalf("LegalSpanBeats len parity")
	}
	for i := range expected.LegalSpanBeats {
		if LegalSpanBeats[i] != expected.LegalSpanBeats[i] {
			t.Fatalf("LegalSpanBeats[%d] parity", i)
		}
	}
}

func TestJCutAndDropTimingConstants(t *testing.T) {
	if JCutPreRollSec != 4.0 {
		t.Fatalf("JCutPreRollSec = %v, want 4.0", JCutPreRollSec)
	}
	if DropOffsetAfterSpeechSec != 0.25 {
		t.Fatalf("DropOffsetAfterSpeechSec = %v, want 0.25", DropOffsetAfterSpeechSec)
	}
}
