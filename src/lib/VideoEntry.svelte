<script lang="ts">
    import {
        Button,
        CodeSnippet,
        Column,
        ImageLoader,
        Link,
        Modal,
        OutboundLink,
        Popover,
        Row,
        Tag,
        Tile,
        Tooltip,
        TooltipIcon,
    } from "carbon-components-svelte";
    import Countdown from "svelte-countdown/src";
    import dayjs from "dayjs";
    import {
        CloudDownload,
        Recording,
        Report,
        UserMultiple,
        Warning,
    } from "carbon-icons-svelte";
    import type { VideoInfo } from "./video";
    import VideoCountdown from "./VideoCountdown.svelte";

    export let info: VideoInfo;

    let log = (async () => {
        let result = await fetch("/api/logz?id=" + info.id);
        if (result.status != 200) {
            throw "no log";
        }

        let text = await result.text();

        return text
            .replaceAll("frame=", "\nframe=")
            .replaceAll("[mpegts", "\n[mpegts");
    })();

    let missing = (async () => {
        let result = await fetch(info.downloadUrl, { method: "HEAD" });
        if (result.status == 404) throw 404;
        return;
    })();

    let logModal = false;
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
                    <p>Livestream finished.</p>
                {/if}

                <buttons>
                    <br />
                    <Button
                        icon={UserMultiple}
                        iconDescription="Submitters"
                        kind="tertiary"
                        on:click={(_) => (submittersModal = true)}
                    />
                    {#await log}
                        <Button
                            icon={Report}
                            iconDescription="FFMpeg Log"
                            kind="tertiary"
                            disabled
                        />
                    {:then}
                        <Button
                            icon={Report}
                            iconDescription="FFMpeg Log"
                            kind="tertiary"
                            on:click={(_) => (logModal = true)}
                        />
                    {/await}
                    {#if info.finished}
                        {#await missing}
                            <Button skeleton />
                        {:then}
                            <Button
                                icon={CloudDownload}
                                href={info.downloadUrl}
                            >
                                Download
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

<Modal bind:open={logModal} size="lg" passiveModal modalHeading={"FFMpeg Log"}>
    {#await log then log}
        <CodeSnippet type="multi" expanded>
            {log}
        </CodeSnippet>
    {/await}
</Modal>

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
