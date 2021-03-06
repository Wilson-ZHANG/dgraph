/*
 * Copyright 2018 Dgraph Labs, Inc.
 *
 * This file is available under the Apache License, Version 2.0,
 * with the Commons Clause restriction.
 */

package zero

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"runtime"

	"github.com/dgraph-io/dgraph/protos/intern"
	"github.com/dgraph-io/dgraph/x"
	"github.com/golang/glog"
)

type Telemetry struct {
	Arch        string
	Cid         string
	ClusterSize int
	DiskUsageMB int64
	NumAlphas   int
	NumGroups   int
	NumTablets  int
	NumZeros    int
	OS          string
	SinceHours  int
	Version     string
}

var keenUrl = "https://api.keen.io/3.0/projects/5b809dfac9e77c0001783ad0/events"

func newTelemetry(ms *intern.MembershipState) *Telemetry {
	if len(ms.Cid) == 0 {
		glog.V(2).Infoln("No CID found yet")
		return nil
	}
	t := &Telemetry{
		Cid:       ms.Cid,
		NumGroups: len(ms.GetGroups()),
		NumZeros:  len(ms.GetZeros()),
		Version:   x.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
	for _, g := range ms.GetGroups() {
		t.NumAlphas += len(g.GetMembers())
		for _, tablet := range g.GetTablets() {
			t.NumTablets++
			t.DiskUsageMB += tablet.GetSpace()
		}
	}
	t.DiskUsageMB /= (1 << 20)
	t.ClusterSize = t.NumAlphas + t.NumZeros
	return t
}

func (t *Telemetry) post() error {
	data, err := json.Marshal(t)
	if err != nil {
		return err
	}
	url := keenUrl + "/dev"
	if len(t.Version) > 0 {
		url = keenUrl + "/pings"
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "D0398E8C83BB30F67C519FDA6175975F680921890C35B36C34BE109544597497CA758881BD7D56CC2355A2F36B4560102CBC3279AC7B27E5391372C36A31167EB0D06BF3764894AD20A0554BAFF14C292A40BC252BB9FF008736A0FD1D44E085")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	glog.V(2).Infof("Telemetry response status: %v", resp.Status)
	glog.V(2).Infof("Telemetry response body: %s", body)
	return nil
}
