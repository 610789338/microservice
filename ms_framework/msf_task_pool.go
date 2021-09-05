package ms_framework

import (
    "time"
)


type TaskPool struct {
    size            int  // 本质上就是协程size
    opCh            chan string
    sizeCh          chan int
    separateCh      chan func()
}

func (t *TaskPool) Start() {
    go t.Monitor()

    go func() {
        for {
            op := <- t.opCh
            switch op {
            case "add":
                t.size += 1

            case "del":
                t.size -= 1

            case "get":
                t.sizeCh <- t.size
            }
        }
    } ()

    go func() {
        for {
            separate := <- t.separateCh
            t.opCh <- "add"
            separate()
            t.opCh <- "del"
        }
    } ()
}

// 每隔1s检测一下pool size，大于0打印出来
func (t *TaskPool) Monitor() {
    ch := time.After(time.Second * 1)

    for {
        _ = <- ch

        size := t.Size()
        if size > 0 {
            WARN_LOG("[task pool] - pool size %d", size)
        }

        ch = time.After(time.Second * 1)
    }
}

func (t *TaskPool) ProduceTask(task func()) {
    t.opCh <- "add"
    defer func() {t.opCh <- "del"} ()
    task()
}

func (t *TaskPool) ProduceTaskSeparate(task func()) {
    t.separateCh <- task
}

func (t *TaskPool) Size() int {
    t.opCh <- "get"
    size := <- t.sizeCh
    return size
}

var gTaskPool *TaskPool = nil

func StartTaskPool() {
    gTaskPool = &TaskPool{opCh: make(chan string), sizeCh: make(chan int), separateCh: make(chan func())}
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
