package websockets

type FrameQueue []Frame

func (queue *FrameQueue) Enqueue(frame Frame) {
	*queue = append(*queue, frame)
}

// Returns first frame from the queue and boolean flag "ok"
func (queue *FrameQueue) Dequeue() (Frame, bool) {
	if len(*queue) < 1 {
		return Frame{}, false
	}
	frame := (*queue)[0]
	*queue = (*queue)[1:]
	return frame, true
}
