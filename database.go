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
				Query:     "CREATE TABLE Proxy (Id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, ContainerId TEXT, ServerPrivateIP TEXT, Port TEXT, Domain TEXT)",
				Arguments: []interface{}{},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot create table Proxy: %s\n", err.Error())
	}

	getAllProxies()
}

func addProxy(proxy *Proxy) {
	_, err := connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "INSERT INTO Proxy( ContainerId, ServerPrivateIP, Port, Domain) VALUES(?, ?, ?, ?)",
				Arguments: []interface{}{proxy.ContainerId, proxy.ServerPrivateIP, proxy.Port, proxy.Domain},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot write to Proxy table: %s\n", err.Error())
	}

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT MAX(Id) FROM Proxy LIMIT 1",
			Arguments: []interface{}{},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot read Max Id value from Proxy table: %s\n", err.Error())
	}

	for rows.Next() {
		var Id int64

		err := rows.Scan(&Id)
		if err != nil {
			fmt.Printf(" Cannot run Scan: %s\n", err.Error())
		} else {
			proxy.Id = Id
		}
	}
}

func deleteProxy(proxyId int64) (result bool) {
	_, err := connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "DELETE FROM Proxy WHERE Id = ?",
				Arguments: []interface{}{proxyId},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot delete a record from Proxy table: %s\n", err.Error())
		return false
	}

	return true
}

func getAllProxies() []Proxy {
	var proxies = []Proxy{}

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, ContainerId, ServerPrivateIP, Port, Domain from Proxy where Id > ?",
			Arguments: []interface{}{0},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot read from Proxy table: %s\n", err.Error())
	}

	for rows.Next() {
		var Id int64
		var ContainerId string
		var ServerPrivateIP string
		var Port string
		var Domain string

		err := rows.Scan(&Id, &ContainerId, &ServerPrivateIP, &Port, &Domain)
		if err != nil {
			fmt.Printf(" Cannot run Scan: %s\n", err.Error())
		}
		loadProxy := Proxy{
			Id:              Id,
			ContainerId:     ContainerId,
			ServerPrivateIP: ServerPrivateIP,
			Port:            Port,
			Domain:          Domain,
		}
		proxies = append(proxies, loadProxy)
	}
	return proxies
}
