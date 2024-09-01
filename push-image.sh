#!/usr/bin/env bash
set -euxo pipefail
# NOTE: Make sure you ran "gcloud auth configure-docker europe-west3-docker.pkg.dev" as a prep step

# if [[ -n "$(git status --porcelain)" ]]; then
#   echo 'Dirty working directory, exiting.'
#   exit 1
# fi

# Change to the script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd $SCRIPT_DIR

REPO=${1:-europe-west3-docker.pkg.dev/sentry-st-testing/main}

IMAGE="${REPO}/go-load-tester"
TAG=$(git rev-parse HEAD)

echo "Pushing ${IMAGE}:${TAG}"
docker build -t $IMAGE:$TAG .

docker push $IMAGE:$TAG
