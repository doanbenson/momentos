package ffmpeg

import (
	"fmt"
	"strings"
	"testing"
)

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func TestAppendInputArgs_NoLoopOnPhotos(t *testing.T) {
	plan := Plan{
		EditAudioPath: "/tmp/edit.m4a",
		Clips: []ClipInput{
			{AssetID: "p1", Order: 0, StartMs: 0, EndMs: 2000, Type: AssetPhoto, LocalPath: "/tmp/p1.jpg"},
			{AssetID: "v1", Order: 1, StartMs: 0, EndMs: 2000, Type: AssetVideo, LocalPath: "/tmp/v1.mp4"},
		},
	}
	args := appendInputArgs(nil, plan)
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "-loop") {
		t.Fatalf("photos/videos must not use -loop 1 (decode once in filtergraph): %v", args)
	}
	if !strings.Contains(joined, "-threads 2 -i /tmp/p1.jpg") {
		t.Fatalf("photo missing capped -threads 2 -i: %v", args)
	}
	if !strings.Contains(joined, "-threads 2 -i /tmp/v1.mp4") {
		t.Fatalf("video missing capped -threads 2 -i: %v", args)
	}
}

func TestBuildFilterComplex_PhotoUsesFiltergraphLoopNotZoompanOnRawJPEG(t *testing.T) {
	plan := Plan{
		EditAudioPath:  "/tmp/edit.m4a",
		StyleProfileID: StyleProfileLifeOnFilmV1,
		Clips: []ClipInput{
			{
				AssetID: "p1", Order: 0, StartMs: 0, EndMs: 952, Type: AssetPhoto, LocalPath: "/tmp/p1.jpg",
				Role: RoleMontage, Act: ActHero, MotionPreset: MotionHandheldSuper8Drift,
				VisualStyle: VisualSuper8Gate, TransitionIn: TransitionNone,
			},
		},
	}
	filter, _, _, err := buildFilterComplex(plan)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(filter, "loop=loop=") {
		t.Fatalf("photo must loop scaled frame in filtergraph\n%s", filter)
	}
	// Act 3 hold: static scale and center crop to the film-print gate.
	gate := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d",
		Super8GateWidth, Super8GateHeight, Super8GateWidth, Super8GateHeight)
	if !strings.Contains(filter, gate) {
		t.Fatalf("act3 photo must static-crop to the gate\n%s", filter)
	}
	if strings.Contains(filter, "sin(2*PI*t") {
		t.Fatalf("act3 photo must not weave the crop\n%s", filter)
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
		// Timeline startMs=500 is not a file seek. Window is 0 .. duration (2000ms).
		"trim=start=0.000:end=2.000,setpts=PTS-STARTPTS",
		"concat=n=2",
		"normalize=0",
		"1080:1920",
		"fps=30",
		// Voice file window is 0 .. duration; StartMs only delays onto the output.
		"atrim=start=0.000:end=2.000,asetpts=PTS-STARTPTS,adelay=1000|1000",
	} {
		if !strings.Contains(filter, need) {
			t.Fatalf("filter missing %q\n%s", need, filter)
		}
	}
	if strings.Contains(filter, "trim=start=0.500") {
		t.Fatalf("video trim must not seek the source at timeline startMs\n%s", filter)
	}
}

func TestBuildFilterComplex_ReflectionSourceWindowMatchesVoice(t *testing.T) {
	// Timeline startMs is 0; the file window is 1.2s–3.2s. Mouth and voice share it.
	plan := Plan{
		EditAudioPath:  "/tmp/edit.m4a",
		StyleProfileID: StyleProfileLifeOnFilmV1,
		Clips: []ClipInput{
			{
				AssetID: "r1", Order: 0, StartMs: 0, EndMs: 2000, Type: AssetVideo, LocalPath: "/tmp/r.mp4",
				Role: RoleReflection, Act: ActReflection, MotionPreset: MotionNativeActionCrop,
				VisualStyle: VisualClean916, TransitionIn: TransitionNone,
				SourceInMs: 1200, SourceOutMs: 3200,
			},
			{
				AssetID: "h1", Order: 1, StartMs: 0, EndMs: 1905, Type: AssetPhoto, LocalPath: "/tmp/h.jpg",
				Role: RoleHeroOutro, Act: ActHero, MotionPreset: MotionHandheldSuper8Drift,
				VisualStyle: VisualSuper8Gate, TransitionIn: TransitionNone,
			},
			{
				AssetID: "vo", Order: 2, StartMs: 2500, EndMs: 4000, Type: AssetVoice, LocalPath: "/tmp/vo.m4a",
				SourceInMs: 400, SourceOutMs: 1900,
			},
		},
	}
	filter, _, _, err := buildFilterComplex(plan)
	if err != nil {
		t.Fatal(err)
	}
	for _, need := range []string{
		"trim=start=1.200:end=3.200,setpts=PTS-STARTPTS",
		"atrim=start=1.200:end=3.200,asetpts=PTS-STARTPTS,adelay=0|0[refvo0]",
		// Voice timeline start delays onto the output; file seek is sourceInMs.
		"atrim=start=0.400:end=1.900,asetpts=PTS-STARTPTS,adelay=2500|2500",
		"noise=",
		"fps=18",
	} {
		if !strings.Contains(filter, need) {
			t.Fatalf("source window graph missing %q\n%s", need, filter)
		}
	}
	refAudio := audioChain(filter, "[0:a]", "[refvo0]")
	if strings.Contains(refAudio, "atrim=0:") || strings.Contains(refAudio, "atrim=start=0") {
		t.Fatalf("reflection voice must not atrim from file t=0 or timeline start\n%s", refAudio)
	}
	if strings.Contains(filter, "trim=start=0.000") {
		t.Fatalf("reflection picture must not trim at timeline startMs\n%s", filter)
	}
}

func audioChain(filter, from, to string) string {
	i := strings.Index(filter, from)
	j := strings.Index(filter, to)
	if i < 0 || j < 0 || j < i {
		return filter
	}
	return filter[i : j+len(to)]
}

func TestBuildFilterComplex_SingleReflectionVOUsesAcopy(t *testing.T) {
	// Single adelay VO must use acopy — ffmpeg "copy" is video-only and rejects audio.
	plan := Plan{
		EditAudioPath:      "/tmp/edit.m4a",
		StyleProfileID:     StyleProfileLifeOnFilmV1,
		ReflectionQuestion: "What made this week yours?",
		Clips: []ClipInput{
			{
				AssetID: "r1", Order: 0, StartMs: 0, EndMs: 3000, Type: AssetVideo, LocalPath: "/tmp/r.mp4",
				Role: RoleReflection, Act: ActReflection, MotionPreset: MotionNativeActionCrop,
				VisualStyle: VisualClean916, TransitionIn: TransitionNone,
			},
			{
				AssetID: "h1", Order: 1, StartMs: 0, EndMs: 1905, Type: AssetPhoto, LocalPath: "/tmp/h.jpg",
				Role: RoleHeroOutro, Act: ActHero, MotionPreset: MotionHandheldSuper8Drift,
				VisualStyle: VisualSuper8Gate, TransitionIn: TransitionNone,
			},
		},
	}
	filter, _, _, err := buildFilterComplex(plan)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(filter, "adelay=") {
		t.Fatalf("expected reflection VO adelay\n%s", filter)
	}
	if !strings.Contains(filter, "acopy[vos]") {
		t.Fatalf("single VO must use acopy[vos], not video copy\n%s", filter)
	}
	// Guard against the regression: audio pad feeding video copy.
	if strings.Contains(filter, "]copy[vos]") {
		t.Fatalf("must not use video copy for VO passthrough\n%s", filter)
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
		"eq=contrast=1.12:saturation=1.18", // punchy Kodak grade
		"960:1280",                         // vintage film print gate
		"0x0C0B0A",                         // warm-black matte
		"curves=master=",
		"colorbalance=",
		"vignette=PI/5",
		"noise=alls=14",
		"fps=18",     // posterize cadence
		"fps=30",     // return to canvas rate without setpts stretch
		"loop=loop=", // hold scaled still in filtergraph
	} {
		if !strings.Contains(filter, need) {
			t.Fatalf("Act 3 Style Profile graph missing %q\n%s", need, filter)
		}
	}
	if strings.Contains(filter, "sin(2*PI*t") {
		t.Fatal("Act 3 must not weave the gate crop")
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
		"afade=t=in", // Edit Audio enters after the Reflection
		"adelay=0|0", // Reflection starts at timeline 0
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
			{AssetID: "a1", Order: 0, StartMs: 0, EndMs: 1000, Role: RoleReflection, SourceInMs: 1200, SourceOutMs: 3200},
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
	if plan.Clips[0].SourceInMs != 1200 || plan.Clips[0].SourceOutMs != 3200 {
		t.Fatalf("source window: %+v", plan.Clips[0])
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

func TestBuildFilterComplex_MotionPresetsDistinct(t *testing.T) {
	presets := []MotionPreset{
		MotionHookSlowPush,
		MotionWhipPunchIn,
		MotionHandheldSuper8Drift,
		MotionNativeActionCrop,
	}
	filters := make([]string, 0, len(presets))
	for _, preset := range presets {
		plan := Plan{
			EditAudioPath:  "/tmp/edit.m4a",
			StyleProfileID: StyleProfileLifeOnFilmV1,
			Clips: []ClipInput{
				{
					AssetID: "p", Order: 0, StartMs: 0, EndMs: 2000, Type: AssetPhoto, LocalPath: "/tmp/p.jpg",
					Role: RoleMontage, Act: ActMontage, MotionPreset: preset, VisualStyle: VisualClean916, TransitionIn: TransitionNone,
				},
				{
					AssetID: "h", Order: 1, StartMs: 0, EndMs: 1905, Type: AssetPhoto, LocalPath: "/tmp/h.jpg",
					Role: RoleHeroOutro, Act: ActHero, MotionPreset: MotionHandheldSuper8Drift, VisualStyle: VisualSuper8Gate, TransitionIn: TransitionNone,
				},
			},
		}
		filter, _, _, err := buildFilterComplex(plan)
		if err != nil {
			t.Fatalf("preset %s: %v", preset, err)
		}
		filters = append(filters, filter)
	}
	for i := 0; i < len(filters); i++ {
		for j := i + 1; j < len(filters); j++ {
			if filters[i] == filters[j] {
				t.Fatalf("motion presets %s and %s produced identical filter graphs", presets[i], presets[j])
			}
		}
	}
	if !strings.Contains(filters[0], "0.0015*on") {
		t.Fatal("hook_slow_push must use slow push zoompan")
	}
	if !strings.Contains(filters[1], "0.012*on") {
		t.Fatal("whip_punch_in must use punch-in zoompan")
	}
	if !strings.Contains(filters[2], "sin(on/") {
		t.Fatal("handheld_super8_drift must use drift zoompan")
	}
	// native_action_crop: scaled loop hold only — no Ken Burns zoompan.
	if !strings.Contains(filters[3], "loop=loop=") {
		t.Fatal("native_action_crop must loop the scaled still")
	}
	if strings.Contains(filters[3], "zoompan=") && strings.Count(filters[3], "zoompan=") > 1 {
		// hero may still use act3 weave; the montage native clip itself must not zoompan.
		montageSeg := filters[3]
		if idx := strings.Index(montageSeg, "[v0]"); idx > 0 {
			montageSeg = montageSeg[:idx+4]
		}
		if strings.Contains(montageSeg, "zoompan=") {
			t.Fatalf("native_action_crop montage must not use zoompan\n%s", montageSeg)
		}
	}
}

func TestBuildFilterComplex_Act3LookOnAllAct3Clips(t *testing.T) {
	plan := Plan{
		EditAudioPath:  "/tmp/edit.m4a",
		StyleProfileID: StyleProfileLifeOnFilmV1,
		Clips: []ClipInput{
			{
				AssetID: "r1", Order: 0, StartMs: 0, EndMs: 1000, Type: AssetVideo, LocalPath: "/tmp/r.mp4",
				Role: RoleReflection, Act: ActReflection, VisualStyle: VisualClean916, TransitionIn: TransitionNone,
			},
			{
				AssetID: "coda", Order: 1, StartMs: 0, EndMs: 1000, Type: AssetPhoto, LocalPath: "/tmp/coda.jpg",
				Role: RoleMontage, Act: ActHero, MotionPreset: MotionHandheldSuper8Drift, VisualStyle: VisualSuper8Gate, TransitionIn: TransitionNone,
			},
			{
				AssetID: "h1", Order: 2, StartMs: 0, EndMs: 1905, Type: AssetPhoto, LocalPath: "/tmp/h.jpg",
				Role: RoleHeroOutro, Act: ActHero, MotionPreset: MotionHandheldSuper8Drift, VisualStyle: VisualSuper8Gate, TransitionIn: TransitionNone,
			},
		},
	}
	filter, _, _, err := buildFilterComplex(plan)
	if err != nil {
		t.Fatal(err)
	}
	gate := fmt.Sprintf("%d:%d", Super8GateWidth, Super8GateHeight)
	needs := []string{
		gate,             // 960×1280 film print
		"0x0C0B0A",       // warm-black matte
		"curves=master=", // lifted matte blacks
		"colorbalance=",  // Kodak Gold / teal shadows
		"eq=contrast=1.12:saturation=1.18",
		"vignette=PI/5",
		"unsharp=",
		"noise=alls=14",
		"fps=18",
		"concat=n=2", // Act3 pair concat, then Act1+Act3 concat
	}
	for _, need := range needs {
		if !strings.Contains(filter, need) {
			t.Fatalf("Act3-all look missing %q\n%s", need, filter)
		}
	}
	// Contiguous Act3 run → shared grade once (memory), not per-clip.
	if strings.Count(filter, "fps=18") != 1 {
		t.Fatalf("expected one shared Act3 look pass, fps=18 count=%d\n%s", strings.Count(filter, "fps=18"), filter)
	}
	if !strings.Contains(filter, "vact3raw") && !strings.Contains(filter, "[vact3]") {
		t.Fatalf("expected shared Act3 concat path\n%s", filter)
	}
}

func TestBuildFilterComplex_MusicFadesInAfterAct1(t *testing.T) {
	plan := Plan{
		EditAudioPath:  "/tmp/edit.m4a",
		StyleProfileID: StyleProfileLifeOnFilmV1,
		Clips: []ClipInput{
			{
				AssetID: "r1", Order: 0, StartMs: 0, EndMs: 5000, Type: AssetVideo, LocalPath: "/tmp/r.mp4",
				Role: RoleReflection, Act: ActReflection, VisualStyle: VisualClean916, TransitionIn: TransitionNone,
			},
			{
				AssetID: "m1", Order: 1, StartMs: 0, EndMs: 1000, Type: AssetPhoto, LocalPath: "/tmp/m.jpg",
				Role: RoleMontage, Act: ActMontage, MotionPreset: MotionWhipPunchIn, VisualStyle: VisualClean916, TransitionIn: TransitionNone,
			},
			{
				AssetID: "h1", Order: 2, StartMs: 0, EndMs: 1905, Type: AssetPhoto, LocalPath: "/tmp/h.jpg",
				Role: RoleHeroOutro, Act: ActHero, MotionPreset: MotionHandheldSuper8Drift, VisualStyle: VisualSuper8Gate, TransitionIn: TransitionNone,
			},
		},
		Markers: []Marker{
			{Type: MarkerJCut, AtMs: 1000}, // 4s before Act1 end (5000)
			{Type: MarkerChorusDrop, AtMs: 5250},
		},
	}
	filter, _, _, err := buildFilterComplex(plan)
	if err != nil {
		t.Fatal(err)
	}
	act1 := msToSec(plan.Act1PictureEndMs())
	picture := msToSec(plan.PictureDurationMs())
	for _, need := range []string{
		"volume=",
		"afade=t=in:st=" + act1,
		fmt.Sprintf("d=%.3f", MusicFadeInSec),
		"apad=whole_dur=" + picture, // bed lasts the picture timeline
		"duration=longest",
		"normalize=0",
		// No flash marker: synthetic white flash on the Reflection cut.
		"color=c=white",
		"setpts=PTS+" + act1 + "/TB",
	} {
		if !strings.Contains(filter, need) {
			t.Fatalf("post-Reflection mix missing %q\n%s", need, filter)
		}
	}
	if strings.Contains(filter, "sidechaincompress=") {
		t.Fatal("bed must not be sidechain-compressed to the Reflection")
	}
	if strings.Contains(filter, "0.18") {
		t.Fatal("music must not use the J-cut pre-roll level")
	}
}

func TestBuildFilterComplex_FlashThreeFramesAndCoalesce(t *testing.T) {
	plan := Plan{
		EditAudioPath:  "/tmp/edit.m4a",
		StyleProfileID: StyleProfileLifeOnFilmV1,
		Clips: []ClipInput{
			{
				AssetID: "r1", Order: 0, StartMs: 0, EndMs: 1000, Type: AssetVideo, LocalPath: "/tmp/r.mp4",
				Role: RoleReflection, Act: ActReflection, VisualStyle: VisualClean916, TransitionIn: TransitionNone,
			},
			{
				AssetID: "m1", Order: 1, StartMs: 0, EndMs: 1000, Type: AssetPhoto, LocalPath: "/tmp/m.jpg",
				Role: RoleMontage, Act: ActMontage, MotionPreset: MotionWhipPunchIn, VisualStyle: VisualClean916, TransitionIn: TransitionWhiteFlash,
			},
			{
				AssetID: "h1", Order: 2, StartMs: 0, EndMs: 1905, Type: AssetPhoto, LocalPath: "/tmp/h.jpg",
				Role: RoleHeroOutro, Act: ActHero, MotionPreset: MotionHandheldSuper8Drift, VisualStyle: VisualSuper8Gate, TransitionIn: TransitionNone,
			},
		},
		Markers: []Marker{
			// Same timestamp as m1 start (1000ms) — must coalesce with transitionIn
			{Type: MarkerFlash, AtMs: 1000, SystemMediaID: "flash", LocalPath: "/tmp/flash.png"},
			{Type: MarkerSFX, AtMs: 1000, SystemMediaID: "sfx-shutter", LocalPath: "/tmp/whoosh.wav"},
		},
	}
	filter, _, _, err := buildFilterComplex(plan)
	if err != nil {
		t.Fatal(err)
	}
	flashDur := threeFrameMs()
	start := msToSec(SnapMsToFrame(1000))
	end := msToSec(SnapMsToFrame(1000) + flashDur)
	if !strings.Contains(filter, "between(t\\,"+start+"\\,"+end+")") {
		t.Fatalf("expected flash window %s→%s\n%s", start, end, filter)
	}
	if !strings.Contains(filter, "setpts=PTS+"+start+"/TB") {
		t.Fatalf("flash frames must be shifted onto the Reflection cut at %s\n%s", start, filter)
	}
	if !strings.Contains(filter, "eof_action=pass") {
		t.Fatalf("flash overlay must not hold a clear frame after the cut\n%s", filter)
	}
	if n := strings.Count(filter, "fade=t=out"); n != 1 {
		t.Fatalf("duplicate flashes at same timestamp must coalesce; fade overlays=%d\n%s", n, filter)
	}
}

func TestBuildFilterComplex_Act1BannerFromReflectionQuestion(t *testing.T) {
	plan := Plan{
		EditAudioPath:      "/tmp/edit.m4a",
		StyleProfileID:     StyleProfileLifeOnFilmV1,
		ReflectionQuestion: "What made you proud this week?",
		Clips: []ClipInput{
			{
				AssetID: "r1", Order: 0, StartMs: 0, EndMs: 2000, Type: AssetVideo, LocalPath: "/tmp/r.mp4",
				Role: RoleReflection, Act: ActReflection, VisualStyle: VisualClean916, TransitionIn: TransitionNone,
			},
			{
				AssetID: "h1", Order: 1, StartMs: 0, EndMs: 1905, Type: AssetPhoto, LocalPath: "/tmp/h.jpg",
				Role: RoleHeroOutro, Act: ActHero, MotionPreset: MotionHandheldSuper8Drift, VisualStyle: VisualSuper8Gate, TransitionIn: TransitionNone,
			},
		},
		Captions: []Caption{{Text: "I shipped it", StartMs: 200, EndMs: 800}},
	}
	filter, _, _, err := buildFilterComplex(plan)
	if err != nil {
		t.Fatal(err)
	}
	for _, need := range []string{
		"What made you proud this week?",
		"I shipped it",
		"drawtext=",
	} {
		if !strings.Contains(filter, need) {
			t.Fatalf("Act 1 banner/captions missing %q\n%s", need, filter)
		}
	}
	// On hosts with a resolvable system font, drawtext must bypass Fontconfig.
	if resolveDrawtextFont() != "" && !strings.Contains(filter, "fontfile=") {
		t.Fatalf("drawtext must set fontfile when a system font is available\n%s", filter)
	}
}

func TestBuildFilterComplex_VoiceOnlyClean916Slate(t *testing.T) {
	plan := Plan{
		EditAudioPath:      "/tmp/edit.m4a",
		StyleProfileID:     StyleProfileLifeOnFilmV1,
		ReflectionQuestion: "What stuck with you?",
		Clips: []ClipInput{
			{
				AssetID: "slate", Order: 0, StartMs: 0, EndMs: 3000, Type: AssetPhoto, LocalPath: "",
				Role: RoleReflection, Act: ActReflection, VisualStyle: VisualClean916, TransitionIn: TransitionNone,
			},
			{
				AssetID: "vo", Order: 1, StartMs: 0, EndMs: 3000, Type: AssetVoice, LocalPath: "/tmp/vo.m4a",
				Role: RoleReflection, Act: ActReflection, VisualStyle: VisualClean916, TransitionIn: TransitionNone,
			},
			{
				AssetID: "h1", Order: 2, StartMs: 0, EndMs: 1905, Type: AssetPhoto, LocalPath: "/tmp/h.jpg",
				Role: RoleHeroOutro, Act: ActHero, MotionPreset: MotionHandheldSuper8Drift, VisualStyle: VisualSuper8Gate, TransitionIn: TransitionNone,
			},
		},
		Captions: []Caption{{Text: "hello", StartMs: 0, EndMs: 500}},
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf("voice-only slate plan should validate: %v", err)
	}
	filter, _, _, err := buildFilterComplex(plan)
	if err != nil {
		t.Fatal(err)
	}
	for _, need := range []string{
		"color=c=",
		"What stuck with you?",
		"hello",
		"adelay=0|0",
	} {
		if !strings.Contains(filter, need) {
			t.Fatalf("voice-only slate missing %q\n%s", need, filter)
		}
	}
	// Slate stream [v0] is color→format only; hero may still use zoompan before concat.
	slateIdx := strings.Index(filter, "color=c=")
	if slateIdx < 0 {
		t.Fatal("expected clean_916 color slate")
	}
	slateEnd := strings.Index(filter[slateIdx:], "[v0]")
	if slateEnd < 0 {
		t.Fatal("expected slate label [v0]")
	}
	slateSeg := filter[slateIdx : slateIdx+slateEnd]
	if strings.Contains(slateSeg, "zoompan=") {
		t.Fatalf("clean_916 Reflection slate must not use motion zoompan\n%s", slateSeg)
	}
}

func TestBuildFilterComplex_ReflectionPlusSixMontageFixture(t *testing.T) {
	clips := []ClipInput{
		{
			AssetID: "r1", Order: 0, StartMs: 0, EndMs: 2000, Type: AssetVideo, LocalPath: "/tmp/r.mp4",
			Role: RoleReflection, Act: ActReflection, VisualStyle: VisualClean916, TransitionIn: TransitionNone,
		},
	}
	for i := 0; i < 5; i++ {
		clips = append(clips, ClipInput{
			AssetID: fmt.Sprintf("m%d", i), Order: i + 1, StartMs: 0, EndMs: 952, Type: AssetPhoto,
			LocalPath: fmt.Sprintf("/tmp/m%d.jpg", i),
			// Act 3 montage (Life on Film look) — matches deterministic director defaults.
			Role: RoleMontage, Act: ActHero, MotionPreset: MotionHandheldSuper8Drift, VisualStyle: VisualSuper8Gate,
			TransitionIn: TransitionNone,
		})
	}
	clips = append(clips, ClipInput{
		AssetID: "h1", Order: 6, StartMs: 0, EndMs: 1905, Type: AssetPhoto, LocalPath: "/tmp/h.jpg",
		Role: RoleHeroOutro, Act: ActHero, MotionPreset: MotionHandheldSuper8Drift, VisualStyle: VisualSuper8Gate, TransitionIn: TransitionNone,
	})
	plan := Plan{
		EditAudioPath:      "/tmp/edit.m4a",
		StyleProfileID:     StyleProfileLifeOnFilmV1,
		ReflectionQuestion: "fixture banner",
		Clips:              clips,
		Captions:           []Caption{{Text: "cue", StartMs: 0, EndMs: 400}},
		Markers: []Marker{
			{Type: MarkerJCut, AtMs: 0},
			{Type: MarkerChorusDrop, AtMs: 2250},
			{Type: MarkerFlash, AtMs: 2000, LocalPath: "/tmp/flash.png"},
		},
	}
	if err := plan.Validate(); err != nil {
		t.Fatal(err)
	}
	filter, _, _, err := buildFilterComplex(plan)
	if err != nil {
		t.Fatal(err)
	}
	// Shared Act3: concat 6 montage/hero → one grade → join with Reflection.
	if !strings.Contains(filter, "concat=n=6") {
		t.Fatalf("1 Reflection + 6 Act3 pictures should concat Act3 as n=6\n%s", filter)
	}
	if strings.Count(filter, "fps=18") != 1 {
		t.Fatalf("shared Act3 should apply fps=18 once, got %d\n%s", strings.Count(filter, "fps=18"), filter)
	}
	for _, need := range []string{"volume=", "afade=t=in", "fixture banner", "fps=18", "noise=", "acopy[vos]"} {
		if !strings.Contains(filter, need) {
			t.Fatalf("1+6 fixture missing %q\n%s", need, filter)
		}
	}
}
