package utils

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"k8s.io/klog/v2"
	extenderv1 "k8s.io/kube-scheduler/extender/v1"
)

func ExtractExtenderArgsFromRequest(r *http.Request) (*extenderv1.ExtenderArgs, error) {
	extenderArgs := &extenderv1.ExtenderArgs{}
	var buf bytes.Buffer
	body := io.TeeReader(r.Body, &buf)

	if err := json.NewDecoder(body).Decode(extenderArgs); err != nil {
		klog.Errorf("failed to decode filter extenderArgs, request: %s, err: %v", buf.String(), err)
		return nil, err
	}
	return extenderArgs, nil
}
