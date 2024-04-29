package main

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/kube-openapi/pkg/util/proto"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/explain"
	"k8s.io/kubectl/pkg/util/openapi"
)

type Options struct {
	// User input
	apiVersion     string
	inputFieldPath string

	// After completion
	inputFieldPathRegex *regexp.Regexp
	gvks                []schema.GroupVersionKind

	// Dependencies
	genericclioptions.IOStreams
	mapper    meta.RESTMapper
	discovery discovery.CachedDiscoveryInterface
	schema    openapi.Resources
}

func NewCmd() *cobra.Command {
	o := NewOptions(genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	})

	cmd := &cobra.Command{
		Use:   "kubectl explore [resource|regex] [flags]",
		Short: "Fuzzy-find the field to explain from all API resources.",
		Example: `
# Fuzzy-find the field to explain from all API resources.
kubectl explore

# Fuzzy-find the field to explain from pod.
kubectl explore pod

# Fuzzy-find the field to explain from fields matching the regex.
kubectl explore pod.*node
kubectl explore spec.*containers
kubectl explore lifecycle
kubectl explore sts.*Account

# Fuzzy-find the field to explain from all API resources in the selected cluster.
kubectl explore --context=onecontext
`,
	}
	cmd.Flags().StringVar(&o.apiVersion, "api-version", o.apiVersion, "Get different explanations for particular API version (API group/version)")
	kubeConfigFlags := defaultConfigFlags().WithWarningPrinter(o.IOStreams)
	flags := cmd.PersistentFlags()
	kubeConfigFlags.AddFlags(flags)
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)
	matchVersionKubeConfigFlags.AddFlags(flags)
	f := cmdutil.NewFactory(matchVersionKubeConfigFlags)

	cmd.Run = func(_ *cobra.Command, args []string) {
		cmdutil.CheckErr(o.Complete(f, args))
		cmdutil.CheckErr(o.Run())
	}
	return cmd
}

func NewOptions(streams genericclioptions.IOStreams) *Options {
	return &Options{
		IOStreams: streams,
	}
}

func (o *Options) Complete(f cmdutil.Factory, args []string) error {
	var err error
	if len(args) == 0 {
		o.inputFieldPathRegex = regexp.MustCompile(".*")
	} else {
		o.inputFieldPathRegex, err = regexp.Compile(args[0])
		if err != nil {
			return err
		}
		o.inputFieldPath = args[0]
	}
	o.discovery, err = f.ToDiscoveryClient()
	if err != nil {
		return err
	}
	o.mapper, err = f.ToRESTMapper()
	if err != nil {
		return err
	}
	o.schema, err = f.OpenAPISchema()
	if err != nil {
		return err
	}
	if o.inputFieldPath == "" {
		gvk, err := o.findGVK()
		if err != nil {
			return err
		}
		o.gvks = []schema.GroupVersionKind{gvk}
	} else {
		var gvk schema.GroupVersionKind
		var err error
		var idx int
		for i := 1; i <= len(o.inputFieldPath); i++ {
			gvk, err = o.getGVK(o.inputFieldPath[:i])
			if err != nil {
				continue
			}
			idx = i
			break
		}
		if gvk.Empty() {
			o.gvks, err = o.listGVKs()
			if err != nil {
				return err
			}
		} else {
			// In this case, the input includes the resource name.

			// The left part of the input should be the resource name.
			// E.g., "hpa", "statefulset", "node", etc.
			left := o.inputFieldPath[:idx]

			// The right part of the input should be the field or regex.
			// E.g., "spec.template.spec.volumes.projected.sources.serviceAcc ", "spec.*containers", "spec.providerID", etc.
			right := strings.TrimLeft(o.inputFieldPath, left)

			o.inputFieldPathRegex, err = regexp.Compile(right)
			if err != nil {
				return err
			}
			o.gvks = []schema.GroupVersionKind{gvk}
		}
	}
	return nil
}

func (o *Options) Run() error {
	pathExplainers := make(map[string]explainer)
	var paths []string
	for _, gvk := range o.gvks {
		visitor := &schemaVisitor{
			pathSchema: make(map[string]proto.Schema),
			prevPath:   strings.ToLower(gvk.Kind),
			err:        nil,
		}
		s := o.schema.LookupResource(gvk)
		if s == nil {
			return fmt.Errorf("no schema found for %s", gvk)
		}
		s.Accept(visitor)
		if visitor.err != nil {
			return visitor.err
		}
		filteredPaths := visitor.listPaths(func(s string) bool {
			return o.inputFieldPathRegex.MatchString(s)
		})
		for _, p := range filteredPaths {
			pathExplainers[p] = explainer{
				schemaByGvk: s,
				gvk:         gvk,
				pathSchema:  visitor.pathSchema,
			}
			paths = append(paths, p)
		}
	}
	if len(paths) == 0 {
		return fmt.Errorf("no paths found for %q", o.inputFieldPath)
	}
	if len(paths) == 1 {
		return pathExplainers[paths[0]].explain(o.Out, paths[0])
	}
	sort.Strings(paths)
	idx, err := fuzzyfinder.Find(
		paths,
		func(i int) string { return paths[i] },
		fuzzyfinder.WithPreviewWindow(func(i, _, _ int) string {
			if i < 0 {
				return ""
			}
			var w bytes.Buffer
			if err := pathExplainers[paths[i]].explain(&w, paths[i]); err != nil {
				return fmt.Sprintf("preview is broken: %s", err)
			}
			return w.String()
		},
		))
	if err != nil {
		return err
	}
	return pathExplainers[paths[idx]].explain(o.Out, paths[idx])
}

func (o *Options) findGVK() (schema.GroupVersionKind, error) {
	gvks, err := o.listGVKs()
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	idx, err := fuzzyfinder.Find(gvks, func(i int) string {
		return strings.ToLower(gvks[i].Kind)
	}, fuzzyfinder.WithPreviewWindow(func(i, _, _ int) string {
		if i < 0 {
			return ""
		}
		return gvks[i].String()
	}))
	if err != nil {
		return schema.GroupVersionKind{}, fmt.Errorf("fuzzy find the API resource: %w", err)
	}
	return gvks[idx], nil
}

// listGVKs returns a list of GroupVersionKinds that is sorted in alphabetical order.
func (o *Options) listGVKs() ([]schema.GroupVersionKind, error) {
	resourceList, err := o.discovery.ServerPreferredResources()
	if err != nil {
		return nil, fmt.Errorf("get all API resources: %w", err)
	}
	if len(resourceList) == 0 {
		return nil, fmt.Errorf("API resources are not found")
	}
	var gvks []schema.GroupVersionKind
	for _, list := range resourceList {
		if len(list.APIResources) == 0 {
			continue
		}
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			continue
		}
		for _, r := range list.APIResources {
			gvks = append(gvks, schema.GroupVersionKind{
				Group:   gv.Group,
				Version: gv.Version,
				Kind:    r.Kind,
			})
		}
	}
	sort.SliceStable(gvks, func(i, j int) bool {
		return gvks[i].Kind < gvks[j].Kind
	})
	return gvks, nil
}

func (o *Options) getGVK(name string) (schema.GroupVersionKind, error) {
	var gvr schema.GroupVersionResource
	var err error
	if len(o.apiVersion) == 0 {
		gvr, _, err = explain.SplitAndParseResourceRequestWithMatchingPrefix(name, o.mapper)
	} else {
		gvr, _, err = explain.SplitAndParseResourceRequest(name, o.mapper)
	}
	if err != nil {
		return schema.GroupVersionKind{}, fmt.Errorf("get the group version resource by %s %s: %w", o.apiVersion, name, err)
	}

	gvk, err := o.mapper.KindFor(gvr)
	if err != nil {
		return schema.GroupVersionKind{}, fmt.Errorf("get a partial resource: %w", err)
	}
	if gvk.Empty() {
		gvk, err = o.mapper.KindFor(gvr.GroupResource().WithVersion(""))
		if err != nil {
			return schema.GroupVersionKind{}, fmt.Errorf("get a partial resource: %w", err)
		}
	}

	if len(o.apiVersion) != 0 {
		apiVer, err := schema.ParseGroupVersion(o.apiVersion)
		if err != nil {
			return schema.GroupVersionKind{}, fmt.Errorf("parse group version by %s: %w", o.apiVersion, err)
		}
		gvk = apiVer.WithKind(gvk.Kind)
	}
	return gvk, nil
}

// Copy from https://github.com/kubernetes/kubectl/blob/4f380d07c5e5bb41a037a72c4b35c7f828ba2d59/pkg/cmd/cmd.go#L95-L97
func defaultConfigFlags() *genericclioptions.ConfigFlags {
	return genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag().WithDiscoveryBurst(300).WithDiscoveryQPS(50.0)
}
