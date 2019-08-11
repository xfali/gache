// Copyright (C) 2019, Xiongfa Li.
// All right reserved.
// @author xiongfa.li
// @version V1.0
// Description: 

package test

import (
    "net"
    "net/http"
    "time"
)

var transport = &http.Transport{
    Proxy: http.ProxyFromEnvironment,
    DialContext: (&net.Dialer{
        Timeout:   30 * time.Second,
        KeepAlive: 30 * time.Second,
        DualStack: true,
    }).DialContext,
    MaxIdleConns:          3000,
    MaxIdleConnsPerHost:   3000,
    MaxConnsPerHost:       5000,
    IdleConnTimeout:       90 * time.Second,
    TLSHandshakeTimeout:   10 * time.Second,
    ExpectContinueTimeout: 1 * time.Second,
}

var httpClient = &http.Client{
    Transport: transport,
    Timeout:   time.Duration(10 * time.Second) * time.Millisecond,
}
