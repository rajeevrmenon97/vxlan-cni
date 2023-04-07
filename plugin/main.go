package main

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"syscall"

	"github.com/containernetworking/cni/pkg/skel"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/vishvananda/netlink"
)

var debug bool = true

type Network struct {
	Name    string `json:"name"`
	Dev     string `json:"dev"`
	VNI     int    `json:"vni"`
	Group   string `json:"group"`
	DstPort int    `json:"dstPort"`
}

func configureBridge(brName string) (*netlink.Bridge, error) {
	var br *netlink.Bridge
	var ok bool

	l, err := netlink.LinkByName(brName)
	if err == nil {
		br, ok = l.(*netlink.Bridge)
		if !ok {
			return nil, fmt.Errorf("%q already exists but is not a bridge", brName)
		}
	} else {
		if !strings.Contains(err.Error(), "Link not found") {
			return nil, fmt.Errorf("Error while querying for link: %v", err)
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
			return nil, err
		}

		if err := netlink.LinkSetUp(br); err != nil {
			return nil, err
		}
	}
	return br, nil
}

func configureVxlan(vxName string, network Network, br *netlink.Bridge) (*netlink.Vxlan, error) {
	var vx *netlink.Vxlan
	var ok bool
	parentLink, err := netlink.LinkByName(network.Dev)
	if err != nil {
		return nil, fmt.Errorf("Error while getting parent interface %q", network.Dev)
	}

	l, err := netlink.LinkByName(vxName)
	if err == nil {
		vx, ok = l.(*netlink.Vxlan)
		if !ok {
			return nil, fmt.Errorf("%q already exists but is not a vxlan link", vxName)
		}
	} else {
		if !strings.Contains(err.Error(), "Link not found") {
			return nil, fmt.Errorf("Error while querying for vxlan link: %v", err)
		}

		vx = &netlink.Vxlan{
			LinkAttrs: netlink.LinkAttrs{
				Name: vxName,
			},
			VxlanId:      network.VNI,
			VtepDevIndex: parentLink.Attrs().Index,
			Group:        net.ParseIP(network.Group),
			Port:         network.DstPort,
		}

		if err := netlink.LinkAdd(vx); err != nil && err != syscall.EEXIST {
			return nil, err
		}

		if err := netlink.LinkSetMaster(vx, br); err != nil {
			return nil, err
		}

		if err := netlink.LinkSetUp(vx); err != nil {
			return nil, err
		}
	}
	return vx, nil
}

func configureVethPairs(containerNs string, ifName string, IP string, br *netlink.Bridge) error {
	netns, err := ns.GetNS(containerNs)
	if err != nil {
		return fmt.Errorf("Error while getting container namespace %q: %v", containerNs, err)
	}

	hostIface := &current.Interface{}
	var handler = func(hostNS ns.NetNS) error {
		hostVeth, containerVeth, err := ip.SetupVeth(ifName, 1500, "", hostNS)
		if err != nil {
			return err
		}
		hostIface.Name = hostVeth.Name

		ipv4Addr, ipv4Net, err := net.ParseCIDR(IP)
		if err != nil {
			return err
		}

		link, err := netlink.LinkByName(containerVeth.Name)
		if err != nil {
			return err
		}

		ipv4Net.IP = ipv4Addr

		addr := &netlink.Addr{IPNet: ipv4Net, Label: ""}
		if err = netlink.AddrAdd(link, addr); err != nil {
			return err
		}

		if err := netlink.LinkSetUp(link); err != nil {
			return err
		}

		return nil
	}

	if err := netns.Do(handler); err != nil {
		return err
	}

	hostVeth, err := netlink.LinkByName(hostIface.Name)
	if err != nil {
		return err
	}

	if err := netlink.LinkSetMaster(hostVeth, br); err != nil {
		return err
	}

	return nil
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

	cniArgsPairs := strings.Split(args.Args, ";")
	cniArgsMap := make(map[string]string)
	for _, pair := range cniArgsPairs {
		keyValue := strings.Split(pair, "=")
		if len(keyValue) != 2 {
			panic("Invalid CNI_ARGS pair")
		}
		cniArgsMap[keyValue[0]] = keyValue[1]
	}
	IP, ok := cniArgsMap["IP"]
	if !ok {
		return fmt.Errorf("IP Address not provided")
	}

	// Configuring the bridge interface
	brName := "br-" + network.Name
	br, err := configureBridge(brName)
	if err != nil {
		return err
	}

	// Configuring the VXLAN interface
	vxName := "vxlan-" + network.Name
	_, err = configureVxlan(vxName, network, br)
	if err != nil {
		return err
	}

	// Configuring the veth pairs
	err = configureVethPairs(args.Netns, args.IfName, IP, br)
	if err != nil {
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
