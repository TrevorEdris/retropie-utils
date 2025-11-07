package integration

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSyncerIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Syncer Integration Suite")
}
