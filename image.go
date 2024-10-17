/*
We create Image each time when we POST /deploy/:environment_id
One Environment can have only one Image
*/

package main

import (
	"fmt"

	"github.com/rqlite/gorqlite"
)

const ImageStatusToBuild = "to_build"
const ImageStatusBuilding = "building"
const ImageStatusReady = "ready"
const ImageStatusError = "error"

type Image struct {
	Id           string
	Status       string
	DeploymentId string
	ErrorMsg     string
}

func addImage(image Image) Image {

	id, err := NanoId(7)
	if err != nil {
		fmt.Println("Cannot generate new NanoId for Deployment:", err)
		return image
	}

	image.Id = id

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "INSERT INTO Image( Id, Status, DeploymentId, ErrorMsg) VALUES(?, ?, ?, ?)",
				Arguments: []interface{}{image.Id, image.Status, image.DeploymentId, image.ErrorMsg},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot write to Deployment table: %s\n", err.Error())
	}
	return image
}

func getImageByDeploymentIdAndStatus(deploymentId string, status string) []Image {

	var images = []Image{}

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, Status, DeploymentId, ErrorMsg from Image WHERE STATUS = ? AND DeploymentId = ?",
			Arguments: []interface{}{status, deploymentId},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot read from Image table: %s\n", err.Error())
	}

	for rows.Next() {
		var Id string
		var Status string
		var DeploymentId string
		var ErrorMsg string

		err := rows.Scan(&Id, &Status, &DeploymentId, &ErrorMsg)
		if err != nil {
			fmt.Printf(" Cannot run Scan: %s\n", err.Error())
		}
		loadedImage := Image{
			Id:           Id,
			Status:       Status,
			DeploymentId: DeploymentId,
			ErrorMsg:     ErrorMsg,
		}
		images = append(images, loadedImage)
	}

	return images

}
