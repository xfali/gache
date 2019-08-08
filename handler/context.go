// Copyright (C) 2019, Xiongfa Li.
// All right reserved.
// @author xiongfa.li
// @version V1.0
// Description: 

package handler

import (
    "errors"
    "fmt"
    "gache/cluster"
    "gache/cluster/gossip"
    "gache/command"
    "gache/config"
    "gache/db"
    "log"
    "strings"
    "sync"
    "time"
)

type Context struct {
    raft       cluster.Replication
    db         *db.GacheDb
    cluster    gossip.Cluster
    clusterMgr ClusterManager
    self       NodeInfo
    mu         sync.Mutex
}

func NewContext(raft cluster.Replication, db *db.GacheDb) *Context {
    var dummyCluster gossip.DummyCluster = 1
    ret := &Context{
        raft:    raft,
        db:      db,
        cluster: &dummyCluster,
    }

    if raft == nil {
        ret.self.Master = true
    } else {
        raft.Listen(ret.Listen)
    }

    return ret
}

func (ctx *Context) Listen(v bool) {
    log.Printf("############ changed %v\n", v)
    ctx.mu.Lock()
    ctx.self.Master = v
    ctx.mu.Unlock()

    ctx.NotifySelf()
}

func (ctx *Context) IsLeader() bool {
    ctx.mu.Lock()
    defer ctx.mu.Unlock()
    return ctx.self.Master
}

func (ctx *Context) SetCluster(conf *config.Config, c gossip.Cluster) {
    ctx.cluster = c
    ctx.self.Addr = c.LocalAddr()
    //ctx.self.Master = ctx.leader.IsSet()
    s := strings.Split(c.LocalAddr(), ":")
    ctx.self.ApiAddr = fmt.Sprintf("%s:%d", s[0], conf.ApiPort)

    b, e, _ := cluster.GetSlots(conf.ClusterSlot)
    ctx.self.SlotBegin = b
    ctx.self.SlotEnd = e

    ctx.NotifySelf()
}

func (ctx *Context) NotifySelf() {
    if ctx.cluster.Enabled() {
        ctx.cluster.UpdateLocal(marshalMeta(ctx.self))
        ctx.clusterMgr.Update(ctx.self)
    }
}

func (ctx *Context) ProcessCmd(cmdReq *command.Request, direct bool) (interface{}, error) {
    if ctx.raft == nil || direct {
        return cmdReq.Process(ctx.db)
    } else {
        b, err := cmdReq.Marshal()
        if err != nil {
            return nil, err
        }
        return nil, ctx.raft.Apply(b, 10*time.Second)
    }
}

func (ctx *Context) ReplicaJoin(addr string) error {
    return ctx.raft.Join(addr)
}

func (ctx *Context) LeaderNodes() ([]NodeInfo, error) {
    return ctx.clusterMgr.Leaders()
}

func (ctx *Context) SelectClusterNode(key string, master bool) (string, error) {
    addr, status := ctx.clusterMgr.FindNode(key, master)
    if status == OK && addr != ctx.self.ApiAddr {
        return addr, nil
    }
    if status == NOT_READY {
        return "", errors.New("Cluster is not ready ")
    }
    return "", nil
}

func (ctx *Context) CheckSelf(key string, leader bool) bool {
    if !ctx.cluster.Enabled() {
        return true
    }

    ctx.mu.Lock()
    defer ctx.mu.Unlock()

    if leader && !ctx.self.Master {
        return false
    }
    slot := CalcSlot(key)
    return ctx.self.CheckSlot(slot)
}

func (ctx *Context) NodeJoin(meta []byte) {
    log.Printf("defaultJoin meta: %s\n", string(meta))

    ctx.clusterMgr.Join(unmarshalMeta(meta))
}

func (ctx *Context) NodeLeave(meta []byte) {
    log.Printf("defaultLeave meta: %s\n", string(meta))
    ctx.clusterMgr.Leave(unmarshalMeta(meta))
}

func (ctx *Context) NodeUpdate(meta []byte) {
    log.Printf("defaultUpdate meta: %s\n", string(meta))
    ctx.clusterMgr.Update(unmarshalMeta(meta))
}
