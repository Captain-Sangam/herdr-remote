package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

const version = "0.1.0"

func main() {
	args := os.Args[1:]
	asJSON := false
	var rest []string
	for _, a := range args {
		if a == "--json" {
			asJSON = true
		} else {
			rest = append(rest, a)
		}
	}
	if len(rest) == 0 {
		usage()
	}
	dir := defaultStateDir()
	var req adminReq
	switch rest[0] {
	case "version":
		fmt.Println(version)
		return
	case "serve":
		serve(dir)
		return
	case "pair":
		sub := ""
		if len(rest) > 1 {
			sub = rest[1]
		}
		switch sub {
		case "":
			req.Cmd = "pair"
		case "status", "allow", "deny", "cancel":
			req.Cmd = "pair_" + sub
		default:
			usage()
		}
	case "devices":
		if len(rest) < 2 {
			usage()
		}
		switch rest[1] {
		case "ls":
			req.Cmd = "devices_ls"
		case "rm":
			if len(rest) < 3 {
				usage()
			}
			req.Cmd, req.ID = "devices_rm", rest[2]
		default:
			usage()
		}
	default:
		usage()
	}
	res, err := adminCall(filepath.Join(dir, "admin.sock"), req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "herdr-remoted:", err)
		os.Exit(1)
	}
	if asJSON || req.Cmd == "pair" {
		os.Stdout.Write(append(res, '\n'))
		return
	}
	switch req.Cmd {
	case "devices_ls":
		var list []map[string]string
		json.Unmarshal(res, &list)
		if len(list) == 0 {
			fmt.Println("no paired devices")
		}
		for _, d := range list {
			fmt.Printf("%s  %s  %s\n", d["id"], d["added"], d["name"])
		}
	case "pair_status":
		var st pairStatus
		json.Unmarshal(res, &st)
		fmt.Println(st.State)
		if st.DeviceName != "" {
			fmt.Printf("phone: %s  code: %s\n", st.DeviceName, st.Code)
		}
	default:
		fmt.Println("ok")
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: herdr-remoted serve | pair [status|allow|deny|cancel] [--json] | devices ls [--json] | devices rm <id> | version")
	os.Exit(2)
}

func serve(dir string) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		log.Fatal(err)
	}
	logPath := filepath.Join(dir, "daemon.log")
	if w, err := openRotating(logPath); err == nil {
		os.Chmod(logPath, 0o600) // launchd creates it with the default mode
		log.SetOutput(w)
	}
	r := &runner{progs: findPrograms()}
	if r.progs.HC == "" || r.progs.HerdrApp == "" {
		log.Fatal("hc or herdr-app not found")
	}
	d, err := newDaemon(dir, r, ":7766", ":7767")
	if err != nil {
		log.Fatal(err)
	}
	if err := d.start(); err != nil {
		log.Fatal(err)
	}
	log.Printf("herdr-remoted %s listening on %d, pin %s", version, d.pairedPort, d.pin())
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	d.stop()
}
