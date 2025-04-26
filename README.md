# Godspeed CLI (Go Version)

<div align="center">
    <img width="200" height="200" src="https://github.com/godspeedsystems/godspeed-cli/blob/main/logo.png">
    <h1 align="center">Godspeed CLI</h1>
<p align="center">
  The official Command Line Interface of Godspeed Framework - Go Edition
</p>
<br>

  <p>
    <a href="https://github.com/godspeedsystems/gs-node-service">
      <img src="https://img.shields.io/badge/contributions-welcome-brightgreen?logo=github" alt="contributions welcome">
    </a>
    <a href="https://discord.com/invite/MKjv3KdD7X">
      <img src="https://img.shields.io/badge/chat-discord-brightgreen.svg?logo=discord&style=flat" alt="Discord">
    </a>
    <a href="https://godspeed.systems/">
      <img src="https://img.shields.io/website?url=https%3A%2F%2Fgodspeed.systems%2F" alt="Website">
    </a>
  </p>
  <br />
</div>

CLI to create and manage [Godspeed](https://github.com/godspeedsystems/gs-node-service) projects, rewritten in Go for improved performance and cross-platform compatibility.

## About

[Godspeed CLI](https://godspeed.systems/docs/microservices-framework/CLI) is the primary way to interact with your Godspeed project from the command line. It provides a bunch of useful functionalities during the project development lifecycle.

This is the Go version of the original Node.js-based CLI tool, offering better performance and more reliable operation across different platforms.

## Installation

### From Source

To build from source, make sure you have Go 1.21 or higher installed:

```bash
git clone https://github.com/godspeedsystems/godspeed-cli-go
cd godspeed-cli-go
go install
```

### Binary Releases

You can download pre-built binaries from the [releases page](https://github.com/godspeedsystems/godspeed-cli-go/releases).

## Basic Usage

Once installed, run `godspeed` from your terminal to see the available commands and usage information. The CLI provides detailed help messages for each command, guiding you through its functionalities and options.

## Supported Commands & Arguments

| Command              | Options                       | Description                                                 |
|----------------------|-------------------------------|-------------------------------------------------------------|
| create <projectName> | --from-template, --from-example | Create a new godspeed project                            |
| dev                  |                               | Start the dev server                                        |
| clean                |                               | Clean the previous build                                    |
| build                |                               | Build the godspeed project                                  |
| plugin               | add, remove, update           | Manage eventsource and datasource plugins for godspeed     |
| devops-plugin        | install, list, remove, update | Manage devops plugins for godspeed                         |
| gen-crud-api         |                               | Scan prisma datasources and generate CRUD APIs              |
| gen-graphql-schema   |                               | Scan graphql events and generate graphql schema             |
| prisma prepare       |                               | Prepare your prisma database for use                        |
| otel                 | enable, disable               | Enable/disable Observability in Godspeed                    |
| link/unlink          |                               | Link/unlink a project to the global environment             |

## Key Features

1. **Project Creation**: Create new Godspeed projects with templates and examples
   ```bash
   godspeed create my-project
   godspeed create my-project --from-example hello-world
   ```

2. **Plugin Management**: Add, remove, and update plugins
   ```bash
   godspeed plugin add @godspeedsystems/plugins-express-as-http
   godspeed plugin remove @godspeedsystems/plugins-express-as-http
   godspeed plugin update
   ```

3. **DevOps Plugin Management**: Manage DevOps plugins for streamlined development
   ```bash
   godspeed devops-plugin install
   godspeed devops-plugin list --installed
   ```

4. **GraphQL Schema Generation**: Generate GraphQL schemas from event definitions
   ```bash
   godspeed gen-graphql-schema
   ```

5. **Database Management**: Prisma database preparation and CRUD API generation
   ```bash
   godspeed prisma prepare
   godspeed gen-crud-api
   ```

6. **Observability**: Enable or disable OpenTelemetry integration
   ```bash
   godspeed otel enable
   godspeed otel disable
   ```

## 📖 Documentation

For a comprehensive understanding of Godspeed CLI and its advanced features, explore the detailed documentation [here](https://godspeed.systems/docs/microservices-framework/guide/get-started).

## Show Your Love ❤️ & Support 🙏

If you find the Godspeed Node.js Service helpful or interesting, we would greatly appreciate your support by following, starring, and subscribing.

<div style="display: flex; align-items: center;">
    <a style="margin: 10px;" href="https://www.linkedin.com/company/godspeed-systems/"><img src="https://badgen.net/static/follow/linkedin/blue"></a>
    <span style="margin-left: 5px;">Follow us for updates and news.</span>
</div>
<div style="display: flex; align-items: center;">
    <a style="margin: 10px;" href="https://github.com/godspeedsystems/gs-node-service/"><img src="https://badgen.net/static/follow/github/Priority-green"></a>
    <span style="margin-left: 5px;">Star our repositories to show your support.</span>
</div>
<div style="display: flex; align-items: center;">
    <a style="margin: 10px;" href="https://www.youtube.com/@godspeed.systems/videos"><img src="https://badgen.net/static/follow/youtube/red"></a>
    <span style="margin-left: 5px;">Subscribe to our channel for tutorials and demos.</span>
</div>

## Ask & Answer Questions

Got questions or need help with Godspeed? Join our [Discord community](https://discord.com/invite/E3WU9dT7UQ). You can ask questions, share knowledge, and assist others.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the Godspeed Source Available License - see the [LICENSE](LICENSE) file for details.