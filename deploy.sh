#!/usr/bin/env bash
#
# Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#


## Usage: deploy.sh
##
## Purpose: Allow to deploy on Bintray (https://bintray.com) Travis Build artifacts
##
##
##


#set -x
#set -e

#### Test variables
#TRAVIS_TAG="3.0.0-RC1"
#TRAVIS_PULL_REQUEST="42"
#TRAVIS_BRANCH="my_branch"
#BINTRAY_API_KEY=AHAHHA

#### Bintray Variables
bintray_base_url="https://api.bintray.com/content"
bintray_api_user="stebenoist"
bintray_repo_distrib="ystia/yorc-engine/distributions"

bintray_releases_path="releases"
bintray_snapshots_pr_path="snapshots/@BRANCH@/pr/@PR_ID@"
bintray_snapshots_path="snapshots/@BRANCH@"

#### Artifacts Variables
artifacts_list="dist/yorc.tgz dist/yorc-server-*-distrib.zip"
distrib_artifact_dir="./dist"
distrib_artifact_basename="yorc-server-"

#### get_version ()
#### Allows to get the current distribution version
get_version () {
  find ${distrib_artifact_dir} -type f -name "${distrib_artifact_basename}*-distrib.zip" | awk -F${distrib_artifact_basename} '{print $2}' | sed "s|-distrib.zip||g" | sed "s|-SNAPSHOT||g"
}

############################################### Main ###############################################
#### Check the env var BINTRAY_API_KEY is set
if [[ ! -n "$BINTRAY_API_KEY" ]]; then
    echo "The env variable BINTRAY_API_KEY is not set" >&2
    exit 1
fi

#### Retrieve the version with distribution artifact
version=$(get_version)
if [[ ! -n "${version}" ]]; then
    echo "No distribution artifact with correct name has been foud." >&2
    exit 1
fi
echo "version is: ${version}"

#### Retrieve bintray expected path
if [[ -n "$TRAVIS_TAG" ]]; then
  echo "Deploying release \"$TRAVIS_TAG\""
  bintray_path=${bintray_releases_path}
else
  if [[ "$TRAVIS_PULL_REQUEST" == "false" ]]; then
    echo "Deploying snapshot from push on branch \"$TRAVIS_BRANCH\""
    bintray_path=$(echo ${bintray_snapshots_path} | sed "s|@BRANCH@|$TRAVIS_BRANCH|g")
  else
    if [[ -n "$TRAVIS_BRANCH" ]]; then
      echo "Deploying snapshot from pr ($TRAVIS_PULL_REQUEST) on branch \"$TRAVIS_BRANCH\""
      bintray_path=$(echo ${bintray_snapshots_pr_path} | sed "s|@BRANCH@|$TRAVIS_BRANCH|g" | sed "s|@PR_ID@|$TRAVIS_PULL_REQUEST|g")
    fi
  fi
fi

if [[ ! -n "${bintray_path}" ]]; then
    echo "Unable to define expected bintray path" >&2
    echo "TRAVIS_TAG=\"$TRAVIS_TAG\"" >&2
    echo "TRAVIS_PULL_REQUEST=\"$TRAVIS_PULL_REQUEST\"" >&2
    echo "TRAVIS_BRANCH=\"$TRAVIS_BRANCH\"" >&2
    exit 1
fi
echo "bintray path is: ${bintray_path}"

#### Loop artifacts list and upload each one to bintray
for artifact in ${artifacts_list}
do
  echo "Deploying artifact ${artifact}"
  filename="$(basename ${artifact})"

  cmd="curl -T ${artifact} -H \"X-Bintray-Publish: 1\" -H \"X-Bintray-Override: 1\" -u ${bintray_api_user}:$BINTRAY_API_KEY ${bintray_base_url}/${bintray_repo_distrib}/${version}/${bintray_path}/${filename}"
  echo "run command: \"${cmd}\""
  eval $cmd
done

