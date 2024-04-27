package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/ktr0731/go-fuzzyfinder"
	"k8s.io/kube-openapi/pkg/util/proto"
)

type explorer struct {
	// inputFieldPath must use the full-formed kind in lower case.
	// e.g., "hpa.spec" --> "horizontalpodautoscaler.spec"
	// "deploy.spec.template" --> "deployment.spec.template"
	// "sts.spec.template" --> "statefulset.spec.template"
	inputFieldPath string
	schemaByGvk    proto.Schema
	schemaVisitor  schemaVisitor
	explainer      explainer
}

func newExplorer(o *Options) (*explorer, error) {
	s := o.Schema.LookupResource(o.gvk)
	if s == nil {
		return nil, fmt.Errorf("no schema found for %s", o.gvk)
	}
	fullformedKind := strings.ToLower(o.gvk.Kind)
	pathSchema := make(map[string]proto.Schema)
	return &explorer{
		inputFieldPath: fullformInputFieldPath(o.inputFieldPath, fullformedKind),
		schemaByGvk:    s,
		schemaVisitor: schemaVisitor{
			prevPath:   fullformedKind,
			pathSchema: pathSchema,
			err:        nil,
		},
		explainer: explainer{
			schemaByGvk: s,
			gvk:         o.gvk,
			pathSchema:  pathSchema,
		},
	}, nil
}

// Convert abbreviated kind to full-formed.
func fullformInputFieldPath(inputFieldPath, fullformedKind string) string {
	if inputFieldPath == "" {
		return ""
	}
	ss := strings.Split(inputFieldPath, ".")
	if !strings.EqualFold(ss[0], fullformedKind) {
		ss[0] = fullformedKind
		return strings.Join(ss, ".")
	}
	return inputFieldPath
}

func (e *explorer) explore(w io.Writer) error {
	e.schemaByGvk.Accept(&e.schemaVisitor)
	if e.schemaVisitor.err != nil {
		return e.schemaVisitor.err
	}
	path, err := e.resolvePathWithUserInput()
	if err != nil {
		return fmt.Errorf("get the path to explain: %w", err)
	}
	return e.explainer.explain(w, path)
}

func (e *explorer) resolvePathWithUserInput() (string, error) {
	paths := e.schemaVisitor.listPaths(func(s string) bool {
		return strings.Contains(s, e.inputFieldPath)
	})
	if len(paths) == 0 {
		return "", fmt.Errorf("no paths found for %q", e.inputFieldPath)
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
			if err := e.explainer.explain(&w, paths[i]); err != nil {
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
