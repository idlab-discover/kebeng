# Kebeng Store

A fully self-hostable, open-source Snap Store backend for distributing and managing Snap packages.


## Getting Started

### Prerequisites
- Go 1.22
- Docker & Docker Compose
- Modified `snapd` and `snapcraft` packages (see [tools](tools))

### Setup
Clone the repository with the following command:
```bash
git clone https://github.com/idlab-discover/kebeng.git
cd kebeng
```

#### Configs
Each service in the `/services` folder requires a `config.yaml`.
An example for each config file (`config-example.yaml`) is present for each of the services in the correct folder.

#### RootKey for the account service
The account service needs a `root-private-key.pem` in orde to create a root account.
Execute the following commands in the `/root` folder:
```bash
cd /services/account/internal
mkdir /keys
openssl genpkey -algorithm RSA -out root-private-key.pem -aes256
```

#### Run
There are 3 different run modes:
- **production**: no test data present, monitoring service not activated
- **testing**: test data is added
- **monitoring**: test data is added and monitoring service is activated

Run them with the following commands:
```bash
# Production
docker compose -f docker-compose.yml down -v --remove-orphans
docker compose -f docker-compose.yml up --build

# Testing
docker compose -f docker-compose.test.yml down -v --remove-orphans
docker compose -f docker-compose.test.yml up --build

# Monitoring
docker compose -f docker-compose.benchmark.yml down -v --remove-orphans
docker compose -f docker-compose.benchmark.yml up --build
```

## Interact with the Kebeng store
### Custom Snap tools
To interact with the Kebeng store, a modified version of the `snapd` and `snapcraft` tools is used.
Follow the two tutorials to create your own custom version of each tool:
- [custom snapd](custom_snapd.md)
- [custom snapcraft](custom_snapcraft.md)