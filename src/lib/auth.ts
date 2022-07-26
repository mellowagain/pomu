
export interface User {
    id: string,
    name: string,
    avatar: string
}

export async function apiUser(): Promise<User> {
    return await fetch('/api/user').then((r) => r.json())
}

export let user = apiUser();
