#!/usr/bin/env bash

scriptDir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
rootDir=$(readlink -f "${scriptDir}/..")

if [[ "${TRAVIS}" != "true" ]] ; then
    echo "This script is designed to publish CI build artifacts"
    exit 0
fi

if [[ -n "${TRAVIS_TAG}" ]] ; then
    deploy_path="yorc-bin-release-local/ystia/yorc/dist/${TRAVIS_TAG}/{1}"
elif [[ "${TRAVIS_PULL_REQUEST}" != "false" ]]; then
    deploy_path="yorc-bin-dev-local/ystia/yorc/dist/PR-${TRAVIS_PULL_REQUEST}/{1}"
else
    deploy_path="yorc-bin-dev-local/ystia/yorc/dist/${TRAVIS_BRANCH}/{1}"
fi

curl -fL https://getcli.jfrog.io | sh

./jfrog rt c --apikey="${ARTIFACTORY_API_KEY}" --url=https://ystia.jfrog.io/ystia ystia
./jfrog rt u --build-name=my-build-name --build-number="${TRAVIS_BUILD_NUMBER}" --regexp "dist/(yorc-.*.tgz)" "${deploy_path}"
./jfrog rt u --build-name=my-build-name --build-number="${TRAVIS_BUILD_NUMBER}" --regexp "dist/(yorc-server.*-distrib.zip)" "${deploy_path}"
./jfrog rt bce my-build-name "${TRAVIS_BUILD_NUMBER}"
./jfrog rt bp my-build-name "${TRAVIS_BUILD_NUMBER}"