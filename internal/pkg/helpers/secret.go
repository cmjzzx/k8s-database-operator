// internal/pkg/helpers/secret.go
package helpers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// GenerateRandomPassword 生成随机密码
func GenerateRandomPassword(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}

// getSecretKeys 根据数据库类型获取用户名和密码的键名
func getSecretKeys(ctx context.Context, databaseType string) (string, string, error) {
	logger := log.FromContext(ctx)
	switch databaseType {
	case "postgres":
		return "postgres-user", "postgres-password", nil
	case "oceanbase-ce":
		return "oceanbase-user", "oceanbase-password", nil
	case "mysql":
		return "mysql-user", "mysql-password", nil
	default:
		err := errors.New("unsupported database type: " + databaseType)
		logger.Error(err, "不支持的数据库类型", "databaseType", databaseType)
		return "", "", err
	}
}

// createNewSecret 创建新的 Secret
func createNewSecret(ctx context.Context, c client.Client, name, namespace, secretName, userKey, passwordKey string) (*corev1.Secret, error) {
	logger := log.FromContext(ctx)

	user, err := GenerateRandomPassword(8) // 生成 8 字节的随机用户名
	if err != nil {
		logger.Error(err, "生成用户名失败")
		return nil, err
	}

	password, err := GenerateRandomPassword(16) // 生成 16 字节的随机密码
	if err != nil {
		logger.Error(err, "生成密码失败")
		return nil, err
	}

	newSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels:    map[string]string{"app": name},
		},
		Data: map[string][]byte{
			userKey:     []byte(user),
			passwordKey: []byte(password),
		},
	}
	if err := c.Create(ctx, newSecret); err != nil {
		logger.Error(err, "创建 Secret 失败")
		return nil, err
	}
	return newSecret, nil
}

// GetOrCreateSecret 获取或创建 Secret
func GetOrCreateSecret(ctx context.Context, c client.Client, name, namespace, databaseType string) (*corev1.Secret, error) {
	logger := log.FromContext(ctx)

	userKey, passwordKey, err := getSecretKeys(ctx, databaseType)
	if err != nil {
		return nil, err
	}

	secretName := databaseType + "-secret"

	// 尝试获取现有 Secret
	existingSecret := &corev1.Secret{}
	err = c.Get(ctx, client.ObjectKey{Name: secretName, Namespace: namespace}, existingSecret)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "获取 Secret 失败")
			return nil, err
		}

		// Secret 不存在，创建新的 Secret
		return createNewSecret(ctx, c, name, namespace, secretName, userKey, passwordKey)
	}

	// Secret 已存在，只更新其他字段
	if _, ok := existingSecret.Data[userKey]; !ok {
		// 如果 Secret 中没有对应的用户名字段，生成并添加
		user, err := GenerateRandomPassword(8) // 生成 8 字节的随机用户名
		if err != nil {
			logger.Error(err, "生成用户名失败")
			return nil, err
		}
		existingSecret.Data[userKey] = []byte(user)
	}

	if _, ok := existingSecret.Data[passwordKey]; !ok {
		// 如果 Secret 中没有对应的密码字段，生成并添加
		password, err := GenerateRandomPassword(16) // 生成 16 字节的随机密码
		if err != nil {
			logger.Error(err, "生成密码失败")
			return nil, err
		}
		existingSecret.Data[passwordKey] = []byte(password)
	}

	if err := c.Update(ctx, existingSecret); err != nil {
		logger.Error(err, "更新 Secret 失败")
		return nil, err
	}

	return existingSecret, nil
}
