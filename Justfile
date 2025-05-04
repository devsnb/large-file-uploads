# File Upload Server Just Commands
# Install 'just' from https://github.com/casey/just

# List available commands
default:
    @just --list

# Stop all running containers
stop:
    docker compose down

# Build the server image
build:
    docker compose build server

# Start all containers in detached mode
up:
    docker compose up -d

# Build and start all containers in detached mode
start: build up
    @echo "Services started. Use 'just logs' to view logs."
    @echo "Creating required bucket..."
    @sleep 3 # Give MinIO time to start up
    @just create-bucket

# View logs from all containers
logs:
    docker compose logs -f

# View logs from server container only
server-logs:
    docker compose logs -f server

# View logs from minio container only
minio-logs:
    docker compose logs -f minio

# View logs from azure container only
azure-logs:
    docker compose logs -f azurite

# Restart just the server container
restart-server:
    docker compose up -d --build server
    @echo "Server restarted. Use 'just server-logs' to view logs."

# View Docker container status
status:
    docker compose ps

# Test server connectivity
test-server:
    curl -v http://localhost:8080/health

# Reset everything - stops containers, removes volumes, and rebuilds
reset:
    docker compose down -v
    docker compose build server
    docker compose up -d
    @echo "Environment reset and restarted."
    @sleep 3 # Give MinIO time to start up
    @just create-bucket
    @echo "Use 'just logs' to view logs."

# Check MinIO buckets
list-buckets:
    docker compose exec minio mc ls myminio

# Create required MinIO bucket
create-bucket:
    @echo "Creating 'uploads' bucket in MinIO..."
    docker compose exec minio mc alias set myminio http://localhost:9000 minioadmin minioadmin
    docker compose exec minio mc mb myminio/uploads --ignore-existing
    docker compose exec minio mc anonymous set download myminio/uploads
    @echo "Bucket 'uploads' created successfully!"

# Start Azure storage emulator (Azurite) container
azure-start:
    docker run -d --name azurite -p 10000:10000 -p 10001:10001 -p 10002:10002 mcr.microsoft.com/azure-storage/azurite
    @echo "Azurite storage emulator started on ports 10000-10002"

# Stop Azure storage emulator container
azure-stop:
    docker stop azurite
    docker rm azurite

# Create a container in Azurite
azure-create-container:
    @echo "Creating 'uploads' container in Azurite..."
    az storage container create --name uploads --connection-string "DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://127.0.0.1:10000/devstoreaccount1;"
    @echo "Container 'uploads' created in Azurite"

# Switch to Azure storage provider
use-azure:
    @echo "Switching to Azure storage provider..."
    export STORAGE_PROVIDER=azure
    export AZURE_STORAGE_ACCOUNT=devstoreaccount1
    export AZURE_STORAGE_KEY=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==
    export AZURE_STORAGE_CONTAINER=uploads
    export AZURE_STORAGE_ENDPOINT=http://127.0.0.1:10000/devstoreaccount1
    @echo "Environment variables set for Azure storage. Restart the server with 'just restart-server'"

# Switch to MinIO storage provider
use-minio:
    @echo "Switching to MinIO storage provider..."
    export STORAGE_PROVIDER=minio
    @echo "Environment variables set for MinIO storage. Restart the server with 'just restart-server'"

# Start with Azure Storage configuration
start-with-azure: build
    @echo "Starting services with Azure Blob Storage configuration..."
    # Modify docker-compose environment to use Azure
    sed -i.bak 's/^      #- STORAGE_PROVIDER=azure/      - STORAGE_PROVIDER=azure/' docker-compose.yml
    sed -i.bak 's/^      #- AZURE_STORAGE_ACCOUNT=devstoreaccount1/      - AZURE_STORAGE_ACCOUNT=devstoreaccount1/' docker-compose.yml
    sed -i.bak 's/^      #- AZURE_STORAGE_KEY=.*/      - AZURE_STORAGE_KEY=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq\/K1SZFPTOtr\/KBHBeksoGMGw==/' docker-compose.yml
    sed -i.bak 's/^      #- AZURE_STORAGE_CONTAINER=uploads/      - AZURE_STORAGE_CONTAINER=uploads/' docker-compose.yml
    sed -i.bak 's/^      #- AZURE_STORAGE_ENDPOINT=http:\/\/azurite:10000\/devstoreaccount1/      - AZURE_STORAGE_ENDPOINT=http:\/\/azurite:10000\/devstoreaccount1/' docker-compose.yml
    
    # Start containers
    docker compose up -d
    @echo "Services started with Azure configuration."
    @echo "Waiting for Azurite to initialize..."
    @sleep 5
    
    # Create the Azure container
    @echo "Creating container in Azurite..."
    docker compose exec azurite mkdir -p /data
    @echo "Azure container created."
    
    @echo "Azure Blob Storage setup complete. Use 'just logs' to view logs."

# Restore the docker-compose.yml to default MinIO configuration
restore-minio-config:
    @echo "Restoring docker-compose.yml to default MinIO configuration..."
    sed -i.bak 's/^      - STORAGE_PROVIDER=azure/      #- STORAGE_PROVIDER=azure/' docker-compose.yml
    sed -i.bak 's/^      - AZURE_STORAGE_ACCOUNT=devstoreaccount1/      #- AZURE_STORAGE_ACCOUNT=devstoreaccount1/' docker-compose.yml
    sed -i.bak 's/^      - AZURE_STORAGE_KEY=.*/      #- AZURE_STORAGE_KEY=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq\/K1SZFPTOtr\/KBHBeksoGMGw==/' docker-compose.yml
    sed -i.bak 's/^      - AZURE_STORAGE_CONTAINER=uploads/      #- AZURE_STORAGE_CONTAINER=uploads/' docker-compose.yml
    sed -i.bak 's/^      - AZURE_STORAGE_ENDPOINT=http:\/\/azurite:10000\/devstoreaccount1/      #- AZURE_STORAGE_ENDPOINT=http:\/\/azurite:10000\/devstoreaccount1/' docker-compose.yml
    @echo "Configuration restored to default MinIO settings."
    
# Start with MinIO storage configuration (default)
start-with-minio: restore-minio-config
    @just start
    
# Run tests with Azure storage
test-azure-upload:
    @echo "Testing file upload with Azure Blob Storage..."
    cd test-client && python -m http.server 8000 &
    @echo "Test client running at http://localhost:8000"
    @echo "1. Open the test client in your browser"
    @echo "2. Upload a file through the interface"
    @echo "3. Check the server logs with 'just logs' to verify upload"
    
# Generate a JWT token for testing (requires jwt command)
generate-token:
    @echo "Generating JWT token for testing..."
    echo '{"sub":"testuser","name":"Test User","iat":'"$(date +%s)"',"exp":'"$(($(date +%s) + 86400))"'}' | jwt encode --secret your-jwt-secret-key
    @echo "Use this token in the Authorization header: Bearer <token>"
