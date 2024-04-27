package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	openapi_v2 "github.com/google/gnostic/openapiv2"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kube-openapi/pkg/util/proto"
	"k8s.io/kubectl/pkg/util/openapi"
)

var k8sVersions = []string{"1.25", "1.26", "1.27", "1.28", "1.29", "1.30"}
var APIResourceByK8sVersion = func() map[string]openapi.Resources {
	resources := make(map[string]openapi.Resources, len(k8sVersions))
	for _, version := range k8sVersions {
		resources[version] = fetchOpenAPIResources(version)
	}
	return resources
}()

const urlToSwaggerJsonFormat = "https://raw.githubusercontent.com/kubernetes/kubernetes/release-%s/api/openapi-spec/swagger.json"

// fetchOpenAPIResources fetches swagger.json from the Kubernetes release on GitHub.
func fetchOpenAPIResources(version string) openapi.Resources {
	resp, err := http.DefaultClient.Get(fmt.Sprintf(urlToSwaggerJsonFormat, version))
	if err != nil {
		panic(fmt.Sprintf("fetch swagger.json: %s", err))
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Sprintf("read response body: %s", err))
	}
	doc, err := openapi_v2.ParseDocument(body)
	if err != nil {
		panic(fmt.Sprintf("parse swagger.json: %s", err))
	}
	r, err := openapi.NewOpenAPIData(doc)
	if err != nil {
		panic(fmt.Sprintf("creates a new resource from the doc: %s", err))
	}
	return r
}
func Test_fullformInputFieldPath(t *testing.T) {
	tests := []struct {
		inputFieldPath string
		fullformedKind string
		want           string
	}{
		{
			inputFieldPath: "sts.spec",
			fullformedKind: "statefulset",
			want:           "statefulset.spec",
		},
		{
			inputFieldPath: "sts",
			fullformedKind: "statefulset",
			want:           "statefulset",
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("make %s full-formed", tt.inputFieldPath), func(t *testing.T) {
			if got := fullformInputFieldPath(tt.inputFieldPath, tt.fullformedKind); got != tt.want {
				t.Errorf("fullformInputFieldPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_explain(t *testing.T) {
	tests := []struct {
		gvk             schema.GroupVersionKind
		expectUnsupport map[string]bool
		// key: path, value: section keys to check
		expectExplainOutput map[string][]string
	}{
		{
			gvk: schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"},
			expectExplainOutput: map[string][]string{
				".spec":                      {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".status":                    {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".metadata":                  {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".spec.hoge":                 {},
				".spec.containers":           {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".spec.affinity.podAffinity": {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".spec.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution.weight": {"KIND", "VERSION", "FIELD", "DESCRIPTION"},
			},
		},
		{
			gvk:             schema.GroupVersionKind{Group: "", Version: "v2", Kind: "Pod"},
			expectUnsupport: map[string]bool{"1.25": true, "1.26": true, "1.27": true, "1.28": true, "1.29": true, "1.30": true},
		},
		{
			gvk: schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Node"},
			expectExplainOutput: map[string][]string{
				".spec":      {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".status":    {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".metadata":  {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".spec.hoge": {},
			},
		},
		{
			gvk: schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Service"},
			expectExplainOutput: map[string][]string{
				".spec":      {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".status":    {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".metadata":  {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".spec.hoge": {},
			},
		},
		{
			gvk: schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"},
			expectExplainOutput: map[string][]string{
				".spec":      {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".status":    {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".metadata":  {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".spec.hoge": {},
			},
		},
		{
			gvk: schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"},
			expectExplainOutput: map[string][]string{
				".spec":      {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".status":    {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".metadata":  {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".spec.hoge": {},
			},
		},
		{
			gvk: schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "DaemonSet"},
			expectExplainOutput: map[string][]string{
				".spec":      {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".status":    {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".metadata":  {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".spec.hoge": {},
			},
		},
		{
			gvk: schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "ReplicaSet"},
			expectExplainOutput: map[string][]string{
				".spec":      {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".status":    {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".metadata":  {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".spec.hoge": {},
			},
		},
		{
			gvk: schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "StatefulSet"},
			expectExplainOutput: map[string][]string{
				".spec":      {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".status":    {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".metadata":  {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".spec.hoge": {},
			},
		},
		{
			gvk: schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "Job"},
			expectExplainOutput: map[string][]string{
				".spec":      {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".status":    {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".metadata":  {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".spec.hoge": {},
			},
		},
		{
			gvk: schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "CronJob"},
			expectExplainOutput: map[string][]string{
				".spec":      {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".status":    {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".metadata":  {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".spec.hoge": {},
			},
		},
		{
			gvk: schema.GroupVersionKind{Group: "autoscaling", Version: "v2", Kind: "HorizontalPodAutoscaler"},
			expectExplainOutput: map[string][]string{
				".spec":             {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".status":           {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".metadata":         {"KIND", "VERSION", "FIELD", "DESCRIPTION", "FIELDS"},
				".spec.hoge":        {},
				".spec.maxReplicas": {"KIND", "VERSION", "FIELD", "DESCRIPTION"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.gvk.String(), func(t *testing.T) {
			for _, version := range k8sVersions {
				schema := APIResourceByK8sVersion[version].LookupResource(tt.gvk)
				if tt.expectUnsupport[version] {
					require.Empty(t, schema, "%s: schema found for %s", version, tt.gvk)
					continue
				}
				require.NotNil(t, schema, "%s: schema not found for %s", version, tt.gvk)
				pathSchema := make(map[string]proto.Schema)
				e := &explainer{
					schemaByGvk: schema,
					gvk:         tt.gvk,
					pathSchema:  pathSchema,
				}
				v := &schemaVisitor{
					prevPath:   "",
					pathSchema: pathSchema,
					err:        nil,
				}
				schema.Accept(v)
				require.NoError(t, v.err, "%s: schemaVisitor must not return an error", version)
				for path, keys := range tt.expectExplainOutput {
					var buf bytes.Buffer
					err := e.explain(&buf, path)
					if len(keys) == 0 {
						require.Error(t, err, "%s: explain %q must return an error", version, path)
					} else {
						for _, key := range keys {
							require.True(t, strings.Contains(buf.String(), key), "%s: explain %q must contain %q: actual output: %q", version, path, key, buf)
						}
					}
				}
			}
		})
	}
}
