package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/rest/fake"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	"k8s.io/kubectl/pkg/scheme"
)

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

func TestOptions_getGVK(t *testing.T) {
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
			got, err := o.getGVK(tt.resourceName)
			if tt.wantErr == "" {
				assert.Nil(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOptions_listGVKs(t *testing.T) {
	stable := metav1.APIResourceList{
		GroupVersion: "v1",
		APIResources: []metav1.APIResource{
			{Name: "pods", Namespaced: true, Kind: "Pod"},
			{Name: "services", Namespaced: true, Kind: "Service"},
			{Name: "namespaces", Namespaced: false, Kind: "Namespace"},
		},
	}
	beta := metav1.APIResourceList{
		GroupVersion: "extensions/v1beta1",
		APIResources: []metav1.APIResource{
			{Name: "deployments", Namespaced: true, Kind: "Deployment"},
			{Name: "ingresses", Namespaced: true, Kind: "Ingress"},
			{Name: "jobs", Namespaced: true, Kind: "Job"},
		},
	}
	beta2 := metav1.APIResourceList{
		GroupVersion: "extensions/v1beta2",
		APIResources: []metav1.APIResource{
			{Name: "deployments", Namespaced: true, Kind: "Deployment"},
			{Name: "ingresses", Namespaced: true, Kind: "Ingress"},
			{Name: "jobs", Namespaced: true, Kind: "Job"},
		},
	}
	extensionsbeta3 := metav1.APIResourceList{GroupVersion: "extensions/v1beta3", APIResources: []metav1.APIResource{{Name: "deployments", Namespaced: true, Kind: "Deployment"}}}
	extensionsbeta4 := metav1.APIResourceList{GroupVersion: "extensions/v1beta4", APIResources: []metav1.APIResource{{Name: "deployments", Namespaced: true, Kind: "Deployment"}}}
	extensionsbeta5 := metav1.APIResourceList{GroupVersion: "extensions/v1beta5", APIResources: []metav1.APIResource{{Name: "deployments", Namespaced: true, Kind: "Deployment"}}}
	extensionsbeta6 := metav1.APIResourceList{GroupVersion: "extensions/v1beta6", APIResources: []metav1.APIResource{{Name: "deployments", Namespaced: true, Kind: "Deployment"}}}
	extensionsbeta7 := metav1.APIResourceList{GroupVersion: "extensions/v1beta7", APIResources: []metav1.APIResource{{Name: "deployments", Namespaced: true, Kind: "Deployment"}}}
	extensionsbeta8 := metav1.APIResourceList{GroupVersion: "extensions/v1beta8", APIResources: []metav1.APIResource{{Name: "deployments", Namespaced: true, Kind: "Deployment"}}}
	extensionsbeta9 := metav1.APIResourceList{GroupVersion: "extensions/v1beta9", APIResources: []metav1.APIResource{{Name: "deployments", Namespaced: true, Kind: "Deployment"}}}
	extensionsbeta10 := metav1.APIResourceList{GroupVersion: "extensions/v1beta10", APIResources: []metav1.APIResource{{Name: "deployments", Namespaced: true, Kind: "Deployment"}}}

	appsbeta1 := metav1.APIResourceList{GroupVersion: "apps/v1beta1", APIResources: []metav1.APIResource{{Name: "deployments", Namespaced: true, Kind: "Deployment"}}}
	appsbeta2 := metav1.APIResourceList{GroupVersion: "apps/v1beta2", APIResources: []metav1.APIResource{{Name: "deployments", Namespaced: true, Kind: "Deployment"}}}
	appsbeta3 := metav1.APIResourceList{GroupVersion: "apps/v1beta3", APIResources: []metav1.APIResource{{Name: "deployments", Namespaced: true, Kind: "Deployment"}}}
	appsbeta4 := metav1.APIResourceList{GroupVersion: "apps/v1beta4", APIResources: []metav1.APIResource{{Name: "deployments", Namespaced: true, Kind: "Deployment"}}}
	appsbeta5 := metav1.APIResourceList{GroupVersion: "apps/v1beta5", APIResources: []metav1.APIResource{{Name: "deployments", Namespaced: true, Kind: "Deployment"}}}
	appsbeta6 := metav1.APIResourceList{GroupVersion: "apps/v1beta6", APIResources: []metav1.APIResource{{Name: "deployments", Namespaced: true, Kind: "Deployment"}}}
	appsbeta7 := metav1.APIResourceList{GroupVersion: "apps/v1beta7", APIResources: []metav1.APIResource{{Name: "deployments", Namespaced: true, Kind: "Deployment"}}}
	appsbeta8 := metav1.APIResourceList{GroupVersion: "apps/v1beta8", APIResources: []metav1.APIResource{{Name: "deployments", Namespaced: true, Kind: "Deployment"}}}
	appsbeta9 := metav1.APIResourceList{GroupVersion: "apps/v1beta9", APIResources: []metav1.APIResource{{Name: "deployments", Namespaced: true, Kind: "Deployment"}}}
	appsbeta10 := metav1.APIResourceList{GroupVersion: "apps/v1beta10", APIResources: []metav1.APIResource{{Name: "deployments", Namespaced: true, Kind: "Deployment"}}}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var list interface{}
		switch req.URL.Path {
		case "/api/v1":
			list = &stable
		case "/apis/extensions/v1beta1":
			list = &beta
		case "/apis/extensions/v1beta2":
			list = &beta2
		case "/apis/extensions/v1beta3":
			list = &extensionsbeta3
		case "/apis/extensions/v1beta4":
			list = &extensionsbeta4
		case "/apis/extensions/v1beta5":
			list = &extensionsbeta5
		case "/apis/extensions/v1beta6":
			list = &extensionsbeta6
		case "/apis/extensions/v1beta7":
			list = &extensionsbeta7
		case "/apis/extensions/v1beta8":
			list = &extensionsbeta8
		case "/apis/extensions/v1beta9":
			list = &extensionsbeta9
		case "/apis/extensions/v1beta10":
			list = &extensionsbeta10
		case "/apis/apps/v1beta1":
			list = &appsbeta1
		case "/apis/apps/v1beta2":
			list = &appsbeta2
		case "/apis/apps/v1beta3":
			list = &appsbeta3
		case "/apis/apps/v1beta4":
			list = &appsbeta4
		case "/apis/apps/v1beta5":
			list = &appsbeta5
		case "/apis/apps/v1beta6":
			list = &appsbeta6
		case "/apis/apps/v1beta7":
			list = &appsbeta7
		case "/apis/apps/v1beta8":
			list = &appsbeta8
		case "/apis/apps/v1beta9":
			list = &appsbeta9
		case "/apis/apps/v1beta10":
			list = &appsbeta10
		case "/api":
			list = &metav1.APIVersions{
				Versions: []string{
					"v1",
				},
			}
		case "/apis":
			list = &metav1.APIGroupList{
				Groups: []metav1.APIGroup{
					{
						Name: "apps",
						Versions: []metav1.GroupVersionForDiscovery{
							{GroupVersion: "apps/v1beta1", Version: "v1beta1"},
							{GroupVersion: "apps/v1beta2", Version: "v1beta2"},
							{GroupVersion: "apps/v1beta3", Version: "v1beta3"},
							{GroupVersion: "apps/v1beta4", Version: "v1beta4"},
							{GroupVersion: "apps/v1beta5", Version: "v1beta5"},
							{GroupVersion: "apps/v1beta6", Version: "v1beta6"},
							{GroupVersion: "apps/v1beta7", Version: "v1beta7"},
							{GroupVersion: "apps/v1beta8", Version: "v1beta8"},
							{GroupVersion: "apps/v1beta9", Version: "v1beta9"},
							{GroupVersion: "apps/v1beta10", Version: "v1beta10"},
						},
					},
					{
						Name: "extensions",
						Versions: []metav1.GroupVersionForDiscovery{
							{GroupVersion: "extensions/v1beta1", Version: "v1beta1"},
							{GroupVersion: "extensions/v1beta2", Version: "v1beta2"},
							{GroupVersion: "extensions/v1beta3", Version: "v1beta3"},
							{GroupVersion: "extensions/v1beta4", Version: "v1beta4"},
							{GroupVersion: "extensions/v1beta5", Version: "v1beta5"},
							{GroupVersion: "extensions/v1beta6", Version: "v1beta6"},
							{GroupVersion: "extensions/v1beta7", Version: "v1beta7"},
							{GroupVersion: "extensions/v1beta8", Version: "v1beta8"},
							{GroupVersion: "extensions/v1beta9", Version: "v1beta9"},
							{GroupVersion: "extensions/v1beta10", Version: "v1beta10"},
						},
					},
				},
			}
		default:
			t.Logf("unexpected request: %s", req.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		output, err := json.Marshal(list)
		if err != nil {
			t.Errorf("unexpected encoding error: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(output)
	}))
	defer server.Close()
	dc := discovery.NewDiscoveryClientForConfigOrDie(&restclient.Config{Host: server.URL})
	tdc := &testDiscoverClient{dc}
	o := &Options{
		Discovery: tdc,
	}
	got, err := o.listGVKs()
	assert.Nil(t, err)
	assert.ElementsMatch(t, got, []schema.GroupVersionKind{
		{Group: "", Version: "v1", Kind: "Pod"},
		{Group: "", Version: "v1", Kind: "Service"},
		{Group: "", Version: "v1", Kind: "Namespace"},
		{Group: "apps", Version: "v1beta1", Kind: "Deployment"},
		{Group: "extensions", Version: "v1beta1", Kind: "Deployment"},
		{Group: "extensions", Version: "v1beta1", Kind: "Ingress"},
		{Group: "extensions", Version: "v1beta1", Kind: "Job"},
	})
}

var _ discovery.CachedDiscoveryInterface = (*testDiscoverClient)(nil)

type testDiscoverClient struct {
	*discovery.DiscoveryClient
}

func (dc *testDiscoverClient) Fresh() bool { return true }

func (dc *testDiscoverClient) Invalidate() {}
