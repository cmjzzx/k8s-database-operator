package helpers

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrl "sigs.k8s.io/controller-runtime/pkg/log"
)

// GenerateImageName 生成完整的镜像名称
func GenerateImageName(baseImage, databaseType, version string) string {
	const imagePrefix = "registry.zwjk.com/middleware/"
	image := baseImage
	if image == "" {
		image = fmt.Sprintf("%s:%s", databaseType, version)
	}
	return fmt.Sprintf("%s%s", imagePrefix, image)
}

// NewDeployment 创建一个新的 Deployment 对象
func NewDeployment(name, namespace, image string, replicas int32, databaseType string) *appsv1.Deployment {
	labels := map[string]string{
		"app": name,
	}

	// 根据数据库类型设置端口和端口名称
	var containerPort int32
	var portName string

	switch databaseType {
	case "mysql":
		containerPort = 3306
		portName = "mysql"
	case "postgres":
		containerPort = 5432
		portName = "postgres"
	case "oceanbase-ce":
		containerPort = 2881
		portName = "oceanbase-ce"
	default:
		containerPort = 3306
		portName = "mysql" // 默认值
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  name,
							Image: image,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: containerPort,
									Name:          portName,
								},
							},
						},
					},
				},
			},
		},
	}
}

// EnsureDeployment 确保 Deployment 存在并更新
func EnsureDeployment(ctx context.Context, c client.Client, deployment *appsv1.Deployment) error {
	logger := ctrl.FromContext(ctx)

	found := &appsv1.Deployment{}
	err := c.Get(ctx, client.ObjectKey{Name: deployment.Name, Namespace: deployment.Namespace}, found)
	if err != nil && client.IgnoreNotFound(err) == nil {
		// Deployment 不存在，创建它
		logger.Info("创建一个新的 Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		if err := c.Create(ctx, deployment); err != nil {
			logger.Error(err, "新的 Deployment 创建失败")
			return err
		}
	} else if err != nil {
		logger.Error(err, "获取 Deployment 失败")
		return err
	} else {
		// Deployment 存在，更新它
		logger.Info("更新已有的 Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		found.Spec = deployment.Spec
		if err := c.Update(ctx, found); err != nil {
			logger.Error(err, "更新 Deployment 失败")
			return err
		}
	}
	return nil
}
