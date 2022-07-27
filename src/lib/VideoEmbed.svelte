<script lang="ts">
    import { ImageLoader } from "carbon-components-svelte";

    import { videoInfoStore, VideoInfo } from "./video";

    let videoInfo: VideoInfo | null = null;
    videoInfoStore.subscribe((info) => (videoInfo = info));
</script>

{#if videoInfo}
    <ImageLoader src={videoInfo.thumbnailUrl} />
    <p><b>{videoInfo.title}</b> by <b>{videoInfo.uploader}</b></p>
{/if}
