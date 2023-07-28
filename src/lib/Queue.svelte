<script lang="ts">
    import {
        InlineNotification,
        NotificationActionButton,
        Row,
    } from "carbon-components-svelte";
    import { onDestroy } from "svelte";

    import SkeletonVideoEntry from "./SkeletonVideoEntry.svelte";
    import type { VideoInfo } from "./video";
    import VideoEntry from "./VideoEntry.svelte";

    let length = 0;

    let abortController = new AbortController();

    async function requestQueue(): Promise<VideoInfo[]> {
        // set length to 0 before fetching in case we exit with an error
        length = 0;

        let results = await fetch("/api/queue", {
            signal: abortController.signal,
        });
        let json: VideoInfo[] = await results.json();

        length = json.length;
        return json;
    }

    // small wrapper function to re-assign `queue` and force svelte to re-fetch the data.
    // this is used by the notification action button to force a refresh in case of an error
    function refreshData() {
        queue = requestQueue();
    }

    onDestroy(() => abortController.abort());

    let queue = requestQueue();
</script>

<svelte:head>
    <title>Queue - pomu.app</title>
</svelte:head>

<Row>
    <h1>
        Queue

        {#if length !== 0}
            ({length})
        {/if}
    </h1>
</Row>

{#await queue}
    <!-- 4 is the average amount of videos in the queue so display that many skeletons -->
    <SkeletonVideoEntry />
    <SkeletonVideoEntry />
    <SkeletonVideoEntry />
    <SkeletonVideoEntry />
{:then result}
    {#if result.length === 0}
        <InlineNotification
            lowContrast
            hideCloseButton
            kind="info"
            subtitle="There are currently no streams in the queue"
        />
    {:else}
        {#each result as info (info.id)}
            <VideoEntry {info} />
        {/each}
    {/if}
{:catch error}
    <InlineNotification
        lowContrast
        hideCloseButton
        kind="error"
        title="Failed to load queue:"
        subtitle={error}
    >
        <svelte:fragment slot="actions">
            <NotificationActionButton on:click={refreshData}
                >Retry</NotificationActionButton
            >
        </svelte:fragment>
    </InlineNotification>
{/await}
