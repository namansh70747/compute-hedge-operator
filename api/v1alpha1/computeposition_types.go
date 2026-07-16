package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Priority describes how tolerant a position is to being paused.
type Priority string

const (
	// PriorityCritical workloads are never paused automatically.
	PriorityCritical Priority = "critical"
	// PriorityBatch workloads may be paused on a sustained price spike when actions are enabled.
	PriorityBatch Priority = "batch"
)

// WorkloadRef points at the Deployment backing a position, used for optional pause/resume.
type WorkloadRef struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// ComputePositionSpec is the desired state of a hedged block of GPU capacity.
type ComputePositionSpec struct {
	// SKU is the GPU type, e.g. H100, H200, B200, RTX5090.
	SKU string `json:"sku"`

	// GPUCount is the number of GPUs covered by this position.
	GPUCount int32 `json:"gpuCount"`

	// HedgedPriceUSDPerHour is the per-GPU price locked in by the operator's hedge.
	HedgedPriceUSDPerHour string `json:"hedgedPriceUSDPerHour"`

	// Priority controls pause eligibility. Defaults to batch.
	Priority Priority `json:"priority,omitempty"`

	// IdleThresholdPct is the utilization below which capacity counts as idle. Defaults to 15.
	IdleThresholdPct int32 `json:"idleThresholdPct,omitempty"`

	// IdleWindowSeconds is how long utilization must stay low before flagging idle. Defaults to 30.
	IdleWindowSeconds int32 `json:"idleWindowSeconds,omitempty"`

	// EnableActions turns on optional pause/resume of the referenced batch workload.
	// Off by default: the controller is advisory unless an operator opts in per position.
	EnableActions bool `json:"enableActions,omitempty"`

	// MaxSpotPriceUSDPerHour is the price above which an opted-in batch workload may be paused.
	MaxSpotPriceUSDPerHour string `json:"maxSpotPriceUSDPerHour,omitempty"`

	// WorkloadRef is the Deployment to pause/resume when EnableActions is set.
	WorkloadRef *WorkloadRef `json:"workloadRef,omitempty"`
}

// ComputePositionStatus is the observed state of the position.
type ComputePositionStatus struct {
	Phase                 string `json:"phase,omitempty"`
	UtilizationPct        int32  `json:"utilizationPct,omitempty"`
	SpotPriceUSDPerHour   string `json:"spotPriceUSDPerHour,omitempty"`
	SpotCostUSDPerHour    string `json:"spotCostUSDPerHour,omitempty"`
	HedgePnLUSDPerHour    string `json:"hedgePnLUSDPerHour,omitempty"`
	HedgeEffectivenessPct int32  `json:"hedgeEffectivenessPct,omitempty"`
	BasisRiskUSDPerHour   string `json:"basisRiskUSDPerHour,omitempty"`
	IdleGPUCount          int32  `json:"idleGPUCount,omitempty"`
	AvailableForSublet    bool   `json:"availableForSublet,omitempty"`
	Recommendation        string `json:"recommendation,omitempty"`
	PriceStale            bool   `json:"priceStale,omitempty"`

	// IdleSince tracks the start of a sustained-idle window across reconciles.
	IdleSince *metav1.Time `json:"idleSince,omitempty"`
	// Paused records whether the controller scaled the workload down.
	Paused bool `json:"paused,omitempty"`
	// OriginalReplicas stores replicas to restore after a pause.
	OriginalReplicas *int32       `json:"originalReplicas,omitempty"`
	LastUpdated      *metav1.Time `json:"lastUpdated,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ComputePosition is a hedged block of GPU capacity tracked against the OCPI index.
type ComputePosition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ComputePositionSpec   `json:"spec,omitempty"`
	Status ComputePositionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ComputePositionList is a list of ComputePosition.
type ComputePositionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ComputePosition `json:"items"`
}
