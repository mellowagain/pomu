<script lang="ts">
    import { ImageLoader } from "carbon-components-svelte";

    import { videoInputInfoStore, VideoInfo } from "./video";

    let videoInfo: VideoInfo | null = null;
    videoInputInfoStore.subscribe((info) => (videoInfo = info));
</script>

{#if videoInfo}
    <ImageLoader src={videoInfo.thumbnailUrl} />
    <p><b>{videoInfo.title}</b> by <b>{videoInfo.uploader}</b></p>
{/if}
