package handlers_test

import (
	"errors"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestHandlers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Handlers Suite")
}

type badResponseWriter struct {
	httptest.ResponseRecorder
}

func (w *badResponseWriter) Write([]byte) (int, error) {
	return 42, errors.New("some bad writer")
}

type badReader struct{}

func (r *badReader) Read(buffer []byte) (int, error) {
	return 0, errors.New("bad")
}
