package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func main() {
	factory := cmdutil.NewFactory(genericclioptions.NewConfigFlags(true))

	cmd := &cobra.Command{
		Use: "kubectl-explore RESOURCE [options]",
		Long: `This command finds fields associated with each supported API resource to explain.

Fields are identified via a simple JSONPath identifier:
	<type>.<fieldName>[.<fieldName>]
`,
		Short: "This command explores the fields associated with each supported API resource.",
		Example: `
# Explore pod fields.
kubectl-explore pod

# Explore "pod.spec.containers" fields.
kubectl-explore pod.spec.containers

# Explore the resource selected by interactive fuzzy-finder.
kubectl-explore
`,
		SilenceUsage:          true,
		SilenceErrors:         true,
		DisableFlagsInUseLine: true,
		Version:               "0.1.0",
		RunE: func(_ *cobra.Command, args []string) error {
			var e *Explorer
			var err error
			if len(args) == 0 {
				e, err = NewExplorer(factory, "")
			} else {
				e, err = NewExplorer(factory, args[0])
			}
			if err != nil {
				return err
			}
			return e.Run(os.Stdout)
		},
	}
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
