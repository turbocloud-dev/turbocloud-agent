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
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/dhamith93/systats"
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
	PublicSSHKey   string
}

type MachineStats struct {
	Id              string
	MachineId       string
	CPUUsage        int64
	AvailableMemory int64
	TotalMemory     int64
	AvailableDisk   int64
	TotalDisk       int64
	CreatedAt       string
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
	err := decodeJSONBody(w, r, &machine, true)

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

	addMachine(&machine, true)

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
		fmt.Println("Cannot convert Machines object into JSON:", err)
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
	machineId := r.PathValue("machineId")
	updateMachineStatus(machineId, MachineStatusProvision)

	zipPath := currentUser.HomeDir + "/" + secret + ".zip"

	http.ServeFile(w, r, zipPath)
}

func handlePublicSSHKeysGet(w http.ResponseWriter, r *http.Request) {
	jsonBytes, err := json.Marshal(getMachinesWithType(MachineTypeBuilder))
	if err != nil {
		fmt.Println("Cannot convert Services object into JSON:", err)
		return
	}

	fmt.Fprint(w, string(jsonBytes))
}

func handleMachineStatsGet(w http.ResponseWriter, r *http.Request) {

	var stats = []MachineStats{}

	//First we should get all machines
	machines := getMachines()

	for _, machine := range machines {

		rows, err := connection.QueryOneParameterized(
			gorqlite.ParameterizedStatement{
				Query:     "SELECT * from Stats" + machine.Id + " ORDER BY CreatedAt DESC LIMIT 1",
				Arguments: []interface{}{},
			},
		)

		if err != nil {
			fmt.Printf(" Cannot read from %s table: %s\n", "Stats"+machine.Id, err.Error())
		}

		rows.Next()

		var Id string
		var CPUUsage int64
		var AvailableMemory int64
		var TotalMemory int64
		var AvailableDisk int64
		var TotalDisk int64
		var CreatedAt string

		err = rows.Scan(&Id, &CPUUsage, &AvailableMemory, &TotalMemory, &AvailableDisk, &TotalDisk, &CreatedAt)
		if err != nil {
			fmt.Printf(" Cannot run Scan: %s\n", err.Error())
		}
		loadedStats := MachineStats{
			Id:              Id,
			MachineId:       machine.Id,
			CPUUsage:        CPUUsage,
			AvailableMemory: AvailableMemory,
			TotalMemory:     TotalMemory,
			AvailableDisk:   AvailableDisk,
			TotalDisk:       TotalDisk,
			CreatedAt:       CreatedAt,
		}
		stats = append(stats, loadedStats)

	}

	jsonBytes, err := json.Marshal(stats)
	if err != nil {
		fmt.Println("Cannot convert MachineStats{} object into JSON:", err)
		return
	}

	fmt.Fprint(w, string(jsonBytes))
}

func handleMachineDelete(w http.ResponseWriter, r *http.Request) {
	machineId := r.PathValue("id")

	if !deleteMachine(machineId) {
		fmt.Println("Cannot delete a record from Machine table")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "")

}

/*Internal*/
func addFirstMachine() {
	var machine Machine
	machine.Name = os.Getenv("TURBOCLOUD_VPN_NODE_NAME")
	machine.VPNIp = os.Getenv("TURBOCLOUD_VPN_NODE_PRIVATE_IP")
	machine.Types = append(machine.Types, MachineTypeLighthouse, MachineTypeBalancer, MachineTypeBuilder, MachineTypeWorkload)
	machine.Domains = append(machine.Domains, os.Getenv("TURBOCLOUD_AGENT_DOMAIN"))
	machine.Status = MachineStatusProvision

	addMachine(&machine, false)
}

/*Database*/
func addMachine(machine *Machine, isGenerateJoinURL bool) {
	id, err := NanoId(7)
	if err != nil {
		fmt.Println("Cannot generate new NanoId for Environment:", err)
		return
	}

	machine.Id = id

	if isGenerateJoinURL {
		joinSecret, err := NanoId(21)
		if err != nil {
			fmt.Println("Cannot generate new NanoId for Environment:", err)
			return
		}

		if len(thisMachine.Domains) > 0 {
			machine.JoinURL = thisMachine.Domains[0] + "/join/" + machine.Id + "/" + joinSecret
		}

		generateNewMachineJoinArchive(*machine, joinSecret)
	}

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "INSERT INTO Machine( Id, VPNIp, PublicIp, CloudPrivateIp, Name, Types, Status, Domains, JoinURL, PublicSSHKey) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
				Arguments: []interface{}{machine.Id, machine.VPNIp, machine.PublicIp, machine.CloudPrivateIp, machine.Name, strings.Join(machine.Types, ";"), machine.Status, strings.Join(machine.Domains, ";"), machine.JoinURL, ""},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot write to Machine table: %s\n", err.Error())
	}
}

func deleteMachine(machineId string) (result bool) {
	_, err := connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "DELETE FROM Machine WHERE Id = ?",
				Arguments: []interface{}{machineId},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot delete a record from Machine table: %s\n", err.Error())
		return false
	}

	return true
}

func getMachines() []Machine {
	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, VPNIp, PublicIp, CloudPrivateIp, Name, Types, Status, Domains, JoinURL, PublicSSHKey from Machine ORDER BY CreatedAt ASC",
			Arguments: []interface{}{},
		},
	)

	return handleMachineQuery(rows, err)
}

func getMachinesWithType(machineType string) []Machine {
	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, VPNIp, PublicIp, CloudPrivateIp, Name, Types, Status, Domains, JoinURL, PublicSSHKey from Machine WHERE Types LIKE ?",
			Arguments: []interface{}{"%" + machineType + "%"},
		},
	)

	return handleMachineQuery(rows, err)
}

func getMachinesByVPNIp(vpnIp string) []Machine {
	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, VPNIp, PublicIp, CloudPrivateIp, Name, Types, Status, Domains, JoinURL, PublicSSHKey from Machine WHERE VPNIp = ?",
			Arguments: []interface{}{vpnIp},
		},
	)

	return handleMachineQuery(rows, err)
}

func updatePublicSSHKey() {
	currentUser, err := user.Current()
	if err != nil {
		fmt.Println("Cannot get home directory, Image.go:", err)
	}
	homeDir := currentUser.HomeDir

	sshPubKeyBytes, err := os.ReadFile(homeDir + "/.ssh/id_rsa.pub") // just pass the file name
	if err != nil {
		fmt.Printf("Cannot get a public SSH key: %s\n", err.Error())
	}

	thisMachine.PublicSSHKey = string(sshPubKeyBytes)

	//Update SSH public key in DB
	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "UPDATE Machine Set PublicSSHKey = ? WHERE Id = ?",
				Arguments: []interface{}{thisMachine.PublicSSHKey, thisMachine.Id},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot update a row in Deployment: %s\n", err.Error())
		return
	}
}

func updateMachineStatus(machineId string, status string) bool {

	//Update machine status
	_, err := connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "UPDATE Machine Set Status = ? WHERE Id = ?",
				Arguments: []interface{}{status, machineId},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot update a row in Machine: %s\n", err.Error())
		return false
	}

	return true
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
		var PublicSSHKey string

		err := rows.Scan(&Id, &VPNIp, &PublicIp, &CloudPrivateIp, &Name, &Types, &Status, &Domains, &JoinURL, &PublicSSHKey)
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
			PublicSSHKey:   PublicSSHKey,
		}
		machines = append(machines, loadedMachine)
	}

	return machines

}

func loadInfoFromVPNCert() {
	fmt.Println("Loading info about this machine from certificate")
	cmd := exec.Command("nebula-cert", "print", "-json", "-path", "/etc/nebula/host.crt")
	nebulaCertOut, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("nebula-cert failed with %s\n", err)
	}

	dec := json.NewDecoder(strings.NewReader(string(nebulaCertOut)))

	var machineInfo MachineVPNInfo
	if err := dec.Decode(&machineInfo); err == io.EOF {
		return
	} else if err != nil {
		fmt.Print(err)
	}
	fmt.Printf("Machine Name: %s\n", machineInfo.Details.Name)
	fmt.Printf("Machine VPN ip: %s\n", machineInfo.Details.Ips[0])

	thisMachine.Name = machineInfo.Details.Name
	//machineInfo.Details.Ips comes in format 192.168.202.1/24 but we need just IP without a mask
	thisMachine.VPNIp = strings.Split(machineInfo.Details.Ips[0], "/")[0]

}

func updatePublicIp() {
	resp_ip, err := http.Get("https://turbocloud.dev/ip")
	if err != nil {
		fmt.Println("Cannot make a request to turbocloud.dev/ip" + err.Error())
	}
	body_ip, err := io.ReadAll(resp_ip.Body)
	if err != nil {
		fmt.Println("Cannot read a response from a request to turbocloud.dev/ip" + err.Error())
	}
	thisMachine.PublicIp = string(body_ip)

	//Update machine status
	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "UPDATE Machine Set PublicIp = ? WHERE Id = ?",
				Arguments: []interface{}{thisMachine.PublicIp, thisMachine.Id},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot update PublicIp in Machine: %s\n", err.Error())
	}

}

func loadMachineInfo() {

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

	updatePublicSSHKey()
	updatePublicIp()
}

func loadMachineStats() {

	createStatsTableIfNeeded()

	for range time.Tick(time.Second * 5) {
		go func() {

			syStats := systats.New()
			cpu, _ := syStats.GetCPU()
			memory, _ := syStats.GetMemory(systats.Megabyte)
			disks, _ := syStats.GetDisks()

			id, err := NanoId(7)
			if err != nil {
				fmt.Println("Cannot generate new NanoId for a new Stats row:", err)
				return
			}

			//Update SSH public key in DB
			_, err = connection.WriteParameterized(
				[]gorqlite.ParameterizedStatement{
					{
						Query:     "INSERT INTO " + "Stats" + thisMachine.Id + "( Id, CPUUsage, AvailableMemory, TotalMemory, AvailableDisk, TotalDisk) VALUES(?, ?, ?, ?, ?, ?)",
						Arguments: []interface{}{id, cpu.LoadAvg, memory.Available, memory.Total, disks[0].Usage.Available, disks[0].Usage.Size, thisMachine.Id},
					},
				},
			)

			if err != nil {
				fmt.Printf(" Cannot update a row in Deployment: %s\n", err.Error())
				return
			}

		}()
	}
}

// We ping machines only from lighthouses
func pingMachines() {

	// We ping machines only from lighthouses
	if !slices.Contains(thisMachine.Types, MachineTypeLighthouse) {
		return
	}

	isChecking := false

	for range time.Tick(time.Second * 3) {
		go func() {
			if isChecking {
				return
			}

			isChecking = true

			//First we should get all machines
			machines := getMachines()

			for _, machine := range machines {

				out, _ := exec.Command("ping", machine.VPNIp, "-c 1", "-w 1").Output()

				var status string
				if strings.Contains(string(out), "1 received") {
					//A machine is online
					status = MachineStatusOnline
				} else {
					//A machine is offline
					status = MachineStatusOffline
				}

				//Update machine status:
				//if new status == offline and current machine state == online
				if status == MachineStatusOffline && machine.Status == MachineStatusCreated {
					continue
				}

				if status == MachineStatusOffline && machine.Status == MachineStatusProvision {
					continue
				}

				_, err := connection.WriteParameterized(
					[]gorqlite.ParameterizedStatement{
						{
							Query:     "UPDATE Machine Set Status = ? WHERE Id = ?",
							Arguments: []interface{}{status, machine.Id},
						},
					},
				)

				if err != nil {
					fmt.Printf(" Cannot update a row in Deployment: %s\n", err.Error())
					return
				}
			}

			isChecking = false
		}()
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

	_, err = executeScriptString(scriptString, func(logLine string) {

	})
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
