package ffmpeg

import (
	"fmt"
	"strings"
)

// buildFilterComplex produces one filter_complex graph for picture + audio.
//
// Picture: scale/pad (or Ken-Burns for photos) → 1080x1920@30 → concat hard-cuts.
// Audio: Edit Audio bed trimmed to picture duration + VO adelay + sidechain ducking stub.
func buildFilterComplex(plan Plan) (filter string, videoLabel, audioLabel string, err error) {
	pics := plan.PictureClips()
	voices := plan.VoiceClips()
	if len(pics) == 0 {
		return "", "", "", fmt.Errorf("no picture clips")
	}

	// Input layout: [0..len(pics)-1] picture, then voices, then Edit Audio last.
	editIdx := len(pics) + len(voices)
	var b strings.Builder
	vLabels := make([]string, 0, len(pics))

	for i, c := range pics {
		label := fmt.Sprintf("v%d", i)
		switch c.Type {
		case AssetPhoto:
			frames := msToFrames(c.DurationMs())
			// scale up then zoompan Ken-Burns; output fixed canvas @ 30fps
			fmt.Fprintf(&b,
				"[%d:v]scale=%d:%d:force_original_aspect_ratio=increase,"+
					"crop=%d:%d,setsar=1,format=yuv420p,"+
					"zoompan=z='min(1.0+0.0015*on,1.12)':x='(iw-iw/zoom)/2':y='(ih-ih/zoom)/2':"+
					"d=%d:s=%dx%d:fps=%d[%s];",
				i, CanvasWidth, CanvasHeight, CanvasWidth, CanvasHeight,
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
		vLabels = append(vLabels, "["+label+"]")
	}

	if len(vLabels) == 1 {
		fmt.Fprintf(&b, "%scopy[vout];", vLabels[0])
	} else {
		fmt.Fprintf(&b, "%sconcat=n=%d:v=1:a=0[vout];", strings.Join(vLabels, ""), len(vLabels))
	}

	totalSec := msToSec(plan.PictureDurationMs())

	// Edit Audio bed
	fmt.Fprintf(&b,
		"[%d:a]aformat=sample_fmts=fltp:sample_rates=48000:channel_layouts=stereo,"+
			"atrim=0:%s,asetpts=PTS-STARTPTS[bed];",
		editIdx, totalSec,
	)

	if len(voices) == 0 {
		b.WriteString("[bed]anull[aout]")
		return strings.TrimSuffix(b.String(), ";"), "vout", "aout", nil
	}

	voLabels := make([]string, 0, len(voices))
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
		voLabels = append(voLabels, "["+label+"]")
	}

	voMix := "vos"
	if len(voLabels) == 1 {
		fmt.Fprintf(&b, "%scopy[%s];", voLabels[0], voMix)
	} else {
		fmt.Fprintf(&b, "%samix=inputs=%d:duration=longest:dropout_transition=0[%s];",
			strings.Join(voLabels, ""), len(voLabels), voMix)
	}

	// Ducking stub: sidechaincompress bed under VO, then remixed with VO
	fmt.Fprintf(&b,
		"[bed][%s]sidechaincompress=threshold=0.05:ratio=8:attack=50:release=300:makeup=1[ducked];"+
			"[ducked][%s]amix=inputs=2:duration=first:dropout_transition=0[aout]",
		voMix, voMix,
	)

	return b.String(), "vout", "aout", nil
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

// appendInputArgs writes -i (and photo loop flags) in graph order: pictures, voices, Edit Audio.
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
	args = append(args, "-i", plan.EditAudioPath)
	return args
}
