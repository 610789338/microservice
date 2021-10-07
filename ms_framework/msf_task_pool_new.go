package ms_framework

import (
    "time"
    // "sync"
    atomic "sync/atomic"
)

var coroutinesSize int = 500

type TaskPool struct {
    taskCh          chan func()
    size            int32             // 任务数量
    separateCh      chan func()
    // lock            sync.RWMutex
}

func (t *TaskPool) Start() {
    go t.Monitor()

    for i := 0; i < coroutinesSize; i++ {
        go func() {
            for {
                task := <- t.taskCh
                task()

                atomic.AddInt32(&t.size, -1)
            }
        } ()
    }

    go func() {
        for {
            separate := <- t.separateCh
            separate()

            atomic.AddInt32(&t.size, -1)
        }
    } ()
}

// 每隔1s检测一下pool size，大于0打印出来
func (t *TaskPool) Monitor() {
    ch := time.After(time.Second * 1)

    for {
        _ = <- ch

        size := t.Size()
        if size != 0 {
            WARN_LOG("[task pool] - pool size %d %d", size, t.size)
        }

        ch = time.After(time.Second * 1)
    }
}

func (t *TaskPool) Size() int32 {
    return atomic.LoadInt32(&t.size)
}

func (t *TaskPool) ProduceTask(task func()) {
    atomic.AddInt32(&t.size, 1)
    t.taskCh <- task
}

func (t *TaskPool) ProduceTaskSeparate(task func()) {
    atomic.AddInt32(&t.size, 1)
    t.separateCh <- task
}

var gTaskPool *TaskPool = nil

func StartTaskPool() {
    // taskCh size 1000w
    gTaskPool = &TaskPool{taskCh: make(chan func(), 10000000), separateCh: make(chan func())}
    gTaskPool.Start()
}

func StopTaskPool() {
    time.Sleep(time.Second)

    for {
        if 0 == gTaskPool.Size() {
            break
        }
        time.Sleep(time.Microsecond)
    }
}
