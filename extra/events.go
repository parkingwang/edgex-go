package extra

// 常用的事件结构

const (
	DirectIn  = byte('I')
	DirectOut = byte('O')
)

// 事件类型
const (
	TypeNop    = "NOP"    // 无操作
	TypeCard   = "CARD"   // 刷卡
	TypeButton = "BUTTON" // 开关事件
	TypeOpen   = "OPEN"   // 开门事件
	TypeClose  = "CLOSE"  // 关门事件
	TypeAlarm  = "ALARM"  // 报警事件
)

// 刷卡事件
type CardEvent struct {
	SerialNum uint32 `json:"sn"`      // 控制序列号
	BoardId   uint32 `json:"boardId"` // 控制主板ID
	DoorId    byte   `json:"doorId"`  // 门号
	Direct    byte   `json:"direct"`  // 进出方向
	CardNO    string `json:"card"`    // 卡号
	Type      string `json:"type"`    // 事件类型
	State     string `json:"state"`   // 状态
	Index     uint32 `json:"index"`   // 内部流水号
}

// 返回方向名称
func DirectName(dir byte) string {
	if dir == DirectIn {
		return "IN"
	} else {
		return "OUT"
	}
}
