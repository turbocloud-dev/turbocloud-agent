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
				Query:     "CREATE TABLE Proxy (Id TEXT NOT NULL PRIMARY KEY, ContainerId TEXT, ServerPrivateIP TEXT, Port TEXT, Domain TEXT)",
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
				Query:     "INSERT INTO Proxy( Id, ContainerId, ServerPrivateIP, Port, Domain) VALUES(?, ?, ?, ?, ?)",
				Arguments: []interface{}{proxy.Id, proxy.ContainerId, proxy.ServerPrivateIP, proxy.Port, proxy.Domain},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot write to Proxy table: %s\n", err.Error())
	}

}

func deleteProxy(proxyId string) (result bool) {
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
		var Id string
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
