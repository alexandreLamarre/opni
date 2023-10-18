package agent

import (
	capabilityv1 "github.com/rancher/opni/pkg/apis/capability/v1"
	controlv1 "github.com/rancher/opni/pkg/apis/control/v1"
	"github.com/rancher/opni/pkg/clients"
	"github.com/rancher/opni/pkg/logger"
	streamext "github.com/rancher/opni/pkg/plugins/apis/apiextensions/stream"
	"github.com/rancher/opni/plugins/metrics/apis/node"
	"github.com/rancher/opni/plugins/metrics/apis/remoteread"
	"github.com/rancher/opni/plugins/metrics/apis/remotewrite"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (p *Plugin) StreamServers() []streamext.Server {
	return []streamext.Server{
		{
			Desc: &capabilityv1.Node_ServiceDesc,
			Impl: p.node,
		},
		{
			Desc: &remoteread.RemoteReadAgent_ServiceDesc,
			Impl: p.node,
		},
	}
}

func (p *Plugin) UseStreamClient(cc grpc.ClientConnInterface) {
	nodeClient := node.NewNodeMetricsCapabilityClient(cc)
	healthListenerClient := controlv1.NewHealthListenerClient(cc)
	identityClient := controlv1.NewIdentityClient(cc)

	p.configureLoggers(identityClient)

	p.httpServer.SetRemoteWriteClient(clients.NewLocker(cc, remotewrite.NewRemoteWriteClient))
	p.ruleStreamer.SetRemoteWriteClient(remotewrite.NewRemoteWriteClient(cc))
	p.node.SetRemoteWriter(clients.NewLocker(cc, remotewrite.NewRemoteWriteClient))

	p.node.SetClients(
		nodeClient,
		identityClient,
		healthListenerClient,
	)

}

func (p *Plugin) configureLoggers(identityClient controlv1.IdentityClient) {
	id, err := identityClient.Whoami(p.ctx, &emptypb.Empty{})
	if err != nil {
		p.logger.Error("error fetching node id", logger.Err(err))
	}
	logger.InitPluginWriter(id.GetId())
}
