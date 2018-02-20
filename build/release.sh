#!/usr/bin/env bash 
set -x
set -e
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

if [[ "$(python -c "import semantic_version; print semantic_version.validate('${version}')" )" != "True" ]]; then
    echo "Parameter -v should be a semver 2.0 compatible version (http://semver.org/)" >&2
    exit 1
fi

# read version
read -r major minor patch prerelease build <<< $(python -c "import semantic_version; v = semantic_version.Version('${version}'); print v.major, v.minor, v.patch, '.'.join(v.prerelease), '.'.join(v.build);")

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
    if [[ "True" != "$(python -c "import semantic_version; print  semantic_version.Version('${version}') >= semantic_version.Version('${checkVers}')" )" ]]; then
        echo "Can't release version ${version} on top of branch ${branch} as its current version is ${currentVersion}" >&2
        exit 1
    fi
fi

# Check branch tags
branchTag=$(git describe --abbrev=0 --tags ${branch}) || {
    branchTag="v0.0.0"
}
branchTag=$(echo $branchTag | sed -e 's/^v\(.*\)$/\1/')

if [[ "True" != "$(python -c "import semantic_version; print  semantic_version.Version('${version}') > semantic_version.Version('${branchTag}')" )" ]]; then
    echo "Can't release version ${version} on top of branch ${branch} as it contains a newer tag: ${branchTag}" >&2
    exit 1
fi

if [[ "develop" == "${branch}" ]] && [[ -z "${prerelease}" ]]; then
    # create release branch
    releaseBranch="release/${major}.${minor}"
    git checkout -b "${releaseBranch}"
fi

# Now checks are passed then tag, build, release and cleanup :)
if [[ -e versions.yaml ]]; then
    # Update version
    sed -i -e "/${componentVersionName}: /c${componentVersionName}: ${version}" versions.yaml
    git commit -m "Prepare release ${version}" versions.yaml
fi

git tag -a v${version} -m "Release tag v${version}"

if [[ -e versions.yaml ]]; then
    # Update version
    nextDevelopmentVersion=""
    if [[ -z "${prerelease}" ]]; then 
        # We are releasing a final version
        nextDevelopmentVersion=$(python -c "import semantic_version; v=semantic_version.Version('${version}'); print v.next_patch()" )
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
    if [[ -e versions.yaml ]]; then
        # Update version
        nextDevelopmentVersion=$(python -c "import semantic_version; v=semantic_version.Version('${version}'); print v.next_minor()" )
        nextDevelopmentVersion="${nextDevelopmentVersion}-SNAPSHOT"
        sed -i -e "/${componentVersionName}: /c${componentVersionName}: ${nextDevelopmentVersion}" versions.yaml
        git commit -m "Prepare for next development cycle ${nextDevelopmentVersion}" versions.yaml
    fi
fi

if [[ -z "${prerelease}" ]]; then
    # Merge on master only final version
    masterTag=$(git describe --abbrev=0 --tags master) || {
        masterTag="v0.0.0"
    }
    masterTag=$(echo ${masterTag} | sed -e 's/^v\(.*\)$/\1/')

    if [[ "True" == "$(python -c "import semantic_version; print  semantic_version.Version('${version}') > semantic_version.Version('${masterTag}')" )" ]]; then
        # We should merge the tag to master as it is our highest release
        git checkout master
        git merge --no-ff "v${version}" -X theirs -m "merging latest tag v${version} into master" || {
                git merge --abort || true
                git reset --hard "v${version}"
        }

    fi
fi


####################################################
# Make our build
####################################################
echo "Building version v${version}"
git checkout "v${version}"
make tools || { exit 2; }
SKIP_TESTS=1 make dist || { exit 2; }

# Push changes
if [ "${dryRun}" = false ] ; then
    set +x
    git push ${PUSH_URL} --all
    git push ${PUSH_URL} --tags
fi

