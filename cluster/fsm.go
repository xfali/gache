// Copyright (C) 2019, Xiongfa Li.
// All right reserved.
// @author xiongfa.li
// @version V1.0
// Description: 

package cluster

import (
    "gache/command"
    "gache/db"
    "github.com/hashicorp/go-msgpack/codec"
    "github.com/hashicorp/raft"
    "io"
    "sync"
)

type GacheFSM struct {
    sync.Mutex
    db *db.GacheDb
}

type GacheSnapshot struct {
    db *db.GacheDb
}

func (m *GacheFSM) Apply(log *raft.Log) interface{} {
    m.Lock()
    defer m.Unlock()

    var cmd command.Request
    err := cmd.Unmarshal(log.Data)
    if err != nil {
        return nil
    }
    _, procErr := cmd.Process(m.db)
    return procErr
}

func (m *GacheFSM) Snapshot() (raft.FSMSnapshot, error) {
    m.Lock()
    defer m.Unlock()

    return &GacheSnapshot{m.db}, nil
}

func (m *GacheFSM) Restore(inp io.ReadCloser) error {
    m.Lock()
    defer m.Unlock()
    defer inp.Close()
    hd := codec.MsgpackHandle{}
    dec := codec.NewDecoder(inp, &hd)

    return dec.Decode(&m.db)
}

func (m *GacheSnapshot) Persist(sink raft.SnapshotSink) error {
    hd := codec.MsgpackHandle{}
    enc := codec.NewEncoder(sink, &hd)
    if err := enc.Encode(m.db); err != nil {
        sink.Cancel()
        return err
    }
    sink.Close()
    return nil
}

func (m *GacheSnapshot) Release() {
}
