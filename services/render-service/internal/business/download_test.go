package business

import "testing"

func TestSuffixFor_ContentTypeWins(t *testing.T) {
	cases := []struct {
		ct, typ, want string
	}{
		{"image/heic", "photo", ".heic"},
		{"image/jpeg; charset=binary", "photo", ".jpg"},
		{"video/quicktime", "video", ".mov"},
		{"video/mp4", "video", ".mp4"},
		{"audio/mp4", "voice", ".m4a"},
		{"application/octet-stream", "photo", ".jpg"},
		{"", "video", ".mp4"},
		{"", "voice", ".m4a"},
		{"", "", ".bin"},
	}
	for _, tc := range cases {
		got := SuffixFor(tc.ct, tc.typ)
		if got != tc.want {
			t.Fatalf("SuffixFor(%q,%q)=%q want %q", tc.ct, tc.typ, got, tc.want)
		}
	}
}

func TestIsHEICPath(t *testing.T) {
	if !isHEICPath(`/tmp/a.heic`, "") {
		t.Fatal("expected .heic")
	}
	if !isHEICPath(`/tmp/a.bin`, "image/heic") {
		t.Fatal("expected heic content-type")
	}
	if isHEICPath(`/tmp/a.jpg`, "image/jpeg") {
		t.Fatal("jpg should not be heic")
	}
}

func TestClassifyEncodeError_LoopIsTerminal(t *testing.T) {
	err := classifyEncodeError(fmtError("ffmpeg exit 1: Option loop not found.\nError opening input file /tmp/x"))
	if !IsTerminal(err) {
		t.Fatalf("want terminal, got %v", err)
	}
	retryable := classifyEncodeError(fmtError("ffmpeg exit 1: Conversion failed!"))
	if IsTerminal(retryable) {
		t.Fatalf("generic encode should retry, got terminal %v", retryable)
	}
}

// tiny helper so test file does not import fmt solely for errors.
type fmtError string

func (e fmtError) Error() string { return string(e) }
