#!/bin/bash

#Set default value for $HOME because it doesn't exist in cloudinit

DEFAULT_HOME='/root'
HOME=${HOME:-$DEFAULT_HOME}


url_download_vpn_certs=""
domain=""
token_to_generate_vpn_certs_url=""
webhook_url=""

server_ip="$(curl https://turbocloud.dev/ip)"
server_ip_without_dots="${server_ip//./-}"
echo $server_ip_without_dots
if [ "$domain" = "" ]; then
    #Set automatic domain
    domain="l-$server_ip_without_dots.dns.turbocloud.dev"
fi

echo $domain