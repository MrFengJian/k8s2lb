package main

import (
        "os"
        "fmt"
        . "k8s2lb/util"
        "flag"
        "k8s2lb/transformer"
        "time"
)

const VERSION="1.0"

func main() {
        configPath := flag.String("conf", "./conf.json", "config file for k8s2lb")
        version:=flag.Bool("version",false,"show version")

        flag.Parse()
        if *version{
                fmt.Println(VERSION)
                os.Exit(0)
        }
        var conf *Conf
        conf, err := LoadConf(*configPath)
        if err != nil {
                fmt.Fprintln(os.Stderr, err)
                os.Exit(1)
        }
        fmt.Println("conf is %v", conf)
        if conf.AutoCleanOrphanPorts {
                go func() {
                        for {
                                err := transformer.SyncPods(conf)
                                if err != nil {
                                        fmt.Fprintln(os.Stderr, "error happens while sync pods.just ignore to next turn", err)
                                }
                                time.Sleep(conf.OrphanPortsResyncInterval * time.Second)
                        }
                }()
        }
        go transformer.SyncServices(conf)
        select {}
}
