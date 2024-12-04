package main

import (
	"fmt"
	"os"
	"time"

	"github.com/rqlite/gorqlite"
)

var connection *gorqlite.Connection

func databaseInit() {
	var err error

	dbURL := "http://" + thisMachine.VPNIp + ":4001"

	registryEnv, isRegistryEnvExists := os.LookupEnv("TURBOCLOUD_DB_HOST")
	if isRegistryEnvExists {
		dbURL = "http://" + registryEnv + ":4001"
	}

	connection, err = gorqlite.Open(dbURL)
	for err != nil {
		fmt.Printf(" Cannot open database: %s\n", err.Error())
		fmt.Println("Will retry to connect after 1 second")
		time.Sleep(1 * time.Second)
		connection, err = gorqlite.Open(dbURL)
	}

	// get rqlite cluster information
	leader, err := connection.Leader()

	for err != nil || leader == "" {
		fmt.Printf(" Cannot get DB leader")
		fmt.Println("Will retry to get a leader after 1 second")
		time.Sleep(1 * time.Second)
		connection, _ = gorqlite.Open(dbURL)
		leader, err = connection.Leader()
	}

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "CREATE TABLE Machine (Id TEXT NOT NULL PRIMARY KEY, VPNIp TEXT, PublicIp TEXT, CloudPrivateIp TEXT, Name TEXT, Types TEXT, Status TEXT, Domains TEXT, JoinURL TEXT, PublicSSHKey TEXT, CreatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL)",
				Arguments: []interface{}{},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot create table Machine: %s\n", err.Error())
	} else {
		//If err == nil, we connect to DB the first time and should add the root/first machine ourselves
		addFirstMachine()
	}

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "CREATE TABLE Proxy (Id TEXT NOT NULL PRIMARY KEY, ContainerId TEXT, ServerPrivateIP TEXT, Port TEXT, Domain TEXT, EnvironmentId TEXT, DeploymentId TEXT, CreatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL)",
				Arguments: []interface{}{},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot create table Proxy: %s\n", err.Error())
	}

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "CREATE TABLE Service (Id TEXT NOT NULL PRIMARY KEY, Name TEXT, ProjectId TEXT, GitURL TEXT, CreatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL)",
				Arguments: []interface{}{},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot create table Service: %s\n", err.Error())
	}

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "CREATE TABLE Environment (Id TEXT NOT NULL PRIMARY KEY, ServiceId TEXT, Name TEXT, Branch TEXT, Domains TEXT, Port TEXT, MachineIds TEXT, GitTag TEXT CreatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL)",
				Arguments: []interface{}{},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot create table Environment: %s\n", err.Error())
	}

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "CREATE TABLE Deployment (Id TEXT NOT NULL PRIMARY KEY, Status TEXT, EnvironmentId TEXT, ImageId TEXT, CreatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL)",
				Arguments: []interface{}{},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot create table Deployment: %s\n", err.Error())
	}

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "CREATE TABLE Image (Id TEXT NOT NULL PRIMARY KEY, Status TEXT, DeploymentId TEXT, EnvironmentId TEXT, ErrorMsg TEXT, CreatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL)",
				Arguments: []interface{}{},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot create table Image: %s\n", err.Error())
	}

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "CREATE TABLE DeploymentJob (Id TEXT NOT NULL PRIMARY KEY, Status TEXT, DeploymentId TEXT, MachineId TEXT, CreatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL)",
				Arguments: []interface{}{},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot create table DeploymentJob: %s\n", err.Error())
	}

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "CREATE TABLE ContainerJob (Id TEXT NOT NULL PRIMARY KEY, Status TEXT, EnvironmentId TEXT, MachineId TEXT, JobType TEXT, CreatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL)",
				Arguments: []interface{}{},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot create table ContainerJob: %s\n", err.Error())
	}

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "CREATE TABLE ImageJob (Id TEXT NOT NULL PRIMARY KEY, Status TEXT, EnvironmentId TEXT, JobType TEXT, CreatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL)",
				Arguments: []interface{}{},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot create table ImageJob: %s\n", err.Error())
	}

	//getAllProxies()
}

func createStatsTableIfNeeded() {
	_, err := connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "CREATE TABLE " + "Stats" + thisMachine.Id + " (Id TEXT NOT NULL PRIMARY KEY, CPUUsage INTEGER, AvailableMemory INTEGER, TotalMemory INTEGER, AvailableDisk INTEGER, TotalDisk INTEGER, CreatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL)",
				Arguments: []interface{}{},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot create table %s: %s\n", "Stats"+thisMachine.Id, err.Error())
	}
}
