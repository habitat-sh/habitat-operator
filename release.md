# Release process

Habitat operator follows [Semantic Versioning](https://semver.org/).

## Pull request

First bump the version in the `VERSION` file which is located in the top level directory of the repository.
Run the following command to update version across files:

    make update-version

Document the changes in this release in the `CHANGELOG.md` file, following the already established pattern. Commit all the changes and `CHANGELOG.md` file under one commit message e.g.: `*: cut 0.2.0 release`. Create a PR with the changes.

## Tag the release

After the above mentioned PR was merged, switch to the updated master branch. Tag the new release with a tag named v<major>.<minor>.<patch>, e.g. `v2.1.3`, and push the tag.

    git tag -a vx.y.z -m 'vx.y.z'
    git push origin vx.y.z

## Generate release image

In the root directory of the repository generate the Docker image and push it to the Kinvolk repo on Docker hub:

    make image
    docker push kinvolk/habitat-operator:vx.y.z
    docker tag kinvolk/habitat-operator:vx.y.z kinvolk/habitat-operator:latest
    docker push kinvolk/habitat-operator:latest

## Generate the Helm chart

Switch to `gh-pages` branch and follow the steps in the `README.md` file.

## Do the release

Head over to GitHub and edit the release notes with the notes that were included in the `CHANGELOG.md` file. Also include the generated Docker image and Helm chart in the release notes.
