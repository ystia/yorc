#!/usr/bin/env bash
# Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#set -x
set -eo pipefail
scriptDir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

componentVersionName="yorc_version"

cd ${scriptDir}/..

python -c "import semantic_version" > /dev/null 2>&1 || {
    echo -e "Python library semantic_version is required.\nPlease install it using:\n\tpip install semantic_version" >&2
    exit 1
}

dryRun=false
version=
PUSH_URL=
while getopts ":dv:p:" opt; do
  case $opt in
    v)
      version=${OPTARG}
      ;;
    d)
      dryRun=true
      ;;
    p)
      PUSH_URL=${OPTARG}
      ;;
    \?)
      echo "Invalid option: -$OPTARG" >&2
      exit 1
      ;;
    :)
      echo "Option -$OPTARG requires an argument." >&2
      exit 1
      ;;
  esac
done

if [[ -z "${version}" ]]; then
    echo "Parameter -v is required to indicate the version to build and tag" >&2
    exit 1
fi

if [[ "$(python -c "import semantic_version; print(semantic_version.validate('${version}'))" )" != "True" ]]; then
    echo "Parameter -v should be a semver 2.0 compatible version (http://semver.org/)" >&2
    exit 1
fi

# read version
read -r major minor patch prerelease build <<< $(python -c "import semantic_version; v = semantic_version.Version('${version}'); print(v.major, v.minor, v.patch, '.'.join(v.prerelease), '.'.join(v.build));")

# Detect correct supporting branch
branch=$(git branch --list -r "*/release/${major}.${minor}")
if [[ -z "${branch}" ]]; then
    branch="develop"
fi
branch=$(echo ${branch} | sed -e "s@^.*/\(release/.*\)@\1@")
echo "Switching to branch ${branch}..."
releaseBranch=${branch}
git checkout ${branch}

if [[ -e versions.yaml ]]; then
    # Check that current version is lower than the release version
    currentVersion=$(grep  "${componentVersionName}:" versions.yaml | head -1 | sed -e 's/^[^:]\+:\s*\(.*\)\s*$/\1/')
    # Change -SNAPSHOT into -0 for comparaison as a snapshot is never revelant
    checkVers=$(echo ${currentVersion} | sed -e "s/-SNAPSHOT/-0/")
    if [[ "True" != "$(python -c "import semantic_version; print(semantic_version.Version('${version}') >= semantic_version.Version('${checkVers}'))" )" ]]; then
        echo "Warning: releasing version ${version} on top of branch ${branch} while its current version is ${currentVersion}" >&2
        read -p "Are you sure? [y/N]" CONFIRM
        if [[ "${CONFIRM}" != "y" && "${CONFIRM}" != "Y" ]] ; then
            exit 1
        fi
    fi
fi

# Check branch tags
branchTag=$(git describe --abbrev=0 --tags ${branch}) || {
    branchTag="v0.0.0"
}
branchTag=$(echo $branchTag | sed -e 's/^v\(.*\)$/\1/')

if [[ "True" != "$(python -c "import semantic_version; print(semantic_version.Version('${version}') > semantic_version.Version('${branchTag}'))" )" ]]; then
    echo "Warning: releasing version ${version} on top of branch ${branch} while it contains a newer tag: ${branchTag}" >&2
    read -p "Are you sure? [y/N]" CONFIRM
    if [[ "${CONFIRM}" != "y" && "${CONFIRM}" != "Y" ]] ; then
        exit 1
    fi
fi

if [[ "develop" == "${branch}" ]] && [[ -z "${prerelease}" ]]; then
    # create release branch
    releaseBranch="release/${major}.${minor}"
    git checkout -b "${releaseBranch}"
    sed -i -e "s@svg?branch=[^)]*@svg?branch=${releaseBranch}@g" README.md
    git commit -m "Update CI links in readme for release ${version}" README.md
fi

# Now checks are passed then tag, build, release and cleanup :)
cherries=()
# Update changelog Release date
sed -i -e "s/^## UNRELEASED.*$/## ${version} ($(LC_ALL=C date +'%B %d, %Y'))/g" CHANGELOG.md
# Update readme for Release number
versionShield=$(echo "${version}" | sed -e 's/-/--/g')
sed -i -e "s@https://img.shields.io/badge/download-[^-]*-blue@https://img.shields.io/badge/download-v${versionShield}-blue@g" \
    -e "s@releases/tag/[^)]*@releases/tag/v${version}@g" README.md
git commit -m "Update changelog and readme for release ${version}" CHANGELOG.md README.md
cherries+=("$(git log -1 --pretty=format:"%h")")

if [[ -e versions.yaml ]]; then
    # Update version
    sed -i -e "/${componentVersionName}: /c${componentVersionName}: ${version}" versions.yaml
    git commit -m "Prepare release ${version}" versions.yaml
fi

git tag -a v${version} -m "Release tag v${version}"

# Update changelog for future release
sed -i -e "2a## UNRELEASED\n" CHANGELOG.md
git commit -m "Update changelog for future release" CHANGELOG.md
cherries+=("$(git log -1 --pretty=format:"%h")")

if [[ -e versions.yaml ]]; then
    # Update version
    nextDevelopmentVersion=""
    if [[ -z "${prerelease}" ]]; then
        # We are releasing a final version
        nextDevelopmentVersion=$(python -c "import semantic_version; v=semantic_version.Version('${version}'); print(v.next_patch())" )
        nextDevelopmentVersion="${nextDevelopmentVersion}-SNAPSHOT"
    else
        # in prerelease revert to version minus prerelease plus -SNAPSHOT
        nextDevelopmentVersion="${major}.${minor}.${patch}-SNAPSHOT"
    fi

    sed -i -e "/${componentVersionName}: /c${componentVersionName}: ${nextDevelopmentVersion}" versions.yaml
    git commit -m "Prepare for next development cycle ${nextDevelopmentVersion}" versions.yaml
fi


if [[ "develop" == "${branch}" ]] && [[ -z "${prerelease}" ]]; then
    # merge back to develop
    git checkout develop

    if [[ ${#cherries[@]} -gt 0 ]] ; then
        git cherry-pick ${cherries[@]}
    fi

    if [[ -e versions.yaml ]]; then
        # Update version
        nextDevelopmentVersion=$(python -c "import semantic_version; v=semantic_version.Version('${version}'); print(v.next_minor())" )
        nextDevelopmentVersion="${nextDevelopmentVersion}-SNAPSHOT"
        sed -i -e "/${componentVersionName}: /c${componentVersionName}: ${nextDevelopmentVersion}" versions.yaml
        git commit -m "Prepare for next development cycle ${nextDevelopmentVersion}" versions.yaml
    fi
fi

if [[ -z "${prerelease}" ]]; then
    # Merge on master only final version
    masterTag=$(git describe --abbrev=0 --tags origin/master) || {
        masterTag="v0.0.0"
    }
    masterTag=$(echo ${masterTag} | sed -e 's/^v\(.*\)$/\1/')

    if [[ "True" == "$(python -c "import semantic_version; print(semantic_version.Version('${version}') > semantic_version.Version('${masterTag}'))" )" ]]; then
        # We should merge the tag to master as it is our highest release
        git checkout master
        git merge --no-ff "v${version}" -X theirs -m "merging latest tag v${version} into master" || {
                git merge --abort || true
                git reset --hard "v${version}"
        }

    fi
fi

# Push changes
if [ "${dryRun}" = false ] ; then
    set +x
    git push ${PUSH_URL} --all
    git push ${PUSH_URL} --tags
fi

