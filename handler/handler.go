// Copyright (C) 2019, Xiongfa Li.
// All right reserved.
// @author xiongfa.li
// @version V1.0
// Description: 

package handler

import (
    "encoding/json"
    "fmt"
    "gache/cluster"
    "gache/cluster/gossip"
    "gache/command"
    "gache/config"
    "gache/db"
    "github.com/hashicorp/raft"
    "io"
    "io/ioutil"
    "log"
    "net/http"
    "strings"
    "sync/atomic"
    "time"
)

type Handler struct {
    methodMap  map[string]http.HandlerFunc
    raft       *raft.Raft
    db         *db.GacheDb
    leader     AtomicBool
    cluster    gossip.Cluster
    clusterMgr ClusterManager
    self       NodeInfo
}

func New(raft *raft.Raft, db *db.GacheDb, notifyCh chan bool) *Handler {
    var dummyCluster defaultCluster = 1
    ret := &Handler{
        methodMap: map[string]http.HandlerFunc{},
        raft:      raft,
        db:        db,
        leader:    0,
        cluster:   &dummyCluster,
    }
    if raft == nil {
        ret.leader.Set()
    }
    ret.methodMap[http.MethodPost] = ret.create
    ret.methodMap[http.MethodPut] = ret.create
    ret.methodMap[http.MethodDelete] = ret.delete
    ret.methodMap[ http.MethodGet] = ret.get

    go func() {
        for {
            select {
            case v, ok := <-notifyCh:
                if !ok {
                    return
                }
                log.Printf("############ changed %v\n", v)
                if v {
                    ret.leader.Set()
                } else {
                    ret.leader.Unset()
                }
                node := ret.self
                node.Master = ret.leader.IsSet()
                b, _ := json.Marshal(node)
                ret.cluster.UpdateLocal(b)
                break
            default:
                break
            }
        }
    }()

    return ret
}

func (ctx *Handler) SetCluster(conf *config.Config, c gossip.Cluster) {
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

func (ctx *Handler) Handle(resp http.ResponseWriter, req *http.Request) {
    handleFunc := ctx.methodMap[req.Method]
    if handleFunc != nil {
        handleFunc(resp, req)
    } else {
        resp.WriteHeader(http.StatusBadRequest)
    }
}

func (ctx *Handler) create(resp http.ResponseWriter, req *http.Request) {
    key := getKey(req)
    clusterRet := ctx.checkCluster(key, true, resp, req)
    if clusterRet == MOVE {
        return
    }
    if  clusterRet != OK {
        resp.WriteHeader(http.StatusBadRequest)
        return
    }
    if !ctx.leader.IsSet() {
        resp.WriteHeader(http.StatusBadRequest)
        resp.Write([]byte("Not leader"))
    }
    value, err := getValue(req)
    if err != nil {
        resp.WriteHeader(http.StatusBadRequest)
        return
    }

    cmdReq := command.Request{
        Cmd: command.SET,
        K:   key,
        V:   value,
    }
    b, err := cmdReq.Marshal()
    if err != nil {
        resp.WriteHeader(http.StatusBadRequest)
        return
    }
    if ctx.raft == nil {
        ctx.db.Set(key, value)
    } else {
        ctx.raft.Apply(b, 10*time.Second)
    }
}

func (ctx *Handler) delete(resp http.ResponseWriter, req *http.Request) {
    key := getKey(req)
    clusterRet := ctx.checkCluster(key, true, resp, req)
    if clusterRet == MOVE {
        return
    }
    if  clusterRet != OK {
        resp.WriteHeader(http.StatusBadRequest)
        return
    }
    if !ctx.leader.IsSet() {
        resp.WriteHeader(http.StatusBadRequest)
        resp.Write([]byte("Not leader"))
    }

    cmdReq := command.Request{
        Cmd: command.DEL,
        K:   key,
    }
    b, err := cmdReq.Marshal()
    if err != nil {
        resp.WriteHeader(http.StatusBadRequest)
        return
    }
    if ctx.raft == nil {
        ctx.db.Delete(key)
    } else {
        ctx.raft.Apply(b, 10*time.Second)
    }
}

func (ctx *Handler) get(resp http.ResponseWriter, req *http.Request) {
    key := getKey(req)
    clusterRet := ctx.checkCluster(key, true, resp, req)
    if clusterRet == MOVE {
        return
    }
    if  clusterRet != OK {
        resp.WriteHeader(http.StatusBadRequest)
        return
    }
    v := ctx.db.Get(key)

    io.WriteString(resp, v)
}

func getKey(req *http.Request) string {
    return req.RequestURI[5:]
}

func getValue(req *http.Request) (string, error) {
    b, err := ioutil.ReadAll(req.Body)
    if err != nil {
        return "", err
    }
    return string(b), nil
}

func (ctx *Handler) Cluster(resp http.ResponseWriter, req *http.Request) {
    vars := req.URL.Query()
    addr := vars.Get("addr")

    err := cluster.DoJoin(addr, ctx.raft)
    if err != nil {
        resp.WriteHeader(http.StatusBadRequest)
    }
}

func (ctx *Handler) checkCluster(key string, master bool, resp http.ResponseWriter, req *http.Request) int32 {
    addr, status := ctx.clusterMgr.FindNode(key, master)
    if status == OK && addr != ctx.self.ApiAddr {
        log.Printf("move to %s\n", addr)
        //注意此处不使用StatusFound，由于302会出于安全考虑将POST重定向时修改为GET。使用307保持Method
        http.Redirect(resp, req, "http://" + addr + req.RequestURI, http.StatusTemporaryRedirect)
        return MOVE
    }
    return status
}

func (ctx *Handler) NodeJoin(meta []byte) {
    log.Printf("defaultJoin meta: %s\n", string(meta))

    ctx.clusterMgr.Join(unmarshalMeta(meta))
}

func (ctx *Handler) NodeLeave(meta []byte) {
    log.Printf("defaultLeave meta: %s\n", string(meta))
    ctx.clusterMgr.Leave(unmarshalMeta(meta))
}

func (ctx *Handler) NodeUpdate(meta []byte) {
    log.Printf("defaultUpdate meta: %s\n", string(meta))
    ctx.clusterMgr.Update(unmarshalMeta(meta))
}

func marshalMeta(node NodeInfo) []byte {
    b, err := json.Marshal(node)
    log.Printf("%v\n", err)
    return b
}

func unmarshalMeta(meta []byte) NodeInfo {
    ret := NodeInfo{}
    err := json.Unmarshal(meta, &ret)
    log.Printf("%v\n", err)
    return ret
}

type defaultCluster int

func (c *defaultCluster) LocalAddr() string {
    return ""
}

func (c *defaultCluster) UpdateLocal(meta []byte) error {
    return nil
}

func (c *defaultCluster) UpdateAndWait(meta []byte, timeout time.Duration) error {
    return nil
}

func (c *defaultCluster) Close() error {
    return nil
}

type AtomicBool int32

func (b *AtomicBool) IsSet() bool { return atomic.LoadInt32((*int32)(b)) == 1 }
func (b *AtomicBool) Set()        { atomic.StoreInt32((*int32)(b), 1) }
func (b *AtomicBool) Unset()      { atomic.StoreInt32((*int32)(b), 0) }
