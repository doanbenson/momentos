"""Temp-file suffix must match Content-Type for Whisper/PyAV decode."""

from momentos_ml.media.download import _suffix_for


def test_suffix_prefers_quicktime_over_generic_video_type():
    assert _suffix_for(asset_type="video", content_type="video/quicktime") == ".mov"


def test_suffix_falls_back_to_asset_type():
    assert _suffix_for(asset_type="voice", content_type=None) == ".m4a"
    assert _suffix_for(asset_type="photo", content_type="application/octet-stream") == ".jpg"
