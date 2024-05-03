package explore

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
	openapiclient "k8s.io/client-go/openapi"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/kube-openapi/pkg/util/proto"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/explain"
	"k8s.io/kubectl/pkg/util/openapi"
)

type Options struct {
	// User input
	apiVersion       string
	inputFieldPath   string
	disablePrintPath bool

	// After completion
	inputFieldPathRegex *regexp.Regexp
	gvrs                []schema.GroupVersionResource

	// Dependencies
	genericclioptions.IOStreams
	discovery             discovery.CachedDiscoveryInterface
	mapper                meta.RESTMapper
	schema                openapi.Resources
	cachedOpenAPIV3Client openapiclient.Client
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
	cmd.Flags().BoolVar(&o.disablePrintPath, "disable-print-path", o.disablePrintPath, "Disable printing the path to explain")
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

// Copy from https://github.com/kubernetes/kubectl/blob/4f380d07c5e5bb41a037a72c4b35c7f828ba2d59/pkg/cmd/cmd.go#L95-L97
func defaultConfigFlags() *genericclioptions.ConfigFlags {
	return genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag().WithDiscoveryBurst(300).WithDiscoveryQPS(50.0)
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
	if c, err := f.OpenAPIV3Client(); err == nil {
		o.cachedOpenAPIV3Client, err = newCachedOpenAPIClient(c)
		if err != nil {
			return err
		}
	} else {
		return err
	}

	if o.inputFieldPath == "" {
		g, err := o.findGVR()
		if err != nil {
			return err
		}
		o.gvrs = []schema.GroupVersionResource{g}
		return nil
	}

	var gotGVR schema.GroupVersionResource
	var idx int
	// Find the first valid resource name in the inputFieldPath.
	for i := 1; i <= len(o.inputFieldPath); i++ {
		gotGVR, err = GetGVR(o, o.inputFieldPath[:i])
		if err != nil {
			continue
		}
		idx = i
		break
	}
	// If the inputFieldPath does not contain a valid resource name,
	// inputFiledPath is treated as a regex directly.
	if gotGVR.Empty() {
		o.gvrs, err = o.listGVRs()
		if err != nil {
			return err
		}
		return nil
	}
	// Overwrite the regex if the inputFieldPath contains a valid resource name.
	var re string
	if strings.HasPrefix(o.inputFieldPath, gotGVR.Resource) {
		// E.g., "nodes.*spec" -> ".*spec"
		re = strings.TrimLeft(o.inputFieldPath, gotGVR.Resource)
	} else if strings.HasPrefix(o.inputFieldPath, singularResource(gotGVR.Resource)) {
		// E.g., "node.*spec" -> ".*spec"
		re = strings.TrimLeft(o.inputFieldPath, singularResource(gotGVR.Resource))
	} else {
		// E.g., "no.*spec" -> ".*spec"
		left := o.inputFieldPath[:idx]
		re = strings.TrimLeft(o.inputFieldPath, left)
	}
	o.inputFieldPathRegex, err = regexp.Compile(re)
	if err != nil {
		return err
	}
	o.gvrs = []schema.GroupVersionResource{gotGVR}

	return nil
}

func (o *Options) Run() error {
	pathExplainers := make(map[string]explainer)
	var paths []string
	for _, gvr := range o.gvrs {
		visitor := &schemaVisitor{
			pathSchema: make(map[string]proto.Schema),
			prevPath:   strings.ToLower(gvr.Resource),
			err:        nil,
		}
		gvk, err := o.mapper.KindFor(gvr)
		if err != nil {
			return fmt.Errorf("get the group version kind: %w", err)
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
				gvr:             gvr,
				openAPIV3Client: o.cachedOpenAPIV3Client,
				enablePrintPath: !o.disablePrintPath,
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

func singularResource(resource string) string {
	if strings.HasSuffix(resource, "s") {
		return resource[:len(resource)-1]
	}
	return resource
}

func (o *Options) listGVRs() ([]schema.GroupVersionResource, error) {
	lists, err := o.discovery.ServerPreferredResources()
	if err != nil {
		return nil, err
	}
	var gvrs []schema.GroupVersionResource
	for _, list := range lists {
		if len(list.APIResources) == 0 {
			continue
		}
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			continue
		}
		for _, resource := range list.APIResources {
			gvrs = append(gvrs, gv.WithResource(resource.Name))
		}
	}
	sort.SliceStable(gvrs, func(i, j int) bool {
		return gvrs[i].String() < gvrs[j].String()
	})
	return gvrs, nil
}

func (o *Options) findGVR() (schema.GroupVersionResource, error) {
	gvrs, err := o.listGVRs()
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	idx, err := fuzzyfinder.Find(gvrs, func(i int) string {
		return gvrs[i].Resource
	}, fuzzyfinder.WithPreviewWindow(func(i, _, _ int) string {
		if i < 0 {
			return ""
		}
		return gvrs[i].String()
	}))
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("fuzzy find the API resource: %w", err)
	}
	return gvrs[idx], nil
}

// TODO: Find a way to mock meta.RESTMapper to avoid defining it as a variable.
var GetGVR = func(o *Options, name string) (schema.GroupVersionResource, error) {
	return o.getGVR(name)
}

func (o *Options) getGVR(name string) (schema.GroupVersionResource, error) {
	var ret schema.GroupVersionResource
	var err error
	if len(o.apiVersion) == 0 {
		ret, _, err = explain.SplitAndParseResourceRequestWithMatchingPrefix(name, o.mapper)
	} else {
		ret, _, err = explain.SplitAndParseResourceRequest(name, o.mapper)
	}
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("get the group version resource by %s %s: %w", o.apiVersion, name, err)
	}
	return ret, nil
}
