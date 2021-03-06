package internal

import (
	"github.com/name5566/leaf/gate"
	"github.com/name5566/leaf/log"
	"reflect"
	pb_msg "server/msg/Protocal"
)

func init() {
	//向当前模块（game 模块）注册 Test 消息的消息处理函数 handleTest
	//handler(&pb_msg.Test{},handleTest)
	handler(&pb_msg.Ping{}, handlePing)
	handler(&pb_msg.LoginInfo_C2S{}, handleLoginInfo)
	handler(&pb_msg.JoinRoom_C2S{}, handleJoinRoom)
	handler(&pb_msg.LeaveRoom_C2S{}, handleLeaveRoom)
	handler(&pb_msg.PlayerAction_C2S{}, handlePlayerAction)
}

// 异步处理
func handler(m interface{}, h interface{}) {
	skeleton.RegisterChanRPC(reflect.TypeOf(m), h)
}

func handlePing(args []interface{}) {
	// 收到的 Hello 消息
	m := args[0].(*pb_msg.Ping)
	a := args[1].(gate.Agent)

	log.Debug("Hello Pong: %v", m)

	HeartBeatHandle(a)

	p, ok := a.UserData().(*Player)
	if ok {
		p.onClientBreathe() // 用户刷新会起新go程
	}
}

func handleLoginInfo(args []interface{}) {
	m := args[0].(*pb_msg.LoginInfo_C2S)
	a := args[1].(gate.Agent)

	log.Debug("handleLoginInfo 用户登录成功~ : %v", m)

	p, ok := a.UserData().(*Player)
	if ok {
		p.Id = m.GetId()
		p.PassWord = m.GetPassWord()
		RegisterPlayer(p)
		c4c.UserLoginCenter(m.GetId(), m.GetPassWord(), func(data *UserInfo) {
			log.Debug("Login用户登录信息: %v ", data)
			p.Id = data.ID
			p.NickName = data.Nick
			p.HeadImg = data.HeadImg
			p.Account = data.Score

			msg := &pb_msg.LoginInfo_S2C{}
			msg.PlayerInfo = new(pb_msg.PlayerInfo)
			msg.PlayerInfo.Id = p.Id
			msg.PlayerInfo.NickName = p.NickName
			msg.PlayerInfo.HeadImg = p.HeadImg
			msg.PlayerInfo.Account = p.Account
			a.WriteMsg(msg)
		})
	}
	//msg := &pb_msg.LoginInfo_S2C{}
	//msg.PlayerInfo = new(pb_msg.PlayerInfo)
	//msg.PlayerInfo.Id = "tomas"
	//msg.PlayerInfo.NickName = "nice"
	//msg.PlayerInfo.HeadImg = "24.png"
	//msg.PlayerInfo.Account = 10000
	//a.WriteMsg(msg)

	// 返回游戏大厅数据
	RspGameHallData(p)

	//判断用户是否存在房间信息,如果有就返回
	if userRoomMap[p.Id] != nil {
		PlayerLoginAgain(p, a)
	}

	//player := p.GetUserRoomInfo()
	//log.Debug("用户断线重连数据: %v", player)
	//if player != nil {
	//	p = player
	//	p.IsOnline = true
	//	p.ConnAgent = a
	//	p.ConnAgent.SetUserData(p)
	//	//返回前端信息
	//	r := p.room.RspRoomData()
	//	enter := &pb_msg.EnterRoom_S2C{}
	//	enter.RoomData = r
	//	p.SendMsg(enter)
	//	log.Debug("用户断线重连成功,返回客户端数据~")
	//}
}

func handleJoinRoom(args []interface{}) {
	m := args[0].(*pb_msg.JoinRoom_C2S)
	a := args[1].(gate.Agent)

	p, ok := a.UserData().(*Player)
	log.Debug("handleJoinRoom 玩家加入房间~ : %v", p.Id)
	log.Debug("<<<+++++++++++++++++++++++++++++++++加入房间~ : %v", p.Id)

	if ok {
		gameHall.PlayerJoinRoom(m.RoomId, p)
	}
}

func handleLeaveRoom(args []interface{}) {
	//m := args[0].(*pb_msg.LeaveRoom_C2S)
	a := args[1].(gate.Agent)

	p, ok := a.UserData().(*Player)
	log.Debug("handleLeaveRoom 玩家退出房间~ : %v", p.Id)

	if ok {
		p.PlayerReqExit()
	}
}

func handlePlayerAction(args []interface{}) {
	m := args[0].(*pb_msg.PlayerAction_C2S)
	a := args[1].(gate.Agent)

	p, ok := a.UserData().(*Player)
	log.Debug("handlePlayerAction 玩家开始行动~ : %v", p.Id)

	if ok {
		p.SetPlayerAction(m)
	}
}
