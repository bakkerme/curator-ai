package llmutil

import "testing"

func TestUnmarshalJSONResponse(t *testing.T) {
	t.Parallel()

	type payload struct {
		Score  float64 `json:"score"`
		Reason string  `json:"reason"`
	}

	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name:    "raw json",
			content: `{"score":0.7,"reason":"ok"}`,
		},
		{
			name:    "json code fence",
			content: "```json\n{\"score\":0.7,\"reason\":\"ok\"}\n```",
		},
		{
			name:    "plain code fence",
			content: "```\n{\"score\":0.7,\"reason\":\"ok\"}\n```",
		},
		{
			name:    "invalid json in code fence",
			content: "```json\nnot-json\n```",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got payload
			err := UnmarshalJSONResponse(tt.content, &got)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if got.Score != 0.7 {
				t.Fatalf("expected score 0.7, got %f", got.Score)
			}
			if got.Reason != "ok" {
				t.Fatalf("expected reason ok, got %q", got.Reason)
			}
		})
	}
}
