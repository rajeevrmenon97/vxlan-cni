package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/version"
)

var debug bool = true

type Network struct {
	Name    string `json:"name"`
	Dev     string `json:"dev"`
	VNI     int    `json:"vni"`
	Group   string `json:"vxlanGroup"`
	DstPort int    `json:"dstPort"`
}

func cmdAdd(args *skel.CmdArgs) error {
	if debug {
		r := strings.NewReplacer("\t", "", "\n", "")
		fmt.Printf("ContainerID:%s\nNetNS:%s\nIfName:%s\nArgs:%s\nPath:%s\nConfig:%s\n",
			args.ContainerID, args.Netns, args.IfName, args.Args, args.Path, r.Replace(string(args.StdinData[:])))
	}

	network := Network{}
	if err := json.Unmarshal(args.StdinData, &network); err != nil {
		return err
	}

	return nil
}

func cmdDel(args *skel.CmdArgs) error {
	return nil
}

func cmdCheck(args *skel.CmdArgs) error {
	return nil
}

func main() {
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, version.All, "CNI vxlan version 0.1")
}
