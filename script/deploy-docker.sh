#!/bin/bash
set -e

if [ -n "$TRAVIS_COMMIT" ]; then
  echo "Deploying PR..."
else
  echo "Skipping deploy PR"
  exit 0
fi

# create docker image containous/traefik
echo "Updating docker containous/traefik image..."
docker login -e $DOCKER_EMAIL -u $DOCKER_USER -p $DOCKER_PASS
docker tag atbore-phx/traefik testci/traefik:${TRAVIS_COMMIT}
docker push testci/traefik:${TRAVIS_COMMIT}
docker tag atbore-phx/traefik testci/traefik:experimental
docker push testci/traefik:experimental

echo "Deployed"
