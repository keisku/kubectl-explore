package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/explain"
	"k8s.io/kubectl/pkg/util/openapi"
)

type Options struct {
	genericclioptions.IOStreams

	APIVersion string

	Mapper    meta.RESTMapper
	Discovery discovery.CachedDiscoveryInterface
	Schema    openapi.Resources
}

func NewCmd() *cobra.Command {
	o := NewOptions(genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	})

	cmd := &cobra.Command{
		Use: "kubectl explore RESOURCE [options]",
		Long: `This command fuzzy-find and explain fields associated with each supported API resource.

Fields are identified via a simple JSONPath identifier:
	<type>.<fieldName>[.<fieldName>]
`,
		Short: "Find documentation for a resource to explain.",
		Example: `
# Explore the resource selected by fuzzy-finder.
kubectl explore

# Explore "pod" fields.
kubectl explore pod

# Explore "pod.spec.containers" fields.
kubectl explore pod.spec.containers

# Explore fields in the selected context.
kubectl explore --context=onecontext
`,
	}
	cmd.Flags().StringVar(&o.APIVersion, "api-version", o.APIVersion, "Get different explanations for particular API version (API group/version)")
	// Use default flags from
	// https://github.com/kubernetes/kubectl/blob/e4426be7778f13d7b8122eee72132ddd089d1397/pkg/cmd/cmd.go#L297
	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag().WithDiscoveryBurst(300).WithDiscoveryQPS(50.0)
	flags := cmd.PersistentFlags()
	kubeConfigFlags.AddFlags(flags)
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)
	matchVersionKubeConfigFlags.AddFlags(flags)
	f := cmdutil.NewFactory(matchVersionKubeConfigFlags)

	cmd.Run = func(_ *cobra.Command, args []string) {
		cmdutil.CheckErr(o.Complete(f))
		cmdutil.CheckErr(o.Validate(args))
		cmdutil.CheckErr(o.Run(args))
	}
	return cmd
}

func NewOptions(streams genericclioptions.IOStreams) *Options {
	return &Options{
		IOStreams: streams,
	}
}

func (o *Options) Complete(f cmdutil.Factory) error {
	var err error
	o.Discovery, err = f.ToDiscoveryClient()
	if err != nil {
		return err
	}
	o.Mapper, err = f.ToRESTMapper()
	if err != nil {
		return err
	}
	o.Schema, err = f.OpenAPISchema()
	if err != nil {
		return err
	}
	return nil
}

func (o *Options) Validate(args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("We accept only this format: explore RESOURCE")
	}

	return nil
}

func (o *Options) Run(args []string) error {
	var inputFieldPath string
	if 0 < len(args) {
		inputFieldPath = args[0]
	}
	name, err := o.getResourceName(args)
	if err != nil {
		return err
	}
	gvk, err := o.gvk(name)
	if err != nil {
		return err
	}
	e, err := NewExplorer(inputFieldPath, name, o.Schema, gvk)
	if err != nil {
		return err
	}
	return e.Explore(o.Out)
}

func (o *Options) getResourceName(args []string) (string, error) {
	var inResource string
	if len(args) == 1 {
		inResource = args[0]
	}
	var name string
	if inResource == "" {
		names, err := allAPIResourceNames(o.Discovery)
		if err != nil {
			return "", err
		}
		idx, err := fuzzyfinder.Find(names, func(i int) string {
			return strings.ToLower(names[i])
		})
		if err != nil {
			return "", fmt.Errorf("fuzzy find the API resource: %w", err)
		}
		name = strings.ToLower(names[idx])
	} else {
		name = strings.Split(inResource, ".")[0]
	}
	return name, nil
}

func allAPIResourceNames(discovery discovery.CachedDiscoveryInterface) ([]string, error) {
	resourceList, err := discovery.ServerPreferredResources()
	if err != nil {
		return nil, fmt.Errorf("get all API resources: %w", err)
	}
	var names []string
	for _, list := range resourceList {
		for _, r := range list.APIResources {
			names = append(names, r.Name)
		}
	}
	return names, nil
}

func (o *Options) gvk(name string) (schema.GroupVersionKind, error) {
	var gvr schema.GroupVersionResource
	var err error
	if len(o.APIVersion) == 0 {
		gvr, _, err = explain.SplitAndParseResourceRequestWithMatchingPrefix(name, o.Mapper)
	} else {
		gvr, _, err = explain.SplitAndParseResourceRequest(name, o.Mapper)
	}
	if err != nil {
		return schema.GroupVersionKind{}, fmt.Errorf("get the group version resource by %s %s: %w", o.APIVersion, name, err)
	}

	gvk, err := o.Mapper.KindFor(gvr)
	if err != nil {
		return schema.GroupVersionKind{}, fmt.Errorf("get a partial resource: %w", err)
	}
	if gvk.Empty() {
		gvk, err = o.Mapper.KindFor(gvr.GroupResource().WithVersion(""))
		if err != nil {
			return schema.GroupVersionKind{}, fmt.Errorf("get a partial resource: %w", err)
		}
	}

	if len(o.APIVersion) != 0 {
		apiVer, err := schema.ParseGroupVersion(o.APIVersion)
		if err != nil {
			return schema.GroupVersionKind{}, fmt.Errorf("parse group version by %s: %w", o.APIVersion, err)
		}
		gvk = apiVer.WithKind(gvk.Kind)
	}
	return gvk, nil
}