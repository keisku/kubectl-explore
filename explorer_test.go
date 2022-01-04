package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	openapi_v2 "github.com/googleapis/gnostic/openapiv2"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubectl/pkg/util/openapi"
)

func Test_Explorer_Explore(t *testing.T) {
	openAPIResources := fetchOpenAPIResources(t)
	tests := []struct {
		inputFieldPath string
		gvk            schema.GroupVersionKind
		wantW          string
		wantErr        string
	}{
		{
			inputFieldPath: "node.spec.hoge",
			gvk: schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "Node",
			},
			wantErr: `explain "node.spec.hoge": field "hoge" does not exist`,
		},
		{
			inputFieldPath: "pod.spec.tolerations.key",
			gvk: schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "Pod",
			},
			wantW: `KIND:     Pod
VERSION:  v1

FIELD:    key <string>

DESCRIPTION:
     Key is the taint key that the toleration applies to. Empty means match all
     taint keys. If the key is empty, operator must be Exists; this combination
     means to match all values and all keys.
`,
		},
		{
			inputFieldPath: "pod.spec.serviceAccount",
			gvk: schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "Pod",
			},
			wantW: `KIND:     Pod
VERSION:  v1

FIELD:    serviceAccount <string>

DESCRIPTION:
     DeprecatedServiceAccount is a depreciated alias for ServiceAccountName.
     Deprecated: Use serviceAccountName instead.
`,
		},
		{
			inputFieldPath: "node.spec",
			gvk: schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "Node",
			},
			wantW: `KIND:     Node
VERSION:  v1

RESOURCE: spec <Object>

DESCRIPTION:
     Spec defines the behavior of a node.
     https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status

     NodeSpec describes the attributes that a node is created with.

FIELDS:
   configSource	<Object>
     Deprecated. If specified, the source of the node's configuration. The
     DynamicKubeletConfig feature gate must be enabled for the Kubelet to use
     this field. This field is deprecated as of 1.22:
     https://git.k8s.io/enhancements/keps/sig-node/281-dynamic-kubelet-configuration

   externalID	<string>
     Deprecated. Not all kubelets will set this field. Remove field after 1.13.
     see: https://issues.k8s.io/61966

   podCIDR	<string>
     PodCIDR represents the pod IP range assigned to the node.

   podCIDRs	<[]string>
     podCIDRs represents the IP ranges assigned to the node for usage by Pods on
     that node. If this field is specified, the 0th entry must match the podCIDR
     field. It may contain at most 1 value for each of IPv4 and IPv6.

   providerID	<string>
     ID of the node assigned by the cloud provider in the format:
     <ProviderName>://<ProviderSpecificNodeID>

   taints	<[]Object>
     If specified, the node's taints.

   unschedulable	<boolean>
     Unschedulable controls node schedulability of new pods. By default, node is
     schedulable. More info:
     https://kubernetes.io/docs/concepts/nodes/node/#manual-node-administration

`,
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf(`Explain "%s"`, tt.inputFieldPath), func(t *testing.T) {
			e, err := NewExplorer(
				tt.inputFieldPath,
				strings.ToLower(tt.gvk.Kind),
				openAPIResources,
				tt.gvk,
			)
			assert.Nil(t, err)
			// Overwrite this func for testing.
			// Usually, the result depends on the user's input.
			getPathToExplain = func(_ *Explorer) (string, error) {
				return tt.inputFieldPath, nil
			}
			var b bytes.Buffer
			err = e.Explore(&b)
			if tt.wantErr == "" {
				assert.Nil(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
			assert.Equal(t, tt.wantW, b.String())
		})
	}
}

const urlToSaggerJsonFormat = "https://raw.githubusercontent.com/kubernetes/kubernetes/release-%s/api/openapi-spec/swagger.json"
const swaggerJsonVersion = "1.23"

// fetchOpenAPIResources fetches swagger.json from the Kubernetes release on GitHub.
func fetchOpenAPIResources(t *testing.T) openapi.Resources {
	t.Helper()

	resp, err := http.DefaultClient.Get(fmt.Sprintf(urlToSaggerJsonFormat, swaggerJsonVersion))
	if err != nil {
		t.Fatalf("fetch swagger.json: %s", err)
		return nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %s", err)
		return nil
	}
	doc, err := openapi_v2.ParseDocument(body)
	if err != nil {
		t.Fatalf("parse swagger.json: %s", err)
		return nil
	}
	r, err := openapi.NewOpenAPIData(doc)
	if err != nil {
		t.Fatalf("creates a new resource from the doc: %s", err)
		return nil
	}
	return r
}
