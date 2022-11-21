export const GRAPHQL_ENDPOINT = "http://paktum.localtest.me/query";

export enum Rating {
    explicit = 'explicit',
    questionable = 'questionable',
    safe = 'safe',
    general = 'general'
}

export interface Image {
    ID: string,
    Url: string,
    ThumbnailUrl: string,
    Tags: string[],
    Tagstring: string,
    Rating: Rating,
    Added: string,
    PHash: string,
    Size: number,
    Width: number,
    Height: number,
    Filename: string,
    Related: NestedImage[],
}

export interface NestedImage {
    ID: string,
    Url: string,
    ThumbnailUrl: string,
    Tags: string[],
    Tagstring: string,
    Rating: Rating,
    Added: string,
    PHash: string,
    Size: number,
    Width: number,
    Height: number,
    Filename: string,
}

export async function customQuery(graphqlQuery: string) {
    let response = await fetch(GRAPHQL_ENDPOINT, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Accept': 'application/json',
        },
        body: JSON.stringify({ query: graphqlQuery })
    });
    return await response.json();
}

export function randomImage() : Promise<Image> {
    return new Promise<Image>((resolve, reject) => {
        customQuery(`
        query {
            randomImage {
                ID
                Url
                ThumbnailUrl
                Tags
                Tagstring
                Rating
                Added
                PHash
                Size
                Width
                Height
                Filename
                Related {
                    ID
                    Url
                    ThumbnailUrl
                    Tags
                    Tagstring
                    Rating
                    Added
                    PHash
                    Size
                    Width
                    Height
                    Filename
                }
            }
        }`)
        .then((response) => resolve(response.data.randomImage as Image))
        .catch((error) => reject(error));
    });
}

export function imageById(id: string) : Promise<Image> {
    return new Promise<Image>((resolve, reject) => {
        customQuery(`
        query {
            image(ID:"${id}") {
                ID
                Url
                ThumbnailUrl
                Tags
                Tagstring
                Rating
                Added
                PHash
                Size
                Width
                Height
                Filename
                Related {
                    ID
                    Url
                    ThumbnailUrl
                    Tags
                    Tagstring
                    Rating
                    Added
                    PHash
                    Size
                    Width
                    Height
                    Filename
                }
            }
        }`)
        .then((response) => resolve(response.data.image as Image))
        .catch((error) => reject(error));
    });
}

export function searchImages(query: string, limit: number, shuffle: boolean, rating: Rating) : Promise<Image[]> {
    return new Promise<Image[]>((resolve, reject) => {
        customQuery(`
        query {
            searchImages(query:"${query}", limit:${limit}, shuffle:${shuffle}, rating:${rating}) {
                ID
                Url
                ThumbnailUrl
                Tags
                Tagstring
                Rating
                Added
                PHash
                Size
                Width
                Height
                Filename
                Related {
                    ID
                    Url
                    ThumbnailUrl
                    Tags
                    Tagstring
                    Rating
                    Added
                    PHash
                    Size
                    Width
                    Height
                    Filename
                }
            }
        }`).then(r => resolve(r.data.searchImages as Image[]))
            .catch((error) => reject(error));
    });
}

