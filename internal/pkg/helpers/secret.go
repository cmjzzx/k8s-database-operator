// internal/pkg/helpers/secret.go
package helpers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// 生成随机密码
func GenerateRandomPassword(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}

// 获取或创建 Secret
func GetOrCreateSecret(ctx context.Context, c client.Client, name, namespace, databaseType string) (*corev1.Secret, error) {
	logger := log.FromContext(ctx)

	// 为不同数据库系统定义密钥名称
	var userKey, passwordKey string
	switch databaseType {
	case "postgres":
		userKey = "postgres-user"
		passwordKey = "postgres-password"
	case "oceanbase-ce":
		userKey = "oceanbase-user"
		passwordKey = "oceanbase-password"
	case "mysql":
		userKey = "mysql-user"
		passwordKey = "mysql-password"
	default:
		// 对于不支持的数据库类型
		return nil, fmt.Errorf("unsupported database type: %s", databaseType)
	}

	secretName := databaseType + "-secret"

	// 尝试获取现有 Secret
	existingSecret := &corev1.Secret{}
	err := c.Get(ctx, client.ObjectKey{Name: secretName, Namespace: namespace}, existingSecret)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "获取 Secret 失败")
			return nil, err
		}

		// Secret 不存在，创建新的 Secret
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
