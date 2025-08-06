package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/rqlite/gorqlite"
)

const DatabaseStatusScheduled = "scheduled_to_deploy"
const DatabasetatusStartingContainers = "starting_containers"
const DatabaseStatusFinished = "finished"
const DatabaseStatusToDelete = "scheduled_to_delete"
const DatabaseStatusDeleted = "deleted"

type Database struct {
	Id         string
	Name       string
	ImageName  string
	VolumeId   string
	Domains    []string
	MachineIds []string
	Status     string
	ContPort   string // A port inside a container or exposed port: docker ... -p HostPort:ContPort
	HostPort   string // A port on a server (host):  docker ... -p HostPort:ContPort, generated on a machine where a container with DB will be deployed
	DataPath   string
	ProjectId  string
	CreatedAt  string
}

type DatabaseVolume struct {
	Id        string
	Name      string
	ImageName string
}

func handleDatabasePost(w http.ResponseWriter, r *http.Request) {
	var database Database
	err := decodeJSONBody(w, r, &database, true)

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

	addDatabase(&database)

	jsonBytes, err := json.Marshal(database)
	if err != nil {
		fmt.Println("Cannot convert Proxy object into JSON:", err)
		return
	}

	fmt.Fprint(w, string(jsonBytes))

}

func handleDatabaseGet(w http.ResponseWriter, r *http.Request) {

	jsonBytes, err := json.Marshal(getAllDatabase())
	if err != nil {
		fmt.Println("Cannot convert Services object into JSON:", err)
		return
	}

	fmt.Fprint(w, string(jsonBytes))
}

func handleDatabaseDelete(w http.ResponseWriter, r *http.Request) {

	serviceId := r.PathValue("id")

	if !scheduleDeleteDatabase(serviceId) {
		fmt.Println("Cannot delete a record from Service table")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "")
}

/*Database*/
func addDatabase(database *Database) {

	id, err := NanoId(7)
	if err != nil {
		fmt.Println("Cannot generate new NanoId for Database:", err)
		return
	}

	database.Id = id
	database.Status = DatabaseStatusScheduled

	addDatabaseVolume(database)

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "INSERT INTO Database( Id, Name, ImageName, VolumeId, Domains, MachineIds, Status, ContPort, DataPath, ProjectId) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
				Arguments: []interface{}{database.Id, database.Name, database.ImageName, database.VolumeId, strings.Join(database.Domains, ";"), strings.Join(database.MachineIds, ";"), database.Status, database.ContPort, database.DataPath, database.ProjectId},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot write to Database table: %s\n", err.Error())
	}

}

func addDatabaseVolume(database *Database) {

	var databaseVolume DatabaseVolume

	id, err := NanoId(7)
	if err != nil {
		fmt.Println("Cannot generate new NanoId for DatabaseVolume:", err)
		return
	}

	databaseVolume.Id = id
	databaseVolume.ImageName = database.ImageName

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "INSERT INTO DatabaseVolume( Id, Name, ImageName) VALUES(?, ?, ?)",
				Arguments: []interface{}{databaseVolume.Id, databaseVolume.Name, databaseVolume.ImageName},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot write to DatabaseVolume table: %s\n", err.Error())
	} else {
		database.VolumeId = databaseVolume.Id
	}

}

func getAllDatabase() []Database {
	var databases = []Database{}

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, Name, ImageName, VolumeId, MachineIds, Domains, Status, ContPort, HostPort, DataPath, ProjectId, CreatedAt from Database",
			Arguments: []interface{}{},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot read from Database table: %s\n", err.Error())
	}

	for rows.Next() {
		var loadedDatabase Database
		var MachineIds string
		var Domains string

		err := rows.Scan(&loadedDatabase.Id, &loadedDatabase.Name, &loadedDatabase.ImageName, &loadedDatabase.VolumeId, &MachineIds, &Domains, &loadedDatabase.Status, &loadedDatabase.ContPort, &loadedDatabase.HostPort, &loadedDatabase.DataPath, &loadedDatabase.ProjectId, &loadedDatabase.CreatedAt)
		if err != nil {
			fmt.Printf(" Cannot run Scan: %s\n", err.Error())
		}

		loadedDatabase.MachineIds = strings.Split(MachineIds, ";")
		loadedDatabase.Domains = strings.Split(Domains, ";")

		databases = append(databases, loadedDatabase)
	}
	return databases
}

func updateDatabaseStatus(databaseId string, status string) error {

	_, err := connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "UPDATE Database SET Status = ? WHERE Id = ?",
				Arguments: []interface{}{status, databaseId},
			},
		},
	)

	if err != nil {
		fmt.Printf("Cannot update a row in Database: %s\n", err.Error())
		return err
	}

	return nil
}

func scheduleDeleteDatabase(databaseId string) (result bool) {

	//To delete a database we should stop a container with Database on a machine with ID Database.MachineId
	//That's why we should update status to DatabaseStatusToDelete in the Database record
	//
	//All servers check for databases that should be deleted
	//After a container is stopped and removed, we can delete a Database record from DB
	//Note: We don't delete volumes, they should be removed in another function deleteDatabaseVolume
	//or can be used with another container on the same machine

	err := updateDatabaseStatus(databaseId, DatabaseStatusToDelete)
	if err != nil {
		fmt.Printf(" Cannot delete a Database record: %s\n", err.Error())
		return false
	}

	return true
}
