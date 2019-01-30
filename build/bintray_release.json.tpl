{
    "package": {
        "name": "distributions",
        "repo": "yorc-engine",
        "subject": "ystia",
        "desc": "Ystia Orchestrator distributions and documentations",
        "website_url": "https://ystia.github.io/",
        "issue_tracker_url": "https://github.com/ystia/yorc/issues",
        "vcs_url": "https://github.com/ystia/yorc",
        "github_use_tag_release_notes": false,
        "github_release_notes_file": "CHANGELOG.md",
        "licenses": ["Apache-2.0"],
        "labels": [],
        "public_download_numbers": true,
        "public_stats": false,
        "attributes": []
    },

    "version": {
        "name": "${VERSION_NAME}",
        "desc": "Ystia Orchestrator distributions and documentations ${VERSION_NAME}",
        "released": "${RELEASE_DATE}",
        "vcs_tag": "${TAG_NAME}",
        "attributes": [],
        "gpgSign": false
    },

    "files":
        [
        {"includePattern": "dist/(yorc-.*\\.tgz)", "uploadPattern": "${VERSION_NAME}/$1"},
        {"includePattern": "dist/(yorc-server-.*-distrib\\.zip)", "uploadPattern": "${VERSION_NAME}/$1"},
        {"includePattern": "pkg/(docker-ystia-yorc-.*\\.tgz)", "uploadPattern": "${VERSION_NAME}/$1"}
        ],
    "publish": true
}

