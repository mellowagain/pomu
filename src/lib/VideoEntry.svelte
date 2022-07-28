<script lang="ts">
    import {
        Column,
        ImageLoader,
        Link,
        OutboundLink,
        Row,
        Tag,
        Tile,
        Tooltip,
    } from "carbon-components-svelte";
    import Countdown from "svelte-countdown/src";
    import dayjs from "dayjs";
    import type { VideoInfo } from "./video";
    import { Recording } from "carbon-icons-svelte";

    export let info: VideoInfo;
</script>

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
                                >{remaining.hours} hour{remaining.hours === 1
                                    ? ""
                                    : "s"}</span
                            >
                        {/if}

                        <span
                            >{remaining.minutes} minute{remaining.minute === 1
                                ? ""
                                : "s"}</span
                        >
                        <span
                            >{remaining.seconds} second{remaining.seconds === 1
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
