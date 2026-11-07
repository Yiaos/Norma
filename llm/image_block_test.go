package llm

import (
	"encoding/json"
	"strings"
	"testing"
)

// Image blocks must serialize to each wire format's inline-image shape:
// Anthropic source/base64, OpenAI image_url data URL, Responses input_image.
func TestImageBlockSerialization(t *testing.T) {
	msg := Message{Role: RoleUser, Content: []ContentBlock{
		TextBlock("what is this?"),
		ImageBlock("image/png", "aGVsbG8="),
	}}

	// OpenAI Chat Completions: user turn becomes an array of parts.
	oam := toOpenAIMessages("", []Message{msg})
	if len(oam) != 1 {
		t.Fatalf("want 1 message, got %d", len(oam))
	}
	parts, ok := oam[0].Content.([]oaContentPart)
	if !ok {
		t.Fatalf("content is not a parts array: %#v", oam[0].Content)
	}
	if len(parts) != 2 || parts[0].Type != "text" || parts[0].Text != "what is this?" {
		t.Fatalf("text part wrong: %+v", parts[0])
	}
	if parts[1].Type != "image_url" || parts[1].ImageURL == nil || parts[1].ImageURL.URL != "data:image/png;base64,aGVsbG8=" {
		t.Fatalf("image part wrong: %+v", parts[1])
	}

	// OpenAI Responses: user item becomes input_text + input_image parts.
	items := toResponsesInput([]Message{msg})
	if len(items) != 1 {
		t.Fatalf("want 1 item, got %d", len(items))
	}
	rparts, ok := items[0].Content.([]orContentPart)
	if !ok {
		t.Fatalf("content is not a parts array: %#v", items[0].Content)
	}
	if len(rparts) != 2 || rparts[0].Type != "input_text" || rparts[1].Type != "input_image" ||
		rparts[1].ImageURL != "data:image/png;base64,aGVsbG8=" {
		t.Fatalf("responses parts wrong: %+v", rparts)
	}

	// Anthropic: messages marshal directly, so the block must carry the
	// source object verbatim.
	b, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{`"type":"image"`, `"source"`, `"media_type":"image/png"`, `"data":"aGVsbG8="`} {
		if !strings.Contains(s, want) {
			t.Fatalf("anthropic JSON missing %s: %s", want, s)
		}
	}

	// Text-only user turns must keep serializing as plain strings on both
	// OpenAI formats (Content is `any` now; the string path must not regress).
	oam = toOpenAIMessages("", []Message{UserText("plain")})
	if oam[0].Content != "plain" {
		t.Fatalf("plain text content regressed: %#v", oam[0].Content)
	}
	items = toResponsesInput([]Message{UserText("plain")})
	if items[0].Content != "plain" {
		t.Fatalf("plain text responses content regressed: %#v", items[0].Content)
	}
}
