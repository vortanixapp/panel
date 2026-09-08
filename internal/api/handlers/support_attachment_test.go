package handlers

import "testing"

func TestSupportAttachmentInfo(t *testing.T) {
	ticket := "11111111-1111-4111-8111-111111111111"
	msg := "22222222-2222-4222-8222-222222222222"

	t.Run("обычное сообщение без вложения", func(t *testing.T) {
		if got := supportAttachmentInfo([]byte(`{}`), ticket, msg); got != nil {
			t.Fatalf("expected nil for plain message, got %v", got)
		}
		if got := supportAttachmentInfo(nil, ticket, msg); got != nil {
			t.Fatalf("expected nil for empty meta, got %v", got)
		}
	})

	t.Run("новое вложение получает ссылку", func(t *testing.T) {
		meta := []byte(`{"attachment":{"name":"screen.png","size":1234,"content_type":"image/png","ext":".png"}}`)
		got := supportAttachmentInfo(meta, ticket, msg)
		if got == nil {
			t.Fatal("expected attachment info")
		}
		if got["name"] != "screen.png" {
			t.Fatalf("unexpected name: %v", got["name"])
		}
		want := "/v1/support/" + ticket + "/attachments/" + msg
		if got["url"] != want {
			t.Fatalf("unexpected url: %v", got["url"])
		}
	})

	t.Run("legacy запись без ext остаётся без ссылки", func(t *testing.T) {
		meta := []byte(`{"attachment":{"name":"old.png","size":"512"}}`)
		got := supportAttachmentInfo(meta, ticket, msg)
		if got == nil {
			t.Fatal("expected attachment info for legacy record")
		}
		if _, hasURL := got["url"]; hasURL {
			t.Fatal("legacy record must not advertise a download url")
		}
	})
}

func TestSupportAttachmentTypes(t *testing.T) {
	for _, ext := range []string{".html", ".svg", ".exe", ".sh", ".php", ""} {
		if _, ok := supportAttachmentTypes[ext]; ok {
			t.Fatalf("extension %q must not be accepted", ext)
		}
	}
	for _, ext := range []string{".png", ".jpg", ".txt", ".zip", ".pdf"} {
		if _, ok := supportAttachmentTypes[ext]; !ok {
			t.Fatalf("extension %q should be accepted", ext)
		}
	}
}
