package helpers

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewCronJob 根据数据库类型创建 CronJob
func NewCronJob(name, namespace, image, schedule, databaseType string) *batchv1.CronJob {
	labels := map[string]string{
		"app": name,
	}

	// 根据数据库类型设置相应的命令和环境变量
	var command []string
	var envVars []corev1.EnvVar

	switch databaseType {
	case "mysql":
		command = []string{"sh", "-c", "mysqldump -h $DB_HOST -P $DB_PORT -u$MYSQL_USER -p$MYSQL_PASSWORD --all-databases > /backup/db-backup.sql"}
		envVars = []corev1.EnvVar{
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
		command = []string{"sh", "-c", "pg_dumpall -h $DB_HOST -p $DB_PORT -U $POSTGRES_USER > /backup/db-backup.sql"}
		envVars = []corev1.EnvVar{
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
		// 使用 obclient 执行 SQL 命令备份数据库
		// ！！！命令不一定正确，请注意！！！
		command = []string{"sh", "-c", "obclient -h $DB_HOST -P $DB_PORT -u $OBD_USER -p$OBD_PASSWORD -e \"BACKUP DATABASE TO '/backup/backup.sql'\""}
		envVars = []corev1.EnvVar{
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
		command = []string{"sh", "-c", "echo 'Unsupported database type'"}
	}

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
	var existing batchv1.CronJob
	err := c.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, &existing)
	if err != nil {
		if errors.IsNotFound(err) {
			// 如果 CronJob 不存在，则创建
			return c.Create(ctx, desired)
		}
		// 如果出现其他错误，返回错误
		return err
	}

	// 如果 CronJob 已存在，则更新
	existing.Spec = desired.Spec
	return c.Update(ctx, &existing)
}

// DeleteCronJob 删除指定的 CronJob 资源
func DeleteCronJob(ctx context.Context, c client.Client, name, namespace string) error {
	// 创建一个 CronJob 对象
	cronJob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	// 尝试获取指定的 CronJob
	if err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, cronJob); err != nil {
		if errors.IsNotFound(err) {
			// 如果 CronJob 不存在，返回 nil，不做任何操作
			return nil
		}
		// 如果出现其他错误，返回错误
		return err
	}

	// 删除 CronJob
	if err := c.Delete(ctx, cronJob); err != nil {
		// 返回删除失败的错误
		return err
	}

	return nil
}
