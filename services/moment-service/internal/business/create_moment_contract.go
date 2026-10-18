package business

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ValidateCreateMomentContract enforces Week Intake Reflection rules (seam 3).
// Reflection is identified by reflectionAssetId (must be in assetIds) plus a non-empty reflectionQuestion.
func ValidateCreateMomentContract(in CreateMomentInput) error {
	if strings.TrimSpace(in.ReflectionQuestion) == "" {
		return fmt.Errorf("%w: reflectionQuestion is required", ErrInvalid)
	}
	if in.ReflectionAssetID == uuid.Nil {
		return fmt.Errorf("%w: reflectionAssetId is required", ErrInvalid)
	}
	found := false
	for _, id := range in.AssetIDs {
		if id == in.ReflectionAssetID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("%w: reflectionAssetId must be included in assetIds", ErrInvalid)
	}
	return nil
}

// validateReflectionAssetType requires the Reflection Asset to be voice or video when media metadata is available.
func validateReflectionAssetType(assetType string) error {
	switch assetType {
	case "voice", "video":
		return nil
	case "":
		// Unknown type (SkipMedia / stub) — membership check is enough.
		return nil
	default:
		return fmt.Errorf("%w: reflectionAssetId must be a voice or video Asset (got %s)", ErrInvalid, assetType)
	}
}
