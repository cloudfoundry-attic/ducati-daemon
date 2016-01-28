package handlers_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/cloudfoundry-incubator/ducati-daemon/handlers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("List", func() {
	It("should return the containers as a JSON list", func() {
		containers := []handlers.Container{
			handlers.Container{ID: "some-container"},
			handlers.Container{ID: "some-other-container"},
		}

		handler := &handlers.ListHandler{
			Containers: containers,
		}

		req, err := http.NewRequest("GET", "/containers", nil)
		Expect(err).NotTo(HaveOccurred())
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)

		Expect(resp.Body.String()).To(MatchJSON(`[
				{
					"id": "some-container"
				},
				{
					"id": "some-other-container"
				}
		]`))
	})

	Context("when there are no containers", func() {
		It("should return an empty list", func() {
			handler := &handlers.ListHandler{
				Containers: nil,
			}

			req, err := http.NewRequest("GET", "/containers", nil)
			Expect(err).NotTo(HaveOccurred())
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)

			Expect(resp.Body.String()).To(MatchJSON(`[]`))

		})
	})

})
