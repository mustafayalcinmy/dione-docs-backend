# Dione Docs API

Dione Docs is a collaborative document editing API built using **Go**, **Gin**, **GORM**, and **PostgreSQL**. It provides user authentication, document management, and permission control functionalities.

## Features
- User authentication (register, login)
- Document creation, retrieval, updating, and deletion
- Document versioning
- Document sharing and permission management
- RESTful API design with Swagger documentation

## Tech Stack
- **Go** (Gin framework)
- **PostgreSQL** (database)
- **GORM** (ORM for database interaction)
- **Swagger** (API documentation)
- **JWT Authentication**

## Installation

### Prerequisites
- Go 1.23.4
- PostgreSQL
- Swag CLI (for generating Swagger documentation)

### Install Swag CLI
```sh
go install github.com/swaggo/swag/cmd/swag@latest
```

### Clone the repository
```sh
git clone https://github.com/dione-docs-backend.git
cd dione-docs-backend
```

### Install dependencies
```sh
go mod tidy
```

### Set up environment variables
Create a `.env` file and configure the following:
```env
PORT=8080
DB_HOST=your_database_host
DB_PORT=your_database_port
DB_USER=your_database_user
DB_PASS=your_database_password
DB_NAME=your_database_name
DB_SSLMODE=disable
JWT_SECRET=your_jwt_secret
```

### Run the application
```sh
go run cmd/main.go
```

### Generate Swagger documentation
```sh
swag init -g cmd/main.go
```

## Swagger Documentation
You can access the API documentation at:
```
http://localhost:8080/swagger/index.html
```

<!-- 
## Contribution
1. Fork the repository
2. Create a new feature branch (`git checkout -b feature-name`)
3. Commit your changes (`git commit -m "Added new feature"`)
4. Push to the branch (`git push origin feature-name`)
5. Open a Pull Request

## License
This project is licensed under the MIT License.
 -->
