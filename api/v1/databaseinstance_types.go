/*
Copyright 2024.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Resources 定义了数据库的资源请求和限制
type Resources struct {
	// Requests 定义了内存和 CPU 的资源请求
	Requests ResourceRequests `json:"requests,omitempty"`
}

// ResourceRequests 定义了数据库的内存和 CPU 请求
type ResourceRequests struct {
	// Memory 表示内存请求
	Memory string `json:"memory,omitempty"`
	// CPU 表示 CPU 请求
	CPU string `json:"cpu,omitempty"`
}

// BackupPolicy 定义了备份策略配置
type BackupPolicy struct {
	// Enabled 指示是否启用备份
	Enabled bool `json:"enabled,omitempty"`

	// Schedule 定义备份的计划
	Schedule string `json:"schedule,omitempty"`

	// Retention 定义备份的保留策略
	Retention string `json:"retention,omitempty"`

	// BackupImage 定义备份容器使用的镜像
	BackupImage string `json:"backupImage,omitempty"`
}

// DatabaseInstanceSpec 定义了 DatabaseInstance 的期望状态
type DatabaseInstanceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - 定义集群的期望状态
	// Important: Run "make" to regenerate code after modifying this file

	// DatabaseType 表示数据库的类型（目前支持 mysql、postgres、oceanbase-ce 这 3 种）
	DatabaseType string `json:"databaseType,omitempty"`

	// Version 表示数据库的版本
	Version string `json:"version,omitempty"`

	// Storage 表示数据库的存储容量
	Storage string `json:"storage,omitempty"`

	// Replicas 表示数据库副本的数量
	Replicas int32 `json:"replicas,omitempty"`

	// Resources 定义了数据库的资源请求和限制
	Resources `json:"resources,omitempty"`

	// BackupPolicy 定义了备份策略配置
	BackupPolicy `json:"backupPolicy,omitempty"`

	// Image 表示数据库的容器镜像，包括版本/标签
	Image string `json:"image,omitempty"`
}

// DatabaseInstanceStatus 定义了 DatabaseInstance 资源被观察到的状态
type DatabaseInstanceStatus struct {
	// Phase 表示数据库实例当前所处的阶段（例如：Pending、Running、Failed）
	Phase string `json:"phase,omitempty"`

	// Message 表示相关状态的附加信息或错误消息
	Message string `json:"message,omitempty"`

	// ReadyReplicas 表示当前已就绪的副本数量
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// LastUpdated 是状态最后一次更新的时间戳
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`

	// Conditions 记录数据库实例的条件或状态
	Conditions []DatabaseInstanceCondition `json:"conditions,omitempty"`
}

// DatabaseInstanceCondition 表示数据库实例的特定方面的状态
type DatabaseInstanceCondition struct {
	// Type 条件的类型（例如：Ready、Available）
	Type string `json:"type"`

	// Status 条件的状态（例如：True、False）
	Status string `json:"status"`

	// LastTransitionTime 是条件从一个状态过渡到另一个状态的最后时间
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`

	// Reason 表示条件最后一次过渡的原因
	Reason string `json:"reason"`

	// Message 表示相关条件的详细信息
	Message string `json:"message"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// DatabaseInstance 是 databaseinstances API 的 Schema
type DatabaseInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec 定义了 DatabaseInstance 的期望状态
	Spec DatabaseInstanceSpec `json:"spec,omitempty"`
	// Status 定义了 DatabaseInstance 资源的观察到的状态
	Status DatabaseInstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DatabaseInstanceList 包含 DatabaseInstance 的列表
type DatabaseInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	// Items 是 DatabaseInstance 的列表
	Items []DatabaseInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DatabaseInstance{}, &DatabaseInstanceList{})
}
