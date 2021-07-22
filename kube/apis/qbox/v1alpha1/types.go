package v1alpha1

import (
	"time"

	appv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WaitDeployment customize deployment resource definition
type Waitdeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              appv1.DeploymentSpec   `json:"spec,omitempty"`
	Status            appv1.DeploymentStatus `json:"status,omitempty"`
	WaitProbe         WaitProbe              `json:"waitProbe,omitempty"`
}

type WaitProbe struct {
	Address string        `json:"address,omitempty"`
	Timeout time.Duration `json:"timeout,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type WaitdeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Waitdeployment `json:"items"`
}
