package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type User struct {
	ID        int64     // 用户ID
	Username  string    //用户名
	Password  string    //密码
	Nickname  string    //昵称
	CreatedAt time.Time //创建时间
}

// 全局数据库连接池
var db *sql.DB

// 数据库初始化
func InitDB() error {
	var err error

	// MySQL 配置
	//os.Getenv("变量名")：从操作系统读取环境变量的值，如果环境变量不存在则返回空字符串 ""
	mysqlHost := os.Getenv("MYSQL_HOST")
	mysqlPort := os.Getenv("MYSQL_PORT")
	mysqlDatabase := os.Getenv("MYSQL_DATABASE")
	mysqlUser := os.Getenv("MYSQL_USER")
	mysqlPassword := os.Getenv("MYSQL_PASSWORD")

	// 设置默认值
	if mysqlHost == "" {
		mysqlHost = "localhost"
	}
	if mysqlPort == "" {
		mysqlPort = "3306"
	}
	if mysqlDatabase == "" {
		mysqlDatabase = "chatroom"
	}
	if mysqlUser == "" {
		mysqlUser = "root"
	}
	if mysqlPassword == "" {
		mysqlPassword = "123456"
	}

	//DSN 是连接数据库的连接字符串，格式为：用户名:密码@协议(地址:端口)/数据库名?参数
	//charset=utf8mb4：支持完整 Unicode（包括 Emoji）
	//parseTime=True：自动解析时间字段为 time.Time
	//loc=Local：使用本地时区
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		mysqlUser, mysqlPassword, mysqlHost, mysqlPort, mysqlDatabase)

	// 连接到 MySQL
	db, err = sql.Open("mysql", dsn)
	// 1. 延迟连接
	// 此时并没有真正建立连接！
	// 只是创建了一个连接池对象
	if err != nil {
		return fmt.Errorf("打开MySQL数据库失败: %v", err)
	}

	log.Println("✅ 使用 MySQL 数据库")

	// 测试连接
	// 这会建立第一个连接
	if err = db.Ping(); err != nil {
		return fmt.Errorf("数据库连接失败: %v", err)
	}

	// 创建用户表
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTO_INCREMENT,
		username VARCHAR(50) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL,
		nickname VARCHAR(50) UNIQUE NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	// Exec 用于执行不返回结果集的语句
	_, err = db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("创建表失败: %v", err)
	}

	// 检查是否有初始数据，没有则插入测试用户
	var count int
	//QueryRow 用于查询单行记录
	//Scan(&count) 将查询结果赋值给 count 变量
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		// 如果表不存在，会返回错误，这里尝试创建表
		log.Printf("检查初始数据失败，可能是新数据库: %v", err)
		count = 0
	}

	if count == 0 {
		testUsers := []struct {
			username string
			password string
			nickname string
		}{
			{"user1", "123456", "用户1"},
			{"user2", "123456", "用户2"},
			{"user3", "123456", "用户3"},
		}

		for _, user := range testUsers {
			_, err := db.Exec(
				"INSERT INTO users (username, password, nickname) VALUES (?, ?, ?)",
				user.username, user.password, user.nickname,
			)
			if err != nil {
				log.Printf("插入测试用户 %s 失败: %v", user.username, err)
			}
		}
		//这里使用Exec函数，为了防止SQL注入
		//SQL 注入（SQL Injection）是一种常见的Web安全漏洞。攻击者通过在用户输入中插入恶意的SQL代码，欺骗后端数据库执行非预期的SQL命令，从而可能：
		//窃取敏感数据（用户名、密码、信用卡信息等）
		//修改数据库数据（篡改用户信息、删除数据）
		//绕过身份验证（无需密码登录系统）
		//执行数据库管理操作（删除表、关闭数据库等）
		log.Println("已创建测试用户: user1/123456, user2/123456, user3/123456")
	}

	log.Println("数据库初始化完成")
	return nil
}

// 数据库关闭
func CloseDB() {
	if db != nil {
		db.Close()
		log.Println("数据库连接已关闭")
	}
}

// 注册新用户
func RegisterUser(username, password, nickname string) error {
	// Exec函数：
	_, err := db.Exec(
		"INSERT INTO users (username, password, nickname) VALUES (?, ?, ?)",
		username, password, nickname,
	)
	return err
}

// 用户登录验证
func LoginUser(username, password string) (*User, error) {
	user := &User{}
	err := db.QueryRow(
		"SELECT id, username, password, nickname, created_at FROM users WHERE username = ? AND password = ?",
		username, password,
	).Scan(&user.ID, &user.Username, &user.Password, &user.Nickname, &user.CreatedAt)

	if err != nil {
		return nil, err
	}
	return user, nil
}

// 检查用户名是否存在
func UsernameExists(username string) (bool, error) {
	var exists bool
	// QueryRow函数：
	err := db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)",
		username,
	).Scan(&exists)
	return exists, err
}

// 检查昵称是否存在
func NicknameExists(nickname string) (bool, error) {
	var exists bool
	err := db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM users WHERE nickname = ?)",
		nickname,
	).Scan(&exists)
	return exists, err
}

// 获取用户信息
func GetUserByUsername(username string) (*User, error) {
	user := &User{}
	err := db.QueryRow(
		"SELECT id, username, password, nickname, created_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.Password, &user.Nickname, &user.CreatedAt)

	if err != nil {
		return nil, err
	}
	return user, nil
}

// 获取用户信息By昵称
func GetUserByNickname(nickname string) (*User, error) {
	user := &User{}
	err := db.QueryRow(
		"SELECT id, username, password, nickname, created_at FROM users WHERE nickname = ?",
		nickname,
	).Scan(&user.ID, &user.Username, &user.Password, &user.Nickname, &user.CreatedAt)

	if err != nil {
		return nil, err
	}
	return user, nil
}

func GetDB() *sql.DB {
	return db
}
