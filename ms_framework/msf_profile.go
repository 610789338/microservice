package ms_framework

import (
    "os"
    "os/signal"
    "syscall"
    "fmt"
    "time"
    "runtime"
    "runtime/pprof"
)

type ProfileMgr struct {
    started bool
    cpu_prof *os.File
    mem_prof *os.File
}

func (p *ProfileMgr) Watch() {
    ch_start := make(chan os.Signal, 1)
    ch_stop := make(chan os.Signal, 1)

    signal.Notify(ch_start, syscall.Signal(0xA)) // kill -10 pid
    signal.Notify(ch_stop, syscall.Signal(0xC))  //kill -12 pid

    for true {
        select {
        case <-ch_start:
            p.Start()
        case <-ch_stop:
            p.Stop()
        }
    }
}

func (p *ProfileMgr) Start() {
    if p.started {
        ERROR_LOG("profile already started")
        return
    }
    
    var err error
    dir, err := os.Getwd()
    if err != nil {
        ERROR_LOG("profile start failed: get current dir failed: %s", err.Error())
        return
    }

    suffix := fmt.Sprintf("_%d_%d.prof", os.Getpid(), time.Now().Unix())
    cpu_filename := fmt.Sprintf("%s/%s_prof_cpu_%s", dir, GlobalCfg.Service, suffix)
    mem_filename := fmt.Sprintf("%s/%s_prof_mem_%s", dir, GlobalCfg.Service, suffix)

    p.cpu_prof, err = os.Create(cpu_filename)
    if err != nil {
        ERROR_LOG("profile started failed: create cpu file failed: %s", err.Error())
        return
    }

    err = pprof.StartCPUProfile(p.cpu_prof)
    if err != nil {
        p.cpu_prof.Close()
        p.mem_prof.Close()
        ERROR_LOG("profile start failed: %s", err.Error())
        return
    }

    p.mem_prof, err = os.Create(mem_filename)
    if err != nil {
        p.cpu_prof.Close()
        ERROR_LOG("profile start failed: create mem file failed: %s", err.Error())
        return
    }

    INFO_LOG("cpu & mem profile start ...")

    p.started = true
}

func (p *ProfileMgr) Stop() {
    if !p.started {
        ERROR_LOG("profile not started")
        return
    }

    p.started = false

    pprof.StopCPUProfile()
    INFO_LOG("cpu profile data saved as %s", p.cpu_prof.Name())
    p.cpu_prof.Close()

    runtime.GC()
    if err := pprof.WriteHeapProfile(p.mem_prof); err != nil {
        ERROR_LOG("mem profile save failed: %s", err.Error())
    } else {
        INFO_LOG("mem profile saved as %s", p.mem_prof.Name())
    }
    p.mem_prof.Close()
}

var profileMgr ProfileMgr

func init() {
    go profileMgr.Watch()
}