package world

type Msg interface{ isMsg() }

type MsgChooseUpgrade struct {
	Choice int // 0 or 1
}

func (MsgChooseUpgrade) isMsg() {}
