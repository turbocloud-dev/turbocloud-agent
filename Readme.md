**Note: The project is in active development - API and workflows are subject to change**

![TurboCloud Web Console](https://turbocloud.dev/img/turbo-cloud-self-hosting-web-console.png)

# TurboCloud | Server Agent

[TurboCloud](https://turbocloud.dev) is a security-first alternative with available source to [Heroku](https://www.heroku.com/), [Render](https://render.com/), [Platform.sh](https://platform.sh/) and other proprietary PaaS / Serverless with no vendor lock-in. Deploy any projects almost anywhere - on virtually any cloud provider/Raspberry Pi/old laptops in minutes.

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
- Rate limiting (experimental)
- VPN (Virtual Private Network) or VPC (Virtual Private Cloud) between different data centers, local machines, and on-premise servers
- Deploy static websites, Node.js, Golang, and virtually any runtime environment
- Load balancer and proxy server
- Autoscaler (experimental)
- CI/CD (Continuous Integration & Continuous Deployment)
- Localhost tunnels: expose local web servers via a public URL with automatic HTTPS and custom domains
- HTTPS-enabled and WSS-enabled custom domains
- Works with virtually any VPS, cloud, dedicated server, or Single Board Computer running Ubuntu 22.04 LTS, so you can choose any cloud provider
- Unlimited environments for each project
- Custom domains for each environment
- GitOps or push-to-deploy
- SSH access to servers
- Resource usage monitoring
- Requires only around 10 MB of RAM and approximately 4.0% CPU usage

### Quickstart

SSH into your server running a clean installation of Ubuntu 22.04, and execute the setup command. Once the installation is complete start the TurboCloud CLI.

```
ssh root@server_ip
curl https://turbocloud.dev/setup | bash -s
turbocloud
```

### License

We like to keep everything as simple as possible, which is why the TurboCloud license consists of just four points:

- You can use TurboCloud (binaries and source code) for free, without any limitations, to deploy and manage projects—provided that neither your projects nor your company generate revenue.
- If your projects or company generate revenue, you need a TurboCloud commercial license (a one-time payment of USD 100) to deploy and manage projects. You can purchase the license at [turbocloud.dev](https://turbocloud.dev/).
- If you want to sell TurboCloud as a service (e.g., as a PaaS), you need a TurboCloud Reseller License. Contact us at [hey@turbocloud.dev](mailto:hey@turbocloud.dev) for more details.
- Redistribution in any form is not allowed.

Contact us at hey@turbocloud.dev if you have any questions.