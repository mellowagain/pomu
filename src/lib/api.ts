
export interface User {
    id: string,
    name: string,
    avatar: string,
    provider: string
}

export async function apiUser(): Promise<User> {
    return await fetch('/api/user').then((r) => r.json())
}

export let user = apiUser();

export interface ApiError {
    status: number,
    statusText: string,
    why: string
}

// To be used in a .then().catch() chain after a fetch.
export async function acceptOnlyOkResponse(response: Response) {
    if (response.ok) { return response }
    throw {
        status: response.status,
        statusText: response.statusText,
        why: await response.text(),
    } as ApiError;
}

// https://stackoverflow.com/a/1909508/11494565
export function delay(fn, ms) {
    let timer = 0;

    return function(...args) {
        clearTimeout(timer);
        timer = setTimeout(fn.bind(this, ...args), ms || 0);
    }
}
