# Paktum
<img src="logo.png" width="150" height="300" align="right" alt="Paktum's mascot">

Paktum is a simple image-server with scraping functionality, ability to detect variants/duplicates using perceptual hashing, tag-based and full text search and can generate thumbnails using imgproxy. It currently features a REST and GraphQL API, documented below.

While Paktum is a single binary, it's ran in a microservice architecture, with each service being a separate "mode". This allows for easier scaling and deployment.

For example, due to the generation of perception hashes being a CPU-intensive task, it's possible to run it on multiple machines.

## Architecture

Paktum uses a persistent Redis and Meilisearch instance to store data and exchange between modes.


## Modes

### Scrape mode
```bash
printf "hug\n" | ./SCRAPE.sh
```

This sends a query to scrape images with the type "hug"  to the scrape container.
The scrape container will then scrape images from the configured sources and send them to the redis DB in packets of 50 images each.

### Process mode
This mode is responsible for processing images, generating perceptual hashes and adding them to the Meilisearch index.

It ingests data from the Redis key that is populated by a scrape container.

Several instances of this can be run at once.

### Cleanup mode
This mode is responsible for removing images from the Meilisearch index that are tagged with banned tags.

It also generates groups of PHashes that are similar to each other, and submits a list of these groups to the Redis DB.
While this algorithm does incur an O(n^n) complexity, it's only a basic distance calculation and will run in under 100ms on over 10.000 images.

By default, an image is considered similar enough to be a variant if the Hamming-distance between their PHashes is equal to or below 10.

This should be called regularly.

### Server mode
This mode is responsible for serving the REST API and serving images.

It uses Meilisearch as search backend and reads the PHash groups from the Redis server.


## GraphQL
There's a full-featured GraphQL API included. This is the preferred API.

You can check the schema in [this file](graph/schema.graphqls) or simply check the landing page of Paktum for a full-featured GraphQL code editor.


## REST API
USE THE GRAPHQL API INSTEAD. This will *work*, but you shouldn't be using it.
GraphQL is a lot more handy.

### /api/search?query={tags}&limit={limit}
Request Method: GET

| Parameter | Type                   | Description                                             |
|-----------|------------------------|---------------------------------------------------------|
| query     | Comma-seperated string | Tags to search for                                      |
| limit     | Integer                | Limit the number of results (min 1, max 50, default 10) |

Response:
A JSON document with results that are shuffled differently each time.
    
```yaml
{
    "results": [Array of image documents, up to limit many],
    "error": "", // Error message, if any
    "total_hits": int // Total number of possible hits, not limited by limit
}
```

### /api/image/{id}
Request Method: GET

| Parameter | Type                   | Description                                             |
|-----------|------------------------|---------------------------------------------------------|
| id        | String                 | ID of the image to get                                  |

Response:

```yaml
{
    "image": {Image document},
    "error": "", // Error message, if any
}
```

### /api/image/{id}/related
Request Method: GET

| Parameter | Type                   | Description                                             |
|-----------|------------------------|---------------------------------------------------------|
| id        | String                 | ID of the image to get                                  |

Response:
```yaml
{
      "results": string[], // Array of image document IDs
      "error": "" // Error message, if any
}
```

### The image document
```yaml
{
    "ID": string, // Unique MD5 of the image
    "URL": string, // URL of the image, always a direct link
    "Tags": string[], // Array of tags
    "Tagstring": string, // Space-seperated string of tags
    "Rating": string // NSFW-rating of the image, either "general", "safe", "questionable" or "explicit"
    "Added": string, // UNIX-Timestamp of when the image was added
    "PHash": uint64, // Perceptual hash of the image
    "Size": int, // Size of the image in bytes
    "Width": int, // Width of the image in pixels
    "Height": int, // Height of the image in pixels
    "Filename": string, // Filename of the image
}
```
