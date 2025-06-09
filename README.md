# Kebeng Store

A fully self-hostable, open-source Snap Store backend for distributing and managing Snap packages.


## Getting Started

### Prerequisites
- Go 1.22
- Docker & Docker Compose

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