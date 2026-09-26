package auth

import "golang.org/x/crypto/bcrypt"

// bcryptHash 生成 bcrypt 哈希。cost 取 10：PHP 的 PASSWORD_DEFAULT 即 bcrypt(cost=10)，
// 与库中历史哈希保持一致。
func bcryptHash(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// bcryptCompare 校验 bcrypt 哈希。
// Go 的 bcrypt 能正确处理 $2y$ 前缀（PHP password_hash 的产物），
// 因此可直接校验库中既有密码。
func bcryptCompare(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}
