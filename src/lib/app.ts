import { writable } from "svelte/store";

export enum Page {
    Video,
    Queue,
    History,
}

export let currentPage = writable(Page.Video)
