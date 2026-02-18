# Development

For local development, a custom docker-compose file can be used.

## Usage

In short, the development instance can be started as follows:

```sh
docker compose -f docker-compose.dev.yml up --build
```

To stop these instances, it is recommended to run:

```sh
docker compose -f docker-compose.dev.yml down
```
or (to erase persistent data):
```sh
docker compose -f docker-compose.dev.yml down -v --remove-orphans
```

When **external** (non-kebeng) dependencies are added, care needs to be taken that both the `go.mod` and `go.dev.mod` files are in sync.

This generally means that for each `go get <dependency>`, a `go get -modfile go.mod.dev <dependency>` needs to be executed (idem for `go mod tidy`).

The advantage of using this docker-compose file during development is that the kebeng services will depend on other kebeng services **as they are present in your local workspace** rather than depend on a pinned GitHub commit. This way, you are not required to push each change of a service to GitHub before the integration can be tested in other services.

## The `docker-compose.dev.yml` file

This docker-compose file is a modified copy of `docker-compose.test.yml`.

Each service in the test docker-compose file is defined with a `-dev` suffix to distinguish them from the services defined in the production, test and benchmark docker-compose files.

To incorporate **local Kebeng changes** without the need to first push the to GitHub and sync them using `go get dependency@<git-commit>`, this docker-compose file relies on `Dockerfile.dev`

## The `Dockerfile.dev` file
The custom Dockerfile (`Dockerfile.dev`) creates an image that has the entire repository as context.

Currently, the image consists of two layers (stages):
- A go module cache (only rebuilt if any of the `go.mod` files in `/services` or `/common` change)
- A builder (rebuilt if any part of the `/services` or `/common` source code is changed)

When the services are built for this image, a custom modfile (`go.dev.mod`) is used.

This custom modfile resolves all dependencies locally (using [replace directives](https://go.dev/ref/mod#go-mod-file-replace)), allowing for iterative local development without the need to push changs to GitHub before they are available.

Additional replace directives can be added to the `go.dev.mod` files if dependencies change.
