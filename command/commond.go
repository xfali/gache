// Copyright (C) 2019, Xiongfa Li.
// All right reserved.
// @author xiongfa.li
// @version V1.0
// Description: 

package command

import (
    "encoding/json"
    "errors"
    "gache/db"
)

const (
    SET = "SET"
    DEL = "DEL"
    GET = "GET"
)

type Request struct {
    Cmd string
    K   string
    V   string
}

type processFunc func(db *db.GacheDb, req *Request) error

var gCmds = map[string]processFunc{
    SET:    ProcessSet,
    DEL:    ProcessDel,
}

type Command interface {
    Process(db *db.GacheDb)
}

func (req *Request) Process(db *db.GacheDb) error {
    if f, ok := gCmds[req.Cmd]; ok {
        return f(db, req)
    }

    return errors.New("Command not found")
}

func (req *Request) Marshal() ([]byte, error) {
    return json.Marshal(req)
}

func (req *Request) Unmarshal(bytes []byte) error {
    return json.Unmarshal(bytes, req)
}

func ProcessSet(db *db.GacheDb, req *Request) error {
    return db.Set(req.K, req.V)
}

func ProcessDel(db *db.GacheDb, req *Request) error {
    return db.Delete(req.K)
}
