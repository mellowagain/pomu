<script lang="ts">
    import { InlineNotification, Loading, Row } from "carbon-components-svelte";
    import { onDestroy } from "svelte";

    import { readable } from "svelte/store";
    import { showNotification } from "./notifications";
    import type { VideoInfo } from "./video";
    import VideoEntry from "./VideoEntry.svelte";

    onDestroy(() => {
        clearTimeout(timeout);
    });

    let timeout: NodeJS.Timeout;
    let queue = readable<Map<string, VideoInfo>>(new Map(), (set) => {
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
                timeout = setTimeout(f, 30000);
            } else {
                console.debug("Page is visible, next update in 10 seconds");
                timeout = setTimeout(f, 10000);
            }
        };
        f();
    });

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
