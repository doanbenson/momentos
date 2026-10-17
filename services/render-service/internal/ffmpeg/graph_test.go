package ffmpeg

import (
	"strings"
	"testing"
)

func TestBuildFilterComplex_PhotoAndVideoAndVO(t *testing.T) {
	plan := Plan{
		EditAudioPath: "/tmp/edit.m4a",
		Clips: []ClipInput{
			{AssetID: "p1", Order: 0, StartMs: 0, EndMs: 2000, Type: AssetPhoto, LocalPath: "/tmp/p1.jpg"},
			{AssetID: "v1", Order: 1, StartMs: 500, EndMs: 2500, Type: AssetVideo, LocalPath: "/tmp/v1.mp4"},
			{AssetID: "vo", Order: 2, StartMs: 1000, EndMs: 3000, Type: AssetVoice, LocalPath: "/tmp/vo.m4a"},
		},
	}
	filter, vLabel, aLabel, err := buildFilterComplex(plan)
	if err != nil {
		t.Fatal(err)
	}
	if vLabel != "vout" || aLabel != "aout" {
		t.Fatalf("labels: %s %s", vLabel, aLabel)
	}
	for _, need := range []string{
		"zoompan=",
		"trim=start=",
		"concat=n=2",
		"sidechaincompress=",
		"1080:1920",
		"fps=30",
	} {
		if !strings.Contains(filter, need) {
			t.Fatalf("filter missing %q\n%s", need, filter)
		}
	}
}

func TestPlanValidate_RejectsBadTiming(t *testing.T) {
	plan := Plan{
		EditAudioPath: "/tmp/edit.m4a",
		Clips: []ClipInput{
			{AssetID: "p1", Order: 0, StartMs: 10, EndMs: 10, Type: AssetPhoto, LocalPath: "/tmp/p.jpg"},
		},
	}
	if err := plan.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestPlanValidate_RejectsDuplicateOrder(t *testing.T) {
	plan := Plan{
		EditAudioPath: "/tmp/edit.m4a",
		Clips: []ClipInput{
			{AssetID: "a", Order: 1, StartMs: 0, EndMs: 1000, Type: AssetPhoto, LocalPath: "/tmp/a.jpg"},
			{AssetID: "b", Order: 1, StartMs: 0, EndMs: 1000, Type: AssetPhoto, LocalPath: "/tmp/b.jpg"},
		},
	}
	if err := plan.Validate(); err == nil {
		t.Fatal("expected duplicate order error")
	}
}
