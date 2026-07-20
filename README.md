# Kartezya HR - Human Resources Management System

A production-ready REST API backend for Human Resources Management System built with Go, PostgreSQL, and Clean Architecture.

## Features

- **Authentication & Authorization**: JWT-based authentication with role-based access control (ADMIN, EMPLOYEE)
- **Employee Management**: Complete CRUD operations for employee profiles
- **Leave Management**: Leave requests, approvals, balance tracking
- **Audit Logging**: Comprehensive audit trail for all system changes
- **Clean Architecture**: Well-structured codebase with separation of concerns
- **Database**: PostgreSQL with GORM ORM and soft delete support
- **Security**: Bcrypt password hashing, JWT tokens, role-based permissions

## Architecture

```
├── cmd/                    # Application entry points
├── internal/
│   ├── config/            # Configuration management
│   ├── database/          # Database connection and migrations
│   ├── domain/            # Domain models and entities
│   ├── handler/           # HTTP handlers (controllers)
│   ├── middleware/        # Authentication and authorization middleware
│   ├── repository/        # Data access layer
│   └── service/           # Business logic layer
├── go.mod                 # Go module file
└── main.go               # Application main file
```

## Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL 12+
- Git

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd kartezya-hr
```

2. Copy environment configuration:
```bash
cp .env.example .env
```

3. Update `.env` with your database configuration:
```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=kartezya_hr
JWT_SECRET=your-super-secret-jwt-key
```

4. Install dependencies:
```bash
go mod download
```

5. Run the application:
```bash
go run main.go
```

The server will start on `http://localhost:8080`

### Default Admin User

After seeding, you can login with:
- Email: `admin@kartezya.com`
- Password: `admin123`

## API Documentation

### Base URL
```
http://localhost:8080/api/v1
```

### Authentication

All protected endpoints require a Bearer token in the Authorization header:
```
Authorization: Bearer <your-jwt-token>
```

### API Endpoints

#### Authentication

| Method | Endpoint | Description | Access |
|--------|----------|-------------|---------|
| POST | `/auth/login` | User login | Public |
| POST | `/auth/register` | User registration | Public |
| GET | `/auth/profile` | Get current user profile | Protected |
| POST | `/auth/logout` | User logout | Protected |

#### Employees

| Method | Endpoint | Description | Access |
|--------|----------|-------------|---------|
| GET | `/employees/me` | Get my profile | Protected |
| GET | `/employees/:id` | Get employee by ID | Protected (Own/Admin) |
| PUT | `/employees/:id` | Update employee | Protected (Own/Admin) |
| POST | `/employees` | Create employee | Admin Only |
| GET | `/employees` | List all employees | Admin Only |

#### Leave Management

| Method | Endpoint | Description | Access |
|--------|----------|-------------|---------|
| POST | `/leave/requests` | Create leave request | Protected |
| GET | `/leave/requests/me` | Get my leave requests | Protected |
| GET | `/leave/requests/:id` | Get leave request | Protected (Own/Admin) |
| POST | `/leave/requests/:id/approve` | Approve leave request | Admin Only |
| POST | `/leave/requests/:id/reject` | Reject leave request | Admin Only |
| GET | `/leave/requests/pending` | Get pending requests | Admin Only |
| GET | `/leave/balances/me` | Get my leave balances | Protected |
| GET | `/leave/types` | List leave types | Protected |
| POST | `/leave/types` | Create leave type | Admin Only |

## Example Requests & Responses

### Login

**Request:**
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@kartezya.com",
    "password": "admin123"
  }'
```

**Response:**
```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2024-01-03T10:30:00Z",
    "user": {
      "id": 1,
      "email": "admin@kartezya.com",
      "is_active": true,
      "created_at": "2024-01-02T10:30:00Z"
    }
  }
}
```

### Create Employee

**Request:**
```bash
curl -X POST http://localhost:8080/api/v1/employees \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-token>" \
  -d '{
    "company_email": "john.doe@kartezya.com",
    "first_name": "John",
    "last_name": "Doe",
    "phone": "+1234567890",
    "address": "123 Main St, City",
    "state": "İstanbul",
    "city": "İstanbul",
    "gender": "Erkek",
    "date_of_birth": "1990-01-15",
    "hire_date": "2024-01-01",
    "total_experience": 5.0,
    "marital_status": "Evli",
    "emergency_contact": "+90555987654",
    "emergency_contact_name": "Jane Doe",
    "emergency_contact_relation": "Eş"
  }'
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": 1,
    "user_id": 2,
    "company_email": "john.doe@kartezya.com",
    "first_name": "John",
    "last_name": "Doe",
    "phone": "+1234567890",
    "address": "123 Main St, City",
    "state": "İstanbul",
    "city": "İstanbul",
    "gender": "Erkek",
    "date_of_birth": "1990-01-15T00:00:00Z",
    "hire_date": "2024-01-01T00:00:00Z",
    "total_experience": 5.0,
    "marital_status": "Evli",
    "emergency_contact": "+90555987654",
    "emergency_contact_name": "Jane Doe",
    "emergency_contact_relation": "Eş",
    "created_at": "2024-01-02T10:30:00Z"
  }
}
```

### Create Leave Request

**Request:**
```bash
curl -X POST http://localhost:8080/api/v1/leave/requests \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-token>" \
  -d '{
    "leave_type_id": 1,
    "start_date": "2024-02-01",
    "end_date": "2024-02-05",
    "reason": "Family vacation",
    "is_paid": true
  }'
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": 1,
    "employee_id": 1,
    "leave_type_id": 1,
    "start_date": "2024-02-01T00:00:00Z",
    "end_date": "2024-02-05T00:00:00Z",
    "requested_days": 5,
    "reason": "Family vacation",
    "status": "PENDING",
    "is_paid": true,
    "created_at": "2024-01-02T10:30:00Z"
  }
}
```

### Approve Leave Request

**Request:**
```bash
curl -X POST http://localhost:8080/api/v1/leave/requests/1/approve \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin-token>"
```

**Response:**
```json
{
  "success": true,
  "data": "Leave request approved successfully"
}
```

## Database Schema

### Key Tables

- **users**: Authentication and user information
- **roles**: System roles (ADMIN, EMPLOYEE)
- **user_roles**: Junction table for user-role relationships
- **employees**: Employee profile information
- **employee_work_information**: Work history and current position
- **companies**: Company information
- **departments**: Organizational departments
- **job_positions**: Job positions within departments
- **leave_types**: Types of leave (Annual, Sick, etc.) with document requirements
- **leave_balances**: Employee leave balances by year
- **leave_requests**: Leave requests and approvals
- **leave_documents**: Documents attached to leave requests
- **audit_logs**: System audit trail

### Relationships

- User has many UserRoles
- Employee belongs to User
- Employee has many LeaveRequests and LeaveBalances
- Company has many Departments
- Department has many JobPositions
- LeaveType has many LeaveBalances and LeaveRequests
- LeaveRequest belongs to Employee and LeaveType

## Business Rules

### Authorization Rules

**ADMIN Role:**
- Full CRUD access on all entities
- Can approve/reject leave requests
- Can manage user roles
- Can view all audit logs

**EMPLOYEE Role:**
- Can view/update only own employee profile
- Can view own work history
- Can create leave requests
- Can view own leave balances and requests
- Cannot modify employee_work_information (read-only)

### Leave Management Rules

1. **Leave Balance Creation**: Automatically created yearly for each employee
2. **Leave Request Validation**: 
   - Cannot request leave for past dates
   - Must have sufficient leave balance
   - Respects leave type configuration
3. **Leave Approval Process**:
   - Approved requests automatically deduct from leave balance
   - Rejected requests don't affect balance
   - Status changes are audit logged
4. **Document Requirements**: Some leave types may require supporting documents based on their is_required_document configuration

### Audit Logging

All CREATE, UPDATE, DELETE operations are automatically logged with:
- Entity name and ID
- Action performed
- Old and new values (JSON)
- User who performed the action
- Timestamp

## Error Handling

The API returns consistent error responses:

```json
{
  "success": false,
  "error": "Error message describing what went wrong"
}
```

Common HTTP status codes:
- `200` - Success
- `201` - Created
- `400` - Bad Request
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not Found
- `500` - Internal Server Error

## Development

### Running Tests
```bash
go test ./...
```

### Database Migrations
Migrations are automatically run on application startup. To manually run migrations:
```bash
go run main.go # Migrations run automatically
```

### Code Structure

The application follows Clean Architecture principles:

- **Domain Layer**: Core business entities and rules
- **Service Layer**: Business logic and use cases
- **Repository Layer**: Data access abstraction
- **Handler Layer**: HTTP request/response handling
- **Middleware Layer**: Cross-cutting concerns (auth, logging)

## Security Considerations

1. **Password Security**: Bcrypt hashing with appropriate cost
2. **JWT Tokens**: Configurable expiration and secure signing
3. **Authorization**: Role-based access control on all endpoints
4. **Input Validation**: Comprehensive request validation
5. **SQL Injection**: Protected by GORM ORM
6. **CORS**: Configurable cross-origin request handling

## Production Deployment

### Environment Variables

Ensure all production environment variables are properly set:

```bash
# Database
DB_HOST=your-db-host
DB_PORT=5432
DB_USER=your-db-user
DB_PASSWORD=your-secure-password
DB_NAME=kartezya_hr_prod
DB_SSLMODE=require

# JWT
JWT_SECRET=your-very-secure-jwt-secret
JWT_EXPIRY_HOURS=24

# Server
SERVER_PORT=8080
GIN_MODE=release
```

### Docker Deployment

Create a `Dockerfile`:
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
CMD ["./main"]
```

### Health Check

The application provides a health check endpoint:
```bash
curl http://localhost:8080/health
```
## DB Backup

```bash
pg_dump -h yamanote.proxy.rlwy.net -p 54606 -U postgres -d railway -f "railway_yedek-$(date +'%d-%m-%Y-%H%M').sql"
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.