import { writable } from "svelte/store";

export interface VideoInputInfo {
    title: string,
    thumbnailUrl: string,
    uploader: string,
}

export let videoInputInfoStore = writable<VideoInputInfo>();

export interface VideoInfo {
    id: string,
    submitters: string[],
    scheduledStart: string,
    finished: boolean,
    title: string,
    channelName: string,
    channelId: string,
    thumbnail: string,
    downloadUrl: string,
    fileSizeBytes: string,
    length: string,
}
