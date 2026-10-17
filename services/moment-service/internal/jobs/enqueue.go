package jobs

// Enqueuer publishes async work for ml (cut-draft) and render-service (render).
type Enqueuer struct{}

func (e *Enqueuer) CutDraft(momentID string) error  { return nil }
func (e *Enqueuer) Render(momentID, cutID string) error { return nil }
