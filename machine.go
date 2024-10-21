package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

const MachineTypeLighthouse = "lighthouse"
const MachineTypeWorkload = "workload"
const MachineTypeBuilder = "builder"
const MachineTypeBalancer = "balancer"

type Machine struct {
	Id             string
	VPNIp          string //IP inside VPN
	PublicIp       string //Public Ip
	CloudPrivateIp string //Private Ip inside data center
	Name           string
	Types          []string
}

type MachineVPNInfoDetails struct {
	Groups []string `json:"groups"`
	Ips    []string `json:"ips"`
	Name   string   `json:"name"`
}

type MachineVPNInfo struct {
	Details MachineVPNInfoDetails `json:"details"`
}

func handleMachinePost(w http.ResponseWriter, r *http.Request) {
	var machine Machine
	err := decodeJSONBody(w, r, &machine)

	if err != nil {
		var mr *malformedRequest
		if errors.As(err, &mr) {
			http.Error(w, mr.msg, mr.status)
		} else {
			log.Print(err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	addMachine(&machine)

	jsonBytes, err := json.Marshal(machine)
	if err != nil {
		fmt.Println("Cannot convert Proxy object into JSON:", err)
		return
	}

	fmt.Fprint(w, string(jsonBytes))

}

func handleMachineGet(w http.ResponseWriter, r *http.Request) {

	jsonBytes, err := json.Marshal(getMachines())
	if err != nil {
		fmt.Println("Cannot convert Services object into JSON:", err)
		return
	}

	fmt.Fprint(w, string(jsonBytes))
}

func loadMachineInfo() {

	fmt.Println("Loading info about this machine from certificate")
	cmd := exec.Command("nebula-cert", "print", "-json", "-path", "/etc/nebula/host.crt")
	nebulaCertOut, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("nebula-cert failed with %s\n", err)
	}

	dec := json.NewDecoder(strings.NewReader(string(nebulaCertOut)))

	var machineInfo MachineVPNInfo
	if err := dec.Decode(&machineInfo); err == io.EOF {
		return
	} else if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Name: %s\n", machineInfo.Details.Name)
	fmt.Printf("VPN ip: %s\n", machineInfo.Details.Ips[0])
}
