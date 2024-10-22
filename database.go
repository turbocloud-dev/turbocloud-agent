package main

import (
	"fmt"

	"github.com/rqlite/gorqlite"
)

var connection *gorqlite.Connection

func databaseInit() {
	var err error
	connection, err = gorqlite.Open("http://") // same only explicitly
	if err != nil {
		fmt.Printf(" Cannot open database: %s\n", err.Error())
	}

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "CREATE TABLE Machine (Id TEXT NOT NULL PRIMARY KEY, VPNIp TEXT, PublicIp TEXT, CloudPrivateIp TEXT, Name TEXT, Types TEXT, CreatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL)",
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
				Query:     "CREATE TABLE Proxy (Id TEXT NOT NULL PRIMARY KEY, ContainerId TEXT, ServerPrivateIP TEXT, Port TEXT, Domain TEXT, CreatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL)",
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
				Query:     "CREATE TABLE Environment (Id TEXT NOT NULL PRIMARY KEY, ServiceId TEXT, Name TEXT, Branch TEXT, Domains TEXT, Port TEXT, MachineIds TEXT, CreatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL)",
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
				Query:     "CREATE TABLE Image (Id TEXT NOT NULL PRIMARY KEY, Status TEXT, DeploymentId TEXT, ErrorMsg TEXT, CreatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL)",
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

	getAllProxies()
}
