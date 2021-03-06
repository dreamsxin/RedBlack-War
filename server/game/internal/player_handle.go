package internal

import (
	"github.com/name5566/leaf/gate"
	"github.com/name5566/leaf/log"
	pb_msg "server/msg/Protocal"
	"time"
)

//HeartBeatHandle 用户心跳处理
func HeartBeatHandle(a gate.Agent) {

	timer := time.Now().UnixNano() / 1e6

	pong := &pb_msg.Pong{
		ServerTime: timer,
	}
	a.WriteMsg(pong)
}

//GetUserRoomInfo 用户重新登陆，获取房间信息
func (p *Player) GetUserRoomInfo() *Player {
	for _, v := range gameHall.roomList {
		if v != nil {
			for _, pl := range v.PlayerList {
				if pl != nil && pl.Id == p.Id {
					return pl
				}
			}
		}
	}
	return nil
}

//PlayerLoginAgain 用户重新登陆
func PlayerLoginAgain(p *Player, a gate.Agent) {
	log.Debug("<------- 用户重新登陆: %v ------->", p.Id)
	p.room = userRoomMap[p.Id]
	for _, v := range p.room.PlayerList {
		if v.Id == p.Id {
			p = v
		}
	}
	p.IsOnline = true
	p.ConnAgent = a
	p.ConnAgent.SetUserData(p)

	//返回前端信息
	//fmt.Println("LoginAgain房间信息:", p.room)
	r := p.room.RspRoomData()
	enter := &pb_msg.EnterRoom_S2C{}
	enter.RoomData = r
	if p.room.GameStat == DownBet {
		enter.GameTime = DownBetTime - p.room.counter
		log.Debug("用户重新登陆 DownBetTime.GameTime: %v", enter.GameTime)
	} else {
		enter.GameTime = SettleTime - p.room.counter
		log.Debug("用户重新登陆 SettleTime.GameTime: %v", enter.GameTime)
	}
	p.SendMsg(enter)

	//更新房间列表
	p.room.UpdatePlayerList()
	maintainList := p.room.PackageRoomPlayerList()
	p.room.BroadCastExcept(maintainList, p)
	log.Debug("用户断线重连成功,返回客户端数据~ ")
}

//PlayerExitRoom 玩家退出房间
func (p *Player) PlayerReqExit() {
	if p.room != nil {
		if p.IsRobot == false && p.IsAction == true {
			if p.ResultMoney == 0 {
				p.LoseResultMoney = 0
				p.LoseResultMoney -= float64(p.DownBetMoneys.RedDownBet)
				p.LoseResultMoney -= float64(p.DownBetMoneys.BlackDownBet)
				p.LoseResultMoney -= float64(p.DownBetMoneys.LuckDownBet)
				timeStr := time.Now().Format("2006-01-02_15:04:05")
				nowTime := time.Now().Unix()
				reason := "ExitRoomResult"

				SurplusPool -= p.LoseResultMoney
				//同时同步赢分和输分
				c4c.UserSyncLoseScore(p, nowTime, timeStr, reason)
			}
		}
		p.room.ExitFromRoom(p)
	} else {
		log.Debug("Player Exit Room, But not found Player Room ~")
	}
}

//SetAction 设置玩家行动
func (p *Player) SetPlayerAction(m *pb_msg.PlayerAction_C2S) {
	//不是下注阶段不能进行下注
	if p.room.GameStat != DownBet {
		//返回前端信息
		msg := &pb_msg.MsgInfo_S2C{}
		msg.Msg = recodeText[RECODE_NOTDOWNBETSTATUS]
		p.SendMsg(msg)
		log.Debug("当前不是下注阶段,玩家不能行动~")
		return
	}

	//判断玩家金额是否足够下注的金额(这里其实金额不足玩家是不能在进行点击事件的。双重安全!)
	if p.Account < float64(m.DownBet) {
		msg := &pb_msg.MsgInfo_S2C{}
		msg.Error = recodeText[RECODE_NOTDOWNMONEY]
		p.SendMsg(msg)

		log.Debug("玩家金额不足,不能进行下注~")
		return
	}

	p.IsAction = m.IsAction
	//判断玩家是否行动,做相应处理
	if p.IsAction == true {
		//记录玩家在该房间总下注 和 房间注池的总金额
		if m.DownPot == pb_msg.PotType_RedPot {
			p.Account -= float64(m.DownBet)
			p.DownBetMoneys.RedDownBet += m.DownBet
			p.TotalAmountBet += m.DownBet
			p.room.PotMoneyCount.RedMoneyCount += m.DownBet
		}
		if m.DownPot == pb_msg.PotType_BlackPot {
			p.Account -= float64(m.DownBet)
			p.DownBetMoneys.BlackDownBet += m.DownBet
			p.TotalAmountBet += m.DownBet
			p.room.PotMoneyCount.BlackMoneyCount += m.DownBet
		}
		if m.DownPot == pb_msg.PotType_LuckPot {
			p.Account -= float64(m.DownBet)
			p.DownBetMoneys.LuckDownBet += m.DownBet
			p.TotalAmountBet += m.DownBet
			p.room.PotMoneyCount.LuckMoneyCount += m.DownBet
		}
		//记录续投下注的金额对应注池
		p.ContinueVot.DownBetMoneys.RedDownBet = p.DownBetMoneys.RedDownBet
		p.ContinueVot.DownBetMoneys.BlackDownBet = p.DownBetMoneys.BlackDownBet
		p.ContinueVot.DownBetMoneys.LuckDownBet = p.DownBetMoneys.LuckDownBet
		p.ContinueVot.TotalMoneyBet = p.ContinueVot.DownBetMoneys.RedDownBet + p.ContinueVot.DownBetMoneys.BlackDownBet + p.ContinueVot.DownBetMoneys.LuckDownBet
	}

	//返回前端玩家行动,更新玩家最新金额
	action := &pb_msg.PlayerAction_S2C{}
	action.Id = p.Id
	action.DownBet = m.DownBet
	action.DownPot = m.DownPot
	action.IsAction = m.IsAction
	action.Account = p.Account
	p.room.BroadCastMsg(action)
	//p.SendMsg(action)

	//广播玩家注池金额
	pot := &pb_msg.PotTotalMoney_S2C{}
	pot.PotMoneyCount = new(pb_msg.PotMoneyCount)
	pot.PotMoneyCount.RedMoneyCount = p.room.PotMoneyCount.RedMoneyCount
	pot.PotMoneyCount.BlackMoneyCount = p.room.PotMoneyCount.BlackMoneyCount
	pot.PotMoneyCount.LuckMoneyCount = p.room.PotMoneyCount.LuckMoneyCount
	p.room.BroadCastMsg(pot)

	//fmt.Println("玩家:", p.Id, "行动 红、黑、Luck下注: ", p.DownBetMoneys, "玩家总下注金额: ", p.TotalAmountBet)
	//fmt.Println("房间池红、黑、Luck总下注: ", p.room.PotMoneyCount, "续投总额:", p.ContinueVot.TotalMoneyBet)
}

//RspGameHallData 返回大厅数据
func RspGameHallData(p *Player) {

	hallTime := &pb_msg.GameHallTime_S2C{}

	hallData := &pb_msg.GameHallData_S2C{}

	p.HallRoomData = nil
	for _, r := range gameHall.roomList {
		ht := &pb_msg.HallTime{}
		hd := &pb_msg.HallData{}
		data := &HallDataList{}

		if r != nil {
			ht.RoomId = r.RoomId
			if r.GameStat == DownBet {
				ht.GameStage = pb_msg.GameStage(DownBet)
				ht.RoomTime = DownBetTime - r.counter
				log.Debug("游戏大厅.DownBetTime : %v", ht.RoomTime)
			} else {
				ht.GameStage = pb_msg.GameStage(Settle)
				ht.RoomTime = SettleTime - r.counter
				log.Debug("游戏大厅 SettleTime : %v", ht.RoomTime)
			}
			hallTime.HallTime = append(hallTime.HallTime, ht)

			data.Rid = r.RoomId
			//最新40局游戏数据、红黑Win顺序列表、每局Win牌局类型、红黑Luck的总数量
			roomGCount := r.RoomGameCount()

			//判断房间数据是否大于40局
			if roomGCount > RoomCordCount {
				//大于40局则截取最新40局数据
				num := roomGCount - RoomCordCount
				data.HallCardTypeList = append(data.HallCardTypeList, r.CardTypeList[num:]...)
				for _, v := range r.RPotWinList {
					if v.RedWin == 1 {
						data.HallRedBlackList = append(data.HallRedBlackList, RedWin)
					}
					if v.BlackWin == 1 {
						data.HallRedBlackList = append(data.HallRedBlackList, BlackWin)
					}
					//截取到40局游戏就退出
					if len(data.HallRedBlackList) == RoomCordCount {
						break
					}
				}
			} else {
				//小于40局则截取全部房间数据
				data.HallCardTypeList = append(data.HallCardTypeList, r.CardTypeList...)
				for _, v := range r.RPotWinList {
					if v.RedWin == 1 {
						data.HallRedBlackList = append(data.HallRedBlackList, RedWin)
					}
					if v.BlackWin == 1 {
						data.HallRedBlackList = append(data.HallRedBlackList, BlackWin)
					}
				}
			}
			hd.RoomId = data.Rid
			hd.CardTypeList = data.HallCardTypeList
			hd.RedBlackList = data.HallRedBlackList
			hallData.HallData = append(hallData.HallData, hd)
			p.HallRoomData = append(p.HallRoomData, data)
		}
	}
	p.SendMsg(hallTime)
	p.SendMsg(hallData)

}
