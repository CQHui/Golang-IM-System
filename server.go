package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Server struct {
	Ip   string
	Port int

	OnlineMap map[string]*User
	mapLock   sync.RWMutex

	//消息广播channel
	Message chan string
}

func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
	}
	return server
}

// 监听Message广播消息
func (this *Server) ListenMessager() {
	for {
		msg := <-this.Message

		this.mapLock.RLocker().Lock()
		//将msg发送给全部的在线user
		for _, cli := range this.OnlineMap {
			cli.C <- msg
		}
		this.mapLock.RLocker().Unlock()
	}
}

func (this *Server) BroadCast(user *User, msg string) {
	sendMsg := "[]" + user.Addr + "]" + user.Name + ":" + msg
	this.Message <- sendMsg
}

func (this *Server) Handler(conn net.Conn) {

	user := NewUser(conn, this)

	user.Online()

	isLive := make(chan bool)

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n == 0 {
				user.Offline()
				return
			}

			if err != nil && err != io.EOF {
				fmt.Println("Conn read error:", err)
				return
			}

			//提取用户消息(去除\n)
			msg := string(buf[:n-1])

			user.DoMessage(msg)

			//刷新用户活跃
			isLive <- true
		}
	}()
	//当前handler阻塞
	for {
		select {
		case <-isLive:
			//不错任何处理,更新定时器
		case <-time.After(time.Second * 100):
			user.sendMsg("你被踢了")
			close(user.C)
			conn.Close()

			return
		}
	}
}

func (this *Server) Start() {
	//socket listen
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", this.Ip, this.Port))
	if err != nil {
		fmt.Println("net.Listen err:", err)
		return
	}

	//close listen socket
	defer listener.Close()

	go this.ListenMessager()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("listener accept err:", err)
			continue
		}

		//do hander

		go this.Handler(conn)
	}

}
