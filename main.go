// Copyright (C) 2019, Xiongfa Li.
// All right reserved.
// @author xiongfa.li
// @version V1.0
// Description: 

package main

import (
    "flag"
    "fmt"
    "gache/config"
    "gache/db"
    "gache/handler"
    "gache/cluster"
    "log"
    "net/http"
)

func main() {
    port := flag.Int("p", 8080, "server port")
    addr := flag.String("raft-addr", ":12345", "raft tcp address")
    dir := flag.String("raft-dir", "/tmp", "raft dir")
    joinAddr := flag.String("raft-join", "", "raft join addr")

    flag.Parse()

    conf := &config.Config{
        RaftTcpAddr:  *addr,
        RaftDir:      *dir,
        RaftJoinAddr: *joinAddr,
    }

    gacheDb := db.New()
    notifyCh :=  make(chan bool, 1)
    c, err := cluster.New(conf, gacheDb, notifyCh)

    if err != nil {
        log.Fatal(err)
    }

    handler := handler.New(c, gacheDb, notifyCh)

    if conf.RaftJoinAddr != "" {
        cluster.Join(conf)
    }

    http.HandleFunc("/key/", handler.Handle)
    http.HandleFunc("/join", handler.Cluster)
    //设置访问的ip和端口
    err = http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
    if err != nil {
        log.Fatal("ListenAndServe:", err)
    }
}
