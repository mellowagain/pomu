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
        Toggle
    } from "carbon-components-svelte";
    import SkeletonVideoEntry from "./SkeletonVideoEntry.svelte";
    import type { HistoryResponse } from "./video";
    import VideoEntry from "./VideoEntry.svelte";
    import { onDestroy, onMount } from "svelte";

    // Search parameters
    let sorting = "desc";
    let displayUnfinished = false;
    let page = 1;
    let limit = 25;

    let history = requestHistory();
    let intervalId;

    function refreshData() {
        history = requestHistory();
    }

    async function requestHistory(): Promise<HistoryResponse> {
        let results = await fetch(`/api/history?page=${page - 1}&limit=${limit}&sort=${sorting}&unfinished=${displayUnfinished}`);
        let json = await results.json();

        return {
            totalItems: +results.headers.get("X-Pomu-Pagination-Total"),
            videos: new Map(json.map(entry => [entry.id, entry]))
        };
    }

    function scheduleRefresh() {
        intervalId = setInterval(refreshData, 30000);
    }

    function handleVisibilityChange() {
        switch (document.visibilityState) {
            case "visible":
                refreshData();
                scheduleRefresh();
                break;
            case "hidden":
                clearInterval(intervalId);
                break;
        }
    }

    onMount(() => scheduleRefresh());
    onDestroy(() => clearInterval(intervalId));
</script>

<svelte:window on:visibilitychange={handleVisibilityChange}/>

<Grid>
    <Row>
        <Column>
            <h1>History</h1>
        </Column>
        <Column></Column>
        <Column>
            <div class="toggler">
                <Toggle
                    labelText="Display unfinished"
                    bind:toggled={displayUnfinished}
                    on:toggle={refreshData}
                />
            </div>
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
            kind="info"
            subtitle="No streams have finished recording"
            on:close={(e) => {
                e.preventDefault();
            }}
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

        {#each [...result.videos.entries()] as [id, info] (id)}
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
        kind="error"
        title="Failed to load history:"
        subtitle={error}
        on:close={(e) => {
            e.preventDefault();
        }}
    >
        <svelte:fragment slot="actions">
            <NotificationActionButton on:click={refreshData}>Refresh</NotificationActionButton>
        </svelte:fragment>
    </InlineNotification>
{/await}

<style>
    .toggler {
        float: right;
    }

    .divider {
        margin-bottom: 1em;
    }
</style>
