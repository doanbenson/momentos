package ffmpeg

import (
	"fmt"
	"strings"
)

// buildFilterComplex produces one filter_complex graph for picture + audio.
//
// Picture: scale/pad (or Ken-Burns for photos) → 1080x1920@30 → concat hard-cuts.
// Act 1: burn Cut captions via drawtext.
// Act 2: marker flash/SFX overlays (system media paths when resolved).
// Act 3: Style Profile look on hero_outro (warm grade, Super 8 border, 18fps posterize, no stretch).
// Audio: Edit Audio bed trimmed to picture duration + VO adelay + sidechain ducking stub.
func buildFilterComplex(plan Plan) (filter string, videoLabel, audioLabel string, err error) {
	pics := plan.PictureClips()
	voices := plan.VoiceClips()
	if len(pics) == 0 {
		return "", "", "", fmt.Errorf("no picture clips")
	}

	flashMarkers := flashMarkersWithPath(plan.Markers)
	sfxMarkers := sfxMarkersWithPath(plan.Markers)
	// Input layout: pictures, voices, flash stills, SFX, optional overlay, Edit Audio last.
	overlayInputs := 0
	if plan.OverlayPath != "" {
		overlayInputs = 1
	}
	editIdx := len(pics) + len(voices) + len(flashMarkers) + len(sfxMarkers) + overlayInputs
	var b strings.Builder
	vLabels := make([]string, 0, len(pics))

	for i, c := range pics {
		label := fmt.Sprintf("v%d", i)
		switch c.Type {
		case AssetPhoto:
			frames := msToFrames(c.DurationMs())
			aspect := "increase"
			cropOrPad := fmt.Sprintf("crop=%d:%d", CanvasWidth, CanvasHeight)
			if c.Role == RoleHeroOutro {
				// Act 3: no stretch — letterbox instead of crop-upscale.
				aspect = "decrease"
				cropOrPad = fmt.Sprintf("pad=%d:%d:(ow-iw)/2:(oh-ih)/2:black", CanvasWidth, CanvasHeight)
			}
			fmt.Fprintf(&b,
				"[%d:v]scale=%d:%d:force_original_aspect_ratio=%s,"+
					"%s,setsar=1,format=yuv420p,"+
					"zoompan=z='min(1.0+0.0015*on,1.12)':x='(iw-iw/zoom)/2':y='(ih-ih/zoom)/2':"+
					"d=%d:s=%dx%d:fps=%d[%s];",
				i, CanvasWidth, CanvasHeight, aspect, cropOrPad,
				frames, CanvasWidth, CanvasHeight, FrameRate, label,
			)
		case AssetVideo:
			startSec := msToSec(c.StartMs)
			endSec := msToSec(c.EndMs)
			fmt.Fprintf(&b,
				"[%d:v]trim=start=%s:end=%s,setpts=PTS-STARTPTS,fps=%d,"+
					"scale=%d:%d:force_original_aspect_ratio=decrease,"+
					"pad=%d:%d:(ow-iw)/2:(oh-ih)/2:black,setsar=1,format=yuv420p[%s];",
				i, startSec, endSec, FrameRate,
				CanvasWidth, CanvasHeight, CanvasWidth, CanvasHeight, label,
			)
		default:
			return "", "", "", fmt.Errorf("picture clip type %q not supported", c.Type)
		}
		if c.Role == RoleHeroOutro && plan.StyleProfileID == StyleProfileLifeOnFilmV1 {
			styled := label + "s"
			appendAct3Style(&b, label, styled)
			label = styled
		}
		vLabels = append(vLabels, "["+label+"]")
	}

	videoSrc := "vbase"
	if len(vLabels) == 1 {
		fmt.Fprintf(&b, "%scopy[%s];", vLabels[0], videoSrc)
	} else {
		fmt.Fprintf(&b, "%sconcat=n=%d:v=1:a=0[%s];", strings.Join(vLabels, ""), len(vLabels), videoSrc)
	}

	videoSrc = appendCaptionBurns(&b, videoSrc, plan.Captions)
	flashInputBase := len(pics) + len(voices)
	videoSrc = appendFlashOverlays(&b, videoSrc, plan.Markers, flashInputBase)
	if plan.OverlayPath != "" {
		overlayIdx := len(pics) + len(voices) + len(flashMarkers) + len(sfxMarkers)
		styled := videoSrc + "ov"
		fmt.Fprintf(&b,
			"[%d:v]scale=%d:%d,format=rgba[%sov];"+
				"[%s][%sov]overlay=0:0:format=auto[%s];",
			overlayIdx, CanvasWidth, CanvasHeight, videoSrc, videoSrc, videoSrc, styled,
		)
		videoSrc = styled
	}
	fmt.Fprintf(&b, "[%s]copy[vout];", videoSrc)

	totalSec := msToSec(plan.PictureDurationMs())

	// Edit Audio bed
	fmt.Fprintf(&b,
		"[%d:a]aformat=sample_fmts=fltp:sample_rates=48000:channel_layouts=stereo,"+
			"atrim=0:%s,asetpts=PTS-STARTPTS[bed];",
		editIdx, totalSec,
	)

	voAndSFX := make([]string, 0, len(voices)+len(sfxMarkers)+len(pics))
	// Act 1: Reflection talking-head video contributes VO from its audio stream.
	timelineMs := 0
	refVO := 0
	for i, c := range pics {
		if c.Role == RoleReflection && c.Type == AssetVideo {
			label := fmt.Sprintf("refvo%d", refVO)
			refVO++
			durSec := msToSec(c.DurationMs())
			fmt.Fprintf(&b,
				"[%d:a]aformat=sample_fmts=fltp:sample_rates=48000:channel_layouts=stereo,"+
					"atrim=0:%s,asetpts=PTS-STARTPTS,adelay=%d|%d[%s];",
				i, durSec, timelineMs, timelineMs, label,
			)
			voAndSFX = append(voAndSFX, "["+label+"]")
		}
		timelineMs += c.DurationMs()
	}
	for i, c := range voices {
		inIdx := len(pics) + i
		label := fmt.Sprintf("vo%d", i)
		durSec := msToSec(c.DurationMs())
		// Voice startMs/endMs = placement on output timeline
		fmt.Fprintf(&b,
			"[%d:a]aformat=sample_fmts=fltp:sample_rates=48000:channel_layouts=stereo,"+
				"atrim=0:%s,asetpts=PTS-STARTPTS,adelay=%d|%d[%s];",
			inIdx, durSec, c.StartMs, c.StartMs, label,
		)
		voAndSFX = append(voAndSFX, "["+label+"]")
	}
	for i, m := range sfxMarkers {
		inIdx := len(pics) + len(voices) + len(flashMarkers) + i
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
		fmt.Fprintf(&b, "%scopy[%s];", voAndSFX[0], voMix)
	} else {
		fmt.Fprintf(&b, "%samix=inputs=%d:duration=longest:dropout_transition=0[%s];",
			strings.Join(voAndSFX, ""), len(voAndSFX), voMix)
	}

	// Ducking stub: sidechaincompress bed under VO/SFX, then remixed
	fmt.Fprintf(&b,
		"[bed][%s]sidechaincompress=threshold=0.05:ratio=8:attack=50:release=300:makeup=1[ducked];"+
			"[ducked][%s]amix=inputs=2:duration=first:dropout_transition=0[aout]",
		voMix, voMix,
	)

	return b.String(), "vout", "aout", nil
}

// appendAct3Style applies life-on-film-v1 look: warm grade, Super 8 border, 18fps posterize (no stretch).
func appendAct3Style(b *strings.Builder, inLabel, outLabel string) {
	fmt.Fprintf(b,
		"[%s]eq=saturation=1.12:gamma_r=1.06:gamma_g=1.03:gamma_b=0.94,"+
			"drawbox=x=0:y=0:w=iw:h=ih:color=black@0.85:t=28,"+
			"fps=%d,fps=%d[%s];",
		inLabel, Act3PosterizeFPS, FrameRate, outLabel,
	)
}

func appendCaptionBurns(b *strings.Builder, inLabel string, captions []Caption) string {
	cur := inLabel
	for i, c := range captions {
		next := fmt.Sprintf("vcapt%d", i)
		start := msToSec(c.StartMs)
		end := msToSec(c.EndMs)
		text := escapeDrawtext(c.Text)
		fmt.Fprintf(b,
			"[%s]drawtext=text='%s':fontsize=52:fontcolor=white:borderw=3:bordercolor=black@"+
				"0.7:x=(w-text_w)/2:y=h*0.78:enable='between(t\\,%s\\,%s)'[%s];",
			cur, text, start, end, next,
		)
		cur = next
	}
	return cur
}

func appendFlashOverlays(b *strings.Builder, inLabel string, markers []Marker, flashInputBase int) string {
	cur := inLabel
	flashIdx := 0
	fileIdx := 0
	for _, m := range markers {
		// chorus_drop is a sync target (±1 frame via SnapMsToFrame); flash executes the accent.
		if m.Type != MarkerFlash {
			continue
		}
		at := SnapMsToFrame(m.AtMs)
		frameMs := oneFrameMs()
		start := msToSec(at)
		end := msToSec(at + frameMs)
		next := fmt.Sprintf("vflash%d", flashIdx)
		flashIdx++
		if m.LocalPath != "" {
			inIdx := flashInputBase + fileIdx
			fileIdx++
			src := fmt.Sprintf("flashsrc%d", flashIdx-1)
			fmt.Fprintf(b,
				"[%d:v]scale=%d:%d,format=yuva420p,fade=t=out:st=0:d=%s:alpha=1[%s];"+
					"[%s][%s]overlay=0:0:enable='between(t\\,%s\\,%s)'[%s];",
				inIdx, CanvasWidth, CanvasHeight, msToSec(frameMs), src,
				cur, src, start, end, next,
			)
		} else {
			// Fallback when pack file missing.
			fmt.Fprintf(b,
				"color=c=white:s=%dx%d:d=%s[flashsrc%d];"+
					"[%s][flashsrc%d]overlay=0:0:enable='between(t\\,%s\\,%s)'[%s];",
				CanvasWidth, CanvasHeight, msToSec(frameMs), flashIdx-1,
				cur, flashIdx-1, start, end, next,
			)
		}
		cur = next
	}
	return cur
}

func flashMarkersWithPath(markers []Marker) []Marker {
	var out []Marker
	for _, m := range markers {
		if m.Type == MarkerFlash && m.LocalPath != "" {
			out = append(out, m)
		}
	}
	return out
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

// appendInputArgs writes -i (and photo loop flags) in graph order:
// pictures, voices, flash stills, SFX, optional overlay, Edit Audio.
func appendInputArgs(args []string, plan Plan) []string {
	for _, c := range plan.PictureClips() {
		if c.Type == AssetPhoto {
			// Still must be looped so zoompan can emit hold-duration frames.
			args = append(args, "-loop", "1", "-t", msToSec(c.DurationMs()), "-i", c.LocalPath)
			continue
		}
		args = append(args, "-i", c.LocalPath)
	}
	for _, c := range plan.VoiceClips() {
		args = append(args, "-i", c.LocalPath)
	}
	for _, m := range flashMarkersWithPath(plan.Markers) {
		args = append(args, "-loop", "1", "-t", msToSec(oneFrameMs()), "-i", m.LocalPath)
	}
	for _, m := range sfxMarkersWithPath(plan.Markers) {
		args = append(args, "-i", m.LocalPath)
	}
	if plan.OverlayPath != "" {
		args = append(args, "-loop", "1", "-t", msToSec(plan.PictureDurationMs()), "-i", plan.OverlayPath)
	}
	args = append(args, "-i", plan.EditAudioPath)
	return args
}
