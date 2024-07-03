# WhoDB

Clidey Build: <img src="https://hello.clidey.com/api/flows/status?id=b32257fa-1415-4847-a0f3-e684f5f76608&secret=cd74dbd5-36ec-42f9-b4f0-12ce9fcc762b" alt="Clidey build status" height="20px" />

### *"Making your database management disappear like magic!"*

## Description
Welcome to **WhoDB** – a powerful, lightweight (~20Mi), and user-friendly database management tool that combines the simplicity of Adminer with superior UX and performance. WhoDB is written in GoLang for optimal speed and efficiency and features interactive graphs for visualizing your entire database schema. Whether you're managing a small project or a complex enterprise system, WhoDB is designed to make your database administration tasks smoother and more intuitive.

## Quick Start

To start using WhoDB right away, you can run it using Docker:

```sh
docker run -it -p 8080:8080 clidey/whodb
```

or using docker-compose

```sh
version: "3.8"
services:
  whodb:
    image: clidey/whodb
    # volumes: # (optional for sqlite) 
    #   - ./sample.db:/db/sample.db
    ports:
      - "8080:8080"
```

Go to http://localhost:8080 and get started!

Or try here: https://whodb.clidey.com/login?host=quick-container-491288b0-3138-48fa-93b4-1e730296c0b7.hello.svc.cluster.local&username=user&password=password&database=Adventureworks

- This is currently populated with fake database from [postgresDBSamples](https://github.com/morenoh149/postgresDBSamples/) and the URL should automatically put the credentials

Or checkout our Demo Video: [![Demo Video](/docs/images/demo-thumbnail.png)](https://youtu.be/w3tOjRt8jGU)

## Features
- **Better UX:** Intuitive and easy-to-use interface
- **Faster Performance:** Built with GoLang for exceptional speed and table virtualization in Frontend
- **Schema Visualization:** Interactive graphs to visualize your entire database schema
- **Inline Editing & Preview:** Easily preview cell or edit inline
- **Current Support:** PostgreSQL, MySQL, SQLite3, MongoDB, & Redis
- **Scratchpad:** Perform database queries in a jupyter notebook like experience

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

## FAQs

**Q: What inspired the creation of WhoDB?**

A: WhoDB was inspired by Adminer due to its lightweight nature and ease of use. We aimed to enhance these qualities with a focus on graph-based visualization and a consistent user experience across different types of databases.

**Q: How does WhoDB handle large queries?**

A: WhoDB supports lazy loading to efficiently manage and display large query results, ensuring smooth performance even with extensive datasets.

**Q: What makes WhoDB different from DBeaver?**

A: While DBeaver is a highly advanced tool written in Java, it can be resource-intensive. WhoDB, on the other hand, is designed to be lightweight and runs with minimal resources, making it accessible to a wider range of users and devices. You can run WhoDB with as little as 50m core and 100Mb RAM. WhoDB is also only ~25Mb compressed size.

**Q: Can I use WhoDB with any type of database?**

A: Yes, WhoDB aims to provide a consistent exploration and editing experience across SQL, NoSQL, and Graph databases. It currently only supports PostgreSQL, MySQL, SQLite3, MongoDB, & Redis.

**Q: How do I deploy WhoDB?**

A: WhoDB can be easily deployed using Docker or Docker Compose. Check "Getting Started" section for more details.

**Q: Is WhoDB suitable for production environments?**

A: While WhoDB is designed for lightweight and efficient database exploration, it is always recommended to evaluate its suitability for your specific production environment and use case.

## Contributing

We welcome contributions from the community! Feel free to open issues or submit pull requests to help improve WhoDB.


## Contact

For any inquiries or support, please reach out to [support@clidey.com](mailto:support@clidey.com).

<div style="width:100%;border-bottom:0.5px solid white;margin:50px 0px;"></div>

*WhoDB - Making your database management disappear like magic!*