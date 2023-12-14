# Prod targets reg the amd agent image for this commit: 
# https://github.com/hyperlane-xyz/hyperlane-monorepo/commit/69850642365251e16b9f23b87d6caf187036b3ee
up-prod:
	@if [ ! -f .env ]; then echo "Error: .env file not found. Please populate the .env file before running this command."; exit 1; fi
	AGENT_BASE_IMAGE=gcr.io/abacus-labs-dev/hyperlane-agent@sha256:854f92966eac6b49e5132e152cc58168ecdddc76c2d390e657b81bdaf1396af0 L2_NODE_URL=http://34.215.163.180:8545 docker compose --profile settlement --profile bridge --profile prod_agents up -d	

up-prod-build:
	@if [ ! -f .env ]; then echo "Error: .env file not found. Please populate the .env file before running this command."; exit 1; fi
	AGENT_BASE_IMAGE=gcr.io/abacus-labs-dev/hyperlane-agent@sha256:854f92966eac6b49e5132e152cc58168ecdddc76c2d390e657b81bdaf1396af0 L2_NODE_URL=http://34.215.163.180:8545 docker compose --profile settlement --profile bridge --profile prod_agents up -d --build

up-prod-settlement:
	@if [ ! -f .env ]; then echo "Error: .env file not found. Please populate the .env file before running this command."; exit 1; fi
	AGENT_BASE_IMAGE=nil L2_NODE_URL=nil docker compose --profile settlement up -d --build	

up-dev:
	@if [ ! -f .env ]; then echo "Error: .env file not found. Please populate the .env file before running this command."; exit 1; fi
	AGENT_BASE_IMAGE=shaspitz/hyperlane-agent-mac:6985064 L2_NODE_URL=http://localhost:8545 docker compose --profile settlement --profile bridge up -d

up-dev-build:
	@if [ ! -f .env ]; then echo "Error: .env file not found. Please populate the .env file before running this command."; exit 1; fi
	AGENT_BASE_IMAGE=shaspitz/hyperlane-agent-mac:6985064 L2_NODE_URL=http://localhost:8545 docker compose --profile settlement --profile bridge up -d --build

up-dev-settlement:
	@if [ ! -f .env ]; then echo "Error: .env file not found. Please populate the .env file before running this command."; exit 1; fi
	AGENT_BASE_IMAGE=shaspitz/hyperlane-agent-mac:6985064 L2_NODE_URL=http://localhost:8545 docker compose --profile settlement up -d --build

down:
	AGENT_BASE_IMAGE=nil L2_NODE_URL=nil docker compose --profile settlement --profile bridge --profile prod_agents down

clean-dbs:
	@read -p "WARNING: This command will wipe all persistent disk data relevant to the containers. Press enter to continue or Ctrl+C to cancel." _ 
	-docker compose --profile settlement --profile bridge down --rmi all --volumes
	-docker compose --profile settlement --profile bridge rm -fv
	docker image prune -f

pull-image:
	docker pull shaspitz/geth-poa:v0

DOCKER_NETWORK_NAME="primev_net"
create-docker-network: 
	@echo "Creating Docker network: $(DOCKER_NETWORK_NAME)..."
	@if ! docker network inspect $(DOCKER_NETWORK_NAME) >/dev/null 2>&1; then \
		docker network create --driver bridge --subnet 172.29.0.0/16 $(DOCKER_NETWORK_NAME); \
	else \
		echo "Network $(DOCKER_NETWORK_NAME) already exists."; \
	fi

deploy-create2:
	@echo "Deploying create2 deployment proxy..."
	./util/deploy_create2.sh "http://localhost:8545"	
