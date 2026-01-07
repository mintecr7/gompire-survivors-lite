package world

type Msg interface{ isMsg() }

func (MsgInput) isMsg() {}

type MsgChooseUpgrade struct {
	Choice int // 0 or 1
}

func (MsgChooseUpgrade) isMsg() {}

type MsgRestart struct{}

func (MsgRestart) isMsg() {}
