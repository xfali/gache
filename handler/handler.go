// Copyright (C) 2019, Xiongfa Li.
// All right reserved.
// @author xiongfa.li
// @version V1.0
// Description: 

package handler

import (
    "encoding/json"
    "gache/command"
    "io"
    "io/ioutil"
    "net/http"
)

type Handler struct {
    methodMap map[string]http.HandlerFunc
    ctx       *Context
}

func New(ctx *Context) *Handler {
    ret := &Handler{
        methodMap: map[string]http.HandlerFunc{},
        ctx:       ctx,
    }
    ret.methodMap[http.MethodPost] = ret.create
    ret.methodMap[http.MethodPut] = ret.create
    ret.methodMap[http.MethodDelete] = ret.delete
    ret.methodMap[ http.MethodGet] = ret.get
    return ret
}

func (ctx *Handler) Handle(resp http.ResponseWriter, req *http.Request) {
    handleFunc := ctx.methodMap[req.Method]
    if handleFunc != nil {
        handleFunc(resp, req)
    } else {
        resp.WriteHeader(http.StatusBadRequest)
        resp.Write([]byte("method not support"))
    }
}

func (handler *Handler) create(resp http.ResponseWriter, req *http.Request) {
    key := getKey(req)
    if !handler.ctx.CheckSelf(key, true) {
        addr, err := handler.ctx.SelectClusterNode(key, true)
        if err != nil {
            resp.WriteHeader(http.StatusBadRequest)
            resp.Write([]byte(err.Error()))
            return
        }
        if addr != "" {
            handler.redirect(addr, resp, req)
            return
        }
    }

    if !handler.ctx.IsLeader() {
        resp.WriteHeader(http.StatusBadRequest)
        resp.Write([]byte("Not leader"))
        return
    }
    value, err := getValue(req)
    if err != nil {
        resp.WriteHeader(http.StatusBadRequest)
        resp.Write([]byte(err.Error()))
        return
    }

    cmdReq := command.Request{
        Cmd: command.SET,
        K:   key,
        V:   value,
    }

    _, procErr := handler.ctx.ProcessCmd(&cmdReq, false)
    if procErr != nil {
        resp.WriteHeader(http.StatusBadRequest)
        resp.Write([]byte(procErr.Error()))
        return
    }
}

func (handler *Handler) delete(resp http.ResponseWriter, req *http.Request) {
    key := getKey(req)
    if !handler.ctx.CheckSelf(key, true) {
        addr, err := handler.ctx.SelectClusterNode(key, true)
        if err != nil {
            resp.WriteHeader(http.StatusBadRequest)
            resp.Write([]byte(err.Error()))
            return
        }
        if addr != "" {
            handler.redirect(addr, resp, req)
            return
        }
    }

    if !handler.ctx.IsLeader() {
        resp.WriteHeader(http.StatusBadRequest)
        resp.Write([]byte("Not leader"))
        return
    }

    cmdReq := command.Request{
        Cmd: command.DEL,
        K:   key,
    }

    handler.ctx.ProcessCmd(&cmdReq, false)
}

func (handler *Handler) get(resp http.ResponseWriter, req *http.Request) {
    key := getKey(req)
    if !handler.ctx.CheckSelf(key, false) {
        addr, err := handler.ctx.SelectClusterNode(key, false)
        if err != nil {
            resp.WriteHeader(http.StatusBadRequest)
            resp.Write([]byte(err.Error()))
            return
        }
        if addr != "" {
            handler.redirect(addr, resp, req)
            return
        }
    }

    cmdReq := command.Request{
        Cmd: command.GET,
        K:   key,
    }
    v, procErr := handler.ctx.ProcessCmd(&cmdReq, true)
    if procErr != nil {
        resp.WriteHeader(http.StatusBadRequest)
        resp.Write([]byte(procErr.Error()))
        return
    }
    io.WriteString(resp, v.(string))
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

func (handler *Handler) Join(resp http.ResponseWriter, req *http.Request) {
    vars := req.URL.Query()
    addr := vars.Get("addr")

    err := handler.ctx.ReplicaJoin(addr)
    if err != nil {
        resp.WriteHeader(http.StatusBadRequest)
        resp.Write([]byte("join raft cluster failed"))
    }
}

func (handler *Handler) Cluster(resp http.ResponseWriter, req *http.Request) {
    ret, err := handler.ctx.LeaderNodes()
    if err != nil {
        resp.WriteHeader(http.StatusBadRequest)
        resp.Write([]byte(err.Error()))
        return
    }

    b, err := json.Marshal(ret)
    if err != nil {
        resp.WriteHeader(http.StatusBadRequest)
        resp.Write([]byte(err.Error()))
        return
    }

    resp.Write(b)
}

func (handler *Handler) redirect(addr string, resp http.ResponseWriter, req *http.Request) {
    //注意此处不使用StatusFound，由于302会出于安全考虑将POST重定向时修改为GET。使用307保持Method
    http.Redirect(resp, req, "http://"+addr+req.RequestURI, http.StatusTemporaryRedirect)
}
