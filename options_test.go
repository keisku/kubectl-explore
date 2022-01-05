package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest/fake"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	"k8s.io/kubectl/pkg/scheme"
)

func TestOptions_gvk(t *testing.T) {
	factory := newFactory(t)
	defer factory.Cleanup()
	mapper, _ := factory.ToRESTMapper()
	tests := []struct {
		apiVersion   string
		resourceName string
		want         schema.GroupVersionKind
		wantErr      string
	}{
		{
			apiVersion:   "v1",
			resourceName: "hoge",
			want:         schema.GroupVersionKind{},
			wantErr:      "get the group version resource by v1 hoge: no matches for /, Resource=hoge",
		},
		{
			resourceName: "hoge",
			want:         schema.GroupVersionKind{},
			wantErr:      "get the group version resource by  hoge: no matches for /, Resource=hoge",
		},
		{
			resourceName: "pod",
			want: schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "Pod",
			},
		},
		{
			apiVersion:   "v1",
			resourceName: "pod",
			want: schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "Pod",
			},
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("Get gvk by %s %s", tt.apiVersion, tt.resourceName), func(t *testing.T) {
			o := &Options{
				APIVersion: tt.apiVersion,
				Mapper:     mapper,
			}
			got, err := o.gvk(tt.resourceName)
			if tt.wantErr == "" {
				assert.Nil(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
			assert.Equal(t, tt.want, got)
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
	return factory
}
