package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

type ClientState struct {
	Authenticated bool
	Nickname      string
	Username      string
}

func main() {

	// 获取服务器地址，默认使用容器网络中的服务名
	//serverAddr := "chatroom-server:8080"
	//if len(os.Args) > 1 {
	//	serverAddr = os.Args[1]
	//}
	serverAddr := os.Getenv("SERVER_HOST")
	serverPort := os.Getenv("SERVER_PORT")
	if serverAddr == "" {
		serverAddr = "localhost"
	}
	if serverPort == "" {
		serverPort = "8080"
	}

	fmt.Println("=== 聊天室客户端 ===")
	fmt.Println("版本: 2.0.0 (带登录注册)")
	fmt.Println("===================")

	// 连接服务器
	conn, err := net.Dial("tcp", serverAddr+":"+serverPort)
	if err != nil {
		fmt.Printf("❌ 无法连接到服务器: %v\n", err)
		fmt.Println("请确保服务器正在运行在 localhost:8080")
		fmt.Print("按回车键退出...")
		bufio.NewScanner(os.Stdin).Scan()
		return
	}
	defer conn.Close()

	fmt.Println("✅ 已连接到聊天室服务器!")

	// 创建消息通道
	serverMsgChan := make(chan string, 100)
	userInputChan := make(chan string, 10)
	stopChan := make(chan bool, 1)

	var wg sync.WaitGroup
	wg.Add(2)

	// 客户端状态
	state := &ClientState{
		Authenticated: false,
		Nickname:      "",
		Username:      "",
	}

	// goroutine 用于接收服务器消息
	go func() {
		defer wg.Done()
		reader := bufio.NewReader(conn)

		for {
			select {
			case <-stopChan:
				return
			default:
				message, err := reader.ReadString('\n')
				if err != nil {
					fmt.Println("\n❌ 与服务器的连接已断开")
					close(serverMsgChan)
					return
				}
				message = strings.TrimSuffix(message, "\n")
				serverMsgChan <- message
			}
		}
	}()

	// goroutine 用于发送消息到服务器
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(os.Stdin)

		for scanner.Scan() {
			text := strings.TrimSpace(scanner.Text())
			if text == "" {
				continue
			}

			userInputChan <- text
		}

		if err := scanner.Err(); err != nil {
			fmt.Printf("读取输入失败: %v\n", err)
		}
		close(userInputChan)
	}()

	// 显示初始认证菜单
	showAuthMenu()

	// 主控制循环
	for {
		select {
		case msg, ok := <-serverMsgChan:
			if !ok {
				// 服务器消息通道关闭
				fmt.Println("\n❌ 与服务器的连接已断开")
				stopChan <- true
				wg.Wait()
				return
			}

			// 显示服务器消息
			fmt.Printf("\r%s\n", msg)

			// 解析和处理服务器消息
			processServerMessage(msg, state)

			// 重新显示提示符
			if state.Authenticated {
				// 检查是否是帮助消息的结束
				if !strings.Contains(msg, "可用命令") &&
					!strings.Contains(msg, "/help -") &&
					!strings.Contains(msg, "/list -") &&
					!strings.Contains(msg, "/whisper -") &&
					!strings.Contains(msg, "/w -") &&
					!strings.Contains(msg, "/exit -") {
					fmt.Printf("[%s] > ", state.Nickname)
				}
			} else {
				showAuthMenu()
			}

		case text, ok := <-userInputChan:
			if !ok {
				// 用户输入通道关闭
				continue
			}

			// 认证阶段处理
			if !state.Authenticated {
				handleUnauthenticatedInput(text, conn, state)
				continue
			}

			// 已认证，处理聊天命令
			if text == "/exit" {
				fmt.Println("正在退出聊天室...")
				conn.Write([]byte("/exit\n"))
				time.Sleep(100 * time.Millisecond)

				stopChan <- true
				wg.Wait()
				return
			}

			_, err := conn.Write([]byte(text + "\n"))
			if err != nil {
				fmt.Printf("发送消息失败: %v\n", err)
				stopChan <- true
				wg.Wait()
				return
			}

		case <-time.After(5 * time.Minute):
			// 超时检查
			fmt.Println("\n⏰ 连接超时")
			stopChan <- true
			wg.Wait()
			return
		}
	}
}

// 显示认证菜单
func showAuthMenu() {
	fmt.Println()
	fmt.Println("=== 认证菜单 ===")
	fmt.Println("1. 登录 - 格式: /login <用户名> <密码>")
	fmt.Println("   示例: /login user1 123456")
	fmt.Println()
	fmt.Println("2. 注册 - 格式: /register <用户名> <密码> <昵称>")
	fmt.Println("   示例: /register newuser 123456 新用户")
	fmt.Println()
	fmt.Println("3. 退出 - 输入: /exit")
	fmt.Println("=================")
	fmt.Print("请输入命令: ")
}

// 处理未认证的输入
func handleUnauthenticatedInput(text string, conn net.Conn, state *ClientState) {
	// 处理认证命令
	if strings.HasPrefix(text, "/login ") || strings.HasPrefix(text, "/register ") {
		// 发送认证命令
		_, err := conn.Write([]byte(text + "\n"))
		if err != nil {
			fmt.Printf("发送认证信息失败: %v\n", err)
		}
	} else if text == "/exit" {
		conn.Write([]byte("/exit\n"))
		fmt.Println("退出程序...")
		os.Exit(0)
	} else {
		fmt.Println("❌ 请先登录或注册")
	}
}

// 处理服务器消息
func processServerMessage(msg string, state *ClientState) {
	// 处理认证响应
	if strings.Contains(msg, "登录成功") || strings.Contains(msg, "🎉 注册成功") {
		state.Authenticated = true

		nickname := ""

		// 处理登录成功消息
		if strings.Contains(msg, "登录成功") && strings.Contains(msg, "欢迎回来") {
			// 查找"欢迎回来"后的昵称
			if idx := strings.Index(msg, "欢迎回来"); idx != -1 {
				// 提取"欢迎回来"之后的所有内容
				temp := msg[idx+len("欢迎回来"):]
				// 去除可能的标点符号和空格
				temp = strings.Trim(temp, " ，！!")
				nickname = strings.TrimSpace(temp)
			}
		}

		// 处理注册成功消息
		if nickname == "" && strings.Contains(msg, "🎉 注册成功") && strings.Contains(msg, "欢迎加入") {
			if idx := strings.Index(msg, "欢迎加入"); idx != -1 {
				temp := msg[idx+len("欢迎加入"):]
				temp = strings.Trim(temp, " ，！!")
				nickname = strings.TrimSpace(temp)
			}
		}

		// 如果还提取不到，尝试更通用的方法
		if nickname == "" {
			// 尝试提取用户名（可能出现在"用户名"之后）
			if strings.Contains(msg, "用户名") {
				// 简化处理：从消息中查找明显的标识
				parts := strings.FieldsFunc(msg, func(r rune) bool {
					return r == '：' || r == ':' || r == '，' || r == ',' || r == '！' || r == '!'
				})
				for _, part := range parts {
					part = strings.TrimSpace(part)
					if len(part) > 1 && !strings.Contains(part, "登录") &&
						!strings.Contains(part, "注册") && !strings.Contains(part, "成功") {
						nickname = part
						break
					}
				}
			}
		}

		// 如果还提取不到，使用用户名（如果有的话）
		if nickname == "" && state.Username != "" {
			nickname = state.Username
		}

		// 如果还提取不到，使用默认值
		if nickname == "" {
			nickname = "用户"
		}

		state.Nickname = nickname
		fmt.Printf("\n🎉 认证成功！您的昵称: %s\n", state.Nickname)
	}

	// 处理帮助消息 - 这是一个多行消息，我们标记它为特殊处理
	isHelpMessage := strings.Contains(msg, "📋 可用命令:") || strings.Contains(msg, "可用命令:")

	// 处理私聊消息的显示优化
	if strings.Contains(msg, "[私聊]") {
		// 私聊消息已经格式化好了，直接显示
		return
	}

	// 处理用户加入/离开消息
	if strings.Contains(msg, "加入了聊天室") || strings.Contains(msg, "离开了聊天室") {
		// 这些消息已经格式化好了
		return
	}

	// 处理欢迎消息
	if strings.Contains(msg, "输入 /help 查看帮助") {
		fmt.Println()
		fmt.Println("💬 现在可以开始聊天了！")
		fmt.Println("📋 可用命令:")
		fmt.Println("   /help        - 显示帮助信息")
		fmt.Println("   /list        - 查看在线用户")
		fmt.Println("   /w 昵称 消息 - 发送私聊消息")
		fmt.Println("   /whisper 昵称 消息 - 发送私聊消息")
		fmt.Println("   /exit        - 退出聊天室")
		fmt.Println("----------------------------------------")
		return
	}

	// 处理错误消息
	if strings.Contains(msg, "❌") {
		// 错误消息，保持红色显示
		return
	}

	// 如果是帮助消息，不重新显示提示符（将在主循环中统一显示）
	if isHelpMessage {
		return
	}
}
