// Copyright (C) 2019, Xiongfa Li.
// All right reserved.
// @author xiongfa.li
// @version V1.0
// Description: 

package db

import "sync"

type GacheDb struct {
    Table map[string]string
    mutex sync.RWMutex
}

func New() *GacheDb {
    return &GacheDb{Table: map[string]string{}}
}

func (db *GacheDb) Set(k, v string) error {
    db.mutex.Lock()
    defer db.mutex.Unlock()

    db.Table[k] = v
    return nil
}

func (db *GacheDb) Get(k string) string {
    db.mutex.Lock()
    defer db.mutex.Unlock()

    return db.Table[k]
}

func (db *GacheDb) Delete(k string) error {
    db.mutex.Lock()
    defer db.mutex.Unlock()

    delete(db.Table, k)
    return nil
}
