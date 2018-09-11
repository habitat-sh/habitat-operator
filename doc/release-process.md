# Release process

Habitat operator follows [Semantic Versioning](https://semver.org/).

## Pull request

First bump the version in the `VERSION` file which is located in the top level directory of the repository.
Then run the following command to update version across files:

    make update-version

Document the important changes in this release in the `CHANGELOG.md` file, following the already established pattern. You can get a list of changes since the last release by running:

    git log --merges v0.8.0...master --reverse

Commit all the generated changes and the changes in the `CHANGELOG.md` file under one commit message e.g.: `*: cut 0.8.0 release`. Create a PR with the changes.

## Tag the release

After the above mentioned PR was merged, switch to the updated master branch and run:

    # Do a dry run first to see if the commands look right:
    hack/tag-release.sh
    # If the commands are right:
    hack/tag-release.sh -r

This will create a new tag on upstream.

## Generate release image

And now run:

    # Do a dry run first to see if the commands look right:
    hack/publish-release.sh
    # If the commands are right:
    hack/publish-release.sh -r

This will create a docker image and push it to the docker repository on docker hub.

## Generate the Helm chart

Switch to `gh-pages` branch and follow the steps in the `README.md` file.

## Do the release

Head over to GitHub and edit the release notes with the notes that were included in the `CHANGELOG.md` file. Also include the generated Docker image and Helm chart in the release notes.
