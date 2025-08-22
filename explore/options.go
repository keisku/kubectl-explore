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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	openapiclient "k8s.io/client-go/openapi"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/kube-openapi/pkg/util/proto"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/openapi"
)

type Options struct {
	// User input
	apiVersion       string
	inputFieldPath   string
	disablePrintPath bool
	showBrackets     bool

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

# Fuzzy-find the field to explain from a specific api-version
kubectl explore --api-version=apps/v1
`,
	}
	cmd.Flags().StringVar(&o.apiVersion, "api-version", o.apiVersion, "Get different explanations for particular API version (API group/version)")
	cmd.Flags().BoolVar(&o.disablePrintPath, "disable-print-path", o.disablePrintPath, "Disable printing the path to explain")
	cmd.Flags().BoolVar(&o.showBrackets, "show-brackets", o.showBrackets, "Enable showing brackets for fields that are arrays")
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

	gvarMap, gvrs, err := o.discover()
	if err != nil {
		return err
	}

	if gvar, ok := gvarMap[o.inputFieldPath]; ok {
		o.inputFieldPathRegex = regexp.MustCompile(".*")
		o.gvrs = []schema.GroupVersionResource{gvar.GroupVersionResource}
		return nil
	}

	var gvar *groupVersionAPIResource
	var resourceIdx int
	for i := len(o.inputFieldPath); i > 0; i-- {
		var ok bool
		gvar, ok = gvarMap[o.inputFieldPath[:i]]
		if ok {
			resourceIdx = i
			break
		}
	}
	// If the inputFieldPath does not contain a valid resource name,
	// inputFiledPath is treated as a regex.
	if gvar == nil {
		o.gvrs = gvrs
		return nil
	}
	// Overwrite the regex if the inputFieldPath contains a valid resource name.
	_, ok := gvarMap[o.inputFieldPath[:resourceIdx]]
	if !ok {
		return fmt.Errorf("no resource found for %s", o.inputFieldPath)
	}
	var re string
	if strings.HasPrefix(o.inputFieldPath, gvar.Resource) {
		re = strings.TrimPrefix(o.inputFieldPath, gvar.Resource)
	} else if strings.HasPrefix(o.inputFieldPath, gvar.Kind) {
		re = strings.TrimPrefix(o.inputFieldPath, gvar.Kind)
	} else if strings.HasPrefix(o.inputFieldPath, gvar.SingularName) {
		re = strings.TrimPrefix(o.inputFieldPath, gvar.SingularName)
	} else {
		for _, shortName := range gvar.ShortNames {
			if strings.HasPrefix(o.inputFieldPath, shortName) {
				re = strings.TrimPrefix(o.inputFieldPath, shortName)
			}
		}
	}
	if re == "" {
		return fmt.Errorf("cannot find resource name in %s", o.inputFieldPath)
	}
	o.inputFieldPathRegex, err = regexp.Compile(re)
	if err != nil {
		return err
	}
	o.gvrs = []schema.GroupVersionResource{gvar.GroupVersionResource}

	return nil
}

func (o *Options) Run() error {
	pathExplainers := make(map[path]explainer)
	var paths []path
	for _, gvr := range o.gvrs {
		visitor := &schemaVisitor{
			pathSchema: make(map[path]proto.Schema),
			prevPath: path{
				original:     strings.ToLower(gvr.Resource),
				withBrackets: strings.ToLower(gvr.Resource),
			},
			err: nil,
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
		filteredPaths := visitor.listPaths(func(s path) bool {
			return o.inputFieldPathRegex.MatchString(s.original)
		})
		for _, p := range filteredPaths {
			pathExplainers[p] = explainer{
				gvr:                 gvr,
				openAPIV3Client:     o.cachedOpenAPIV3Client,
				enablePrintPath:     !o.disablePrintPath,
				enablePrintBrackets: o.showBrackets,
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
	sort.SliceStable(paths, func(i, j int) bool {
		return paths[i].original < paths[j].original
	})
	idx, err := fuzzyfinder.Find(
		paths,
		func(i int) string { return paths[i].original },
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

// serverPreferredResources fetches and filters the server's preferred resources.
// If o.apiVersion is set, it only returns resources matching that API version.
func (o *Options) serverPreferredResources() ([]*metav1.APIResourceList, error) {
	lists, err := o.discovery.ServerPreferredResources()
	if err != nil {
		return nil, err
	}
	if o.apiVersion == "" {
		return lists, nil
	}

	var filteredLists []*metav1.APIResourceList
	for _, list := range lists {
		if list.GroupVersion == o.apiVersion {
			filteredLists = append(filteredLists, list)
		}
	}
	if len(filteredLists) == 0 {
		return nil, fmt.Errorf("no resources found for API version %q", o.apiVersion)
	}
	return filteredLists, nil
}

func (o *Options) listGVRs() ([]schema.GroupVersionResource, error) {
	lists, err := o.serverPreferredResources()
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
	if len(gvrs) == 0 {
		if o.apiVersion != "" {
			return schema.GroupVersionResource{}, fmt.Errorf("no resources found for API version: %s", o.apiVersion)
		}
		return schema.GroupVersionResource{}, fmt.Errorf("no resources found")
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

type groupVersionAPIResource struct {
	schema.GroupVersionResource
	metav1.APIResource
}

func (o *Options) discover() (map[string]*groupVersionAPIResource, []schema.GroupVersionResource, error) {
	lists, err := o.serverPreferredResources()
	if err != nil {
		return nil, nil, err
	}
	var gvrs []schema.GroupVersionResource
	m := make(map[string]*groupVersionAPIResource)
	for _, list := range lists {
		if len(list.APIResources) == 0 {
			continue
		}
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			continue
		}
		for _, resource := range list.APIResources {
			gvr := gv.WithResource(resource.Name)
			gvrs = append(gvrs, gvr)
			r := groupVersionAPIResource{
				GroupVersionResource: gvr,
				APIResource:          resource,
			}
			m[resource.Name] = &r
			m[resource.Kind] = &r
			m[resource.SingularName] = &r
			for _, shortName := range resource.ShortNames {
				m[shortName] = &r
			}
		}
	}
	sort.SliceStable(gvrs, func(i, j int) bool {
		return gvrs[i].String() < gvrs[j].String()
	})
	return m, gvrs, nil
}
