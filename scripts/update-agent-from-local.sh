#!/bin/bash

##Parameters
##-i 1.2.3.4.5 - public ip address of a server, SSH access is required

##Example:
##
##./update-agent-from-local.sh -i 12.32.22.43
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

go build -o turbocloud
scp turbocloud root@$public_ip:/root


ssh root@$public_ip 'bash -s' <<'ENDSSH'

sudo chmod +x /root/turbocloud
sudo mv /root/turbocloud /usr/local/bin/turbocloud

sudo systemctl stop turbocloud-agent.service
sudo systemctl start turbocloud-agent.service

ENDSSH
