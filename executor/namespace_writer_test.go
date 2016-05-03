package executor_test

import (
	"errors"
	"os"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("NamespaceWriter", func() {
	var (
		nsWriter *executor.NamespaceWriter
		ns       *fakes.Namespace
		writer   *fakes.Writer
		logger   *lagertest.TestLogger
	)

	BeforeEach(func() {
		ns = &fakes.Namespace{}
		ns.MarshalJSONReturns([]byte(`{ "namespace": "my-namespace", "inode": "some-inode" }`), nil)
		ns.ExecuteStub = func(callback func(*os.File) error) error {
			return callback(nil)
		}

		logger = lagertest.NewTestLogger("test")

		writer = &fakes.Writer{}
		nsWriter = &executor.NamespaceWriter{
			Logger:    logger,
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

	It("logs the operation", func() {
		nsWriter.Write([]byte{})

		Expect(logger).To(gbytes.Say("write.write-called.*namespace.*some-inode"))
		Expect(logger).To(gbytes.Say("write.write-complete.*namespace.*some-inode"))
	})

	Context("when setting the namespae fails", func() {
		BeforeEach(func() {
			ns.ExecuteReturns(errors.New("peanuts"))
		})

		It("returns a meaningful error", func() {
			_, err := nsWriter.Write([]byte{})
			Expect(err).To(MatchError("namespace execute: peanuts"))
		})

		It("logs the error", func() {
			nsWriter.Write([]byte{})

			Expect(logger).To(gbytes.Say("write.namespace-execute-failed.*peanuts"))
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
