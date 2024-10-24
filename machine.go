package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"

	"github.com/rqlite/gorqlite"
)

// Machine Types
const MachineTypeLighthouse = "lighthouse"
const MachineTypeWorkload = "workload"
const MachineTypeBuilder = "builder"
const MachineTypeBalancer = "balancer"

// Machine Statuses
const MachineStatusCreated = "created"
const MachineStatusProvision = "provision"
const MachineStatusOnline = "online"
const MachineStatusOffline = "offline"

type Machine struct {
	Id             string
	VPNIp          string //IP inside VPN
	PublicIp       string //Public Ip
	CloudPrivateIp string //Private Ip inside data center
	Name           string
	Types          []string
	Status         string
	Domains        []string
	JoinURL        string
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

	//Generate VPN Ip
	const privateIpMask = "192.168.202."
	const IpMaskMin = 2
	const IpMaskMax = 254
	randomMask := rand.IntN(IpMaskMax+1-IpMaskMin) + IpMaskMin

	for len(getMachinesByVPNIp(privateIpMask+strconv.Itoa(randomMask))) > 0 {
		randomMask = rand.IntN(IpMaskMax+1-IpMaskMin) + IpMaskMin
	}

	machine.VPNIp = privateIpMask + strconv.Itoa(randomMask)
	machine.Status = MachineStatusCreated

	joinSecret, err := NanoId(21)
	if err != nil {
		fmt.Println("Cannot generate new NanoId for Environment:", err)
		return
	}

	if len(thisMachine.Domains) > 0 {
		machine.JoinURL = thisMachine.Domains[0] + "/join/" + joinSecret
	}

	addMachine(&machine)

	generateNewMachineJoinArchive(machine, joinSecret)

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

func handleJoinGet(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "applicaiton/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=turbo_join.zip")

	currentUser, err := user.Current()
	if err != nil {
		fmt.Println("Cannot get current user, machine.go:", err)
	}
	secret := r.PathValue("secret")

	zipPath := currentUser.HomeDir + "/" + secret + ".zip"

	http.ServeFile(w, r, zipPath)
}

func addFirstMachine() {
	var machine Machine
	machine.Name = os.Getenv("TURBOCLOUD_VPN_NODE_NAME")
	machine.VPNIp = os.Getenv("TURBOCLOUD_VPN_NODE_PRIVATE_IP")
	machine.Types = append(machine.Types, MachineTypeLighthouse, MachineTypeBalancer, MachineTypeBuilder, MachineTypeWorkload)
	machine.Domains = append(machine.Domains, os.Getenv("TURBOCLOUD_AGENT_DOMAIN"))
	machine.Status = MachineStatusProvision

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
				Query:     "INSERT INTO Machine( Id, VPNIp, PublicIp, CloudPrivateIp, Name, Types, Status, Domains, JoinURL) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)",
				Arguments: []interface{}{machine.Id, machine.VPNIp, machine.PublicIp, machine.CloudPrivateIp, machine.Name, strings.Join(machine.Types, ";"), machine.Status, strings.Join(machine.Domains, ";"), machine.JoinURL},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot write to Machine table: %s\n", err.Error())
	}
}

func getMachines() []Machine {
	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, VPNIp, PublicIp, CloudPrivateIp, Name, Types, Status, Domains, JoinURL from Machine",
			Arguments: []interface{}{},
		},
	)

	return handleMachineQuery(rows, err)
}

func getMachinesByVPNIp(vpnIp string) []Machine {
	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, VPNIp, PublicIp, CloudPrivateIp, Name, Types, Status, Domains, JoinURL from Machine WHERE VPNIp = ?",
			Arguments: []interface{}{vpnIp},
		},
	)

	return handleMachineQuery(rows, err)
}

func handleMachineQuery(rows gorqlite.QueryResult, err error) []Machine {

	var machines = []Machine{}

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
		var Status string
		var Domains string
		var JoinURL string

		err := rows.Scan(&Id, &VPNIp, &PublicIp, &CloudPrivateIp, &Name, &Types, &Status, &Domains, &JoinURL)
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
			Status:         Status,
			Domains:        strings.Split(Domains, ";"),
			JoinURL:        JoinURL,
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
	//machineInfo.Details.Ips comes in format 192.168.202.1/24 but we need just IP without a mask
	thisMachine.VPNIp = strings.Split(machineInfo.Details.Ips[0], "/")[0]

	//Load other details about this machine from DB by machine_name
	machines := getMachines()

	for _, machine := range machines {
		if machine.Name == thisMachine.Name {
			thisMachine.Id = machine.Id
			thisMachine.PublicIp = machine.PublicIp
			thisMachine.CloudPrivateIp = machine.CloudPrivateIp
			thisMachine.Types = machine.Types
			thisMachine.Domains = machine.Domains
		}
	}

}

func generateNewMachineJoinArchive(newMachine Machine, joinSecret string) {

	scriptTemplate := createTemplate("add_machine", `
	#!/bin/sh
	cd {{.HOME_DIR}}
	rm {{.MACHINE_NAME}}.crt
	rm {{.MACHINE_NAME}}.key
	nebula-cert sign -ca-crt /etc/nebula/ca.crt -ca-key /etc/nebula/ca.key -name "{{.MACHINE_NAME}}" -ip "{{.MACHINE_VPN_IP}}/24" -groups "devs" && sudo ufw allow from {{.MACHINE_VPN_IP}}
`)
	currentUser, err := user.Current()
	if err != nil {
		fmt.Println("Cannot get home directory, Image.go:", err)
	}

	homeDir := currentUser.HomeDir + "/"

	var templateBytes bytes.Buffer
	templateData := map[string]string{
		"HOME_DIR":       homeDir,
		"MACHINE_NAME":   newMachine.Name,
		"MACHINE_VPN_IP": newMachine.VPNIp,
	}

	if err := scriptTemplate.Execute(&templateBytes, templateData); err != nil {
		fmt.Println("Cannot execute template for Caddyfile:", err)
	}

	scriptString := templateBytes.String()

	err = executeScriptString(scriptString)
	if err != nil {
		fmt.Println("Cannot generate new certificates for a new machine")
		return
	}

	createZipArchive(newMachine, joinSecret, homeDir)
}

func createZipArchive(newMachine Machine, joinSecret string, homeDir string) {
	fmt.Println("Creating zip archive with certificates")
	archive, err := os.Create(homeDir + joinSecret + ".zip")
	if err != nil {
		fmt.Println("Cannot create zip archive with certificates: ", err)
		return
	}
	defer archive.Close()
	zipWriter := zip.NewWriter(archive)

	//Adding ca.crt
	caFile, err := os.Open("/etc/nebula/ca.crt")
	if err != nil {
		fmt.Println("Cannot open /etc/nebula/ca.crt to add to an archive: ", err)
	}
	defer caFile.Close()

	w1, err := zipWriter.Create("ca.crt")
	if err != nil {
		fmt.Println("Cannot create records for ca.crt in the archive: ", err)
	}
	if _, err := io.Copy(w1, caFile); err != nil {
		fmt.Println("Cannot copy ca.crt to the archive: ", err)
	}

	//Adding node_config.yaml
	node_config, err := os.Open(homeDir + "node_config.yaml")
	if err != nil {
		fmt.Println("Cannot open HOME/node_config.yaml to add to an archive: ", err)
	}
	defer node_config.Close()

	w1, err = zipWriter.Create("config.yaml")
	if err != nil {
		fmt.Println("Cannot create records for node_config.yaml in the archive: ", err)
	}
	if _, err := io.Copy(w1, node_config); err != nil {
		fmt.Println("Cannot copy node_config.yaml to the archive: ", err)
	}

	//Adding crt
	crtPath := homeDir + newMachine.Name + ".crt"
	crtFile, err := os.Open(crtPath)
	if err != nil {
		fmt.Println("Cannot open "+crtPath+" to add to an archive: ", err)
	}
	defer crtFile.Close()

	w1, err = zipWriter.Create("host.crt")
	if err != nil {
		fmt.Println("Cannot create records for host.crt in the archive: ", err)
	}
	if _, err := io.Copy(w1, crtFile); err != nil {
		fmt.Println("Cannot copy host.crt to the archive: ", err)
	}

	//Adding key
	keyPath := homeDir + newMachine.Name + ".key"
	keyFile, err := os.Open(keyPath)
	if err != nil {
		fmt.Println("Cannot open "+keyPath+" to add to an archive: ", err)
	}
	defer keyFile.Close()

	w1, err = zipWriter.Create("host.key")
	if err != nil {
		fmt.Println("Cannot create records for host.crt in the archive: ", err)
	}
	if _, err := io.Copy(w1, keyFile); err != nil {
		fmt.Println("Cannot copy host.crt to the archive: ", err)
	}

	fmt.Println("Closing zip with join certificates")
	zipWriter.Close()

}
