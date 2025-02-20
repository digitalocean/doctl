# Snap

Snap packages are automatically built and uploaded as part of the GitHub Actions
release workflow.

To build a snap package locally for testing, first install `snapcraft`.

On Ubuntu, run:

    sudo snap install snapcraft --classic

Or on MacOS, run:

    brew install snapcraft

Finally, build the package by running:

    make snap

More details about the snap package can be found in the `snap/snapcraft.yaml` file.
