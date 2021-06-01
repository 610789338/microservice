package ms_framework


import (
	"time"
)


type FlowVelocityCounter struct {
	Counter		string

	lastTime 	int64
	cnt 		int32
	totalCnt  	int64
	velocity    int32
	frequency   time.Duration  // ms

	ch 			chan string
}

func (f *FlowVelocityCounter) Start() {
	f.ch = make(chan string)
	if 0 == f.frequency {
		f.frequency = 2000  // default 2s
	}

	f.lastTime = GetNowTimestampMs()

	time2Count := time.After(time.Millisecond * f.frequency)

	INFO_LOG("FlowVelocityCounter Start %+v", f)

	go func() {
		stop := false
		for !stop {
			select {
			case op := <- f.ch:
				// INFO_LOG("FlowVelocityCounter count %s %+v", op, f)

				switch op {
				case "c":
					f.cnt += 1
					f.totalCnt += 1

				case "s":
					INFO_LOG("FlowVelocityCounter stop %+v", f)
					stop = true
				}

			case <- time2Count:
				nowMs := GetNowTimestampMs()

				if nowMs > f.lastTime {
					f.velocity = int32(float64(f.cnt)/float64(nowMs - f.lastTime)*1000)

					if f.velocity > 0 {
						WARN_LOG("flow velocity counter - %s - velocity: %v/s  total: %v", f.Counter, f.velocity, f.totalCnt)
					}

					f.lastTime = nowMs
					f.cnt = 0
				}
				
				time2Count = time.After(time.Millisecond * f.frequency)
			}
		}
	} ()
}

func (f *FlowVelocityCounter) Stop() {
	f.ch <- "s"
}

func (f *FlowVelocityCounter) Count() {
	f.ch <- "c"
}

func (f *FlowVelocityCounter) GetTotalCount() int64 {
	return f.totalCnt
}

var rpcFvc *FlowVelocityCounter

func StartRpcFvc() {
	rpcFvc = &FlowVelocityCounter{Counter: "rpc ops"}
	rpcFvc.Start()
}

func RpcFvcCount() {
	rpcFvc.Count()
}

func StopRpcFvc() {
	rpcFvc.Stop()
}

// func init() {

// 	fvc := &FlowVelocityCounter{Counter: "fvc test"}
// 	fvc.Start()

// 	go func() {
// 		loop := 300000
// 		for loop > 0 {
// 			fvc.Count()
// 			time.Sleep(time.Microsecond)
// 			loop -= 1
// 		}

// 		fvc.Stop()
// 	} ()
// }
