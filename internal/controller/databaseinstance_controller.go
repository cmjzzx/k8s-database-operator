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

package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"                   // 导入 Kubernetes 的 apps/v1
	corev1 "k8s.io/api/core/v1"                   // 导入 Kubernetes 的 core/v1 包
	"k8s.io/apimachinery/pkg/api/errors"          // 错误处理
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1" // 导入 metav1 包
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr" // 导入 intstr 包

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	databasev1 "git.ucmed.cn/pdd/k8s-database-operator/api/v1" // 导入 databasev1
)

// 定义 DatabaseInstanceReconciler 结构体，负责调节 DatabaseInstance 对象的状态
type DatabaseInstanceReconciler struct {
	// 通过嵌入 client.Client 类型，获得 Client 接口的所有方法
	// 这样就可以直接在 DatabaseInstanceReconciler 的方法中使用 Client 提供的方法，如 Get、Create、Update 和 Delete
	client.Client
	// Scheme 是 controller-runtime 提供的用于将资源对象的类型与其 JSON 或 YAML 表示之间进行映射的对象
	// 在 Reconciler 中使用 Scheme 可确保正确处理资源的类型和转换
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=apps.zwjk.com,resources=databaseinstances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.zwjk.com,resources=databaseinstances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.zwjk.com,resources=databaseinstances/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DatabaseInstance object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *DatabaseInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 获取当前的 DatabaseInstance 实例
	var dbInstance databasev1.DatabaseInstance
	if err := r.Get(ctx, req.NamespacedName, &dbInstance); err != nil {
		logger.Error(err, "未获取到 DatabaseInstance 实例")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 定义镜像的前缀
	const imagePrefix = "registry.zwjk.com/middleware/"

	// 获取镜像名称
	image := dbInstance.Spec.Image
	if image == "" {
		image = fmt.Sprintf("%s:%s", dbInstance.Spec.DatabaseType, dbInstance.Spec.Version)
	}

	// 在镜像名称前面加上前缀
	image = fmt.Sprintf("%s%s", imagePrefix, image)

	// 构造期望的 Deployment 对象
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dbInstance.Name,
			Namespace: dbInstance.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &dbInstance.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": dbInstance.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": dbInstance.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "database",
							Image: image,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 3306,
								},
							},
						},
					},
				},
			},
		},
	}

	// 确保 Deployment 存在
	found := &appsv1.Deployment{}
	err := r.Get(ctx, client.ObjectKey{Name: dbInstance.Name, Namespace: dbInstance.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Deployment 不存在，创建它
		logger.Info("创建一个新的 Deployment", "Deployment.Namespace", dbInstance.Namespace, "Deployment.Name", dbInstance.Name)
		if err := r.Create(ctx, deployment); err != nil {
			logger.Error(err, "新的 Deployment 失败失败")
			return ctrl.Result{}, err
		}
	} else if err != nil {
		logger.Error(err, "获取 Deployment 失败")
		return ctrl.Result{}, err
	} else {
		// Deployment 存在，更新它
		logger.Info("更新已有的 Deployment", "Deployment.Namespace", dbInstance.Namespace, "Deployment.Name", dbInstance.Name)
		found.Spec = deployment.Spec
		if err := r.Update(ctx, found); err != nil {
			logger.Error(err, "更新 Deployment 失败")
			return ctrl.Result{}, err
		}
	}

	// 构造期望的 Service 对象
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dbInstance.Name,
			Namespace: dbInstance.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": dbInstance.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       3306, // 可以修改成实际的数据库端口
					TargetPort: intstr.FromInt(3306),
				},
			},
		},
	}

	// 确保 Service 存在
	foundService := &corev1.Service{}
	err = r.Get(ctx, client.ObjectKey{Name: dbInstance.Name, Namespace: dbInstance.Namespace}, foundService)
	if err != nil && errors.IsNotFound(err) {
		// Service 不存在，创建它
		logger.Info("创建一个新的 Service", "Service.Namespace", dbInstance.Namespace, "Service.Name", dbInstance.Name)
		if err := r.Create(ctx, service); err != nil {
			logger.Error(err, "新的 Service 创建失败")
			return ctrl.Result{}, err
		}
	} else if err != nil {
		logger.Error(err, "获取 Service 失败")
		return ctrl.Result{}, err
	} else {
		// Service 存在，更新它
		// 更新 Service 时，排除不可以更改的字段如 ClusterIP、Type 等字段，只更新可以更改的字段
		updatedService := foundService.DeepCopy()
		updatedService.Spec.Ports = service.Spec.Ports
		updatedService.Spec.Selector = service.Spec.Selector

		logger.Info("更新 Service", "Service.Namespace", dbInstance.Namespace, "Service.Name", dbInstance.Name)
		if err := r.Update(ctx, updatedService); err != nil {
			logger.Error(err, "更新 Service 失败")
			return ctrl.Result{}, err
		}
	}

	// 更新 DatabaseInstance 状态
	dbInstance.Status = databasev1.DatabaseInstanceStatus{
		Phase:         "Running",    // 根据实际情况设置
		Message:       "数据库实例正在运行中", // 根据实际情况设置
		ReadyReplicas: 1,            // 根据实际情况设置
		LastUpdated:   metav1.Now(), // 设置为当前时间
		Conditions: []databasev1.DatabaseInstanceCondition{
			{
				Type:               "Ready",
				Status:             "True",
				LastTransitionTime: metav1.Now(),
				Reason:             "Deployment completed successfully",
				Message:            "数据库实例已成功部署完成",
			},
		},
	}

	if err := r.Status().Update(ctx, &dbInstance); err != nil {
		logger.Error(err, "更新 DatabaseInstance 状态失败")
		return ctrl.Result{}, err
	}

	// 返回 Reconcile 结果
	return ctrl.Result{}, nil
}

// SetupWithManager 将控制器与 Manager 管理器进行配置和绑定
// 通过这种配置，我们自定义的控制器 DatabaseInstanceReconciler 就能够获取到 DatabaseInstance 自定义资源的状态变化事件通知
// 并根据这些通知执行 Reconcile 方法来调整资源的状态，完成调节的动作
func (r *DatabaseInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasev1.DatabaseInstance{}).
		Complete(r)
}
