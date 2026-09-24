package domain

import "testing"

func TestNoteBlockCommentInputNormalization(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mentions   []MentionReference
		wantErr    bool
		wantBody   string
		wantTarget string
	}{
		{
			name:       "normalizes body and mention strings",
			body:       "  comment body  ",
			mentions:   []MentionReference{{Kind: " person ", TargetID: " target ", DisplayText: " display ", RangeStart: 1, RangeLength: 7}},
			wantBody:   "comment body",
			wantTarget: "target",
		},
		{
			name:    "rejects blank body",
			body:    "   ",
			wantErr: true,
		},
		{
			name:     "rejects unknown mention kind",
			body:     "comment body",
			mentions: []MentionReference{{Kind: "unknown", TargetID: "target", DisplayText: "display"}},
			wantErr:  true,
		},
		{
			name:     "rejects negative mention range",
			body:     "comment body",
			mentions: []MentionReference{{Kind: "person", TargetID: "target", DisplayText: "display", RangeStart: -1}},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, mentions, err := normalizeNoteBlockCommentInput(tt.body, tt.mentions)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalize error = %v, want error %t", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if body != tt.wantBody {
				t.Fatalf("body = %q, want %q", body, tt.wantBody)
			}
			if len(mentions) != 1 || mentions[0].TargetID != tt.wantTarget {
				t.Fatalf("mentions = %#v, want normalized target %q", mentions, tt.wantTarget)
			}
		})
	}
}

func TestNoteBlockCommentParentIDNormalization(t *testing.T) {
	if got := normalizeParentCommentID(nil); got != nil {
		t.Fatalf("nil parent = %v, want nil", got)
	}
	empty := "  "
	if got := normalizeParentCommentID(&empty); got != nil {
		t.Fatalf("blank parent = %v, want nil", got)
	}
	parent := " parent-1 "
	got := normalizeParentCommentID(&parent)
	if got == nil || *got != "parent-1" {
		t.Fatalf("parent = %v, want trimmed parent", got)
	}
}
