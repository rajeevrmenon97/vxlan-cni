package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"syscall"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/vishvananda/netlink"
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

	brName := "br-" + network.Name
	var br *netlink.Bridge
	var ok bool

	l, err := netlink.LinkByName(brName)
	if err == nil {
		br, ok = l.(*netlink.Bridge)
		if !ok {
			return fmt.Errorf("%q already exists but is not a bridge", brName)
		}
	} else {
		if !strings.Contains(err.Error(), "Link not found") {
			return fmt.Errorf("Error while querying for link: %v", err)
		}

		br = &netlink.Bridge{
			LinkAttrs: netlink.LinkAttrs{
				Name:   brName,
				MTU:    1500,
				TxQLen: -1,
			},
		}

		err := netlink.LinkAdd(br)
		if err != nil && err != syscall.EEXIST {
			return err
		}

		if err := netlink.LinkSetUp(br); err != nil {
			return err
		}
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
