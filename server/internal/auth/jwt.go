// Package auth 实现 JWT 令牌的签发与校验。
//
// 关键契约（前端与既有数据依赖）：
//   - HS256 对称签名，密钥来自配置 JWT_ACCESS_SECRET / JWT_REFRESH_SECRET
//   - access / refresh 使用**不同**密钥
//   - 业务声明全部嵌套在 extend 对象内，无顶层 id/username
//   - access 有效期 28800s（8 小时，沿用上游契约值）
//   - exp/nbf 允许 60s 时钟偏差
//   - Authorization 头解析比常规更严格：恰好一个空格、恰好两个字面 Bearer
package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/config"
)

const (
	// Issuer 与上游 SaiAdmin 6.x 签发者保持一致（旧令牌仍在有效期内时不会因此失效）
	Issuer = "webman.tinywan.cn"

	// AccessExp access 令牌有效期（8 小时），沿用上游契约值
	AccessExp = 8 * 60 * 60
	// RefreshExp refresh 令牌有效期（7 天）
	RefreshExp = 604800

	// Leeway 时钟偏差冗余
	Leeway = 60 * time.Second
)

// accessSecret / refreshSecret 从配置读取；两者都必须在启动前配好
func accessSecret() []byte  { return []byte(config.C.JWTAccessSecret) }
func refreshSecret() []byte { return []byte(config.C.JWTRefreshSecret) }

// Extend 业务声明，必须嵌套在 extend 下
type Extend struct {
	AccessExp int    `json:"access_exp"`
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Type      string `json:"type"`
	Plat      string `json:"plat"`
}

// Claims 完整载荷
type Claims struct {
	Extend Extend `json:"extend"`
	jwt.RegisteredClaims
}

// TokenPair 登录响应体（字段名与前端 Api.Auth.LoginResponse 一致）
type TokenPair struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func baseClaims(exp int) jwt.RegisteredClaims {
	now := time.Now()
	return jwt.RegisteredClaims{
		Issuer:    Issuer,
		Audience:  jwt.ClaimStrings{Issuer},
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(exp) * time.Second)),
	}
}

// Generate 签发 access + refresh 令牌
func Generate(id int, username, loginType string) (TokenPair, error) {
	ext := Extend{
		AccessExp: AccessExp,
		ID:        id,
		Username:  username,
		Type:      loginType,
		Plat:      "saiadmin",
	}

	accessTok := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		Extend:           ext,
		RegisteredClaims: baseClaims(AccessExp),
	})
	accessStr, err := accessTok.SignedString(accessSecret())
	if err != nil {
		return TokenPair{}, err
	}

	refreshTok := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		Extend:           ext,
		RegisteredClaims: baseClaims(RefreshExp),
	})
	refreshStr, err := refreshTok.SignedString(refreshSecret())
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		TokenType:    "Bearer",
		ExpiresIn:    AccessExp,
		AccessToken:  accessStr,
		RefreshToken: refreshStr,
	}, nil
}

// ErrInvalidToken 所有解析失败统一归一到该错误，
// 以便中间件输出与上游 SaiAdmin 6.x 完全一致的 401 提示。
var ErrInvalidToken = errors.New("invalid token")

// Parse 校验并解析 access 令牌。
// 契约：不校验 aud（上游同样未传 expected audience），仅校验签名与时间。
func Parse(tokenStr string) (*Extend, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithLeeway(Leeway),
	)
	claims := &Claims{}
	_, err := parser.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return accessSecret(), nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}
	return &claims.Extend, nil
}

// ExtractToken 解析 Authorization 头，规则与上游 SaiAdmin 6.x 一致（比常规更严格）：
//   - 非空，且不等于字面量 "undefined"
//   - 恰好包含 2 个 '.'（JWS 三段）
//   - 按单个空格切分恰好得到 2 段
//   - 首段必须**大小写敏感**地等于 "Bearer"
//   - 次段非空且不等于 "undefined"
func ExtractToken(header string) (string, error) {
	if header == "" || header == "undefined" {
		return "", ErrInvalidToken
	}
	if strings.Count(header, ".") != 2 {
		return "", ErrInvalidToken
	}
	parts := strings.Split(header, " ")
	if len(parts) != 2 {
		return "", ErrInvalidToken
	}
	if parts[0] != "Bearer" {
		return "", ErrInvalidToken
	}
	if parts[1] == "" || parts[1] == "undefined" {
		return "", ErrInvalidToken
	}
	return parts[1], nil
}

// HashPassword 生成 bcrypt 哈希，并将前缀规范为 $2y$
func HashPassword(pw string) (string, error) {
	h, err := bcryptHash(pw)
	if err != nil {
		return "", err
	}
	// Go 产出 $2a$，而 PHP 的 password_hash 产出 $2y$；两者算法等价，
	// 统一为 $2y$ 以便与上游 SaiAdmin 6.x 的历史数据互相校验/迁移。
	if strings.HasPrefix(h, "$2a$") {
		h = "$2y$" + h[4:]
	}
	return h, nil
}

// CheckPassword 校验密码。库中历史哈希为 $2y$ 前缀，Go 的 bcrypt 可直接校验。
func CheckPassword(hash, pw string) bool {
	return bcryptCompare(hash, pw)
}
