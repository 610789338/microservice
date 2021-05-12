package ms_framework


import (
	"time"
)


type FlowVelocityCounter struct {
	counter		string
	lastTime 	int64
	cnt 		int32
	totalCnt  	int64
	velocity    int32
	output		bool

	ch 			chan string
}

func (f *FlowVelocityCounter) Start() {
	if nil == f.ch {
		panic("ch nil")
	}

	f.lastTime = GetNowTimestampMs()

	INFO_LOG("FlowVelocityCounter Start %+v", f)

	go func() {
		for true {
			select {
			case op := <- f.ch:
				// INFO_LOG("FlowVelocityCounter count %s %+v", op, f)

				switch op {
				case "c":
					f.cnt += 1
					f.totalCnt += 1

				case "r":
					nowMs := GetNowTimestampMs()

					if nowMs > f.lastTime {
						f.velocity = int32(float64(f.cnt)/float64(nowMs - f.lastTime)*1000)

						if f.velocity > 0 && f.output {
							INFO_LOG("[@@@ FlowVelocityCount: %s] - velocity: %v/s  total: %v", f.counter, f.velocity, f.totalCnt)
						}

						f.lastTime = nowMs
						f.cnt = 0
					}
				}
			}
		}
	} ()

	go func() {
		for true {
			select {
			case <- time.After(time.Second * 2):
				f.ch <- "r"
			}
		}
	} ()
}

func (f *FlowVelocityCounter) Count() {
	f.ch <- "c"
}

// func init() {

// 	fvc := &FlowVelocityCounter{counter: "RpcOps", output: true, ch: make(chan string)}
// 	fvc.Start()

// 	for true {
// 		fvc.Count()
// 		time.Sleep(time.Millisecond)
// 	}
// }
