#!/usr/bin/env bats

# global variables ############################################################
CONTAINER_NAME="prometheus-net-discovery"
CST_VERSION="latest" # version of GoogleContainerTools/container-structure-test
HADOLINT_VERSION="v1.18.0"

# build container to test the behavior ########################################
@test "build container" {
  docker build -t $CONTAINER_NAME -f Dockerfile . >&2
}

# functions ###################################################################

function debug() {
  status="$1"
  output="$2"
  if [[ ! "${status}" -eq "0" ]]; then
  echo "status: ${status}"
  echo "output: ${output}"
  fi
}

function start_container() {
  run docker run --rm \
  -v "$(pwd)/test/data:/mnt/" \
  -i $CONTAINER_NAME
}

###############################################################################
## linter #####################################################################
###############################################################################

@test "start hadolint" {
  docker run --rm -i hadolint/hadolint:$HADOLINT_VERSION < Dockerfile
  debug "${status}" "${output}" "${lines}"
  [[ "${status}" -eq 0 ]]
}

@test "start container-structure-test" {

  # init
  mkdir -p "$HOME/bin"
  export PATH=$PATH:$HOME/bin

  # check the os
  if [[ "$OSTYPE" == "linux-gnu"* ]]; then
          cst_os="linux"
  elif [[ "$OSTYPE" == "darwin"* ]]; then
          cst_os="darwin"
  else
          skip "This test is not supported on your OS platform 😒"
  fi

  # donwload the container-structure-test binary
  cst_bin_name="container-structure-test-$cst_os-amd64"
  cst_download_url="https://storage.googleapis.com/container-structure-test/$CST_VERSION/$cst_bin_name"

  if [ ! -f "$HOME/bin/container-structure-test" ]; then
    curl -LO $cst_download_url
    chmod +x $cst_bin_name
    mv $cst_bin_name $HOME/bin/container-structure-test
  fi

  bash -c container-structure-test test --image ${IMAGE} -q --config test/structure_test.yaml

  debug "${status}" "${output}" "${lines}"

  [[ "${status}" -eq 0 ]]
}

@test "start yamllint" {
  docker run --rm -i -v $(pwd):/data cytopia/yamllint .
}

@test "start mdl" {
# TODO: add mdl lint
}

@test "start helm lint" {
  docker run --rm -i -v $(pwd):/data quay.io/helmpack/chart-testing sh -c "cd /data && ct lint --all"
}

###############################################################################
## test cases #################################################################
###############################################################################

###############################################################################
## general cases ##############################################################
###############################################################################

@test "Smoke test" {}
# TODO: Just start and test if the daemon is running

@test "Run scan" {}
# TODO: Start and scan the network
