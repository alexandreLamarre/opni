//go:build !minimal

package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	managementv1 "github.com/rancher/opni/pkg/apis/management/v1"
	"github.com/rancher/opni/pkg/clients"
	"github.com/rancher/opni/pkg/config"
	"github.com/rancher/opni/pkg/config/v1beta1"
	cliutil "github.com/rancher/opni/pkg/opni/util"
	"github.com/spf13/cobra"
	"go.etcd.io/etcd/etcdctl/v3/ctlv3"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"sigs.k8s.io/yaml"
)

func BuildDebugCmd() *cobra.Command {
	debugCmd := &cobra.Command{
		Use:   "debug",
		Short: "Various debugging commands",
	}
	debugCmd.AddCommand(BuildDebugReloadCmd())
	debugCmd.AddCommand(BuildDebugGetConfigCmd())
	debugCmd.AddCommand(BuildDebugEtcdctlCmd())
	debugCmd.AddCommand(BuildDebugDashboardSettingsCmd())
	ConfigureManagementCommand(debugCmd)
	return debugCmd
}

func BuildDebugGetConfigCmd() *cobra.Command {
	var format string
	debugGetConfigCmd := &cobra.Command{
		Use:   "get-config",
		Short: "Print the current gateway config to stdout",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := mgmtClient.GetConfig(cmd.Context(), &emptypb.Empty{})
			if err != nil {
				return err
			}
			if len(config.Documents) == 0 {
				lg.Warn("server returned no configuration")
				return nil
			}
			for _, doc := range config.Documents {
				switch format {
				case "json":
					buf := new(bytes.Buffer)
					err := json.Indent(buf, doc.Json, "", "  ")
					if err != nil {
						return err
					}
					fmt.Println(buf.String())
				case "yaml":
					data, err := yaml.JSONToYAML(doc.Json)
					if err != nil {
						return err
					}
					fmt.Println(string(data))
				default:
					return fmt.Errorf("unknown format: %s", format)
				}
			}
			return nil
		},
	}
	debugGetConfigCmd.Flags().StringVarP(&format, "format", "f", "yaml", "Output format (yaml|json)")
	return debugGetConfigCmd
}

func BuildDebugReloadCmd() *cobra.Command {
	debugReloadCmd := &cobra.Command{
		Use:   "reload",
		Short: "Signal the gateway to reload and apply any config updates",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := mgmtClient.GetConfig(cmd.Context(), &emptypb.Empty{})
			if err != nil {
				return err
			}
			docsNoSchema := []*managementv1.ConfigDocument{}
			for _, doc := range config.Documents {
				docsNoSchema = append(docsNoSchema, &managementv1.ConfigDocument{
					Json: doc.Json,
				})
			}
			_, err = mgmtClient.UpdateConfig(cmd.Context(), &managementv1.UpdateConfigRequest{
				Documents: docsNoSchema,
			})
			return err
		},
	}
	return debugReloadCmd
}

func BuildDebugEtcdctlCmd() *cobra.Command {
	debugEtcdctlCmd := &cobra.Command{
		Use:                "etcdctl",
		Short:              "embedded auto-configured etcdctl",
		Long:               "To specify a gateway address, use the OPNI_ADDRESS environment variable.",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			addr := os.Getenv("OPNI_ADDRESS")
			var gatewayConfig *v1beta1.GatewayConfig
			if addr != "" {
				c, err := clients.NewManagementClient(cmd.Context(), clients.WithAddress(addr))
				if err == nil {
					mgmtClient = c
				} else {
					lg.Warnf("failed to create management client: %v", err)
				}
			}
			conf, err := mgmtClient.GetConfig(cmd.Context(), &emptypb.Empty{})
			if err != nil {
				return err
			}
			objects, err := config.LoadObjects(conf.YAMLDocuments())
			if err != nil {
				return err
			}
			objects.Visit(
				func(config *v1beta1.GatewayConfig) {
					if gatewayConfig == nil {
						gatewayConfig = config
					}
				},
			)
			if gatewayConfig == nil {
				return fmt.Errorf("no gateway config found")
			}
			if gatewayConfig.Spec.Storage.Type != v1beta1.StorageTypeEtcd {
				return fmt.Errorf("storage type is not etcd")
			}
			endpoints := gatewayConfig.Spec.Storage.Etcd.Endpoints
			argv := []string{"etcdctl", fmt.Sprintf("--endpoints=%s", strings.Join(endpoints, ","))}
			if gatewayConfig.Spec.Storage.Etcd.Certs != nil {
				cert := gatewayConfig.Spec.Storage.Etcd.Certs.ClientCert
				key := gatewayConfig.Spec.Storage.Etcd.Certs.ClientKey
				ca := gatewayConfig.Spec.Storage.Etcd.Certs.ServerCA
				argv = append(argv,
					fmt.Sprintf("--cacert=%s", ca),
					fmt.Sprintf("--cert=%s", cert),
					fmt.Sprintf("--key=%s", key),
				)
			}

			os.Args = append(argv, args...)
			return ctlv3.Start()
		},
	}
	return debugEtcdctlCmd
}

func BuildDebugDashboardSettingsCmd() *cobra.Command {
	debugDashboardSettingsCmd := &cobra.Command{
		Use:   "dashboard-settings",
		Short: "Manage dashboard settings",
	}
	debugDashboardSettingsCmd.AddCommand(BuildDebugDashboardSettingsGetCmd())
	debugDashboardSettingsCmd.AddCommand(BuildDebugDashboardSettingsUpdateCmd())
	return debugDashboardSettingsCmd
}

func BuildDebugDashboardSettingsGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get the dashboard settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			settings, err := mgmtClient.GetDashboardSettings(cmd.Context(), &emptypb.Empty{})
			if err != nil {
				return err
			}
			data, err := protojson.MarshalOptions{
				Multiline:       true,
				EmitUnpopulated: true,
			}.Marshal(settings)
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		},
	}
	return cmd
}

func BuildDebugDashboardSettingsUpdateCmd() *cobra.Command {
	var reset bool
	var defaultImageRepository string
	var defaultTokenTtl string
	var defaultTokenLabels []string
	var userSettings []string
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update dashboard settings",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var settings *managementv1.DashboardSettings
			if reset {
				settings = &managementv1.DashboardSettings{}
			} else {
				var err error
				settings, err = mgmtClient.GetDashboardSettings(cmd.Context(), &emptypb.Empty{})
				if err != nil {
					return err
				}
			}

			if settings.Global == nil {
				settings.Global = &managementv1.DashboardGlobalSettings{}
			}
			if defaultImageRepository != "" {
				settings.Global.DefaultImageRepository = defaultImageRepository
			}
			if defaultTokenTtl != "" {
				d, err := time.ParseDuration(defaultTokenTtl)
				if err != nil {
					return err
				}
				settings.Global.DefaultTokenTtl = durationpb.New(d)
			}
			if defaultTokenLabels != nil {
				kv, err := cliutil.ParseKeyValuePairs(defaultTokenLabels)
				if err != nil {
					return err
				}
				settings.Global.DefaultTokenLabels = kv
			}

			if userSettings != nil {
				kv, err := cliutil.ParseKeyValuePairs(userSettings)
				if err != nil {
					return err
				}
				settings.User = kv
			}

			_, err := mgmtClient.UpdateDashboardSettings(cmd.Context(), settings)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&defaultImageRepository, "global.default-image-repository", "", "Default image repository for helm command templates")
	cmd.Flags().StringVar(&defaultTokenTtl, "global.default-token-ttl", "", "Default token TTL")
	cmd.Flags().StringSliceVar(&defaultTokenLabels, "global.default-token-labels", nil, "Default token labels (key-value pairs)")
	cmd.Flags().StringSliceVar(&userSettings, "user", []string{}, "User settings (key-value pairs)")
	cmd.Flags().BoolVar(&reset, "reset", false, "Reset settings to default values. If other flags are specified, they will be applied on top of the default values.")

	return cmd
}

func init() {
	AddCommandsToGroup(Debug, BuildDebugCmd())
}
