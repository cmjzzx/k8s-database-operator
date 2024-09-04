package helpers

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrl "sigs.k8s.io/controller-runtime/pkg/log"
)

// getCronJobConfig 根据数据库类型获取命令和环境变量
func getCronJobConfig(name, databaseType string) ([]string, []corev1.EnvVar) {
	switch databaseType {
	case "mysql":
		return []string{"sh", "-c", "mysqldump -h $DB_HOST -P $DB_PORT -u$MYSQL_USER -p$MYSQL_PASSWORD --all-databases > /backup/db-backup.sql"},
			[]corev1.EnvVar{
				{
					Name: "MYSQL_USER",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "mysql-secret",
							},
							Key: "mysql-user",
						},
					},
				},
				{
					Name: "MYSQL_PASSWORD",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "mysql-secret",
							},
							Key: "mysql-password",
						},
					},
				},
				{
					Name:  "DB_HOST",
					Value: name,
				},
				{
					Name:  "DB_PORT",
					Value: "3306", // MySQL 默认端口
				},
			}
	case "postgres":
		return []string{"sh", "-c", "pg_dumpall -h $DB_HOST -p $DB_PORT -U $POSTGRES_USER > /backup/db-backup.sql"},
			[]corev1.EnvVar{
				{
					Name: "POSTGRES_USER",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "postgres-secret",
							},
							Key: "postgres-user",
						},
					},
				},
				{
					Name: "POSTGRES_PASSWORD",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "postgres-secret",
							},
							Key: "postgres-password",
						},
					},
				},
				{
					Name:  "DB_HOST",
					Value: name,
				},
				{
					Name:  "DB_PORT",
					Value: "5432", // PostgreSQL 默认端口
				},
			}
	case "oceanbase-ce":
		return []string{"sh", "-c", "obclient -h $DB_HOST -P $DB_PORT -u $OBD_USER -p$OBD_PASSWORD -e \"BACKUP DATABASE TO '/backup/backup.sql'\""},
			[]corev1.EnvVar{
				{
					Name: "OBD_USER",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "oceanbase-ce-secret",
							},
							Key: "oceanbase-user",
						},
					},
				},
				{
					Name: "OBD_PASSWORD",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "oceanbase-ce-secret",
							},
							Key: "oceanbase-password",
						},
					},
				},
				{
					Name:  "DB_HOST",
					Value: name,
				},
				{
					Name:  "DB_PORT",
					Value: "2881", // OceanBase-CE 默认端口
				},
			}
	default:
		return []string{"sh", "-c", "echo 'Unsupported database type'"}, nil
	}
}

// NewCronJob 根据数据库类型创建 CronJob
func NewCronJob(name, namespace, image, schedule, databaseType string) *batchv1.CronJob {
	labels := map[string]string{
		"app": name,
	}

	command, envVars := getCronJobConfig(name, databaseType)

	// 定义 CronJob
	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: batchv1.CronJobSpec{
			Schedule: schedule,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: labels,
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:    name,
									Image:   image,
									Command: command,
									Env:     envVars,
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "backup-volume",
											MountPath: "/backup",
										},
									},
								},
							},
							RestartPolicy: corev1.RestartPolicyOnFailure,
							Volumes: []corev1.Volume{
								{
									Name: "backup-volume",
									VolumeSource: corev1.VolumeSource{
										// 使用存储卷声明来实现挂载，需事先完成 backup-pvc 的创建，不会自动创建 PVC
										// 事先进行 PVC 的创建时，需确保已经创建了相应的存储类，否则 PVC 和 PV 无法创建成功
										PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
											ClaimName: "backup-pvc",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// EnsureCronJob 确保 CronJob 资源存在，如果不存在则创建，如果存在则更新
func EnsureCronJob(ctx context.Context, c client.Client, desired *batchv1.CronJob) error {
	logger := ctrl.FromContext(ctx)

	var existing batchv1.CronJob
	err := c.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, &existing)
	if err != nil {
		if errors.IsNotFound(err) {
			// 如果 CronJob 不存在，则创建
			logger.Info("创建一个新的 CronJob", "CronJob.Namespace", desired.Namespace, "CronJob.Name", desired.Name)
			if err := c.Create(ctx, desired); err != nil {
				logger.Error(err, "新的 CronJob 创建失败")
				return err
			}
		} else {
			// 如果出现其他错误，返回错误
			logger.Error(err, "获取 CronJob 失败")
			return err
		}
	} else {
		// 如果 CronJob 已存在，则更新
		logger.Info("更新已有的 CronJob", "CronJob.Namespace", desired.Namespace, "CronJob.Name", desired.Name)
		existing.Spec = desired.Spec
		if err := c.Update(ctx, &existing); err != nil {
			logger.Error(err, "更新 CronJob 失败")
			return err
		}
	}
	return nil
}

// DeleteCronJob 删除指定的 CronJob 资源
func DeleteCronJob(ctx context.Context, c client.Client, name, namespace string) error {
	logger := ctrl.FromContext(ctx)

	// 创建一个 CronJob 对象
	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	// 尝试获取指定的 CronJob
	if err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, cronJob); err != nil {
		if errors.IsNotFound(err) {
			// 如果 CronJob 不存在，返回 nil，不做任何操作
			logger.Info("CronJob 不存在", "CronJob.Namespace", namespace, "CronJob.Name", name)
			return nil
		}
		// 如果出现其他错误，返回错误
		logger.Error(err, "获取 CronJob 失败")
		return err
	}

	// 删除 CronJob
	if err := c.Delete(ctx, cronJob); err != nil {
		// 返回删除失败的错误
		logger.Error(err, "删除 CronJob 失败")
		return err
	}

	logger.Info("成功删除 CronJob", "CronJob.Namespace", namespace, "CronJob.Name", name)
	return nil
}
