package main

import (
	"net"
	"strings"
)

type User struct {
	Name string
	Addr string
	C    chan string
	conn net.Conn

	server *Server
}

// 创建一个用户的API
func NewUser(conn net.Conn, server *Server) *User {
	userAddr := conn.RemoteAddr().String()

	user := &User{
		Name:   userAddr,
		Addr:   userAddr,
		C:      make(chan string),
		conn:   conn,
		server: server,
	}

	go user.ListenMessage()
	return user
}

func (this *User) ListenMessage() {
	for {
		msg := <-this.C

		this.conn.Write([]byte(msg + "\n"))
	}
}

func (this *User) Online() {
	this.server.mapLock.Lock()
	this.server.OnlineMap[this.Name] = this
	this.server.mapLock.Unlock()
	this.server.BroadCast(this, "上线")
}

func (this *User) Offline() {
	this.server.mapLock.Lock()
	delete(this.server.OnlineMap, this.Name)
	this.server.mapLock.Unlock()
	this.server.BroadCast(this, "下线")
}

func (this *User) DoMessage(msg string) {
	if msg == "who" {
		this.server.mapLock.RLocker().Lock()
		for _, user := range this.server.OnlineMap {
			onlineMsg := "[]" + user.Addr + "]" + user.Name + ":" + "在线...\n"
			this.sendMsg(onlineMsg)
		}
		this.server.mapLock.RLocker().Unlock()

	} else if len(msg) > 7 && msg[:7] == "rename|" {
		newName := strings.Split(msg, "|")[1]

		_, ok := this.server.OnlineMap[newName]
		if ok {
			this.sendMsg("当前用户名被使用\n")
		} else {
			this.server.mapLock.RLocker().Lock()
			delete(this.server.OnlineMap, this.Name)
			this.server.OnlineMap[newName] = this
			this.server.mapLock.RLocker().Unlock()

			this.Name = newName
			this.sendMsg("您已经更新完成用户名, " + newName + "\n")
		}
	} else if len(msg) > 4 && msg[:3] == "to|" {
		remoteName := strings.Split(msg, "|")[1]
		if remoteName == "" {
			this.sendMsg("消息格式不正确, 使用\"to|张三|message\"格式 \n")
		}
		remoteUser, ok := this.server.OnlineMap[remoteName]
		if !ok {
			this.sendMsg("该用户名不存在 " + remoteName + "\n")
			return
		}

		content := strings.Split(msg, "|")[2]
		if content == "" {
			this.sendMsg("无消息内容")
			return
		}
		remoteUser.sendMsg(this.Name + "对您说:" + content)
	}
	this.server.BroadCast(this, msg)
}

func (this *User) sendMsg(msg string) {
	this.conn.Write([]byte(msg + "\n"))
}
