"""Claim envelope → CutDraftJob parsing."""

from momentos_ml.worker.jobs import parse_claim_response


def test_parse_claim_merges_envelope_moment_id_when_payload_empty():
    job = parse_claim_response(
        {
            "id": "9b5c01f1-52cc-4312-a2df-c8e75556f298",
            "momentId": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
            "jobType": "cut-draft",
            "payload": {},
        }
    )
    assert job.job_id == "9b5c01f1-52cc-4312-a2df-c8e75556f298"
    assert job.moment_id == "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
    assert job.job_type == "cut-draft"


def test_parse_claim_uses_payload_moment_id():
    job = parse_claim_response(
        {
            "id": "job-1",
            "momentId": "envelope-moment",
            "jobType": "cut-draft",
            "payload": {
                "momentId": "payload-moment",
                "mode": "draft",
                "assets": [],
            },
        }
    )
    assert job.moment_id == "payload-moment"


def test_parse_claim_string_payload():
    job = parse_claim_response(
        {
            "id": "job-2",
            "momentId": "m-2",
            "payload": '{"momentId":"m-2","mode":"draft"}',
        }
    )
    assert job.moment_id == "m-2"
    assert job.mode == "draft"
