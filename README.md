# WhoDB

Clidey Build: <img src="https://hello.clidey.com/api/flows/status?id=b32257fa-1415-4847-a0f3-e684f5f76608&secret=cd74dbd5-36ec-42f9-b4f0-12ce9fcc762b" alt="Clidey build status" height="20px" />

### *"Making your database management disappear like magic!"*

## Description
Welcome to **WhoDB** – a powerful and user-friendly database management tool that combines the simplicity of Adminer with superior UX and performance. WhoDB is written in GoLang for optimal speed and efficiency and features interactive graphs for visualizing your entire database schema. Whether you're managing a small project or a complex enterprise system, WhoDB is designed to make your database administration tasks smoother and more intuitive.

## Quick Start

To start using WhoDB right away, you can run it using Docker:

```sh
docker run -it -p 8080:8080 clidey/whodb
```

Go to http://localhost:8080 and get started!

Or try here: https://whodb.clidey.com/login?host=quick-container-491288b0-3138-48fa-93b4-1e730296c0b7.hello.svc.cluster.local&username=user&password=password&database=Adventureworks

- This is currently populated with fake database from [postgresDBSamples](https://github.com/morenoh149/postgresDBSamples/) and the URL should automatically put the credentials

Or checkout our Demo Video: [![Demo Video](/docs/images/demo-thumbnail.png)](https://youtu.be/w3tOjRt8jGU)

## Features
- **Better UX:** Intuitive and easy-to-use interface.
- **Faster Performance:** Built with GoLang for exceptional speed.
- **Schema Visualization:** Interactive graphs to visualize your entire database schema.
- **Current Support:** PostgreSQL, MySQL

## Documentation

Check the following [Documentation README](/docs/docs.md)

## Development Setup

If you want to run and develop WhoDB locally, follow these steps:

### Prerequisites
- GoLang (latest version recommended)
- PNPM (latest version recommended)

## Backend Setup

Navigate to the core/ directory and run the GoLang application:

```sh
cd core/
go run .
```

## Frontend Setup

Navigate to the frontend/ directory and run the React frontend:

```sh
cd frontend/
pnpm i && pnpm start
```

## Contributing

We welcome contributions from the community! Feel free to open issues or submit pull requests to help improve WhoDB.


## Contact

For any inquiries or support, please reach out to [support@clidey.com](mailto:support@clidey.com).

<div style="width:100%;border-bottom:0.5px solid white;margin:50px 0px;"></div>

*WhoDB - Making your database management disappear like magic!*