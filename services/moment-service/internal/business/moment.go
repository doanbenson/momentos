package business

// Service holds Moment / Cut use cases. Persistence via database package; jobs via jobs package.
type Service struct{}

// CreateFromIntake creates a Moment from Week Intake Asset refs and enqueues cut-draft.
func (s *Service) CreateFromIntake(_ /* principalID */ string, _ /* assetIDs */ []string) error {
	return nil
}
