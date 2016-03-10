package locks_test

import (
	"github.com/cloudfoundry-incubator/ducati-daemon/locks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NamedMutex", func() {
	Describe("Lock / unlock lifecycle", func() {
		Context("when lock is called twice", func() {
			Context("when the lock name is the same", func() {
				It("blocks on the second call to Lock until Unlock is called", func() {
					g := &locks.NamedMutex{}

					g.Lock("some key")

					finished := make(chan bool)
					go func() {
						g.Lock("some key")
						finished <- true
					}()

					Consistently(finished).ShouldNot(Receive())

					g.Unlock("some key")

					Eventually(finished).Should(Receive())
				})
			})

			Context("when the second lock is using a different name", func() {
				It("does not block", func() {
					g := &locks.NamedMutex{}

					g.Lock("some key")

					finished := make(chan bool)
					go func() {
						g.Lock("some other key")
						finished <- true
					}()

					Eventually(finished).Should(Receive())
				})
			})
		})
	})
})
