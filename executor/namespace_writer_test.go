package executor_test

import (
	"errors"
	"os"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NamespaceWriter", func() {
	var nsWriter *executor.NamespaceWriter
	var ns *fakes.Namespace
	var writer *fakes.Writer

	BeforeEach(func() {
		ns = &fakes.Namespace{}
		ns.ExecuteStub = func(callback func(*os.File) error) error {
			return callback(nil)
		}

		writer = &fakes.Writer{}
		nsWriter = &executor.NamespaceWriter{
			Namespace: ns,
			Writer:    writer,
		}
	})

	It("writes to the wrapped writer in the associated namespace", func() {
		ns.ExecuteStub = func(callback func(*os.File) error) error {
			Expect(writer.WriteCallCount()).To(Equal(0))
			err := callback(nil)
			Expect(writer.WriteCallCount()).To(Equal(1))
			return err
		}

		written, err := nsWriter.Write([]byte{})
		Expect(err).NotTo(HaveOccurred())
		Expect(written).To(Equal(0))

		Expect(ns.ExecuteCallCount()).To(Equal(1))
	})

	It("returns the bytes written by the wrapped writer", func() {
		writer.WriteReturns(100, nil)

		written, err := nsWriter.Write([]byte{})
		Expect(err).NotTo(HaveOccurred())
		Expect(written).To(Equal(100))
	})

	Context("when setting the namespae fails", func() {
		BeforeEach(func() {
			ns.ExecuteReturns(errors.New("peanuts"))
		})

		It("returns a meaningful error", func() {
			_, err := nsWriter.Write([]byte{})
			Expect(err).To(MatchError("namespace execute: peanuts"))
		})
	})

	Context("when the wrapped writer fails", func() {
		BeforeEach(func() {
			writer.WriteReturns(0, errors.New("applesauce"))
		})

		It("returns the original error", func() {
			_, err := nsWriter.Write([]byte{})
			Expect(err).To(Equal(errors.New("applesauce")))
		})
	})
})
