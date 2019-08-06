// Copyright (C) 2019, Xiongfa Li.
// All right reserved.
// @author xiongfa.li
// @version V1.0
// Description: 

package handler

import (
    "encoding/json"
    "errors"
    "fmt"
    "gache/cluster"
    "gache/cluster/gossip"
    "gache/command"
    "gache/config"
    "gache/db"
    "gache/utils"
    "log"
    "strings"
    "time"
)

type Context struct {
    raft       cluster.Replication
    db         *db.GacheDb
    leader     utils.AtomicBool
    cluster    gossip.Cluster
    clusterMgr ClusterManager
    self       NodeInfo
}

func NewContext(raft cluster.Replication, db *db.GacheDb) *Context {
    var dummyCluster gossip.DummyCluster = 1
    ret := &Context{
        raft:    raft,
        db:      db,
        leader:  0,
        cluster: &dummyCluster,
    }

    if raft == nil {
        ret.leader.Set()
    } else {
        raft.Listen(ret.Listen)
    }

    return ret
}

func (ctx *Context) Listen(v bool) {
    log.Printf("############ changed %v\n", v)
    if v {
        ctx.leader.Set()
    } else {
        ctx.leader.Unset()
    }
    node := ctx.self
    node.Master = ctx.leader.IsSet()
    b, _ := json.Marshal(node)
    ctx.cluster.UpdateLocal(b)
}

func (ctx *Context) IsLeader() bool {
    return ctx.leader.IsSet()
}

func (ctx *Context) SetCluster(conf *config.Config, c gossip.Cluster) {
    ctx.cluster = c
    ctx.self.Addr = c.LocalAddr()
    ctx.self.Master = ctx.leader.IsSet()
    s := strings.Split(c.LocalAddr(), ":")
    ctx.self.ApiAddr = fmt.Sprintf("%s:%d", s[0], conf.ApiPort)

    b, e, _ := cluster.GetSlots(conf.ClusterSlot)
    ctx.self.SlotBegin = b
    ctx.self.SlotEnd = e

    c.UpdateLocal(marshalMeta(ctx.self))
    ctx.clusterMgr.Join(ctx.self)
}

func (ctx *Context) ProcessCmd(cmdReq *command.Request, direct bool) (interface{}, error) {
    b, err := cmdReq.Marshal()
    if err != nil {
        return nil, err
    }
    if ctx.raft == nil || direct {
        return cmdReq.Process(ctx.db)
    } else {
        return nil, ctx.raft.Apply(b, 10*time.Second)
    }
}

func (ctx *Context) ReplicaJoin(addr string) error {
    return ctx.raft.Join(addr)
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
