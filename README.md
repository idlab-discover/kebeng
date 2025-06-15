# Kebeng Store

A fully self-hostable, open-source Snap Store backend for distributing and managing Snap packages.


## Getting Started

### Prerequisites
- Go 1.22
- Docker & Docker Compose
- Modified `snapd` and `snapcraft` packages are required (see [Custom Snap tools](#custom-snap-tools) below)

### Setup
Clone the repository with the following command:
```bash
git clone https://github.com/idlab-discover/kebeng.git
cd kebeng
```

#### Configs
Each service in the `/services` folder requires a `config.yaml`.
An example for each config file (`config-example.yaml`) is present for each of the services in the correct folder.

#### RootKey for the assertion service
The assertion service needs a `root-private-key.pem` in orde to create assertions for the root account.
Execute the following commands in the `/root` folder:
```bash
cd /services/assertion/internal
mkdir /keys
openssl genpkey -algorithm RSA -out root-private-key.pem -aes256
```

#### Run
There are 3 different run modes:
- **production**: no test data present, monitoring service not activated
- **testing**: test data is added (except for snap packages, unless they are added to `/test-data/minio/` -> see note)
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
| note: you can find the test packages [here](https://drive.google.com/drive/folders/1O0bTcIKCArTn0rMwMKbboI2_iDhlQtGT?usp=sharing). Download and put them in `/test-data/minio/`

## Interact with the Kebeng store
### Custom Snap tools
To interact with the Kebeng store, a modified version of the `snapd` and `snapcraft` tools are used.
Follow the two tutorials to create your own custom version of each tool:
- [custom snapd](custom_snapd.md)
- [custom snapcraft](custom_snapcraft.md)

### Install custom tools
After you created the custom tools, you can install them to interact with the Kebeng store.
For development, [lxc](https://linuxcontainers.org/) containers are used to install the tools on.
This way, your custom `snapd` and `snapcraft` don't interfere with the official tools that might be present on your host machine.
Make sure to push both custom snap packages to your container.

To install the tools on a lxc container, execute the following steps:

1. Install the official `snapd` tool in your container. This is needed to install the custom snapcraft tool

2. Install the custom `snapcraft` tool that is locally available in your container.

3. Install the custom `snapd` tool once the custom snapcraft tool is installed.

4. On your host machine where the official snapcraft tool is installed (and you are logged in):
```bash
snapcraft export-login my-snapcraft-login.txt
```
Push the file to the lxc container.

4. In the lxc container:
```bash
export SNAPCRAFT_STORE_CREDENTIALS=$(cat my-snapcraft-login.txt)
```

5. Add the `STORE_IP` variable to `/etc/environment` in your lxc container and set it to the IP/URL where your Kebeng store is hosted.
Restart the snapd service. (`sudo systemctl restart snapd`)

6. Add the `STORE_IP` variable to `.profile` in your lxc container and set it to the IP/URL where your Kebeng store is hosted.
Source the `.profile` file. (`source .profile`)