package auth

import (
	"fmt"
	"regexp"
	"strings"

	"chatroom/database"
)

type AuthResult struct {
	Success bool           //操作是否成功
	Message string         //结果消息（用户友好）
	User    *database.User //用户信息（成功时返回）
}

var (
	// 用户名：3-20位字母、数字、下划线
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)

	// 昵称：2-20位中文、字母、数字、下划线
	// \p{Han} 匹配所有中文字符
	// 防止过长输入导致数据库溢出
	nicknameRegex = regexp.MustCompile(`^[\p{Han}a-zA-Z0-9_]{2,20}$`)
)

// 用户注册
func Register(username, password, nickname string) AuthResult {
	// 输入验证
	// MatchString：用于检查字符串是否匹配预先编译好的正则表达式模式
	if !usernameRegex.MatchString(username) {
		return AuthResult{
			Success: false,
			Message: "用户名必须为3-20位的字母、数字或下划线",
		}
	}

	if len(password) < 6 {
		return AuthResult{
			Success: false,
			Message: "密码长度至少6个字符",
		}
	}

	if !nicknameRegex.MatchString(nickname) {
		return AuthResult{
			Success: false,
			Message: "昵称必须为2-20位的中文、字母、数字或下划线",
		}
	}

	// 检查用户名是否已存在（为了不起名重复）
	exists, err := database.UsernameExists(username)
	if err != nil {
		return AuthResult{
			Success: false,
			Message: "系统错误，请稍后重试",
		}
	}
	if exists {
		return AuthResult{
			Success: false,
			Message: fmt.Sprintf("用户名 '%s' 已被使用", username),
		}
	}

	// 检查昵称是否已存在
	exists, err = database.NicknameExists(nickname)
	if err != nil {
		return AuthResult{
			Success: false,
			Message: "系统错误，请稍后重试",
		}
	}
	if exists {
		return AuthResult{
			Success: false,
			Message: fmt.Sprintf("昵称 '%s' 已被使用", nickname),
		}
	}

	// 创建用户
	err = database.RegisterUser(username, password, nickname)
	if err != nil {
		return AuthResult{
			Success: false,
			Message: "注册失败，请稍后重试",
		}
	}

	// 获取用户信息
	user, err := database.GetUserByUsername(username)
	if err != nil {
		return AuthResult{
			Success: false,
			Message: "注册成功，但获取用户信息失败",
		}
	}

	return AuthResult{
		Success: true,
		Message: "🎉 注册成功！",
		User:    user,
	}
}

// 用户登录
func Login(username, password string) AuthResult {
	// 输入验证
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)

	if username == "" || password == "" {
		return AuthResult{
			Success: false,
			Message: "用户名和密码不能为空",
		}
	}

	// 验证用户
	user, err := database.LoginUser(username, password)
	if err != nil {
		return AuthResult{
			Success: false,
			Message: "用户名或密码错误",
		}
	}

	return AuthResult{
		Success: true,
		Message: "✅ 登录成功！",
		User:    user,
	}
}
