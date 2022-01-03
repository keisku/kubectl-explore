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
	type fields struct {
		APIVersion string
		kind       string
	}
	tests := []struct {
		fields  fields
		want    schema.GroupVersionKind
		wantErr string
	}{
		{
			fields: fields{
				APIVersion: "v1",
				kind:       "hoge",
			},
			want:    schema.GroupVersionKind{},
			wantErr: "get the group version resource by v1 hoge: no matches for /, Resource=hoge",
		},
		{
			fields: fields{
				kind: "hoge",
			},
			want:    schema.GroupVersionKind{},
			wantErr: "get the group version resource by  hoge: no matches for /, Resource=hoge",
		},
		{
			fields: fields{
				kind: "pod",
			},
			want: schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "Pod",
			},
		},
		{
			fields: fields{
				APIVersion: "v1",
				kind:       "pod",
			},
			want: schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "Pod",
			},
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("Get gvk by %s %s", tt.fields.APIVersion, tt.fields.kind), func(t *testing.T) {
			o := &Options{
				APIVersion: tt.fields.APIVersion,
				Mapper:     mapper,
				kind:       tt.fields.kind,
			}
			got, err := o.gvk()
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
