package media

import (
	"strings"
	"testing"
)

func TestKeyUsesAuthorPrefixAndExtension(t *testing.T) {
	key, err := Key(42, 123456789, "image/png")
	if err != nil {
		t.Fatalf("Key: %v", err)
	}
	if key != "media/42/123456789.png" {
		t.Fatalf("key inesperada: %s", key)
	}
	if !OwnedBy(key, 42) {
		t.Fatal("chave deveria pertencer ao autor 42")
	}
	if OwnedBy(key, 4) {
		t.Fatal("prefixo media/4 não pode casar com media/42")
	}
	if OwnedBy(key, 43) {
		t.Fatal("chave do autor 42 não pertence ao 43")
	}
}

func TestKeyRejectsUnsupportedType(t *testing.T) {
	if _, err := Key(1, 1, "application/pdf"); err == nil {
		t.Fatal("esperava erro para application/pdf")
	}
	if _, err := Key(1, 1, "text/plain"); err == nil {
		t.Fatal("esperava erro para text/plain")
	}
}

func TestAllowedContentTypeNormalizes(t *testing.T) {
	ext, ok := AllowedContentType(" Image/JPEG ")
	if !ok || ext != "jpg" {
		t.Fatalf("got %q, %v; want jpg, true", ext, ok)
	}
	ext, ok = AllowedContentType("image/webp")
	if !ok || ext != "webp" {
		t.Fatalf("got %q, %v; want webp, true", ext, ok)
	}
}

func TestConfigEnabledRequiresBucket(t *testing.T) {
	if (Config{}).Enabled() {
		t.Fatal("config sem bucket não pode estar habilitada")
	}
	if !(Config{Bucket: "b"}).Enabled() {
		t.Fatal("config com bucket deveria estar habilitada")
	}
}

func TestURLJoinsBase(t *testing.T) {
	s := &S3{}
	got := s.URL("https://cdn.example.com/", "media/1/a.png")
	if got != "https://cdn.example.com/media/1/a.png" {
		t.Fatalf("url: %s", got)
	}
	if strings.HasPrefix(s.URL("", "media/1/a.png"), "http") {
		t.Fatal("sem base a chave deve voltar crua")
	}
}
