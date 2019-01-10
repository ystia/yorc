#!/usr/bin/env bash

scriptDir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
rootDir=$(readlink -f "${scriptDir}/..")

if [[ "${TRAVIS}" != "true" ]] ; then
    echo "This script is designed to publish CI build artifacts"
    exit 0
fi

if [[ -n "${TRAVIS_TAG}" ]] ; then
    deploy_path="yorc-engine-product-ystia-dist/ystia/yorc/dist/${TRAVIS_TAG}/{1}"
elif [[ "${TRAVIS_PULL_REQUEST}" != "false" ]]; then
    deploy_path="yorc-bin-dev-local/ystia/yorc/dist/PR-${TRAVIS_PULL_REQUEST}/{1}"
else
    deploy_path="yorc-bin-dev-local/ystia/yorc/dist/${TRAVIS_BRANCH}/{1}"
fi

curl -fL https://getcli.jfrog.io | sh

build_name="yorc-travis-ci"

./jfrog rt c --apikey="${ARTIFACTORY_API_KEY}" --url=https://ystia.jfrog.io/ystia ystia
./jfrog rt u --build-name="${build_name}" --build-number="${TRAVIS_BUILD_NUMBER}" --regexp "dist/(yorc-.*.tgz)" "${deploy_path}"
./jfrog rt u --build-name="${build_name}" --build-number="${TRAVIS_BUILD_NUMBER}" --regexp "dist/(yorc-server.*-distrib.zip)" "${deploy_path}"
# Do not publish environment variables as it may expose some secrets
#./jfrog rt bce "${build_name}" "${TRAVIS_BUILD_NUMBER}"
./jfrog rt bag "${build_name}" "${TRAVIS_BUILD_NUMBER}"
./jfrog rt bp "${build_name}" "${TRAVIS_BUILD_NUMBER}"