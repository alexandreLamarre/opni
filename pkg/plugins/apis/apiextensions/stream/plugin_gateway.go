package stream

import (
	"context"
	"errors"
	"io"
	"runtime"
	"strings"

	"github.com/hashicorp/go-plugin"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/kralicky/totem"
	"github.com/prometheus/client_golang/prometheus"
	streamv1 "github.com/rancher/opni/pkg/apis/stream/v1"
	"github.com/rancher/opni/pkg/auth/cluster"
	"github.com/rancher/opni/pkg/logger"
	"github.com/rancher/opni/pkg/plugins/apis/apiextensions"
	"go.opentelemetry.io/otel/attribute"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

type GatewayStreamApiExtensionPluginOptions struct {
	metricsConfig GatewayStreamMetricsConfig
}

type GatewayStreamApiExtensionPluginOption func(*GatewayStreamApiExtensionPluginOptions)

func (o *GatewayStreamApiExtensionPluginOptions) apply(opts ...GatewayStreamApiExtensionPluginOption) {
	for _, op := range opts {
		op(o)
	}
}

type GatewayStreamMetricsConfig struct {
	// Prometheus registerer
	Registerer prometheus.Registerer

	// A function called on each stream's Connect that returns a list of static
	// labels to attach to all metrics collected for that stream.
	LabelsForStream func(context.Context) []attribute.KeyValue
}

func WithMetrics(conf GatewayStreamMetricsConfig) GatewayStreamApiExtensionPluginOption {
	return func(o *GatewayStreamApiExtensionPluginOptions) {
		o.metricsConfig = conf
	}
}

func NewGatewayPlugin(p StreamAPIExtension, opts ...GatewayStreamApiExtensionPluginOption) plugin.Plugin {
	options := GatewayStreamApiExtensionPluginOptions{}
	options.apply(opts...)

	pc, _, _, ok := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)
	name := "unknown"
	if ok {
		fnName := fn.Name()
		name = fnName[strings.LastIndex(fnName, "plugins/")+len("plugins/") : strings.LastIndex(fnName, ".")]
	}

	ext := &gatewayStreamExtensionServerImpl{
		name:          name,
		logger:        logger.NewPluginLogger().Named(name).Named("stream"),
		metricsConfig: options.metricsConfig,
	}
	if p != nil {
		if options.metricsConfig.Registerer != nil {
			exporter, err := otelprometheus.New(
				otelprometheus.WithRegisterer(options.metricsConfig.Registerer),
				otelprometheus.WithoutScopeInfo(),
				otelprometheus.WithoutTargetInfo(),
			)
			if err != nil {
				panic("failed to create otel prometheus exporter")
			}
			ext.meterProvider = metric.NewMeterProvider(metric.WithReader(exporter),
				metric.WithResource(resource.NewSchemaless(attribute.Key("plugin").String(name))))
		}
		servers := p.StreamServers()
		for _, srv := range servers {
			descriptor, err := grpcreflect.LoadServiceDescriptor(srv.Desc)
			if err != nil {
				panic(err)
			}
			ext.servers = append(ext.servers, &richServer{
				Server:   srv,
				richDesc: descriptor,
			})
		}
		if clientHandler, ok := p.(StreamClientHandler); ok {
			ext.clientHandler = clientHandler
		}
		if clientDisconnectHandler, ok := p.(StreamClientDisconnectHandler); ok {
			ext.clientDisconnectHandler = clientDisconnectHandler
		}
	}
	return &streamApiExtensionPlugin[*gatewayStreamExtensionServerImpl]{
		extensionSrv: ext,
	}
}

type gatewayStreamExtensionServerImpl struct {
	streamv1.UnimplementedStreamServer
	apiextensions.UnsafeStreamAPIExtensionServer

	name                    string
	servers                 []*richServer
	clientHandler           StreamClientHandler
	clientDisconnectHandler StreamClientDisconnectHandler
	logger                  *zap.SugaredLogger
	metricsConfig           GatewayStreamMetricsConfig
	meterProvider           *metric.MeterProvider
}

// Implements streamv1.StreamServer
func (e *gatewayStreamExtensionServerImpl) Connect(stream streamv1.Stream_ConnectServer) error {
	id := cluster.StreamAuthorizedID(stream.Context())

	e.logger.With(
		zap.String("id", id),
	).Debug("stream connected")

	opts := []totem.ServerOption{totem.WithName("plugin_" + e.name)}

	if e.meterProvider != nil {
		var labels []attribute.KeyValue
		if e.metricsConfig.LabelsForStream != nil {
			labels = e.metricsConfig.LabelsForStream(stream.Context())
		}

		opts = append(opts, totem.WithMetrics(e.meterProvider, labels...))
	}

	ts, err := totem.NewServer(stream, opts...)

	if err != nil {
		e.logger.With(
			zap.Error(err),
		).Error("failed to create stream server")
		return err
	}
	for _, srv := range e.servers {
		ts.RegisterService(srv.Desc, srv.Impl)
	}

	_, errC := ts.Serve()

	e.logger.Debug("stream server started")

	err = <-errC
	if errors.Is(err, io.EOF) {
		e.logger.Debug("stream server exited")
	} else {
		e.logger.With(
			zap.Error(err),
		).Warn("stream server exited with error")
	}
	return err
}

// ConnectInternal implements apiextensions.StreamAPIExtensionServer
func (e *gatewayStreamExtensionServerImpl) ConnectInternal(stream apiextensions.StreamAPIExtension_ConnectInternalServer) error {
	if e.clientHandler == nil {
		stream.SendHeader(metadata.Pairs("accept-internal-stream", "false"))
		return nil
	}
	stream.SendHeader(metadata.Pairs("accept-internal-stream", "true"))

	e.logger.Debug("internal gateway stream connected")

	ts, err := totem.NewServer(stream)
	if err != nil {
		return err
	}
	cc, errC := ts.Serve()
	select {
	case err := <-errC:
		if errors.Is(err, io.EOF) {
			e.logger.Debug("stream disconnected")
		} else {
			e.logger.With(
				zap.Error(err),
			).Warn("stream disconnected with error")
		}
		return err
	default:
	}

	e.logger.Debug("calling client handler")
	go e.clientHandler.UseStreamClient(cc)

	defer func() {
		if e.clientDisconnectHandler != nil {
			e.logger.Debug("calling disconnect handler")
			e.clientDisconnectHandler.StreamDisconnected()
		}
	}()

	return <-errC
}
