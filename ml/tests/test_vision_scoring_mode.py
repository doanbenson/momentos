"""VISION_SCORING=metadata skips Gemini multimodal calls."""

from unittest.mock import MagicMock, patch

from momentos_ml.config import Settings
from momentos_ml.features.extract import AssetFeature
from momentos_ml.vision.score import score_montage_candidates


def _settings(**kwargs) -> Settings:
    base = dict(
        cut_planner="gemini",
        vision_scoring="metadata",
        gemini_api_key="fake-key",
        gemini_model="gemini-3.8-flash",
        moment_service_url=None,
        job_queue_url=None,
        internal_service_token=None,
        edit_audio_path=None,
        poll_interval_s=0.1,
        beat_cache_dir=".beat_cache_test",
    )
    base.update(kwargs)
    return Settings(**base)


def test_metadata_mode_does_not_call_gemini():
    feats = [
        AssetFeature(asset_id="a", type="photo", intake_order=0),
        AssetFeature(asset_id="b", type="photo", intake_order=1),
    ]
    with patch("momentos_ml.vision.score._score_gemini") as gemini:
        scores = score_montage_candidates(feats, settings=_settings())
        gemini.assert_not_called()
    assert len(scores) == 2
    assert scores[0].score >= scores[1].score


def test_gemini_mode_invokes_gemini_scorer():
    feats = [AssetFeature(asset_id="a", type="photo", intake_order=0)]
    fake = [MagicMock()]
    with patch("momentos_ml.vision.score._score_gemini", return_value=fake) as gemini:
        out = score_montage_candidates(
            feats, settings=_settings(vision_scoring="gemini")
        )
        gemini.assert_called_once()
    assert out is fake
