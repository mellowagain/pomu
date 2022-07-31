import { readable, writable } from "svelte/store";
import { showNotification } from "./notifications";

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

export let queue = readable<Map<string, VideoInfo>>(new Map(), (set) => {
    const f = async () => {
        try {
            let results = await fetch("/api/queue").then((r) => r.json());
            // transform from array which contains id into id -> array map
            let map = new Map();
            for (let r of results) {
                map.set(r.id, r);
            }

            set(map);
        } catch (e) {
            showNotification({
                title: "Failed to get queue",
                description: e.text,
                kind: "error",
                timeout: 5000,
            });
        }

        if (document.hidden) {
            console.debug("Page is hidden, next update in 30 seconds");
            setTimeout(f, 30000);
        } else {
            console.debug("Page is visible, next update in 10 seconds");
            setTimeout(f, 10000);
        }
    };
    f();
});

export let history = readable<Map<string, VideoInfo>>(new Map(), (set) => {
    const f = async () => {
        try {
            let results = await fetch("/api/history").then((r) => r.json());
            // transform from array which contains id into id -> array map
            let map = new Map();
            for (let r of results) {
                map.set(r.id, r);
            }

            set(map);
        } catch (e) {
            showNotification({
                title: "Failed to get history",
                description: e.text,
                kind: "error",
                timeout: 5000,
            });
        }
        if (document.hidden) {
            console.debug("Page is hidden, next update in 30 seconds");
            setTimeout(f, 30000);
        } else {
            console.debug("Page is visible, next update in 10 seconds");
            setTimeout(f, 10000);
        }
    };
    f();
});
