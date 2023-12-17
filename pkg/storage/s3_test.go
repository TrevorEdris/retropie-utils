package storage_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TrevorEdris/retropie-utils/pkg/storage"
)

var _ = Describe("S3", func() {
	When("storage is not enabled", func() {
		It("is a no-op", func() {
			client, err := storage.NewS3Storage(storage.S3Config{
				Enabled: false,
			})
			Expect(err).NotTo(HaveOccurred())
			err = client.Store(context.TODO(), nil)
			Expect(err).NotTo(HaveOccurred())
			err = client.StoreAll(context.TODO(), nil)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
