package ffmpeg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/doanbenson/momentos/services/render-service/internal/songpick"
)

// buildFilterComplex produces one filter_complex graph for picture + audio.
//
// Picture: Motion Preset / Visual Style from Cut → 1080x1920@30 → concat hard-cuts.
// Act 1: Reflection question banner + Cut captions via drawtext; clean_916 slate for voice-only.
// Act 2: coalesced 3-frame flash + SFX; J-cut Edit Audio volume envelope.
// Act 3: Super 8 gate / Kodak / grain / 18fps on all act=3 clips.
// Audio: Edit Audio bed + VO adelay + sidechain ducking stub.
func buildFilterComplex(plan Plan) (filter string, videoLabel, audioLabel string, err error) {
	pics := plan.PictureClips()
	voices := plan.VoiceClips()
	if len(pics) == 0 {
		return "", "", "", fmt.Errorf("no picture clips")
	}

	flashes := coalesceFlashEvents(plan)
	sfxMarkers := sfxMarkersWithPath(plan.Markers)
	// Input layout: pictures (non-slate), voices, flash stills, SFX, optional overlay, Edit Audio last.
	overlayInputs := 0
	if plan.OverlayPath != "" {
		overlayInputs = 1
	}
	filePics := pictureFileInputs(pics)
	editIdx := len(filePics) + len(voices) + len(flashesWithPath(flashes)) + len(sfxMarkers) + overlayInputs
	var b strings.Builder

	// Build each picture clip at canvas size WITHOUT per-clip Act3 grade.
	// Act3 look is applied once after concatenating the Act3 run (memory: one noise/fps
	// chain instead of N) when Act1… then Act3… are contiguous — the Life on Film shape.
	built := make([]picLabel, 0, len(pics))
	fileIdx := 0
	for i, c := range pics {
		label := fmt.Sprintf("v%d", i)
		switch {
		case c.isClean916Slate():
			durSec := msToSec(c.DurationMs())
			fmt.Fprintf(&b,
				"color=c=%s:s=%dx%d:d=%s:r=%d,format=yuv420p,setsar=1[%s];",
				Clean916SlateColor, CanvasWidth, CanvasHeight, durSec, FrameRate, label,
			)
		case c.Type == AssetPhoto:
			inIdx := fileIdx
			fileIdx++
			frames := msToFrames(c.DurationMs())
			appendPhotoClip(&b, inIdx, label, c, frames)
		case c.Type == AssetVideo:
			inIdx := fileIdx
			fileIdx++
			appendVideoClip(&b, inIdx, label, c)
		default:
			return "", "", "", fmt.Errorf("picture clip type %q not supported", c.Type)
		}
		act3 := c.usesAct3Look() && plan.StyleProfileID == StyleProfileLifeOnFilmV1
		built = append(built, picLabel{label: label, act3: act3})
	}

	videoSrc, err := concatWithSharedAct3Style(&b, built)
	if err != nil {
		return "", "", "", err
	}
	// #region agent log
	act3N := 0
	for _, c := range built {
		if c.act3 {
			act3N++
		}
	}
	filterSoFar := b.String()
	agentDbg("H2", "graph.go:buildFilterComplex:layout", "picture layout + scale audit", map[string]any{
		"pictureCount":       len(built),
		"act3Count":          act3N,
		"sharedAct3Pass":     act3N > 0 && !act3Interleaved(built),
		"filterLen":          b.Len(),
		"scaleSegments":      scaleSegments(filterSoFar),
		"hasCropT":           strings.Contains(filterSoFar, "sin(2*PI*t"),
		"hasAct1Overscan":    strings.Contains(filterSoFar, fmt.Sprintf("scale=%d:%d", Act1OverscanWidth, Act1OverscanHeight)),
		"hasAct3Overscan":    strings.Contains(filterSoFar, fmt.Sprintf("scale=%d:%d", Act3DriftOverscanWidth, Act3DriftOverscanHeight)),
	})
	// #endregion

	videoSrc = appendBannerAndCaptions(&b, videoSrc, plan)
	flashInputBase := len(filePics) + len(voices)
	videoSrc = appendFlashOverlays(&b, videoSrc, flashes, flashInputBase)
	if plan.OverlayPath != "" {
		overlayIdx := len(filePics) + len(voices) + len(flashesWithPath(flashes)) + len(sfxMarkers)
		styled := videoSrc + "ov"
		// Single-frame overlay PNG: loop in filtergraph (no -loop 1 on input).
		fmt.Fprintf(&b,
			"[%d:v]scale=%d:%d,format=rgba,loop=loop=-1:size=1:start=0,setpts=N/%d/TB[%sov];"+
				"[%s][%sov]overlay=0:0:format=auto:shortest=1[%s];",
			overlayIdx, CanvasWidth, CanvasHeight, FrameRate, videoSrc, videoSrc, videoSrc, styled,
		)
		videoSrc = styled
	}
	fmt.Fprintf(&b, "[%s]copy[vout];", videoSrc)

	totalSec := msToSec(plan.PictureDurationMs())
	jCutStart := plan.JCutStartMs()
	act1End := plan.Act1PictureEndMs()
	// Full Edit Audio energy at chorus_drop (or Act1 end + drop_offset) — ADR 0019.
	energyMs := chorusDropMs(plan)
	if energyMs < act1End {
		energyMs = act1End
	}

	// Edit Audio bed with J-cut volume envelope (ADR 0019).
	fmt.Fprintf(&b,
		"[%d:a]aformat=sample_fmts=fltp:sample_rates=48000:channel_layouts=stereo,"+
			"atrim=0:%s,asetpts=PTS-STARTPTS,"+
			"volume='if(lt(t\\,%s)\\,0\\,if(lt(t\\,%s)\\,%g\\,1))'[bed];",
		editIdx, totalSec,
		msToSec(jCutStart), msToSec(energyMs), JCutEditAudioVolume,
	)

	voAndSFX := make([]string, 0, len(voices)+len(sfxMarkers)+len(pics))
	// Act 1: Reflection talking-head video contributes VO from its audio stream.
	timelineMs := 0
	refVO := 0
	for i, c := range pics {
		if c.Role == RoleReflection && c.Type == AssetVideo && !c.isClean916Slate() {
			label := fmt.Sprintf("refvo%d", refVO)
			refVO++
			durSec := msToSec(c.DurationMs())
			inIdx := pictureFileIndex(pics, i)
			fmt.Fprintf(&b,
				"[%d:a]aformat=sample_fmts=fltp:sample_rates=48000:channel_layouts=stereo,"+
					"atrim=0:%s,asetpts=PTS-STARTPTS,adelay=%d|%d[%s];",
				inIdx, durSec, timelineMs, timelineMs, label,
			)
			voAndSFX = append(voAndSFX, "["+label+"]")
		}
		timelineMs += c.DurationMs()
	}
	for i, c := range voices {
		inIdx := len(filePics) + i
		label := fmt.Sprintf("vo%d", i)
		durSec := msToSec(c.DurationMs())
		fmt.Fprintf(&b,
			"[%d:a]aformat=sample_fmts=fltp:sample_rates=48000:channel_layouts=stereo,"+
				"atrim=0:%s,asetpts=PTS-STARTPTS,adelay=%d|%d[%s];",
			inIdx, durSec, c.StartMs, c.StartMs, label,
		)
		voAndSFX = append(voAndSFX, "["+label+"]")
	}
	for i, m := range sfxMarkers {
		inIdx := len(filePics) + len(voices) + len(flashesWithPath(flashes)) + i
		label := fmt.Sprintf("sfx%d", i)
		at := SnapMsToFrame(m.AtMs)
		fmt.Fprintf(&b,
			"[%d:a]aformat=sample_fmts=fltp:sample_rates=48000:channel_layouts=stereo,"+
				"asetpts=PTS-STARTPTS,adelay=%d|%d[%s];",
			inIdx, at, at, label,
		)
		voAndSFX = append(voAndSFX, "["+label+"]")
	}

	if len(voAndSFX) == 0 {
		b.WriteString("[bed]anull[aout]")
		return strings.TrimSuffix(b.String(), ";"), "vout", "aout", nil
	}

	voMix := "vos"
	if len(voAndSFX) == 1 {
		// acopy — ffmpeg's "copy" filter is video-only and rejects audio pads.
		fmt.Fprintf(&b, "%sacopy[%s];", voAndSFX[0], voMix)
		// #region agent log
		agentDbg("F", "graph.go:buildFilterComplex:vo-passthrough", "single VO uses acopy", map[string]any{
			"voCount": 1,
			"label":   voAndSFX[0],
		})
		// #endregion
	} else {
		fmt.Fprintf(&b, "%samix=inputs=%d:duration=longest:dropout_transition=0[%s];",
			strings.Join(voAndSFX, ""), len(voAndSFX), voMix)
	}

	fmt.Fprintf(&b,
		"[bed][%s]sidechaincompress=threshold=0.05:ratio=8:attack=50:release=300:makeup=1[ducked];"+
			"[ducked][%s]amix=inputs=2:duration=first:dropout_transition=0[aout]",
		voMix, voMix,
	)

	return b.String(), "vout", "aout", nil
}

// appendPhotoClip builds one photo segment with Motion Preset.
// Decode path: scale/crop the single still FIRST, then loop that tiny frame in RAM
// (never -loop 1 on the JPEG input — that re-decodes 12MP+ every frame).
func appendPhotoClip(b *strings.Builder, inIdx int, label string, c ClipInput, frames int) {
	if c.usesAct3Look() {
		appendAct3PhotoMotion(b, inIdx, label, c, frames)
		return
	}
	motion := c.MotionPreset
	if motion == "" {
		motion = MotionHookSlowPush
	}
	// Static overscan scale (no t/n/pos) → crop to canvas → loop hold.
	base := fmt.Sprintf(
		"[%d:v]scale=%d:%d:force_original_aspect_ratio=increase,"+
			"crop=%d:%d,setsar=1,format=yuv420p,"+
			"loop=loop=%d:size=1:start=0,setpts=N/%d/TB,fps=%d",
		inIdx, Act1OverscanWidth, Act1OverscanHeight, CanvasWidth, CanvasHeight,
		frames, FrameRate, FrameRate,
	)
	switch motion {
	case MotionNativeActionCrop:
		fmt.Fprintf(b, "%s[%s];", base, label)
	case MotionWhipPunchIn:
		// Mild punch on already-scaled frames (cheap vs zoompan on 12MP).
		fmt.Fprintf(b,
			"%s,zoompan=z='min(1.0+0.012*on,1.28)':x='(iw-iw/zoom)/2':y='(ih-ih/zoom)/2':"+
				"d=%d:s=%dx%d:fps=%d[%s];",
			base, frames, CanvasWidth, CanvasHeight, FrameRate, label,
		)
	case MotionHandheldSuper8Drift:
		fmt.Fprintf(b,
			"%s,zoompan=z='min(1.0+0.0008*on,1.08)':"+
				"x='(iw-iw/zoom)/2+8*sin(on/18)':y='(ih-ih/zoom)/2+6*cos(on/22)':"+
				"d=%d:s=%dx%d:fps=%d[%s];",
			base, frames, CanvasWidth, CanvasHeight, FrameRate, label,
		)
	default: // MotionHookSlowPush
		fmt.Fprintf(b,
			"%s,zoompan=z='min(1.0+0.0015*on,1.12)':x='(iw-iw/zoom)/2':y='(ih-ih/zoom)/2':"+
				"d=%d:s=%dx%d:fps=%d[%s];",
			base, frames, CanvasWidth, CanvasHeight, FrameRate, label,
		)
	}
}

// appendAct3PhotoMotion: static overscan scale → loop → fixed-size crop weave.
// Output stays at gate size; appendAct3Style grades + pads onto the canvas.
// Never put t/n/pos in scale= (init eval_mode rejects them; eval=frame breaks concat size).
func appendAct3PhotoMotion(b *strings.Builder, inIdx int, label string, c ClipInput, frames int) {
	gateW, gateH := Super8GateWidth, Super8GateHeight
	if c.MotionPreset == MotionNativeActionCrop {
		fmt.Fprintf(b,
			"[%d:v]scale=%d:%d:force_original_aspect_ratio=increase,"+
				"crop=%d:%d,setsar=1,format=yuv420p,"+
				"loop=loop=%d:size=1:start=0,setpts=N/%d/TB,fps=%d[%s];",
			inIdx, gateW, gateH, gateW, gateH,
			frames, FrameRate, FrameRate, label,
		)
		return
	}
	// Loop before crop so t advances per held frame; crop w/h stay constant.
	xOff, yOff := "6*sin(2*PI*t*1.5)", "5*cos(2*PI*t*1.1)"
	if c.MotionPreset == MotionWhipPunchIn {
		// One-frame punch via larger first-frame offset; then settle into weave.
		xOff = "if(lt(t\\,0.04)\\,18\\,6*sin(2*PI*t*1.5))"
		yOff = "if(lt(t\\,0.04)\\,14\\,5*cos(2*PI*t*1.1))"
	}
	fmt.Fprintf(b,
		"[%d:v]scale=%d:%d:force_original_aspect_ratio=increase,"+
			"setsar=1,format=yuv420p,"+
			"loop=loop=%d:size=1:start=0,setpts=N/%d/TB,fps=%d,"+
			"crop=w=%d:h=%d:x='(iw-%d)/2+%s':y='(ih-%d)/2+%s'[%s];",
		inIdx, Act3DriftOverscanWidth, Act3DriftOverscanHeight,
		frames, FrameRate, FrameRate,
		gateW, gateH, gateW, xOff, gateH, yOff, label,
	)
}

type picLabel struct {
	label string
	act3  bool
}

// concatWithSharedAct3Style concatenates picture labels and applies Life on Film
// grade once across a contiguous Act3 run (Reflection Act1 + montage Act3).
// Falls back to per-clip Act3 if Act1/Act3 are interleaved (unexpected for MVP).
func concatWithSharedAct3Style(b *strings.Builder, built []picLabel) (string, error) {
	if len(built) == 0 {
		return "", fmt.Errorf("no picture labels")
	}

	// Detect Contiguous prefix of !act3 then suffix of act3.
	firstAct3 := -1
	interleaved := false
	for i, c := range built {
		if c.act3 {
			if firstAct3 < 0 {
				firstAct3 = i
			}
		} else if firstAct3 >= 0 {
			interleaved = true
			break
		}
	}

	if interleaved || firstAct3 < 0 {
		// Per-clip Act3 (or no Act3): style each act3 label then concat all.
		labels := make([]string, 0, len(built))
		for i, c := range built {
			lab := c.label
			if c.act3 {
				styled := fmt.Sprintf("%ss", lab)
				appendAct3Style(b, lab, styled)
				lab = styled
			}
			labels = append(labels, "["+lab+"]")
			_ = i
		}
		return concatVideoLabels(b, labels, "vbase"), nil
	}

	pre := built[:firstAct3]
	act3 := built[firstAct3:]

	var parts []string
	if len(pre) > 0 {
		preLabels := make([]string, len(pre))
		for i, c := range pre {
			preLabels[i] = "[" + c.label + "]"
		}
		parts = append(parts, "["+concatVideoLabels(b, preLabels, "vact1")+"]")
	}
	if len(act3) > 0 {
		aLabels := make([]string, len(act3))
		for i, c := range act3 {
			aLabels[i] = "[" + c.label + "]"
		}
		joined := concatVideoLabels(b, aLabels, "vact3raw")
		appendAct3Style(b, joined, "vact3")
		parts = append(parts, "[vact3]")
	}
	return concatVideoLabels(b, parts, "vbase"), nil
}

func concatVideoLabels(b *strings.Builder, labels []string, out string) string {
	if len(labels) == 0 {
		return out
	}
	if len(labels) == 1 {
		fmt.Fprintf(b, "%scopy[%s];", labels[0], out)
		return out
	}
	fmt.Fprintf(b, "%sconcat=n=%d:v=1:a=0[%s];", strings.Join(labels, ""), len(labels), out)
	return out
}

func act3Interleaved(built []picLabel) bool {
	firstAct3 := -1
	for i, c := range built {
		if c.act3 {
			if firstAct3 < 0 {
				firstAct3 = i
			}
		} else if firstAct3 >= 0 {
			return true
		}
	}
	return false
}

func appendVideoClip(b *strings.Builder, inIdx int, label string, c ClipInput) {
	startSec := msToSec(c.StartMs)
	endSec := msToSec(c.EndMs)
	aspect := "decrease"
	padColor := "black"
	if c.usesAct3Look() {
		padColor = Act3CanvasPadColor
	}
	motion := c.MotionPreset
	if motion == MotionNativeActionCrop || motion == "" {
		fmt.Fprintf(b,
			"[%d:v]trim=start=%s:end=%s,setpts=PTS-STARTPTS,fps=%d,"+
				"scale=%d:%d:force_original_aspect_ratio=%s,"+
				"pad=%d:%d:(ow-iw)/2:(oh-ih)/2:%s,setsar=1,format=yuv420p[%s];",
			inIdx, startSec, endSec, FrameRate,
			CanvasWidth, CanvasHeight, aspect,
			CanvasWidth, CanvasHeight, padColor, label,
		)
		return
	}
	// Photo-style motion on video: trim then mild zoompan over hold frames.
	frames := msToFrames(c.DurationMs())
	fmt.Fprintf(b,
		"[%d:v]trim=start=%s:end=%s,setpts=PTS-STARTPTS,fps=%d,"+
			"scale=%d:%d:force_original_aspect_ratio=%s,"+
			"pad=%d:%d:(ow-iw)/2:(oh-ih)/2:%s,setsar=1,format=yuv420p,",
		inIdx, startSec, endSec, FrameRate,
		CanvasWidth, CanvasHeight, aspect,
		CanvasWidth, CanvasHeight, padColor,
	)
	switch motion {
	case MotionWhipPunchIn:
		fmt.Fprintf(b,
			"zoompan=z='min(1.0+0.012*on,1.28)':x='(iw-iw/zoom)/2':y='(ih-ih/zoom)/2':"+
				"d=%d:s=%dx%d:fps=%d[%s];",
			frames, CanvasWidth, CanvasHeight, FrameRate, label,
		)
	case MotionHandheldSuper8Drift:
		fmt.Fprintf(b,
			"zoompan=z='min(1.0+0.0008*on,1.08)':"+
				"x='(iw-iw/zoom)/2+8*sin(on/18)':y='(ih-ih/zoom)/2+6*cos(on/22)':"+
				"d=%d:s=%dx%d:fps=%d[%s];",
			frames, CanvasWidth, CanvasHeight, FrameRate, label,
		)
	default:
		fmt.Fprintf(b,
			"zoompan=z='min(1.0+0.0015*on,1.12)':x='(iw-iw/zoom)/2':y='(ih-ih/zoom)/2':"+
				"d=%d:s=%dx%d:fps=%d[%s];",
			frames, CanvasWidth, CanvasHeight, FrameRate, label,
		)
	}
}

// appendAct3Style applies @lifeonfilm.27 vintage look on Act 3 clips:
// fit to film print, Kodak Gold grade, vignette, 18fps grain, warm-black matte.
func appendAct3Style(b *strings.Builder, inLabel, outLabel string) {
	fmt.Fprintf(b,
		"[%s]scale=%d:%d:force_original_aspect_ratio=decrease,"+
			"pad=%d:%d:(ow-iw)/2:(oh-ih)/2:%s,"+
			"curves=master='0/0.06 0.25/0.27 0.75/0.82 1/0.97',"+
			"colorbalance=rs=-0.03:gs=0.04:bs=-0.02:rm=0.04:gm=0.02:bm=-0.04:rh=0.10:gh=0.05:bh=-0.08,"+
			"eq=contrast=1.12:saturation=1.18:brightness=0.02,"+
			"vignette=PI/5,"+
			"unsharp=5:5:0.3:5:5:0.0,"+
			"fps=%d,fps=%d,"+
			"noise=alls=14:allf=t+u,"+
			"pad=%d:%d:(ow-iw)/2:(oh-ih)/2:%s[%s];",
		inLabel,
		Super8GateWidth, Super8GateHeight,
		Super8GateWidth, Super8GateHeight, Act3CanvasPadColor,
		Act3PosterizeFPS, FrameRate,
		CanvasWidth, CanvasHeight, Act3CanvasPadColor, outLabel,
	)
}

func appendBannerAndCaptions(b *strings.Builder, inLabel string, plan Plan) string {
	cur := inLabel
	act1End := plan.Act1PictureEndMs()
	fontOpt := drawtextFontfileOpt()
	if plan.ReflectionQuestion != "" && act1End > 0 {
		next := "vbanner"
		text := escapeDrawtext(plan.ReflectionQuestion)
		end := msToSec(act1End)
		fmt.Fprintf(b,
			"[%s]drawtext=text='%s'%s:fontsize=44:fontcolor=white:borderw=2:bordercolor=black@"+
				"0.7:x=(w-text_w)/2:y=h*0.08:enable='between(t\\,0\\,%s)'[%s];",
			cur, text, fontOpt, end, next,
		)
		cur = next
	}
	return appendCaptionBurns(b, cur, plan.Captions)
}

func appendCaptionBurns(b *strings.Builder, inLabel string, captions []Caption) string {
	cur := inLabel
	fontOpt := drawtextFontfileOpt()
	for i, c := range captions {
		next := fmt.Sprintf("vcapt%d", i)
		start := msToSec(c.StartMs)
		end := msToSec(c.EndMs)
		text := escapeDrawtext(c.Text)
		fmt.Fprintf(b,
			"[%s]drawtext=text='%s'%s:fontsize=52:fontcolor=white:borderw=3:bordercolor=black@"+
				"0.7:x=(w-text_w)/2:y=h*0.78:enable='between(t\\,%s\\,%s)'[%s];",
			cur, text, fontOpt, start, end, next,
		)
		cur = next
	}
	return cur
}

// drawtextFontfileOpt returns :fontfile='…' so drawtext does not rely on Fontconfig
// (Winget FFmpeg on Windows crashes when fonts.conf is missing).
func drawtextFontfileOpt() string {
	path := resolveDrawtextFont()
	if path == "" {
		return ""
	}
	opt := ":fontfile='" + escapeDrawtextFontPath(path) + "'"
	// #region agent log
	agentDbg("A", "graph.go:drawtextFontfileOpt", "resolved drawtext fontfile", map[string]any{
		"path": path,
		"opt":  opt,
	})
	// #endregion
	return opt
}

func resolveDrawtextFont() string {
	candidates := []string{
		os.Getenv("RENDER_DRAWTEXT_FONT"),
		filepath.Join(os.Getenv("WINDIR"), "Fonts", "arial.ttf"),
		`C:\Windows\Fonts\arial.ttf`,
		`C:\Windows\Fonts\segoeui.ttf`,
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/System/Library/Fonts/Supplemental/Arial.ttf",
		"/System/Library/Fonts/Helvetica.ttc",
	}
	for _, p := range candidates {
		if p == "" {
			continue
		}
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	return ""
}

// escapeDrawtextFontPath formats a filesystem path for an ffmpeg filtergraph fontfile=.
func escapeDrawtextFontPath(path string) string {
	p := filepath.ToSlash(path)
	// Filtergraph treats ':' as an option separator — escape drive + any other colons.
	p = strings.ReplaceAll(p, `:`, `\:`)
	p = strings.ReplaceAll(p, `'`, `\'`)
	return p
}

// flashEvent is one coalesced white-flash accent on the timeline.
type flashEvent struct {
	AtMs      int
	LocalPath string
}

func coalesceFlashEvents(plan Plan) []flashEvent {
	byAt := map[int]flashEvent{}
	order := make([]int, 0)

	add := func(atMs int, path string) {
		at := SnapMsToFrame(atMs)
		if existing, ok := byAt[at]; ok {
			if existing.LocalPath == "" && path != "" {
				existing.LocalPath = path
				byAt[at] = existing
			}
			return
		}
		byAt[at] = flashEvent{AtMs: at, LocalPath: path}
		order = append(order, at)
	}

	for _, m := range plan.Markers {
		if m.Type == MarkerFlash {
			add(m.AtMs, m.LocalPath)
		}
	}
	timeline := 0
	for _, c := range plan.PictureClips() {
		if c.TransitionIn == TransitionWhiteFlash {
			add(timeline, "")
		}
		timeline += c.DurationMs()
	}
	out := make([]flashEvent, 0, len(order))
	for _, at := range order {
		out = append(out, byAt[at])
	}
	return out
}

func flashesWithPath(events []flashEvent) []flashEvent {
	var out []flashEvent
	for _, e := range events {
		if e.LocalPath != "" {
			out = append(out, e)
		}
	}
	return out
}

func appendFlashOverlays(b *strings.Builder, inLabel string, events []flashEvent, flashInputBase int) string {
	cur := inLabel
	flashIdx := 0
	fileIdx := 0
	flashDur := threeFrameMs()
	for _, e := range events {
		at := e.AtMs
		start := msToSec(at)
		end := msToSec(at + flashDur)
		next := fmt.Sprintf("vflash%d", flashIdx)
		flashIdx++
		if e.LocalPath != "" {
			inIdx := flashInputBase + fileIdx
			fileIdx++
			src := fmt.Sprintf("flashsrc%d", flashIdx-1)
			flashFrames := msToFrames(flashDur)
			fmt.Fprintf(b,
				"[%d:v]scale=%d:%d,format=yuva420p,"+
					"loop=loop=%d:size=1:start=0,setpts=N/%d/TB,"+
					"fade=t=out:st=0:d=%s:alpha=1[%s];"+
					"[%s][%s]overlay=0:0:enable='between(t\\,%s\\,%s)'[%s];",
				inIdx, CanvasWidth, CanvasHeight,
				flashFrames, FrameRate, msToSec(flashDur), src,
				cur, src, start, end, next,
			)
		} else {
			fmt.Fprintf(b,
				"color=c=white:s=%dx%d:d=%s[flashsrc%d];"+
					"[%s][flashsrc%d]overlay=0:0:enable='between(t\\,%s\\,%s)'[%s];",
				CanvasWidth, CanvasHeight, msToSec(flashDur), flashIdx-1,
				cur, flashIdx-1, start, end, next,
			)
		}
		cur = next
	}
	return cur
}

func sfxMarkersWithPath(markers []Marker) []Marker {
	var out []Marker
	for _, m := range markers {
		if m.Type == MarkerSFX && m.LocalPath != "" {
			out = append(out, m)
		}
	}
	return out
}

func pictureFileInputs(pics []ClipInput) []ClipInput {
	var out []ClipInput
	for _, c := range pics {
		if c.isClean916Slate() {
			continue
		}
		out = append(out, c)
	}
	return out
}

func pictureFileIndex(pics []ClipInput, pictureOrd int) int {
	idx := 0
	for i, c := range pics {
		if i == pictureOrd {
			return idx
		}
		if !c.isClean916Slate() {
			idx++
		}
	}
	return idx
}

func chorusDropMs(plan Plan) int {
	for _, m := range plan.Markers {
		if m.Type == MarkerChorusDrop {
			return SnapMsToFrame(m.AtMs)
		}
	}
	act1 := plan.Act1PictureEndMs()
	if act1 <= 0 {
		return 0
	}
	return SnapMsToFrame(act1 + int(songpick.DropOffsetAfterSpeechSec*1000))
}

func escapeDrawtext(s string) string {
	r := strings.NewReplacer(
		`\`, `\\`,
		`'`, `\'`,
		`:`, `\:`,
		`%`, `\%`,
	)
	return r.Replace(s)
}

func oneFrameMs() int {
	ms := MsPerFrame + 0.5
	return int(ms)
}

func threeFrameMs() int {
	return oneFrameMs() * FlashFrames
}

func msToSec(ms int) string {
	sec := float64(ms) / 1000.0
	return fmt.Sprintf("%.3f", sec)
}

func msToFrames(ms int) int {
	frames := (ms * FrameRate) / 1000
	if frames < 1 {
		frames = 1
	}
	return frames
}

// appendInputArgs writes -i in graph order with per-input decoder thread caps.
// Photos are a SINGLE still frame (no -loop 1) — duration is made in filtergraph
// via loop=size=1 after scale, so mjpeg never re-decodes 12MP every frame.
func appendInputArgs(args []string, plan Plan) []string {
	for _, c := range plan.PictureClips() {
		if c.isClean916Slate() {
			continue
		}
		args = append(args, "-threads", "2", "-i", c.LocalPath)
	}
	for _, c := range plan.VoiceClips() {
		args = append(args, "-threads", "2", "-i", c.LocalPath)
	}
	for _, e := range flashesWithPath(coalesceFlashEvents(plan)) {
		// Tiny flash PNG: single frame; overlay filter holds duration.
		args = append(args, "-threads", "2", "-i", e.LocalPath)
	}
	for _, m := range sfxMarkersWithPath(plan.Markers) {
		args = append(args, "-threads", "2", "-i", m.LocalPath)
	}
	if plan.OverlayPath != "" {
		args = append(args, "-threads", "2", "-i", plan.OverlayPath)
	}
	args = append(args, "-threads", "2", "-i", plan.EditAudioPath)
	return args
}
