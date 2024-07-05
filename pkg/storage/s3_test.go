package storage_test

import (
	"context"
	"github.com/TrevorEdris/retropie-utils/pkg/errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TrevorEdris/retropie-utils/pkg/storage"
)

var _ = Describe("S3", func() {
	Context("storage is not enabled", func() {
		It("everything is a no-op", func() {
			client, err := storage.NewS3Storage(context.TODO(), storage.S3Config{
				Enabled: false,
			})
			Expect(err).NotTo(HaveOccurred())
			err = client.Store(context.TODO(), "", nil)
			Expect(err).NotTo(HaveOccurred())
			err = client.StoreAll(context.TODO(), "", nil)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("storage is enabled", func() {
		It("retrieve is not implemented", func() {
			client, err := storage.NewS3Storage(context.TODO(), storage.S3Config{
				Enabled: true,
			})
			Expect(err).NotTo(HaveOccurred())
			f, err := client.Retrieve(context.TODO(), storage.RetrieveFileRequest{})
			Expect(f).To(BeNil())
			Expect(err).To(MatchError(errors.NotImplementedError))
		})
	})
})
