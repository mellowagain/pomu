<script lang="ts">
    import { ImageLoader } from "carbon-components-svelte";

    import { videoInputInfoStore, VideoInfo, VideoInputInfo } from "./video";

    let videoInfo: VideoInputInfo | null = null;
    videoInputInfoStore.subscribe((info) => (videoInfo = info));
</script>

{#if videoInfo}
    <ImageLoader src={videoInfo.thumbnailUrl} />
    <p><b>{videoInfo.title}</b> by <b>{videoInfo.uploader}</b></p>
{/if}
