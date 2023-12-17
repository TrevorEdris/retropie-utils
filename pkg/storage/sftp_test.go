package storage_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TrevorEdris/retropie-utils/pkg/errors"
	"github.com/TrevorEdris/retropie-utils/pkg/storage"
)

var _ = Describe("Sftp", func() {
	It("is not implemented", func() {
		client, err := storage.NewSFTPStorage(storage.SFTPConfig{})
		Expect(err).NotTo(HaveOccurred())
		err = client.Store(context.TODO(), nil)
		Expect(err).To(MatchError(errors.NotImplementedError))
		err = client.StoreAll(context.TODO(), nil)
		Expect(err).To(MatchError(errors.NotImplementedError))
	})
})
