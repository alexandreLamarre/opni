package management_test

import (
	"testing"

	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher/opni/pkg/test"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	k8sClient  client.Client
	restConfig *rest.Config
	scheme     *runtime.Scheme
	k8sManager ctrl.Manager
	nc         *nats.Conn
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Logging Plugin Suite")
}

var _ = BeforeSuite(func() {
	env := test.Environment{
		TestBin: "../../../../../../testbin/bin",
		CRDDirectoryPaths: []string{
			"../../../../../../config/crd/bases",
			"../../../../../../config/crd/opensearch",
			"../../../../../../test/resources",
		},
	}

	var err error
	restConfig, scheme, err = env.StartK8s()
	Expect(err).NotTo(HaveOccurred())

	nc, err = env.StartEmbeddedJetstream()
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(restConfig, client.Options{
		Scheme: scheme,
	})
	Expect(err).NotTo(HaveOccurred())

	k8sManager = env.StartManager(
		restConfig,
	)

	DeferCleanup(func() {
		By("tearing down the test environment")
		nc.Close()
		err := env.Stop()
		Expect(err).NotTo(HaveOccurred())
	})
})
