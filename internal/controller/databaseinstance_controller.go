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

	"k8s.io/apimachinery/pkg/api/errors"

	// 导入 Kubernetes 的 apps/v1
	// 导入 Kubernetes 的 core/v1 包
	// 错误处理
	// 导入 metav1 包
	"k8s.io/apimachinery/pkg/runtime"

	// 导入 intstr 包
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	databasev1 "git.ucmed.cn/pdd/k8s-database-operator/api/v1"    // 导入 databasev1
	"git.ucmed.cn/pdd/k8s-database-operator/internal/pkg/helpers" // 辅助函数
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
	logger.Info("开始处理 Reconcile", "资源名称", req.NamespacedName)

	// 获取当前的 DatabaseInstance 实例
	var dbInstance databasev1.DatabaseInstance
	if err := r.Get(ctx, req.NamespacedName, &dbInstance); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("未找到 DatabaseInstance", "资源名称", req.NamespacedName)
			return ctrl.Result{}, nil // 资源未找到，不需要返回错误
		}
		logger.Error(err, "获取 DatabaseInstance 失败", "资源名称", req.NamespacedName)
		return ctrl.Result{}, err
	}

	// 生成镜像名称
	image := helpers.GenerateImageName(dbInstance.Spec.Image, dbInstance.Spec.DatabaseType, dbInstance.Spec.Version)

	// 提取参数
	instanceName := dbInstance.Name
	namespace := dbInstance.Namespace
	replicas := dbInstance.Spec.Replicas
	databaseType := dbInstance.Spec.DatabaseType

	// 处理 Secret
	secretName := instanceName + "-secret"
	_, err := helpers.GetOrCreateSecret(ctx, r.Client, secretName, namespace, databaseType)
	if err != nil {
		return ctrl.Result{}, err
	}

	// 创建或更新 Deployment
	deployment := helpers.NewDeployment(instanceName, namespace, image, replicas, databaseType)
	if err := helpers.EnsureDeployment(ctx, r.Client, deployment); err != nil {
		return ctrl.Result{}, err
	}

	// 创建或更新 Service
	service := helpers.NewService(instanceName, namespace, databaseType)
	if err := helpers.EnsureService(ctx, r.Client, service); err != nil {
		return ctrl.Result{}, err
	}

	// 创建或更新 CronJob
	if dbInstance.Spec.BackupPolicy.Enabled {
		// 获取备份镜像，如果 BackupPolicy 中指定了镜像，则使用它，否则使用 dbInstance.Spec.BackupImage
		backupImage := dbInstance.Spec.BackupPolicy.BackupImage
		if backupImage == "" {
			backupImage = dbInstance.Spec.BackupImage // 如果 BackupImage 为空，则使用默认的备份镜像
		}

		cronJob := helpers.NewCronJob(
			instanceName+"-backup",
			namespace,
			backupImage, // 使用备份镜像
			dbInstance.Spec.BackupPolicy.Schedule,
			databaseType,
		)
		if err := helpers.EnsureCronJob(ctx, r.Client, cronJob); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		// 如果备份策略未启用，确保没有存在的 CronJob
		if err := helpers.DeleteCronJob(ctx, r.Client, instanceName+"-backup", namespace); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 更新 DatabaseInstance 状态
	if err := helpers.UpdateDatabaseInstanceStatus(ctx, r.Client, &dbInstance); err != nil {
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
