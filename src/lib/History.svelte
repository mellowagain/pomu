<script lang="ts">
    import { InlineNotification, Loading, Row } from "carbon-components-svelte";

    import { readable } from "svelte/store";
    import { showNotification } from "./notifications";
    import type { VideoInfo } from "./video";
    import VideoEntry from "./VideoEntry.svelte";

    let loading = true;

    let history = readable<Map<string, VideoInfo>>(new Map(), (set) => {
        const f = async () => {
            try {
                let results = await fetch("/api/history").then((r) => r.json());
                // transform from array which contains id into id -> array map
                let map = new Map();
                for (let r of results) {
                    map.set(r.id, r);
                }

                set(map);

                loading = false;
            } catch (e) {
                showNotification({
                    title: "Failed to get history",
                    description: e.text,
                    kind: "error",
                    timeout: 5000,
                });
                loading = false;
            }

            if (document.hidden) {
                console.debug("Page is hidden, next update in 30 seconds");
                setTimeout(f, 30000);
            } else {
                console.debug("Page is visible, next update in 5 seconds");
                setTimeout(f, 5000);
            }
        };
        f();
    });
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
