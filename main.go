// Copyright (C) 2019, Xiongfa Li.
// All right reserved.
// @author xiongfa.li
// @version V1.0
// Description: 

package main

import (
    "flag"
    "fmt"
    "gache/cluster"
    "gache/cluster/gossip"
    "gache/config"
    "gache/db"
    "gache/handler"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
)

func main() {
    port := flag.Int("p", 8000, "server port")
    addr := flag.String("raft-addr", "", "raft tcp address, format: :7000")
    dir := flag.String("raft-dir", "/tmp", "raft dir")
    joinAddr := flag.String("raft-join", "", "raft join addr")
    gossipPort := flag.Int("cluster-port", 9000, "cluster port")
    gossipMember := flag.String("cluster-members", "", "member list: HOST1:PORT1,HOST2:PORT2,HOST3:PORT3")
    gossipSlots := flag.String("cluster-slot", "", "Slot: 0-16383")

    flag.Parse()

    conf := &config.Config{
        RaftTcpAddr:  *addr,
        RaftDir:      *dir,
        RaftJoinAddr: *joinAddr,

        ClusterPort:     *gossipPort,
        ClusterMemebers: *gossipMember,
        ClusterSlot:     *gossipSlots,

        ApiPort: *port,
    }

    gacheDb := db.New()
    notifyCh := make(chan bool, 1)
    var servers []shutdown
    if conf.RaftTcpAddr != "" {
        raft, err := cluster.New(conf, gacheDb, notifyCh)
        if err != nil {
            log.Fatal(err)
        }
        servers = append(servers, func() error {
            raft.Shutdown()
            return nil
        })
    }

    handler := handler.New(nil, gacheDb, notifyCh)

    if conf.RaftJoinAddr != "" {
        cluster.Join(conf)
    }

    if conf.ClusterSlot != "" {
        c, err := gossip.Startup(conf, &gossip.NodeDelegate{
            Enabled:    true,
            JoinFunc:   handler.NodeJoin,
            LeaveFunc:  handler.NodeLeave,
            UpdateFunc: handler.NodeUpdate,
        })
        if err != nil {
            closeAll(servers)
            os.Exit(-1)
        }
        handler.SetCluster(conf, c)
        servers = append(servers, c.Close)
    }

    http.HandleFunc("/key/", handler.Handle)
    http.HandleFunc("/join", handler.Cluster)
    //设置访问的ip和端口
    s := &http.Server{
        Addr:           fmt.Sprintf(":%d", conf.ApiPort),
        Handler:        nil,
        ReadTimeout:    15 * time.Second,
        WriteTimeout:   15 * time.Second,
        IdleTimeout:    15 * time.Second,
        MaxHeaderBytes: 1 << 20,
    }

    go s.ListenAndServe()
    servers = append(servers, s.Close)

    handleSignal(servers)
}

type shutdown func() error

func handleSignal(c []shutdown) {
    quitChan := make(chan os.Signal)
    signal.Notify(quitChan,
        syscall.SIGINT,
        syscall.SIGTERM,
        syscall.SIGHUP,
    )
    <-quitChan

    closeAll(c)
    log.Println("server gracefully shutdown")
    close(quitChan)
}

func closeAll(c []shutdown) {
    for _, v := range c {
        if err := v(); nil != err {
            log.Fatalf("server shutdown failed, err: %v\n", err)
        }
    }
}
