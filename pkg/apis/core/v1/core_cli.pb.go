// Code generated by cli_gen.go DO NOT EDIT.
// source: github.com/rancher/opni/pkg/apis/core/v1/core.proto

package v1

import (
	context "context"
	cli "github.com/rancher/opni/internal/codegen/cli"
	flagutil "github.com/rancher/opni/pkg/util/flagutil"
	cobra "github.com/spf13/cobra"
	pflag "github.com/spf13/pflag"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	strings "strings"
)

type contextKey_Pinger_type struct{}

var contextKey_Pinger contextKey_Pinger_type

func ContextWithPingerClient(ctx context.Context, client PingerClient) context.Context {
	return context.WithValue(ctx, contextKey_Pinger, client)
}

func PingerClientFromContext(ctx context.Context) (PingerClient, bool) {
	client, ok := ctx.Value(contextKey_Pinger).(PingerClient)
	return client, ok
}

var extraCmds_Pinger []*cobra.Command

func addExtraPingerCmd(custom *cobra.Command) {
	extraCmds_Pinger = append(extraCmds_Pinger, custom)
}

func BuildPingerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "pinger",
		Short:             ``,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
	}

	cmd.AddCommand(BuildPingerPingCmd())
	cli.AddOutputFlag(cmd)
	return cmd
}

func BuildPingerPingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "ping",
		Short:             "",
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, ok := PingerClientFromContext(cmd.Context())
			if !ok {
				cmd.PrintErrln("failed to get client from context")
				return nil
			}
			response, err := client.Ping(cmd.Context(), &emptypb.Empty{})
			if err != nil {
				return err
			}
			cli.RenderOutput(cmd, response)
			return nil
		},
	}
	return cmd
}

func (in *Revision) FlagSet(prefix ...string) *pflag.FlagSet {
	fs := pflag.NewFlagSet("Revision", pflag.ExitOnError)
	fs.SortFlags = true
	fs.Var(flagutil.IntPtrValue(nil, &in.Revision), strings.Join(append(prefix, "revision"), "."), "A numerical revision uniquely identifying a specific version of the resource. Larger values are newer, but this should otherwise be treated as opaque.")
	return fs
}
