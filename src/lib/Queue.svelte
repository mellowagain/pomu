<script lang="ts">
    import {
        Column,
        ImageLoader,
        InlineNotification,
        Link,
        Loading,
        OutboundLink,
        Row,
        Tag,
        Tile,
        Tooltip,
    } from "carbon-components-svelte";

    import { readable, writable } from "svelte/store";
    import { showNotification } from "./notifications";
    import dayjs from "dayjs";
    import type { VideoInfo } from "./video";
    import { Recording } from "carbon-icons-svelte";
    import Countdown from "svelte-countdown/src/Countdown.svelte";

    let loading = true;

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

                loading = false;
            } catch (e) {
                showNotification({
                    title: "Failed to get queue",
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

    let started;
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
    <Tile style="margin: 20px">
        <Row padding>
            <Column>
                <div style="width: 200px">
                    <ImageLoader src={info.thumbnail} />
                </div>
            </Column>
            <Column>
                <Link href="https://youtu.be/{info.id}" target="_blank">
                    <h4>{info.title}</h4>
                </Link>
                <br />
                <h5>
                    <OutboundLink
                        href="https://youtube.com/channel/{info.channelId}"
                    >
                        {info.channelName}
                    </OutboundLink>
                </h5>
            </Column>
            <Column>
                <p>
                    {#if Date.now() > new Date(info.scheduledStart).getTime()}
                        Live since
                    {:else}
                        Scheduled for
                    {/if}

                    {new Date(info.scheduledStart).toTimeString()}
                </p>
                <br />

                <Countdown
                    from={dayjs(info.scheduledStart)}
                    dateFormat="x"
                    let:remaining
                >
                    {#if !remaining.done}
                        <p>
                            Starts in
                            {#if remaining.days > 0}
                                <span
                                    >{remaining.days + remaining.months * 30} day{remaining.days ===
                                    1
                                        ? ""
                                        : "s"}</span
                                >
                            {/if}

                            {#if remaining.hours > 0}
                                <span
                                    >{remaining.hours} hour{remaining.hours ===
                                    1
                                        ? ""
                                        : "s"}</span
                                >
                            {/if}

                            <span
                                >{remaining.minutes} minute{remaining.minute ===
                                1
                                    ? ""
                                    : "s"}</span
                            >
                            <span
                                >{remaining.seconds} second{remaining.seconds ===
                                1
                                    ? ""
                                    : "s"}</span
                            >
                        </p>
                    {:else}
                        <Tag icon={Recording} type="red">Live</Tag>
                    {/if}
                </Countdown>

                <Tooltip triggerText="Submitters">
                    {#each info.submitters as submitter}
                        /{submitter}/
                    {/each}
                </Tooltip>
            </Column>
        </Row>
    </Tile>
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
