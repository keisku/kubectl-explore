package main

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/ktr0731/go-fuzzyfinder"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kube-openapi/pkg/util/proto"
	"k8s.io/kubectl/pkg/explain"
	"k8s.io/kubectl/pkg/util/openapi"
)

// Explorer fields associated with each supported API resource to explain.
type Explorer struct {
	openAPISchema  openapi.Resources
	err            error
	inputFieldPath string
	prevPath       string
	pathSchema     map[string]proto.Schema
	schemaByGvk    proto.Schema
	gvk            schema.GroupVersionKind
}

// NewExplorer initializes Explorer.
func NewExplorer(fieldPath, kind string, r openapi.Resources, gvk schema.GroupVersionKind) (*Explorer, error) {
	s := r.LookupResource(gvk)
	if s == nil {
		return nil, fmt.Errorf("%#v is not found on the Open API schema", gvk)
	}
	return &Explorer{
		openAPISchema:  r,
		inputFieldPath: fieldPath,
		prevPath:       kind,
		pathSchema:     make(map[string]proto.Schema),
		schemaByGvk:    s,
		gvk:            gvk,
	}, nil
}

// Explore finds the field explanation, for example "pod.spec", "cronJob.spec.jobTemplate", etc.
func (e *Explorer) Explore(w io.Writer) error {
	e.schemaByGvk.Accept(e)
	if e.err != nil {
		return e.err
	}

	path, err := getPathToExplain(e)
	if err != nil {
		return fmt.Errorf("get the path to explain: %w", err)
	}

	return e.explain(w, path)
}

// getPathToExplain gets the path to explain by a user's input.
// Define this func as a variable for overwriting when tests.
var getPathToExplain = func(e *Explorer) (string, error) {
	paths := e.paths()
	if len(paths) == 0 {
		return "", nil
	}
	if len(paths) == 1 {
		return paths[0], nil
	}
	idx, err := fuzzyfinder.Find(
		paths,
		func(i int) string { return paths[i] },
		fuzzyfinder.WithPreviewWindow(func(i, _, _ int) string {
			// Prevent panic when no previews.
			// When the search result of fuzzy-find is 0, there is nothing to preview.
			// Then, index as a callback argument is -1.
			// https://github.com/ktr0731/go-fuzzyfinder/blob/3cbd4a4d9c88fe437ece2cbf91fbaf2fa0aa665f/fuzzyfinder.go#L270-L272
			if i < 0 {
				return ""
			}
			var w bytes.Buffer
			if err := e.explain(&w, paths[i]); err != nil {
				return fmt.Sprintf("preview is broken: %s", err)
			}
			return w.String()
		}),
	)
	if err != nil {
		return "", err
	}
	return paths[idx], nil
}

// paths returns paths explorer collects. paths that don't contain
// the path a user input will be ignored.
func (e *Explorer) paths() []string {
	ps := make([]string, 0, len(e.pathSchema))
	for p := range e.pathSchema {
		if strings.Contains(p, e.inputFieldPath) {
			ps = append(ps, p)
		}
	}
	sort.Strings(ps)
	return ps
}

// explain explains the field associated with the given path.
func (e *Explorer) explain(w io.Writer, path string) error {
	// This is the case that selected resource doesn't have any fields to explain.
	if path == "" {
		return explain.PrintModelDescription([]string{}, w, e.schemaByGvk, e.gvk, false)
	}
	// This is the case that path specifies the top-level field,
	// for example, "pod.spec", "pod.metadata"
	if strings.Count(path, ".") == 1 {
		fieldPath := []string{path[strings.LastIndex(path, ".")+1:]}
		return explain.PrintModelDescription(fieldPath, w, e.schemaByGvk, e.gvk, false)
	}

	// get the parent schema to explain.
	// e.g. "pod.spec.containers.env" -> "pod.spec.containers"
	parent, ok := e.pathSchema[path[:strings.LastIndex(path, ".")]]
	if !ok {
		return fmt.Errorf("%q is not found", path)
	}

	// get the key from the path.
	// e.g. "pod.spec.containers.env" -> "env"
	fieldPath := []string{path[strings.LastIndex(path, ".")+1:]}
	if err := explain.PrintModelDescription(fieldPath, w, parent, e.gvk, false); err != nil {
		return fmt.Errorf("explain %q: %w", path, err)
	}
	return nil
}

var _ proto.SchemaVisitor = (*Explorer)(nil)

func (e *Explorer) VisitKind(k *proto.Kind) {
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

func (e *Explorer) VisitReference(r proto.Reference) {
	if _, ok := visitedReferences[r.Reference()]; ok {
		return
	}
	visitedReferences[r.Reference()] = struct{}{}
	r.SubSchema().Accept(e)
	delete(visitedReferences, r.Reference())
}

func (e *Explorer) VisitPrimitive(p *proto.Primitive) {
	// Nothing to do.
}

func (e *Explorer) VisitArray(a *proto.Array) {
	a.SubType.Accept(e)
}

func (e *Explorer) VisitMap(m *proto.Map) {
	m.SubType.Accept(e)
}
