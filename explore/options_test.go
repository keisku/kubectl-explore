package explore_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	openapi_v2 "github.com/google/gnostic/openapiv2"
	"github.com/keisku/kubectl-explore/explore"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	openapiclient "k8s.io/client-go/openapi"
	"k8s.io/client-go/rest"
	clienttestutil "k8s.io/client-go/util/testing"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	"k8s.io/kubectl/pkg/util/openapi"
)

const (
	openAPISpecV3PathURLFormat = "https://github.com/kubernetes/kubernetes/tree-commit-info/release-%s/api/openapi-spec/v3"
	rawGithubusercontent       = "https://raw.githubusercontent.com"
	openAPISpecV3DirFormat     = "/kubernetes/kubernetes/release-%s/api/openapi-spec/v3/"
	openAPISpecV3FileURLFormat = rawGithubusercontent + openAPISpecV3DirFormat + "%s"
	swaggerURLFormat           = "https://raw.githubusercontent.com/kubernetes/kubernetes/release-%s/api/openapi-spec/swagger.json"
)

var k8sVersions = []string{"1.25", "1.26", "1.27", "1.28", "1.29", "1.30", "1.31", "1.32", "1.33", "1.34"}

func openAPIResources(version string) (openapi.Resources, error) {
	resp, err := http.DefaultClient.Get(fmt.Sprintf(swaggerURLFormat, version))
	if err != nil {
		return nil, fmt.Errorf("fetch swagger.json: %s", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %s", err)
	}
	doc, err := openapi_v2.ParseDocument(body)
	if err != nil {
		return nil, fmt.Errorf("parse swagger.json: %s", err)
	}
	r, err := openapi.NewOpenAPIData(doc)
	if err != nil {
		return nil, fmt.Errorf("creates a new resource from the doc: %s", err)
	}
	return r, nil
}

func openAPISpecV3FilePaths(version string) ([]string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf(openAPISpecV3PathURLFormat, version), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get response: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	jsonData := make(map[string]interface{})
	if err := json.Unmarshal(body, &jsonData); err != nil {
		return nil, fmt.Errorf("unmarshal response body: %w", err)
	}
	var paths []string
	for fileName := range jsonData {
		paths = append(paths, fmt.Sprintf(openAPISpecV3FileURLFormat, version, fileName))
	}
	return paths, nil
}

var openAPISpecV3Directories map[string]string = func() map[string]string {
	m := make(map[string]string)
	var wg sync.WaitGroup
	wg.Add(len(k8sVersions))
	for _, version := range k8sVersions {
		go func(version string) {
			testdata, err := makeOpenAPISpecV3Directory(version)
			if err != nil {
				panic(err)
			}
			m[version] = testdata
			wg.Done()
		}(version)
	}
	wg.Wait()
	return m
}()

func makeOpenAPISpecV3Directory(version string) (string, error) {
	paths, err := openAPISpecV3FilePaths(version)
	if err != nil {
		return "", fmt.Errorf("get openapi spec v3 file paths: %w", err)
	}
	testdataDir := filepath.Join(os.TempDir(), "kubectl-explore.d", version)
	if err := os.MkdirAll(testdataDir, 0o755); err != nil {
		return "", fmt.Errorf("create testdata directory: %w", err)
	}
	for _, path := range paths {
		filePathURL, err := url.Parse(path)
		if err != nil {
			return "", fmt.Errorf("parse file path URL: %w", err)
		}
		fpath := filepath.Join(testdataDir, convertFilename(filepath.Base(filePathURL.Path)))
		if os.MkdirAll(filepath.Dir(fpath), 0o755); err != nil {
			return "", fmt.Errorf("create directory: %w", err)
		}
		f, err := os.Create(fpath)
		if err != nil {
			return "", fmt.Errorf("create file: %w", err)
		}
		resp, err := http.DefaultClient.Get(path)
		if err != nil {
			return "", fmt.Errorf("get response: %w", err)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("read response body: %w", err)
		}
		resp.Body.Close()
		if _, err := f.Write(body); err != nil {
			return "", fmt.Errorf("write response body: %w", err)
		}
	}
	return testdataDir, nil
}

func convertFilename(filename string) string {
	parts := strings.Split(filename, "__")
	newPath := strings.Join(parts, "/")
	newPath = strings.Replace(newPath, "_openapi.json", ".json", -1)
	return newPath
}

func Test_Run(t *testing.T) {
	fakeServers := make(map[string]*clienttestutil.FakeOpenAPIServer)
	for _, version := range k8sVersions {
		testdata := openAPISpecV3Directories[version]
		fakeServer, err := clienttestutil.NewFakeOpenAPIV3Server(testdata)
		if err != nil {
			t.Fatalf("failed to create fake openapi server: %s", err)
			return
		}
		fakeServers[version] = fakeServer
		t.Cleanup(func() {
			os.RemoveAll(testdata)
			fakeServer.HttpServer.Close()
		})
	}
	fakeCachedDiscoveryClient := cmdtesting.NewFakeCachedDiscoveryClient()
	fakeCachedDiscoveryClient.PreferredResources = []*v1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []v1.APIResource{
				{
					Name:         "nodes",
					SingularName: "node",
					Namespaced:   false,
					Kind:         "Node",
					ShortNames:   []string{"no"},
				},
			},
		},
		{
			GroupVersion: "autoscaling/v2",
			APIResources: []v1.APIResource{
				{
					Name:         "horizontalpodautoscalers",
					SingularName: "horizontalpodautoscaler",
					Namespaced:   true,
					Kind:         "HorizontalPodAutoscaler",
					ShortNames:   []string{"hpa"},
				},
			},
		},
		{
			GroupVersion: "autoscaling/v1",
			APIResources: []v1.APIResource{
				{
					Name:         "horizontalpodautoscalers",
					SingularName: "horizontalpodautoscaler",
					Namespaced:   true,
					Kind:         "HorizontalPodAutoscaler",
					ShortNames:   []string{"hpa"},
				},
			},
		},
		{
			GroupVersion: "storage.k8s.io/v1",
			APIResources: []v1.APIResource{
				{
					Name:         "csistoragecapacities",
					SingularName: "csistoragecapacity",
					Namespaced:   true,
					Kind:         "CSIStorageCapacity",
					ShortNames:   []string{},
				},
			},
		},
		{
			GroupVersion: "v1",
			APIResources: []v1.APIResource{
				{
					Name:         "componentstatuses",
					SingularName: "componentstatus",
					Namespaced:   false,
					Kind:         "ComponentStatus",
					ShortNames:   []string{"cs"},
				},
			},
		},
	}
	tests := []struct {
		apiVersion       string
		inputFieldPath   string
		disablePrintPath bool
		showBrackets     bool
		expectRunError   bool
		expectKeywords   []string
		unexpectKeywords []string
	}{
		{
			inputFieldPath:   "no.*pro",
			disablePrintPath: false,
			showBrackets:     false,
			expectRunError:   false,
			expectKeywords: []string{
				"Node",
				"providerID",
				"PATH: nodes.spec.providerID",
			},
		},
		{
			inputFieldPath:   "node.*pro",
			disablePrintPath: false,
			showBrackets:     false,
			expectRunError:   false,
			expectKeywords: []string{
				"Node",
				"providerID",
				"PATH: nodes.spec.providerID",
			},
		},
		{
			inputFieldPath:   "nodes.*pro",
			disablePrintPath: false,
			showBrackets:     false,
			expectRunError:   false,
			expectKeywords: []string{
				"Node",
				"providerID",
				"PATH: nodes.spec.providerID",
			},
		},
		{
			inputFieldPath:   "providerID",
			disablePrintPath: false,
			showBrackets:     false,
			expectRunError:   false,
			expectKeywords: []string{
				"Node",
				"providerID",
				"PATH: nodes.spec.providerID",
			},
		},
		{
			apiVersion:       "autoscaling/v2",
			inputFieldPath:   "hpa.*own.*id",
			disablePrintPath: false,
			showBrackets:     false,
			expectRunError:   false,
			expectKeywords: []string{
				"autoscaling",
				"HorizontalPodAutoscaler",
				"v2",
				"PATH: horizontalpodautoscalers.metadata.ownerReferences.uid",
			},
		},
		{
			apiVersion:       "autoscaling/v1",
			inputFieldPath:   "horizontalpodautoscalers.*own.*id",
			disablePrintPath: false,
			showBrackets:     false,
			expectRunError:   false,
			expectKeywords: []string{
				"autoscaling",
				"HorizontalPodAutoscaler",
				"v1",
				"PATH: horizontalpodautoscalers.metadata.ownerReferences.uid",
			},
		},
		{
			apiVersion:       "autoscaling/v2",
			inputFieldPath:   "horizontalpodautoscaler.*own.*id",
			disablePrintPath: false,
			showBrackets:     false,
			expectRunError:   false,
			expectKeywords: []string{
				"autoscaling",
				"HorizontalPodAutoscaler",
				"v2",
				"PATH: horizontalpodautoscalers.metadata.ownerReferences.uid",
			},
		},
		{
			inputFieldPath:   "csistoragecapacity.maximumVolumeSize",
			disablePrintPath: false,
			showBrackets:     false,
			expectRunError:   false,
			expectKeywords: []string{
				"CSIStorageCapacity",
				"storage.k8s.io",
				"v1",
				"PATH: csistoragecapacities.maximumVolumeSize",
			},
		},
		{
			inputFieldPath:   "csistoragecapacities.maximumVolumeSize",
			disablePrintPath: false,
			showBrackets:     false,
			expectRunError:   false,
			expectKeywords: []string{
				"CSIStorageCapacity",
				"storage.k8s.io",
				"v1",
				"PATH: csistoragecapacities.maximumVolumeSize",
			},
		},
		{
			inputFieldPath:   "CSIStorageCapacity.*VolumeSize",
			disablePrintPath: false,
			showBrackets:     false,
			expectRunError:   false,
			expectKeywords: []string{
				"CSIStorageCapacity",
				"storage.k8s.io",
				"v1",
				"PATH: csistoragecapacities.maximumVolumeSize",
			},
		},
		{
			inputFieldPath:   "nodes.status.conditions.type",
			disablePrintPath: false,
			showBrackets:     true,
			expectRunError:   false,
			expectKeywords: []string{
				"Node",
				"type",
				"PATH: nodes.status.conditions[].type",
			},
		},
		{
			inputFieldPath:   "nodes.status.conditions.type",
			disablePrintPath: true,
			showBrackets:     true,
			expectRunError:   false,
			expectKeywords: []string{
				"Node",
				"type",
			},
			unexpectKeywords: []string{
				"PATH: nodes.status.conditions[].type",
				"PATH: nodes.status.conditions.type",
			},
		},
	}
	for _, tt := range tests {
		for _, version := range k8sVersions {
			t.Run(fmt.Sprintf(
				"version: %s inputFieldPath: %s, disablePrintPath: %v, showBrackets: %v",
				version,
				tt.inputFieldPath,
				tt.disablePrintPath,
				tt.showBrackets,
			), func(t *testing.T) {
				fakeServer := fakeServers[version]
				fakeDiscoveryClient := discovery.NewDiscoveryClientForConfigOrDie(&rest.Config{Host: fakeServer.HttpServer.URL})
				tf := cmdtesting.NewTestFactory()
				defer tf.Cleanup()
				tf.WithDiscoveryClient(fakeCachedDiscoveryClient)
				tf.OpenAPIV3ClientFunc = func() (openapiclient.Client, error) {
					return fakeDiscoveryClient.OpenAPIV3(), nil
				}
				tf.OpenAPISchemaFunc = func() (openapi.Resources, error) {
					return openAPIResources(version)
				}
				tf.ClientConfigVal = cmdtesting.DefaultClientConfig()

				var stdin bytes.Buffer
				var stdout bytes.Buffer
				var errout bytes.Buffer
				opts := explore.NewOptions(genericclioptions.IOStreams{
					In:     &stdin,
					Out:    &stdout,
					ErrOut: &errout,
				})
				explore.SetDisablePrintPath(opts, tt.disablePrintPath)
				explore.SetShowBrackets(opts, tt.showBrackets)
				explore.SetAPIVersion(opts, tt.apiVersion)
				require.NoError(t, opts.Complete(tf, []string{tt.inputFieldPath}))
				err := opts.Run()
				if tt.expectRunError {
					require.NotNil(t, err)
				} else {
					require.NoError(t, err)
				}
				for _, keyword := range tt.expectKeywords {
					require.Contains(t, stdout.String(), keyword)
				}
				for _, keyword := range tt.unexpectKeywords {
					require.NotContains(t, stdout.String(), keyword)
				}
			})
		}
	}
}
