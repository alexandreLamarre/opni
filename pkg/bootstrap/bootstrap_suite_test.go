package bootstrap_test

import (
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	_ "github.com/rancher/opni/pkg/test/setup"
	_ "github.com/rancher/opni/plugins/example/test" // Required for incluster_test.go
)

func TestBootstrap(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bootstrap Suite")
}

var ctrl *gomock.Controller

var _ = BeforeSuite(func() {
	ctrl = gomock.NewController(GinkgoT())
})
