package storage_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TrevorEdris/retropie-utils/pkg/errors"
	"github.com/TrevorEdris/retropie-utils/pkg/storage"
)

var _ = Describe("Gdrive", func() {
	It("is not implemented", func() {
		client, err := storage.NewGoogleDriveStorage(storage.GDriveConfig{})
		Expect(err).NotTo(HaveOccurred())
		err = client.Store(context.TODO(), nil)
		Expect(err).To(MatchError(errors.NotImplementedError))
		err = client.StoreAll(context.TODO(), nil)
		Expect(err).To(MatchError(errors.NotImplementedError))
	})
})
