/*
We create Image each time when we POST /deploy/:environment_id
One Environment can have only one Image
*/

package main

import (
	"bytes"
	"fmt"
	"os/user"
	"strings"

	"github.com/rqlite/gorqlite"
)

const ImageStatusToBuild = "to_build"
const ImageStatusBuilding = "building"
const ImageStatusReady = "ready"
const ImageStatusError = "error"

type Image struct {
	Id            string
	Status        string
	DeploymentId  string
	EnvironmentId string
	ErrorMsg      string
}

func addImage(image Image) Image {

	id, err := NanoId(7)
	if err != nil {
		fmt.Println("Cannot generate new NanoId for Deployment:", err)
		return image
	}

	image.Id = strings.ToLower(id)

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "INSERT INTO Image( Id, Status, DeploymentId, EnvironmentId, ErrorMsg) VALUES(?, ?, ?, ?, ?)",
				Arguments: []interface{}{image.Id, image.Status, image.DeploymentId, image.EnvironmentId, image.ErrorMsg},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot write to Image table: %s\n", err.Error())
	}
	return image
}

func updateImageStatus(image Image, status string) error {

	_, err := connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "UPDATE Image SET Status = ? WHERE Id = ?",
				Arguments: []interface{}{status, image.Id},
			},
		},
	)

	if err != nil {
		fmt.Printf("Cannot update a row in Image: %s\n", err.Error())
		return err
	}

	return nil
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

func getImagesByEnvironmentId(environmentId string) []Image {

	var images = []Image{}

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, Status, DeploymentId, EnvironmentId, ErrorMsg from Image WHERE EnvironmentId = ? ORDER BY CreatedAt DESC",
			Arguments: []interface{}{environmentId},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot read from Image table: %s\n", err.Error())
	}

	for rows.Next() {
		var Id string
		var Status string
		var DeploymentId string
		var EnvironmentId string
		var ErrorMsg string

		err := rows.Scan(&Id, &Status, &DeploymentId, &EnvironmentId, &ErrorMsg)
		if err != nil {
			fmt.Printf(" Cannot run Scan: %s\n", err.Error())
		}
		loadedImage := Image{
			Id:            Id,
			Status:        Status,
			DeploymentId:  DeploymentId,
			EnvironmentId: EnvironmentId,
			ErrorMsg:      ErrorMsg,
		}
		images = append(images, loadedImage)
	}

	return images

}

func buildImage(image Image, deployment Deployment) {

	err := updateImageStatus(image, ImageStatusBuilding)

	if err != nil {
		fmt.Printf(" Cannot update a row in Image: %s\n", err.Error())
		return
	}

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "UPDATE Deployment SET Status = ? WHERE Id = ?",
				Arguments: []interface{}{DeploymentStatusBuildingImage, deployment.Id},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot update a row in Deployment: %s\n", err.Error())
		return
	}

	//Create default scripting
	randomId, err := NanoId(10)
	if err != nil {
		fmt.Println("Cannot generate new NanoId for Deployment:", err)
		return
	}

	//Get environment
	environment := getEnvironmentById(deployment.EnvironmentId)

	//Get service
	service := getServiceById(environment.ServiceId)

	//Get previous image that we should remove
	//We keep only last 2 images
	images := getImagesByEnvironmentId(deployment.EnvironmentId)
	rmImageCmd := ""
	if len(images) > 2 {
		rmImageCmd = "docker image rm " + images[2].Id
		rmImageCmd += "\ndocker image rm " + containerRegistryIp + ":7000/" + images[2].Id
	}

	sourceFolder := randomId
	gitCloneCMD := `git clone --recurse-submodules -b ` + environment.Branch + ` ` + service.GitURL + ` ` + sourceFolder

	if deployment.SourceFolder != "" {
		sourceFolder = deployment.SourceFolder
		gitCloneCMD = ""
	}

	scriptTemplate := createTemplate("caddyfile", `
	#!/bin/sh
	cd {{.HOME_DIR}}
	{{.GIT_CLONE_CMD}}
	docker build {{.LOCAL_FOLDER}} -t {{.IMAGE_ID}}
	docker image tag {{.IMAGE_ID}} {{.CONTAINER_REGISTRY_IP}}:7000/{{.IMAGE_ID}}
	docker image push {{.CONTAINER_REGISTRY_IP}}:7000/{{.IMAGE_ID}}
	#docker manifest inspect --insecure {{.CONTAINER_REGISTRY_IP}}:7000/{{.IMAGE_ID}}
	rm -rf {{.LOCAL_FOLDER}}
	{{.RM_OLD_IMAGE_CMD}}
`)

	currentUser, err := user.Current()
	if err != nil {
		fmt.Println("Cannot get home directory, Image.go:", err)
	}

	homeDir := currentUser.HomeDir

	var templateBytes bytes.Buffer
	templateData := map[string]string{
		"GIT_CLONE_CMD":         gitCloneCMD,
		"HOME_DIR":              homeDir,
		"LOCAL_FOLDER":          sourceFolder,
		"IMAGE_ID":              image.Id,
		"CONTAINER_REGISTRY_IP": containerRegistryIp,
		"RM_OLD_IMAGE_CMD":      rmImageCmd,
	}

	if err := scriptTemplate.Execute(&templateBytes, templateData); err != nil {
		fmt.Println("Cannot execute template for Caddyfile:", err)
	}

	scriptString := templateBytes.String()

	err = executeScriptString(scriptString)
	if err != nil {
		fmt.Println("Cannot build the image")
		return
	}

	fmt.Println("Image " + image.Id + " has been built. Update status to " + ImageStatusReady)

	err = updateImageStatus(image, ImageStatusReady)

	if err != nil {
		fmt.Printf(" Cannot update a row in Image: %s\n", err.Error())
		return
	}

	updateDeploymentStatus(deployment, DeploymentStatusStartingContainers)

	if err != nil {
		fmt.Printf(" Cannot update a row in Deployment: %s\n", err.Error())
		return
	}
}
