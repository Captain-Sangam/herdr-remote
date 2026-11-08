package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"time"
)

// The CLI talks to the daemon over admin.sock: one JSON request line, one
// JSON response line, and the daemon answers within a second.

type adminReq struct {
	Cmd string `json:"cmd"`
	ID  string `json:"id,omitempty"`
}

type adminResp struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

func (d *Daemon) serveAdmin() error {
	os.Remove(d.adminPath)
	ln, err := net.Listen("unix", d.adminPath)
	if err != nil {
		return err
	}
	if err := os.Chmod(d.adminPath, 0o600); err != nil {
		ln.Close()
		return err
	}
	d.adminLn = ln
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go d.adminConn(c)
		}
	}()
	return nil
}

func (d *Daemon) adminConn(c net.Conn) {
	defer c.Close()
	c.SetDeadline(time.Now().Add(5 * time.Second))
	line, err := bufio.NewReaderSize(c, 64<<10).ReadBytes('\n')
	if err != nil && len(line) == 0 {
		return
	}
	var req adminReq
	resp := adminResp{OK: true}
	if err := json.Unmarshal(line, &req); err != nil {
		resp = adminResp{Error: "bad request"}
	} else if res, err := d.handleAdmin(req); err != nil {
		resp = adminResp{Error: err.Error()}
	} else if resp.Result, err = json.Marshal(res); err != nil {
		resp = adminResp{Error: err.Error()}
	}
	b, _ := json.Marshal(resp)
	c.Write(append(b, '\n'))
}

func (d *Daemon) handleAdmin(req adminReq) (any, error) {
	switch req.Cmd {
	case "pair":
		return d.pair.open()
	case "pair_status":
		return d.pair.status(), nil
	case "pair_allow":
		return "ok", d.pair.allow()
	case "pair_deny":
		return "ok", d.pair.deny()
	case "pair_cancel":
		d.pair.cancel()
		return "ok", nil
	case "devices_ls":
		out := []map[string]string{}
		for _, x := range d.devices.all() {
			out = append(out, map[string]string{"id": x.ID, "name": x.Name, "added": x.Added})
		}
		return out, nil
	case "devices_rm":
		return "ok", d.revoke(req.ID)
	}
	return nil, fmt.Errorf("unknown command %q", req.Cmd)
}

// adminCall is the client side.
func adminCall(sockPath string, req adminReq) (json.RawMessage, error) {
	c, err := net.DialTimeout("unix", sockPath, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("is herdr-remoted running? %w", err)
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(5 * time.Second))
	b, _ := json.Marshal(req)
	if _, err := c.Write(append(b, '\n')); err != nil {
		return nil, err
	}
	line, err := bufio.NewReaderSize(c, 1<<20).ReadBytes('\n')
	if err != nil && len(line) == 0 {
		return nil, err
	}
	var resp adminResp
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, errors.New(resp.Error)
	}
	return resp.Result, nil
}
