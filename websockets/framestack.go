package websockets

type FrameStack []Frame

func (stack FrameStack) GetFrame() Frame {
	return Frame{}
}
