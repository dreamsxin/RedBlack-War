package internal

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/name5566/leaf/log"
	"io/ioutil"
	"net/http"
	"reflect"
	"server/conf"
	"strconv"
	"strings"
	"time"
)

//CGTokenRsp 接受Token结构体
type CGTokenRsp struct {
	Token string
}

//CGCenterRsp 中心返回消息结构体
type CGCenterRsp struct {
	Status string
	Code   int
	Msg    *CGTokenRsp
}

//Conn4Center 连接到Center(中心服务器)的网络协议处理器
type Conn4Center struct {
	GameId    string
	centerUrl string
	token     string
	DevKey    string
	conn      *websocket.Conn

	//除于登录成功状态
	LoginStat bool

	//待处理的用户登录请求
	waitUser map[string]*UserCallback
}

//Init 初始化
func (c4c *Conn4Center) Init() {
	c4c.GameId = conf.Server.GameID
	c4c.DevKey = conf.Server.DevKey
	c4c.LoginStat = false

	c4c.waitUser = make(map[string]*UserCallback)
}

//onDestroy 销毁用户
func (c4c *Conn4Center) onDestroy() {
	log.Debug("Conn4Center onDestroy ~")
	//c4c.UserLogoutCenter("991738698","123456") //todo  测试用户
}

//ReqCenterToken 向中心服务器请求token
func (c4c *Conn4Center) ReqCenterToken() {
	// 拼接center Url
	url4Center := fmt.Sprintf("http://swoole.0717996.com/Token/getToken?dev_key=%s&dev_name=%s", c4c.DevKey, conf.Server.DevName)

	log.Debug("<--- Center access Url --->: %v ", url4Center)

	resp, err1 := http.Get(url4Center)
	if err1 != nil {
		panic(err1.Error())
	}
	log.Debug("<--- resp --->: %v ", resp)

	defer resp.Body.Close()

	if err1 == nil && resp.StatusCode == 200 {
		body, err2 := ioutil.ReadAll(resp.Body)
		if err2 != nil {
			panic(err2.Error())
		}
		//log.Debug("<----- resp.StatusCode ----->: %v", resp.StatusCode)
		log.Debug("<--- body --->: %v ,<--- err2 --->: %v", string(body), err2)

		var t CGCenterRsp
		err3 := json.Unmarshal(body, &t)
		log.Debug("<--- err3 --->: %v <--- Results --->: %v", err3, t)

		if t.Status == "SUCCESS" && t.Code == 200 {
			log.Debug("<--- CenterToken --->: %v", t.Msg.Token)
			c4c.token = t.Msg.Token
			c4c.CreatConnect()
		} else {
			log.Fatal("<--- Request Token Fail~ --->")
		}
	}
}

//CreatConnect 和Center建立链接
func (c4c *Conn4Center) CreatConnect() {
	//c4c.centerUrl = "ws://47.75.152.229:9502/"
	c4c.centerUrl = "ws" + strings.TrimPrefix(conf.Server.CenterServer, "http")

	conn, rsp, err := websocket.DefaultDialer.Dial(c4c.centerUrl, nil)
	c4c.conn = conn
	log.Debug("<--- Dial rsp --->: %v", rsp)

	if err != nil {
		log.Fatal(err.Error())
	} else {
		c4c.Run()
	}
}

//Run 开始运行,监听中心服务器的返回
func (c4c *Conn4Center) Run() {
	ticker := time.NewTicker(time.Second * 5)
	go func() {
		for { //循环
			<-ticker.C
			c4c.onBreath()
		}
	}()

	go func() {
		for {
			typeId, message, err := c4c.conn.ReadMessage()
			if err != nil {
				log.Debug("Here is error by ReadMessage~")
				log.Error(err.Error())
			}
			log.Debug("Receive a message from Center~")
			log.Debug("typeId: %v", typeId)
			log.Debug("message: %v", string(message))
			c4c.onReceive(typeId, message)
		}
	}()

	c4c.ServerLoginCenter()
}

//onBreath 中心服心跳
func (c4c *Conn4Center) onBreath() {
	err := c4c.conn.WriteMessage(websocket.TextMessage, []byte(""))
	if err != nil {
		log.Error(err.Error())
	}
}

//onReceive 接收消息
func (c4c *Conn4Center) onReceive(messType int, messBody []byte) {
	if messType == websocket.TextMessage {
		baseData := &BaseMessage{}

		decoder := json.NewDecoder(strings.NewReader(string(messBody)))
		decoder.UseNumber()

		err := decoder.Decode(&baseData)
		if err != nil {
			log.Error(err.Error())
		}
		log.Debug("<-------- baseData -------->: %v", baseData)

		switch baseData.Event {
		case msgServerLogin:
			c4c.onServerLogin(baseData.Data)
			log.Debug("<-------- baseData onServerLogin -------->")
			break
		case msgUserLogin:
			c4c.onUserLogin(baseData.Data)
			log.Debug("<-------- baseData msgUserLogin -------->")
			break
		case msgUserWinScore:
			c4c.onUserWinScore(baseData.Data)
			log.Debug("<-------- baseData msgUserWinScore -------->")
			break
		case msgUserLoseScore:
			c4c.onUserLoseScore(baseData.Data)
			log.Debug("<-------- baseData msgUserLoseScore -------->")
			break
		default:
			log.Error("Receive a message but don't identify~")
		}
	}
}

//onServerLogin 服务器登录
func (c4c *Conn4Center) onServerLogin(msgBody interface{}) {
	log.Debug("<-------- onServerLogin -------->: %v", msgBody)
	data, ok := msgBody.(map[string]interface{})
	if ok {
		//fmt.Println(data["status"], reflect.TypeOf(data["status"]))
		//fmt.Println(data["code"], reflect.TypeOf(data["code"]))
		//fmt.Println(data["msg"], reflect.TypeOf(data["msg"]))
		code, err := data["code"].(json.Number).Int64()
		//fmt.Println("code,err", code, err)
		if err != nil {
			log.Fatal(err.Error())
		}

		fmt.Println(code, reflect.TypeOf(code))
		if data["status"] == "SUCCESS" && code == 200 {
			fmt.Println("<-------- serverLogin success~!!! -------->")

			c4c.LoginStat = true
		}
	}
}

//onUserLogin 收到中心服的用户登录回应
func (c4c *Conn4Center) onUserLogin(msgBody interface{}) {
	log.Debug("<-------- onUserLogin -------->: %v", msgBody)
	data, ok := msgBody.(map[string]interface{})
	log.Debug("data:%v, ok:%v", data, ok)

	//fmt.Println(data["status"], reflect.TypeOf(data["status"]))
	//fmt.Println(data["code"], reflect.TypeOf(data["code"]))
	//fmt.Println(data["msg"], reflect.TypeOf(data["msg"]))
	code, err := data["code"].(json.Number).Int64()
	//fmt.Println("code,err", code, err)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if data["status"] == "SUCCESS" && code == 200 {
		log.Debug("<-------- UserLogin SUCCESS~ -------->")
		userInfo, ok := data["msg"].(map[string]interface{})
		var strId string
		var userData *UserCallback
		if ok {
			log.Debug("userInfo: %v", userInfo)
			gameUser, uok := userInfo["game_user"].(map[string]interface{})
			if uok {
				log.Debug("gameUser: %v", gameUser)
				nick := gameUser["game_nick"]
				headImg := gameUser["game_img"]
				userId := gameUser["id"]
				log.Debug("nick: %v", nick)
				log.Debug("headImg: %v", headImg)
				log.Debug("userId: %v %v", userId, reflect.TypeOf(userId))

				intID, err := userId.(json.Number).Int64()
				if err != nil {
					log.Fatal(err.Error())
				}
				strId = strconv.Itoa(int(intID))
				log.Debug("strId: %v %v", strId, reflect.TypeOf(strId))

				//找到等待登录玩家
				userData, ok = c4c.waitUser[strId]
				if ok {
					userData.Data.HeadImg = headImg.(string)
					userData.Data.Nick = nick.(string)
				}
			}
			gameAccount, okA := userInfo["game_account"].(map[string]interface{})

			if okA {
				log.Debug("<-------- gameAccount -------->: %v", gameAccount)
				balance := gameAccount["balance"]
				log.Debug("<-------- balance -------->: %v %v", balance, reflect.TypeOf(balance))
				floatBalance, err := balance.(json.Number).Float64()
				if err != nil {
					log.Error(err.Error())
				}

				userData.Data.Score = floatBalance

				//调用玩家绑定回调函数
				if userData.Callback != nil {
					userData.Callback(&userData.Data)
				}
			}
		}
	}
}

func (c4c *Conn4Center) onUserWinScore(msgBody interface{}) {
	log.Debug("<-------- onUserWinScore -------->: %v", msgBody)
	data, ok := msgBody.(map[string]interface{})

	log.Debug("<-------- data -------->:%v, <-------- ok -------->:%v", data, ok)

	//fmt.Println(data["status"], reflect.TypeOf(data["status"]))
	//fmt.Println(data["code"], reflect.TypeOf(data["code"]))
	//fmt.Println(data["msg"], reflect.TypeOf(data["msg"]))
	code, err := data["code"].(json.Number).Int64()
	//fmt.Println("code,err", code, err)
	if err != nil {
		log.Error(err.Error())
	}

	log.Debug("data:%v, ok:%v", data, ok)
	if data["status"] == "SUCCESS" && code == 200 {
		log.Debug("<-------- UserWinScore SUCCESS~ -------->")
		userInfo, ok := data["msg"].(map[string]interface{})
		if ok {
			userId := userInfo["id"]
			log.Debug("userId: %v, %v", userId, reflect.TypeOf(userId))

			intID, err := userId.(json.Number).Int64()
			if err != nil {
				log.Error(err.Error())
				return
			}
			strID := strconv.Itoa(int(intID))
			log.Debug("<-------- strID -------->: %v, %v", strID, reflect.TypeOf(strID))

			jsonScore := userInfo["final_pay"]
			score, err := jsonScore.(json.Number).Float64()

			log.Debug("<--------- final win score: %v", score)

			if err != nil {
				log.Error(err.Error())
				return
			}
		}
	}
}

func (c4c *Conn4Center) onUserLoseScore(msgBody interface{}) {
	log.Debug("<-------- onUserLoseScore -------->: %v", msgBody)
	data, ok := msgBody.(map[string]interface{})

	log.Debug("data:%v, ok:%v", data, ok)

	//fmt.Println(data["status"], reflect.TypeOf(data["status"]))
	//fmt.Println(data["code"], reflect.TypeOf(data["code"]))
	//fmt.Println(data["msg"], reflect.TypeOf(data["msg"]))
	code, err := data["code"].(json.Number).Int64()
	//fmt.Println("code,err", code, err)
	if err != nil {
		log.Error(err.Error())
	}

	fmt.Println(code, err)
	if data["status"] == "SUCCESS" && code == 200 {
		fmt.Println("UserWinScore SUCCESS~")
		userInfo, ok := data["msg"].(map[string]interface{})
		if ok {
			userId := userInfo["id"]
			log.Debug("userId: %v, %v", userId, reflect.TypeOf(userId))

			intID, err := userId.(json.Number).Int64()
			if err != nil {
				log.Error(err.Error())
				return
			}
			strID := strconv.Itoa(int(intID))
			log.Debug("<-------- strID -------->: %v, %v", strID, reflect.TypeOf(strID))

			jsonScore := userInfo["final_pay"]
			score, err := jsonScore.(json.Number).Float64()

			log.Debug("<-------- final lose score -------->: %v", score)
			if err != nil {
				log.Error(err.Error())
				return
			}
		}
	}
}

//ServerLoginCenter 服务器登录Center
func (c4c *Conn4Center) ServerLoginCenter() {
	log.Debug("<-------- ServerLoginCenter c4c.token -------->: %v", c4c.token)
	baseData := &BaseMessage{}
	baseData.Event = msgServerLogin
	baseData.Data = ServerLogin{
		Host:   conf.Server.CenterServer,
		Port:   conf.Server.CenterServerPort,
		GameId: c4c.GameId,
		Token:  c4c.token,
		DevKey: c4c.DevKey,
	}
	// 发送消息到中心服
	c4c.SendMsg2Center(baseData)
}

//UserLoginCenter 用户登录
func (c4c *Conn4Center) UserLoginCenter(userId string, password string, callback func(data *UserInfo)) {
	if !c4c.LoginStat {
		log.Debug("<-------- server not ready~!!! -------->")
		return
	}

	log.Debug("<-------- UserLoginCenter c4c.token -------->: %v", c4c.token)
	baseData := &BaseMessage{}
	baseData.Event = msgUserLogin
	baseData.Data = &UserReq{
		ID:       userId,
		PassWord: password,
		GameId:   c4c.GameId,
		Token:    c4c.token,
		DevKey:   c4c.DevKey}

	c4c.SendMsg2Center(baseData)

	//加入待处理map，等待处理
	c4c.waitUser[userId] = &UserCallback{}
	c4c.waitUser[userId].Data.ID = userId
	c4c.waitUser[userId].Callback = callback
}

//UserLogoutCenter 用户登出
func (c4c *Conn4Center) UserLogoutCenter(userId string) {
	log.Debug("<-------- UserLogoutCenter c4c.token -------->: %v", c4c.token)
	base := &BaseMessage{}
	base.Event = msgUserLogout
	base.Data = &UserReq{
		ID: userId,
		//PassWord: password,
		GameId: c4c.GameId,
		Token:  c4c.token,
		DevKey: c4c.DevKey,
	}
	// 发送消息到中心服
	c4c.SendMsg2Center(base)
}

//SendMsg2Center 发送消息到中心服
func (c4c *Conn4Center) SendMsg2Center(data interface{}) {
	// Json序列化
	codeData, err1 := json.Marshal(data)
	if err1 != nil {
		log.Error(err1.Error())
	}
	log.Debug("<-------- 发送消息中心服 -------->: %v", string(codeData))

	err2 := c4c.conn.WriteMessage(websocket.TextMessage, []byte(codeData))
	if err2 != nil {
		log.Fatal(err2.Error())
	}
}

//UserSyncWinScore 同步赢分
func (c4c *Conn4Center) UserSyncWinScore(p *Player, timeUnix int64, timeStr, reason string) {
	winOrder := p.Id + "_" + timeStr + "_win"
	log.Debug("<-------- GenWinOrder -------->: %v", winOrder)
	baseData := &BaseMessage{}
	baseData.Event = msgUserWinScore
	userWin := &UserChangeScore{}
	userWin.Auth.Token = c4c.token
	userWin.Auth.DevKey = c4c.DevKey
	userWin.Info.CreateTime = timeUnix
	userWin.Info.GameId = c4c.GameId
	userWin.Info.ID = p.Id
	userWin.Info.LockMoney = 0
	userWin.Info.Money = p.WinResultMoney
	userWin.Info.Order = winOrder
	userWin.Info.PayReason = reason
	userWin.Info.PreMoney = 0
	userWin.Info.RoundId = p.room.RoomId
	baseData.Data = userWin

	c4c.SendMsg2Center(baseData)
}

//UserSyncWinScore 同步输分
func (c4c *Conn4Center) UserSyncLoseScore(p *Player, timeUnix int64, timeStr, reason string) {
	loseOrder := p.Id + "_" + timeStr + "_lose"
	log.Debug("<-------- GenLoseOrder -------->: %v", loseOrder)

	baseData := &BaseMessage{}
	baseData.Event = msgUserLoseScore
	userLose := &UserChangeScore{}
	userLose.Auth.Token = c4c.token
	userLose.Auth.DevKey = c4c.DevKey
	userLose.Info.CreateTime = timeUnix
	userLose.Info.GameId = c4c.GameId
	userLose.Info.ID = p.Id
	userLose.Info.LockMoney = 0
	userLose.Info.Money = p.LoseResultMoney
	userLose.Info.Order = loseOrder
	userLose.Info.PayReason = reason
	userLose.Info.PreMoney = 0
	userLose.Info.RoundId = p.room.RoomId
	baseData.Data = userLose

	c4c.SendMsg2Center(baseData)
}

//UserSyncScoreChange 同步尚未同步过的输赢分
func (c4c *Conn4Center) UserSyncScoreChange(p *Player, reason string) {
	timeStr := time.Now().Format("2006-01-02_15:04:05")
	nowTime := time.Now().Unix()

	//同时同步赢分和输分
	c4c.UserSyncWinScore(p, nowTime, timeStr, reason)
	c4c.UserSyncLoseScore(p, nowTime, timeStr, reason)
}
