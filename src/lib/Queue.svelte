<script lang="ts">
    import { InlineNotification, Loading, Row } from "carbon-components-svelte";

    import { readable } from "svelte/store";
    import { showNotification } from "./notifications";
    import { queue } from "./video";
    import VideoEntry from "./VideoEntry.svelte";

    let loading = true;
    queue.subscribe((_) => (loading = false));
</script>

<Row>
    <h1>
        Queue

        {#if $queue.size > 0 && !loading}
            ({$queue.size})
        {/if}
    </h1>

    {#if loading}
        <Loading />
    {/if}
</Row>

{#each [...$queue.entries()] as [id, info] (id)}
    <VideoEntry {info} />
{/each}

{#if $queue.size === 0 && !loading}
    <InlineNotification
        lowContrast
        kind="info"
        subtitle="There are currently no streams in the queue"
        on:close={(e) => {
            e.preventDefault();
        }}
    />
{/if}

<style>
</style>
