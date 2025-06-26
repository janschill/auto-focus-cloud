#!/bin/bash

# Get the latest tag from git
LATEST_TAG=$(git tag -l "v*" --sort=-version:refname | head -n1)

if [ -z "$LATEST_TAG" ]; then
    echo "No existing tags found. Next version will be v1"
    NEXT_VERSION="v1"
else
    echo "Latest tag: $LATEST_TAG"
    CURRENT_NUM=$(echo $LATEST_TAG | sed 's/v//')
    NEXT_NUM=$((CURRENT_NUM + 1))
    NEXT_VERSION="v$NEXT_NUM"
    echo "Next version: $NEXT_VERSION"
fi

# Optionally create the tag
if [ "$1" = "--create" ]; then
    echo "Creating tag $NEXT_VERSION"
    git tag $NEXT_VERSION
    echo "Tag created. Push with: git push origin $NEXT_VERSION"
fi