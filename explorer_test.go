package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"

	openapi_v2 "github.com/googleapis/gnostic/openapiv2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest/fake"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/kubectl/pkg/util/openapi"
)

func Test_Explorer_Run(t *testing.T) {
	factory := newFactory(t)
	defer factory.Cleanup()
	tests := []struct {
		inputFieldPath string
		wantW          string
		wantErr        string
	}{
		{
			inputFieldPath: "node.spec.hoge",
			wantErr:        `explain "node.spec.hoge": field "hoge" does not exist`,
		},
		{
			inputFieldPath: "hoge.foo.bar",
			wantErr:        "get the group version resource by hoge: no matches for /, Resource=hoge",
		},
		{
			inputFieldPath: "pod.spec.tolerations.key",
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
			e, err := NewExplorer(factory, tt.inputFieldPath)
			assert.Nil(t, err)
			// Overwrite this func for testing.
			// Usually, the result depends on the user's input.
			getPathToExplain = func(_ []string) (string, error) {
				return tt.inputFieldPath, nil
			}
			var b bytes.Buffer
			err = e.Run(&b)
			if tt.wantErr == "" {
				assert.Nil(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
			assert.Equal(t, tt.wantW, b.String())
		})
	}
}

func newFactory(t *testing.T) *cmdtesting.TestFactory {
	t.Helper()

	factory := cmdtesting.NewTestFactory().WithNamespace("test")
	factory.Client = &fake.RESTClient{
		NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
		GroupVersion:         corev1.SchemeGroupVersion,
		Client: fake.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("request url: %#v, and request: %#v", req.URL, req)
		}),
	}
	factory.ClientConfigVal = cmdtesting.DefaultClientConfig()
	factory.OpenAPISchemaFunc = func() (openapi.Resources, error) { return fetchOpenAPIResources(t), nil }
	return factory
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
