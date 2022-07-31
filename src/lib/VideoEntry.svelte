<script lang="ts">
    import {
        Button,
        Column,
        ImageLoader,
        Link,
        Modal,
        OutboundLink,
        Row,
        SkeletonPlaceholder,
        Tile,
        TooltipIcon,
    } from "carbon-components-svelte";
    import dayjs from "dayjs";
    import duration from "dayjs/plugin/duration";
    import relativeTime from "dayjs/plugin/relativeTime";
    import {
        CloudDownload,
        Information,
        UserMultiple,
    } from "carbon-icons-svelte";
    import type { VideoInfo } from "./video";
    import VideoCountdown from "./VideoCountdown.svelte";
    import { humanizeFileSize } from "./video";
    import VideoLog from "./VideoLog.svelte";

    export let info: VideoInfo;

    dayjs.extend(duration);
    dayjs.extend(relativeTime);

    $: humanLength = dayjs.duration(+info.length, "seconds").humanize();
    $: realLength = dayjs.duration(+info.length, "seconds").format("HH:mm:ss");

    $: humanFileSize = humanizeFileSize(+info.fileSizeBytes);

    let downloadAvailable = (async () => {
        let result = await fetch(info.downloadUrl, { method: "HEAD" });
        if (result.status == 404) throw 404;
        return;
    })();

    let submittersModal = false;
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
        <info-container>
            <Column>
                {#if !info.finished}
                    <p>
                        {#if Date.now() > new Date(info.scheduledStart).getTime()}
                            Live since
                        {:else}
                            Scheduled for
                        {/if}

                        {new Date(info.scheduledStart).toTimeString()}
                    </p>
                    <br />
                    <VideoCountdown from={info.scheduledStart} />
                {:else}
                    <p>
                        Livestream was {humanLength} long. <TooltipIcon
                            icon={Information}
                            tooltipText={realLength}
                        />
                    </p>
                {/if}

                <buttons>
                    <br />
                    <Button
                        icon={UserMultiple}
                        iconDescription="Submitters"
                        kind="tertiary"
                        on:click={(_) => (submittersModal = true)}
                    />
                    <VideoLog downloadUrl={info.downloadUrl} id={info.id} />
                    {#if info.finished}
                        {#await downloadAvailable}
                            <Button skeleton />
                        {:then}
                            <Button
                                icon={CloudDownload}
                                href={info.downloadUrl}
                            >
                                Download ({humanFileSize})
                            </Button>
                        {:catch}
                            <Button icon={CloudDownload} disabled>
                                Not found
                            </Button>
                        {/await}
                    {/if}
                </buttons>
            </Column>
        </info-container>
    </Row>
</Tile>

<Modal
    bind:open={submittersModal}
    size="sm"
    passiveModal
    modalHeading={"Submitted by"}
>
    {#each info.submitters as submitter}
        /{submitter}/
    {/each}
</Modal>

<style>
    info-container {
        position: relative;
    }
    buttons {
        display: block;
    }
</style>
