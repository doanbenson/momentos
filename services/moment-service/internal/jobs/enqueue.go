package jobs

// Job enqueue/claim/complete lives in internal/business + internal/database
// (Postgres SKIP LOCKED per ADR 0001). This package is retained as the
// documented jobs layer seam from claude/context.md.
