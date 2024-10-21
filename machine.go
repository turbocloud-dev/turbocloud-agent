package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/rqlite/gorqlite"
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

var thisMachine Machine

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

func addFirstMachine() {
	var machine Machine
	machine.Name = os.Getenv("TURBOCLOUD_VPN_NODE_NAME")
	machine.VPNIp = os.Getenv("TURBOCLOUD_VPN_NODE_PRIVATE_IP")
	machine.Types = append(machine.Types, MachineTypeLighthouse, MachineTypeBalancer, MachineTypeBuilder, MachineTypeWorkload)
	addMachine(&machine)
}

/*Database*/
func addMachine(machine *Machine) {
	id, err := NanoId(7)
	if err != nil {
		fmt.Println("Cannot generate new NanoId for Environment:", err)
		return
	}

	machine.Id = id

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "INSERT INTO Machine( Id, VPNIp, PublicIp, CloudPrivateIp, Name, Types) VALUES(?, ?, ?, ?, ?, ?)",
				Arguments: []interface{}{machine.Id, machine.VPNIp, machine.PublicIp, machine.CloudPrivateIp, machine.Name, strings.Join(machine.Types, ";")},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot write to Machine table: %s\n", err.Error())
	}
}

func getMachines() []Machine {
	var machines = []Machine{}

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, VPNIp, PublicIp, CloudPrivateIp, Name, Types from Machine",
			Arguments: []interface{}{},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot read from Environment table: %s\n", err.Error())
	}

	for rows.Next() {
		var Id string
		var VPNIp string
		var PublicIp string
		var CloudPrivateIp string
		var Name string
		var Types string

		err := rows.Scan(&Id, &VPNIp, &PublicIp, &CloudPrivateIp, &Name, &Types)
		if err != nil {
			fmt.Printf(" Cannot run Scan: %s\n", err.Error())
		}
		loadedMachine := Machine{
			Id:             Id,
			VPNIp:          VPNIp,
			PublicIp:       PublicIp,
			CloudPrivateIp: CloudPrivateIp,
			Name:           Name,
			Types:          strings.Split(Types, ";"),
		}
		machines = append(machines, loadedMachine)
	}

	return machines
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
	fmt.Printf("Machine Name: %s\n", machineInfo.Details.Name)
	fmt.Printf("Machine VPN ip: %s\n", machineInfo.Details.Ips[0])

	thisMachine.Name = machineInfo.Details.Name
	thisMachine.VPNIp = machineInfo.Details.Ips[0]

	//Load other details about this machine from DB by machine_name
	machines := getMachines()

	for _, machine := range machines {
		if machine.Name == thisMachine.Name {
			thisMachine.Id = machine.Id
			thisMachine.PublicIp = machine.PublicIp
			thisMachine.CloudPrivateIp = machine.CloudPrivateIp
			thisMachine.Types = machine.Types
		}
	}

}
