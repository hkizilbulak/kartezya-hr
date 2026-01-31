# Kartezya HR API Documentation

## Overview

This document provides comprehensive information about the Kartezya HR API, including Swagger documentation and Postman collection usage.

## API Documentation

### Swagger/OpenAPI Documentation

The API is fully documented using Swagger/OpenAPI 2.0 specifications. All endpoints include:
- Complete parameter documentation
- Request/response schemas
- Authentication requirements
- Error responses
- Example payloads

#### Accessing Swagger UI

1. Start the application: `go run main.go`
2. Visit: `http://localhost:8080/swagger/index.html`

#### Regenerating Swagger Documentation

```bash
# Install swag if not already installed
go install github.com/swaggo/swag/cmd/swag@latest

# Regenerate documentation
swag init -g main.go -o docs/
```

### Postman Collection

A complete Postman collection is provided at `postman/kartezya-hr-complete-api.postman_collection.json` with:

- All 40+ API endpoints
- Pre-configured environment variables
- Automatic authentication token management
- Sample request payloads
- Organized folder structure

#### Importing the Collection

1. Open Postman
2. Click "Import" 
3. Select `postman/kartezya-hr-complete-api.postman_collection.json`
4. The collection will be imported with all endpoints organized by feature

#### Using the Collection

1. **Authentication**: Start with the "Login" request in the Authentication folder
2. **Auto-token**: The collection automatically extracts and sets the auth token from login responses
3. **Variables**: Update the `baseUrl` variable if your server runs on a different port
4. **Sample Data**: All requests include realistic sample payloads

## API Endpoints Summary

### Authentication (5 endpoints)
- `POST /auth/login` - User login
- `POST /auth/logout` - User logout
- `POST /auth/change-password` - Change user password (authenticated)
- `POST /auth/validate-reset-token` - Validate password reset token
- `POST /auth/reset-password` - Reset user password with token
- `POST /auth/send-password-reset-email-batch` - Send password reset emails to multiple users (authenticated)

### Companies (5 endpoints)
- `POST /companies` - Create company
- `GET /companies` - List companies (paginated)
- `GET /companies/{id}` - Get company by ID
- `PUT /companies/{id}` - Update company
- `DELETE /companies/{id}` - Delete company

### Departments (6 endpoints)
- `POST /departments` - Create department
- `GET /departments` - List departments (paginated)
- `GET /departments/lookup` - Get departments lookup
- `GET /departments/{id}` - Get department by ID
- `PUT /departments/{id}` - Update department
- `DELETE /departments/{id}` - Delete department

### Job Positions (6 endpoints)
- `POST /job-positions` - Create job position
- `GET /job-positions` - List job positions (paginated)
- `GET /job-positions/lookup` - Get job positions lookup
- `GET /job-positions/{id}` - Get job position by ID
- `PUT /job-positions/{id}` - Update job position
- `DELETE /job-positions/{id}` - Delete job position

### Employees (5 endpoints)
- `POST /employees` - Create employee
- `GET /employees` - List employees
- `GET /employees/{id}` - Get employee by ID
- `PUT /employees/{id}` - Update employee
- `DELETE /employees/{id}` - Delete employee

### Work Information (6 endpoints)
- `POST /work-information` - Create work information
- `GET /work-information` - List work information (paginated)
- `GET /work-information/{id}` - Get work information by ID
- `GET /work-information/me` - Get my work information
- `PUT /work-information/{id}` - Update work information
- `DELETE /work-information/{id}` - Delete work information

### Leave Management

#### Leave Types (6 endpoints)
- `POST /leave/types` - Create leave type
- `GET /leave/types` - List leave types (paginated)
- `GET /leave/types/lookup` - Get leave types lookup
- `GET /leave/types/{id}` - Get leave type by ID
- `PUT /leave/types/{id}` - Update leave type
- `DELETE /leave/types/{id}` - Delete leave type

#### Leave Requests (8 endpoints)
- `POST /leave/requests` - Create leave request
- `GET /leave/requests` - Get all leave requests (Admin only)
- `GET /leave/requests/me` - Get my leave requests
- `GET /leave/requests/{id}` - Get leave request by ID
- `PUT /leave/requests/{id}` - Update leave request
- `POST /leave/requests/{id}/approve` - Approve leave request
- `POST /leave/requests/{id}/reject` - Reject leave request
- `POST /leave/requests/{id}/cancel` - Cancel leave request

#### Leave Balances (1 endpoint)
- `GET /leave/balances/me` - Get my leave balances

## Authentication

All endpoints (except login) require Bearer token authentication:

```
Authorization: Bearer <token>
```

The token is obtained from the login endpoint and automatically handled in the Postman collection.

## Role-Based Access Control

- **Admin**: Full access to all endpoints
- **Employee**: Limited access to personal data and leave requests

## Error Handling

All endpoints return consistent error responses:

```json
{
  "success": false,
  "error": "Error message",
  "details": "Optional detailed error information"
}
```

## Pagination

List endpoints support pagination with query parameters:
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 10)
- `sort`: Sort field
- `direction`: Sort direction (ASC/DESC)

## Sample Workflows

### 1. Complete Employee Onboarding

```
1. Login as Admin
2. Create Company (if needed)
3. Create Department
4. Create Job Position
5. Create Employee
6. Create Work Information for Employee
```

### 2. Employee Leave Request Flow

```
1. Employee login
2. View leave balances: GET /leave/balances/me
3. Submit leave request: POST /leave/requests
4. Admin approves: POST /leave/requests/{id}/approve
```

### 3. Organizational Setup

```
1. Create Companies
2. Create Departments for each Company
3. Create Job Positions
4. Use lookup endpoints to populate dropdowns in UI
```

## Development

### Adding New Endpoints

1. Add Swagger annotations to handler methods:
```go
// @Summary Endpoint summary
// @Description Detailed description
// @Tags tag-name
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param param body/path/query Type true "Description"
// @Success 200 {object} ResponseType
// @Failure 400 {object} APIResponse
// @Router /endpoint [method]
```

2. Regenerate Swagger docs: `swag init -g main.go -o docs/`
3. Update Postman collection with new requests

### Testing

Use the Postman collection for comprehensive API testing:
1. Import the collection
2. Set up environment variables
3. Run the entire collection or specific folders
4. Use Postman's test scripts for automated validation

## Support

For API support or questions:
1. Check Swagger documentation at `/swagger/index.html`
2. Use the Postman collection for testing
3. Review this README for usage guidelines