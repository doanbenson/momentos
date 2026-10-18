// Package songpick holds MVP Song Pick / Style Profile constants (ADR 0007).
// Numeric parity source: packages/contracts/mvp-constants.json
package songpick

const (
	SongPickID     = "live-a-life-you-will-remember"
	SongPickTitle  = "Live A Life You Will Remember"
	StyleProfileID = "life-on-film-v1"
	VibeID         = "live-a-life-remember-v0"

	// NominalBPM calibrates offline planning; Beat Grid still comes from Edit Audio.
	NominalBPM = 126

	HeroOutroSpanBeats  = 4
	PhotoHoldBeats      = 2
	ChorusDropBeatIndex = 96

	// J-cut / post-speech drop timing (ADR 0019). ml plans markers; render mixes.
	JCutPreRollSec           = 4.0
	DropOffsetAfterSpeechSec = 0.25
)

// LegalSpanBeats are the only legal clip lengths in beats for this Song Pick.
var LegalSpanBeats = []int{1, 2, 4}

// IsLegalSpanBeats reports whether span is a legal 1/2/4-beat placement length.
func IsLegalSpanBeats(span int) bool {
	for _, v := range LegalSpanBeats {
		if v == span {
			return true
		}
	}
	return false
}
