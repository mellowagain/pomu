<script lang="ts">
    import {
        Column,
        Dropdown,
        Grid,
        InlineNotification,
        NotificationActionButton,
        Pagination,
        PaginationSkeleton,
        Row,
    } from "carbon-components-svelte";
    import SkeletonVideoEntry from "./SkeletonVideoEntry.svelte";
    import type {HistoryResponse, VideoInfo} from "./video";
    import VideoEntry from "./VideoEntry.svelte";
    import { onDestroy } from "svelte";

    let sorting = "desc";
    let page = 1;
    let limit = 25;

    let abortController = new AbortController();

    async function requestHistory(): Promise<HistoryResponse> {
        let results = await fetch(`/api/history?page=${page - 1}&limit=${limit}&sort=${sorting}`, {
            signal: abortController.signal
        });
        let json: VideoInfo[] = await results.json();

        return {
            totalItems: +results.headers.get("X-Pomu-Pagination-Total"),
            videos: json
        };
    }

    // small wrapper function to re-assign `history` and force svelte to re-fetch the data.
    // this is used by basically every button below
    function refreshData() {
        history = requestHistory();
    }

    onDestroy(() => abortController.abort());

    let history = requestHistory();
</script>

<Grid>
    <Row>
        <Column>
            <h1>History</h1>
        </Column>
        <Column></Column>
        <Column>
            <!-- todo: add search bar here -->
        </Column>
        <Column>
            <Dropdown
                titleText="Sorting"
                bind:selectedId={sorting}
                items={[
                    { id: "asc", text: "Oldest" },
                    { id: "desc", text: "Newest" }
                ]}
                on:select={refreshData}
            />
        </Column>
    </Row>
</Grid>

<div class="divider"></div>

{#await history}
    <PaginationSkeleton />

    {#each Array(limit) as _, i}
        <SkeletonVideoEntry />
    {/each}

    <PaginationSkeleton />
{:then result}
    {#if result.videos.size === 0}
        <InlineNotification
            lowContrast
            hideCloseButton
            kind="info"
            subtitle="No streams have finished recording"
        />
    {:else}
        <Pagination
            totalItems={result.totalItems}
            pageSizes={[25, 50, 75, 100]}
            bind:pageSize={limit}
            bind:page
            on:click:button--previous={refreshData}
            on:click:button--next={refreshData}
        />

        {#each result.videos as info (info.id)}
            <VideoEntry {info} />
        {/each}

        <Pagination
            totalItems={result.totalItems}
            pageSizes={[25, 50, 75, 100]}
            bind:pageSize={limit}
            bind:page
            on:update={refreshData}
            on:click:button--previous={refreshData}
            on:click:button--next={refreshData}
        />
    {/if}
{:catch error}
    <InlineNotification
        lowContrast
        hideCloseButton
        kind="error"
        title="Failed to load history:"
        subtitle={error}
    >
        <svelte:fragment slot="actions">
            <NotificationActionButton on:click={refreshData}>Retry</NotificationActionButton>
        </svelte:fragment>
    </InlineNotification>
{/await}

<style>
    .divider {
        margin-bottom: 1em;
    }
</style>
