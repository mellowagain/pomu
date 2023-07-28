import { writable } from "svelte/store";

export enum Page {
    Video,
    Queue,
    History,
}

let currentOpenedPage = Page.Video;

switch (window.location.pathname) {
    case "/queue":
        currentOpenedPage = Page.Queue;
        break;
    case "/history":
        currentOpenedPage = Page.History;
        break;
    default:
        if (window.location.pathname.startsWith("/archive/")) {
            currentOpenedPage = Page.History;
        } else {
            currentOpenedPage = Page.Video;
        }

        break;
}

export let currentPage = writable(currentOpenedPage)
