#!/bin/bash

##Parameters
##-i 1.2.3.4.5 - public ip address of a server
 
##Example:
##
##curl https://turbocloud.dev/quick-start | bash -s -- -i 12.32.22.43
##
##

public_ip=""

while getopts i: option
do 
    case "${option}"
        in
        i)public_ip=${OPTARG};;
    esac
done

ssh root@$public_ip server_project_folder=$server_project_folder 'bash -s' <<'ENDSSH'

echo "Checking if TurboCloud is installed on server with IP $public_ip"

status_code=$(curl --write-out %{http_code} --silent --output /dev/null localhost:5445/hey)

    if [[ "$status_code" -ne 200 ]] ; then
        echo "Installing TurboCloud agent and all required tools..."
        curl https://turbocloud.dev/setup | bash -s
    else
        echo "TurboCloud is installed already"
    fi

ENDSSH

if [[ $(lsof -i tcp:5445 ) ]]; then
    lsof -i tcp:5445 | awk 'NR!=1 {print $2}' | xargs kill
else
    echo "Port 5445 is free"
fi

echo "Starting port mapping for 5445. TurboCloud API uses port 5445, all requests to API will be secured by SSH."

ssh -o ExitOnForwardFailure=yes -f -N -L 5445:localhost:5445 root@$public_ip

status_code=$(curl --write-out %{http_code} --silent --output /dev/null localhost:5445/hey)
echo "Checking that TurboCloud API is available"
echo "Response code to GET /hey: $status_code"

echo "TurboCloud is installed on the server with IP: $public_ip"
echo "Open https://console.turbocloud.dev in your browser to add and manage apps, services, databases, and localhost tunnels."

