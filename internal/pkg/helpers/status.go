package helpers

import (
	"context"

	databasev1 "github.com/cmjzzx/k8s-database-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrl "sigs.k8s.io/controller-runtime/pkg/log"
)

// createDatabaseInstanceCondition 创建 DatabaseInstanceCondition
func createDatabaseInstanceCondition() databasev1.DatabaseInstanceCondition {
	return databasev1.DatabaseInstanceCondition{
		Type:               "Ready",
		Status:             "True",
		LastTransitionTime: metav1.Now(),
		Reason:             "Deployment completed successfully",
		Message:            "数据库实例已成功部署完成",
	}
}

// UpdateDatabaseInstanceStatus 更新 DatabaseInstance 的状态
func UpdateDatabaseInstanceStatus(ctx context.Context, c client.Client, dbInstance *databasev1.DatabaseInstance) error {
	logger := ctrl.FromContext(ctx)

	dbInstance.Status = databasev1.DatabaseInstanceStatus{
		Phase:         "Running",    // 根据实际情况设置
		Message:       "数据库实例正在运行中", // 根据实际情况设置
		ReadyReplicas: 1,            // 根据实际情况设置
		LastUpdated:   metav1.Now(), // 设置为当前时间
		Conditions: []databasev1.DatabaseInstanceCondition{
			createDatabaseInstanceCondition(),
		},
	}

	if err := c.Status().Update(ctx, dbInstance); err != nil {
		logger.Error(err, "更新 DatabaseInstance 状态失败", "DatabaseInstance.Namespace", dbInstance.Namespace, "DatabaseInstance.Name", dbInstance.Name)
		return err
	}

	logger.Info("成功更新 DatabaseInstance 状态", "DatabaseInstance.Namespace", dbInstance.Namespace, "DatabaseInstance.Name", dbInstance.Name)
	return nil
}
