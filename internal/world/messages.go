package world

type Msg interface{ isMsg() }

func (MsgInput) isMsg() {}

type MsgChooseUpgrade struct {
	Choice int // 0 or 1
}

func (MsgChooseUpgrade) isMsg() {}

type MsgRestart struct{}

func (MsgRestart) isMsg() {}

type MsgTogglePause struct{}

func (MsgTogglePause) isMsg() {}

type MsgSaveSnapshot struct {
	Path  string
	Reply chan<- error
}

func (MsgSaveSnapshot) isMsg() {}

type MsgLoadSnapshot struct {
	Path  string
	Reply chan<- error
}

func (MsgLoadSnapshot) isMsg() {}
