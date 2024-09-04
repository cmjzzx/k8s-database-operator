package helpers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrl "sigs.k8s.io/controller-runtime/pkg/log"
)

// getServiceConfig 根据数据库类型获取服务端口和端口名称
func getServiceConfig(databaseType string) (int32, string) {
	switch databaseType {
	case "mysql":
		return 3306, "mysql"
	case "postgres":
		return 5432, "postgres"
	case "oceanbase-ce":
		return 2881, "oceanbase-ce"
	default:
		return 3306, "mysql" // 默认值
	}
}

// NewService 创建一个新的 Service 对象
func NewService(name, namespace, databaseType string) *corev1.Service {
	labels := map[string]string{
		"app": name,
	}

	servicePort, portName := getServiceConfig(databaseType)

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Port:       servicePort,
					TargetPort: intstr.FromInt(int(servicePort)),
					Name:       portName,
				},
			},
		},
	}
}

// EnsureService 确保 Service 存在并更新
func EnsureService(ctx context.Context, c client.Client, service *corev1.Service) error {
	logger := ctrl.FromContext(ctx)

	found := &corev1.Service{}
	err := c.Get(ctx, client.ObjectKey{Name: service.Name, Namespace: service.Namespace}, found)
	if err != nil && client.IgnoreNotFound(err) == nil {
		// Service 不存在，创建它
		logger.Info("创建一个新的 Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
		if err := c.Create(ctx, service); err != nil {
			logger.Error(err, "新的 Service 创建失败")
			return err
		}
	} else if err != nil {
		logger.Error(err, "获取 Service 失败")
		return err
	} else {
		// Service 存在，更新它
		updatedService := found.DeepCopy()
		updatedService.Spec.Ports = service.Spec.Ports
		updatedService.Spec.Selector = service.Spec.Selector

		logger.Info("更新 Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
		if err := c.Update(ctx, updatedService); err != nil {
			logger.Error(err, "更新 Service 失败")
			return err
		}
	}
	return nil
}
