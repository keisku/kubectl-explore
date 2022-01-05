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

func NewOptions(streams genericclioptions.IOStreams) *Options {
	return &Options{
		IOStreams: streams,
	}
}

func NewCmd() *cobra.Command {
	f := cmdutil.NewFactory(genericclioptions.NewConfigFlags(true))
	o := NewOptions(genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	})

	cmd := &cobra.Command{
		Use: "kubectl explore RESOURCE [options]",
		Long: `This command finds fields associated with each supported API resource to explain.

Fields are identified via a simple JSONPath identifier:
	<type>.<fieldName>[.<fieldName>]
`,
		Short: "Find documentation for a resource to explain.",
		Example: `
# Explore pod fields.
kubectl explore pod

# Explore "pod.spec.containers" fields.
kubectl explore pod.spec.containers

# Explore the resource selected by interactive fuzzy-finder.
kubectl explore
`,
		Run: func(_ *cobra.Command, args []string) {
			cmdutil.CheckErr(o.Complete(f))
			cmdutil.CheckErr(o.Validate(args))
			cmdutil.CheckErr(o.Run(args))
		},
	}
	cmd.Flags().StringVar(&o.APIVersion, "api-version", o.APIVersion, "Get different explanations for particular API version (API group/version)")
	return cmd
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

func allAPIResources(discovery discovery.CachedDiscoveryInterface) ([]string, error) {
	resourceList, err := discovery.ServerPreferredResources()
	if err != nil {
		return nil, fmt.Errorf("get all API resources: %w", err)
	}
	var kinds []string
	for _, list := range resourceList {
		for _, r := range list.APIResources {
			kinds = append(kinds, r.Kind)
		}
	}
	return kinds, nil
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
	kind, err := o.getKind(args)
	if err != nil {
		return err
	}
	gvk, err := o.gvk(kind)
	if err != nil {
		return err
	}
	e, err := NewExplorer(inputFieldPath, kind, o.Schema, gvk)
	if err != nil {
		return err
	}
	return e.Explore(o.Out)
}

func (o *Options) getKind(args []string) (string, error) {
	var inResource string
	if len(args) == 1 {
		inResource = args[0]
	}
	var kind string
	if inResource == "" {
		rs, err := allAPIResources(o.Discovery)
		if err != nil {
			return "", err
		}
		idx, err := fuzzyfinder.Find(rs, func(i int) string {
			return strings.ToLower(rs[i])
		})
		if err != nil {
			return "", fmt.Errorf("fuzzy find the API resource: %w", err)
		}
		kind = strings.ToLower(rs[idx])
	} else {
		kind = strings.Split(inResource, ".")[0]
	}
	return kind, nil
}

func (o *Options) gvk(kind string) (schema.GroupVersionKind, error) {
	var gvr schema.GroupVersionResource
	var err error
	if len(o.APIVersion) == 0 {
		gvr, _, err = explain.SplitAndParseResourceRequestWithMatchingPrefix(kind, o.Mapper)
	} else {
		gvr, _, err = explain.SplitAndParseResourceRequest(kind, o.Mapper)
	}
	if err != nil {
		return schema.GroupVersionKind{}, fmt.Errorf("get the group version resource by %s %s: %w", o.APIVersion, kind, err)
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
