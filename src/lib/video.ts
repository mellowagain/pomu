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

export function humanizeFileSize(sizeBytes: number) {
    let mbSize = sizeBytes / (1000 * 1000);
    let gbSize = mbSize / 1000;

    if (gbSize < 1) {
        return Math.round(mbSize) + " MB";
    }

    return gbSize.toFixed(1) + " GB";
}

export interface HistoryResponse {
    videos: VideoInfo[],
    totalItems: number
}
