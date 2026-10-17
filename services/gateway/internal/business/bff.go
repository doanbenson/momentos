package business

// BFF orchestrates calls to moment-service, media-service, and render-service.
// No domain persistence lives here.
type BFF struct {
	MomentURL string
	MediaURL  string
	RenderURL string
}
