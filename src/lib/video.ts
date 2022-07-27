import { writable } from "svelte/store";

export interface VideoInfo {
    title: string,
    thumbnailUrl: string,
    uploader: string,
}

export let videoInfoStore = writable<VideoInfo>();
