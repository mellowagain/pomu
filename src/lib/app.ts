import { writable } from "svelte/store";

export enum Page {
    Video,
    Queue,
    History,
}

let currentOpenedPage;

switch (window.location.pathname) {
    case "/queue":
        currentOpenedPage = Page.Queue;
        break;
    case "/history":
        currentOpenedPage = Page.History;
        break;
    default:
        currentOpenedPage = Page.Video;
        break;
}

export let currentPage = writable(currentOpenedPage)
