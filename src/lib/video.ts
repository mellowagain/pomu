import { writable } from "svelte/store";

export interface VideoInputInfo {
    title: string,
    thumbnailUrl: string,
    uploader: string,
}

export let videoInputInfoStore = writable<VideoInputInfo>();


export interface VideoInfo {

}
