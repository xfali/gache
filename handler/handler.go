// Copyright (C) 2019, Xiongfa Li.
// All right reserved.
// @author xiongfa.li
// @version V1.0
// Description: 

package handler

import (
    "gache/cluster"
    "gache/command"
    "gache/db"
    "github.com/hashicorp/raft"
    "io"
    "io/ioutil"
    "log"
    "net/http"
    "sync/atomic"
    "time"
)

type Handler struct {
    methodMap map[string]http.HandlerFunc
    raft      *raft.Raft
    db        *db.GacheDb
    leader    AtomicBool
}

func New(raft *raft.Raft, db *db.GacheDb, notifyCh chan bool) *Handler {
    ret := &Handler{
        methodMap: map[string]http.HandlerFunc{},
        raft:      raft,
        db:        db,
        leader:    0,
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
                break
            default:
                break
            }
        }
    }()

    return ret
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
    if !ctx.leader.IsSet() {
        resp.WriteHeader(http.StatusBadRequest)
        resp.Write([]byte("Not leader"))
    }
    value, err := getValue(req)
    if err != nil {
        resp.WriteHeader(http.StatusBadRequest)
        return
    }
    key := getKey(req)

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
    ctx.raft.Apply(b, 10*time.Second)
}

func (ctx *Handler) delete(resp http.ResponseWriter, req *http.Request) {
    if !ctx.leader.IsSet() {
        resp.WriteHeader(http.StatusBadRequest)
        resp.Write([]byte("Not leader"))
    }

    key := getKey(req)
    cmdReq := command.Request{
        Cmd: command.DEL,
        K:   key,
    }
    b, err := cmdReq.Marshal()
    if err != nil {
        resp.WriteHeader(http.StatusBadRequest)
        return
    }
    ctx.raft.Apply(b, 10*time.Second)
}

func (ctx *Handler) get(resp http.ResponseWriter, req *http.Request) {
    key := getKey(req)
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

type AtomicBool int32

func (b *AtomicBool) IsSet() bool { return atomic.LoadInt32((*int32)(b)) == 1 }
func (b *AtomicBool) Set()        { atomic.StoreInt32((*int32)(b), 1) }
func (b *AtomicBool) Unset()      { atomic.StoreInt32((*int32)(b), 0) }
