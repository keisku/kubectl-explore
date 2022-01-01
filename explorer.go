package main

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/ktr0731/go-fuzzyfinder"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/kube-openapi/pkg/util/proto"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/explain"
	"k8s.io/kubectl/pkg/util/openapi"
)

var _ proto.SchemaVisitor = (*explorer)(nil)

type explorer struct {
	discovery      discovery.CachedDiscoveryInterface
	restMapper     meta.RESTMapper
	openAPISchema  openapi.Resources
	err            error
	inputFieldPath string
	kind           string
	rootSchema     proto.Schema
	prevPath       string
	pathSchema     map[string]proto.Schema
}

func newExplorer(factory cmdutil.Factory, fieldPath string) (*explorer, error) {
	discovery, err := factory.ToDiscoveryClient()
	if err != nil {
		return nil, fmt.Errorf("get the discovery client from the factory: %w", err)
	}
	restMapper, err := factory.ToRESTMapper()
	if err != nil {
		return nil, fmt.Errorf("get the rest mapper from the factory: %w", err)
	}
	openAPISchema, err := factory.OpenAPISchema()
	if err != nil {
		return nil, fmt.Errorf("get the Open API schema from the factory: %w", err)
	}
	kind := fieldPath
	if 0 < strings.Count(fieldPath, ".") {
		kind = fieldPath[:strings.Index(fieldPath, ".")]
	}
	if kind == "" {
		kinds, err := allKinds(discovery)
		if err != nil {
			return nil, err
		}
		// block until user selects the kind.
		idx, err := fuzzyfinder.Find(kinds, func(i int) string {
			return strings.ToLower(kinds[i])
		})
		if err != nil {
			return nil, fmt.Errorf("fuzzy find kind: %w", err)
		}
		kind = strings.ToLower(kinds[idx])
	}
	return &explorer{
		discovery:      discovery,
		restMapper:     restMapper,
		openAPISchema:  openAPISchema,
		inputFieldPath: fieldPath,
		kind:           kind,
		prevPath:       kind,
		pathSchema:     make(map[string]proto.Schema),
	}, nil
}

func allKinds(discovery discovery.CachedDiscoveryInterface) ([]string, error) {
	resourceList, err := discovery.ServerPreferredResources()
	if err != nil {
		return nil, fmt.Errorf("get the resources: %w", err)
	}
	var kinds []string
	for _, list := range resourceList {
		for _, r := range list.APIResources {
			kinds = append(kinds, r.Kind)
		}
	}
	return kinds, nil
}

func (e *explorer) run(w io.Writer) error {
	gvk, err := e.gvk()
	if err != nil {
		return err
	}
	e.explore(gvk)
	if e.err != nil {
		return e.err
	}

	path, err := findPath(e.paths())
	if err != nil {
		return fmt.Errorf("find the path: %w", err)
	}

	// This is the case that path specifies the top-level field,
	// for example, "pod.spec", "pod.metadata"
	if strings.Count(path, ".") == 1 {
		fieldPath := []string{path[strings.LastIndex(path, ".")+1:]}
		return explain.PrintModelDescription(fieldPath, w, e.rootSchema, gvk, false)
	}

	// get the parent schema to explain.
	// e.g. "pod.spec.containers.env" -> "pod.spec.containers"
	parent, ok := e.pathSchema[path[:strings.LastIndex(path, ".")]]
	if !ok {
		return fmt.Errorf("%s is not found", path)
	}

	// get the key from the path.
	// e.g. "pod.spec.containers.env" -> "env"
	fieldPath := []string{path[strings.LastIndex(path, ".")+1:]}
	if err := explain.PrintModelDescription(fieldPath, w, parent, gvk, false); err != nil {
		return fmt.Errorf(`explain "%s": %w`, path, err)
	}
	return nil
}

// For overwriting when tests.
var findPath = func(paths []string) (string, error) {
	if len(paths) == 1 {
		return paths[0], nil
	}
	idx, err := fuzzyfinder.Find(paths, func(i int) string {
		return paths[i]
	})
	if err != nil {
		return "", err
	}
	return paths[idx], nil
}

func (e *explorer) explore(gvk schema.GroupVersionKind) {
	e.rootSchema = e.openAPISchema.LookupResource(gvk)
	e.rootSchema.Accept(e)
}

// paths returns paths explorer collects. paths that don't contain
// the path a user input will be ignored.
func (e *explorer) paths() []string {
	ps := make([]string, 0, len(e.pathSchema))
	for p := range e.pathSchema {
		if strings.Contains(p, e.inputFieldPath) {
			ps = append(ps, p)
		}
	}
	sort.Strings(ps)
	return ps
}

func (e *explorer) gvk() (schema.GroupVersionKind, error) {
	gvr, _, err := explain.SplitAndParseResourceRequest(e.kind, e.restMapper)
	if err != nil {
		return schema.GroupVersionKind{}, fmt.Errorf("get the group version resource by %s: %w", e.kind, err)
	}
	gvk, err := e.restMapper.KindFor(gvr)
	if err != nil {
		return schema.GroupVersionKind{}, fmt.Errorf("get a partial resource: %w", err)
	}
	if gvk.Empty() {
		gvk, err = e.restMapper.KindFor(gvr.GroupResource().WithVersion(""))
		if err != nil {
			return schema.GroupVersionKind{}, fmt.Errorf("get a partial resource: %w", err)
		}
	}
	return gvk, nil
}

func (e *explorer) VisitKind(k *proto.Kind) {
	keys := k.Keys()
	paths := make([]string, len(keys))
	for i, key := range keys {
		paths[i] = strings.Join([]string{e.prevPath, key}, ".")
	}
	for i, key := range keys {
		schema, err := explain.LookupSchemaForField(k, []string{key})
		if err != nil {
			e.err = err
			return
		}
		e.pathSchema[paths[i]] = schema
		e.prevPath = paths[i]
		schema.Accept(e)
	}
}

var visitedReferences = map[string]struct{}{}

func (e *explorer) VisitReference(r proto.Reference) {
	if _, ok := visitedReferences[r.Reference()]; ok {
		return
	}
	visitedReferences[r.Reference()] = struct{}{}
	r.SubSchema().Accept(e)
	delete(visitedReferences, r.Reference())
}

func (e *explorer) VisitPrimitive(p *proto.Primitive) {
	// Nothing to do.
}

func (e *explorer) VisitArray(a *proto.Array) {
	a.SubType.Accept(e)
}

func (e *explorer) VisitMap(m *proto.Map) {
	m.SubType.Accept(e)
}
