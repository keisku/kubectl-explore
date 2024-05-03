package explore

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	openapiclient "k8s.io/client-go/openapi"
	explainv2 "k8s.io/kubectl/pkg/explain/v2"
)

type explainer struct {
	gvr             schema.GroupVersionResource
	openAPIV3Client openapiclient.Client
	enablePrintPath bool
}

func (e explainer) explain(w io.Writer, path string) error {
	if path == "" {
		return errors.New("path must be provided")
	}
	fields := strings.Split(path, ".")
	if len(fields) > 0 {
		// Remove resource name
		fields = fields[1:]
	}
	if e.enablePrintPath {
		w.Write([]byte(fmt.Sprintf("PATH: %s\n", path)))
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
