package business

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

// Seam 3: Week Intake / CreateMoment requires Reflection Asset + reflectionQuestion.

func TestCreateMomentContract_rejectsMissingReflectionQuestion(t *testing.T) {
	reflectionID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	err := ValidateCreateMomentContract(CreateMomentInput{
		Principal:          "user-1",
		AssetIDs:           []uuid.UUID{reflectionID},
		ReflectionAssetID:  reflectionID,
		ReflectionQuestion: "",
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
}

func TestCreateMomentContract_rejectsMissingReflectionAsset(t *testing.T) {
	photoID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	err := ValidateCreateMomentContract(CreateMomentInput{
		Principal:          "user-1",
		AssetIDs:           []uuid.UUID{photoID},
		ReflectionQuestion: "What made this week yours?",
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
}

func TestCreateMomentContract_rejectsReflectionAssetNotInAssetIds(t *testing.T) {
	reflectionID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	photoID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	err := ValidateCreateMomentContract(CreateMomentInput{
		Principal:          "user-1",
		AssetIDs:           []uuid.UUID{photoID},
		ReflectionAssetID:  reflectionID,
		ReflectionQuestion: "What made this week yours?",
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
}

func TestCreateMomentContract_acceptsReflectionAndQuestion(t *testing.T) {
	reflectionID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	photoID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	err := ValidateCreateMomentContract(CreateMomentInput{
		Principal:          "user-1",
		AssetIDs:           []uuid.UUID{reflectionID, photoID},
		ReflectionAssetID:  reflectionID,
		ReflectionQuestion: "What made this week yours?",
	})
	if err != nil {
		t.Fatalf("want ok, got %v", err)
	}
}
