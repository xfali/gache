// Copyright (C) 2019, Xiongfa Li.
// All right reserved.
// @author xiongfa.li
// @version V1.0
// Description: 

package test

import (
    "bytes"
    "fmt"
    "io"
    "io/ioutil"
    "testing"
)

func BenchmarkSet(b *testing.B) {
    addr := "localhost:8001"
    for i := 0; i < b.N; i++ {
        url := fmt.Sprintf("http://%s/key/%d", addr, i)
        post(url, []byte(url), b)
    }
}

func BenchmarkGet(b *testing.B) {
    addr := "localhost:8001"
    for i := 0; i < b.N; i++ {
        url := fmt.Sprintf("http://%s/key/%d", addr, i)
        get(url, b)
    }
}

func BenchmarkPSet(b *testing.B) {
    addr := "localhost:8001"
    b.RunParallel(func(pb *testing.PB) {
        i := 0
        for pb.Next() {
            i++
            url := fmt.Sprintf("http://%s/key/%d", addr, i)
            post(url, []byte(url), b)
        }
    })
}

func BenchmarkPGet(b *testing.B) {
    addr := "localhost:8001"
    b.RunParallel(func(pb *testing.PB) {
        i := 0
        for pb.Next() {
            i++
            url := fmt.Sprintf("http://%s/key/%d", addr, i)
            get(url, b)
        }
    })
}

func post(url string, data []byte, l testing.TB) {
    ret, err := httpClient.Post(url, "", bytes.NewReader(data))
    if err != nil {
        l.Fatal(err)
    }
    defer ret.Body.Close()
    io.Copy(ioutil.Discard, ret.Body)
}

func get(url string, l testing.TB) {
    ret, err := httpClient.Get(url)
    if err != nil {
        l.Fatal(err)
    }
    defer ret.Body.Close()
    io.Copy(ioutil.Discard, ret.Body)
}
