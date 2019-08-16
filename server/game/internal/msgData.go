package internal

import (
	"encoding/json"
	"fmt"
)

const (
	RECODE_BREATHSTOP       = 1000
	RECODE_PLAYERDESTORY    = 1001
	RECODE_PLAYERBREAKLINE  = 1002
	RECODE_MONEYNOTFULL     = 1003
	RECODE_JOINROOMIDERR    = 1004
	RECODE_PEOPLENOTFULL    = 1005
	RECODE_SELLTENOTDOWNBET = 1006
)

var recodeText = map[int32]string{
	RECODE_BREATHSTOP:       "用户长时间未响应心跳,停止心跳",
	RECODE_PLAYERDESTORY:    "用户已在其他地方登录",
	RECODE_PLAYERBREAKLINE:  "玩家断开服务器连接,关闭链接",
	RECODE_MONEYNOTFULL:     "玩家金额不足,设为观战",
	RECODE_JOINROOMIDERR:    "请求加入的房间号不正确",
	RECODE_PEOPLENOTFULL:    "房间人数不够,不能开始游戏",
	RECODE_SELLTENOTDOWNBET: "当前结算阶段,不能进行操作",
}

func jsonData() {
	reCode, err := json.Marshal(recodeText)
	if err != nil {
		fmt.Println("json.Marshal err:", err)
		return
	}

	data := string(reCode)
	fmt.Println("S2C jsonData String ~", data)
}
