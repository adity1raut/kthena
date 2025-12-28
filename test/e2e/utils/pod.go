/*
Copyright The Volcano Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// GetRouterPod is a helper function to get the router pod
func GetRouterPod(t *testing.T, kubeClient kubernetes.Interface, kthenaNamespace string) *corev1.Pod {
	deployment, err := kubeClient.AppsV1().Deployments(kthenaNamespace).Get(context.Background(), "kthena-router", metav1.GetOptions{})
	require.NoError(t, err, "Failed to get router deployment")

	// Build label selector from deployment selector
	labelSelector := ""
	for key, value := range deployment.Spec.Selector.MatchLabels {
		if labelSelector != "" {
			labelSelector += ","
		}
		labelSelector += key + "=" + value
	}

	// Get router pod
	pods, err := kubeClient.CoreV1().Pods(kthenaNamespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	require.NoError(t, err, "Failed to list router pods")
	require.NotEmpty(t, pods.Items, "No router pods found")

	return &pods.Items[0]
}

// ExecInPod executes a command in a pod and returns the output
func ExecInPod(t *testing.T, config *rest.Config, pod *corev1.Pod, container string, command []string) (string, string, error) {
	req := kubernetes.NewForConfigOrDie(config).CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec")

	req.VersionedParams(&corev1.PodExecOptions{
		Container: container,
		Command:   command,
		Stdout:    true,
		Stderr:    true,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return "", "", fmt.Errorf("failed to create executor: %w", err)
	}

	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(context.Background(), remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil && err != io.EOF {
		return stdout.String(), stderr.String(), fmt.Errorf("failed to execute command: %w", err)
	}

	return stdout.String(), stderr.String(), nil
}
