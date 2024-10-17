package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
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
