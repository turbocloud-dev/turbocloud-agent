![TurboCloud Web Console](https://turbocloud.dev/img/turbo-cloud-self-hosting-web-console.png)

# TurboCloud | Server Agent

[TurboCloud](https://turbocloud.dev) is a security-first alternative to [Heroku](https://www.heroku.com/), [Render](https://render.com/), [Platform.sh](https://platform.sh/) and other proprietary PaaS / Serverless with no vendor lock-in. Deploy any projects almost anywhere - on virtually any cloud provider/Raspberry Pi/old laptops in minutes.

More info about the project: [turbocloud.dev](https://turbocloud.dev)

Contact us if you have any questions: hey[a]turbocloud.dev

**Don't forget to click on Star if you like the project.**

### Main features

- Single binary
- No Ops & no infrastructure management
- Deploy directly from a local repository or from GitHub, Bitbucket, and GitLab
- Deployments with or without a Dockerfile
- Includes a built-in container registry (no need for a third-party container registry)
- WAF (Web Application Firewall) with a set of generic attack detection rules recommended by OWASP
- No configuration in YAML files is required
- Rate limiting (WIP)
- VPN (Virtual Private Network) or VPC (Virtual Private Cloud) between different data centers, local machines, and on-premise servers
- Deploy static websites, Node.js, Golang, and virtually any runtime environment
- Load balancer and proxy server
- Autoscaler (WIP)
- CI/CD (Continuous Integration & Continuous Deployment)
- Localhost tunnels: expose local web servers via a public URL with automatic HTTPS and custom domains (WIP)
- HTTPS-enabled and WSS-enabled custom domains
- Works with virtually any VPS, cloud, dedicated server, or Single Board Computer running Ubuntu 22.04 LTS, so you can choose any cloud provider
- Unlimited environments for each project
- Custom domains for each environment
- GitOps or push-to-deploy
- SSH access to servers
- Resource usage monitoring
- Requires only around 50 MB of RAM and approximately 4.0% CPU usage on servers with 1vCPU

### Quickstart

#### What You Need

- A server (for example, a server on Hetzner, DigitalOcean, Scaleway, etc.) with a public IP (at least one public IP per project), SSH access, and Ubuntu 22.04
- Dockerfile in the root folder of your project (haven't Dockerfile yet - just search or ask AI "create dockerfile for Node.js" or "create dockerfile for go"), etc.
- (Optional) Domain name

#### Deploy from local folders
In the root folder of your project on your local machine, run the following command
(replace **server_public_ip** with the IP of your server and **port_your_app_listens_to** with a port you use in your service/app):
```
curl https://turbocloud.dev/deploy | bash -s -- -i server_public_ip -p port_your_app_listens_to
```

To use a custom domain, add the -d parameter (make sure your domain’s A record points to your server’s IP address):
```
curl https://turbocloud.dev/deploy | bash -s -- -i server_public_ip -p port_your_app_listens_to -d some_domain.com
```

Once installation is complete, open <a href="https://console.turbocloud.dev">console.turbocloud.dev</a> in a browser to add and manage servers/apps/databases/localhost tunnels.

---

#### Deploy from GitHub and Bitbucket

SSH into your server running a clean Ubuntu 22.04 installation, and run the following command:

```
curl https://turbocloud.dev/setup | bash -s
```

Once the installation is complete, open <a href="https://console.turbocloud.dev">console.turbocloud.dev</a> in your browser to add and manage servers, apps, databases, and localhost tunnels.

---

#### Deploy from GitHub and Bitbucket (without SSHing into your server)

If you prefer to install remotely, run the following command on your development machine (replace `<code>server_public_ip</code>` with your server's actual IP address; SSH access is still required):

```
curl https://turbocloud.dev/quick-start | bash -s -- -i server_public_ip
```

Once the installation is complete, open <a href="https://console.turbocloud.dev">console.turbocloud.dev</a> in your browser to add and manage servers, apps, databases, and localhost tunnels.

---

### TurboCloud Agent Development

To quickly update the agent on a server, you can use the `update-agent-from-local.sh` script (tested on Linux and macOS). This script builds a new agent locally, uploads it to the server, and restarts the agent service:

```bash
./scripts/update-agent-from-local.sh -i server_ip_with_agent
```


### License

GNU GENERAL PUBLIC LICENSE

Contact us at hey@turbocloud.dev if you have any questions.
