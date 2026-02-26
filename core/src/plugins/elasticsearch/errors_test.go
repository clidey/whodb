package elasticsearch

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/elastic/go-elasticsearch/v8/esapi"
)

func TestFormatElasticErrorReadsAndRestoresBody(t *testing.T) {
	original := []byte("  something went wrong \n")
	res := &esapi.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(bytes.NewReader(original)),
	}

	msg := formatElasticError(res)
	if msg != "something went wrong" {
		t.Fatalf("expected trimmed body message, got %q", msg)
	}

	// Ensure callers can still read the body after formatting.
	restored, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("failed to read restored body: %v", err)
	}
	if !bytes.Equal(restored, original) {
		t.Fatalf("expected restored body %q, got %q", string(original), string(restored))
	}
}

func TestFormatElasticErrorHandlesNil(t *testing.T) {
	if got := formatElasticError(nil); got != "unknown error" {
		t.Fatalf("expected unknown error, got %q", got)
	}
}
