package business

import (
	"errors"
	"fmt"
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

// Cap 20 = total assetIds including the Reflection (ADR 0016 / seam 3).

func TestCreateMomentContract_acceptsTwentyAssetsIncludingReflection(t *testing.T) {
	reflectionID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ids := make([]uuid.UUID, 0, 20)
	ids = append(ids, reflectionID)
	for i := 0; i < 19; i++ {
		ids = append(ids, uuid.MustParse(fmt.Sprintf("22222222-2222-2222-2222-%012d", i+1)))
	}
	err := ValidateCreateMomentContract(CreateMomentInput{
		Principal:          "user-1",
		AssetIDs:           ids,
		ReflectionAssetID:  reflectionID,
		ReflectionQuestion: "What made this week yours?",
	})
	if err != nil {
		t.Fatalf("want ok for 20 Assets, got %v", err)
	}
}

func TestCreateMomentContract_rejectsTwentyOneAssets(t *testing.T) {
	reflectionID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ids := make([]uuid.UUID, 0, 21)
	ids = append(ids, reflectionID)
	for i := 0; i < 20; i++ {
		ids = append(ids, uuid.MustParse(fmt.Sprintf("22222222-2222-2222-2222-%012d", i+1)))
	}
	err := ValidateCreateMomentContract(CreateMomentInput{
		Principal:          "user-1",
		AssetIDs:           ids,
		ReflectionAssetID:  reflectionID,
		ReflectionQuestion: "What made this week yours?",
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid for 21 Assets, got %v", err)
	}
}
