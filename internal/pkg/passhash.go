// Package pkg 放置与业务无关的通用工具。
package pkg

import "golang.org/x/crypto/bcrypt"

// HashPassword 生成 bcrypt 密码哈希。
func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(hash), err
}

// CheckPassword 校验明文与 bcrypt 哈希是否匹配。
func CheckPassword(plain, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
