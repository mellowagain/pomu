<script lang="ts">
    import {
        Button,
        Column,
        ImageLoader,
        InlineNotification,
        Link,
        ListItem,
        Modal,
        OutboundLink,
        Row,
        SkeletonText,
        Tile, Tooltip, TooltipDefinition,
        TooltipIcon,
        UnorderedList,
    } from "carbon-components-svelte";
    import dayjs from "dayjs";
    import duration from "dayjs/plugin/duration";
    import relativeTime from "dayjs/plugin/relativeTime";
    import {
        CloudDownload,
        Information,
        UserMultiple,
    } from "carbon-icons-svelte";
    import type { VideoDownload, VideoInfo } from "./video";
    import VideoCountdown from "./VideoCountdown.svelte";
    import { humanizeFileSize } from "./video";
    import VideoLog from "./VideoLog.svelte";
    import type { User } from "./api";
    import { showNotification } from "./notifications";

    export let info: VideoInfo;

    dayjs.extend(duration);
    dayjs.extend(relativeTime);

    $: startDate = new Date(info.scheduledStart);
    $: humanLength = dayjs.duration(+info.length, "seconds").humanize();
    $: realLength = dayjs.duration(+info.length, "seconds").format("HH:mm:ss");

    $: humanFileSize = humanizeFileSize(+info.fileSizeBytes);

    let downloadAvailable = async () => {
        let result = await fetch(info.downloadUrl, { method: "HEAD" });
        if (result.status == 404) throw 404;
        return;
    };

    async function downloadsAmount(): Promise<number> {
        let results = await fetch(`/api/video/${info.id}/downloads`);
        let json: VideoDownload = await results.json();

        return json.downloads;
    }

    async function submittersToUsers(submitters: string[]): Promise<User[]> {
        let results: User[] = [];

        for (let submitter of submitters) {
            if (submitter == "pomu.app") {
                results.push({
                    avatar: "pomu.app",
                    id: "im pomu",
                    name: "pomu.app",
                    provider: "self",
                });
                continue;
            }

            let data = submitter.split("/");

            if (data.length == 1) {
                // this is a legacy submitters list
                results.push({
                    avatar: "",
                    id: data[0],
                    name: data[0],
                    provider: "google",
                });
            } else {
                let provider = data[0];
                let id = data[1];

                let user = await fetch(`/api/user/${provider}/${id}`)
                    .then((res) => res.json())
                    .then((user: User) => user)
                    .catch(
                        (_) =>
                            <User>{
                                id: id,
                                avatar: "",
                                name: `${id} (using ${provider})`,
                                provider: provider,
                            }
                    );

                results.push(user);
            }
        }

        return results;
    }

    async function copyArchiveToClipboard() {
        await navigator.clipboard.writeText(`${window.location.origin}/archive/${info.id}`);

        showNotification({
            title: "Copied permalink to clipboard",
            description: "",
            timeout: 5000,
            kind: "success"
        });
    }

    let submittersModal = false;
</script>

<Tile style="margin: 20px">
    <Row padding>
        <Column style="flex-grow: 0">
            <div style="width: 200px">
                <ImageLoader src={info.thumbnail} />
            </div>
        </Column>
        <Column>
            <Link href="https://youtu.be/{info.id}" target="_blank">
                <h4>{info.title}</h4>
            </Link>

            {#if info.finished}
                <copy-container>
                    <h4 on:click={copyArchiveToClipboard}>#</h4>
                </copy-container>
            {/if}

            <br />
            <h5>
                <OutboundLink
                    href="https://youtube.com/channel/{info.channelId}"
                >
                    {info.channelName}
                </OutboundLink>
                {#if info.finished}
                    <br />
                    <TooltipDefinition tooltipText="{startDate.toDateString()} {startDate.toTimeString()}">
                        Streamed {dayjs(info.scheduledStart).fromNow()}
                    </TooltipDefinition>

                    {#await downloadsAmount() then downloadCount}
                        <Tooltip hideIcon triggerText="{downloadCount} Downloads">
                            <p>Downloads are counted globally across all pomu.app users</p>
                        </Tooltip>
                    {/await}
                {/if}
            </h5>
        </Column>
        <info-container>
            <Column>
                {#if !info.finished}
                    <p>
                        {#if Date.now() > startDate.getTime()}
                            Live since
                        {:else}
                            Scheduled for
                        {/if}

                        {startDate.toTimeString()}
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
                        {#await downloadAvailable()}
                            <Button skeleton />
                        {:then _}
                            <Button
                                icon={CloudDownload}
                                href={info.downloadUrl}
                                download
                            >
                                Download ({humanFileSize})
                            </Button>
                        {:catch _}
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
    modalHeading="Submitted by"
>
    {#if submittersModal}
        {#await submittersToUsers(info.submitters)}
            <SkeletonText paragraph width="50%" />
        {:then users}
            <UnorderedList>
                {#each users as user}
                    {#if user.provider === "self"}
                        <ListItem>
                            Automatically added to queue by pomu.app.
                            <OutboundLink
                                href="https://github.com/mellowagain/pomu/wiki/Automatic-Submissions-using-Holodex-API"
                            >
                                Learn more
                            </OutboundLink>
                        </ListItem>
                    {:else}
                        <ListItem>{user.name}</ListItem>
                    {/if}
                {/each}
            </UnorderedList>
        {:catch error}
            <InlineNotification
                lowContrast
                hideCloseButton
                kind="error"
                subtitle="Failed to load submitters"
            />
        {/await}
    {/if}
</Modal>

<style>
    info-container {
        position: relative;
    }

    buttons {
        display: block;
    }

    copy-container {
        color: gray;
        cursor: pointer;
        user-select: none;
        display: inline-flex;
        align-items: center;
    }

    copy-container:hover {
        color: white;
    }
</style>
