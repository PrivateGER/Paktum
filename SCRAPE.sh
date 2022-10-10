#!/usr/bin/env sh

# This script is used to invoke the scraping docker container

# Mode 1: Read from file
if [ "$1" = "file" ]; then
    cat "$2" | docker-compose run --rm -T image_scraper
    exit 0
fi

# Mode 2: Read from stdin
cat | docker-compose run --rm -T image_scraper
