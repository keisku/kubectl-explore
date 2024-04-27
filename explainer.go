package main

import (
	"fmt"
	"io"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kube-openapi/pkg/util/proto"
	"k8s.io/kubectl/pkg/explain"
)

type explainer struct {
	schemaByGvk proto.Schema
	gvk         schema.GroupVersionKind
	pathSchema  map[string]proto.Schema
}

// explain explains the field associated with the given path.
func (e *explainer) explain(w io.Writer, path string) error {
	if path == "" {
		return fmt.Errorf("path is empty: gvk=%s", e.gvk)
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
