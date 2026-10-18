package business

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/doanbenson/momentos/services/moment-service/internal/database"
)

func TestApplyEditorPatch_trimReflowPreservesLookAndRendersWithoutCutDraft(t *testing.T) {
	reflectionID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	montageID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	momentID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	act1 := 1
	act3 := 3
	clean := "clean_916"
	gate := "super8_gate"
	none := "none"
	hard := "hard_cut"
	drift := "handheld_super8_drift"
	clips := []clipEdit{
		{
			AssetID: reflectionID, Order: 0, StartMs: 100, EndMs: 8100,
			Role: "reflection", Act: &act1, VisualStyle: &clean, TransitionIn: &none,
		},
		{
			AssetID: montageID, Order: 1, StartMs: 8100, EndMs: 11100,
			Role: "montage", Act: &act3, MotionPreset: &drift, VisualStyle: &gate, TransitionIn: &hard,
		},
	}
	captions := []captionEdit{
		{Text: "week", StartMs: 1000, EndMs: 1800},
		{Text: "drop", StartMs: 9000, EndMs: 9500},
	}
	markers := json.RawMessage(`[{"type":"flash","atMs":9000}]`)
	store := &fakeStore{
		moment: &database.Moment{
			ID:         momentID,
			UserID:     "user-1",
			Status:     database.StatusReady,
			SongPickID: "live-a-life-you-will-remember",
			AssetIDs:   []uuid.UUID{reflectionID, montageID},
		},
		cut: &database.Cut{
			ID:         uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd"),
			MomentID:   momentID,
			Version:    4,
			SongPickID: "live-a-life-you-will-remember",
			Clips:      mustJSON(t, clips),
			Captions:   mustJSON(t, captions),
			Markers:    markers,
		},
	}
	svc := &Service{Store: store, SongPickID: DefaultSongPickID, SongPickTitle: DefaultSongPickTitle}

	var patch EditorPatch
	if err := json.Unmarshal([]byte(`{
		"trims":[{"assetId":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","sourceInMs":500,"sourceOutMs":6500}],
		"reRender":true
	}`), &patch); err != nil {
		t.Fatal(err)
	}
	dto, err := svc.ApplyEditorPatch(context.Background(), "user-1", momentID, patch)
	if err != nil {
		t.Fatal(err)
	}

	if dto.Version != 5 {
		t.Fatalf("version = %d, want 5", dto.Version)
	}
	got := decodeClipsForTest(t, dto.Clips)
	if len(got) != 2 {
		t.Fatalf("clips = %d", len(got))
	}
	ref, mont := got[0], got[1]
	if ref.AssetID != reflectionID || ref.Order != 0 || ref.StartMs != 0 || ref.EndMs != 6000 {
		t.Fatalf("reflection timeline = %+v", ref)
	}
	if ref.SourceInMs == nil || *ref.SourceInMs != 500 || ref.SourceOutMs == nil || *ref.SourceOutMs != 6500 {
		t.Fatalf("reflection source = in %v out %v", ref.SourceInMs, ref.SourceOutMs)
	}
	if ref.Role != "reflection" || ref.Act == nil || *ref.Act != 1 || ref.MotionPreset != nil ||
		ref.VisualStyle == nil || *ref.VisualStyle != clean || ref.TransitionIn == nil || *ref.TransitionIn != none {
		t.Fatalf("reflection look = %+v", ref)
	}
	if mont.AssetID != montageID || mont.Order != 1 || mont.StartMs != 6000 || mont.EndMs != 9000 {
		t.Fatalf("montage timeline = %+v", mont)
	}
	if mont.SourceInMs != nil || mont.SourceOutMs != nil {
		t.Fatalf("montage source window should stay unset")
	}
	if mont.Role != "montage" || mont.Act == nil || *mont.Act != 3 || mont.MotionPreset == nil || *mont.MotionPreset != drift ||
		mont.VisualStyle == nil || *mont.VisualStyle != gate || mont.TransitionIn == nil || *mont.TransitionIn != hard {
		t.Fatalf("montage look = %+v", mont)
	}

	gotCaps := decodeCaptionsForTest(t, dto.Captions)
	if len(gotCaps) != 2 || gotCaps[0].Text != "week" || gotCaps[0].StartMs != 500 || gotCaps[0].EndMs != 1300 {
		t.Fatalf("reflection caption = %+v", gotCaps)
	}
	if gotCaps[1].Text != "drop" || gotCaps[1].StartMs != 9000 || gotCaps[1].EndMs != 9500 {
		t.Fatalf("non-reflection caption = %+v", gotCaps[1])
	}
	if string(dto.Markers) != string(markers) {
		t.Fatalf("markers = %s", dto.Markers)
	}
	if len(store.assetIDs) != 2 || store.assetIDs[0] != reflectionID || store.assetIDs[1] != montageID {
		t.Fatalf("moment assets = %v", store.assetIDs)
	}
	if store.status != database.StatusRendering {
		t.Fatalf("status = %s", store.status)
	}
	if len(store.jobs) != 1 || store.jobs[0].JobType != database.JobRender {
		t.Fatalf("jobs = %+v", jobTypes(store.jobs))
	}
	for _, kind := range store.superseded {
		if kind == database.JobCutDraft {
			t.Fatal("reRender superseded a cut-draft job")
		}
	}
	var payload RenderPayload
	if err := json.Unmarshal(store.jobs[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.JobType != string(database.JobRender) {
		t.Fatalf("payload jobType = %s", payload.JobType)
	}
	rendered := decodeClipsForTest(t, payload.Cut.Clips)
	if len(rendered) != 2 || rendered[0].SourceInMs == nil || *rendered[0].SourceInMs != 500 || rendered[0].EndMs != 6000 {
		t.Fatalf("render payload clips = %+v", rendered)
	}
}

func TestApplyEditorPatch_trimWithoutFlagsPersistsAndDoesNotEnqueue(t *testing.T) {
	reflectionID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	momentID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	act := 1
	visual := "clean_916"
	transition := "none"
	store := newCutStore(t, momentID, "user-1", []clipEdit{{
		AssetID: reflectionID, Order: 0, StartMs: 0, EndMs: 4000,
		Role: "reflection", Act: &act, VisualStyle: &visual, TransitionIn: &transition,
	}}, nil)
	svc := &Service{Store: store}
	var patch EditorPatch
	if err := json.Unmarshal([]byte(`{"trims":[{"assetId":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","sourceInMs":100,"sourceOutMs":1100}]}`), &patch); err != nil {
		t.Fatal(err)
	}
	dto, err := svc.ApplyEditorPatch(context.Background(), "user-1", momentID, patch)
	if err != nil {
		t.Fatal(err)
	}
	got := decodeClipsForTest(t, dto.Clips)
	if got[0].StartMs != 0 || got[0].EndMs != 1000 || got[0].SourceInMs == nil || *got[0].SourceInMs != 100 {
		t.Fatalf("clip = %+v", got[0])
	}
	if len(store.jobs) != 0 {
		t.Fatalf("enqueued %v", jobTypes(store.jobs))
	}
	if store.status != "" {
		t.Fatalf("status changed to %s", store.status)
	}
}

func TestApplyEditorPatch_reDraftStillEnqueuesCutDraft(t *testing.T) {
	reflectionID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	momentID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	act := 1
	visual := "clean_916"
	transition := "none"
	store := newCutStore(t, momentID, "user-1", []clipEdit{{
		AssetID: reflectionID, StartMs: 0, EndMs: 4000,
		Role: "reflection", Act: &act, VisualStyle: &visual, TransitionIn: &transition,
	}}, nil)
	svc := &Service{Store: store, SongPickID: DefaultSongPickID, SongPickTitle: DefaultSongPickTitle}
	var patch EditorPatch
	if err := json.Unmarshal([]byte(`{"reDraft":true,"reRender":true}`), &patch); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ApplyEditorPatch(context.Background(), "user-1", momentID, patch); err != nil {
		t.Fatal(err)
	}
	if len(store.jobs) != 1 || store.jobs[0].JobType != database.JobCutDraft {
		t.Fatalf("jobs = %v", jobTypes(store.jobs))
	}
	if store.status != database.StatusDrafting {
		t.Fatalf("status = %s", store.status)
	}
}

func TestApplyEditorPatch_rejectsBadTrimAndLastAsset(t *testing.T) {
	reflectionID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	montageID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	momentID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	act := 1
	visual := "clean_916"
	transition := "none"
	two := []clipEdit{
		{AssetID: reflectionID, StartMs: 0, EndMs: 4000, Role: "reflection", Act: &act, VisualStyle: &visual, TransitionIn: &transition},
		{AssetID: montageID, StartMs: 4000, EndMs: 7000, Role: "montage", Act: &act, VisualStyle: &visual, TransitionIn: &transition},
	}

	t.Run("sourceOut equals sourceIn", func(t *testing.T) {
		store := newCutStore(t, momentID, "user-1", two, nil)
		err := applyPatch(t, store, momentID, `{"trims":[{"assetId":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","sourceInMs":1000,"sourceOutMs":1000}]}`)
		if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "sourceOutMs") {
			t.Fatalf("err = %v", err)
		}
		if store.cut.Version != 4 || len(store.jobs) != 0 {
			t.Fatalf("version %d jobs %v", store.cut.Version, jobTypes(store.jobs))
		}
	})
	t.Run("sourceOut before sourceIn", func(t *testing.T) {
		store := newCutStore(t, momentID, "user-1", two, nil)
		err := applyPatch(t, store, momentID, `{"trims":[{"assetId":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","sourceInMs":2000,"sourceOutMs":1500}]}`)
		if !errors.Is(err, ErrInvalid) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("remove last clip", func(t *testing.T) {
		store := newCutStore(t, momentID, "user-1", two[:1], nil)
		err := applyPatch(t, store, momentID, `{"removeAssetId":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"}`)
		if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "last asset") {
			t.Fatalf("err = %v", err)
		}
		if store.cut.Version != 4 || len(store.jobs) != 0 || len(store.assetIDs) != 0 {
			t.Fatalf("persisted version %d jobs %v assets %v", store.cut.Version, jobTypes(store.jobs), store.assetIDs)
		}
	})
	t.Run("remove last moment asset before cut", func(t *testing.T) {
		store := &fakeStore{moment: &database.Moment{
			ID: momentID, UserID: "user-1", Status: database.StatusDrafting,
			SongPickID: "song", AssetIDs: []uuid.UUID{reflectionID},
		}}
		err := applyPatch(t, store, momentID, `{"removeAssetId":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"}`)
		if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "last asset") {
			t.Fatalf("err = %v", err)
		}
		if len(store.assetIDs) != 0 {
			t.Fatalf("assets updated: %v", store.assetIDs)
		}
	})
}

func TestApplyCutEdits_replaceResetsSourceWindow(t *testing.T) {
	from := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	to := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	act := 1
	visual := "clean_916"
	transition := "none"
	in, out := 1000, 5000
	clips := []clipEdit{{
		AssetID: from, Order: 0, StartMs: 200, EndMs: 4200,
		SourceInMs: &in, SourceOutMs: &out,
		Role: "reflection", Act: &act, VisualStyle: &visual, TransitionIn: &transition,
	}}
	captions := []captionEdit{{Text: "hi", StartMs: 500, EndMs: 900}}
	var patch EditorPatch
	if err := json.Unmarshal([]byte(`{"replace":{"fromAssetId":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","toAssetId":"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"}}`), &patch); err != nil {
		t.Fatal(err)
	}
	rawClips, rawCaps, ids, err := applyCutEdits(mustJSON(t, clips), mustJSON(t, captions), patch)
	if err != nil {
		t.Fatal(err)
	}
	got := decodeClipsForTest(t, rawClips)
	if len(got) != 1 || got[0].AssetID != to || got[0].Role != "reflection" || got[0].StartMs != 0 || got[0].EndMs != 4000 {
		t.Fatalf("clip = %+v", got)
	}
	if got[0].SourceInMs == nil || *got[0].SourceInMs != 0 || got[0].SourceOutMs == nil || *got[0].SourceOutMs != 4000 {
		t.Fatalf("source = %v %v", got[0].SourceInMs, got[0].SourceOutMs)
	}
	if got[0].VisualStyle == nil || *got[0].VisualStyle != visual || got[0].TransitionIn == nil || *got[0].TransitionIn != transition {
		t.Fatalf("look = %+v", got[0])
	}
	gotCaps := decodeCaptionsForTest(t, rawCaps)
	if len(gotCaps) != 1 || gotCaps[0].StartMs != 1500 || gotCaps[0].EndMs != 1900 {
		t.Fatalf("captions = %+v", gotCaps)
	}
	if len(ids) != 1 || ids[0] != to {
		t.Fatalf("assets = %v", ids)
	}
}

func TestListMoments_newestFirstOmitsRenderURL(t *testing.T) {
	older := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	newer := older.Add(2 * time.Hour)
	key := "renders/ready.mp4"
	oldID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	newID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	otherID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	store := &fakeStore{moments: []database.Moment{
		{ID: oldID, UserID: "user-1", Status: database.StatusDrafting, SongPickID: "live-a-life-you-will-remember", CreatedAt: older},
		{ID: otherID, UserID: "user-2", Status: database.StatusReady, SongPickID: "other", CreatedAt: newer},
		{ID: newID, UserID: "user-1", Status: database.StatusReady, SongPickID: "live-a-life-you-will-remember", CreatedAt: newer, RenderStorageKey: &key},
	}}
	svc := &Service{Store: store}
	items, err := svc.ListMoments(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len = %d", len(items))
	}
	if items[0].ID != newID || items[0].Status != database.StatusReady || !items[0].CreatedAt.Equal(newer) || items[0].SongPickID != "live-a-life-you-will-remember" {
		t.Fatalf("first = %+v", items[0])
	}
	if items[0].RenderURL != nil {
		t.Fatalf("renderUrl = %s", *items[0].RenderURL)
	}
	if items[1].ID != oldID || items[1].Status != database.StatusDrafting {
		t.Fatalf("second = %+v", items[1])
	}
	if _, err := svc.ListMoments(context.Background(), ""); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("empty principal err = %v", err)
	}
}

func newCutStore(t *testing.T, momentID uuid.UUID, user string, clips []clipEdit, captions []captionEdit) *fakeStore {
	t.Helper()
	ids := make([]uuid.UUID, len(clips))
	for i, c := range clips {
		ids[i] = c.AssetID
	}
	caps := json.RawMessage(`[]`)
	if captions != nil {
		caps = mustJSON(t, captions)
	}
	return &fakeStore{
		moment: &database.Moment{
			ID: momentID, UserID: user, Status: database.StatusReady,
			SongPickID: "live-a-life-you-will-remember", AssetIDs: ids,
		},
		cut: &database.Cut{
			ID: uuid.New(), MomentID: momentID, Version: 4,
			SongPickID: "live-a-life-you-will-remember",
			Clips:      mustJSON(t, clips),
			Captions:   caps,
			Markers:    json.RawMessage(`[]`),
		},
	}
}

func applyPatch(t *testing.T, store *fakeStore, momentID uuid.UUID, raw string) error {
	t.Helper()
	var patch EditorPatch
	if err := json.Unmarshal([]byte(raw), &patch); err != nil {
		t.Fatal(err)
	}
	svc := &Service{Store: store, SongPickID: DefaultSongPickID, SongPickTitle: DefaultSongPickTitle}
	_, err := svc.ApplyEditorPatch(context.Background(), store.moment.UserID, momentID, patch)
	return err
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func decodeClipsForTest(t *testing.T, raw json.RawMessage) []clipEdit {
	t.Helper()
	var clips []clipEdit
	if err := json.Unmarshal(raw, &clips); err != nil {
		t.Fatal(err)
	}
	return clips
}

func decodeCaptionsForTest(t *testing.T, raw json.RawMessage) []captionEdit {
	t.Helper()
	var captions []captionEdit
	if err := json.Unmarshal(raw, &captions); err != nil {
		t.Fatal(err)
	}
	return captions
}

func jobTypes(jobs []*database.Job) []database.JobType {
	out := make([]database.JobType, len(jobs))
	for i, j := range jobs {
		out[i] = j.JobType
	}
	return out
}

type fakeStore struct {
	moment     *database.Moment
	cut        *database.Cut
	moments    []database.Moment
	jobs       []*database.Job
	superseded []database.JobType
	status     database.MomentStatus
	assetIDs   []uuid.UUID
}

func (f *fakeStore) GetMomentForUser(_ context.Context, id uuid.UUID, userID string) (*database.Moment, error) {
	if f.moment == nil || f.moment.ID != id || f.moment.UserID != userID {
		return nil, database.ErrNotFound
	}
	cp := *f.moment
	cp.AssetIDs = append([]uuid.UUID(nil), f.moment.AssetIDs...)
	return &cp, nil
}

func (f *fakeStore) ListMomentsForUser(_ context.Context, userID string) ([]database.Moment, error) {
	out := make([]database.Moment, 0)
	for _, m := range f.moments {
		if m.UserID == userID {
			out = append(out, m)
		}
	}
	return out, nil
}

func (f *fakeStore) GetCutByMoment(_ context.Context, momentID uuid.UUID) (*database.Cut, error) {
	if f.cut == nil || f.cut.MomentID != momentID {
		return nil, database.ErrNotFound
	}
	return f.copyCut(), nil
}

func (f *fakeStore) UpsertCut(_ context.Context, momentID uuid.UUID, songPickID string, clips, captions, markers json.RawMessage) (*database.Cut, error) {
	if f.cut == nil || f.cut.MomentID != momentID {
		f.cut = &database.Cut{ID: uuid.New(), MomentID: momentID, Version: 1}
	} else {
		f.cut.Version++
	}
	f.cut.SongPickID = songPickID
	f.cut.Clips = append(json.RawMessage(nil), clips...)
	f.cut.Captions = append(json.RawMessage(nil), captions...)
	f.cut.Markers = append(json.RawMessage(nil), markers...)
	return f.copyCut(), nil
}

func (f *fakeStore) UpdateMomentAssetIDs(_ context.Context, id uuid.UUID, assetIDs []uuid.UUID) error {
	if f.moment == nil || f.moment.ID != id {
		return database.ErrNotFound
	}
	f.assetIDs = append([]uuid.UUID(nil), assetIDs...)
	f.moment.AssetIDs = append([]uuid.UUID(nil), assetIDs...)
	return nil
}

func (f *fakeStore) SetMomentStatus(_ context.Context, id uuid.UUID, status database.MomentStatus, _, _ *string) error {
	if f.moment == nil || f.moment.ID != id {
		return database.ErrNotFound
	}
	f.status = status
	f.moment.Status = status
	return nil
}

func (f *fakeStore) SupersedeActiveJobs(_ context.Context, _ uuid.UUID, jobType database.JobType) error {
	f.superseded = append(f.superseded, jobType)
	return nil
}

func (f *fakeStore) InsertJob(_ context.Context, j *database.Job) error {
	cp := *j
	cp.Payload = append(json.RawMessage(nil), j.Payload...)
	f.jobs = append(f.jobs, &cp)
	return nil
}

func (f *fakeStore) copyCut() *database.Cut {
	cp := *f.cut
	cp.Clips = append(json.RawMessage(nil), f.cut.Clips...)
	cp.Captions = append(json.RawMessage(nil), f.cut.Captions...)
	cp.Markers = append(json.RawMessage(nil), f.cut.Markers...)
	return &cp
}

func (f *fakeStore) GetIdempotency(context.Context, string, string) (*database.IdempotencyRecord, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeStore) GetMoment(context.Context, uuid.UUID) (*database.Moment, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeStore) CreateMomentWithJob(context.Context, *database.Moment, *database.Job, *database.IdempotencyRecord) error {
	return errors.New("not implemented")
}
func (f *fakeStore) ClaimJob(context.Context, database.JobType, string, time.Duration) (*database.Job, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeStore) HeartbeatJob(context.Context, uuid.UUID, string, time.Duration) error {
	return errors.New("not implemented")
}
func (f *fakeStore) GetJob(context.Context, uuid.UUID) (*database.Job, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeStore) CompleteJobSuccess(context.Context, uuid.UUID, json.RawMessage) error {
	return errors.New("not implemented")
}
func (f *fakeStore) CompleteJobFailure(context.Context, uuid.UUID, string, string) error {
	return errors.New("not implemented")
}
func (f *fakeStore) AttachCut(context.Context, uuid.UUID, uuid.UUID) error {
	return errors.New("not implemented")
}
func (f *fakeStore) SetMomentReady(context.Context, uuid.UUID, string) error {
	return errors.New("not implemented")
}
