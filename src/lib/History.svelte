<script lang="ts">
    import { InlineNotification, Loading, Row } from "carbon-components-svelte";

    import { readable } from "svelte/store";
    import { showNotification } from "./notifications";
    import { history } from "./video";
    import VideoEntry from "./VideoEntry.svelte";

    let loading = true;
    history.subscribe((_) => (loading = false));
</script>

<Row>
    <h1>
        History

        {#if $history.size > 0 && !loading}
            ({$history.size})
        {/if}
    </h1>

    {#if loading}
        <Loading />
    {/if}
</Row>

{#each [...$history.entries()] as [id, info] (id)}
    <VideoEntry {info} />
{/each}

{#if $history.size === 0 && !loading}
    <InlineNotification
        lowContrast
        kind="info"
        subtitle="No streams have finished recording."
        on:close={(e) => {
            e.preventDefault();
        }}
    />
{/if}
