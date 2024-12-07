package explore

import (
	"fmt"
	"io"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	openapiclient "k8s.io/client-go/openapi"
	explainv2 "k8s.io/kubectl/pkg/explain/v2"
)

type explainer struct {
	gvr                 schema.GroupVersionResource
	openAPIV3Client     openapiclient.Client
	enablePrintPath     bool
	enablePrintBrackets bool
}

func (e explainer) explain(w io.Writer, path path) error {
	if path.isEmpty() {
		return fmt.Errorf("path must not be empty: %#v", path)
	}
	fields := strings.Split(path.original, ".")
	if len(fields) > 0 {
		// Remove resource name
		fields = fields[1:]
	}
	if e.enablePrintPath {
		if e.enablePrintBrackets {
			w.Write([]byte(fmt.Sprintf("PATH: %s\n", path.withBrackets)))
		} else {
			w.Write([]byte(fmt.Sprintf("PATH: %s\n", path.original)))
		}
	}
	return explainv2.PrintModelDescription(
		fields,
		w,
		e.openAPIV3Client,
		e.gvr,
		false,
		"plaintext",
	)
}
