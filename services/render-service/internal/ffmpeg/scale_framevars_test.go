package ffmpeg

import (
	"fmt"
	"strings"
	"testing"
)

func TestBuildFilterComplex_ScaleFiltersRejectFrameVars(t *testing.T) {
	plan := Plan{
		EditAudioPath:  "/tmp/edit.m4a",
		StyleProfileID: StyleProfileLifeOnFilmV1,
		Clips: []ClipInput{
			{
				AssetID: "r1", Order: 0, StartMs: 0, EndMs: 1000, Type: AssetPhoto, LocalPath: "/tmp/r.jpg",
				Role: RoleReflection, Act: ActReflection, MotionPreset: MotionHookSlowPush,
				VisualStyle: VisualClean916, TransitionIn: TransitionNone,
			},
			{
				AssetID: "m1", Order: 1, StartMs: 0, EndMs: 1000, Type: AssetPhoto, LocalPath: "/tmp/m.jpg",
				Role: RoleMontage, Act: ActMontage, MotionPreset: MotionHandheldSuper8Drift,
				VisualStyle: VisualSuper8Gate, TransitionIn: TransitionNone,
			},
			{
				AssetID: "h1", Order: 2, StartMs: 0, EndMs: 1905, Type: AssetPhoto, LocalPath: "/tmp/h.jpg",
				Role: RoleHeroOutro, Act: ActHero, MotionPreset: MotionHandheldSuper8Drift,
				VisualStyle: VisualSuper8Gate, TransitionIn: TransitionNone,
			},
		},
	}
	filter, _, _, err := buildFilterComplex(plan)
	if err != nil {
		t.Fatal(err)
	}
	segs := scaleSegments(filter)
	agentDbg("H1", "scale_framevars_test.go", "scale segments post-fix", map[string]any{
		"count":           len(segs),
		"segments":        segs,
		"hasCropT":        strings.Contains(filter, "sin(2*PI*t"),
		"hasAct1Overscan": strings.Contains(filter, fmt.Sprintf("scale=%d:%d", Act1OverscanWidth, Act1OverscanHeight)),
		"hasAct3Overscan": strings.Contains(filter, fmt.Sprintf("scale=%d:%d", Act3DriftOverscanWidth, Act3DriftOverscanHeight)),
	})
	for _, s := range segs {
		if strings.Contains(s, "*t") || strings.Contains(s, "(t") ||
			strings.Contains(s, "*n") || strings.Contains(s, "(n") ||
			strings.Contains(s, "pos") {
			t.Fatalf("scale segment must not use frame vars: %s", s)
		}
	}
	if !strings.Contains(filter, fmt.Sprintf("scale=%d:%d", Act1OverscanWidth, Act1OverscanHeight)) {
		t.Fatalf("act1 must use static overscan scale %d:%d", Act1OverscanWidth, Act1OverscanHeight)
	}
	if !strings.Contains(filter, fmt.Sprintf("scale=%d:%d", Act3DriftOverscanWidth, Act3DriftOverscanHeight)) {
		t.Fatalf("act3 drift must use static overscan scale %d:%d", Act3DriftOverscanWidth, Act3DriftOverscanHeight)
	}
	if !strings.Contains(filter, "sin(2*PI*t") {
		t.Fatal("act3 drift must put t in crop, not scale")
	}
}
