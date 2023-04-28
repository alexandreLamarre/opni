package cortex_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	_ "github.com/rancher/opni/pkg/test/setup"
)

func TestCortex(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cortex Suite")
}
