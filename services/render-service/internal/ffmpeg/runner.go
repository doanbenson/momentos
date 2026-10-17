package ffmpeg

// Runner invokes ffmpeg. Keep all encoding details here — not in ml/.
type Runner struct{}

func (r *Runner) RenderFromCut(_ cutJSON []byte, _ outPath string) error {
	return nil
}
