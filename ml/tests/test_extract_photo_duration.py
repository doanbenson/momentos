"""Photos and zero-duration Assets get a synthetic source hold window."""

from momentos_ml.features.extract import JobAsset, extract_features


def test_photo_gets_synthetic_10s_duration():
    features = extract_features(
        [
            {"id": "p1", "type": "photo", "order": 0},
            JobAsset(id="p2", type="photo", order=1, durationMs=0),
        ]
    )
    assert features[0].duration_ms == 10_000
    assert features[1].duration_ms == 10_000


def test_zero_duration_video_gets_synthetic_10s():
    features = extract_features(
        [{"id": "v1", "type": "video", "order": 0, "durationMs": 0}]
    )
    assert features[0].duration_ms == 10_000


def test_positive_video_duration_preserved():
    features = extract_features(
        [{"id": "v1", "type": "video", "order": 0, "durationMs": 3500}]
    )
    assert features[0].duration_ms == 3500
