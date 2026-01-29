package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"chatroom/auth"
	"chatroom/database"

	"net/http"
	"os"
)

// 一个已连接的客户端，包含网络连接、用户信息和通信通道。
type Client struct {
	conn    net.Conn       // TCP连接
	user    *database.User // 用户信息
	channel chan string    // 消息通道（用于向客户端发送消息）
}

// 封装广播消息，包含内容和发送者信息。
type BroadcastMessage struct {
	Content string  //消息内容
	Sender  *Client //发送者
}

type ChatRoom struct {
	clients    map[*Client]bool      // 在线客户端集合
	broadcast  chan BroadcastMessage // 广播消息通道
	join       chan *Client          // 加入通道
	leave      chan *Client          // 离开通道
	mutex      sync.RWMutex          // 读写锁
	workerPool chan struct{}         // 工作池（限流）
}

// 添加健康检查处理函数
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// 创建新的聊天室
func NewChatRoom() *ChatRoom {
	return &ChatRoom{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan BroadcastMessage, 1000),
		join:       make(chan *Client, 100),
		leave:      make(chan *Client, 100),
		workerPool: make(chan struct{}, 10),
	}
}

// 立即发送消息给指定客户端（带超时）
func sendToClient(client *Client, msg string, timeout time.Duration) bool {
	if client == nil {
		return false
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case client.channel <- msg:
		return true //尝试发送消息
	case <-timer.C:
		return false //超时未发送成功
	}
}

// 发送消息给所有客户端（排除发送者）
func (cr *ChatRoom) sendToAll(msg string, exclude *Client) {
	cr.mutex.RLock()
	defer cr.mutex.RUnlock()

	clientCount := len(cr.clients)
	if clientCount == 0 {
		return
	}

	// 只在发送普通消息时打印日志
	shouldLog := true
	if strings.Contains(msg, "离开了聊天室") ||
		strings.Contains(msg, "加入聊天室") ||
		strings.Contains(msg, "加入了聊天室") {
		shouldLog = false
	}

	if shouldLog {
		fmt.Printf("📢 广播消息: %s (在线: %d人)\n", msg, clientCount)
	}

	// 收集需要移除的客户端
	var toRemove []*Client

	for client := range cr.clients {
		// 排除发送者
		if exclude != nil && client == exclude {
			continue
		}

		// 尝试发送，最多等待500ms
		if !sendToClient(client, msg, 500*time.Millisecond) {
			toRemove = append(toRemove, client)
		}
	}

	// 移除发送失败的客户端
	if len(toRemove) > 0 {
		go func(clients []*Client) {
			for _, client := range clients {
				select {
				case cr.leave <- client:
				case <-time.After(100 * time.Millisecond):
					// 如果leave通道满，直接处理
					go cr.handleLeaveDirectly(client)
				}
			}
		}(toRemove)
	}
}

// 处理客户端离开（不通过通道）
func (cr *ChatRoom) handleLeaveDirectly(client *Client) {
	cr.mutex.Lock()

	//离开的用户的昵称，和是否还在 在线列表中
	var nickname string
	var existed bool

	//如果在线用户列表中还有这个人，就将其移除
	if _, existed = cr.clients[client]; existed {
		nickname = client.user.Nickname
		delete(cr.clients, client)
		close(client.channel)
	}

	clientCount := len(cr.clients)
	cr.mutex.Unlock()

	if existed {
		fmt.Printf("👋 %s 离开聊天室\n", nickname)
		fmt.Printf("📊 服务器状态 - 在线用户: %d 人\n", clientCount)

		// 广播离开消息
		leaveMsg := fmt.Sprintf("%s 离开了聊天室", nickname)
		go cr.sendToAll(leaveMsg, nil)

		// 延迟关闭连接
		go func() {
			time.Sleep(100 * time.Millisecond)
			client.conn.Close()
		}()
	}
}

// 处理客户端加入
func (cr *ChatRoom) handleJoin(client *Client) {
	cr.mutex.Lock()
	cr.clients[client] = true
	nickname := client.user.Nickname
	clientCount := len(cr.clients)
	cr.mutex.Unlock()

	fmt.Printf("✅ %s 加入聊天室 (在线: %d人)\n", nickname, clientCount)
	fmt.Printf("📊 服务器状态 - 在线用户: %d 人\n", clientCount)

	// 广播欢迎消息
	go cr.sendToAll(fmt.Sprintf("%s 加入了聊天室！", nickname), nil)
}

// 处理广播消息
func (cr *ChatRoom) handleBroadcast(bm BroadcastMessage) {
	// 获取工作池令牌
	cr.workerPool <- struct{}{}
	defer func() { <-cr.workerPool }()

	// 发送消息（排除发送者自己）
	cr.sendToAll(bm.Content, bm.Sender)
}

// 主消息处理循环
func (cr *ChatRoom) run() {
	for {
		select {
		case client := <-cr.join:
			cr.handleJoin(client)

		case client := <-cr.leave:
			cr.handleLeaveDirectly(client)

		case bm := <-cr.broadcast:
			go cr.handleBroadcast(bm)
		}
	}
}

// 安全地查找客户端
func (cr *ChatRoom) findClientByNickname(nickname string) *Client {
	cr.mutex.RLock()
	defer cr.mutex.RUnlock()

	for client := range cr.clients {
		if client.user.Nickname == nickname {
			return client
		}
	}
	return nil
}

// 获取所有在线用户
func (cr *ChatRoom) getOnlineUsers() []string {
	cr.mutex.RLock()
	defer cr.mutex.RUnlock()

	users := make([]string, 0, len(cr.clients))
	for client := range cr.clients {
		users = append(users, client.user.Nickname)
	}
	return users
}

// 检查客户端是否还在线
func (cr *ChatRoom) isClientOnline(client *Client) bool {
	cr.mutex.RLock()
	defer cr.mutex.RUnlock()

	_, exists := cr.clients[client]
	return exists
}

// 处理客户端连接
func (cr *ChatRoom) handleClient(conn net.Conn) {
	// 错误恢复
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("客户端处理异常: %v\n", r)
		}
		conn.Close()
	}()

	// 设置初始超时
	conn.SetReadDeadline(time.Now().Add(120 * time.Second))

	scanner := bufio.NewScanner(conn)
	var client *Client
	authenticated := false

	// 认证循环
	for !authenticated && scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())

		if text == "" {
			conn.Write([]byte("请输入命令\n"))
			continue
		}

		parts := strings.Fields(text)
		command := parts[0]

		switch command {
		case "/login":
			if len(parts) != 3 {
				conn.Write([]byte("❌ 格式错误，正确格式: /login <用户名> <密码>\n"))
				continue
			}

			username := parts[1]
			password := parts[2]

			result := auth.Login(username, password)
			if !result.Success {
				conn.Write([]byte("❌ " + result.Message + "\n"))
				continue
			}

			// 检查用户是否已在线
			if cr.findClientByNickname(result.User.Nickname) != nil {
				conn.Write([]byte("❌ 该用户已在其他位置登录\n"))
				continue
			}

			client = &Client{
				conn:    conn,
				user:    result.User,
				channel: make(chan string, 50),
			}
			authenticated = true
			successMsg := fmt.Sprintf("登录成功！欢迎回来，%s！\n", result.User.Nickname)
			conn.Write([]byte(successMsg))

		case "/register":
			if len(parts) != 4 {
				// 使用更简洁的错误消息，避免客户端显示混乱
				conn.Write([]byte("❌ 格式错误，正确格式: /register <用户名> <密码> <昵称>\n"))
				continue
			}

			username := parts[1]
			password := parts[2]
			nickname := parts[3]

			result := auth.Register(username, password, nickname)
			if !result.Success {
				conn.Write([]byte("❌ " + result.Message + "\n"))
				continue
			}

			client = &Client{
				conn:    conn,
				user:    result.User,
				channel: make(chan string, 50),
			}
			authenticated = true

			// 发送详细的成功消息，包含昵称信息
			successMsg := fmt.Sprintf("🎉 注册成功！欢迎加入，%s！\n", result.User.Nickname)
			conn.Write([]byte(successMsg))

		case "/exit":
			conn.Write([]byte("👋 再见！\n"))
			return

		default:
			conn.Write([]byte("❌ 未知命令。请使用 /login 或 /register\n"))
		}
	}

	if !authenticated {
		fmt.Printf("⏰ 客户端认证超时或连接断开: %s\n", conn.RemoteAddr())
		return
	}

	// 重置超时
	conn.SetReadDeadline(time.Time{})

	// 加入聊天室
	cr.join <- client

	// 确保离开时清理
	defer func() {
		if cr.isClientOnline(client) {
			cr.leave <- client
		}
	}()

	time.Sleep(50 * time.Millisecond)
	conn.Write([]byte("💡 输入 /help 查看可用命令\n"))

	// 启动消息读取协程
	msgChan := make(chan string, 10)
	readDone := make(chan bool)

	go func() {
		defer func() {
			close(msgChan)
			readDone <- true
		}()

		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("客户端 %s 读取异常: %v\n", client.user.Nickname, r)
			}
		}()

		for scanner.Scan() {
			msg := strings.TrimSpace(scanner.Text())
			if msg != "" {
				msgChan <- msg
			}
		}

		// 如果扫描器出错（通常是连接断开）
		if err := scanner.Err(); err != nil {
			fmt.Printf("客户端 %s 读取错误: %v\n", client.user.Nickname, err)
		}
	}()

	// 处理客户端消息
	for {
		select {
		case msg, ok := <-msgChan:
			if !ok {
				// 消息通道关闭，连接断开
				return
			}

			// 处理退出命令
			if msg == "/exit" {
				fmt.Printf("客户端 %s 请求退出\n", client.user.Nickname)
				sendToClient(client, "👋 再见！", 1*time.Second)
				return
			}

			// 处理其他命令
			switch {
			case msg == "/help":
				helpMsg := "📋 可用命令:\n" +
					"/help - 显示帮助信息\n" +
					"/list - 查看在线用户\n" +
					"/whisper <昵称> <消息> - 发送私聊消息\n" +
					"/w <昵称> <消息> - 发送私聊消息(简写)\n" +
					"/exit - 退出聊天室\n"
				sendToClient(client, helpMsg, 1*time.Second)

			case msg == "/list":
				users := cr.getOnlineUsers()
				response := fmt.Sprintf("👥 在线用户 (%d人): %s", len(users), strings.Join(users, ", "))
				sendToClient(client, response, 1*time.Second)

			case strings.HasPrefix(msg, "/whisper ") || strings.HasPrefix(msg, "/w "):
				parts := strings.SplitN(msg, " ", 3)
				if len(parts) < 3 {
					sendToClient(client, "❌ 格式错误: /whisper <昵称> <消息>", 1*time.Second)
					continue
				}

				targetName := parts[1]
				privateMsg := parts[2]

				if targetName == client.user.Nickname {
					sendToClient(client, "❌ 不能给自己发送私聊消息", 1*time.Second)
					continue
				}

				// 查找目标用户
				target := cr.findClientByNickname(targetName)

				if target == nil {
					sendToClient(client, fmt.Sprintf("❌ 用户 '%s' 不在线", targetName), 1*time.Second)
				} else {
					// 再次检查目标是否在线
					if !cr.isClientOnline(target) {
						sendToClient(client, fmt.Sprintf("❌ 用户 '%s' 已离线", targetName), 1*time.Second)
						continue
					}

					// 发送私聊消息
					sendToClient(target, fmt.Sprintf("📨 [私聊] %s: %s", client.user.Nickname, privateMsg), 1*time.Second)
					sendToClient(client, fmt.Sprintf("📤 [私聊] 发送给 %s: %s", targetName, privateMsg), 1*time.Second)
				}

			default:
				// 普通聊天消息
				cr.broadcast <- BroadcastMessage{
					Content: fmt.Sprintf("%s: %s", client.user.Nickname, msg),
					Sender:  client,
				}
			}

		// 从服务器接收消息
		case serverMsg, ok := <-client.channel:
			if !ok {
				// 客户端通道关闭
				return
			}

			conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
			_, err := fmt.Fprintln(conn, serverMsg)
			if err != nil {
				// 写入失败，连接断开
				return
			}

		case <-readDone:
			// 读取协程完成，连接断开
			return
		}
	}
}

func main() {
	// 初始化数据库
	fmt.Println("🔄 正在初始化数据库...")
	err := database.InitDB()
	if err != nil {
		fmt.Printf("❌ 数据库初始化失败: %v\n", err)
		fmt.Print("按回车键退出...")
		fmt.Scanln()
		return
	}
	defer database.CloseDB()

	fmt.Println("✅ 数据库初始化成功")

	go func() {
		http.HandleFunc("/health", healthCheckHandler)
		http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
			if database.GetDB() != nil {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("READY"))
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("NOT READY"))
			}
		})

		healthPort := os.Getenv("HEALTH_PORT")
		if healthPort == "" {
			healthPort = "8081"
		}

		fmt.Printf("健康检查服务运行在端口 %s\n", healthPort)
		http.ListenAndServe(":"+healthPort, nil)
	}()

	// 启动服务器
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Printf("❌ 启动服务器失败: %v\n", err)
		return
	}
	defer listener.Close()

	fmt.Println("✅ 聊天室服务器已启动")
	fmt.Println("📡 监听地址: localhost:8080")
	fmt.Println("🔐 支持登录/注册功能")
	fmt.Printf("💾 数据库: MySQL (%s)\n", os.Getenv("MYSQL_HOST"))
	fmt.Println("🔄 等待客户端连接...")
	fmt.Println("=================================")

	// 创建聊天室
	chatRoom := NewChatRoom()

	// 启动消息处理协程
	go chatRoom.run()

	// 接受客户端连接
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("⚠️  接受连接失败: %v\n", err)
			continue
		}

		clientAddr := conn.RemoteAddr().String()
		fmt.Printf("🔗 新客户端连接: %s\n", clientAddr)

		// 处理客户端
		go chatRoom.handleClient(conn)
	}
}
