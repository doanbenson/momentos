package ffmpeg

import (
	"strings"
	"testing"
)

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func TestAppendInputArgs_LoopOnlyOnPhotos(t *testing.T) {
	plan := Plan{
		EditAudioPath: "/tmp/edit.m4a",
		Clips: []ClipInput{
			{AssetID: "p1", Order: 0, StartMs: 0, EndMs: 2000, Type: AssetPhoto, LocalPath: "/tmp/p1.jpg"},
			{AssetID: "v1", Order: 1, StartMs: 0, EndMs: 2000, Type: AssetVideo, LocalPath: "/tmp/v1.mp4"},
		},
	}
	args := appendInputArgs(nil, plan)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-loop 1 -t 2.000 -i /tmp/p1.jpg") {
		t.Fatalf("photo missing loop args: %v", args)
	}
	// Video must be a bare -i, never -loop before its path.
	vidAt := -1
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-i" && args[i+1] == "/tmp/v1.mp4" {
			vidAt = i
			break
		}
	}
	if vidAt < 0 {
		t.Fatalf("video -i missing: %v", args)
	}
	if vidAt >= 2 && args[vidAt-2] == "-loop" {
		t.Fatalf("video must not use -loop: %v", args)
	}
}

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

func TestSnapMsToFrame_ChorusDropWithinOneFrame(t *testing.T) {
	// 1005ms at 30fps → nearest frame 1000ms (frame 30); delta ≤ one frame.
	got := SnapMsToFrame(1005)
	if got != 1000 {
		t.Fatalf("SnapMsToFrame(1005)=%d want 1000", got)
	}
	if abs(got-1005) > oneFrameMs() {
		t.Fatalf("snap delta %d exceeds ±1 frame", abs(got-1005))
	}
}

func TestBuildFilterComplex_Act2FlashAndSFXMarkers(t *testing.T) {
	plan := Plan{
		EditAudioPath:  "/tmp/edit.m4a",
		StyleProfileID: StyleProfileLifeOnFilmV1,
		Clips: []ClipInput{
			{AssetID: "r1", Order: 0, StartMs: 0, EndMs: 1000, Type: AssetVideo, LocalPath: "/tmp/r.mp4", Role: RoleReflection},
			{AssetID: "m1", Order: 1, StartMs: 0, EndMs: 1000, Type: AssetPhoto, LocalPath: "/tmp/m.jpg", Role: RoleMontage},
			{AssetID: "h1", Order: 2, StartMs: 0, EndMs: 1905, Type: AssetPhoto, LocalPath: "/tmp/h.jpg", Role: RoleHeroOutro},
		},
		Markers: []Marker{
			{Type: MarkerChorusDrop, AtMs: 1005},
			{Type: MarkerFlash, AtMs: 1005, SystemMediaID: "flash", LocalPath: "/tmp/flash.png"},
			{Type: MarkerSFX, AtMs: 1005, SystemMediaID: "sfx-whoosh", LocalPath: "/tmp/whoosh.wav"},
		},
	}
	filter, _, _, err := buildFilterComplex(plan)
	if err != nil {
		t.Fatal(err)
	}
	snapped := msToSec(SnapMsToFrame(1005))
	for _, need := range []string{
		"fade=t=out", // pack flash.png path (not synthetic white)
		"overlay=",
		"between(t\\," + snapped + "\\,",
		"adelay=1000|1000", // SFX snapped to frame
		"sfx0",
	} {
		if !strings.Contains(filter, need) {
			t.Fatalf("Act 2 marker graph missing %q\n%s", need, filter)
		}
	}
	if strings.Contains(filter, "color=c=white") {
		t.Fatal("flash with LocalPath must not use synthetic white color source")
	}
}

func TestBuildFilterComplex_Act3HeroOutroStyleProfile(t *testing.T) {
	plan := Plan{
		EditAudioPath:  "/tmp/edit.m4a",
		StyleProfileID: StyleProfileLifeOnFilmV1,
		Clips: []ClipInput{
			{AssetID: "r1", Order: 0, StartMs: 0, EndMs: 1000, Type: AssetVideo, LocalPath: "/tmp/r.mp4", Role: RoleReflection},
			{AssetID: "h1", Order: 1, StartMs: 0, EndMs: 2000, Type: AssetPhoto, LocalPath: "/tmp/h.jpg", Role: RoleHeroOutro},
		},
	}
	filter, _, _, err := buildFilterComplex(plan)
	if err != nil {
		t.Fatal(err)
	}
	for _, need := range []string{
		"eq=saturation=",                       // warm grade
		"drawbox=",                             // Super 8 border
		"fps=18",                               // posterize
		"fps=30",                               // return to canvas rate without setpts stretch
		"force_original_aspect_ratio=decrease", // no stretch on hero_outro photo
	} {
		if !strings.Contains(filter, need) {
			t.Fatalf("Act 3 Style Profile graph missing %q\n%s", need, filter)
		}
	}
	if strings.Contains(filter, "setpts=PTS*") {
		t.Fatal("Act 3 must not stretch timeline via setpts multiplier")
	}
}

func TestBuildFilterComplex_Act1BurnsCaptionsFromCut(t *testing.T) {
	plan := Plan{
		EditAudioPath:  "/tmp/edit.m4a",
		StyleProfileID: StyleProfileLifeOnFilmV1,
		Clips: []ClipInput{
			{AssetID: "r1", Order: 0, StartMs: 0, EndMs: 2000, Type: AssetVideo, LocalPath: "/tmp/r.mp4", Role: RoleReflection},
			{AssetID: "h1", Order: 1, StartMs: 0, EndMs: 2000, Type: AssetPhoto, LocalPath: "/tmp/h.jpg", Role: RoleHeroOutro},
		},
		Captions: []Caption{
			{Text: "hello week", StartMs: 200, EndMs: 800},
		},
	}
	filter, _, _, err := buildFilterComplex(plan)
	if err != nil {
		t.Fatal(err)
	}
	for _, need := range []string{
		"drawtext=",
		"hello week",
		"between(t\\,0.200\\,0.800)",
		"sidechaincompress=", // Reflection video VO ducked under bed
		"adelay=0|0",         // Reflection starts at timeline 0
	} {
		if !strings.Contains(filter, need) {
			t.Fatalf("Act 1 caption graph missing %q\n%s", need, filter)
		}
	}
}

func TestPlanValidate_LifeOnFilmAcceptsHeroOutro(t *testing.T) {
	plan := Plan{
		EditAudioPath:  "/tmp/edit.m4a",
		StyleProfileID: StyleProfileLifeOnFilmV1,
		Clips: []ClipInput{
			{AssetID: "r1", Order: 0, StartMs: 0, EndMs: 2000, Type: AssetVideo, LocalPath: "/tmp/r.mp4", Role: RoleReflection},
			{AssetID: "h1", Order: 1, StartMs: 0, EndMs: 2000, Type: AssetPhoto, LocalPath: "/tmp/h.jpg", Role: RoleHeroOutro},
		},
		Captions: []Caption{{Text: "ok", StartMs: 0, EndMs: 500}},
		Markers:  []Marker{{Type: MarkerChorusDrop, AtMs: 1000}},
	}
	if err := plan.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestPlanFromWire_ConsumesRolesCaptionsMarkers(t *testing.T) {
	wire := CutWire{
		ID:             "cut-1",
		MomentID:       "m-1",
		Version:        1,
		EditAudio:      "/tmp/edit.m4a",
		StyleProfileID: StyleProfileLifeOnFilmV1,
		Clips: []CutWireClip{
			{AssetID: "a1", Order: 0, StartMs: 0, EndMs: 1000, Role: RoleReflection},
			{AssetID: "a2", Order: 1, StartMs: 0, EndMs: 2000, Role: RoleHeroOutro},
		},
		Captions: []Caption{{Text: "hi", StartMs: 0, EndMs: 400}},
		Markers:  []Marker{{Type: MarkerFlash, AtMs: 500, SystemMediaID: "flash-1"}},
		Assets: map[string]AssetLocal{
			"a1": {Type: AssetVideo, LocalPath: "/tmp/a1.mp4"},
			"a2": {Type: AssetPhoto, LocalPath: "/tmp/a2.jpg"},
		},
	}
	plan, err := planFromWire(wire)
	if err != nil {
		t.Fatal(err)
	}
	if plan.StyleProfileID != StyleProfileLifeOnFilmV1 {
		t.Fatalf("style %q", plan.StyleProfileID)
	}
	if plan.Clips[0].Role != RoleReflection || plan.Clips[1].Role != RoleHeroOutro {
		t.Fatalf("roles: %+v", plan.Clips)
	}
	if len(plan.Captions) != 1 || plan.Captions[0].Text != "hi" {
		t.Fatalf("captions: %+v", plan.Captions)
	}
	if len(plan.Markers) != 1 || plan.Markers[0].Type != MarkerFlash {
		t.Fatalf("markers: %+v", plan.Markers)
	}
}

func TestPlanValidate_LifeOnFilmRequiresHeroOutro(t *testing.T) {
	plan := Plan{
		EditAudioPath:  "/tmp/edit.m4a",
		StyleProfileID: StyleProfileLifeOnFilmV1,
		Clips: []ClipInput{
			{AssetID: "r1", Order: 0, StartMs: 0, EndMs: 2000, Type: AssetVideo, LocalPath: "/tmp/r.mp4", Role: RoleReflection},
			{AssetID: "m1", Order: 1, StartMs: 0, EndMs: 1000, Type: AssetPhoto, LocalPath: "/tmp/m.jpg", Role: RoleMontage},
		},
	}
	err := plan.Validate()
	if err == nil {
		t.Fatal("expected missing hero_outro error")
	}
	if !strings.Contains(err.Error(), "hero_outro") {
		t.Fatalf("error should mention hero_outro: %v", err)
	}
}
