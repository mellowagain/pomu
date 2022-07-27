<script lang="ts">
    import { Column, ImageLoader, Row, Tile } from "carbon-components-svelte";

    import { readable, writable } from "svelte/store";
    import { showNotification } from "./notifications";
    import type { VideoInfo } from "./video";

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
        };
        f();
        setInterval(f, 5000);
    });

    function getHoursMinutesInFuture(time: Date): [number, number] {
        let diff = new Date(time.getTime() - Date.now());
        return [diff.getHours(), diff.getMinutes()];
    }
</script>

<Row>
    <h1>Queue</h1>
</Row>

{#each [...$queue.entries()] as [id, info] (id)}
    <Tile style="margin: 20px">
        <Row padding>
            <Column>
                <div style="width: 200px">
                    <ImageLoader src={info.thumbnail} />
                </div>
            </Column>
            <Column>
                <p>
                    {#if Date.now() > new Date(info.scheduledStart).getTime()}
                        Started
                    {:else}
                        Starting
                    {/if}
                    at {new Date(info.scheduledStart).toTimeString()}
                </p>
                <br />
                {#if Date.now() < new Date(info.scheduledStart).getTime()}
                    <p>
                        (thats in {getHoursMinutesInFuture(
                            new Date(info.scheduledStart)
                        )[0]}h {getHoursMinutesInFuture(
                            new Date(info.scheduledStart)
                        )}m )
                    </p>
                {/if}
            </Column>
            <Column>
                <h4>{info.title}</h4>
                <br />
                <h5>{info.channelName}</h5>
            </Column>
        </Row>
    </Tile>
{/each}

<style>
</style>
