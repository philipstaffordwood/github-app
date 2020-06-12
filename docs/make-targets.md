# Make targets:

## Generic

* `help`           - Lists valid targets

## Setup and Config
* `setup`          - Install required dependencies esc

## Build and Install
* `build`(default) - Build binaries
* `install`        - Installs binary locally (needs admin privileges)
* `linux`          - Build for Linux
* `darwin`         - Build for Darwin
* `compress`       - Uses UPX to compress the executable
* `docker`         - Build docker image

## Documentation
* `serve-docs`     - Serves the MkDocs docs locally
* `build-api-docs` - Build golang docs
* `build-docs`     - Build MkDocs docs

[//]: # (TODO: * `deploy-docs`    - Deploy MkDocs to Netlify)

# Usage
Normal first time use:
```shell
make setup        # make sure esc and github-release are installed
make              # do a local build
make compress     # compress the built executable
sudo make install # install the executable to /usr/local/bin/ 
```

