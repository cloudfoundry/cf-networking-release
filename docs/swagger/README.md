# CF Networking API Documentation

This directory contains the OpenAPI/Swagger documentation for the Cloud Foundry Networking API.

## Generated Files

- **`swagger.yaml`** - OpenAPI 2.0 specification in YAML format
- **`swagger.json`** - OpenAPI 2.0 specification in JSON format  
- **`docs.go`** - Go source file for embedding documentation in the application
- **`index.html`** - Swagger UI viewer for the documentation

## API Overview

The CF Networking API provides endpoints for managing network policies and tags in Cloud Foundry:

### 🔐 Authentication

All endpoints require OAuth2 authentication with appropriate scopes:
- **`network.admin`** - Full access to all policy operations
- **`network.write`** - Write access to policies (for space developers)

### 📋 Endpoints

| Method | Path | Description | Required Scope |
|--------|------|-------------|----------------|
| GET | `/policies` | List network policies | `network.write` |
| POST | `/policies` | Create network policies | `network.write` |
| POST | `/policies/delete` | Delete network policies | `network.write` |
| GET | `/tags` | List tag mappings | `network.admin` |

## Viewing the Documentation

### Option 1: Swagger Editor (Online)
1. Go to [https://editor.swagger.io/](https://editor.swagger.io/)
2. Copy the content of `swagger.yaml` and paste it in the editor

### Option 2: Local Swagger UI
1. Serve this directory with a local web server:
   ```bash
   # Using Python
   python -m http.server 8000
   
   # Using Node.js
   npx http-server
   
   # Using Go
   go run -m http.FileServer -addr=:8000 .
   ```
2. Open http://localhost:8000 in your browser

### Option 3: Swagger UI Docker
```bash
docker run -p 8080:8080 -v $(pwd):/usr/share/nginx/html/swagger -e SWAGGER_JSON=/swagger/swagger.json swaggerapi/swagger-ui
```

## Regenerating Documentation

To regenerate the OpenAPI specification after code changes:

```bash
./scripts/generate-swagger.sh
```

This script will:
1. Install `swaggo/swag` if not present
2. Scan the CF Networking codebase for Swag comments
3. Generate updated `swagger.yaml`, `swagger.json`, and `docs.go` files

## API Usage Examples

### Authentication
```bash
# Get OAuth token
export TOKEN=$(cf oauth-token)

# Use with API
curl -H "Authorization: $TOKEN" \
  https://api.bosh-lite.com/networking/v1/external/policies
```

### List Policies
```bash
curl -H "Authorization: $TOKEN" \
  "https://api.bosh-lite.com/networking/v1/external/policies"
```

### Create Policy
```bash
curl -H "Authorization: $TOKEN" \
  -H "Content-Type: application/json" \
  -X POST \
  -d '{
    "policies": [
      {
        "source": {"id": "app-1-guid"},
        "destination": {
          "id": "app-2-guid",
          "protocol": "tcp",
          "ports": {"start": 8080, "end": 8080}
        }
      }
    ]
  }' \
  "https://api.bosh-lite.com/networking/v1/external/policies"
```

## Schema Information

The API uses these main data structures:

- **Policy** - Defines network connectivity between source and destination applications
- **Source/Destination** - Application identifiers and network configuration
- **Ports** - Port range specification (start/end)
- **Tag** - Unique identifier assigned to policy groups
- **PoliciesPayload** - Container for multiple policies with count information

For detailed schema information, see the generated `swagger.yaml` file or view in Swagger UI.