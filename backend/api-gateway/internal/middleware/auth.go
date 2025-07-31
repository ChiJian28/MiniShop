package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"api-gateway/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

// AuthMiddleware 认证中间件
type AuthMiddleware struct {
	config *config.AuthConfig
	logger *logrus.Logger
}

// NewAuthMiddleware 创建认证中间件
func NewAuthMiddleware(cfg *config.AuthConfig, logger *logrus.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		config: cfg,
		logger: logger,
	}
}

// JWTAuth JWT认证中间件
func (am *AuthMiddleware) JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !am.config.Enable {
			c.Next()
			return
		}

		// 检查白名单
		if am.isWhitelisted(c.Request.URL.Path) {
			c.Next()
			return
		}

		// 获取token
		token := am.getTokenFromRequest(c)
		if token == "" {
			am.logger.WithFields(logrus.Fields{
				"ip":   c.ClientIP(),
				"path": c.Request.URL.Path,
			}).Warn("缺少认证token")

			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "缺少认证token",
				"code":  401,
			})
			c.Abort()
			return
		}

		// 验证token
		claims, err := am.validateJWT(token)
		if err != nil {
			am.logger.WithFields(logrus.Fields{
				"ip":    c.ClientIP(),
				"path":  c.Request.URL.Path,
				"error": err.Error(),
			}).Warn("token验证失败")

			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "token验证失败",
				"code":  401,
			})
			c.Abort()
			return
		}

		// 设置用户信息到上下文
		c.Set("user_id", claims["user_id"])
		c.Set("username", claims["username"])
		c.Set("roles", claims["roles"])

		c.Next()
	}
}

// SignatureAuth 签名校验中间件
func (am *AuthMiddleware) SignatureAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !am.config.Signature.Enable {
			c.Next()
			return
		}

		// 检查白名单
		if am.isWhitelisted(c.Request.URL.Path) {
			c.Next()
			return
		}

		// 验证签名
		if err := am.validateSignature(c); err != nil {
			am.logger.WithFields(logrus.Fields{
				"ip":    c.ClientIP(),
				"path":  c.Request.URL.Path,
				"error": err.Error(),
			}).Warn("签名验证失败")

			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "签名验证失败",
				"code":  401,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// getTokenFromRequest 从请求中获取token
func (am *AuthMiddleware) getTokenFromRequest(c *gin.Context) string {
	// 从Authorization header获取
	auth := c.GetHeader("Authorization")
	if auth != "" {
		// Bearer token格式
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
		return auth
	}

	// 从查询参数获取
	if token := c.Query("token"); token != "" {
		return token
	}

	// 从cookie获取
	if token, err := c.Cookie("token"); err == nil {
		return token
	}

	return ""
}

// validateJWT 验证JWT token
func (am *AuthMiddleware) validateJWT(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(am.config.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// 检查过期时间
		if exp, ok := claims["exp"]; ok {
			if expTime, ok := exp.(float64); ok {
				if time.Now().Unix() > int64(expTime) {
					return nil, fmt.Errorf("token已过期")
				}
			}
		}

		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// validateSignature 验证请求签名
func (am *AuthMiddleware) validateSignature(c *gin.Context) error {
	// 获取必需的header
	timestamp := c.GetHeader("timestamp")
	nonce := c.GetHeader("nonce")
	signature := c.GetHeader("signature")

	// 检查必需的header
	for _, header := range am.config.Signature.RequiredHeaders {
		if c.GetHeader(header) == "" {
			return fmt.Errorf("缺少必需的header: %s", header)
		}
	}

	// 验证时间戳
	if timestamp == "" {
		return fmt.Errorf("缺少timestamp")
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp format")
	}

	// 检查时间戳是否在有效期内
	now := time.Now().Unix()
	if now-ts > int64(am.config.Signature.Expire.Seconds()) {
		return fmt.Errorf("请求已过期")
	}

	// 验证nonce
	if nonce == "" {
		return fmt.Errorf("缺少nonce")
	}

	// 验证签名
	if signature == "" {
		return fmt.Errorf("缺少signature")
	}

	expectedSignature := am.generateSignature(c, timestamp, nonce)
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return fmt.Errorf("签名不匹配")
	}

	return nil
}

// generateSignature 生成签名
func (am *AuthMiddleware) generateSignature(c *gin.Context, timestamp, nonce string) string {
	// 收集所有参数
	params := make(map[string]string)

	// 添加URL参数
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}

	// 添加form参数（如果是POST请求）
	if c.Request.Method == "POST" {
		c.Request.ParseForm()
		for key, values := range c.Request.PostForm {
			if len(values) > 0 {
				params[key] = values[0]
			}
		}
	}

	// 添加固定参数
	params["timestamp"] = timestamp
	params["nonce"] = nonce
	params["method"] = c.Request.Method
	params["path"] = c.Request.URL.Path

	// 按key排序
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// 构建签名字符串
	var signStr strings.Builder
	for i, key := range keys {
		if i > 0 {
			signStr.WriteString("&")
		}
		signStr.WriteString(fmt.Sprintf("%s=%s", key, params[key]))
	}

	// 添加secret
	signStr.WriteString("&secret=")
	signStr.WriteString(am.config.Signature.Secret)

	// 生成HMAC-SHA256签名
	h := hmac.New(sha256.New, []byte(am.config.Signature.Secret))
	h.Write([]byte(signStr.String()))
	return hex.EncodeToString(h.Sum(nil))
}

// isWhitelisted 检查路径是否在白名单中
func (am *AuthMiddleware) isWhitelisted(path string) bool {
	for _, whitePath := range am.config.Whitelist {
		if strings.HasPrefix(path, whitePath) {
			return true
		}
	}
	return false
}

// GenerateJWT 生成JWT token
func (am *AuthMiddleware) GenerateJWT(userID interface{}, username string, roles []string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"roles":    roles,
		"iat":      now.Unix(),
		"exp":      now.Add(am.config.TokenExpire).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(am.config.JWTSecret))
}

// GenerateRefreshToken 生成刷新token
func (am *AuthMiddleware) GenerateRefreshToken(userID interface{}) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"user_id": userID,
		"type":    "refresh",
		"iat":     now.Unix(),
		"exp":     now.Add(am.config.RefreshExpire).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(am.config.JWTSecret))
}

// RefreshToken 刷新token
func (am *AuthMiddleware) RefreshToken(refreshToken string) (string, error) {
	// 验证刷新token
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(am.config.JWTSecret), nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// 检查是否是刷新token
		if tokenType, ok := claims["type"].(string); !ok || tokenType != "refresh" {
			return "", fmt.Errorf("invalid refresh token")
		}

		// 生成新的访问token
		userID := claims["user_id"]
		return am.GenerateJWT(userID, "", []string{})
	}

	return "", fmt.Errorf("invalid refresh token")
}
